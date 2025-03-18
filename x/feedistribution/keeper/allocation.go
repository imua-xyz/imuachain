package keeper

import (
	"cosmossdk.io/math"
	"github.com/ExocoreNetwork/exocore/x/feedistribution/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// AllocateRewardsByEpoch performs reward and fee distribution to all operators for the AVS with same epoch
// configuration based on the F1 fee distribution specification.
func (k Keeper) AllocateRewardsByEpoch(ctx sdk.Context, epochIdentifier string, endingEpochNumber int64) error {
	avsList := k.avsKeeper.GetEpochEndAVSs(ctx, epochIdentifier, endingEpochNumber)
	for _, avs := range avsList {
		_, rewardDistribution, err := k.AVSRewardDistributionByParam(ctx, avs)
		if err != nil {
			ctx.Logger().Error("AllocateTokensByEpoch: failed to reward distribution by params, skipping the avs", "avs", avs, "err", err)
			continue
		}
		if len(rewardDistribution.Rewards) == 0 {
			ctx.Logger().Info("AllocateTokensByEpoch: there isn't any rewards to distribute, skipping the avs", "avs", avs)
			continue
		}
		if len(rewardDistribution.OperatorRewardProportions) == 0 {
			// distribute the rewards to the community pool
			err := k.UpdateAVSCommunityPool(ctx, avs, true, rewardDistribution.Rewards)
			if err != nil {
				ctx.Logger().Error("AllocateTokensByEpoch: failed to add rewards to the avs fee pool, skipping the avs", "avs", avs, "err", err)
			} else {
				ctx.Logger().Info("AllocateTokensByEpoch: add all rewards to the avs fee pool because of the zero or negative total voting power", "avs", avs, "err", err)
			}
			// continue distributing the rewards for the other AVSs
			continue
		}
		remaining, err := k.AllocateRewardsToOperators(ctx, avs, epochIdentifier, rewardDistribution)
		if err != nil {
			ctx.Logger().Error("AllocateTokensByEpoch: failed to distribute the rewards to operators, skipping the avs", "avs", avs, "err", err)
			// continue handling the remaining and the other AVSs
		}
		if len(remaining) != 0 {
			// add the remaining rewards to the community pool
			err := k.UpdateAVSCommunityPool(ctx, avs, true, rewardDistribution.Rewards)
			if err != nil {
				ctx.Logger().Error("AllocateTokensByEpoch: failed to add the remaining rewards to the avs fee pool, skipping the avs", "avs", avs, "err", err)
			}
		}
		// continue handling the other AVSs
	}
	return nil
}

// AllocateRewardsToOperators allocate the rewards to the related operators for an AVS
// the remaining rewards will be returned.
func (k Keeper) AllocateRewardsToOperators(ctx sdk.Context, avsAddr, epochIdentifier string, rewardDistribution *types.AVSRewardDistribution) (sdk.DecCoins, error) {
	// calculate the community tax, then allocate the remaining rewards to the operators.
	// use a same community tax for all AVS
	// todo: consider setting different tax rates for different AVSs.
	communityTax, err := k.GetCommunityTax(ctx)
	if err != nil {
		ctx.Logger().Error("AllocateTokensByEpoch: failed to get the community tax, skipping tha avs", "avs", avsAddr, "err", err)
		return nil, err
	}
	remaining := rewardDistribution.Rewards
	proportion := math.LegacyOneDec().Sub(communityTax)
	rewardsForOperators := rewardDistribution.Rewards.MulDecTruncate(proportion)
	for _, operatorProportion := range rewardDistribution.OperatorRewardProportions {
		reward := rewardsForOperators.MulDecTruncate(operatorProportion.RewardProportion)
		// calculate the commission for the operator
		ops, err := k.StakingKeeper.OperatorInfo(ctx, operatorProportion.OperatorAddr)
		if err != nil {
			ctx.Logger().Error("AllocateRewardsToOperators: Failed to get operator info, skipping the operator", "error", err, "operator", operatorProportion.OperatorAddr)
			// continue distributing reward for the other operators
			continue
		}
		rewardsForStakers := reward
		commission := reward.MulDecTruncate(ops.GetCommission().Rate)
		err = k.UpdateOperatorAccumulatedCommission(ctx, operatorProportion.OperatorAddr, avsAddr, true, commission)
		if err != nil {
			ctx.Logger().Error("AllocateRewardsToOperators: Failed to distribute the commission to the operator, skipping the operator", "error", err, "operator", operatorProportion.OperatorAddr, "avs", avsAddr)
			// distribute the commission to stakers if adding the commission to the operator fails.
		} else {
			rewardsForStakers = rewardsForStakers.Sub(commission)
			// update current commission
			ctx.EventManager().EmitEvent(
				sdk.NewEvent(
					types.EventTypeCommission,
					sdk.NewAttribute(sdk.AttributeKeyAmount, commission.String()),
					sdk.NewAttribute(types.AttributeKeyOperator, operatorProportion.OperatorAddr),
					sdk.NewAttribute(types.AttributeKeyAvsAddress, avsAddr),
				),
			)
		}
		// split the reward to multiple assets pool
		leftover, err := k.SplitRewardsToAssetsPool(ctx, operatorProportion.OperatorAddr, avsAddr, epochIdentifier, rewardsForStakers)
		if err != nil {
			ctx.Logger().Error("AllocateRewardsToOperators: Failed to allocate rewards to the stakers", "error", err, "operator", operatorProportion.OperatorAddr, "avs", avsAddr)
		}
		// update the outstanding rewards for the operator
		err = k.UpdateOperatorOutstandingRewards(ctx, operatorProportion.OperatorAddr, avsAddr, true, reward)
		if err != nil {
			ctx.Logger().Error("AllocateRewardsToOperators: Failed to update the operator outstanding rewards", "error", err, "operator", operatorProportion.OperatorAddr, "avs", avsAddr)
		}
		// calculate the remaining  rewards, it will be distributed to the community pool.
		remaining = remaining.Sub(reward).Add(leftover...)
	}
	return remaining, err
}

// SplitRewardsToAssetsPool : split the rewards to multiple assets pool, then the reward of each
// asset pool can be allocated to the stakers whose staking has changed through F1 distribution.
// After distribution, the remaining leftover rewards will be returned to be accounted for in the community pool.
func (k Keeper) SplitRewardsToAssetsPool(ctx sdk.Context, operator, avsAddr, epochIdentifier string, rewards sdk.DecCoins) (sdk.DecCoins, error) {
	// split the rewards by multiple assets
	// get the list of assets supported by the AVS at the time of the last voting power update.
	assets, err := k.operatorKeeper.GetLastVotingPowerAVSAssets(ctx, avsAddr)
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
			continue
		} else if assetUSDValue.GT(optedUSDValue.ActiveUSDValue) ||
			assetUSDValue.IsNegative() {
			return nil, types.ErrInvalidAssetUSDValue.Wrapf("error in SplitRewardsToAssetsPool", "assetUSDValue:%s,operatorUSDValue:%s", assetUSDValue, optedUSDValue.ActiveUSDValue)
		}
		assetRewards := rewards.MulDecTruncate(assetUSDValue.QuoTruncate(optedUSDValue.ActiveUSDValue))
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
	}
	return remaining, nil
}

func (k Keeper) AllocateTokensToSingleStaker(ctx sdk.Context, stakerAddress string, reward sdk.DecCoins) {
	logger := k.Logger()
	currentStakerRewards := k.GetStakerRewards(ctx, stakerAddress)
	currentStakerRewards.Rewards = currentStakerRewards.Rewards.Add(reward...)
	k.SetStakerRewards(ctx, stakerAddress, currentStakerRewards)
	logger.Info("allocate tokens to single staker successfully", "allocated amount is", currentStakerRewards.Rewards.String())
}
