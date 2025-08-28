package keeper

import (
	"sort"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	feedistributiontypes "github.com/imua-xyz/imuachain/x/feedistribution/types"
	operatortypes "github.com/imua-xyz/imuachain/x/operator/types"
)

func (k Keeper) isOperatorPeriodInitialized(ctx sdk.Context, operator, assetID, epochIdentifier string) bool {
	return k.HasOperatorCurrentRewards(ctx, operator, assetID, epochIdentifier)
}

func (k Keeper) initializeOperatorPeriod(ctx sdk.Context, operator, assetID, epochIdentifier string) error {
	// initialize the historical rewards
	// the period in the historical rewards starts from 0
	err := k.SetOperatorHistoricalRewards(ctx, operator, assetID, epochIdentifier, 0,
		feedistributiontypes.OperatorHistoricalRewards{
			CumulativeRewardRatios: feedistributiontypes.NewCommonAVSRewards(),
			// set the reference count to 1 because it will be referenced by the current reward.
			ReferenceCount: 1,
		})
	if err != nil {
		return err
	}
	// initialize the current rewards
	err = k.SetOperatorCurrentRewards(ctx, operator, assetID, epochIdentifier,
		feedistributiontypes.OperatorCurrentRewards{
			Rewards: feedistributiontypes.NewCommonAVSRewards(),
			// the period in current rewards starts from 1.
			Period: 1,
		})
	if err != nil {
		return err
	}

	// initialize all related delegations
	// When initializing an operator, there are three types of delegations to handle:
	// 1. Delegations created before the current epoch and not modified during this epoch.
	// 2. Delegations created before the current epoch but modified during this epoch.
	// 3. Delegations created during the current epoch.

	// The first two types should be initialized immediately after the operator is initialized,
	// because they are eligible to receive rewards from the current epoch. This ensures that
	// their starting info is correctly recorded before rewards for this epoch are distributed.

	// The third type is not initialized here. Instead, it will be initialized when processing
	// delegation changes during the current epoch. These delegations will not receive rewards
	// for the current epoch — their rewards start accumulating from the next epoch.

	// For the first type, since no change has occurred in the current epoch, we can directly use
	// the currently retrieved delegated amount as the starting stake.

	// For the second type, since a change has occurred, we must use the stored preStake as the
	// starting stake.

	// In both cases, the startingEpochNumber and the referenced period should point to the
	// previous epoch and the previous period, respectively, to ensure that rewards from the
	// current epoch are not skipped during future calculations.
	stakerList, err := k.delegationKeeper.GetStakersByOperator(ctx, operator, assetID)
	if err != nil {
		return err
	}

	changedDelegationByStaker := make(map[string]sdk.Dec)
	if k.HasStakeChangedDelegations(ctx, epochIdentifier, operator, assetID) {
		changedDelegations, err := k.GetStakeChangedDelegations(ctx, epochIdentifier, operator, assetID)
		if err != nil {
			return err
		}
		changedDelegationByStaker = changedDelegations.DelegationChangesByStaker()
	}

	currentEpochInfo, exist := k.epochsKeeper.GetEpochInfo(ctx, epochIdentifier)
	if !exist {
		return feedistributiontypes.ErrEpochNotFound.Wrapf("initializeOperatorPeriod, EpochIdentifier:%s", epochIdentifier)
	}
	startEpochNumber := uint64(currentEpochInfo.CurrentEpoch - 1)
	prePeriod := uint64(0)
	var usePreStake bool
	var preStake sdk.Dec
	for _, stakerID := range stakerList.Stakers {
		// initialize the starting info for the delegation
		stake, ok := changedDelegationByStaker[stakerID]
		if ok {
			if !stake.IsPositive() {
				// the delegation is type 3
				continue
			}
			// the delegation is type 2
			usePreStake = true
			preStake = stake
		} else {
			// the delegation is type 1
			usePreStake = false
		}
		err = k.initializeDelegationStartingInfo(ctx, operator, stakerID, assetID, epochIdentifier,
			usePreStake, preStake, startEpochNumber, prePeriod)
		if err != nil {
			return err
		}
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
		currentRewardRatio = feedistributiontypes.NewCommonAVSRewards()
	} else {
		currentRewardRatio, err = feedistributiontypes.CommonAVSRewards(currentRewards.Rewards).CalculateRewardRatio(preDelegationAmount)
		if err != nil {
			return 0, err
		}
	}

	// fetch historical currentRewards for last period
	historicalReward, err := k.GetOperatorHistoricalReward(ctx, operator, assetID, epochIdentifier, currentRewards.Period-1)
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
			Rewards: feedistributiontypes.NewCommonAVSRewards(),
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
	historical, err := k.GetOperatorHistoricalReward(ctx, operator, assetID, epochIdentifier, period)
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
	historical, err := k.GetOperatorHistoricalReward(ctx, operator, assetID, epochIdentifier, period)
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
			err = k.UpdateOperatorUnclaimedRewards(ctx, operator, avsReward.AVSAddress, false,
				feedistributiontypes.DeltaOperatorUnclaimedRewards{
					OutstandingRewards: avsReward.Rewards,
				})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (k Keeper) getDelegatedAmountAndAssetDecimal(ctx sdk.Context, operator sdk.AccAddress, assetID string) (math.Int, uint32, error) {
	// the delegation amount doesn't have any change.
	assetInfo, err := k.assetsKeeper.GetStakingAssetInfo(ctx, assetID)
	if err != nil {
		return math.Int{}, 0, err
	}
	operatorAssetInfo, err := k.assetsKeeper.GetOperatorSpecifiedAssetInfo(ctx, operator, assetID)
	if err != nil {
		return math.Int{}, 0, err
	}
	return operatorAssetInfo.TotalAmount, assetInfo.AssetBasicInfo.Decimals, nil
}

func (k Keeper) getDelegatedAmountAtPreEpochEnd(ctx sdk.Context, operator, assetID, epochIdentifier string) (sdk.Dec, error) {
	// get the delegation amount at the end of the previous epoch.
	if k.HasStakeChangedDelegations(ctx, epochIdentifier, operator, assetID) {
		delegationChangeInfo, err := k.GetStakeChangedDelegations(ctx, epochIdentifier, operator, assetID)
		if err != nil {
			return sdk.Dec{}, err
		}
		return delegationChangeInfo.TotalAmount, nil
	}
	operatorAccAddr, err := sdk.AccAddressFromBech32(operator)
	if err != nil {
		return sdk.Dec{}, err
	}
	delegatedAmount, decimal, err := k.getDelegatedAmountAndAssetDecimal(ctx, operatorAccAddr, assetID)
	if err != nil {
		return sdk.Dec{}, err
	}
	return feedistributiontypes.ScaleIntByDecimals(delegatedAmount, decimal), nil
}

// HandleOperatorSlashEvent handles the slash event for an operator.
// It increases the period and reference count, then stores the slash event
// for future reward calculations.
func (k Keeper) HandleOperatorSlashEvent(ctx sdk.Context, operator sdk.AccAddress, slashProportion sdk.Dec,
	slashAssetsPool []operatortypes.SlashAssetAmount, slashUnclaimedRewards []operatortypes.SlashFromUnclaimedRewards,
) error {
	if slashProportion.GT(math.LegacyOneDec()) || slashProportion.IsNegative() {
		return feedistributiontypes.ErrInvalidInputParameter.Wrapf(
			"HandleOperatorSlashEvent: fraction must be >=0 and <=1, current fraction: %s", slashProportion)
	}
	// get the slashed rewards asssets list from the input map `slashUnclaimedRewards`
	assetIDSet := make(map[string]struct{})
	// collect assetIDs
	for _, s := range slashUnclaimedRewards {
		for _, sa := range s.SlashAssets {
			assetIDSet[sa.AssetID] = struct{}{}
		}
	}
	// convert map keys to slice
	slashedRewardAssets := make([]string, 0, len(assetIDSet))
	for id := range assetIDSet {
		slashedRewardAssets = append(slashedRewardAssets, id)
	}
	// sort for deterministic order
	sort.Strings(slashedRewardAssets)

	// the slash event will influence all epochs
	allEpochIdentifiers := k.avsKeeper.GetEpochsUsedByAllAVSs(ctx)
	for _, slashAsset := range slashAssetsPool {
		curDelegationAmount, assetDecimal, err := k.getDelegatedAmountAndAssetDecimal(ctx, operator, slashAsset.AssetID)
		if err != nil {
			return err
		}
		var preDelegationAmount sdk.Dec
		for _, epochIdentifier := range allEpochIdentifiers {
			epochInfo, exist := k.epochsKeeper.GetEpochInfo(ctx, epochIdentifier)
			if !exist {
				return feedistributiontypes.ErrEpochNotFound.Wrapf("HandleOperatorSlashEvent, EpochIdentifier:%s", epochIdentifier)
			}
			// For the case where multiple slashes occur within a single epoch, we can still apply the same logic:
			// the accumulated rewards after the first slash should always be zero, so the reward ratio will be zero
			// regardless of the preDelegationAmount. We still store the subsequent slashes after the first one to
			// accurately calculate the stake amount after multiple slashes. Therefore, there's no need to handle
			// multiple slashes in a single epoch separately.
			// get the delegation amount at the end of the previous epoch.
			if k.HasStakeChangedDelegations(ctx, epochInfo.Identifier, operator.String(), slashAsset.AssetID) {
				delegationChangeInfo, err := k.GetStakeChangedDelegations(ctx, epochInfo.Identifier, operator.String(), slashAsset.AssetID)
				if err != nil {
					return err
				}
				preDelegationAmount = delegationChangeInfo.TotalAmount
			} else {
				// The delegation amount remains unchanged. We should use the amount before the slash is executed,
				// because the rewards being processed are from epochs prior to the slash.
				// The current delegation amount is after the slash, so we add back the slashed amount.
				preDelegationAmount = feedistributiontypes.ScaleIntByDecimals(
					curDelegationAmount.Add(slashAsset.Amount), assetDecimal)
			}
			// For a new operator, it will be initialized when rewards are first allocated to it.
			// Therefore, if a slash occurs during its first active epoch, it might not have been initialized yet.
			// We need to initialize it before creating a new period for the slash event to handle this case.
			if !k.isOperatorPeriodInitialized(ctx, operator.String(), slashAsset.AssetID, epochIdentifier) {
				// Initialize the currentRewardRatio currentRewards and period of the operator.
				// This case occurs when distributing rewards to an operator for the first time.
				// At this point, the operator's previous rewards should be zero,
				// and no currentRewardRatio currentRewards state has been recorded.
				err = k.initializeOperatorPeriod(ctx, operator.String(), slashAsset.AssetID, epochIdentifier)
				if err != nil {
					return err
				}
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
			err = k.SetOperatorSlashEvent(ctx, operator.String(), slashAsset.AssetID, epochInfo.Identifier, uint64(epochInfo.CurrentEpoch), uint64(ctx.BlockHeight()),
				feedistributiontypes.OperatorSlashEvent{
					OperatorPeriod:      endingPeriod,
					Fraction:            slashProportion,
					SlashedRewardAssets: slashedRewardAssets,
				})
			if err != nil {
				return err
			}
		}
	}
	return nil
}
