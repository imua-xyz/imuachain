package keeper

import (
	"errors"
	"strconv"

	assetstype "github.com/ExocoreNetwork/exocore/x/assets/types"

	sdkmath "cosmossdk.io/math"
	operatortypes "github.com/ExocoreNetwork/exocore/x/operator/types"
	oracletypes "github.com/ExocoreNetwork/exocore/x/oracle/types"
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// VotingPowerActivationDuration indicates how many epochs after the current epoch's voting power calculation
// will be activated and used. By default, this is set to 1 for all AVS. If we want to support custom configurations
// for AVS, we need to add it to the AVS info.
const VotingPowerActivationDuration = 1

// UpdateVotingPower update the voting power of the specified AVS and its operators at
// the end of epoch.
func (k *Keeper) UpdateVotingPower(ctx sdk.Context, avsAddr, epochIdentifier string, epochNumber int64, isForSlash bool) error {
	// get assets supported by the AVS
	// the mock keeper returns all registered assets.
	assets, getAssetsErr := k.avsKeeper.GetAVSSupportedAssets(ctx, avsAddr)
	// check if self USD value is more than the minimum self delegation.
	minimumSelfDelegation, getSelfDelegationErr := k.avsKeeper.GetAVSMinimumSelfDelegation(ctx, avsAddr)
	// set the voting power to zero if an error is returned, which may prevent malicious behavior
	// where errors are intentionally triggered to avoid updating the voting power.
	if getAssetsErr != nil || assets == nil || getSelfDelegationErr != nil {
		ctx.Logger().Info("UpdateVotingPower the assets list supported by AVS is nil or can't get AVS info", "getAssetsErr", getAssetsErr, "getSelfDelegationErr", getSelfDelegationErr)
		// using cache context to ensure the atomicity of the operation.
		cc, writeFunc := ctx.CacheContext()
		// clear the voting power regarding this AVS if there isn't any assets supported by it.
		err := k.DeleteAllOperatorsUSDValueForAVS(cc, avsAddr)
		if err != nil {
			return err
		}
		err = k.DeleteAVSUSDValue(cc, avsAddr)
		if err != nil {
			return err
		}
		writeFunc()
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
	isSnapshotChanged := false
	votingPowerSet := make([]*operatortypes.OperatorVotingPower, 0)
	avsVotingPower := sdkmath.LegacyNewDec(0)
	opFunc := func(operator string, optedUSDValues *operatortypes.OperatorOptedUSDValue) error {
		// clear the old voting power for the operator
		lastOptedUSDValue := optedUSDValues
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

		// prepare the voting power set in advance
		if optedUSDValues.ActiveUSDValue.IsPositive() {
			votingPowerSet = append(votingPowerSet, &operatortypes.OperatorVotingPower{
				OperatorAddr: operator,
				VotingPower:  optedUSDValues.ActiveUSDValue,
			})
		}
		// check whether the voting power snapshot should be changed
		// The snapshot will be updated even if only one operator's active voting power changes.
		if !isSnapshotChanged && !lastOptedUSDValue.ActiveUSDValue.Equal(optedUSDValues.ActiveUSDValue) {
			isSnapshotChanged = true
		}
		return nil
	}
	// using cache context to ensure the atomicity of the operation.
	cc, writeFunc := ctx.CacheContext()
	// iterate all operators of the AVS to update their voting power
	// and calculate the voting power for AVS
	err = k.IterateOperatorsForAVS(cc, avsAddr, true, opFunc)
	if err != nil {
		return err
	}
	// set the voting power for AVS
	err = k.SetAVSUSDValue(cc, avsAddr, avsVotingPower)
	if err != nil {
		return err
	}

	// TODO: Consider not addressing the dogfood AVS, as its historical voting power
	// has already been stored by CometBFT.

	// set voting power snapshot
	// When the snapshot helper does not exist, it represents the initial state of AVS,
	// where no snapshot information has been stored. Therefore, it is necessary to store
	// both the snapshot and the helper information.
	var snapshotHelper operatortypes.SnapshotHelper
	if !k.HasSnapshotHelper(cc, avsAddr) {
		isSnapshotChanged = true
	} else {
		snapshotHelper, err = k.GetSnapshotHelper(cc, avsAddr)
		if err != nil {
			return err
		}
	}
	votingPowerSnapshot := operatortypes.VotingPowerSnapshot{
		Height: cc.BlockHeight(),
	}
	// The voting power calculated at the end of the current epoch will be
	// used in the epoch following the VotingPowerActivationDuration.
	// Therefore, when storing the voting power snapshot, we directly store
	// the voting power information that is activated for the specified epoch.
	// This way, during the slashing process, we do not need to consider the
	// impact of VotingPowerActivationDuration. It can be directly used.
	snapshotKey := assetstype.GetJoinedStoreKey(avsAddr, epochIdentifier, strconv.FormatInt(epochNumber+VotingPowerActivationDuration, 10))
	if isForSlash {
		// When generating the snapshot key, there's no need to add VotingPowerActivationDuration
		// to the epochNumber, because when a slash triggers a snapshot update, it updates the
		// voting power information used in the current epoch.
		snapshotKey = assetstype.GetJoinedStoreKey(
			avsAddr, epochIdentifier,
			strconv.FormatInt(epochNumber, 10),
			strconv.FormatInt(ctx.BlockHeight(), 10))
	}
	if snapshotHelper.HasOptOut || isSnapshotChanged {
		votingPowerSnapshot.TotalVotingPower = avsVotingPower
		votingPowerSnapshot.VotingPowerSet = votingPowerSet
		snapshotHelper.LastChangedKey = string(snapshotKey)
		// clear the hasOptOut flag if it's certain that the snapshot will be updated
		snapshotHelper.HasOptOut = false
	}
	votingPowerSnapshot.LastChangedKey = snapshotHelper.LastChangedKey
	if !isForSlash {
		// clear the slash flag at the end of the epoch
		snapshotHelper.HasSlash = false
	}
	err = k.SetSnapshotHelper(cc, avsAddr, &snapshotHelper)
	if err != nil {
		return err
	}
	err = k.SetVotingPowerSnapshot(cc, snapshotKey, &votingPowerSnapshot)
	if err != nil {
		return err
	}

	writeFunc()
	return nil
}

// EndBlock : update the assets' share when their prices change
func (k *Keeper) EndBlock(_ sdk.Context, _ abci.RequestEndBlock) []abci.ValidatorUpdate {
	return []abci.ValidatorUpdate{}
}
