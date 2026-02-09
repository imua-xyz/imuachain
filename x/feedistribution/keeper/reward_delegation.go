package keeper

import (
	"sort"

	"github.com/ethereum/go-ethereum/common"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	assetstype "github.com/imua-xyz/imuachain/x/assets/types"
	"github.com/imua-xyz/imuachain/x/avs/types"
	delegationkeeper "github.com/imua-xyz/imuachain/x/delegation/keeper"
	delegationtype "github.com/imua-xyz/imuachain/x/delegation/types"
	feedistributiontypes "github.com/imua-xyz/imuachain/x/feedistribution/types"
)

func (k Keeper) BatchRedelegateClaimedRewards(ctx sdk.Context, epochIdentifier string, avsList, stakerList []string) error {
	// Store related reward delegations so we can handle their reward distribution,
	// using operator and assetID as map keys.
	delegationChangeInfos := make(map[string]map[string]feedistributiontypes.DelegationChangeInfo, 0)
	cc, writeFunc := ctx.CacheContext()
	// TODO: Consider adjusting the implementation to prevent a failed staker from affecting the redelegation of others.
	// iterate to handle all staker rewards
	for _, staker := range stakerList {
		// check whether the staker wants to redelegate the rewards.
		if !k.HasStakerRewardParams(ctx, staker) {
			continue
		}
		stakerRewardParams, err := k.GetStakerRewardParams(ctx, staker)
		if err != nil {
			return feedistributiontypes.ErrFailedToRedelegateRewards.Wrap(err.Error())
		}
		if !stakerRewardParams.RedelegateReward {
			continue
		}
		// check if the operator has been slashed or frozen
		operatorAccAddr, err := sdk.AccAddressFromBech32(stakerRewardParams.RedelegateOperatorAddr)
		if err != nil {
			return feedistributiontypes.ErrFailedToRedelegateRewards.Wrapf("invalid operator address:%s", err)
		}
		if k.SlashKeeper.IsOperatorFrozen(ctx, operatorAccAddr) {
			return feedistributiontypes.ErrFailedToRedelegateRewards.Wrap(delegationtype.ErrOperatorIsFrozen.Error())
		}

		for _, avs := range avsList {
			if k.HasStakerClaimedRewards(ctx, staker, avs) {
				stakerClaimedRewards, err := k.GetStakerClaimedRewards(ctx, staker, avs)
				if err != nil {
					return feedistributiontypes.ErrFailedToRedelegateRewards.Wrap(err.Error())
				}

				newOutstandingRewards := append(sdk.DecCoins(nil), stakerClaimedRewards.OutstandingRewards...)
				delegationRewardsShare := feedistributiontypes.RewardsDelegationShare{
					OperatorAddr: stakerRewardParams.RedelegateOperatorAddr,
					Shares:       sdk.NewDecCoins(),
				}
				// iterate over all reward assets to calculate the delegation amount for specific reward asset
				indicesToRemove := make([]int, 0)
				for i, reward := range stakerClaimedRewards.OutstandingRewards {
					assetID, rewardAsset, err := k.GetAVSRewardAssetByDenomination(ctx, avs, reward.Denom)
					if err != nil {
						return feedistributiontypes.ErrFailedToRedelegateRewards.Wrap(err.Error())
					}

					// check if the reward asset can be redelegated
					if k.assetsKeeper.IsStakingAsset(ctx, assetID) && reward.IsPositive() {
						rewardAmount := feedistributiontypes.UnscaleDecToInt(reward.Amount, rewardAsset.RewardAssetInfo.DenominationExponent)
						if !rewardAmount.IsPositive() {
							continue
						}
						// redelegate the reward
						share, preDelegatedAmount, err := k.delegationKeeper.DelegateTo(cc, &delegationtype.DelegationOrUndelegationParams{
							OperatorAddress: operatorAccAddr,
							OpAmount:        rewardAmount,
							RewardAsset:     true,
							RewardAssetID:   assetID,
							RewardStakerID:  staker,
						})
						if err != nil {
							return feedistributiontypes.ErrFailedToRedelegateRewards.Wrap(err.Error())
						}

						// reward redelegation may trigger a reward distribution if the operator
						// already has delegated assets.
						if k.assetsKeeper.IsOperatorAssetExist(ctx, operatorAccAddr, assetID) {
							stakingAssetInfo, err := k.assetsKeeper.GetStakingAssetInfo(ctx, assetID)
							if err != nil {
								return feedistributiontypes.ErrFailedToRedelegateRewards.Wrap(err.Error())
							}
							stakingAssetDecimal := stakingAssetInfo.AssetBasicInfo.Decimals

							_, operatorExist := delegationChangeInfos[stakerRewardParams.RedelegateOperatorAddr]
							if !operatorExist {
								delegationChangeInfos[stakerRewardParams.RedelegateOperatorAddr] = make(map[string]feedistributiontypes.DelegationChangeInfo, 0)
							}
							_, assetExist := delegationChangeInfos[stakerRewardParams.RedelegateOperatorAddr][assetID]
							if !assetExist {
								// get the operator asset amount before all delegations using `ctx` instead of `cc`
								preOperatorAssets, err := k.assetsKeeper.GetOperatorSpecifiedAssetInfo(ctx, operatorAccAddr, assetID)
								if err != nil {
									return feedistributiontypes.ErrFailedToRedelegateRewards.Wrap(err.Error())
								}
								delegationChangeInfos[stakerRewardParams.RedelegateOperatorAddr][assetID] = feedistributiontypes.DelegationChangeInfo{
									StakerDelegationChanges: make([]feedistributiontypes.StakerDelegationChange, 0),
									TotalAmount:             feedistributiontypes.ScaleIntByDecimals(preOperatorAssets.TotalAmount, stakingAssetDecimal),
								}
							}
							delegationChanges := delegationChangeInfos[stakerRewardParams.RedelegateOperatorAddr][assetID]
							delegationChanges.AppendUniqueStakerID(staker, preDelegatedAmount, stakingAssetDecimal)
							delegationChangeInfos[stakerRewardParams.RedelegateOperatorAddr][assetID] = delegationChanges
						}

						indicesToRemove = append(indicesToRemove, i)
						delegationRewardsShare.Shares = delegationRewardsShare.Shares.Add(sdk.NewDecCoinFromDec(reward.Denom, share))
					}
				}
				// Remove elements in reverse order to maintain indices
				for i := len(indicesToRemove) - 1; i >= 0; i-- {
					idx := indicesToRemove[i]
					newOutstandingRewards = append(newOutstandingRewards[:idx], newOutstandingRewards[idx+1:]...)
				}
				stakerClaimedRewards.OutstandingRewards = newOutstandingRewards
				stakerClaimedRewards.DelegationRewardsShares = feedistributiontypes.RewardsDelegationShares(stakerClaimedRewards.DelegationRewardsShares).Add(delegationRewardsShare)
				err = k.SetStakerClaimedRewards(cc, staker, avs, stakerClaimedRewards)
				if err != nil {
					return feedistributiontypes.ErrFailedToRedelegateRewards.Wrap(err.Error())
				}
			}
		}
	}

	epochInfo, exist := k.epochsKeeper.GetEpochInfo(ctx, epochIdentifier)
	if !exist {
		return feedistributiontypes.ErrFailedToRedelegateRewards.Wrap(types.ErrEpochNotFound.Error())
	}
	// handle the reward distribution resulting from the above delegations.
	// Iterate over delegationChangeInfos in deterministic order:
	// 1. Collect and sort operator keys
	// 2. For each operator, collect and sort asset keys
	// 3. Process DelegationChangeInfo by operator → asset
	operators := make([]string, 0, len(delegationChangeInfos))
	for op := range delegationChangeInfos {
		operators = append(operators, op)
	}
	sort.Strings(operators) // sort operators by lexicographical order
	for _, op := range operators {
		assetsMap := delegationChangeInfos[op]

		// collect and sort asset keys
		assets := make([]string, 0, len(assetsMap))
		for asset := range assetsMap {
			assets = append(assets, asset)
		}
		sort.Strings(assets) // sort assets by lexicographical order

		seenStakers := make(map[string]interface{}, 0)
		for _, asset := range assets {
			info := assetsMap[asset]
			// handle the delegation change for reward distribution
			_, err := k.handleOperatorAssetDelegationChanges(cc, epochInfo, seenStakers, op, asset, &info)
			if err != nil {
				// Return the error directly, unlike the error handling for staking assets.
				// Because failing to redelegate the rewards won't affect the future reward distribution
				// for staking assets. Therefore, we revert all states even if only one delegation fails.
				return err
			}
		}
	}

	writeFunc()
	return nil
}

func (k Keeper) UndelegateClaimedRewards(
	ctx sdk.Context, stakerID, assetID string,
	operatorAccAddr sdk.AccAddress, instantUnbonding bool, amount math.Int,
) error {
	if amount.IsNil() || !amount.IsPositive() {
		return feedistributiontypes.ErrFailedToUndelegateRewards.Wrapf("invalid amount: %v", amount)
	}
	lackingUndelegationAmount := amount
	totalCompletedAmount := math.ZeroInt()

	reduceDelegationShare := func(
		ctx sdk.Context,
		stakerID, assetID string,
		operatorAccAddr sdk.AccAddress,
		instantSlashRatio sdk.Dec, _ math.Int,
		preOperatorAssetState assetstype.OperatorAssetInfo,
	) ([]delegationtype.UndelegationAmountPerAVS, math.Int, error) {
		// The delegated asset might come from multiple AVSs, so we iterate over all AVSs
		// to calculate the undelegated amount from each AVS. The iteration stops once the
		// collected amount equals the expected undelegation amount. Then the undelegation
		// will be handled as a single undelegation in the delegation module.
		undelegationAmounts := make([]delegationtype.UndelegationAmountPerAVS, 0)
		opFunc := func(avs string, rewards *feedistributiontypes.StakerClaimedRewards) (bool, bool, error) {
			if !lackingUndelegationAmount.IsPositive() {
				// break the iteration once the entire expected amount has been reduced.
				return true, false, nil
			}
			if !k.IsAVSRewardAssetByAssetID(ctx, avs, assetID) {
				// continue iterating the next AVS
				return false, false, nil
			}
			rewardAsset, err := k.GetAVSRewardAsset(ctx, avs, assetID)
			if err != nil {
				return true, false, err
			}
			rewardShares := feedistributiontypes.RewardsDelegationShares(rewards.DelegationRewardsShares).DelegationSharesOf(operatorAccAddr.String())
			if rewardShares == nil {
				// continue iterating the next AVS
				return false, false, nil
			}
			assetShare := rewardShares.AmountOf(rewardAsset.RewardAssetInfo.RewardDenomination)
			if !assetShare.IsPositive() {
				// continue iterating the next AVS
				return false, false, nil
			}
			assetAmount, err := delegationkeeper.TokensFromShares(assetShare, preOperatorAssetState.TotalShare, preOperatorAssetState.TotalAmount)
			if err != nil {
				return true, false, err
			}
			if !assetAmount.IsPositive() {
				// continue iterating the next AVS
				return false, false, nil
			}
			amountFromCurAVS := math.MinInt(assetAmount, lackingUndelegationAmount)
			undelegationAmountPerAVS := delegationtype.UndelegationAmountPerAVS{
				AvsAddress:            avs,
				Amount:                amountFromCurAVS,
				ActualCompletedAmount: amountFromCurAVS,
			}

			// apply the slashing for instant reward undelegation
			if !instantSlashRatio.IsNil() {
				if instantSlashRatio.IsNegative() || instantSlashRatio.GT(math.LegacyOneDec()) {
					return true, false, feedistributiontypes.ErrFailedToUndelegateRewards.Wrapf("invalid instant slash ratio:%s", instantSlashRatio)
				} else if !instantSlashRatio.IsZero() {
					slashed := instantSlashRatio.MulInt(undelegationAmountPerAVS.Amount).TruncateInt()
					undelegationAmountPerAVS.ActualCompletedAmount = undelegationAmountPerAVS.Amount.Sub(slashed)
				}
			}

			lackingUndelegationAmount = lackingUndelegationAmount.Sub(amountFromCurAVS)
			totalCompletedAmount = totalCompletedAmount.Add(undelegationAmountPerAVS.ActualCompletedAmount)
			undelegationAmounts = append(undelegationAmounts, undelegationAmountPerAVS)

			// update the claimed rewards
			undelegateShare, err := delegationkeeper.SharesFromTokens(preOperatorAssetState.TotalShare, amountFromCurAVS, preOperatorAssetState.TotalAmount)
			if err != nil {
				return true, false, err
			}
			rewards.DelegationRewardsShares = feedistributiontypes.RewardsDelegationShares(rewards.DelegationRewardsShares).Sub(
				feedistributiontypes.RewardsDelegationShares{
					{
						OperatorAddr: operatorAccAddr.String(),
						Shares:       sdk.NewDecCoins(sdk.NewDecCoinFromDec(rewardAsset.RewardAssetInfo.RewardDenomination, undelegateShare)),
					},
				})
			rewards.PendingUndelegationRewards = rewards.PendingUndelegationRewards.Add(sdk.NewDecCoinFromDec(rewardAsset.RewardAssetInfo.RewardDenomination, feedistributiontypes.ScaleIntByDecimals(amountFromCurAVS, rewardAsset.RewardAssetInfo.DenominationExponent)))
			if !instantSlashRatio.IsNil() && instantSlashRatio.IsPositive() {
				slashedAmount := undelegationAmountPerAVS.Amount.Sub(undelegationAmountPerAVS.ActualCompletedAmount)
				rewards.PendingSlashedRewards = rewards.PendingSlashedRewards.Add(sdk.NewDecCoinFromDec(rewardAsset.RewardAssetInfo.RewardDenomination, feedistributiontypes.ScaleIntByDecimals(slashedAmount, rewardAsset.RewardAssetInfo.DenominationExponent)))
			}
			return false, true, nil
		}
		// iterate to withdraw rewards from multiple AVSs, because different AVSs might
		// use the same asset as reward.
		err := k.IterateStakerClaimedRewards(ctx, stakerID, true, opFunc)
		if err != nil {
			return nil, math.Int{}, err
		}
		if lackingUndelegationAmount.IsPositive() {
			return nil, math.Int{}, feedistributiontypes.ErrFailedToUndelegateRewards.Wrapf("not enough delegable amount, lacking: %s", lackingUndelegationAmount)
		}
		return undelegationAmounts, totalCompletedAmount, nil
	}

	err := k.delegationKeeper.UndelegateFrom(ctx, &delegationtype.DelegationOrUndelegationParams{
		OperatorAddress: operatorAccAddr,
		OpAmount:        amount,
		// The txID in the undelegation key is unnecessary after introducing a unique undelegation ID.
		// TODO: Consider removing all code related to using the txID in the undelegation key.
		TxHash:                common.Hash{},
		InstantUnbonding:      instantUnbonding,
		RewardAsset:           true,
		RewardAssetID:         assetID,
		RewardStakerID:        stakerID,
		ReduceDelegationShare: reduceDelegationShare,
	})
	if err != nil {
		return err
	}
	return nil
}

func (k Keeper) CompleteRewardUndelegation(ctx sdk.Context, record delegationtype.UndelegationRecord) error {
	if !record.RewardAsset {
		// do nothing if it isn't a reward undelegation
		return nil
	}
	// iterate over all related AVSs in the undelegation record
	for _, undelegationPerAVS := range record.RewardUndelegations {
		rewardAsset, err := k.GetAVSRewardAsset(ctx, undelegationPerAVS.AvsAddress, record.AssetId)
		if err != nil {
			return feedistributiontypes.ErrFailedToCompleteRewardsUndelegation.Wrap(err.Error())
		}

		stakerClaimedRewards, err := k.GetStakerClaimedRewards(ctx, record.StakerId, undelegationPerAVS.AvsAddress)
		if err != nil {
			return feedistributiontypes.ErrFailedToCompleteRewardsUndelegation.Wrap(err.Error())
		}

		// update pendingUndelegationRewards
		subRewards := sdk.NewDecCoins(sdk.NewDecCoinFromDec(rewardAsset.RewardAssetInfo.RewardDenomination, feedistributiontypes.ScaleIntByDecimals(undelegationPerAVS.Amount, rewardAsset.RewardAssetInfo.DenominationExponent)))
		pendingUndelegationRewards, isNegative := stakerClaimedRewards.PendingUndelegationRewards.SafeSub(subRewards)
		if isNegative {
			return feedistributiontypes.ErrFailedToCompleteRewardsUndelegation.Wrapf("pending undelegation rewards have negative amount after update,pendingUndelegationRewards:%s", pendingUndelegationRewards)
		}
		stakerClaimedRewards.PendingUndelegationRewards = pendingUndelegationRewards

		// update pendingSlashedRewards
		slashedAmount := undelegationPerAVS.Amount.Sub(undelegationPerAVS.ActualCompletedAmount)
		slashedRewards := sdk.NewDecCoins(sdk.NewDecCoinFromDec(rewardAsset.RewardAssetInfo.RewardDenomination, feedistributiontypes.ScaleIntByDecimals(slashedAmount, rewardAsset.RewardAssetInfo.DenominationExponent)))
		pendingSlashedRewards, isNegative := stakerClaimedRewards.PendingSlashedRewards.SafeSub(slashedRewards)
		if isNegative {
			return feedistributiontypes.ErrFailedToCompleteRewardsUndelegation.Wrapf("pending slashed rewards have negative amount after update,pendingSlashedRewards:%s", pendingSlashedRewards)
		}
		stakerClaimedRewards.PendingSlashedRewards = pendingSlashedRewards

		// update withdrawableRewards
		stakerClaimedRewards.WithdrawableRewards = stakerClaimedRewards.WithdrawableRewards.Add(sdk.NewDecCoinFromDec(rewardAsset.RewardAssetInfo.RewardDenomination, feedistributiontypes.ScaleIntByDecimals(undelegationPerAVS.ActualCompletedAmount, rewardAsset.RewardAssetInfo.DenominationExponent)))

		err = k.SetStakerClaimedRewards(ctx, record.StakerId, undelegationPerAVS.AvsAddress, stakerClaimedRewards)
		if err != nil {
			return feedistributiontypes.ErrFailedToCompleteRewardsUndelegation.Wrap(err.Error())
		}
	}
	return nil
}

func (k Keeper) SlashRewardUndelegation(ctx sdk.Context, record *delegationtype.UndelegationRecord, slashProportion math.LegacyDec) error {
	if record == nil || !record.RewardAsset {
		// do nothing if it isn't a reward undelegation
		return nil
	}
	// iterate over all related AVSs in the undelegation record
	for i, undelegationPerAVS := range record.RewardUndelegations {
		rewardAsset, err := k.GetAVSRewardAsset(ctx, undelegationPerAVS.AvsAddress, record.AssetId)
		if err != nil {
			return feedistributiontypes.ErrFailedToCompleteRewardsUndelegation.Wrap(err.Error())
		}
		stakerClaimedRewards, err := k.GetStakerClaimedRewards(ctx, record.StakerId, undelegationPerAVS.AvsAddress)
		if err != nil {
			return feedistributiontypes.ErrFailedToCompleteRewardsUndelegation.Wrap(err.Error())
		}
		// calculate the slashed reward amount from each AVS
		if undelegationPerAVS.ActualCompletedAmount.IsZero() {
			// do nothing because there isn't amount to be slashed
			continue
		}

		// update pendingUndelegationRewards and the completed amount in the input undelegation record
		expectedSlashAmount := slashProportion.MulInt(undelegationPerAVS.Amount).TruncateInt()
		actualSlashAmount := math.MinInt(expectedSlashAmount, undelegationPerAVS.ActualCompletedAmount)
		record.RewardUndelegations[i].ActualCompletedAmount = undelegationPerAVS.ActualCompletedAmount.Sub(actualSlashAmount)
		stakerClaimedRewards.PendingSlashedRewards = stakerClaimedRewards.PendingSlashedRewards.Add(sdk.NewDecCoinFromDec(rewardAsset.RewardAssetInfo.RewardDenomination, feedistributiontypes.ScaleIntByDecimals(actualSlashAmount, rewardAsset.RewardAssetInfo.DenominationExponent)))

		err = k.SetStakerClaimedRewards(ctx, record.StakerId, undelegationPerAVS.AvsAddress, stakerClaimedRewards)
		if err != nil {
			return feedistributiontypes.ErrFailedToCompleteRewardsUndelegation.Wrap(err.Error())
		}
	}
	return nil
}
