package keeper

import (
	"fmt"

	"github.com/imua-xyz/imuachain/utils"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	assetstype "github.com/imua-xyz/imuachain/x/assets/types"
	delegationtype "github.com/imua-xyz/imuachain/x/delegation/types"
)

var sentinelValue = []byte{1}

func (k Keeper) AllDelegationStates(ctx sdk.Context) (delegationStates []delegationtype.DelegationStates, err error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), delegationtype.KeyPrefixRestakerDelegationInfo)
	iterator := sdk.KVStorePrefixIterator(store, nil)
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
		// only used at genesis, so no events
	}
	return nil
}

func (k Keeper) IterateDelegations(ctx sdk.Context, iteratorPrefix []byte, opFunc delegationtype.DelegationOpFunc) error {
	if opFunc == nil {
		return delegationtype.ErrInvalidInputParameter.Wrapf("opFunc callback is nil")
	}
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
		// read-only, so no events
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
	if stakerID == "" {
		return delegationtype.ErrInvalidInputParameter.Wrapf("null stakerID")
	}
	if assetID == "" {
		return delegationtype.ErrInvalidInputParameter.Wrapf("null assetID")
	}
	return k.IterateDelegations(ctx, delegationtype.IteratorPrefixForStakerAsset(stakerID, assetID), opFunc)
}

func (k Keeper) IterateDelegationsForStaker(ctx sdk.Context, stakerID string, opFunc delegationtype.DelegationOpFunc) error {
	if stakerID == "" {
		return delegationtype.ErrInvalidInputParameter.Wrapf("null stakerID")
	}
	return k.IterateDelegations(ctx, []byte(stakerID), opFunc)
}

func (k Keeper) UndelegatableAmount(ctx sdk.Context, assetID, operator string, amounts *delegationtype.DelegationAmounts) (sdkmath.Int, sdkmath.Int, error) {
	opAccAddr, err := sdk.AccAddressFromBech32(operator)
	if err != nil {
		return sdkmath.ZeroInt(), sdkmath.ZeroInt(), delegationtype.ErrOperatorAddrIsNotAccAddr
	}
	// get the asset state of operator
	operatorAsset, err := k.assetsKeeper.GetOperatorSpecifiedAssetInfo(ctx, opAccAddr, assetID)
	if err != nil {
		return sdkmath.ZeroInt(), sdkmath.ZeroInt(), err
	}
	stakingAmount, err := TokensFromShares(amounts.UndelegatableShare, operatorAsset.TotalShare, operatorAsset.TotalAmount)
	if err != nil {
		return sdkmath.ZeroInt(), sdkmath.ZeroInt(), err
	}
	rewardAmount, err := TokensFromShares(amounts.RewardUndelegatableShare, operatorAsset.TotalShare, operatorAsset.TotalAmount)
	if err != nil {
		return sdkmath.ZeroInt(), sdkmath.ZeroInt(), err
	}
	return stakingAmount, rewardAmount, nil
}

// TotalDelegatedAmountForStakingAsset query the total delegation amount of the specified staker and staking asset.
// It needs to be calculated from the share and amount of the asset pool.
func (k Keeper) TotalDelegatedAmountForStakingAsset(ctx sdk.Context, stakerID string, assetID string) (amount sdkmath.Int, err error) {
	amount = sdkmath.ZeroInt()
	opFunc := func(keys *delegationtype.SingleDelegationInfoReq, amounts *delegationtype.DelegationAmounts) (bool, error) {
		if amounts.UndelegatableShare.IsZero() {
			return false, nil
		}
		stakingAssetAmount, _, err := k.UndelegatableAmount(ctx, assetID, keys.GetOperatorAddr(), amounts)
		if err != nil {
			return true, err
		}
		amount = amount.Add(stakingAssetAmount)
		return false, nil
	}
	// read-only, so no event
	err = k.IterateDelegationsForStakerAndAsset(ctx, stakerID, assetID, opFunc)
	return amount, err
}

// AllDelegatedInfoForStakerAsset returns all delegated information of the specified staker and staking asset
// the key of return value is the operator address, and the value is the asset amount.
func (k *Keeper) AllDelegatedInfoForStakerAsset(ctx sdk.Context, stakerID string, assetID string) (map[string]sdkmath.Int, error) {
	ret := make(map[string]sdkmath.Int)
	opFunc := func(keys *delegationtype.SingleDelegationInfoReq, amounts *delegationtype.DelegationAmounts) (bool, error) {
		stakingAssetAmount, _, err := k.UndelegatableAmount(ctx, assetID, keys.GetOperatorAddr(), amounts)
		if err != nil {
			return true, err
		}
		ret[keys.OperatorAddr] = stakingAssetAmount
		return false, nil
	}
	err := k.IterateDelegationsForStakerAndAsset(ctx, stakerID, assetID, opFunc)
	if err != nil {
		return nil, err
	}
	// not used so no event
	return ret, nil
}

// UpdateDelegationState is used to update the staker's asset amount that is delegated to a specified operator.
func (k Keeper) UpdateDelegationState(ctx sdk.Context, stakerID, assetID, opAddr string, deltaAmounts *delegationtype.DeltaDelegationAmounts) (bool, delegationtype.DelegationAmounts, error) {
	if deltaAmounts == nil {
		return false, delegationtype.DelegationAmounts{}, errorsmod.Wrap(
			assetstype.ErrInputPointerIsNil,
			fmt.Sprintf("UpdateDelegationState opAddr:%v,deltaAmounts:%v", opAddr, deltaAmounts),
		)
	}
	// check operator address validation
	_, err := sdk.AccAddressFromBech32(opAddr)
	if err != nil {
		return false, delegationtype.DelegationAmounts{}, delegationtype.ErrOperatorAddrIsNotAccAddr
	}
	store := prefix.NewStore(ctx.KVStore(k.storeKey), delegationtype.KeyPrefixRestakerDelegationInfo)
	// todo: think about the difference between init and update in future
	shareIsZero := false
	singleStateKey := utils.GetJoinedStoreKey(stakerID, assetID, opAddr)
	delegationState := delegationtype.DelegationAmounts{
		PendingUndelegationAmount:       sdkmath.ZeroInt(),
		UndelegatableShare:              sdkmath.LegacyZeroDec(),
		RewardPendingUndelegationAmount: sdkmath.ZeroInt(),
		RewardUndelegatableShare:        sdkmath.LegacyZeroDec(),
	}

	value := store.Get(singleStateKey)
	if value != nil {
		k.cdc.MustUnmarshal(value, &delegationState)
	}

	preState := delegationtype.DelegationAmounts{
		UndelegatableShare:              delegationState.UndelegatableShare.Clone(),
		PendingUndelegationAmount:       sdkmath.NewIntFromBigInt(delegationState.PendingUndelegationAmount.BigInt()),
		RewardUndelegatableShare:        delegationState.RewardUndelegatableShare.Clone(),
		RewardPendingUndelegationAmount: sdkmath.NewIntFromBigInt(delegationState.RewardPendingUndelegationAmount.BigInt()),
	}
	err = assetstype.UpdateAssetValue(&delegationState.PendingUndelegationAmount, &deltaAmounts.PendingUndelegationAmount)
	if err != nil {
		return shareIsZero, preState, errorsmod.Wrap(err, "UpdateDelegationState PendingUndelegationAmount error")
	}

	err = assetstype.UpdateAssetDecValue(&delegationState.UndelegatableShare, &deltaAmounts.UndelegatableShare)
	if err != nil {
		return shareIsZero, preState, errorsmod.Wrap(err, "UpdateDelegationState UndelegatableShare error")
	}

	err = assetstype.UpdateAssetValue(&delegationState.RewardPendingUndelegationAmount, &deltaAmounts.RewardPendingUndelegationAmount)
	if err != nil {
		return shareIsZero, preState, errorsmod.Wrap(err, "UpdateDelegationState RewardPendingUndelegationAmount error")
	}

	err = assetstype.UpdateAssetDecValue(&delegationState.RewardUndelegatableShare, &deltaAmounts.RewardUndelegatableShare)
	if err != nil {
		return shareIsZero, preState, errorsmod.Wrap(err, "UpdateDelegationState RewardUndelegatableShare error")
	}

	if delegationState.UndelegatableShare.Add(delegationState.RewardUndelegatableShare).IsZero() {
		shareIsZero = true
	}

	// todo: we should delete the delegation state if both the share and the PendingUndelegationAmount are zero
	// to reduce the state storage.
	// But the implementation might not be done here, because delegation removal needs to consider
	// whether a zero-amount delegation still has unclaimed rewards. In addition, the removal should
	// not be applied immediately when the amount becomes zero, since rewards are distributed per epoch.

	// save single operator delegation state
	bz := k.cdc.MustMarshal(&delegationState)
	store.Set(singleStateKey, bz)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			delegationtype.EventTypeDelegationStateUpdated,
			sdk.NewAttribute(delegationtype.AttributeKeyStakerID, stakerID),
			sdk.NewAttribute(delegationtype.AttributeKeyAssetID, assetID),
			sdk.NewAttribute(delegationtype.AttributeKeyOperatorAddr, opAddr),
			sdk.NewAttribute(delegationtype.AttributeKeyPendingUndelegationAmountDelta, deltaAmounts.PendingUndelegationAmount.String()),
			sdk.NewAttribute(delegationtype.AttributeKeyUndelegatableShareDelta, deltaAmounts.UndelegatableShare.String()),
			sdk.NewAttribute(delegationtype.AttributeKeyRewardUndelegationShareDelta, deltaAmounts.RewardUndelegatableShare.String()),
			sdk.NewAttribute(delegationtype.AttributeKeyRewardPendingUndelegationDelta, deltaAmounts.RewardPendingUndelegationAmount.String()),
		),
	)

	return shareIsZero, preState, nil
}

// GetSingleDelegationInfo query the staker's asset information that has been delegated to the specified operator.
func (k *Keeper) GetSingleDelegationInfo(ctx sdk.Context, stakerID, assetID, operatorAddr string) (*delegationtype.DelegationAmounts, error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), delegationtype.KeyPrefixRestakerDelegationInfo)
	singleStateKey := utils.GetJoinedStoreKey(stakerID, assetID, operatorAddr)
	delegationState := delegationtype.DelegationAmounts{}
	value := store.Get(singleStateKey)
	if value == nil {
		return nil, delegationtype.ErrNoKeyInTheStore.Wrapf("QuerySingleDelegationInfo: key is %s", singleStateKey)
	}
	k.cdc.MustUnmarshal(value, &delegationState)
	return &delegationState, nil
}

// GetDelegationInfoWithAmounts returns not only the staker's asset information delegated to the
// specified operator but also the delegated amounts calculated from shares for staking and reward asset.
func (k *Keeper) GetDelegationInfoWithAmounts(ctx sdk.Context, stakerID, assetID, operatorAddr string) (*delegationtype.DelegationAmounts, sdkmath.Int, sdkmath.Int, error) {
	delegationAmounts, err := k.GetSingleDelegationInfo(ctx, stakerID, assetID, operatorAddr)
	if err != nil {
		return nil, sdkmath.Int{}, sdkmath.Int{}, err
	}
	// calculate the maximum undelegatable amount
	stakingAssetAmount, rewardAssetAmount, err := k.UndelegatableAmount(ctx, assetID, operatorAddr, delegationAmounts)
	if err != nil {
		return nil, sdkmath.Int{}, sdkmath.Int{}, err
	}
	return delegationAmounts, stakingAssetAmount, rewardAssetAmount, nil
}

// GetDelegationInfo query the staker's asset info that has been delegated.
func (k *Keeper) GetDelegationInfo(ctx sdk.Context, stakerID, assetID string) (*delegationtype.QueryDelegationInfoResponse, error) {
	var ret delegationtype.QueryDelegationInfoResponse
	ret.DelegationInfos = make([]*delegationtype.DelegationInfoAndOperator, 0)
	opFunc := func(keys *delegationtype.SingleDelegationInfoReq, amounts *delegationtype.DelegationAmounts) (bool, error) {
		// calculate the maximum undelegatable amount
		stakingAssetAmount, rewardAssetAmount, err := k.UndelegatableAmount(ctx, assetID, keys.OperatorAddr, amounts)
		if err != nil {
			return false, err
		}
		ret.DelegationInfos = append(ret.DelegationInfos,
			&delegationtype.DelegationInfoAndOperator{
				Operator: keys.OperatorAddr,
				DelegationInfo: &delegationtype.SingleDelegationInfo{
					DelegationAmounts:            amounts,
					MaxUndelegatableAmount:       stakingAssetAmount,
					MaxUndelegatableRewardAmount: rewardAssetAmount,
				},
			},
		)
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
	key := utils.GetJoinedStoreKey(operator, assetID, stakerID)
	store.Set(key, sentinelValue)
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			delegationtype.EventTypeStakerAppended,
			sdk.NewAttribute(delegationtype.AttributeKeyStakerID, stakerID),
			sdk.NewAttribute(delegationtype.AttributeKeyAssetID, assetID),
			sdk.NewAttribute(delegationtype.AttributeKeyOperatorAddr, operator),
		),
	)
	return nil
}

func (k *Keeper) DeleteStakerForOperator(ctx sdk.Context, operator, assetID, stakerID string) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), delegationtype.KeyPrefixStakersByOperator)
	key := utils.GetJoinedStoreKey(operator, assetID, stakerID)
	if !store.Has(key) {
		return delegationtype.ErrNoKeyInTheStore
	}
	store.Delete(key)
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			delegationtype.EventTypeStakerRemoved,
			sdk.NewAttribute(delegationtype.AttributeKeyStakerID, stakerID),
			sdk.NewAttribute(delegationtype.AttributeKeyAssetID, assetID),
			sdk.NewAttribute(delegationtype.AttributeKeyOperatorAddr, operator),
		),
	)
	return nil
}

func (k *Keeper) DeleteStakersListForOperator(ctx sdk.Context, operator, assetID string) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), delegationtype.KeyPrefixStakersByOperator)
	key := utils.GetJoinedStoreKey(operator, assetID)
	// iterate over all stakers and delete them
	deleted := false
	iterator := sdk.KVStorePrefixIterator(store, key)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		store.Delete(iterator.Key())
		deleted = true
	}
	if !deleted {
		return delegationtype.ErrNoKeyInTheStore
	}
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			delegationtype.EventTypeAllStakersRemoved,
			sdk.NewAttribute(delegationtype.AttributeKeyAssetID, assetID),
			sdk.NewAttribute(delegationtype.AttributeKeyOperatorAddr, operator),
		),
	)
	return nil
}

func (k Keeper) HasStakerList(ctx sdk.Context, operator, assetID string) bool {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), delegationtype.KeyPrefixStakersByOperator)
	key := utils.GetJoinedStoreKey(operator, assetID)
	iterator := sdk.KVStorePrefixIterator(store, key)
	defer iterator.Close()
	return iterator.Valid()
}

func (k Keeper) GetStakersByOperator(
	ctx sdk.Context, operator, assetID string,
) (stakerList delegationtype.StakerList, err error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), delegationtype.KeyPrefixStakersByOperator)
	key := utils.GetJoinedStoreKey(operator, assetID)
	iterator := sdk.KVStorePrefixIterator(store, key)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		// parse the key to get the staker ID. this is done using the function
		// and not using the `+1:` approach in case the delimiter changes in
		// the future.
		keys, err := utils.ParseJoinedKeyWithCount(iterator.Key(), 3)
		if err != nil {
			return delegationtype.StakerList{}, err
		}
		stakerList.Stakers = append(stakerList.Stakers, keys[2])
	}
	// we do not return an error if the staker list is empty
	return stakerList, nil
}

func (k Keeper) AllStakerList(ctx sdk.Context) (keyList []string, err error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), delegationtype.KeyPrefixStakersByOperator)
	iterator := sdk.KVStorePrefixIterator(store, nil)
	defer iterator.Close()

	ret := make([]string, 0)
	for ; iterator.Valid(); iterator.Next() {
		ret = append(ret, string(iterator.Key()))
	}
	return ret, nil
}

func (k Keeper) SetAllStakerList(ctx sdk.Context, keyList []string) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), delegationtype.KeyPrefixStakersByOperator)
	for i := range keyList {
		store.Set([]byte(keyList[i]), sentinelValue)
	}
	// only used at genesis, so no events
	return nil
}

func (k *Keeper) SetStakerShareToZero(ctx sdk.Context, operator, assetID string, stakerList delegationtype.StakerList) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), delegationtype.KeyPrefixRestakerDelegationInfo)
	for _, stakerID := range stakerList.Stakers {
		singleStateKey := utils.GetJoinedStoreKey(stakerID, assetID, operator)
		value := store.Get(singleStateKey)
		if value != nil {
			// TODO: check if pendingUndelegation==0 => just delete this item instead of update share to zero, otherwise this item will be left in the storage forever with zero value
			delegationState := delegationtype.DelegationAmounts{}
			k.cdc.MustUnmarshal(value, &delegationState)

			undelegatableShareDelta := delegationState.UndelegatableShare.Neg()
			rewardUndelegatableShareDelta := delegationState.RewardUndelegatableShare.Neg()
			delegationState.UndelegatableShare = sdkmath.LegacyZeroDec()
			delegationState.RewardUndelegatableShare = sdkmath.LegacyZeroDec()

			bz := k.cdc.MustMarshal(&delegationState)
			store.Set(singleStateKey, bz)
			ctx.EventManager().EmitEvent(
				sdk.NewEvent(
					delegationtype.EventTypeDelegationStateUpdated,
					sdk.NewAttribute(delegationtype.AttributeKeyStakerID, stakerID),
					sdk.NewAttribute(delegationtype.AttributeKeyAssetID, assetID),
					sdk.NewAttribute(delegationtype.AttributeKeyOperatorAddr, operator),
					sdk.NewAttribute(delegationtype.AttributeKeyPendingUndelegationAmountDelta, sdk.ZeroInt().String()),
					sdk.NewAttribute(delegationtype.AttributeKeyUndelegatableShareDelta, undelegatableShareDelta.String()),
					sdk.NewAttribute(delegationtype.AttributeKeyRewardUndelegationShareDelta, rewardUndelegatableShareDelta.String()),
					sdk.NewAttribute(delegationtype.AttributeKeyRewardPendingUndelegationDelta, sdk.ZeroInt().String())),
			)
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
		keys, err := utils.ParseJoinedKeyWithCount(iterator.Key(), 3)
		if err != nil {
			return nil, err
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
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			delegationtype.EventTypeOperatorAssociated,
			sdk.NewAttribute(delegationtype.AttributeKeyStakerID, stakerID),
			sdk.NewAttribute(delegationtype.AttributeKeyOperatorAddr, operatorAddr),
		),
	)
	return nil
}

func (k *Keeper) DeleteAssociatedOperator(ctx sdk.Context, stakerID string) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), delegationtype.KeyPrefixAssociatedOperatorByStaker)
	store.Delete([]byte(stakerID))
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			delegationtype.EventTypeOperatorDisassociated,
			sdk.NewAttribute(delegationtype.AttributeKeyStakerID, stakerID),
		),
	)
	return nil
}

func (k *Keeper) GetAssociatedOperator(ctx sdk.Context, stakerID string) string {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), delegationtype.KeyPrefixAssociatedOperatorByStaker)
	value := store.Get([]byte(stakerID))
	if value != nil {
		return string(value)
	}
	return ""
}

func (k *Keeper) GetAssociatedStakers(ctx sdk.Context, operator string) ([]string, error) {
	if _, err := sdk.AccAddressFromBech32(operator); err != nil {
		return nil, delegationtype.ErrOperatorAddrIsNotAccAddr
	}
	store := prefix.NewStore(ctx.KVStore(k.storeKey), delegationtype.KeyPrefixAssociatedOperatorByStaker)
	iterator := sdk.KVStorePrefixIterator(store, nil)
	defer iterator.Close()

	// assuming we support 5 client chains and each chain has only one stakerID
	// associated with an operator, this capacity should be sufficient.
	// we can of course support more or less than that, but this is a good
	// starting point.
	// ideally, we should have a reverse lookup stored for this.
	ret := make([]string, 0, 5)
	for ; iterator.Valid(); iterator.Next() {
		if string(iterator.Value()) == operator {
			ret = append(ret, string(iterator.Key()))
		}
	}
	return ret, nil
}

func (k *Keeper) GetAllAssociations(ctx sdk.Context) ([]delegationtype.StakerToOperator, error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), delegationtype.KeyPrefixAssociatedOperatorByStaker)
	iterator := sdk.KVStorePrefixIterator(store, nil)
	defer iterator.Close()

	ret := make([]delegationtype.StakerToOperator, 0)
	for ; iterator.Valid(); iterator.Next() {
		ret = append(ret, delegationtype.StakerToOperator{
			StakerId: string(iterator.Key()),
			Operator: string(iterator.Value()),
		})
	}
	return ret, nil
}
