package keeper

import (
	"cosmossdk.io/math"
	assetstype "github.com/ExocoreNetwork/exocore/x/assets/types"
	"github.com/ExocoreNetwork/exocore/x/avs/types"
	epochsTypes "github.com/ExocoreNetwork/exocore/x/epochs/types"
	feedistributiontypes "github.com/ExocoreNetwork/exocore/x/feedistribution/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

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
	assetInfo, err := k.assetsKeeper.GetStakingAssetInfo(ctx, assetID)
	if err != nil {
		return err
	}
	divisor := math.NewIntWithDecimal(1, int(assetInfo.AssetBasicInfo.Decimals)) // #nosec G115
	stake := sdk.NewDecFromInt(delegatedAmount).QuoInt(divisor)
	startingInfo := feedistributiontypes.DelegationStartingInfo{
		PreviousPeriod:   previousPeriod,
		Stake:            stake,
		EpochNumber:      uint64(epochInfo.CurrentEpoch),
		EpochStartHeight: uint64(epochInfo.CurrentEpochStartHeight),
	}
	err = k.SetDelegationStartingInfo(ctx, delegationKey, epochInfo.Identifier, startingInfo)
	if err != nil {
		return err
	}
	return nil
}

// calculate the rewards accrued by a delegation between two periods
func (k Keeper) calculateDelegationRewardsBetween(ctx sdk.Context, endingPeriod uint64, operator, assetID,
	epochIdentifier string, startingInfo feedistributiontypes.DelegationStartingInfo) (feedistributiontypes.CommonAVSRewards, error) {
	// sanity check
	if startingInfo.PreviousPeriod > endingPeriod {
		return nil, feedistributiontypes.ErrInvalidInputParameter.Wrapf("startingPeriod cannot be greater than endingPeriod, start:%d,end:%d", startingInfo.PreviousPeriod, endingPeriod)
	}

	// sanity check
	if startingInfo.Stake.IsNegative() {
		return nil, feedistributiontypes.ErrInvalidInputParameter.Wrapf("stake should not be negative, stake:%s", startingInfo.Stake)
	}

	// return staking * (ending - starting)
	starting, err := k.GetOperatorHistoricalRewards(ctx, operator, assetID, epochIdentifier, startingInfo.PreviousPeriod)
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
			assetID, epochIdentifier, startingInfo.PreviousPeriod, endingPeriod)
	}
	// note: necessary to truncate so we don't allow withdrawing more rewards than owed
	rewards, err := difference.CalculateRewards(startingInfo.Stake)
	if err != nil {
		return nil, err
	}
	return rewards, nil
}

// calculateDelegationRewards calculates the rewards accrued by a delegation
func (k Keeper) calculateDelegationRewards(ctx sdk.Context, endingPeriod uint64, operator, assetID,
	epochIdentifier string, startingInfo feedistributiontypes.DelegationStartingInfo) (feedistributiontypes.CommonAVSRewards, error) {

	return nil, nil
}

// DistributeRewardsToDelegation distributes the rewards to a delegation with changed stake
func (k Keeper) distributeRewardsToDelegation(ctx sdk.Context, endingPeriod uint64, operator, assetID,
	epochIdentifier string, startingInfo feedistributiontypes.DelegationStartingInfo) error {

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
			err = k.distributeRewardsToDelegation(ctx, endingPeriod, operator, assetID, epochInfo.Identifier, startingInfo)
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
