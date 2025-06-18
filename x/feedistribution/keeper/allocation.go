package keeper

import (
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/imua-xyz/imuachain/x/feedistribution/types"
)

// AllocateRewardsByEpoch performs reward and fee distribution to all operators for the AVS with same epoch
// configuration based on the F1 fee distribution specification.
func (k Keeper) AllocateRewardsByEpoch(ctx sdk.Context, epochIdentifier string, endingEpochNumber int64) error {
	avsList := k.avsKeeper.GetEpochEndAVSs(ctx, epochIdentifier, endingEpochNumber)
	for _, avs := range avsList {
		err := k.AllocateRewardsByAVS(ctx, avs, epochIdentifier)
		if err != nil {
			ctx.Logger().Info("AllocateTokensByEpoch: failed to allocate rewards by avs, skipping the avs",
				"err", err, "avs", avs)
		}
		// continue handling the other AVSs
	}
	return nil
}

func (k Keeper) AllocateRewardsByAVS(ctx sdk.Context, avs, epochIdentifier string) error {
	isDogfood, rewardAndProportions, err := k.AVSRewardAndProportionsByParam(ctx, avs)
	if err != nil {
		return err
	}
	if len(rewardAndProportions.Rewards) == 0 {
		ctx.Logger().Info("AllocateTokensByEpoch: there isn't any rewards to distribute, skipping the avs", "isDogfood", isDogfood, "avs", avs)
		return nil
	}

	// this function will be called by the epoch hook, so using cache context
	// to ensure the state atomicity.
	cc, writeFunc := ctx.CacheContext()
	// update the reward asset state
	for _, token := range rewardAndProportions.Rewards {
		assetID, err := k.GetAVSRewardAssetIDBySymbol(ctx, avs, token.Denom)
		if err != nil {
			return err
		}
		err = k.UpdateAVSRewardAssetState(cc, avs, assetID, &types.DeltaAVSRewardAssetState{
			RewardAllocationTotal: token.Amount,
		})
		if err != nil {
			return err
		}
	}
	if len(rewardAndProportions.OperatorRewardProportions) == 0 {
		// distribute the rewards to the community pool
		err := k.UpdateAVSCommunityPool(cc, avs, true, rewardAndProportions.Rewards)
		if err != nil {
			return err
		}
		ctx.Logger().Info("AllocateTokensByEpoch: add all rewards to the avs fee pool when the operator rewards proportion hasn't been configured", "avs", avs, "err", err)
		writeFunc()
		return nil
	}
	remaining, err := k.AllocateRewardsToOperators(cc, avs, epochIdentifier, rewardAndProportions)
	if err != nil {
		return err
	}
	if len(remaining) != 0 {
		// add the remaining rewards to the community pool
		err = k.UpdateAVSCommunityPool(cc, avs, true, remaining)
		if err != nil {
			return err
		}
	}
	writeFunc()
	return nil
}

// AllocateRewardsToOperators allocate the rewards to the related operators for an AVS
// the remaining rewards will be returned.
func (k Keeper) AllocateRewardsToOperators(ctx sdk.Context, avsAddr, epochIdentifier string, rewardsAndProportions types.EpochRewardsAndProportions) (sdk.DecCoins, error) {
	// calculate the community tax, then allocate the remaining rewards to the operators.
	// use a same community tax for all AVS
	// todo: consider setting different tax rates for different AVSs.
	communityTax, err := k.GetCommunityTax(ctx)
	if err != nil {
		return nil, types.ErrFailedToAllocateRewardsForOperators.Wrapf("failed to get the community tax,err:%s", err)
	}
	remaining := rewardsAndProportions.Rewards
	proportion := math.LegacyOneDec().Sub(communityTax)
	rewardsForOperators := rewardsAndProportions.Rewards.MulDecTruncate(proportion)

	for _, operatorProportion := range rewardsAndProportions.OperatorRewardProportions {
		reward := rewardsForOperators.MulDecTruncate(operatorProportion.RewardProportion)
		// calculate the commission for the operator
		ops, err := k.StakingKeeper.OperatorInfo(ctx, operatorProportion.OperatorAddr)
		if err != nil {
			return nil, types.ErrFailedToAllocateRewardsForOperators.Wrapf("failed to get operator info,operator:%s,err:%s", operatorProportion.OperatorAddr, err)
		}
		rewardsForStakers := reward
		commission := reward.MulDecTruncate(ops.GetCommission().Rate)
		err = k.UpdateOperatorAccumulatedCommission(ctx, operatorProportion.OperatorAddr, avsAddr, true, commission)
		if err != nil {
			return nil, types.ErrFailedToAllocateRewardsForOperators.Wrapf("failed to distribute the commission to the operator,operator:%s,err:%s", operatorProportion.OperatorAddr, err)
		}
		// update current commission
		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				types.EventTypeCommission,
				sdk.NewAttribute(sdk.AttributeKeyAmount, commission.String()),
				sdk.NewAttribute(types.AttributeKeyOperator, operatorProportion.OperatorAddr),
				sdk.NewAttribute(types.AttributeKeyAvsAddress, avsAddr),
			),
		)

		rewardsForStakers = rewardsForStakers.Sub(commission)
		// split the reward to multiple assets pool
		leftover, err := k.SplitRewardsToAssetsPool(ctx, operatorProportion.OperatorAddr, avsAddr, epochIdentifier, rewardsForStakers)
		if err != nil {
			return nil, types.ErrFailedToAllocateRewardsForOperators.Wrapf("SplitRewardsToAssetsPool,avs:%s,operator:%s,err:%s", avsAddr, operatorProportion.OperatorAddr, err)
		}
		// update the outstanding rewards for the operator
		err = k.UpdateOperatorOutstandingRewards(ctx, operatorProportion.OperatorAddr, avsAddr, true, reward)
		if err != nil {
			return nil, types.ErrFailedToAllocateRewardsForOperators.Wrapf("failed to update the operator outstanding rewards,operator:%s,err:%s", operatorProportion.OperatorAddr, err)
		}
		// calculate the remaining  rewards, it will be distributed to the community pool.
		remaining = remaining.Sub(reward).Add(leftover...)
	}
	return remaining, nil
}

// SplitRewardsToAssetsPool : split the rewards to multiple assets pool, then the reward of each
// asset pool can be allocated to the stakers whose staking has changed through F1 distribution.
// After distribution, the remaining leftover rewards will be returned to be accounted for in the community pool.
func (k Keeper) SplitRewardsToAssetsPool(ctx sdk.Context, operator, avsAddr, epochIdentifier string, rewards sdk.DecCoins) (sdk.DecCoins, error) {
	// split the rewards by multiple assets
	// get the list of assets supported by the AVS at the time of the recent ended epoch.
	// because the voting power update is per epoch.
	assets, err := k.operatorKeeper.GetRecentEndedEpochAVSAssets(ctx, avsAddr)
	if err != nil {
		return nil, err
	}
	// get the operator opted USD value
	optedUSDValue, err := k.operatorKeeper.GetOperatorOptedUSDValue(ctx, avsAddr, operator)
	if err != nil {
		return nil, err
	}
	remaining := rewards
	// calculate and set the rewards for each asset.
	for _, assetID := range assets {
		if !k.operatorKeeper.HasOperatorAssetUSDValue(ctx, epochIdentifier, operator, assetID) {
			// no rewards for assets that are not owned by the operator.
			continue
		}
		// get the USD value for asset
		assetUSDValue, err := k.operatorKeeper.GetOperatorAssetUSDValue(ctx, epochIdentifier, operator, assetID)
		if err != nil {
			return nil, err
		}
		if assetUSDValue.IsZero() {
			// no rewards for assets with a zero USD value.
			ctx.Logger().Info("SplitRewardsToAssetsPool: no rewards for assets with a zero USD value.", "EpochIdentifier", epochIdentifier, "operator", operator, "assetID", assetID)
			continue
		} else if assetUSDValue.GT(optedUSDValue.ActiveUSDValue) ||
			assetUSDValue.IsNegative() {
			// The opted USD value is the sum of the USD values of multiple assets, so the USD value of
			// each individual asset must be less than or equal to the opted USD value.
			return nil, types.ErrInvalidAssetUSDValue.Wrapf("error in SplitRewardsToAssetsPool,assetUSDValue:%s,operatorUSDValue:%s", assetUSDValue, optedUSDValue.ActiveUSDValue)
		}

		assetRewards := rewards.MulDecTruncate(assetUSDValue.QuoTruncate(optedUSDValue.ActiveUSDValue))
		if assetRewards.IsAllPositive() {
			if !k.isOperatorPeriodInitialized(ctx, operator, assetID, epochIdentifier) {
				// Initialize the currentRewardRatio currentRewards and period of the operator.
				// This case occurs when distributing rewards to an operator for the first time.
				// At this point, the operator's previous rewards should be zero,
				// and no currentRewardRatio currentRewards state has been recorded.
				err = k.initializeOperatorPeriod(ctx, operator, assetID, epochIdentifier)
				if err != nil {
					return nil, err
				}
			}
			err = k.UpdateOperatorCurrentRewards(
				ctx, operator, assetID, epochIdentifier,
				true, types.CommonAVSRewardData{
					Rewards:    assetRewards,
					AVSAddress: avsAddr,
				})
			if err != nil {
				return nil, err
			}
			remaining = remaining.Sub(assetRewards)
		} else {
			ctx.Logger().Error("SplitRewardsToAssetsPool: assetRewards isn't all positive")
		}
	}
	return remaining, nil
}
