package keeper

import (
	"errors"
	"strconv"
	"time"

	assetstype "github.com/ExocoreNetwork/exocore/x/assets/types"

	sdkmath "cosmossdk.io/math"
	operatortypes "github.com/ExocoreNetwork/exocore/x/operator/types"
	oracletypes "github.com/ExocoreNetwork/exocore/x/oracle/types"
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

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
		EpochIdentifier: epochIdentifier,
		EpochNumber:     epochNumber,
		BlockTime:       ctx.BlockTime(),
	}

	// The voting power calculated at the end of the current epoch will be applied
	// to the next epoch. Therefore, when storing the voting power snapshot, we use
	// the `start_height` of the next epoch as the key. This ensures that during the
	// slashing process, there is no need to account for voting power activation delay;
	// it can be used directly.
	// Use the current height as the snapshot height when handling snapshots triggered
	// by slashing. This prevents stakers from escaping slashes through backrunning
	// undelegation.
	snapshotHeight := ctx.BlockHeight()
	if !isForSlash {
		// clear the slash flag at the end of the epoch
		snapshotHelper.HasSlash = false
		// Use the start height of the next epoch as the snapshot key.
		// The start height of the next epoch should be the current height plus 1,
		// as voting power is updated at the end of the epoch.
		snapshotHeight++
		votingPowerSnapshot.EpochNumber++
	}
	if snapshotHelper.HasOptOut || isSnapshotChanged {
		votingPowerSnapshot.TotalVotingPower = avsVotingPower
		votingPowerSnapshot.VotingPowerSet = votingPowerSet
		snapshotHelper.LastChangedHeight = snapshotHeight
		// clear the hasOptOut flag if it's certain that the snapshot will be updated
		snapshotHelper.HasOptOut = false
	}
	votingPowerSnapshot.LastChangedHeight = snapshotHelper.LastChangedHeight

	err = k.SetSnapshotHelper(cc, avsAddr, &snapshotHelper)
	if err != nil {
		return err
	}
	snapshotKey := assetstype.GetJoinedStoreKey(avsAddr, strconv.FormatInt(snapshotHeight, 10))
	err = k.SetVotingPowerSnapshot(cc, snapshotKey, &votingPowerSnapshot)
	if err != nil {
		return err
	}

	writeFunc()
	return nil
}

func (k *Keeper) ClearVotingPowerSnapshot(ctx sdk.Context, avs string) error {
	// calculate the time before which the snapshot should be cleared.
	unbondingDuration, err := k.avsKeeper.GetAVSUnbondingDuration(ctx, avs)
	if err != nil {
		return operatortypes.ErrFailToClearVPSnapshot.Wrapf("ClearVotingPowerSnapshot: failed to get the avs unbonding duration, err:%s, avs:%s", err, avs)
	}
	epochInfo, err := k.avsKeeper.GetAVSEpochInfo(ctx, avs)
	if err != nil {
		return operatortypes.ErrFailToClearVPSnapshot.Wrapf("ClearVotingPowerSnapshot: failed to get the avs epoch information, err:%s, avs:%s", err, avs)
	}

	clearTime := ctx.BlockTime().Add(-epochInfo.Duration * time.Duration(unbondingDuration))
	err = k.RemoveVotingPowerSnapshot(ctx, avs, clearTime)
	if err != nil {
		ctx.Logger().Error("Failed to get the avs epoch information", "avs", avs, "error", err)
		return operatortypes.ErrFailToClearVPSnapshot.Wrapf("ClearVotingPowerSnapshot: failed to remove voting power snapshot, err:%s, avs:%s", err, avs)
	}
	return nil
}

// EndBlock : update the assets' share when their prices change
func (k *Keeper) EndBlock(_ sdk.Context, _ abci.RequestEndBlock) []abci.ValidatorUpdate {
	return []abci.ValidatorUpdate{}
}
