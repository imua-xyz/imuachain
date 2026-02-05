package keeper

import (
	"sort"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/imua-xyz/imuachain/utils"
	feedistributiontypes "github.com/imua-xyz/imuachain/x/feedistribution/types"
	operatortypes "github.com/imua-xyz/imuachain/x/operator/types"
	oracletype "github.com/imua-xyz/imuachain/x/oracle/types"
)

// calculateRewardUSDValue calculates the USD value of a specific reward asset.
// `assetPrices` is used to cache prices of assets to avoid repeated queries.
// `supportedAssets` represents the set of valid assets for which the USD value will be calculated.
// If `supportedAssets` is nil, the function will always calculate the USD value of input reward asset.
// it will return the reward assetID if the calculated USD value is positive.
func (k *Keeper) calculateRewardUSDValue(
	ctx sdk.Context, avs,
	symbol string, supportedAssets map[string]struct{},
	assetPrices map[string]oracletype.Price, amount sdk.Dec,
) (string, sdkmath.LegacyDec, error) {
	if !amount.IsPositive() {
		ctx.Logger().Info("UpdateAllRewardsUSDForOperator: skip the reward with no-positive amount", "avs", avs, "symbol", symbol)
		return "", sdkmath.LegacyZeroDec(), nil
	}
	// get the assetID by rewardSourceAVS and symbol
	assetID, rewardAsset, err := k.GetAVSRewardAssetByDenomination(ctx, avs, symbol)
	if err != nil {
		return "", sdkmath.LegacyDec{}, err
	}

	if supportedAssets != nil {
		_, exist := supportedAssets[assetID]
		if !exist {
			// the reward asset isn't supported by the receivingAVS, skipping it.
			return "", sdkmath.LegacyZeroDec(), nil
		}
	}

	// get the price of the reward asset
	price, ok := assetPrices[assetID]
	if !ok {
		price, err = k.OracleKeeper.GetSpecifiedAssetsPrice(ctx, assetID)
		if err != nil {
			ctx.Logger().Error("UpdateAllRewardsUSDForOperator: failed to get the price of reward asset", "assetID", assetID, "err", err)
			// failed to get the price of reward asset, skipping it.
			return "", sdkmath.LegacyZeroDec(), nil
		}
		assetPrices[assetID] = price
	}
	if !price.Value.IsPositive() {
		// reward asset with a non-positive price can't contribute any USD value, skipping it.
		return "", sdkmath.LegacyZeroDec(), nil
	}
	// get the decimal in staking asset
	assetInfo, err := k.assetsKeeper.GetStakingAssetInfo(ctx, assetID)
	if err != nil {
		return "", sdkmath.LegacyZeroDec(), nil
	}
	// calculate the USD value of each reward asset
	usdPerAsset := utils.CalculateRewardUSDValue(amount, rewardAsset.RewardAssetInfo.DenominationExponent, assetInfo.AssetBasicInfo.Decimals, price.Value, price.Decimal)
	return assetID, usdPerAsset, nil
}

// calcOperatorRewardsUSDValue iterates operator rewards and calculates USD values.
// For each denom with positive USD value, it invokes the handler callback.
func (k *Keeper) calcOperatorRewardsUSDValue(
	ctx sdk.Context,
	rewardSourceAVS string,
	unclaimedRewards feedistributiontypes.OperatorUnclaimedRewards,
	supportedAssets map[string]struct{},
	assetPrices map[string]oracletype.Price,
	handler func(denom string, usdValue sdkmath.LegacyDec) error,
) (map[string]struct{}, sdkmath.LegacyDec, error) {
	usdValuedAssets := make(map[string]struct{})
	totalUSDValue := sdkmath.LegacyZeroDec()
	// iterate over the outstanding rewards
	for _, outstandingReward := range unclaimedRewards.OutstandingRewards {
		// handle the outstanding rewards earned from staking token.
		assetID, outstandingUSDValue, err := k.calculateRewardUSDValue(ctx, rewardSourceAVS, outstandingReward.Denom, supportedAssets, assetPrices, outstandingReward.Amount)
		if err != nil {
			return nil, sdkmath.LegacyDec{}, err
		}
		if outstandingUSDValue.IsPositive() {
			_, exist := usdValuedAssets[assetID]
			if !exist {
				usdValuedAssets[assetID] = struct{}{}
			}
		}

		// handle the rewards earned by compounding
		compoundingRewards := feedistributiontypes.CompoundingRewards(unclaimedRewards.RewardsFromCompounding).RewardsOf(outstandingReward.Denom)
		compoundingUSDValue := sdkmath.LegacyZeroDec()
		for _, rewardsPerAsset := range compoundingRewards {
			for _, reward := range rewardsPerAsset.Rewards {
				compoundingAssetID, usdValuePerAsset, err := k.calculateRewardUSDValue(ctx, rewardsPerAsset.AVSAddress, reward.Denom, supportedAssets, assetPrices, reward.Amount)
				if err != nil {
					return nil, sdkmath.LegacyDec{}, err
				}
				if usdValuePerAsset.IsPositive() {
					_, exist := usdValuedAssets[compoundingAssetID]
					if !exist {
						usdValuedAssets[compoundingAssetID] = struct{}{}
					}
					compoundingUSDValue.AddMut(usdValuePerAsset)
				}
			}
		}

		totalCompoundingUSDValue := compoundingUSDValue.Add(outstandingUSDValue)
		if totalCompoundingUSDValue.IsPositive() {
			// call the handler function
			if handler != nil {
				if err := handler(outstandingReward.Denom, totalCompoundingUSDValue); err != nil {
					return nil, sdkmath.LegacyDec{}, err
				}
			}
			totalUSDValue.AddMut(totalCompoundingUSDValue)
		}
	}

	return usdValuedAssets, totalUSDValue, nil
}

// UpdateAllRewardsUSDForOperator calculate and update all compounding rewards USD values for operator
// The rewards USD value of every AVS will be stored to calculate the compounding rewards.
// And the total USD values from all AVS rewards will be returned.
func (k *Keeper) UpdateAllRewardsUSDForOperator(
	ctx sdk.Context,
	receivingAVS, operator string,
	supportedAssets map[string]struct{},
) (sdkmath.LegacyDec, error) {
	assetPrices := make(map[string]oracletype.Price, 0)
	validRewardUSDs := make(map[string]interface{}, 0)
	totalUSDValue := sdk.ZeroDec()

	isDisableRewardsCompounding, err := k.operatorKeeper.IsCompoundRewardsDisabled(ctx, operator)
	if err != nil {
		return sdkmath.LegacyDec{}, err
	}
	if !isDisableRewardsCompounding {
		opFunc := func(rewardSourceAVS string, rewards *feedistributiontypes.OperatorUnclaimedRewards) (bool, bool, error) {
			// calculate and set the USD value for specific operator and rewardSourceAVS
			_, avsRewardsUSD, err := k.calcOperatorRewardsUSDValue(ctx, rewardSourceAVS, *rewards, supportedAssets, assetPrices,
				func(denom string, usdValue sdkmath.LegacyDec) error {
					// set the USD value for specific AVS reward asset
					if err := k.operatorKeeper.SetOperatorRewardUSDValue(ctx, receivingAVS, rewardSourceAVS, operator, denom, usdValue); err != nil {
						return err
					}
					key := string(utils.GetJoinedStoreKey(receivingAVS, operator, rewardSourceAVS, denom))
					validRewardUSDs[key] = nil
					return nil
				})
			if err != nil {
				return false, false, err
			}
			totalUSDValue.AddMut(avsRewardsUSD)
			return false, false, nil
		}
		err := k.IterateOperatorUnclaimedRewards(ctx, operator, false, opFunc)
		if err != nil {
			return sdkmath.LegacyDec{}, err
		}
	}

	// remove the invalid rewards USD values
	err = k.operatorKeeper.RemoveAllStaleOperatorRewardUSDs(ctx, receivingAVS, operator, validRewardUSDs)
	if err != nil {
		return sdkmath.LegacyDec{}, err
	}
	return totalUSDValue, nil
}

// OperatorTotalRewardsUSDValue calculates the USD value of all valid reward assets for a specific operator.
// A reward asset is considered valid only if it has been registered as a staking asset
// and its price can be obtained from the oracle module.
// It returns the USD value and the detailed source mapping: AVS -> assetID -> nil.
func (k *Keeper) OperatorTotalRewardsUSDValue(
	ctx sdk.Context, operator string,
) (map[string]map[string]struct{}, sdkmath.LegacyDec, error) {
	usdValueSources := make(map[string]map[string]struct{}, 0)
	assetPrices := make(map[string]oracletype.Price, 0)
	totalUSDValue := sdk.ZeroDec()

	opFunc := func(rewardSourceAVS string, rewards *feedistributiontypes.OperatorUnclaimedRewards) (bool, bool, error) {
		// calculate the USD value for specific operator and rewardSourceAVS
		usdValuedAssets, avsRewardsUSD, err := k.calcOperatorRewardsUSDValue(
			ctx, rewardSourceAVS, *rewards,
			nil, assetPrices, nil)
		if err != nil {
			return false, false, err
		}
		if avsRewardsUSD.IsPositive() {
			usdValueSources[rewardSourceAVS] = usdValuedAssets
			totalUSDValue.AddMut(avsRewardsUSD)
		}
		return false, false, nil
	}
	err := k.IterateOperatorUnclaimedRewards(ctx, operator, false, opFunc)
	if err != nil {
		return nil, sdkmath.LegacyDec{}, err
	}
	return usdValueSources, totalUSDValue, nil
}

// convertSlashStates converts a nested map (avs -> assetID -> amount)
// into a sorted slice of SlashFromUnclaimedRewards.
// The result is deterministic:
// - the outer slice is sorted by avs
// - the inner slice (SlashAssets) is sorted by assetID
func convertSlashStates(
	slashStatesMap map[string]map[string]sdkmath.Int,
) []operatortypes.SlashFromUnclaimedRewards {
	// collect and sort AVS keys
	avsList := make([]string, 0, len(slashStatesMap))
	for avs := range slashStatesMap {
		avsList = append(avsList, avs)
	}
	sort.Strings(avsList)

	result := make([]operatortypes.SlashFromUnclaimedRewards, 0, len(avsList))
	for _, avs := range avsList {
		assetMap := slashStatesMap[avs]

		// collect and sort asset IDs
		assetIDs := make([]string, 0, len(assetMap))
		for assetID := range assetMap {
			assetIDs = append(assetIDs, assetID)
		}
		sort.Strings(assetIDs)

		// build sorted SlashAssets
		slashAssets := make([]operatortypes.SlashAssetAmount, 0, len(assetIDs))
		for _, assetID := range assetIDs {
			slashAssets = append(slashAssets, operatortypes.SlashAssetAmount{
				AssetID: assetID,
				Amount:  assetMap[assetID],
			})
		}

		// append to result
		result = append(result, operatortypes.SlashFromUnclaimedRewards{
			Avs:         avs,
			SlashAssets: slashAssets,
		})
	}
	return result
}

func (k *Keeper) SlashOperatorUnclaimedRewards(
	ctx sdk.Context, operator string,
	slashSources map[string]map[string]struct{},
	slashProportion sdkmath.LegacyDec,
) ([]operatortypes.SlashFromUnclaimedRewards, error) {
	if slashProportion.IsNil() || slashProportion.IsZero() {
		return nil, nil
	} else if slashProportion.IsNegative() || slashProportion.GT(sdkmath.LegacyOneDec()) {
		return nil, feedistributiontypes.ErrFailedToSlashUnclaimedRewards.Wrapf("invalid slash proportion:%s", slashProportion)
	}
	slashStatesMap := make(map[string]map[string]sdkmath.Int, 0)
	opFunc := func(rewardSourceAVS string, unclaimedRewards *feedistributiontypes.OperatorUnclaimedRewards) (bool, bool, error) {
		relatedAssets, avsExist := slashSources[rewardSourceAVS]
		if !avsExist || len(relatedAssets) == 0 {
			return false, false, nil
		}
		outstandingRewardsSlashedTotal := sdk.NewDecCoins()
		compoundingRewardsSlashedTotal := feedistributiontypes.NewCompoundingRewards()
		isChanged := false
		// iterate over the outstanding rewards
		for _, outstandingReward := range unclaimedRewards.OutstandingRewards {
			if outstandingReward.Amount.IsPositive() {
				// slash from outstanding rewards
				assetID, rewardAsset, err := k.GetAVSRewardAssetByDenomination(ctx, rewardSourceAVS, outstandingReward.Denom)
				if err != nil {
					return true, false, err
				}
				_, assetExist := relatedAssets[assetID]
				if assetExist {
					slashAmountDec := outstandingReward.Amount.Mul(slashProportion)
					outstandingRewardsSlashedTotal = outstandingRewardsSlashedTotal.Add(sdk.NewDecCoinFromDec(outstandingReward.Denom, slashAmountDec))
					slashAmountInt := feedistributiontypes.UnscaleDecToInt(slashAmountDec, rewardAsset.RewardAssetInfo.DenominationExponent)

					_, slashAVSExist := slashStatesMap[rewardSourceAVS]
					if !slashAVSExist {
						slashStatesMap[rewardSourceAVS] = make(map[string]sdkmath.Int)
					}
					_, slashAssetExist := slashStatesMap[rewardSourceAVS][assetID]
					if !slashAssetExist {
						slashStatesMap[rewardSourceAVS][assetID] = sdkmath.ZeroInt()
					}
					slashStatesMap[rewardSourceAVS][assetID] = slashStatesMap[rewardSourceAVS][assetID].Add(slashAmountInt)

					if !isChanged {
						isChanged = true
					}
				}
			}
			compoundingRewards := feedistributiontypes.CompoundingRewards(unclaimedRewards.RewardsFromCompounding).RewardsOf(outstandingReward.Denom)
			// slash from the rewards earned by compounding
			for _, rewardsPerAsset := range compoundingRewards {
				for _, reward := range rewardsPerAsset.Rewards {
					assetID, rewardAsset, err := k.GetAVSRewardAssetByDenomination(ctx, rewardsPerAsset.AVSAddress, reward.Denom)
					if err != nil {
						return true, false, err
					}
					_, assetExist := relatedAssets[assetID]
					if assetExist {
						slashAmountDec := reward.Amount.Mul(slashProportion)
						newCompoundingRewardsSlashed := feedistributiontypes.CompoundingRewardsPerAsset{
							RewardDenomination: outstandingReward.Denom,
							Rewards: feedistributiontypes.NewCommonAVSRewards(
								feedistributiontypes.CommonAVSRewardData{
									AVSAddress: rewardsPerAsset.AVSAddress,
									Rewards:    sdk.NewDecCoins(sdk.NewDecCoinFromDec(reward.Denom, slashAmountDec)),
								}),
						}
						compoundingRewardsSlashedTotal = compoundingRewardsSlashedTotal.Add(newCompoundingRewardsSlashed)
						slashAmountInt := feedistributiontypes.UnscaleDecToInt(slashAmountDec, rewardAsset.RewardAssetInfo.DenominationExponent)

						_, slashAVSExist := slashStatesMap[rewardsPerAsset.AVSAddress]
						if !slashAVSExist {
							slashStatesMap[rewardsPerAsset.AVSAddress] = make(map[string]sdkmath.Int)
						}
						_, slashAssetExist := slashStatesMap[rewardsPerAsset.AVSAddress][assetID]
						if !slashAssetExist {
							slashStatesMap[rewardsPerAsset.AVSAddress][assetID] = sdkmath.ZeroInt()
						}
						slashStatesMap[rewardsPerAsset.AVSAddress][assetID] = slashStatesMap[rewardsPerAsset.AVSAddress][assetID].Add(slashAmountInt)

						if !isChanged {
							isChanged = true
						}
					}
				}
			}
		}

		if isChanged {
			unclaimedRewards.OutstandingRewardsSlashed = unclaimedRewards.OutstandingRewardsSlashed.Add(outstandingRewardsSlashedTotal...)
			unclaimedRewards.CompoundingRewardsSlashed = feedistributiontypes.CompoundingRewards(unclaimedRewards.CompoundingRewardsSlashed).Add(compoundingRewardsSlashedTotal...)

			unclaimedRewards.OutstandingRewards = unclaimedRewards.OutstandingRewards.Sub(outstandingRewardsSlashedTotal)
			unclaimedRewards.RewardsFromCompounding = feedistributiontypes.CompoundingRewards(unclaimedRewards.RewardsFromCompounding).Sub(compoundingRewardsSlashedTotal)
			return false, true, nil
		}
		return false, false, nil
	}
	err := k.IterateOperatorUnclaimedRewards(ctx, operator, true, opFunc)
	if err != nil {
		return nil, err
	}

	// convert the slashed states map to sorted slice and return.
	return convertSlashStates(slashStatesMap), nil
}
