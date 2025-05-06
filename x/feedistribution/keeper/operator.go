package keeper

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	feedistributiontypes "github.com/imua-xyz/imuachain/x/feedistribution/types"
	operatortypes "github.com/imua-xyz/imuachain/x/operator/types"
)

func (k Keeper) initializeOperatorPeriod(ctx sdk.Context, operator, assetID, epochIdentifier string) error {
	// initialize the historical rewards
	// the period in the historical rewards starts from 0
	err := k.SetOperatorHistoricalRewards(ctx, operator, assetID, epochIdentifier, 0,
		feedistributiontypes.OperatorHistoricalRewards{
			CumulativeRewardRatios: make([]feedistributiontypes.CommonAVSRewardData, 0),
			// set the reference count to 1 because it will be referenced by the current reward.
			ReferenceCount: 1,
		})
	if err != nil {
		return err
	}
	// initialize the current rewards
	err = k.SetOperatorCurrentRewards(ctx, operator, assetID, epochIdentifier,
		feedistributiontypes.OperatorCurrentRewards{
			Rewards: make([]feedistributiontypes.CommonAVSRewardData, 0),
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
	preDelegationAmount sdk.Dec,
) (uint64, error) {
	if preDelegationAmount.IsNegative() {
		return 0, feedistributiontypes.ErrInvalidInputParameter.Wrapf(
			"IncrementOperatorPeriod, the previous delegation amount is negative, amount:%s", preDelegationAmount)
	}
	if !k.HasOperatorCurrentRewards(ctx, operator, assetID, epochIdentifier) {
		// Initialize the currentRewardRatio currentRewards and period of the operator.
		// This case occurs when processing an operator's delegation changes for the first time.
		// At this point, the operator's previous delegation amount should be zero,
		// and no currentRewardRatio currentRewards state has been recorded.
		return 0, k.initializeOperatorPeriod(ctx, operator, assetID, epochIdentifier)
	}
	// fetch currentRewardRatio currentRewards
	currentRewards, err := k.GetOperatorCurrentRewards(ctx, operator, assetID, epochIdentifier)
	if err != nil {
		return 0, err
	}

	// calculate currentRewardRatio reward ratio
	var currentRewardRatio []feedistributiontypes.CommonAVSRewardData
	if preDelegationAmount.IsZero() {
		if len(currentRewards.Rewards) != 0 {
			// This case shouldn't exist; if this exception occurs, we distribute these currentRewards to the community pool
			// because we can't calculate the ratio for zero-token operators.
			ctx.Logger().Info("IncrementOperatorPeriod, the previous total delegation amount is zero but the currentRewards isn't null")
			err = k.RedirectOperatorRewardsToCommunityPool(ctx, operator, currentRewards.Rewards)
			if err != nil {
				ctx.Logger().Error("IncrementOperatorPeriod: Failed to redirect the operator currentRewards to the community pool", "error", err, "operator", operator)
				return 0, err
			}
		}
		// currentRewardRatio reward ratio should be null
		currentRewardRatio = make([]feedistributiontypes.CommonAVSRewardData, 0)
	} else {
		currentRewardRatio, err = feedistributiontypes.CommonAVSRewards(currentRewards.Rewards).CalculateRewardRatio(preDelegationAmount)
		if err != nil {
			return 0, err
		}
	}

	// fetch historical currentRewards for last period
	historicalReward, err := k.GetOperatorHistoricalRewards(ctx, operator, assetID, epochIdentifier, currentRewards.Period-1)
	if err != nil {
		return 0, err
	}
	// decrement reference count
	err = k.decrementReferenceCount(ctx, operator, assetID, epochIdentifier, currentRewards.Period-1)
	if err != nil {
		return 0, err
	}

	// set new historical currentRewards with reference count of 1
	// because it will be referenced by the current period
	currentCumulativeRatios := feedistributiontypes.CommonAVSRewards(currentRewardRatio).Add(historicalReward.CumulativeRewardRatios...)
	err = k.SetOperatorHistoricalRewards(ctx, operator, assetID, epochIdentifier, currentRewards.Period,
		feedistributiontypes.OperatorHistoricalRewards{
			CumulativeRewardRatios: currentCumulativeRatios,
			ReferenceCount:         1,
		})
	if err != nil {
		return 0, err
	}

	// set currentRewards for the operator, incrementing period by 1
	err = k.SetOperatorCurrentRewards(ctx, operator, assetID, epochIdentifier,
		feedistributiontypes.OperatorCurrentRewards{
			Rewards: make([]feedistributiontypes.CommonAVSRewardData, 0),
			Period:  currentRewards.Period + 1,
		})
	if err != nil {
		return 0, err
	}

	return currentRewards.Period, nil
}

// decrement the reference count for a historical rewards value, and delete if zero references remain
func (k Keeper) decrementReferenceCount(ctx sdk.Context, operator, assetID, epochIdentifier string,
	period uint64,
) error {
	historical, err := k.GetOperatorHistoricalRewards(ctx, operator, assetID, epochIdentifier, period)
	if err != nil {
		return err
	}
	if historical.ReferenceCount == 0 {
		return feedistributiontypes.ErrInvalidInputParameter.Wrapf("decrementReferenceCount, cannot set negative reference count")
	}
	historical.ReferenceCount--
	if historical.ReferenceCount == 0 {
		err = k.DeleteOperatorHistoricalRewards(ctx, operator, assetID, epochIdentifier, period)
	} else {
		err = k.SetOperatorHistoricalRewards(ctx, operator, assetID, epochIdentifier, period, historical)
	}
	return err
}

// increment the reference count for a historical rewards value
func (k Keeper) incrementReferenceCount(ctx sdk.Context, operator, assetID, epochIdentifier string,
	period uint64,
) error {
	historical, err := k.GetOperatorHistoricalRewards(ctx, operator, assetID, epochIdentifier, period)
	if err != nil {
		return err
	}
	// In the implementation of cosmos-sdk, it checks whether the reference count is greater than 2
	// before increasing it. In cosmos-sdk, reward distribution is handled per block, so each delegation
	// changes the operator's total delegation amount, which results in a new period being created.
	// This ensures that a period is referenced by at most one delegation and the current rewards,
	// meaning the count must be less than or equal to 2.
	// In the Imua protocol, rewards are distributed per epoch, so a period may be referenced
	// by multiple delegations. Therefore, we do not check the upper limit of the reference count here.
	historical.ReferenceCount++
	return k.SetOperatorHistoricalRewards(ctx, operator, assetID, epochIdentifier, period, historical)
}

func (k Keeper) RedirectOperatorRewardsToCommunityPool(ctx sdk.Context, operator string,
	rewards []feedistributiontypes.CommonAVSRewardData,
) error {
	for _, avsReward := range rewards {
		if len(avsReward.Rewards) != 0 {
			// distribute the rewards to the community pool
			err := k.UpdateAVSCommunityPool(ctx, avsReward.AVSAddress, true, avsReward.Rewards)
			if err != nil {
				return err
			}
			// update the outstanding rewards for the operator
			err = k.UpdateOperatorOutstandingRewards(ctx, operator, avsReward.AVSAddress, false, avsReward.Rewards)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (k Keeper) getOperatorCurrentDelegatedAmount(ctx sdk.Context, operator sdk.AccAddress, assetID string) (sdk.Dec, error) {
	// the delegation amount doesn't have any change.
	assetInfo, err := k.assetsKeeper.GetStakingAssetInfo(ctx, assetID)
	if err != nil {
		return sdk.ZeroDec(), err
	}
	operatorAssetInfo, err := k.assetsKeeper.GetOperatorSpecifiedAssetInfo(ctx, operator, assetID)
	if err != nil {
		return sdk.ZeroDec(), err
	}
	return feedistributiontypes.ScaleIntByDecimals(operatorAssetInfo.TotalAmount, assetInfo.AssetBasicInfo.Decimals), nil
}

func (k Keeper) getDelegatedAmountAtPreEpochEnd(ctx sdk.Context, operator, assetID, epochIdentifier string) (sdk.Dec, error) {
	// get the delegation amount at the end of the previous epoch.
	if k.HasStakeChangedDelegations(ctx, epochIdentifier, operator, assetID) {
		delegationChangeInfo, err := k.GetStakeChangedDelegations(ctx, epochIdentifier, operator, assetID)
		if err != nil {
			return sdk.ZeroDec(), err
		}
		return delegationChangeInfo.TotalAmount, nil
	}
	operatorAccAddr, err := sdk.AccAddressFromBech32(operator)
	if err != nil {
		return sdk.ZeroDec(), err
	}
	return k.getOperatorCurrentDelegatedAmount(ctx, operatorAccAddr, assetID)
}

// HandleOperatorSlashEvent handles the slash event for an operator.
// It increases the period and reference count, then stores the slash event
// for future reward calculations.
func (k Keeper) HandleOperatorSlashEvent(ctx sdk.Context, operator sdk.AccAddress, slashProportion sdk.Dec,
	slashAssetsPool []operatortypes.SlashFromAssetsPool,
) error {
	if slashProportion.GT(math.LegacyOneDec()) || slashProportion.IsNegative() {
		return feedistributiontypes.ErrInvalidInputParameter.Wrapf(
			"HandleOperatorSlashEvent: fraction must be >=0 and <=1, current fraction: %s", slashProportion)
	}
	// the slash event will influence all epochs
	allEpochIdentifiers := k.avsKeeper.GetEpochsUsedByAllAVSs(ctx)
	for _, slashAsset := range slashAssetsPool {
		curDelegationAmount, err := k.getOperatorCurrentDelegatedAmount(ctx, operator, slashAsset.AssetID)
		if err != nil {
			return err
		}
		var preDelegationAmount sdk.Dec
		for _, epochIdentifier := range allEpochIdentifiers {
			epochInfo, exist := k.epochsKeeper.GetEpochInfo(ctx, epochIdentifier)
			if !exist {
				return feedistributiontypes.ErrEpochNotFound.Wrapf("HandleOperatorSlashEvent, epochIdentifier:%s", epochIdentifier)
			}
			// get the delegation amount at the end of the previous epoch.
			if k.HasStakeChangedDelegations(ctx, epochInfo.Identifier, operator.String(), slashAsset.AssetID) {
				delegationChangeInfo, err := k.GetStakeChangedDelegations(ctx, epochInfo.Identifier, operator.String(), slashAsset.AssetID)
				if err != nil {
					return err
				}
				preDelegationAmount = delegationChangeInfo.TotalAmount
			} else {
				// the delegation amount doesn't have any change.
				preDelegationAmount = curDelegationAmount
			}
			// increase the periods for the slashed operator and assets.
			// because the total asset amount is changed.
			endingPeriod, err := k.IncrementOperatorPeriod(ctx, operator.String(), slashAsset.AssetID, epochInfo.Identifier, preDelegationAmount)
			if err != nil {
				return err
			}
			// increment reference count on period we need to track
			err = k.incrementReferenceCount(ctx, operator.String(), slashAsset.AssetID, epochInfo.Identifier, endingPeriod)
			if err != nil {
				return err
			}
			err = k.SetOperatorSlashEvent(ctx, operator.String(), slashAsset.AssetID, epochInfo.Identifier, uint64(epochInfo.CurrentEpoch),
				feedistributiontypes.OperatorSlashEvent{
					OperatorPeriod: endingPeriod,
					Fraction:       slashProportion,
				})
			if err != nil {
				return err
			}
		}
	}

	// clear the delegation changes for all epochs
	for _, epochIdentifier := range allEpochIdentifiers {
		err := k.DeleteStakeChangedDelegationsByEpoch(ctx, epochIdentifier)
		if err != nil {
			return err
		}
	}
	return nil
}
