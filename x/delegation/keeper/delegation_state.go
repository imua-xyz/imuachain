package keeper

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"

	assetstype "github.com/ExocoreNetwork/exocore/x/assets/types"
	delegationtype "github.com/ExocoreNetwork/exocore/x/delegation/types"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) AllDelegationStates(ctx sdk.Context) (delegationStates []delegationtype.DelegationStates, err error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), delegationtype.KeyPrefixRestakerDelegationInfo)
	iterator := sdk.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	ret := make([]delegationtype.DelegationStates, 0)
	for ; iterator.Valid(); iterator.Next() {
		var stateInfo delegationtype.DelegationAmounts
		k.cdc.MustUnmarshal(iterator.Value(), &stateInfo)
		ret = append(ret, delegationtype.DelegationStates{
			Key:    string(iterator.Key()),
			States: stateInfo,
		})
	}
	return ret, nil
}

func (k Keeper) SetAllDelegationStates(ctx sdk.Context, delegationStates []delegationtype.DelegationStates) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), delegationtype.KeyPrefixRestakerDelegationInfo)
	for i := range delegationStates {
		singleElement := delegationStates[i]
		bz := k.cdc.MustMarshal(&singleElement.States)
		store.Set([]byte(singleElement.Key), bz)
	}
	return nil
}

func (k Keeper) IterateDelegations(ctx sdk.Context, iteratorPrefix []byte, opFunc delegationtype.DelegationOpFunc) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), delegationtype.KeyPrefixRestakerDelegationInfo)
	iterator := sdk.KVStorePrefixIterator(store, iteratorPrefix)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var amounts delegationtype.DelegationAmounts
		k.cdc.MustUnmarshal(iterator.Value(), &amounts)
		keys, err := delegationtype.ParseStakerAssetIDAndOperator(iterator.Key())
		if err != nil {
			return err
		}
		isBreak, err := opFunc(keys, &amounts)
		if err != nil {
			return err
		}
		if isBreak {
			break
		}
	}
	return nil
}

// IterateDelegationsForStakerAndAsset processes all operations
// that require iterating over delegations for a specified staker and asset.
func (k Keeper) IterateDelegationsForStakerAndAsset(ctx sdk.Context, stakerID string, assetID string, opFunc delegationtype.DelegationOpFunc) error {
	return k.IterateDelegations(ctx, delegationtype.IteratorPrefixForStakerAsset(stakerID, assetID), opFunc)
}

func (k Keeper) IterateDelegationsForStaker(ctx sdk.Context, stakerID string, opFunc delegationtype.DelegationOpFunc) error {
	return k.IterateDelegations(ctx, []byte(stakerID), opFunc)
}

// TotalDelegatedAmountForStakerAsset query the total delegation amount of the specified staker and asset.
// It needs to be calculated from the share and amount of the asset pool.
func (k Keeper) TotalDelegatedAmountForStakerAsset(ctx sdk.Context, stakerID string, assetID string) (amount sdkmath.Int, err error) {
	amount = sdkmath.ZeroInt()
	opFunc := func(keys *delegationtype.SingleDelegationInfoReq, amounts *delegationtype.DelegationAmounts) (bool, error) {
		if amounts.UndelegatableShare.IsZero() {
			return false, nil
		}
		opAccAddr := sdk.MustAccAddressFromBech32(keys.GetOperatorAddr())
		// get the asset state of operator
		operatorAsset, err := k.assetsKeeper.GetOperatorSpecifiedAssetInfo(ctx, opAccAddr, assetID)
		if err != nil {
			return true, err
		}
		singleAmount, err := TokensFromShares(amounts.UndelegatableShare, operatorAsset.TotalShare, operatorAsset.TotalAmount)
		if err != nil {
			return true, err
		}
		amount = amount.Add(singleAmount)
		return false, nil
	}
	err = k.IterateDelegationsForStakerAndAsset(ctx, stakerID, assetID, opFunc)
	return amount, err
}

// AllDelegatedInfoForStakerAsset returns all delegated information of the specified staker and asset
// the key of return value is the operator address, and the value is the asset amount.
func (k *Keeper) AllDelegatedInfoForStakerAsset(ctx sdk.Context, stakerID string, assetID string) (map[string]sdkmath.Int, error) {
	ret := make(map[string]sdkmath.Int)
	opFunc := func(keys *delegationtype.SingleDelegationInfoReq, amounts *delegationtype.DelegationAmounts) (bool, error) {
		opAccAddr := sdk.MustAccAddressFromBech32(keys.GetOperatorAddr())
		// get the asset state of operator
		operatorAsset, err := k.assetsKeeper.GetOperatorSpecifiedAssetInfo(ctx, opAccAddr, assetID)
		if err != nil {
			return true, err
		}
		singleAmount, err := TokensFromShares(amounts.UndelegatableShare, operatorAsset.TotalShare, operatorAsset.TotalAmount)
		if err != nil {
			return true, err
		}
		ret[keys.OperatorAddr] = singleAmount
		return false, nil
	}
	err := k.IterateDelegationsForStakerAndAsset(ctx, stakerID, assetID, opFunc)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// UpdateDelegationState is used to update the staker's asset amount that is delegated to a specified operator.
// Compared to `UpdateStakerDelegationTotalAmount`,they use the same kv store, but in this function the store key needs to add the operator address as a suffix.
func (k Keeper) UpdateDelegationState(ctx sdk.Context, stakerID, assetID, opAddr string, deltaAmounts *delegationtype.DeltaDelegationAmounts) (bool, error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), delegationtype.KeyPrefixRestakerDelegationInfo)
	// todo: think about the difference between init and update in future
	shareIsZero := false
	if deltaAmounts == nil {
		return false, errorsmod.Wrap(
			assetstype.ErrInputPointerIsNil,
			fmt.Sprintf("UpdateDelegationState opAddr:%v,deltaAmounts:%v", opAddr, deltaAmounts),
		)
	}
	// check operator address validation
	_, err := sdk.AccAddressFromBech32(opAddr)
	if err != nil {
		return shareIsZero, delegationtype.ErrOperatorAddrIsNotAccAddr
	}
	singleStateKey := assetstype.GetJoinedStoreKey(stakerID, assetID, opAddr)
	delegationState := delegationtype.DelegationAmounts{
		WaitUndelegationAmount: sdkmath.ZeroInt(),
		UndelegatableShare:     sdkmath.LegacyZeroDec(),
	}

	value := store.Get(singleStateKey)
	if value != nil {
		k.cdc.MustUnmarshal(value, &delegationState)
	}
	err = assetstype.UpdateAssetValue(&delegationState.WaitUndelegationAmount, &deltaAmounts.WaitUndelegationAmount)
	if err != nil {
		return shareIsZero, errorsmod.Wrap(err, "UpdateDelegationState WaitUndelegationAmount error")
	}

	err = assetstype.UpdateAssetDecValue(&delegationState.UndelegatableShare, &deltaAmounts.UndelegatableShare)
	if err != nil {
		return shareIsZero, errorsmod.Wrap(err, "UpdateDelegationState UndelegatableShare error")
	}

	if delegationState.UndelegatableShare.IsZero() {
		shareIsZero = true
	}

	// todo: should we delete the delegation state if both the share and the WaitUndelegationAmount are zero
	// to reduce the state storage?

	// save single operator delegation state
	bz := k.cdc.MustMarshal(&delegationState)
	store.Set(singleStateKey, bz)

	return shareIsZero, nil
}

// GetSingleDelegationInfo query the staker's asset amount that has been delegated to the specified operator.
func (k *Keeper) GetSingleDelegationInfo(ctx sdk.Context, stakerID, assetID, operatorAddr string) (*delegationtype.DelegationAmounts, error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), delegationtype.KeyPrefixRestakerDelegationInfo)
	singleStateKey := assetstype.GetJoinedStoreKey(stakerID, assetID, operatorAddr)
	delegationState := delegationtype.DelegationAmounts{}
	value := store.Get(singleStateKey)
	if value == nil {
		return nil, delegationtype.ErrNoKeyInTheStore.Wrapf("QuerySingleDelegationInfo: key is %s", singleStateKey)
	}
	k.cdc.MustUnmarshal(value, &delegationState)
	return &delegationState, nil
}

// GetDelegationInfo query the staker's asset info that has been delegated.
func (k *Keeper) GetDelegationInfo(ctx sdk.Context, stakerID, assetID string) (*delegationtype.QueryDelegationInfoResponse, error) {
	var ret delegationtype.QueryDelegationInfoResponse
	ret.DelegationInfos = make(map[string]*delegationtype.DelegationAmounts)
	opFunc := func(keys *delegationtype.SingleDelegationInfoReq, amounts *delegationtype.DelegationAmounts) (bool, error) {
		ret.DelegationInfos[keys.OperatorAddr] = amounts
		return false, nil
	}
	err := k.IterateDelegationsForStakerAndAsset(ctx, stakerID, assetID, opFunc)
	if err != nil {
		return nil, err
	}
	return &ret, nil
}

func (k *Keeper) AppendStakerForOperator(ctx sdk.Context, operator, assetID, stakerID string) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), delegationtype.KeyPrefixStakersByOperator)
	Key := assetstype.GetJoinedStoreKey(operator, assetID)
	stakers := delegationtype.StakerList{}
	value := store.Get(Key)
	if value != nil {
		k.cdc.MustUnmarshal(value, &stakers)
	}
	for _, v := range stakers.Stakers {
		if v == stakerID {
			return nil
		}
	}
	stakers.Stakers = append(stakers.Stakers, stakerID)
	bz := k.cdc.MustMarshal(&stakers)
	store.Set(Key, bz)
	return nil
}

func (k *Keeper) DeleteStakerForOperator(ctx sdk.Context, operator, assetID, stakerID string) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), delegationtype.KeyPrefixStakersByOperator)
	Key := assetstype.GetJoinedStoreKey(operator, assetID)
	stakers := delegationtype.StakerList{}
	if !store.Has(Key) {
		return delegationtype.ErrNoKeyInTheStore
	}
	value := store.Get(Key)
	k.cdc.MustUnmarshal(value, &stakers)
	for i, v := range stakers.Stakers {
		if v == stakerID {
			stakers.Stakers = append(stakers.Stakers[:i], stakers.Stakers[i+1:]...)
			break
		}
	}
	bz := k.cdc.MustMarshal(&stakers)
	store.Set(Key, bz)
	return nil
}

func (k *Keeper) DeleteStakersListForOperator(ctx sdk.Context, operator, assetID string) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), delegationtype.KeyPrefixStakersByOperator)
	Key := assetstype.GetJoinedStoreKey(operator, assetID)
	if !store.Has(Key) {
		return delegationtype.ErrNoKeyInTheStore
	}
	store.Delete(Key)
	return nil
}

func (k Keeper) HasStakerList(ctx sdk.Context, operator, assetID string) bool {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), delegationtype.KeyPrefixStakersByOperator)
	Key := assetstype.GetJoinedStoreKey(operator, assetID)
	return store.Has(Key)
}

func (k Keeper) GetStakersByOperator(ctx sdk.Context, operator, assetID string) (delegationtype.StakerList, error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), delegationtype.KeyPrefixStakersByOperator)
	Key := assetstype.GetJoinedStoreKey(operator, assetID)
	value := store.Get(Key)
	if value == nil {
		return delegationtype.StakerList{}, delegationtype.ErrNoKeyInTheStore
	}
	stakerList := delegationtype.StakerList{}
	k.cdc.MustUnmarshal(value, &stakerList)
	return stakerList, nil
}

func (k Keeper) AllStakerList(ctx sdk.Context) (stakerList []delegationtype.StakersByOperator, err error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), delegationtype.KeyPrefixStakersByOperator)
	iterator := sdk.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	ret := make([]delegationtype.StakersByOperator, 0)
	for ; iterator.Valid(); iterator.Next() {
		var stakers delegationtype.StakerList
		k.cdc.MustUnmarshal(iterator.Value(), &stakers)
		ret = append(ret, delegationtype.StakersByOperator{
			Key:     string(iterator.Key()),
			Stakers: stakers.Stakers,
		})
	}
	return ret, nil
}

func (k Keeper) SetAllStakerList(ctx sdk.Context, stakersByOperator []delegationtype.StakersByOperator) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), delegationtype.KeyPrefixStakersByOperator)
	for i := range stakersByOperator {
		singleElement := stakersByOperator[i]
		bz := k.cdc.MustMarshal(&delegationtype.StakerList{Stakers: singleElement.Stakers})
		store.Set([]byte(singleElement.Key), bz)
	}
	return nil
}

func (k *Keeper) SetStakerShareToZero(ctx sdk.Context, operator, assetID string, stakerList delegationtype.StakerList) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), delegationtype.KeyPrefixRestakerDelegationInfo)
	for _, stakerID := range stakerList.Stakers {
		singleStateKey := assetstype.GetJoinedStoreKey(stakerID, assetID, operator)
		value := store.Get(singleStateKey)
		if value != nil {
			// TODO: check if pendingUndelegation==0 => just delete this item instead of update share to zero, otherwise this item will be left in the storage forever with zero value
			delegationState := delegationtype.DelegationAmounts{}
			k.cdc.MustUnmarshal(value, &delegationState)
			delegationState.UndelegatableShare = sdkmath.LegacyZeroDec()
			bz := k.cdc.MustMarshal(&delegationState)
			store.Set(singleStateKey, bz)
		}
	}
	return nil
}

// DelegationStateByOperatorAssets get the specified assets state delegated to the specified operator
// assetsFilter: assetID->nil, it's used to filter the specified assets
// the first return value is a nested map, its type is: stakerID->assetID->DelegationAmounts
// It means all delegation information related to the specified operator and filtered by the specified asset IDs
func (k Keeper) DelegationStateByOperatorAssets(ctx sdk.Context, operatorAddr string, assetsFilter map[string]interface{}) (map[string]map[string]delegationtype.DelegationAmounts, error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), delegationtype.KeyPrefixRestakerDelegationInfo)
	iterator := sdk.KVStorePrefixIterator(store, nil)
	defer iterator.Close()

	ret := make(map[string]map[string]delegationtype.DelegationAmounts, 0)
	for ; iterator.Valid(); iterator.Next() {
		var amounts delegationtype.DelegationAmounts
		k.cdc.MustUnmarshal(iterator.Value(), &amounts)
		keys, err := assetstype.ParseJoinedKey(iterator.Key())
		if err != nil {
			return nil, err
		}
		if len(keys) != 3 {
			continue
		}
		restakerID, assetID, findOperatorAddr := keys[0], keys[1], keys[2]
		if operatorAddr != findOperatorAddr {
			continue
		}
		_, assetIDExist := assetsFilter[assetID]
		_, restakerIDExist := ret[restakerID]
		if assetIDExist {
			if !restakerIDExist {
				ret[restakerID] = make(map[string]delegationtype.DelegationAmounts)
			}
			ret[restakerID][assetID] = amounts
		}
	}
	return ret, nil
}

func (k *Keeper) SetAssociatedOperator(ctx sdk.Context, stakerID, operatorAddr string) error {
	_, err := sdk.AccAddressFromBech32(operatorAddr)
	if err != nil {
		return delegationtype.ErrOperatorAddrIsNotAccAddr
	}
	store := prefix.NewStore(ctx.KVStore(k.storeKey), delegationtype.KeyPrefixAssociatedOperatorByStaker)
	store.Set([]byte(stakerID), []byte(operatorAddr))
	return nil
}

func (k *Keeper) DeleteAssociatedOperator(ctx sdk.Context, stakerID string) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), delegationtype.KeyPrefixAssociatedOperatorByStaker)
	store.Delete([]byte(stakerID))
	return nil
}

func (k *Keeper) GetAssociatedOperator(ctx sdk.Context, stakerID string) (string, error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), delegationtype.KeyPrefixAssociatedOperatorByStaker)
	value := store.Get([]byte(stakerID))
	if value != nil {
		return string(value), nil
	}
	return "", nil
}

func (k *Keeper) GetAllAssociations(ctx sdk.Context) ([]delegationtype.StakerToOperator, error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), delegationtype.KeyPrefixAssociatedOperatorByStaker)
	iterator := sdk.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	ret := make([]delegationtype.StakerToOperator, 0)
	for ; iterator.Valid(); iterator.Next() {
		ret = append(ret, delegationtype.StakerToOperator{
			StakerID: string(iterator.Key()),
			Operator: string(iterator.Value()),
		})
	}
	return ret, nil
}
