package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/distribution/types"
)

// IncrementOperatorPeriod : increment operator period, returning the period just ended
// The operator’s period needs to be incremented whenever the delegated stake changes,
// regardless of whether the operator is serving any AVSs.
func (k Keeper) IncrementOperatorPeriod(ctx sdk.Context, operator, assetID, epochIdentifier string) (uint64, error) {
	// fetch current rewards
	rewards, err := k.GetOperatorCurrentRewards(ctx, operator, assetID, epochIdentifier)
	if err != nil {
		return rewards.Period, err
	}
	if !k.operatorKeeper.HasOperatorAssetUSDValue(ctx, epochIdentifier, operator, assetID) {
		ctx.Logger().Info("IncrementOperatorPeriod, operator asset USD value doesn't exist", "epochIdentifier", epochIdentifier, "operator", operator, "assetID", assetID)
		//
		return rewards.Period, nil
	}
	// get the USD value for asset
	assetUSDValue, err := k.operatorKeeper.GetOperatorAssetUSDValue(ctx, epochIdentifier, operator, assetID)
	if err != nil {
		return rewards.Period, err
	}
	// calculate current ratio
	var current sdk.DecCoins
	if assetUSDValue.IsZero() {
		return rewards.Period, nil
	}
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
