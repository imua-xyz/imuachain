package keeper

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/imua-xyz/imuachain/utils"
	assetstype "github.com/imua-xyz/imuachain/x/assets/types"
	feedistributiontypes "github.com/imua-xyz/imuachain/x/feedistribution/types"
	"github.com/imua-xyz/imuachain/x/operator/types"
)

func (k Keeper) SetStakeChangedDelegations(ctx sdk.Context, epochIdentifier, operator, assetID string,
	delegationChangeInfo feedistributiontypes.DelegationChangeInfo,
) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), feedistributiontypes.KeyPrefixStakeChangeDelegations)
	key := assetstype.GetJoinedStoreKey(epochIdentifier, operator, assetID)
	b := k.cdc.MustMarshal(&delegationChangeInfo)
	store.Set(key, b)
	return nil
}

func (k Keeper) GetStakeChangedDelegations(ctx sdk.Context, epochIdentifier, operator, assetID string) (feedistributiontypes.DelegationChangeInfo, error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), feedistributiontypes.KeyPrefixStakeChangeDelegations)
	key := assetstype.GetJoinedStoreKey(epochIdentifier, operator, assetID)
	b := store.Get(key)
	if b == nil {
		return feedistributiontypes.DelegationChangeInfo{}, feedistributiontypes.ErrNoKeyInTheStore.Wrapf(
			"GetStakeChangedDelegations, epochIdentifier:%s,operator:%s,assetID:%s", epochIdentifier, operator, assetID)
	}
	delegationChangeInfo := feedistributiontypes.DelegationChangeInfo{}
	k.cdc.MustUnmarshal(b, &delegationChangeInfo)
	return delegationChangeInfo, nil
}

func (k Keeper) HasStakeChangedDelegations(ctx sdk.Context, epochIdentifier, operator, assetID string) bool {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), feedistributiontypes.KeyPrefixStakeChangeDelegations)
	key := assetstype.GetJoinedStoreKey(epochIdentifier, operator, assetID)
	return store.Has(key)
}

func (k Keeper) DeleteStakeChangedDelegationsByEpoch(ctx sdk.Context, epochIdentifier string) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), feedistributiontypes.KeyPrefixStakeChangeDelegations)
	iterator := sdk.KVStorePrefixIterator(store, assetstype.GetJoinedStoreKeyForPrefix(epochIdentifier))
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		store.Delete(iterator.Key())
	}
	return nil
}

// IterateStakeChangedDelegations iterates over all delegations with changed stakes.
func (k *Keeper) IterateStakeChangedDelegations(ctx sdk.Context, isUpdate bool, iteratePrefix []byte,
	opFunc func(epochIdentifier, operator, assetID string, delegationChangeInfo *feedistributiontypes.DelegationChangeInfo,
	) (bool, error),
) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), feedistributiontypes.KeyPrefixStakeChangeDelegations)
	iterator := sdk.KVStorePrefixIterator(store, iteratePrefix)
	defer iterator.Close()

	updatedKeyValues := make([]utils.KeyValueT[*feedistributiontypes.DelegationChangeInfo], 0)
	for ; iterator.Valid(); iterator.Next() {
		var DelegationChangeInfo feedistributiontypes.DelegationChangeInfo
		keys, err := assetstype.ParseJoinedStoreKey(iterator.Key(), 3)
		if err != nil {
			return err
		}
		k.cdc.MustUnmarshal(iterator.Value(), &DelegationChangeInfo)
		isBreak, err := opFunc(keys[0], keys[1], keys[2], &DelegationChangeInfo)
		if err != nil {
			return err
		}
		if isBreak {
			break
		}
		if isUpdate {
			updatedKeyValues = append(updatedKeyValues, utils.KeyValueT[*feedistributiontypes.DelegationChangeInfo]{
				Key:   append([]byte(nil), iterator.Key()...),
				Value: &DelegationChangeInfo,
			})
		}
	}
	for _, updateKeyValue := range updatedKeyValues {
		bz := k.cdc.MustMarshal(updateKeyValue.Value)
		store.Set(updateKeyValue.Key, bz)
	}
	return nil
}

// SetAVSFeePool : set the fee pool distribution info for AVS
func (k Keeper) SetAVSFeePool(ctx sdk.Context, avsAddr string, feePool feedistributiontypes.FeePool) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), feedistributiontypes.KeyPrefixFeePools)
	b := k.cdc.MustMarshal(&feePool)
	store.Set(common.HexToAddress(avsAddr).Bytes(), b)
	return nil
}

// GetAVSFeePool : get the global fee pool distribution info
func (k Keeper) GetAVSFeePool(ctx sdk.Context, avsAddr string) (feePool feedistributiontypes.FeePool, err error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), feedistributiontypes.KeyPrefixFeePools)
	b := store.Get(common.HexToAddress(avsAddr).Bytes())
	if b == nil {
		return feedistributiontypes.FeePool{}, feedistributiontypes.ErrNoKeyInTheStore.Wrapf("GetAVSFeePool, avsAddr:%s", avsAddr)
	}
	fp := feedistributiontypes.FeePool{}
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
	feePool := feedistributiontypes.FeePool{
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

// SetOperatorCommission : set accumulated commission for the avs and operator
func (k Keeper) SetOperatorCommission(ctx sdk.Context, operator, avsAddr string, commission feedistributiontypes.OperatorCommission) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), feedistributiontypes.KeyPrefixOperatorCommission)
	bz := k.cdc.MustMarshal(&commission)
	key := assetstype.GetJoinedStoreKey(operator, avsAddr)
	store.Set(key, bz)
	return nil
}

// GetOperatorCommission : get the commission for the avs and operator
func (k Keeper) GetOperatorCommission(ctx sdk.Context, operator, avsAddr string) (feedistributiontypes.OperatorCommission, error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), feedistributiontypes.KeyPrefixOperatorCommission)
	key := assetstype.GetJoinedStoreKey(operator, avsAddr)
	b := store.Get(key)
	if b == nil {
		return feedistributiontypes.OperatorCommission{}, feedistributiontypes.ErrNoKeyInTheStore.Wrapf("GetOperatorCommission, operator:%s,avsAddr:%s", operator, avsAddr)
	}
	commission := feedistributiontypes.OperatorCommission{}
	k.cdc.MustUnmarshal(b, &commission)
	return commission, nil
}

// HasOperatorCommission : check whether the accumulated commission for the avs and operator exists
func (k Keeper) HasOperatorCommission(ctx sdk.Context, operator, avsAddr string) bool {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), feedistributiontypes.KeyPrefixOperatorCommission)
	key := assetstype.GetJoinedStoreKey(operator, avsAddr)
	return store.Has(key)
}

// IncreaseOperatorCommission : increase the commission for the avs and operator
func (k Keeper) IncreaseOperatorCommission(ctx sdk.Context, operator, avsAddr string, deltaCommission sdk.DecCoins) error {
	if len(deltaCommission) == 0 {
		return nil
	}
	// set the initialized value
	commission := feedistributiontypes.OperatorCommission{
		UnwithdrawnCommission: make([]sdk.DecCoin, 0),
		WithdrawnCommission:   make([]sdk.DecCoin, 0),
	}
	var err error
	if k.HasOperatorCommission(ctx, operator, avsAddr) {
		commission, err = k.GetOperatorCommission(ctx, operator, avsAddr)
		if err != nil {
			return err
		}
	}

	commission.UnwithdrawnCommission = commission.UnwithdrawnCommission.Add(deltaCommission...)
	err = k.SetOperatorCommission(ctx, operator, avsAddr, commission)
	if err != nil {
		return err
	}
	return nil
}

// SetOperatorOutstandingRewards : set outstanding avs rewards for the operator
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

// GetOperatorOutstandingRewards : get the outstanding avs rewards for the operator
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

// HasOperatorOutstandingRewards : check whether the outstanding avs rewards exists for the operator
func (k Keeper) HasOperatorOutstandingRewards(ctx sdk.Context, operator, avsAddr string) bool {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), feedistributiontypes.KeyPrefixOperatorOutstandingRewards)
	key := assetstype.GetJoinedStoreKey(operator, avsAddr)
	return store.Has(key)
}

// UpdateOperatorOutstandingRewards : increase or decrease the outstanding avs rewards for the operator
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
	bz := k.cdc.MustMarshal(&rewards)
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
	// It sets 1 as the start period and initializes the rewards slice as null.
	// set the initialized value
	rewards := feedistributiontypes.OperatorCurrentRewards{
		Rewards: make([]feedistributiontypes.CommonAVSRewardData, 0),
		// the period in current rewards starts from 1.
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
	rewards.Period++
	return k.SetOperatorCurrentRewards(ctx, operator, assetID, epochIdentifier, rewards)
}

// SetOperatorHistoricalRewards : set the historical rewards for the specific operator, epochIdentifier, assetID
// and period
func (k Keeper) SetOperatorHistoricalRewards(ctx sdk.Context, operator, assetID, epochIdentifier string,
	period uint64, historicalRewards feedistributiontypes.OperatorHistoricalRewards,
) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), feedistributiontypes.KeyPrefixOperatorHistoricalRewards)
	bz := k.cdc.MustMarshal(&historicalRewards)
	// this encoding ensures the key is ordered by period.
	periodHexStr := hexutil.Encode(sdk.Uint64ToBigEndian(period))
	key := assetstype.GetJoinedStoreKey(operator, assetID, epochIdentifier, periodHexStr)
	store.Set(key, bz)
	return nil
}

// DeleteOperatorHistoricalRewards : delete the historical rewards for the specific operator, epochIdentifier, assetID
// and period.
func (k Keeper) DeleteOperatorHistoricalRewards(ctx sdk.Context, operator, assetID, epochIdentifier string,
	period uint64,
) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), feedistributiontypes.KeyPrefixOperatorHistoricalRewards)
	// this encoding ensures the key is ordered by period.
	periodHexStr := hexutil.Encode(sdk.Uint64ToBigEndian(period))
	key := assetstype.GetJoinedStoreKey(operator, assetID, epochIdentifier, periodHexStr)
	store.Delete(key)
	return nil
}

// GetOperatorHistoricalReward : get the historical rewards for the specific operator, epochIdentifier, assetID
// and period.
func (k Keeper) GetOperatorHistoricalReward(ctx sdk.Context, operator, assetID, epochIdentifier string,
	period uint64,
) (feedistributiontypes.OperatorHistoricalRewards, error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), feedistributiontypes.KeyPrefixOperatorHistoricalRewards)
	// this encoding ensures the key is ordered by period.
	periodHexStr := hexutil.Encode(sdk.Uint64ToBigEndian(period))
	key := assetstype.GetJoinedStoreKey(operator, assetID, epochIdentifier, periodHexStr)
	b := store.Get(key)
	if b == nil {
		return feedistributiontypes.OperatorHistoricalRewards{}, feedistributiontypes.ErrNoKeyInTheStore.Wrapf("GetOperatorHistoricalReward, operator:%s,assetID:%s,epochIdentifier:%s,period:%d", operator, assetID, epochIdentifier, period)
	}
	historicalReward := feedistributiontypes.OperatorHistoricalRewards{}
	k.cdc.MustUnmarshal(b, &historicalReward)
	return historicalReward, nil
}

// OperatorRewardsForAllPeriods : get the operator historical rewards for all periods
func (k Keeper) OperatorRewardsForAllPeriods(ctx sdk.Context, operator, assetID, epochIdentifier string) ([]feedistributiontypes.OperatorHistoricalRewardsAndPeriod, error) {
	ret := make([]feedistributiontypes.OperatorHistoricalRewardsAndPeriod, 0)
	iterationPrefix := assetstype.GetJoinedStoreKeyForPrefix(operator, assetID, epochIdentifier)

	opFunc := func(_, _, _ string, period uint64, operatorHistoricalReward *feedistributiontypes.OperatorHistoricalRewards) (bool, error) {
		ret = append(ret, feedistributiontypes.OperatorHistoricalRewardsAndPeriod{
			Period:                    period,
			OperatorHistoricalRewards: *operatorHistoricalReward,
		})
		return false, nil
	}

	err := k.IterateOperatorHistoricalRewards(ctx, false, iterationPrefix, opFunc)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// HasOperatorHistoricalRewards : check whether the historical rewards for the specific operator, EpochIdentifier
// assetID and period exists.
func (k Keeper) HasOperatorHistoricalRewards(ctx sdk.Context, operator, assetID, epochIdentifier string, period uint64) bool {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), feedistributiontypes.KeyPrefixOperatorHistoricalRewards)
	// this encoding ensures the key is ordered by period.
	periodHexStr := hexutil.Encode(sdk.Uint64ToBigEndian(period))
	key := assetstype.GetJoinedStoreKey(operator, assetID, epochIdentifier, periodHexStr)
	return store.Has(key)
}

// IterateOperatorHistoricalRewards iterates over all operator historical rewards
func (k *Keeper) IterateOperatorHistoricalRewards(ctx sdk.Context, isUpdate bool, iteratePrefix []byte,
	opFunc func(operator, assetID, epochIdentifier string, period uint64, operatorHistoricalReward *feedistributiontypes.OperatorHistoricalRewards) (bool, error),
) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), feedistributiontypes.KeyPrefixOperatorHistoricalRewards)
	iterator := sdk.KVStorePrefixIterator(store, iteratePrefix)
	defer iterator.Close()

	updatedKeyValues := make([]utils.KeyValueT[*feedistributiontypes.OperatorHistoricalRewards], 0)
	for ; iterator.Valid(); iterator.Next() {
		var operatorHistoricalReward feedistributiontypes.OperatorHistoricalRewards
		keys, err := assetstype.ParseJoinedStoreKey(iterator.Key(), 4)
		if err != nil {
			return err
		}
		k.cdc.MustUnmarshal(iterator.Value(), &operatorHistoricalReward)
		periodBytes, err := hexutil.Decode(keys[3])
		if err != nil {
			return err
		}
		isBreak, err := opFunc(keys[0], keys[1], keys[2], sdk.BigEndianToUint64(periodBytes), &operatorHistoricalReward)
		if err != nil {
			return err
		}
		if isBreak {
			break
		}
		if isUpdate {
			updatedKeyValues = append(updatedKeyValues, utils.KeyValueT[*feedistributiontypes.OperatorHistoricalRewards]{
				Key:   append([]byte(nil), iterator.Key()...),
				Value: &operatorHistoricalReward,
			})
		}
	}
	for _, updateKeyValue := range updatedKeyValues {
		bz := k.cdc.MustMarshal(updateKeyValue.Value)
		store.Set(updateKeyValue.Key, bz)
	}
	return nil
}

// SetDelegationStartingInfo : set the starting information for the delegation
func (k Keeper) SetDelegationStartingInfo(ctx sdk.Context, delegationKey, epochIdentifier string, startingInfo feedistributiontypes.DelegationStartingInfo) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), feedistributiontypes.KeyPrefixDelegationStartingInfo)
	bz := k.cdc.MustMarshal(&startingInfo)
	key := assetstype.GetJoinedStoreKey(delegationKey, epochIdentifier)
	store.Set(key, bz)
	return nil
}

// GetDelegationStartingInfo : get the starting information for the delegation
func (k Keeper) GetDelegationStartingInfo(ctx sdk.Context, delegationKey, epochIdentifier string) (feedistributiontypes.DelegationStartingInfo, error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), feedistributiontypes.KeyPrefixDelegationStartingInfo)
	key := assetstype.GetJoinedStoreKey(delegationKey, epochIdentifier)
	b := store.Get(key)
	if b == nil {
		return feedistributiontypes.DelegationStartingInfo{}, feedistributiontypes.ErrNoKeyInTheStore.Wrapf("GetDelegationStartingInfo, delegationKey:%s,epochIdentifier:%s", delegationKey, epochIdentifier)
	}
	startingInfo := feedistributiontypes.DelegationStartingInfo{}
	k.cdc.MustUnmarshal(b, &startingInfo)
	return startingInfo, nil
}

// DeleteDelegationStartingInfo : delete the starting information for the delegation
func (k Keeper) DeleteDelegationStartingInfo(ctx sdk.Context, delegationKey, epochIdentifier string) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), feedistributiontypes.KeyPrefixDelegationStartingInfo)
	key := assetstype.GetJoinedStoreKey(delegationKey, epochIdentifier)
	store.Delete(key)
	return nil
}

// HasDelegationStartingInfo : check whether the starting information for the delegation exists.
func (k Keeper) HasDelegationStartingInfo(ctx sdk.Context, delegationKey, epochIdentifier string) bool {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), feedistributiontypes.KeyPrefixDelegationStartingInfo)
	key := assetstype.GetJoinedStoreKey(delegationKey, epochIdentifier)
	return store.Has(key)
}

// SetOperatorSlashEvent : set the operator slash event in distribution module
func (k Keeper) SetOperatorSlashEvent(ctx sdk.Context, operator, assetID, epochIdentifier string,
	epochNumber, blockHeight uint64, slashEvent feedistributiontypes.OperatorSlashEvent,
) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), feedistributiontypes.KeyPrefixOperatorSlashEvent)
	bz := k.cdc.MustMarshal(&slashEvent)
	// this encoding ensures the key is ordered by epoch number.
	epochNumberHexStr := hexutil.Encode(sdk.Uint64ToBigEndian(epochNumber))
	heightHexStr := hexutil.Encode(sdk.Uint64ToBigEndian(blockHeight))
	key := assetstype.GetJoinedStoreKey(operator, assetID, epochIdentifier, epochNumberHexStr, heightHexStr)
	store.Set(key, bz)
	return nil
}

// GetOperatorSlashEvent : get the operator slash event in distribution module
func (k Keeper) GetOperatorSlashEvent(ctx sdk.Context, operator, assetID, epochIdentifier string,
	epochNumber, blockHeight uint64,
) (feedistributiontypes.OperatorSlashEvent, error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), feedistributiontypes.KeyPrefixOperatorSlashEvent)
	// this encoding ensures the key is ordered by epoch number.
	epochNumberHexStr := hexutil.Encode(sdk.Uint64ToBigEndian(epochNumber))
	heightHexStr := hexutil.Encode(sdk.Uint64ToBigEndian(blockHeight))
	key := assetstype.GetJoinedStoreKey(operator, assetID, epochIdentifier, epochNumberHexStr, heightHexStr)
	b := store.Get(key)
	if b == nil {
		return feedistributiontypes.OperatorSlashEvent{}, feedistributiontypes.ErrNoKeyInTheStore.Wrapf(
			"GetOperatorSlashEvent, operator:%s,epochIdentifier:%s,epochNumber:%d,blockHeight:%d", operator,
			epochIdentifier, epochNumber, blockHeight)
	}
	slashEvent := feedistributiontypes.OperatorSlashEvent{}
	k.cdc.MustUnmarshal(b, &slashEvent)
	return slashEvent, nil
}

// GetOperatorSlashEvents : get the operator slash events in distribution module
func (k Keeper) GetOperatorSlashEvents(ctx sdk.Context, operator, assetID, epochIdentifier string,
) ([]feedistributiontypes.OperatorSlashEventAndHeight, error) {
	currentEpochInfo, exist := k.epochsKeeper.GetEpochInfo(ctx, epochIdentifier)
	if !exist {
		return nil, feedistributiontypes.ErrEpochNotFound.Wrapf("GetOperatorSlashEvents, EpochIdentifier:%s", epochIdentifier)
	}
	ret := make([]feedistributiontypes.OperatorSlashEventAndHeight, 0)
	err := k.IterateOperatorSlashEventsBetween(
		ctx, operator, assetID, epochIdentifier,
		uint64(types.InitialEpochNumber), uint64(currentEpochInfo.CurrentEpoch),
		func(epochNumber, blockHeight uint64, event feedistributiontypes.OperatorSlashEvent) (stop bool, err error) {
			ret = append(ret, feedistributiontypes.OperatorSlashEventAndHeight{
				EpochNumber:        epochNumber,
				BlockHeight:        blockHeight,
				OperatorSlashEvent: event,
			})
			return false, nil
		})
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// IterateOperatorSlashEventsBetween iterates over slash events between epoch numbers, inclusive
func (k Keeper) IterateOperatorSlashEventsBetween(ctx sdk.Context, operator, assetID, epochIdentifier string,
	startingEpochNumber uint64, endingEpochNumber uint64,
	handler func(epochNumber, blockHeight uint64, event feedistributiontypes.OperatorSlashEvent) (stop bool, err error),
) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), feedistributiontypes.KeyPrefixOperatorSlashEvent)
	epochNumberHexStr := hexutil.Encode(sdk.Uint64ToBigEndian(startingEpochNumber))
	startKey := assetstype.GetJoinedStoreKey(operator, assetID, epochIdentifier, epochNumberHexStr)
	// Add 1 to include all slash events in the ending epoch
	epochNumberHexStr = hexutil.Encode(sdk.Uint64ToBigEndian(endingEpochNumber + 1))
	endKey := assetstype.GetJoinedStoreKey(operator, assetID, epochIdentifier, epochNumberHexStr)

	iter := store.Iterator(
		startKey,
		endKey,
	)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		var event feedistributiontypes.OperatorSlashEvent
		k.cdc.MustUnmarshal(iter.Value(), &event)
		keys, err := assetstype.ParseJoinedStoreKey(iter.Key(), 5)
		if err != nil {
			return err
		}
		epochNumberBigEndian, err := hexutil.Decode(keys[3])
		if err != nil {
			return err
		}
		epochNumber := sdk.BigEndianToUint64(epochNumberBigEndian)

		blockHeightBigEndian, err := hexutil.Decode(keys[4])
		if err != nil {
			return err
		}
		blockHeight := sdk.BigEndianToUint64(blockHeightBigEndian)

		isStop, err := handler(epochNumber, blockHeight, event)
		if err != nil {
			return err
		}
		if isStop {
			break
		}
	}
	return nil
}

// SetStakerClaimedRewards : set the claimed avs rewards for the staker
func (k Keeper) SetStakerClaimedRewards(ctx sdk.Context, stakerID, avsAddr string,
	rewards feedistributiontypes.StakerClaimedRewards,
) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), feedistributiontypes.KeyPrefixStakerClaimedRewards)
	bz := k.cdc.MustMarshal(&rewards)
	key := assetstype.GetJoinedStoreKey(stakerID, avsAddr)
	store.Set(key, bz)
	return nil
}

// GetStakerClaimedRewards : get the claimed avs rewards for the staker
func (k Keeper) GetStakerClaimedRewards(ctx sdk.Context, stakerID,
	avsAddr string,
) (feedistributiontypes.StakerClaimedRewards, error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), feedistributiontypes.KeyPrefixStakerClaimedRewards)
	key := assetstype.GetJoinedStoreKey(stakerID, avsAddr)
	b := store.Get(key)
	if b == nil {
		return feedistributiontypes.StakerClaimedRewards{}, feedistributiontypes.ErrNoKeyInTheStore.Wrapf("GetStakerClaimedRewards, stakerID:%s,avsAddr:%s", stakerID, avsAddr)
	}
	rewards := feedistributiontypes.StakerClaimedRewards{}
	k.cdc.MustUnmarshal(b, &rewards)
	return rewards, nil
}

// HasStakerClaimedRewards : check whether the claimed avs rewards exists for the operator
func (k Keeper) HasStakerClaimedRewards(ctx sdk.Context, stakerID, avsAddr string) bool {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), feedistributiontypes.KeyPrefixStakerClaimedRewards)
	key := assetstype.GetJoinedStoreKey(stakerID, avsAddr)
	return store.Has(key)
}

// IncreaseStakerOutstandingRewards : increase the outstanding avs rewards for the staker.
func (k Keeper) IncreaseStakerOutstandingRewards(ctx sdk.Context, stakerID, avsAddr string, deltaRewards sdk.DecCoins) error {
	if len(deltaRewards) == 0 {
		return nil
	}
	// set the initialized value
	rewards := feedistributiontypes.StakerClaimedRewards{
		OutstandingRewards: make([]sdk.DecCoin, 0),
		WithdrawnRewards:   make([]sdk.DecCoin, 0),
	}
	var err error
	if k.HasStakerClaimedRewards(ctx, stakerID, avsAddr) {
		rewards, err = k.GetStakerClaimedRewards(ctx, stakerID, avsAddr)
		if err != nil {
			return err
		}
	}

	rewards.OutstandingRewards = rewards.OutstandingRewards.Add(deltaRewards...)
	err = k.SetStakerClaimedRewards(ctx, stakerID, avsAddr, rewards)
	if err != nil {
		return err
	}
	return nil
}

func GenericIterateStoreWithUpdate[T codec.ProtoMarshaler](
	ctx sdk.Context,
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	keyPrefix []byte,
	iteratePrefix []byte,
	isUpdate bool,
	keyNumber int,
	unmarshal func([]byte) (T, error),
	opFunc func(keys []string, value T) (bool, bool, error),
) error {
	store := prefix.NewStore(ctx.KVStore(storeKey), keyPrefix)
	iterator := sdk.KVStorePrefixIterator(store, iteratePrefix)
	defer iterator.Close()

	updatedKeyValues := make([]utils.KeyValueT[T], 0)
	for ; iterator.Valid(); iterator.Next() {
		keys, err := assetstype.ParseJoinedStoreKey(iterator.Key(), keyNumber)
		if err != nil {
			return err
		}

		value, err := unmarshal(iterator.Value())
		if err != nil {
			return err
		}

		isBreak, isChanged, err := opFunc(keys, value)
		if err != nil {
			return err
		}
		if isBreak {
			break
		}

		if isUpdate && isChanged {
			updatedKeyValues = append(updatedKeyValues, utils.KeyValueT[T]{
				Key:   append([]byte(nil), iterator.Key()...),
				Value: value,
			})
		}
	}
	for _, updateKeyValue := range updatedKeyValues {
		store.Set(updateKeyValue.Key, cdc.MustMarshal(updateKeyValue.Value))
	}
	return nil
}

// IterateStakerClaimedRewards : iterates the claimed rewards for a staker and does some external operations.
// `isUpdate` is a flag to indicate whether the change of the state should be set to the store.
func (k Keeper) IterateStakerClaimedRewards(
	ctx sdk.Context,
	stakerID string,
	isUpdate bool,
	opFunc func(avs string, rewards *feedistributiontypes.StakerClaimedRewards) (bool, bool, error),
) error {
	return GenericIterateStoreWithUpdate[*feedistributiontypes.StakerClaimedRewards](
		ctx,
		k.cdc,
		k.storeKey,
		feedistributiontypes.KeyPrefixStakerClaimedRewards,
		assetstype.GetJoinedStoreKeyForPrefix(stakerID),
		isUpdate,
		2,
		func(bz []byte) (*feedistributiontypes.StakerClaimedRewards, error) {
			var r feedistributiontypes.StakerClaimedRewards
			k.cdc.MustUnmarshal(bz, &r)
			return &r, nil
		},
		func(keys []string, value *feedistributiontypes.StakerClaimedRewards) (bool, bool, error) {
			return opFunc(keys[1], value)
		},
	)
}

// IterateOperatorCommissions : iterates the commissions for an operator
// and does some external operations.
// `isUpdate` is a flag to indicate whether the change of the state should be set to the store.
func (k Keeper) IterateOperatorCommissions(ctx sdk.Context, operator string, isUpdate bool,
	opFunc func(avs string, commissions *feedistributiontypes.OperatorCommission) (bool, bool, error),
) error {
	return GenericIterateStoreWithUpdate[*feedistributiontypes.OperatorCommission](
		ctx,
		k.cdc,
		k.storeKey,
		feedistributiontypes.KeyPrefixOperatorCommission,
		assetstype.GetJoinedStoreKeyForPrefix(operator),
		isUpdate,
		2,
		func(bz []byte) (*feedistributiontypes.OperatorCommission, error) {
			var c feedistributiontypes.OperatorCommission
			k.cdc.MustUnmarshal(bz, &c)
			return &c, nil
		},
		func(keys []string, value *feedistributiontypes.OperatorCommission) (bool, bool, error) {
			return opFunc(keys[1], value)
		},
	)
}

// GetStakerAllClaimedRewards : get the claimed rewards from all AVSs for the staker
func (k Keeper) GetStakerAllClaimedRewards(ctx sdk.Context, stakerID string,
) ([]feedistributiontypes.StakerClaimedRewardsPerAVS, error) {
	allAVSRewards := make([]feedistributiontypes.StakerClaimedRewardsPerAVS, 0)
	opFunc := func(avs string, rewards *feedistributiontypes.StakerClaimedRewards) (bool, bool, error) {
		allAVSRewards = append(allAVSRewards, feedistributiontypes.StakerClaimedRewardsPerAVS{
			AVSAddress:     avs,
			ClaimedRewards: *rewards,
		})
		return false, true, nil
	}
	// iterate to withdraw rewards from multiple AVSs, because different AVSs might
	// use the same asset as reward.
	err := k.IterateStakerClaimedRewards(ctx, stakerID, false, opFunc)
	if err != nil {
		return nil, err
	}
	return allAVSRewards, nil
}
