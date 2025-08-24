package keeper

import (
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/imua-xyz/imuachain/utils"
	feedistributiontypes "github.com/imua-xyz/imuachain/x/feedistribution/types"
	oracletype "github.com/imua-xyz/imuachain/x/oracle/types"
)

func (k *Keeper) HandleOperatorRewardsUSDValues(
	ctx sdk.Context, receivingAVS, rewardSourceAVS, operator string,
	unclaimedRewards feedistributiontypes.OperatorUnclaimedRewards,
	validRewardUSDs map[string]interface{},
	calculateUSDValue func(avs, symbol string, amount sdk.Dec) (sdkmath.LegacyDec, error),
) (sdkmath.LegacyDec, error) {
	totalUSDValue := sdkmath.LegacyZeroDec()
	// iterate over the outstanding rewards
	for _, outstandingReward := range unclaimedRewards.OutstandingRewards {
		// handle the outstanding rewards earned from staking token.
		outstandingUSDValue, err := calculateUSDValue(rewardSourceAVS, outstandingReward.Denom, outstandingReward.Amount)
		if err != nil {
			return sdkmath.LegacyDec{}, err
		}

		// handle the rewards earned by compounding
		compoundingRewards := feedistributiontypes.CompoundingRewards(unclaimedRewards.RewardsFromCompounding).RewardsOf(outstandingReward.Denom)
		compoundingUSDValue := sdkmath.LegacyZeroDec()
		for _, rewardsPerAsset := range compoundingRewards {
			for _, reward := range rewardsPerAsset.Rewards {
				usdValuePerAsset, err := calculateUSDValue(rewardsPerAsset.AVSAddress, reward.Denom, reward.Amount)
				if err != nil {
					return sdkmath.LegacyDec{}, err
				}
				compoundingUSDValue.AddMut(usdValuePerAsset)
			}
		}
		// set the USD value for specific AVS reward asset
		totalCompoundingUSDValue := compoundingUSDValue.Add(outstandingUSDValue)
		if totalCompoundingUSDValue.IsPositive() {
			err = k.operatorKeeper.SetOperatorRewardUSDValue(ctx, receivingAVS, rewardSourceAVS, operator, outstandingReward.Denom, totalCompoundingUSDValue)
			if err != nil {
				return sdkmath.LegacyDec{}, err
			}
			totalUSDValue.AddMut(totalCompoundingUSDValue)
			key := string(utils.GetJoinedStoreKey(receivingAVS, operator, rewardSourceAVS, outstandingReward.Denom))
			validRewardUSDs[key] = nil
		}
	}
	return totalUSDValue, nil
}

// UpdateAllRewardsUSDForOperator calculate and update all compounding rewards USD values for operator
// The rewards USD value of every AVS will be stored to calculate the compounding rewards.
// And the total USD values from all AVS rewards will be returned.
func (k *Keeper) UpdateAllRewardsUSDForOperator(
	ctx sdk.Context,
	receivingAVS, operator string,
	assetsMap map[string]interface{},
) (sdkmath.LegacyDec, error) {
	assetPrices := make(map[string]oracletype.Price, 0)
	calculateUSDValue := func(avs, symbol string, amount sdk.Dec) (sdkmath.LegacyDec, error) {
		if !amount.IsPositive() {
			ctx.Logger().Info("UpdateAllRewardsUSDForOperator: skip the reward with no-positive amount", "avs", avs, "symbol", symbol)
			return sdkmath.LegacyZeroDec(), nil
		}
		// get the assetID by rewardSourceAVS and symbol
		assetID, err := k.GetAVSRewardAssetIDBySymbol(ctx, avs, symbol)
		if err != nil {
			return sdkmath.LegacyDec{}, err
		}
		_, exist := assetsMap[assetID]
		if !exist {
			// the reward asset isn't supported by the receivingAVS, skipping it.
			return sdkmath.LegacyZeroDec(), nil
		}

		// get the price of the reward asset
		price, ok := assetPrices[assetID]
		if !ok {
			price, err = k.OracleKeeper.GetSpecifiedAssetsPrice(ctx, assetID)
			if err != nil {
				return sdkmath.LegacyDec{}, err
			}
			assetPrices[assetID] = price
		}
		if !price.Value.IsPositive() {
			// reward asset with a non-positive price can't contribute any USD value, skipping it.
			return sdkmath.LegacyZeroDec(), nil
		}
		// calculate the USD value of each reward asset
		usdPerAsset := utils.CalculateDecUSDValue(amount, price.Value, price.Decimal)
		return usdPerAsset, nil
	}

	validRewardUSDs := make(map[string]interface{}, 0)
	totalUSDValue := sdk.ZeroDec()
	opFunc := func(rewardSourceAVS string, rewards *feedistributiontypes.OperatorUnclaimedRewards) (bool, bool, error) {
		// calculate and set the USD value for specific operator and rewardSourceAVS
		avsRewardsUSD, err := k.HandleOperatorRewardsUSDValues(ctx, receivingAVS, rewardSourceAVS, operator, *rewards, validRewardUSDs, calculateUSDValue)
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
	// remove the invalid rewards USD values
	err = k.operatorKeeper.RemoveAllStaleOperatorRewardUSDs(ctx, receivingAVS, operator, validRewardUSDs)
	if err != nil {
		return sdkmath.LegacyDec{}, err
	}
	return totalUSDValue, nil
}
