package keeper

import (
	"errors"

	sdkmath "cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	operatortypes "github.com/imua-xyz/imuachain/x/operator/types"
	oracletypes "github.com/imua-xyz/imuachain/x/oracle/types"
)

// UpdateVotingPower update the voting power of the specified AVS and its operators at
// the end of epoch.
func (k *Keeper) UpdateVotingPower(ctx sdk.Context, avsAddr string) error {
	// get assets supported by the AVS
	// the mock keeper returns all registered assets.
	assets, err := k.avsKeeper.GetAVSSupportedAssets(ctx, avsAddr)
	if err != nil {
		return err
	}
	if assets == nil {
		ctx.Logger().Info("UpdateVotingPower the assets list supported by AVS is nil")
		// clear the voting power regarding this AVS if there isn't any assets supported by it.
		err = k.DeleteAllOperatorsUSDValueForAVS(ctx, avsAddr)
		if err != nil {
			return err
		}
		err = k.DeleteAVSUSDValue(ctx, avsAddr)
		if err != nil {
			return err
		}
		return nil
	}

	// get the prices and decimals of assets
	decimals, err := k.assetsKeeper.GetAssetsDecimal(ctx, assets)
	if err != nil {
		return err
	}
	prices, err := k.oracleKeeper.GetMultipleAssetsPrices(ctx, assets)
	// TODO: for now, we ignore the error when the price round is not found and set the price to 1 to avoid panic
	if err != nil {
		// TODO: when assetID is not registered in oracle module, this error will finally lead to panic
		if !errors.Is(err, oracletypes.ErrGetPriceRoundNotFound) {
			ctx.Logger().Error("fail to get price from oracle, since current assetID is not bonded with oracle token", "details:", err)
			return err
		}
		// TODO: for now, we ignore the error when the price round is not found and set the price to 1 to avoid panic
	}
	// update the voting power of operators and AVS
	avsVotingPower := sdkmath.LegacyNewDec(0)
	// check if self USD value is more than the minimum self delegation.
	minimumSelfDelegation, err := k.avsKeeper.GetAVSMinimumSelfDelegation(ctx, avsAddr)
	if err != nil {
		return err
	}
	opFunc := func(operator string, optedUSDValues *operatortypes.OperatorOptedUSDValue) error {
		// clear the old voting power for the operator
		*optedUSDValues = operatortypes.OperatorOptedUSDValue{
			TotalUSDValue:  sdkmath.LegacyNewDec(0),
			SelfUSDValue:   sdkmath.LegacyNewDec(0),
			ActiveUSDValue: sdkmath.LegacyNewDec(0),
		}
		stakingInfo, err := k.CalculateUSDValueForOperator(ctx, false, operator, assets, decimals, prices)
		if err != nil {
			return err
		}
		optedUSDValues.SelfUSDValue = stakingInfo.SelfStaking
		optedUSDValues.TotalUSDValue = stakingInfo.Staking
		if stakingInfo.SelfStaking.GTE(minimumSelfDelegation) {
			optedUSDValues.ActiveUSDValue = stakingInfo.Staking
			avsVotingPower = avsVotingPower.Add(optedUSDValues.TotalUSDValue)
		}
		return nil
	}
	// iterate all operators of the AVS to update their voting power
	// and calculate the voting power for AVS
	err = k.IterateOperatorsForAVS(ctx, avsAddr, true, opFunc)
	if err != nil {
		return err
	}
	// set the voting power for AVS
	err = k.SetAVSUSDValue(ctx, avsAddr, avsVotingPower)
	if err != nil {
		return err
	}
	return nil
}

// EndBlock : update the assets' share when their prices change
func (k *Keeper) EndBlock(_ sdk.Context, _ abci.RequestEndBlock) []abci.ValidatorUpdate {
	return []abci.ValidatorUpdate{}
}
