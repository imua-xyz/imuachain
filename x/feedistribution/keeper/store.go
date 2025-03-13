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
	_, impactfulEpochs, err := k.operatorKeeper.GetImpactfulEpochsAndAVSsForOperator(ctx, operator.String())
	if err != nil {
		return err
	}
	delegationKey := assetstype.GetJoinedStoreKey(stakerID, assetID, operator.String())
	delegationKeys := &feedistributiontypes.StakeChangeDelegations{
		DelegationKeys: make([]string, 0),
	}
	for _, epochIdentifier := range impactfulEpochs {
		value := store.Get([]byte(epochIdentifier))
		if value != nil {
			k.cdc.MustUnmarshal(value, delegationKeys)
		}
		delegationKeys.AppendUniqueDelegationKey(string(delegationKey))
		bz := k.cdc.MustMarshal(delegationKeys)
		store.Set([]byte(epochIdentifier), bz)
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

// AddRewardsToCommunityPool : add the rewards to community pool
func (k Keeper) AddRewardsToCommunityPool(ctx sdk.Context, avsAddr string, rewards sdk.DecCoins) error {
	feePool, err := k.GetAVSFeePool(ctx, avsAddr)
	if err != nil {
		return err
	}
	feePool.CommunityPool = feePool.CommunityPool.Add(rewards...)
	err = k.SetAVSFeePool(ctx, avsAddr, feePool)
	if err != nil {
		return err
	}
	return nil
}

// UpdateAVSCommunityPool : increase or decrease the rewards of AVS community pool
// the isIncrease flag is used to indicate whether the update is an increase or a decrease
func (k Keeper) UpdateAVSCommunityPool(ctx sdk.Context, avsAddr string, isIncrease bool, rewards sdk.DecCoins) error {
	feePool, err := k.GetAVSFeePool(ctx, avsAddr)
	if err != nil {
		return err
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

// UpdateOperatorAccumulatedCommission : increase or decrease the commission for the avs and operator
// the isIncrease flag is used to indicate whether the update is an increase or a decrease
func (k Keeper) UpdateOperatorAccumulatedCommission(ctx sdk.Context, operator, avsAddr string, isIncrease bool, deltaCommission sdk.DecCoins) error {
	commission, err := k.GetOperatorAccumulatedCommission(ctx, operator, avsAddr)
	if err != nil {
		return err
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

// UpdateOperatorOutstandingRewards : increase or decrease the outstanding rewards for the avs and operator
// the isIncrease flag is used to indicate whether the update is an increase or a decrease
func (k Keeper) UpdateOperatorOutstandingRewards(ctx sdk.Context, operator, avsAddr string, isIncrease bool, deltaRewards sdk.DecCoins) error {
	rewards, err := k.GetOperatorOutstandingRewards(ctx, operator, avsAddr)
	if err != nil {
		return err
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
