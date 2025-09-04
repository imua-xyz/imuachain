package keeper

import (
	"github.com/ethereum/go-ethereum/common"

	sdkmath "cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	operatortypes "github.com/imua-xyz/imuachain/x/operator/types"
)

// UpdateVotingPower update the voting power of the specified AVS and its operators at
// the end of epoch.
func (k *Keeper) UpdateVotingPower(ctx sdk.Context, avsAddr, epochIdentifier string, epochNumber int64, isForSlash bool) error {
	// get assets supported by the AVS
	avsAssetsList, avsAssetsMap, err := k.avsKeeper.GetAVSSupportedAssets(ctx, avsAddr)
	if err != nil {
		return err
	}
	// get minimum self delegation amount
	minimumSelfDelegation, err := k.avsKeeper.GetAVSMinimumSelfDelegation(ctx, avsAddr)
	if err != nil {
		return err
	}

	// update the voting power of operators and AVS
	isSnapshotChanged := false
	votingPowerSet := make([]*operatortypes.OperatorVotingPower, 0)
	avsVotingPower := sdkmath.LegacyZeroDec()
	hasOptedOperator := false
	deletedOperators := make([]string, 0)
	opFunc := func(operator string, optedUSDValues *operatortypes.OperatorOptedUSDValue) error {
		// check if the operator is opted out but not effective, the usd value of these operators
		// should be deleted when updating the voting power
		if k.IsOptedOutButNotEffective(ctx, operator, avsAddr) {
			deletedOperators = append(deletedOperators, operator)
			// mark the snapshotChanged flag
			if !isSnapshotChanged {
				isSnapshotChanged = true
			}
			// continue handle the other operators
			return nil
		}
		if !hasOptedOperator {
			hasOptedOperator = true
		}
		// clear the old voting power for the operator
		lastOptedUSDValue := *optedUSDValues
		*optedUSDValues = operatortypes.OperatorOptedUSDValue{
			TotalUSDValue:  sdkmath.LegacyZeroDec(),
			SelfUSDValue:   sdkmath.LegacyZeroDec(),
			ActiveUSDValue: sdkmath.LegacyZeroDec(),
		}
		stakingInfo, err := k.AggregateOperatorUSDValue(ctx, epochIdentifier, operator, avsAssetsList)
		if err != nil {
			return err
		}
		// calculate and store the USD value from compounding rewards
		rewardsUSDValue, err := k.distributionKeeper.UpdateAllRewardsUSDForOperator(ctx, avsAddr, operator, avsAssetsMap)
		if err != nil {
			return err
		}
		optedUSDValues.SelfUSDValue = stakingInfo.SelfStaking
		optedUSDValues.TotalUSDValue = stakingInfo.Staking.Add(rewardsUSDValue)
		// check if self USD value is more than the minimum self delegation.
		if stakingInfo.SelfStaking.GTE(minimumSelfDelegation) {
			optedUSDValues.ActiveUSDValue = optedUSDValues.TotalUSDValue
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
	err = k.IterateOperatorUSDValuesForAVS(cc, avsAddr, true, opFunc)
	if err != nil {
		return err
	}
	// Delete the USD values for the operators that have opted out in the current epoch.
	err = k.DeleteOperatorUSDValues(cc, avsAddr, deletedOperators)
	if err != nil {
		return err
	}

	// set the voting power for AVS
	err = k.SetAVSUSDValue(cc, avsAddr, avsVotingPower)
	if err != nil {
		return err
	}

	// clear the AVS asset list at the time of the last voting power update
	err = k.DeleteAllAVSAssetsPerEpoch(cc)
	if err != nil {
		return err
	}

	// TODO: Consider not addressing the dogfood AVS, as its historical voting power
	// has already been stored by CometBFT.

	// set voting power snapshot
	// When the snapshot helper does not exist, it represents the initial state of AVS,
	// where no snapshot information has been stored. Therefore, it is necessary to store
	// both the snapshot and the helper information.
	snapshotHelper := operatortypes.SnapshotHelper{}
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
	}

	// The voting power calculated at the end of the current epoch will be applied
	// to the next epoch. Therefore, when storing the voting power snapshot, we use
	// the `start_height` of the next epoch as the key. This ensures that during the
	// slashing process, there is no need to account for voting power activation delay;
	// it can be used directly.
	// Use the current height as the snapshot height when handling snapshots triggered
	// by slashing. This prevents stakers from escaping slashes through backrunning
	// undelegation.
	// Use the start height of the next epoch as the snapshot key.
	// The start height of the next epoch should be the current height,
	// as the `AfterEpochEnd` is called in the beginBlock of next epoch's start height.
	snapshotHeight := ctx.BlockHeight()
	if !isForSlash {
		// the epoch number should plus 1, as it's updated after the hook `AfterEpochEnd` is called
		votingPowerSnapshot.EpochNumber++
	}
	isSetSnapshot := true
	// For cases where there is no opt-out operation, IterateOperatorUSDValuesForAVS does not detect any voting
	// power changes, and no operator has opted into the AVS, the voting power information doesn't need
	// to be saved in the snapshot. Because it can be fetched through falling back to the last snapshot
	// where the voting power was changed.
	// In the case where the AVS no longer has any operators serving it, meaning the `hasOptedOperator`
	// flag is false, the system won't store a snapshot, even if it is a snapshot without voting power
	// information. As a result, when querying the historical voting power using snapshots, the system
	// will fall back to the last snapshot where the voting power was updated to zero.
	if isSnapshotChanged {
		votingPowerSnapshot.TotalVotingPower = avsVotingPower
		votingPowerSnapshot.OperatorVotingPowers = votingPowerSet
		snapshotHelper.LastChangedHeight = snapshotHeight
	} else if !hasOptedOperator {
		// don’t set the snapshot if no operator has opted into the AVS,
		// except for the first epoch after all operators have opted out of this AVS.
		isSetSnapshot = false
	}
	votingPowerSnapshot.LastChangedHeight = snapshotHelper.LastChangedHeight

	err = k.SetSnapshotHelper(cc, avsAddr, snapshotHelper)
	if err != nil {
		return err
	}

	if isSetSnapshot {
		snapshotKey := operatortypes.KeyForVotingPowerSnapshot(common.HexToAddress(avsAddr), snapshotHeight)
		err = k.SetVotingPowerSnapshot(cc, snapshotKey, &votingPowerSnapshot)
		if err != nil {
			return err
		}
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
	clearEpochNumber := epochInfo.CurrentEpoch - int64(unbondingDuration) // #nosec G115
	if clearEpochNumber < 0 {
		return nil
	}
	err = k.RemoveVotingPowerSnapshot(ctx, avs, clearEpochNumber)
	if err != nil {
		ctx.Logger().Error("Failed to remove voting power snapshot", "avs", avs, "error", err)
		return operatortypes.ErrFailToClearVPSnapshot.Wrapf("ClearVotingPowerSnapshot: failed to remove voting power snapshot, err:%s, avs:%s", err, avs)
	}
	return nil
}

// EndBlock : do nothing, because the voting power will be updated by epoch hook.
func (k *Keeper) EndBlock(_ sdk.Context, _ abci.RequestEndBlock) []abci.ValidatorUpdate {
	return []abci.ValidatorUpdate{}
}
