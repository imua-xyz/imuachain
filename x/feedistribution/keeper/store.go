package keeper

import (
	assetstype "github.com/ExocoreNetwork/exocore/x/assets/types"
	feedistributiontypes "github.com/ExocoreNetwork/exocore/x/feedistribution/types"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/ethereum/go-ethereum/common"
)

func (k Keeper) MarkStakeChangeDelegations(ctx sdk.Context, stakerID, assetID string, operator sdk.AccAddress) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), feedistributiontypes.KeyPrefixStakeChangeDelegations)
	delegationKey := assetstype.GetJoinedStoreKey(stakerID, assetID, operator.String())
	delegationKeys := &feedistributiontypes.StakeChangeDelegations{
		DelegationKeys: make([]string, 0),
	}
	// The reason for marking delegations with stake changes for all epochs instead of only the impactful
	// epochs is that we need to update the operator’s period whenever the delegated stake changes,
	// regardless of whether the operator is serving any AVSs.
	// This is because the reward distribution for a restaker might not occur during the opting-in period.
	// For example, the staker might delegate additional stake, triggering the reward distribution lazily
	// after the operator has opted out.
	// If we don’t update the period for operators who have opted out of an AVS, the reward calculation
	// cannot correctly determine the stake and reward ratio for a staker. This is because the staker might
	// have delegated or undelegated tokens, altering the delegated stake during the opting-out period.
	allEpochs := k.epochsKeeper.AllEpochInfos(ctx)
	for _, epochInfo := range allEpochs {
		value := store.Get([]byte(epochInfo.Identifier))
		if value != nil {
			k.cdc.MustUnmarshal(value, delegationKeys)
		}
		delegationKeys.AppendUniqueDelegationKey(string(delegationKey))
		bz := k.cdc.MustMarshal(delegationKeys)
		store.Set([]byte(epochInfo.Identifier), bz)
	}
	return nil
}

func (k Keeper) DeleteStakeChangeDelegations(ctx sdk.Context, epochIdentifier string) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), feedistributiontypes.KeyPrefixStakeChangeDelegations)
	store.Delete([]byte(epochIdentifier))
	return nil
}

// SetAVSFeePool : set the fee pool distribution info for AVS
func (k Keeper) SetAVSFeePool(ctx sdk.Context, avsAddr string, feePool types.FeePool) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), feedistributiontypes.KeyPrefixFeePools)
	b := k.cdc.MustMarshal(&feePool)
	store.Set(common.HexToAddress(avsAddr).Bytes(), b)
	return nil
}

// GetAVSFeePool : get the global fee pool distribution info
func (k Keeper) GetAVSFeePool(ctx sdk.Context, avsAddr string) (feePool types.FeePool, err error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), feedistributiontypes.KeyPrefixFeePools)
	b := store.Get(common.HexToAddress(avsAddr).Bytes())
	if b == nil {
		return types.FeePool{}, feedistributiontypes.ErrNoKeyInTheStore.Wrapf("GetAVSFeePool, avsAddr:%s", avsAddr)
	}
	fp := types.FeePool{}
	k.cdc.MustUnmarshal(b, &fp)
	return fp, nil
}

// HasAVSFeePool : check whether the avs fee pool exists.
func (k Keeper) HasAVSFeePool(ctx sdk.Context, avsAddr string) bool {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), feedistributiontypes.KeyPrefixFeePools)
	return store.Has(common.HexToAddress(avsAddr).Bytes())
}

// UpdateAVSCommunityPool : increase or decrease the rewards of AVS community pool
// the isIncrease flag is used to indicate whether the update is an increase or a decrease
func (k Keeper) UpdateAVSCommunityPool(ctx sdk.Context, avsAddr string, isIncrease bool, rewards sdk.DecCoins) error {
	if len(rewards) == 0 {
		return nil
	}
	// set the initialized value
	feePool := types.FeePool{
		CommunityPool: make([]sdk.DecCoin, 0),
	}
	var err error
	if k.HasAVSFeePool(ctx, avsAddr) {
		feePool, err = k.GetAVSFeePool(ctx, avsAddr)
		if err != nil {
			return err
		}
	}
	if isIncrease {
		feePool.CommunityPool = feePool.CommunityPool.Add(rewards...)
	} else {
		var negative bool
		feePool.CommunityPool, negative = feePool.CommunityPool.SafeSub(rewards)
		if negative {
			return feedistributiontypes.ErrNegativeCoinAmount.Wrapf("UpdateAVSCommunityPool,avsAddr:%s", avsAddr)
		}
	}

	err = k.SetAVSFeePool(ctx, avsAddr, feePool)
	if err != nil {
		return err
	}
	return nil
}

// SetOperatorAccumulatedCommission : set accumulated commission for the avs and operator
func (k Keeper) SetOperatorAccumulatedCommission(ctx sdk.Context, operator, avsAddr string, commission feedistributiontypes.OperatorAccumulatedCommission) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), feedistributiontypes.KeyPrefixOperatorAccumulatedCommission)
	var bz []byte

	if commission.Commission.IsZero() {
		bz = k.cdc.MustMarshal(&feedistributiontypes.OperatorAccumulatedCommission{})
	} else {
		bz = k.cdc.MustMarshal(&commission)
	}

	key := assetstype.GetJoinedStoreKey(operator, avsAddr)
	store.Set(key, bz)
	return nil
}

// GetOperatorAccumulatedCommission : get the accumulated commission for the avs and operator
func (k Keeper) GetOperatorAccumulatedCommission(ctx sdk.Context, operator, avsAddr string) (feedistributiontypes.OperatorAccumulatedCommission, error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), feedistributiontypes.KeyPrefixOperatorAccumulatedCommission)
	key := assetstype.GetJoinedStoreKey(operator, avsAddr)
	b := store.Get(key)
	if b == nil {
		return feedistributiontypes.OperatorAccumulatedCommission{}, feedistributiontypes.ErrNoKeyInTheStore.Wrapf("GetOperatorAccumulatedCommission, operator:%s,avsAddr:%s", operator, avsAddr)
	}
	commission := feedistributiontypes.OperatorAccumulatedCommission{}
	k.cdc.MustUnmarshal(b, &commission)
	return commission, nil
}

// HasOperatorAccumulatedCommission : check whether the accumulated commission for the avs and operator exists
func (k Keeper) HasOperatorAccumulatedCommission(ctx sdk.Context, operator, avsAddr string) bool {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), feedistributiontypes.KeyPrefixOperatorAccumulatedCommission)
	key := assetstype.GetJoinedStoreKey(operator, avsAddr)
	return store.Has(key)
}

// UpdateOperatorAccumulatedCommission : increase or decrease the commission for the avs and operator
// the isIncrease flag is used to indicate whether the update is an increase or a decrease
func (k Keeper) UpdateOperatorAccumulatedCommission(ctx sdk.Context, operator, avsAddr string, isIncrease bool, deltaCommission sdk.DecCoins) error {
	if len(deltaCommission) == 0 {
		return nil
	}
	// set the initialized value
	commission := feedistributiontypes.OperatorAccumulatedCommission{
		Commission: make([]sdk.DecCoin, 0),
	}
	var err error
	if k.HasOperatorAccumulatedCommission(ctx, operator, avsAddr) {
		commission, err = k.GetOperatorAccumulatedCommission(ctx, operator, avsAddr)
		if err != nil {
			return err
		}
	}

	if isIncrease {
		commission.Commission = commission.Commission.Add(deltaCommission...)
	} else {
		var negative bool
		commission.Commission, negative = commission.Commission.SafeSub(deltaCommission)
		if negative {
			return feedistributiontypes.ErrNegativeCoinAmount.Wrapf("UpdateOperatorAccumulatedCommission,operator:%s,avsAddr:%s", operator, avsAddr)
		}
	}

	err = k.SetOperatorAccumulatedCommission(ctx, operator, avsAddr, commission)
	if err != nil {
		return err
	}
	return nil
}

// SetOperatorOutstandingRewards : set outstanding rewards for the avs and operator
func (k Keeper) SetOperatorOutstandingRewards(ctx sdk.Context, operator, avsAddr string, rewards feedistributiontypes.OperatorOutstandingRewards) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), feedistributiontypes.KeyPrefixOperatorOutstandingRewards)
	var bz []byte

	if rewards.Rewards.IsZero() {
		bz = k.cdc.MustMarshal(&feedistributiontypes.OperatorOutstandingRewards{})
	} else {
		bz = k.cdc.MustMarshal(&rewards)
	}

	key := assetstype.GetJoinedStoreKey(operator, avsAddr)
	store.Set(key, bz)
	return nil
}

// GetOperatorOutstandingRewards : get the outstanding rewards for the avs and operator
func (k Keeper) GetOperatorOutstandingRewards(ctx sdk.Context, operator, avsAddr string) (feedistributiontypes.OperatorOutstandingRewards, error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), feedistributiontypes.KeyPrefixOperatorOutstandingRewards)
	key := assetstype.GetJoinedStoreKey(operator, avsAddr)
	b := store.Get(key)
	if b == nil {
		return feedistributiontypes.OperatorOutstandingRewards{}, feedistributiontypes.ErrNoKeyInTheStore.Wrapf("GetOperatorOutstandingRewards, operator:%s,avsAddr:%s", operator, avsAddr)
	}
	rewards := feedistributiontypes.OperatorOutstandingRewards{}
	k.cdc.MustUnmarshal(b, &rewards)
	return rewards, nil
}

// HasOperatorOutstandingRewards : check whether the outstanding rewards for the avs and operator exists
func (k Keeper) HasOperatorOutstandingRewards(ctx sdk.Context, operator, avsAddr string) bool {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), feedistributiontypes.KeyPrefixOperatorOutstandingRewards)
	key := assetstype.GetJoinedStoreKey(operator, avsAddr)
	return store.Has(key)
}

// UpdateOperatorOutstandingRewards : increase or decrease the outstanding rewards for the avs and operator
// the isIncrease flag is used to indicate whether the update is an increase or a decrease
func (k Keeper) UpdateOperatorOutstandingRewards(ctx sdk.Context, operator, avsAddr string, isIncrease bool, deltaRewards sdk.DecCoins) error {
	if len(deltaRewards) == 0 {
		return nil
	}
	// set the initialized value
	rewards := feedistributiontypes.OperatorOutstandingRewards{
		Rewards: make([]sdk.DecCoin, 0),
	}
	var err error
	if k.HasOperatorOutstandingRewards(ctx, operator, avsAddr) {
		rewards, err = k.GetOperatorOutstandingRewards(ctx, operator, avsAddr)
		if err != nil {
			return err
		}
	}

	if isIncrease {
		rewards.Rewards = rewards.Rewards.Add(deltaRewards...)
	} else {
		var negative bool
		rewards.Rewards, negative = rewards.Rewards.SafeSub(deltaRewards)
		if negative {
			return feedistributiontypes.ErrNegativeCoinAmount.Wrapf("UpdateOperatorOutstandingRewards,operator:%s,avsAddr:%s", operator, avsAddr)
		}
	}

	err = k.SetOperatorOutstandingRewards(ctx, operator, avsAddr, rewards)
	if err != nil {
		return err
	}
	return nil
}

// SetOperatorCurrentRewards : set current rewards for the specific operator, epochIdentifier and assetID
func (k Keeper) SetOperatorCurrentRewards(ctx sdk.Context, operator, assetID, epochIdentifier string, rewards feedistributiontypes.OperatorCurrentRewards) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), feedistributiontypes.KeyPrefixOperatorCurrentRewards)
	var bz []byte
	bz = k.cdc.MustMarshal(&rewards)
	key := assetstype.GetJoinedStoreKey(operator, assetID, epochIdentifier)
	store.Set(key, bz)
	return nil
}

// GetOperatorCurrentRewards : get the current rewards for the specific operator, epochIdentifier and assetID
func (k Keeper) GetOperatorCurrentRewards(ctx sdk.Context, operator, assetID, epochIdentifier string) (feedistributiontypes.OperatorCurrentRewards, error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), feedistributiontypes.KeyPrefixOperatorCurrentRewards)
	key := assetstype.GetJoinedStoreKey(operator, assetID, epochIdentifier)
	b := store.Get(key)
	if b == nil {
		return feedistributiontypes.OperatorCurrentRewards{}, feedistributiontypes.ErrNoKeyInTheStore.Wrapf("GetOperatorCurrentRewards, operator:%s,assetID:%s,epochIdentifier:%s", operator, assetID, epochIdentifier)
	}
	rewards := feedistributiontypes.OperatorCurrentRewards{}
	k.cdc.MustUnmarshal(b, &rewards)
	return rewards, nil
}

// HasOperatorCurrentRewards : check whether the current rewards for the specific operator, epochIdentifier
// and assetID exists.
func (k Keeper) HasOperatorCurrentRewards(ctx sdk.Context, operator, assetID, epochIdentifier string) bool {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), feedistributiontypes.KeyPrefixOperatorCurrentRewards)
	key := assetstype.GetJoinedStoreKey(operator, assetID, epochIdentifier)
	return store.Has(key)
}

// UpdateOperatorCurrentRewards : increase or decrease the current rewards for the specific operator,
// epochIdentifier and assetID. The isIncrease flag is used to indicate whether the update is an
// increase or a decrease
func (k Keeper) UpdateOperatorCurrentRewards(ctx sdk.Context, operator, assetID, epochIdentifier string, isIncrease bool, deltaRewards feedistributiontypes.CommonAVSRewardData) error {
	if len(deltaRewards.Rewards) == 0 {
		return nil
	}
	// set the initialized value
	rewards := feedistributiontypes.OperatorCurrentRewards{
		Rewards: make([]*feedistributiontypes.CommonAVSRewardData, 0),
		// period starts from 1.
		Period: 1,
	}
	var err error
	if k.HasOperatorCurrentRewards(ctx, operator, assetID, epochIdentifier) {
		rewards, err = k.GetOperatorCurrentRewards(ctx, operator, assetID, epochIdentifier)
		if err != nil {
			return err
		}
	}
	err = rewards.UpdateReward(isIncrease, deltaRewards)
	if err != nil {
		return err
	}

	err = k.SetOperatorCurrentRewards(ctx, operator, assetID, epochIdentifier, rewards)
	if err != nil {
		return err
	}
	return nil
}

// IncreasePeriodForOperator : increase the period for the specific operator, assetID and epoch identifier
func (k Keeper) IncreasePeriodForOperator(ctx sdk.Context, operator, assetID, epochIdentifier string) error {
	rewards, err := k.GetOperatorCurrentRewards(ctx, operator, assetID, epochIdentifier)
	if err != nil {
		return err
	}
	rewards.Period += 1
	return k.SetOperatorCurrentRewards(ctx, operator, assetID, epochIdentifier, rewards)
}
