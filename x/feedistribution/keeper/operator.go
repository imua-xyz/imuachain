package keeper

import (
	feedistributiontypes "github.com/ExocoreNetwork/exocore/x/feedistribution/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/distribution/types"
)

func (k Keeper) initializeOperatorPeriod(ctx sdk.Context, operator, assetID, epochIdentifier string) error {
	// initialize the historical rewards
	// the period in the historical rewards starts from 0
	err := k.SetOperatorHistoricalRewards(ctx, operator, assetID, epochIdentifier, 0,
		feedistributiontypes.OperatorHistoricalRewards{
			CumulativeRewardRatios: make([]*feedistributiontypes.CommonAVSRewardData, 0),
			// set the reference count to 1 because it will be referenced by the current reward.
			ReferenceCount: 1,
		})
	if err != nil {
		return err
	}
	// initialize the current rewards
	err = k.SetOperatorCurrentRewards(ctx, operator, assetID, epochIdentifier,
		feedistributiontypes.OperatorCurrentRewards{
			Rewards: make([]*feedistributiontypes.CommonAVSRewardData, 0),
			// the period in current rewards starts from 1.
			Period: 1,
		})
	if err != nil {
		return err
	}
	return nil
}

// IncrementOperatorPeriod : increment operator period, returning the period just ended
// The operator’s period needs to be incremented whenever the delegated stake changes,
// regardless of whether the operator is serving any AVSs.
func (k Keeper) IncrementOperatorPeriod(ctx sdk.Context, operator, assetID, epochIdentifier string,
	totalDelegationAmount sdk.Dec) (uint64, error) {
	if !k.HasOperatorCurrentRewards(ctx, operator, assetID, epochIdentifier) {
		// Initialize the current rewards and period of the operator.
		// This case occurs when the operator did not have any delegations of this asset at the end
		// of the previous epoch, which means all delegations happened in the current epoch.
		return 0, k.initializeOperatorPeriod(ctx, operator, assetID, epochIdentifier)
	}
	// fetch current rewards
	rewards, err := k.GetOperatorCurrentRewards(ctx, operator, assetID, epochIdentifier)
	if err != nil {
		return rewards.Period, err
	}

	if !totalDelegationAmount.IsPositive() {
		return 0, feedistributiontypes.ErrInvalidInputParameter.Wrapf(
			"IncrementOperatorPeriod, the total delegation amount isn't positive, amount:%s", totalDelegationAmount)
	}

	// calculate current reward ratio
	var current sdk.DecCoins
	// get the
	// note: necessary to truncate so we don't allow withdrawing more rewards than owed
	current = rewards.Rewards.QuoDecTruncate(sdk.NewDecFromInt(val.GetTokens()))

	// fetch historical rewards for last period
	historical := k.GetValidatorHistoricalRewards(ctx, val.GetOperator(), rewards.Period-1).CumulativeRewardRatio

	// decrement reference count
	k.decrementReferenceCount(ctx, val.GetOperator(), rewards.Period-1)

	// set new historical rewards with reference count of 1
	k.SetValidatorHistoricalRewards(ctx, val.GetOperator(), rewards.Period, types.NewValidatorHistoricalRewards(historical.Add(current...), 1))

	// set current rewards, incrementing period by 1
	k.SetValidatorCurrentRewards(ctx, val.GetOperator(), types.NewValidatorCurrentRewards(sdk.DecCoins{}, rewards.Period+1))

	return rewards.Period
}
