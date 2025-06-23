package keeper

import (
	"strconv"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	assetstype "github.com/imua-xyz/imuachain/x/assets/types"
	"github.com/imua-xyz/imuachain/x/avs/types"
	delegationtype "github.com/imua-xyz/imuachain/x/delegation/types"
	epochsTypes "github.com/imua-xyz/imuachain/x/epochs/types"
	feedistributiontypes "github.com/imua-xyz/imuachain/x/feedistribution/types"
)

func (k Keeper) MarkChangedDelegations(ctx sdk.Context, stakerID, assetID string, operator sdk.AccAddress,
	preDelegatedAmount math.Int, prevAssetState assetstype.OperatorAssetInfo,
) error {
	// The reason for marking delegations with stake changes for all epochs instead of only the impactful
	// epochs is that we need to update the operator’s period whenever the delegated stake changes,
	// regardless of whether the operator is serving any AVSs.
	// This is because the reward distribution for a restaker might not occur during the opting-in period.
	// For example, the staker might delegate additional stake, triggering the reward distribution lazily
	// after the operator has opted out.
	// If we don’t update the period for operators who have opted out of an AVS, the reward calculation
	// cannot correctly determine the stake and reward ratio for a staker. This is because the staker might
	// have delegated or undelegated tokens, altering the delegated stake during the opting-out period.
	allEpochs := k.avsKeeper.GetEpochsUsedByAllAVSs(ctx)
	assetInfo, err := k.assetsKeeper.GetStakingAssetInfo(ctx, assetID)
	if err != nil {
		return err
	}
	for _, epochIdentifier := range allEpochs {
		delegationChangeInfo := feedistributiontypes.DelegationChangeInfo{
			StakerDelegationChanges: make([]feedistributiontypes.StakerDelegationChange, 0),
		}
		if k.HasStakeChangedDelegations(ctx, epochIdentifier, operator.String(), assetID) {
			delegationChangeInfo, err = k.GetStakeChangedDelegations(ctx, epochIdentifier, operator.String(), assetID)
			if err != nil {
				return err
			}
		} else {
			// This is the first delegation/undelegation that changes the total delegated amount.
			// The total delegation amount of the operator at the end of the previous epoch needs to be saved.
			// get the current total delegation amount from the operator assets information
			// store it as a decimal type.
			delegationChangeInfo.TotalAmount = feedistributiontypes.ScaleIntByDecimals(
				prevAssetState.TotalAmount, assetInfo.AssetBasicInfo.Decimals)
		}

		isAppend := delegationChangeInfo.AppendUniqueStakerID(stakerID, preDelegatedAmount, assetInfo.AssetBasicInfo.Decimals)
		if isAppend {
			err = k.SetStakeChangedDelegations(ctx, epochIdentifier, operator.String(), assetID, delegationChangeInfo)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (k Keeper) HandleChangedDelegations(ctx sdk.Context, epochIdentifier string) error {
	epochInfo, exist := k.epochsKeeper.GetEpochInfo(ctx, epochIdentifier)
	if !exist {
		return types.ErrEpochNotFound
	}
	opFunc := func(epochIdentifier, operator, assetID string, delegationChangeInfo *feedistributiontypes.DelegationChangeInfo) (bool, error) {
		// Check if the operator asset period have been initialized. Return directly if not,
		// since no rewards have been accumulated for the delegations.
		if !k.isOperatorPeriodInitialized(ctx, operator, assetID, epochIdentifier) {
			return false, nil
		}
		// this function will be called by the epoch hook, so using cache context
		// to ensure the state atomicity.
		// increase the period for the operator with changed delegations.
		cc, writeFunc := ctx.CacheContext()
		endingPeriod, err := k.IncrementOperatorPeriod(cc, operator, assetID, epochIdentifier, delegationChangeInfo.TotalAmount)
		if err != nil {
			// Just log the error as a reminder; do not return it to avoid interrupting the handling
			// of other operators.
			ctx.Logger().Error("HandleChangedDelegations, failed to increment the period", "operator",
				operator, "assetID", assetID, "epochIdentifier", epochIdentifier, "err", err)
			return false, nil
		}
		writeFunc()
		// distribute the reward to the delegation with changed stakes.
		err = k.DistributeRewardsToDelegations(ctx, endingPeriod, &epochInfo, operator, assetID, *delegationChangeInfo)
		if err != nil {
			// Just log the error as a reminder; do not return it to avoid interrupting the handling
			// of other operators.
			ctx.Logger().Error("HandleChangedDelegations, failed to distribute rewards to delegations",
				"endingPeriod", endingPeriod, "operator", operator, "assetID", assetID,
				"epochIdentifier", epochIdentifier, "err", err)
			return false, nil
		}
		// emit the events for delegation distribution
		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				feedistributiontypes.EventTypeDistributeRewardToDelegations,
				sdk.NewAttribute(feedistributiontypes.AttributeKeyEpochIdentifier, epochIdentifier),
				sdk.NewAttribute(feedistributiontypes.AttributeKeyOperator, operator),
				sdk.NewAttribute(feedistributiontypes.AttributeKeyAssetID, assetID),
				sdk.NewAttribute(feedistributiontypes.AttributeKeyEndingPeriod, strconv.FormatUint(endingPeriod, 10)),
				sdk.NewAttribute(feedistributiontypes.AttributeKeyStakers, delegationChangeInfo.StakersAsString()),
				sdk.NewAttribute(feedistributiontypes.AttributeKeyPreDelegatedTotalAmount, delegationChangeInfo.TotalAmount.String()),
			),
		)
		return false, nil
	}
	return k.IterateStakeChangedDelegations(ctx, false, assetstype.GetJoinedStoreKeyForPrefix(epochIdentifier), opFunc)
}

func (k Keeper) initializeDelegationStartingInfo(
	ctx sdk.Context, operator, stakerID, assetID, epochIdentifier string,
	usePreStake bool, preStake sdk.Dec, startingEpochNumber uint64, previousPeriod uint64,
) error {
	stake := preStake
	if !usePreStake {
		// get the current stake of the delegation
		_, delegatedAmount, err := k.delegationKeeper.GetDelegationInfoWithAmount(ctx, stakerID, assetID, operator)
		if err != nil {
			return err
		}
		assetInfo, err := k.assetsKeeper.GetStakingAssetInfo(ctx, assetID)
		if err != nil {
			return err
		}
		stake = feedistributiontypes.ScaleIntByDecimals(delegatedAmount, assetInfo.AssetBasicInfo.Decimals)
	}

	delegationKey := string(assetstype.GetJoinedStoreKey(stakerID, assetID, operator))
	if !stake.IsPositive() {
		// Delete the starting info when the delegated amount is zero or negative.
		// Since this delegation won't generate any rewards, we don't need to save
		// the starting info for it.
		return k.DeleteDelegationStartingInfo(ctx, delegationKey, epochIdentifier)
	}

	startingInfo := feedistributiontypes.DelegationStartingInfo{
		PreviousPeriod: previousPeriod,
		Stake:          stake,
		EpochNumber:    startingEpochNumber,
	}
	err := k.SetDelegationStartingInfo(ctx, delegationKey, epochIdentifier, startingInfo)
	if err != nil {
		return err
	}

	// increase the reference count
	err = k.incrementReferenceCount(ctx, operator, assetID, epochIdentifier, previousPeriod)
	if err != nil {
		return err
	}
	return nil
}

func (k Keeper) reinitializeDelegationStartingInfo(
	ctx sdk.Context, isEndEpoch bool,
	preStake sdk.Dec, operator, stakerID,
	assetID string, epochInfo *epochsTypes.EpochInfo, previousPeriod uint64,
) error {
	usePreStake := false
	startingEpochNumber := uint64(epochInfo.CurrentEpoch)
	if !isEndEpoch {
		// The stake should remain unchanged when the isEndEpoch flag is false.
		// This function is used for reward distribution triggered by a staker-initiated claim
		// transaction. In this case, only the rewards before the current epochs will be distributed,
		// while the rewards from the current epoch will be distributed when the delegation stake
		// changes in the future. Therefore, the stake should be the last recorded stake after the
		// reward distribution. The current real-time stake cannot be used here, as there may be
		// delegations or undelegations affecting the stake after the reward distribution.
		usePreStake = true
		// If isEndEpoch is false, the delegation must should start from the previous epoch,
		// since the current epoch's rewards have not been distributed yet.
		startingEpochNumber = uint64(epochInfo.CurrentEpoch - 1)
	}
	return k.initializeDelegationStartingInfo(
		ctx, operator, stakerID, assetID, epochInfo.Identifier,
		usePreStake, preStake, startingEpochNumber, previousPeriod)
}

// calculate the rewards accrued by a delegation between two periods
func (k Keeper) calculateDelegationRewardsBetween(ctx sdk.Context, startingPeriod, endingPeriod uint64, operator, assetID,
	epochIdentifier string, stake sdk.Dec,
) (feedistributiontypes.CommonAVSRewards, error) {
	// sanity check
	if startingPeriod > endingPeriod {
		return nil, feedistributiontypes.ErrInvalidInputParameter.Wrapf("startingPeriod cannot be greater than endingPeriod, start:%d,end:%d", startingPeriod, endingPeriod)
	}

	// sanity check
	if stake.IsNegative() {
		return nil, feedistributiontypes.ErrInvalidInputParameter.Wrapf("stake should not be negative, stake:%s", stake)
	}

	// return staking * (ending - starting)
	starting, err := k.GetOperatorHistoricalReward(ctx, operator, assetID, epochIdentifier, startingPeriod)
	if err != nil {
		return nil, err
	}
	ending, err := k.GetOperatorHistoricalReward(ctx, operator, assetID, epochIdentifier, endingPeriod)
	if err != nil {
		return nil, err
	}
	difference, hasNeg := feedistributiontypes.CommonAVSRewards(ending.CumulativeRewardRatios).SafeSub(starting.CumulativeRewardRatios)
	if hasNeg {
		return nil, feedistributiontypes.ErrNegativeAVSRewards.Wrapf("calculateDelegationRewardsBetween returns negative avs rewards, operator:%s, assetID:%s, epochIdentifier:%s, startPeriod：%d,endPeriod:%d", operator,
			assetID, epochIdentifier, startingPeriod, endingPeriod)
	}
	// note: necessary to truncate so we don't allow withdrawing more rewards than owed
	rewards, err := difference.CalculateRewards(stake)
	if err != nil {
		return nil, err
	}
	return rewards, nil
}

// calculateDelegationRewards calculates the rewards accrued by a delegation
func (k Keeper) calculateDelegationRewards(ctx sdk.Context, endingPeriod uint64, operator, assetID string,
	currentEpochInfo *epochsTypes.EpochInfo, startingInfo feedistributiontypes.DelegationStartingInfo,
) (feedistributiontypes.CommonAVSRewards, sdk.Dec, error) {
	epochIdentifier := currentEpochInfo.Identifier
	currentEpochNumber := uint64(currentEpochInfo.CurrentEpoch)
	startingEpochNumber := startingInfo.EpochNumber
	rewards := make([]feedistributiontypes.CommonAVSRewardData, 0)

	// check the epoch number
	if startingEpochNumber >= currentEpochNumber {
		return nil, sdk.Dec{}, feedistributiontypes.ErrInvalidInputParameter.Wrapf("calculateDelegationRewards: the epoch number in starting Info is greater than or equal to the current epoch number, startEpochNumber:%d,currentEpochNumber:%d", startingEpochNumber, currentEpochNumber)
	}
	// check the period
	startingPeriod := startingInfo.PreviousPeriod
	stake := startingInfo.Stake
	if startingPeriod >= endingPeriod {
		return nil, sdk.Dec{}, feedistributiontypes.ErrInvalidInputParameter.Wrapf("calculateDelegationRewards: the period in starting Info is greater than or equal to the ending period, startPeriod:%d,endingPeriod:%d",
			startingPeriod, endingPeriod)
	}

	opFunc := func(_, _ uint64, event feedistributiontypes.OperatorSlashEvent) (stop bool, err error) {
		slashEndingPeriod := event.OperatorPeriod
		if slashEndingPeriod > startingPeriod {
			rewardsBetweenPeriod, err := k.calculateDelegationRewardsBetween(ctx, startingPeriod, slashEndingPeriod, operator, assetID, epochIdentifier, stake)
			if err != nil {
				return false, err
			}
			rewards = feedistributiontypes.CommonAVSRewards(rewards).Add(rewardsBetweenPeriod...)
			// Note: It is necessary to truncate so we don't allow withdrawing
			// more rewards than owed.
			stake = stake.MulTruncate(math.LegacyOneDec().Sub(event.Fraction))
			startingPeriod = slashEndingPeriod
		}
		return false, nil
	}
	err := k.IterateOperatorSlashEventsBetween(ctx, operator, assetID, epochIdentifier, startingInfo.EpochNumber,
		currentEpochNumber, opFunc)
	if err != nil {
		return rewards, sdk.Dec{}, err
	}

	// TODO: In the implementation of the Cosmos SDK, it checks the stake by comparing it with the current stake
	// to handle the precision loss caused by truncation.
	// We don't check it here because we handle reward distribution per epoch, so the compared value should
	// be the stake at the time of the last voting power update.
	// If we want to handle it, the compared stake needs to be saved in the delegation change information,
	// just like the total delegated amount.
	// It might be unnecessary to address it now, because the precision loss in the reward is likely to be negligible
	// or acceptable.

	// calculate rewards for final period
	rewardsBetweenPeriod, err := k.calculateDelegationRewardsBetween(ctx, startingPeriod, endingPeriod, operator, assetID, epochIdentifier, stake)
	if err != nil {
		return rewards, sdk.Dec{}, err
	}
	rewards = feedistributiontypes.CommonAVSRewards(rewards).Add(rewardsBetweenPeriod...)
	return rewards, stake, nil
}

// DistributeRewardsToDelegation distributes rewards to a delegation.
// It is used in two cases:
//  1. Distributing rewards to delegations with changed stakes, which occurs at the end of an epoch.
//  2. Distributing rewards to delegations triggered by a staker's active claim transaction,
//     which occurs during the epoch.
//
// The `isEndEpoch` flag is used to distinguish these two cases.
func (k Keeper) distributeRewardsToDelegation(
	ctx sdk.Context,
	isEndEpoch bool, endingPeriod uint64,
	operator, stakerID, assetID string,
	epochInfo *epochsTypes.EpochInfo, startingInfo feedistributiontypes.DelegationStartingInfo,
) (feedistributiontypes.CommonAVSRewards, error) {
	allAVSRewardsRaw, lastStake, err := k.calculateDelegationRewards(ctx, endingPeriod, operator, assetID, epochInfo, startingInfo)
	if err != nil {
		return nil, err
	}
	totalReward := make(feedistributiontypes.CommonAVSRewards, 0)
	for _, rewardsRawPerAVS := range allAVSRewardsRaw {
		outstanding, err := k.GetOperatorOutstandingRewards(ctx, operator, rewardsRawPerAVS.AVSAddress)
		if err != nil {
			ctx.Logger().Error("distributeRewardsToDelegation: failed to get the outstanding rewards",
				"operator", operator, "avs", rewardsRawPerAVS.AVSAddress, "err", err)
			return nil, err
		}
		// This check is from the implementation of the Cosmos SDK.
		// Not sure if this edge case also exists in the Imua protocol,
		// but adding it here to avoid exceptions.
		// defensive edge case may happen on the very final digits
		// of the decCoins due to operation order of the distribution mechanism.
		rewards := rewardsRawPerAVS.Rewards.Intersect(outstanding.Rewards)
		if !rewards.IsEqual(rewardsRawPerAVS.Rewards) {
			ctx.Logger().Error(
				"rounding error distributing rewards to delegation",
				"operator", operator,
				"avs", rewardsRawPerAVS.AVSAddress,
				"got", rewards.String(),
				"expected", rewardsRawPerAVS.Rewards.String(),
			)
		}
		// move the rewards to staker from the operator outstanding rewards.
		err = k.UpdateStakerOutstandingRewards(ctx, stakerID, rewardsRawPerAVS.AVSAddress, true, rewards)
		if err != nil {
			return nil, err
		}
		err = k.UpdateOperatorOutstandingRewards(ctx, operator, rewardsRawPerAVS.AVSAddress, false, rewards)
		if err != nil {
			return nil, err
		}
		totalReward = append(totalReward, feedistributiontypes.CommonAVSRewardData{
			AVSAddress: rewardsRawPerAVS.AVSAddress,
			Rewards:    rewards,
		})
	}
	// decrement reference count of starting period
	err = k.decrementReferenceCount(ctx, operator, assetID, epochInfo.Identifier, startingInfo.PreviousPeriod)
	if err != nil {
		return nil, err
	}
	// reinitialize the starting info for the delegation.
	err = k.reinitializeDelegationStartingInfo(ctx, isEndEpoch, lastStake, operator, stakerID, assetID,
		epochInfo, endingPeriod)
	if err != nil {
		return nil, err
	}
	return totalReward, nil
}

// DistributeRewardsToDelegations is used to distribute the rewards to the delegations with changed stake lazily.
// It will be used at the end of epoch.
func (k Keeper) DistributeRewardsToDelegations(ctx sdk.Context, endingPeriod uint64, epochInfo *epochsTypes.EpochInfo,
	operator, assetID string, delegationChangeInfo feedistributiontypes.DelegationChangeInfo,
) error {
	var err error
	for _, stakerDelegationChange := range delegationChangeInfo.StakerDelegationChanges {
		// This function is called by the epoch hook. It uses a cache context
		// to ensure atomicity of state updates. A separate cache context is
		// created for each staker, so that if the reward distribution for a
		// single delegation fails, only the state changes for that delegation
		// are reverted. This prevents failures from affecting other delegations.
		cc, writeFunc := ctx.CacheContext()
		// initialize the delegation without the starting information.
		delegationKey := string(assetstype.GetJoinedStoreKey(stakerDelegationChange.StakerId, assetID, operator))
		if !k.HasDelegationStartingInfo(ctx, delegationKey, epochInfo.Identifier) {
			// Delegations that are either created after the operator is initialized,
			// or created before but within the same epoch as the operator initialization,
			// will be initialized here, since they won't accumulate rewards for the current epoch.
			// The rewards for these delegation will be accumulated starting from the next epoch,
			// so the current epoch number is used as the starting point.
			err = k.initializeDelegationStartingInfo(
				cc, operator, stakerDelegationChange.StakerId, assetID, epochInfo.Identifier,
				false, math.LegacyZeroDec(), uint64(epochInfo.CurrentEpoch), endingPeriod)
			if err != nil {
				// Just log the error as a reminder; do not return it to avoid interrupting the handling
				// of other stakers.
				ctx.Logger().Error("DistributeRewardsToDelegations, failed to initialize the starting info for the  delegation", "endingPeriod", endingPeriod, "delegationKey", delegationKey,
					"epochIdentifier", epochInfo.Identifier, "err", err)
				continue
			}
		} else {
			// get the starting information for the specific delegation
			startingInfo, err := k.GetDelegationStartingInfo(ctx, delegationKey, epochInfo.Identifier)
			if err != nil {
				// Just log the error as a reminder; do not return it to avoid interrupting the handling
				// of other stakers.
				ctx.Logger().Error("DistributeRewardsToDelegations, failed to get the starting info for the  delegation", "delegationKey", delegationKey,
					"epochIdentifier", epochInfo.Identifier, "err", err)
				continue
			}
			// distribute the rewards for a delegation.
			_, err = k.distributeRewardsToDelegation(cc, true, endingPeriod, operator, stakerDelegationChange.StakerId, assetID, epochInfo, startingInfo)
			if err != nil {
				// Just log the error as a reminder; do not return it to avoid interrupting the handling
				// of other stakers.
				ctx.Logger().Error("DistributeRewardsToDelegations, failed to distribute rewards to the  delegation", "delegationKey", delegationKey,
					"epochIdentifier", epochInfo.Identifier, "err", err)
				continue
			}
		}
		writeFunc()
	}
	return nil
}

// ClaimDelegationRewards allows the staker to actively claim their rewards.
// It is triggered when the staker submits a transaction. So the transaction will be
// handled during the epoch.
func (k Keeper) ClaimDelegationRewards(ctx sdk.Context, stakerID string) (feedistributiontypes.CommonAVSRewards, error) {
	totalClaimedRewards := make(feedistributiontypes.CommonAVSRewards, 0)
	allEpochIdentifiers := k.avsKeeper.GetEpochsUsedByAllAVSs(ctx)
	opFunc := func(keys *delegationtype.SingleDelegationInfoReq, _ *delegationtype.DelegationAmounts) (bool, error) {
		for _, epochIdentifier := range allEpochIdentifiers {
			epochInfo, exist := k.epochsKeeper.GetEpochInfo(ctx, epochIdentifier)
			if !exist {
				return false, feedistributiontypes.ErrEpochNotFound.Wrapf("StakerClaimDelegationReward, EpochIdentifier:%s", epochIdentifier)
			}
			// get the starting info
			delegationKey := string(assetstype.GetJoinedStoreKey(stakerID, keys.AssetId, keys.OperatorAddr))
			if !k.HasDelegationStartingInfo(ctx, delegationKey, epochInfo.Identifier) {
				// no rewards for the delegation without starting information.
				continue
			}
			startingInfo, err := k.GetDelegationStartingInfo(ctx, delegationKey, epochInfo.Identifier)
			if err != nil {
				return false, err
			}

			if startingInfo.EpochNumber >= uint64(epochInfo.CurrentEpoch) {
				// this case shouldn't exist, so return an error
				return false, feedistributiontypes.ErrInvalidStartingInfo.Wrapf("StakerClaimDelegationReward, epoch number in starting info should be less than the current epoch number, startingEpochNumber:%d, current:%d", startingInfo.EpochNumber, epochInfo.CurrentEpoch)
			} else if startingInfo.EpochNumber == uint64(epochInfo.CurrentEpoch-1) {
				// No rewards if the delegation started in the previous epoch,
				// because the current epoch's reward has not been distributed yet.
				// It will be distributed at the end of the current epoch.
				continue
			}
			if !startingInfo.Stake.IsPositive() {
				return false, feedistributiontypes.ErrInvalidStartingInfo.Wrapf("StakerClaimDelegationReward, stake in starting info should be positive, stake:%s", startingInfo.Stake)
			}
			// increase the period for the operator before distributing the rewards
			delegatedAmountAtPreEpoch, err := k.getDelegatedAmountAtPreEpochEnd(ctx, keys.OperatorAddr, keys.AssetId, epochInfo.Identifier)
			if err != nil {
				return false, err
			}
			endingPeriod, err := k.IncrementOperatorPeriod(ctx, keys.OperatorAddr, keys.AssetId, epochInfo.Identifier, delegatedAmountAtPreEpoch)
			if err != nil {
				return false, err
			}
			// distribute the rewards for a delegation.
			delegationRewards, err := k.distributeRewardsToDelegation(ctx, false, endingPeriod, keys.OperatorAddr, stakerID,
				keys.AssetId, &epochInfo, startingInfo)
			if err != nil {
				return false, err
			}
			totalClaimedRewards = totalClaimedRewards.Add(delegationRewards...)
		}

		return false, nil
	}
	err := k.delegationKeeper.IterateDelegationsForStaker(ctx, stakerID, opFunc)
	if err != nil {
		return nil, err
	}
	return totalClaimedRewards, nil
}

// GetStakerUnclaimedRewards queries the unclaimed rewards for a staker.
// Unlike ClaimDelegationRewards, it does not trigger a claim operation.
func (k Keeper) GetStakerUnclaimedRewards(ctx sdk.Context, stakerID string) ([]feedistributiontypes.CommonAVSRewardData, error) {
	allEpochIdentifiers := k.avsKeeper.GetEpochsUsedByAllAVSs(ctx)
	ret := make([]feedistributiontypes.CommonAVSRewardData, 0)
	opFunc := func(keys *delegationtype.SingleDelegationInfoReq, _ *delegationtype.DelegationAmounts) (bool, error) {
		for i := range allEpochIdentifiers {
			epochIdentifier := allEpochIdentifiers[i]
			epochInfo, exist := k.epochsKeeper.GetEpochInfo(ctx, epochIdentifier)
			if !exist {
				return false, feedistributiontypes.ErrEpochNotFound.Wrapf("GetStakerUnclaimedRewards,epochIdentifier:%s", epochIdentifier)
			}
			// get the starting info
			delegationKey := string(assetstype.GetJoinedStoreKey(stakerID, keys.AssetId, keys.OperatorAddr))
			if !k.HasDelegationStartingInfo(ctx, delegationKey, epochInfo.Identifier) {
				// no rewards for the delegation without starting information.
				continue
			}
			startingInfo, err := k.GetDelegationStartingInfo(ctx, delegationKey, epochInfo.Identifier)
			if err != nil {
				return false, err
			}
			if startingInfo.EpochNumber >= uint64(epochInfo.CurrentEpoch) {
				// this case shouldn't exist, so return an error
				return false, feedistributiontypes.ErrInvalidStartingInfo.Wrapf("StakerClaimDelegationReward, epoch number in starting info should be less than the current epoch number, startingEpochNumber:%d, current:%d", startingInfo.EpochNumber, epochInfo.CurrentEpoch)
			} else if startingInfo.EpochNumber == uint64(epochInfo.CurrentEpoch-1) {
				// No rewards if the delegation started in the previous epoch,
				// because the current epoch's reward has not been distributed yet.
				// It will be distributed at the end of the current epoch.
				continue
			}
			if !startingInfo.Stake.IsPositive() {
				return false, feedistributiontypes.ErrInvalidStartingInfo.Wrapf("StakerClaimDelegationReward, stake in starting info should be positive, stake:%s", startingInfo.Stake)
			}
			// increase the period for the operator before distributing the rewards
			delegatedAmountAtPreEpoch, err := k.getDelegatedAmountAtPreEpochEnd(ctx, keys.OperatorAddr, keys.AssetId, epochInfo.Identifier)
			if err != nil {
				return false, err
			}
			endingPeriod, err := k.IncrementOperatorPeriod(ctx, keys.OperatorAddr, keys.AssetId, epochInfo.Identifier, delegatedAmountAtPreEpoch)
			if err != nil {
				return false, err
			}
			// calculate the rewards
			allAVSRewardsRaw, _, err := k.calculateDelegationRewards(ctx, endingPeriod, keys.OperatorAddr, keys.AssetId, &epochInfo, startingInfo)
			if err != nil {
				return false, err
			}
			ret = append(ret, allAVSRewardsRaw...)
		}

		return false, nil
	}
	err := k.delegationKeeper.IterateDelegationsForStaker(ctx, stakerID, opFunc)
	if err != nil {
		return nil, err
	}
	return ret, nil
}
