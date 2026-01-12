package keeper

import (
	"strings"

	"github.com/imua-xyz/imuachain/utils"

	assetstype "github.com/imua-xyz/imuachain/x/assets/types"

	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	operatortypes "github.com/imua-xyz/imuachain/x/operator/types"
)

// UpdateOperatorSlashInfo This is a function to store the slash info related to an operator
// The stored state is: operator + '/' + AVSAddr + '/' + slashId -> OperatorSlashInfo
// Now this function will be called by `slash` function implemented in 'state_update.go' when there is a slash event occurs.
func (k *Keeper) UpdateOperatorSlashInfo(ctx sdk.Context, operatorAddr, avsAddr, slashID string, slashInfo operatortypes.OperatorSlashInfo) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), operatortypes.KeyPrefixOperatorSlashInfo)

	// check operator address validation
	_, err := sdk.AccAddressFromBech32(operatorAddr)
	if err != nil {
		return assetstype.ErrInvalidOperatorAddr
	}
	// check the validation of slash info
	slashContract, err := k.avsKeeper.GetAVSSlashContract(ctx, avsAddr)
	if err != nil {
		return err
	}
	if slashInfo.SlashContract != slashContract {
		return operatortypes.ErrSlashInfo.Wrapf("err slashContract:%s, stored contract:%s", slashInfo.SlashContract, slashContract)
	}
	if slashInfo.EventHeight > slashInfo.SubmittedHeight {
		return operatortypes.ErrSlashInfo.Wrapf("err SubmittedHeight:%v,EventHeight:%v", slashInfo.SubmittedHeight, slashInfo.EventHeight)
	}

	if slashInfo.SlashProportion.IsNil() || slashInfo.SlashProportion.IsNegative() || slashInfo.SlashProportion.GT(sdkmath.LegacyNewDec(1)) {
		return operatortypes.ErrSlashInfo.Wrapf("err SlashProportion:%v", slashInfo.SlashProportion)
	}

	// save single operator delegation state
	bz := k.cdc.MustMarshal(&slashInfo)
	slashInfoKey := utils.GetJoinedStoreKey(operatorAddr, strings.ToLower(avsAddr), slashID)
	store.Set(slashInfoKey, bz)
	// TODO: add an event for the slash info
	return nil
}

// GetOperatorSlashInfo This is a function to retrieve the slash info related to an operator
// Now this function hasn't been called. In the future, it might be called by the grpc query.
// Additionally, it might be used when implementing the veto function
func (k *Keeper) GetOperatorSlashInfo(ctx sdk.Context, avsAddr, operatorAddr, slashID string) (changeState *operatortypes.OperatorSlashInfo, err error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), operatortypes.KeyPrefixOperatorSlashInfo)
	slashInfoKey := utils.GetJoinedStoreKey(operatorAddr, strings.ToLower(avsAddr), slashID)
	value := store.Get(slashInfoKey)
	if value == nil {
		return nil, operatortypes.ErrNoKeyInTheStore.Wrapf("GetOperatorSlashInfo: key is %s", slashInfoKey)
	}
	operatorSlashInfo := operatortypes.OperatorSlashInfo{}
	k.cdc.MustUnmarshal(value, &operatorSlashInfo)
	return &operatorSlashInfo, nil
}

// AllOperatorSlashInfo return all slash information for the specified operator and AVS
func (k *Keeper) AllOperatorSlashInfo(ctx sdk.Context, avsAddr, operatorAddr string) (map[string]*operatortypes.OperatorSlashInfo, error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), operatortypes.KeyPrefixOperatorSlashInfo)
	prefix := utils.GetJoinedStoreKey(operatorAddr, strings.ToLower(avsAddr))

	ret := make(map[string]*operatortypes.OperatorSlashInfo, 0)
	iterator := sdk.KVStorePrefixIterator(store, prefix)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		var slashInfo operatortypes.OperatorSlashInfo
		k.cdc.MustUnmarshal(iterator.Value(), &slashInfo)
		keys := utils.ParseJoinedKey(iterator.Key())
		ret[keys[2]] = &slashInfo
	}
	return ret, nil
}

// StoreSlashStakerShareSnapshot This is a function to store the slashed delegation share snapshot for slash veto
func (k *Keeper) StoreSlashStakerShareSnapshot(ctx sdk.Context, operatorAddr, assetID, slashID string) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), operatortypes.KeyPrefixSlashStakerShareSnapshot)
	// get all stakers related the asset pool
	stakerList, err := k.delegationKeeper.GetStakersByOperator(ctx, operatorAddr, assetID)
	if err != nil {
		return err
	}
	avsList := k.distributionKeeper.GetAVSListByRewardAssetID(ctx, assetID)

	for _, stakerID := range stakerList.Stakers {
		// get the staker's delegation share
		delegationInfo, err := k.delegationKeeper.GetSingleDelegationInfo(ctx, stakerID, assetID, operatorAddr)
		if err != nil {
			return err
		}
		stakerUndelegatableSharesSnapshot := &operatortypes.StakerUndelegatableSharesSnapshot{
			StakingUndelegatableShare: delegationInfo.UndelegatableShare,
		}
		if delegationInfo.RewardUndelegatableShare.IsPositive() {
			if len(avsList) == 0 {
				// This case shouldn't occur, because it means a reward asset isn't registered as a reward asset by any AVS,
				// but a staker still owns the undelegatable share of this reward.
				// If we want to support reward asset deregistration in the future, we should wait for all rewards
				// to be undelegated and claimed before deregistering an AVS reward. Currently, this functionality is not supported.
				return operatortypes.ErrStoreStakerShareSnapshot.Wrapf("rewardUndelegatableShare is positive, but avsList is empty, assetID:%s, stakerID:%s", assetID, stakerID)
			}
			rewardUndelegatableShareBreakdown, err := k.distributionKeeper.GetRewardUndelegatableShareBreakdown(ctx, stakerID, assetID, operatorAddr, avsList)
			if err != nil {
				return err
			}
			stakerUndelegatableSharesSnapshot.RewardUndelegatableShareBreakdown = rewardUndelegatableShareBreakdown
		}
		bz := k.cdc.MustMarshal(stakerUndelegatableSharesSnapshot)
		store.Set(utils.GetJoinedStoreKey(slashID, assetID, stakerID), bz)
	}
	return nil
}

// IterateSlashStakerShareSnapshot This is a function to iterate the slashed delegation share
// snapshot for slash veto.
func (k *Keeper) IterateSlashStakerShareSnapshot(
	ctx sdk.Context, slashID, assetID string,
	opFunc func(stakerID string, shares *operatortypes.StakerUndelegatableSharesSnapshot) (stop bool, err error),
) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), operatortypes.KeyPrefixSlashStakerShareSnapshot)
	iterator := sdk.KVStorePrefixIterator(store, utils.GetJoinedStoreKey(slashID, assetID))
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		var shares operatortypes.StakerUndelegatableSharesSnapshot
		k.cdc.MustUnmarshal(iterator.Value(), &shares)
		keys, err := utils.ParseJoinedKeyWithCount(iterator.Key(), 3)
		if err != nil {
			return err
		}
		stop, err := opFunc(keys[2], &shares)
		if err != nil {
			return err
		}
		if stop {
			break
		}
	}
	return nil
}

func (k *Keeper) SetAllSlashStakerShareSnapshot(ctx sdk.Context, stakerSlashShareSnapshots []operatortypes.StakerSlashShareSnapshot) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), operatortypes.KeyPrefixSlashStakerShareSnapshot)
	for i := range stakerSlashShareSnapshots {
		snapshot := stakerSlashShareSnapshots[i]
		bz := k.cdc.MustMarshal(&snapshot.Value)
		store.Set([]byte(snapshot.Key), bz)
	}
	return nil
}

func (k *Keeper) GetAllSlashStakerShareSnapshot(ctx sdk.Context) ([]operatortypes.StakerSlashShareSnapshot, error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), operatortypes.KeyPrefixSlashStakerShareSnapshot)
	iterator := sdk.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()
	ret := make([]operatortypes.StakerSlashShareSnapshot, 0)

	for ; iterator.Valid(); iterator.Next() {
		var snapshot operatortypes.StakerUndelegatableSharesSnapshot
		k.cdc.MustUnmarshal(iterator.Value(), &snapshot)
		ret = append(ret, operatortypes.StakerSlashShareSnapshot{
			Key: string(iterator.Key()),
			Value: snapshot,
		})
	}
	return ret, nil
}

func (k *Keeper) SetAllSlashStates(ctx sdk.Context, slashStates []operatortypes.OperatorSlashState) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), operatortypes.KeyPrefixOperatorSlashInfo)
	for i := range slashStates {
		state := slashStates[i]
		bz := k.cdc.MustMarshal(&state.Info)
		store.Set([]byte(state.Key), bz)
	}
	return nil
}

func (k *Keeper) GetAllSlashStates(ctx sdk.Context) ([]operatortypes.OperatorSlashState, error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), operatortypes.KeyPrefixOperatorSlashInfo)
	iterator := sdk.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	ret := make([]operatortypes.OperatorSlashState, 0)
	for ; iterator.Valid(); iterator.Next() {
		var slashInfo operatortypes.OperatorSlashInfo
		k.cdc.MustUnmarshal(iterator.Value(), &slashInfo)
		ret = append(ret, operatortypes.OperatorSlashState{
			Key:  string(iterator.Key()),
			Info: slashInfo,
		})
	}
	return ret, nil
}
