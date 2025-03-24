package keeper

import (
	"cosmossdk.io/math"
	assetstype "github.com/ExocoreNetwork/exocore/x/assets/types"
	"github.com/ExocoreNetwork/exocore/x/avs/types"
	epochsTypes "github.com/ExocoreNetwork/exocore/x/epochs/types"
	feedistributiontypes "github.com/ExocoreNetwork/exocore/x/feedistribution/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) MarkStakeChangedDelegations(ctx sdk.Context, stakerID, assetID string, operator sdk.AccAddress, prevAssetState assetstype.OperatorAssetInfo) error {
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
	var err error
	for _, epochInfo := range allEpochs {
		delegationChangeInfo := feedistributiontypes.DelegationChangeInfo{
			StakerIds: make([]string, 0),
		}
		if k.HasStakeChangedDelegations(ctx, epochInfo.Identifier, operator.String(), assetID) {
			delegationChangeInfo, err = k.GetStakeChangedDelegations(ctx, epochInfo.Identifier, operator.String(), assetID)
			if err != nil {
				return err
			}
		} else {
			// This is the first delegation/undelegation that changes the delegated amount.
			// The total delegation amount of the operator at the end of the previous epoch needs to be saved.
			// get the current total delegation amount from the operator assets information
			// store it as a decimal type.
			assetInfo, err := k.assetsKeeper.GetStakingAssetInfo(ctx, assetID)
			if err != nil {
				return err
			}
			divisor := math.NewIntWithDecimal(1, int(assetInfo.AssetBasicInfo.Decimals)) // #nosec G115
			delegationChangeInfo.TotalAmount = sdk.NewDecFromInt(prevAssetState.TotalAmount).QuoInt(divisor)
		}

		delegationChangeInfo.AppendUniqueStakerID(stakerID)
		err = k.SetStakeChangedDelegations(ctx, epochInfo.Identifier, operator.String(), assetID, delegationChangeInfo)
		if err != nil {
			return err
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
		// increase the period for the operator with changed delegations.
		endingPeriod, err := k.IncrementOperatorPeriod(ctx, operator, assetID, epochIdentifier, delegationChangeInfo.TotalAmount)
		if err != nil {
			// Just log the error as a reminder; do not return it to avoid interrupting the handling
			// of other operators.
			ctx.Logger().Error("HandleChangedDelegations, failed to increment the period", "operator",
				operator, "assetID", assetID, "epochIdentifier", epochIdentifier, "err", err)
			return false, nil
		}
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
		return false, nil
	}
	return k.IterateStakeChangedDelegations(ctx, false, []byte(epochIdentifier), opFunc)
}

func (k Keeper) initializeDelegationStartingInfo(ctx sdk.Context, delegationKey, operator, stakerID,
	assetID string, epochInfo *epochsTypes.EpochInfo, previousPeriod uint64) error {
	// increase the reference count
	err := k.incrementReferenceCount(ctx, operator, assetID, epochInfo.Identifier, previousPeriod)
	if err != nil {
		return err
	}
	// get the current stake of the delegation
	_, delegatedAmount, err := k.delegationKeeper.GetDelegationInfoWithAmount(ctx, stakerID, assetID, operator)
	if err != nil {
		return err
	}
	if !delegatedAmount.IsPositive() {
		// Delete the starting info when the delegated amount is zero or negative.
		// Since this delegation won't generate any rewards, we don't need to save
		// the starting info for it.
		return k.DeleteDelegationStartingInfo(ctx, delegationKey, epochInfo.Identifier)
	}
	assetInfo, err := k.assetsKeeper.GetStakingAssetInfo(ctx, assetID)
	if err != nil {
		return err
	}
	divisor := math.NewIntWithDecimal(1, int(assetInfo.AssetBasicInfo.Decimals)) // #nosec G115
	stake := sdk.NewDecFromInt(delegatedAmount).QuoInt(divisor)
	startingInfo := feedistributiontypes.DelegationStartingInfo{
		PreviousPeriod: previousPeriod,
		Stake:          stake,
		EpochNumber:    uint64(epochInfo.CurrentEpoch),
	}
	err = k.SetDelegationStartingInfo(ctx, delegationKey, epochInfo.Identifier, startingInfo)
	if err != nil {
		return err
	}
	return nil
}

// calculate the rewards accrued by a delegation between two periods
func (k Keeper) calculateDelegationRewardsBetween(ctx sdk.Context, startingPeriod, endingPeriod uint64, operator, assetID,
	epochIdentifier string, stake sdk.Dec) (feedistributiontypes.CommonAVSRewards, error) {
	// sanity check
	if startingPeriod > endingPeriod {
		return nil, feedistributiontypes.ErrInvalidInputParameter.Wrapf("startingPeriod cannot be greater than endingPeriod, start:%d,end:%d", startingPeriod, endingPeriod)
	}

	// sanity check
	if stake.IsNegative() {
		return nil, feedistributiontypes.ErrInvalidInputParameter.Wrapf("stake should not be negative, stake:%s", stake)
	}

	// return staking * (ending - starting)
	starting, err := k.GetOperatorHistoricalRewards(ctx, operator, assetID, epochIdentifier, startingPeriod)
	if err != nil {
		return nil, err
	}
	ending, err := k.GetOperatorHistoricalRewards(ctx, operator, assetID, epochIdentifier, endingPeriod)
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
func (k Keeper) calculateDelegationRewards(ctx sdk.Context, endingPeriod uint64, operator, assetID,
	epochIdentifier string, startingInfo feedistributiontypes.DelegationStartingInfo) (feedistributiontypes.CommonAVSRewards, error) {
	currentEpochInfo, isExist := k.epochsKeeper.GetEpochInfo(ctx, epochIdentifier)
	if !isExist {
		return nil, feedistributiontypes.ErrEpochNotFound
	}
	currentEpochNumber := uint64(currentEpochInfo.CurrentEpoch)
	startingEpochNumber := startingInfo.EpochNumber
	rewards := make([]feedistributiontypes.CommonAVSRewardData, 0)
	// check the epoch number
	if startingEpochNumber > currentEpochNumber {
		return nil, feedistributiontypes.ErrInvalidInputParameter.Wrapf("calculateDelegationRewards: the epoch number in starting Info is greater than the current epoch number, startEpochNumber:%d,currentEpochNumber:%d",
			startingEpochNumber, currentEpochNumber)
	} else if startingEpochNumber == currentEpochNumber {
		// no rewards yet if the delegation starts from current epoch
		return rewards, nil
	}
	// check the period
	startingPeriod := startingInfo.PreviousPeriod
	stake := startingInfo.Stake
	if startingPeriod > endingPeriod {
		return nil, feedistributiontypes.ErrInvalidInputParameter.Wrapf("calculateDelegationRewards: the period in starting Info is greater than the ending period, startPeriod:%d,endingPeriod:%d",
			startingPeriod, endingPeriod)
	} else if startingPeriod == endingPeriod {
		// no rewards yet if the delegation starts from current epoch
		return rewards, nil
	}
	opFunc := func(epochNumber uint64, event feedistributiontypes.OperatorSlashEvent) (stop bool, err error) {
		endingPeriod := event.OperatorPeriod
		if endingPeriod > startingPeriod {
			rewardsBetweenPeriod, err := k.calculateDelegationRewardsBetween(ctx, startingPeriod, endingPeriod, operator, assetID, epochIdentifier, stake)
			if err != nil {
				return false, err
			}
			rewards = feedistributiontypes.CommonAVSRewards(rewards).Add(rewardsBetweenPeriod...)
			// Note: It is necessary to truncate so we don't allow withdrawing
			// more rewards than owed.
			stake = stake.MulTruncate(math.LegacyOneDec().Sub(event.Fraction))
			startingPeriod = endingPeriod
		}
		return false, nil
	}
	err := k.IterateOperatorSlashEventsBetween(ctx, operator, assetID, epochIdentifier, startingInfo.EpochNumber,
		currentEpochNumber, opFunc)
	if err != nil {
		return rewards, err
	}

	// TODO: In the implementation of the Cosmos SDK, it checks the stake by comparing it with the current stake
	// to handle the precision loss caused by truncation.
	// We don't check it here because we handle reward distribution per epoch, so the compared value should
	// be the stake at the time of the last voting power update.
	// If we want to handle it, the compared stake needs to be saved in the delegation change information,
	// just like the total delegated amount.

	// calculate rewards for final period
	rewardsBetweenPeriod, err := k.calculateDelegationRewardsBetween(ctx, startingPeriod, endingPeriod, operator, assetID, epochIdentifier, stake)
	if err != nil {
		return rewards, err
	}
	rewards = feedistributiontypes.CommonAVSRewards(rewards).Add(rewardsBetweenPeriod...)
	return rewards, nil
}

// DistributeRewardsToDelegation distributes the rewards to a delegation with changed stake
func (k Keeper) distributeRewardsToDelegation(ctx sdk.Context, endingPeriod uint64, operator, stakerID, assetID string,
	epochInfo *epochsTypes.EpochInfo, startingInfo feedistributiontypes.DelegationStartingInfo) error {
	allAVSRewardsRaw, err := k.calculateDelegationRewards(ctx, endingPeriod, operator, assetID, epochInfo.Identifier, startingInfo)
	if err != nil {
		return err
	}
	for _, rewardsRawPerAVS := range allAVSRewardsRaw {
		outstanding, err := k.GetOperatorOutstandingRewards(ctx, operator, rewardsRawPerAVS.AVSAddress)
		if err != nil {
			ctx.Logger().Error("distributeRewardsToDelegation: failed to get the outstanding rewards",
				"operator", operator, "avs", rewardsRawPerAVS.AVSAddress, "err", err)
			return err
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
			return err
		}
		err = k.UpdateOperatorOutstandingRewards(ctx, operator, rewardsRawPerAVS.AVSAddress, false, rewards)
		if err != nil {
			return err
		}
	}
	// decrement reference count of starting period
	err = k.decrementReferenceCount(ctx, operator, assetID, epochInfo.Identifier, startingInfo.PreviousPeriod)
	if err != nil {
		return err
	}
	// reinitialize the starting info for the delegation.
	delegationKey := string(assetstype.GetJoinedStoreKey(stakerID, assetID, operator))
	err = k.initializeDelegationStartingInfo(ctx, delegationKey, operator, stakerID, assetID, epochInfo, endingPeriod)
	if err != nil {
		return err
	}
	return nil
}

func (k Keeper) DistributeRewardsToDelegations(ctx sdk.Context, endingPeriod uint64, epochInfo *epochsTypes.EpochInfo,
	operator, assetID string, delegationChangeInfo feedistributiontypes.DelegationChangeInfo) error {
	var err error
	for _, stakerID := range delegationChangeInfo.StakerIds {
		// initialize the delegation without the starting information.
		delegationKey := string(assetstype.GetJoinedStoreKey(stakerID, assetID, operator))
		if !k.HasDelegationStartingInfo(ctx, delegationKey, epochInfo.Identifier) {
			err = k.initializeDelegationStartingInfo(ctx, delegationKey, operator, stakerID, assetID, epochInfo, endingPeriod)
			if err != nil {
				// Just log the error as a reminder; do not return it to avoid interrupting the handling
				// of other stakers.
				ctx.Logger().Error("DistributeRewardsToDelegations, failed to initialize the starting info for the  delegation", "endingPeriod", endingPeriod, "delegationKey", delegationKey,
					"epochIdentifier", epochInfo.Identifier, "err", err)
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
			err = k.distributeRewardsToDelegation(ctx, endingPeriod, operator, stakerID, assetID, epochInfo, startingInfo)
			if err != nil {
				// Just log the error as a reminder; do not return it to avoid interrupting the handling
				// of other stakers.
				ctx.Logger().Error("DistributeRewardsToDelegations, failed to distribute rewards to the  delegation", "delegationKey", delegationKey,
					"epochIdentifier", epochInfo.Identifier, "err", err)
			}
		}
	}
	return nil
}
