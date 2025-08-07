package keeper

import (
	"strconv"

	"github.com/imua-xyz/imuachain/x/avs/types"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	epochstypes "github.com/imua-xyz/imuachain/x/epochs/types"
)

// EpochsHooksWrapper is the wrapper structure that implements the epochs hooks for the avs
// keeper.
type EpochsHooksWrapper struct {
	keeper *Keeper
}

// Interface guard
var _ epochstypes.EpochHooks = EpochsHooksWrapper{}

// EpochsHooks returns the epochs hooks wrapper. It follows the "accept interfaces, return
// concretes" pattern.
func (k *Keeper) EpochsHooks() EpochsHooksWrapper {
	return EpochsHooksWrapper{k}
}

// AfterEpochEnd is called after an epoch ends. It is called during the BeginBlock function.
func (wrapper EpochsHooksWrapper) AfterEpochEnd(
	ctx sdk.Context, epochIdentifier string, epochNumber int64,
) {
	// get all the task info bypass the epoch end
	// threshold calculation, signature verification, nosig quantity statistics
	taskResList := wrapper.keeper.GetTaskStatisticalEpochEndAVSs(ctx, epochIdentifier, epochNumber)
	// TODO:There should be a retry mechanism or compensation mechanism to handle cases of failure
	if len(taskResList) != 0 {
		groupedTasks := wrapper.keeper.GroupTasksByIDAndAddress(taskResList)
		for key, value := range groupedTasks {
			taskAddr, taskID, err := parseGroupKey(key)
			if err != nil {
				ctx.Logger().Error("Failed to parse group key", "key", key, "error", err)
				continue
			}
			avsInfo := wrapper.keeper.GetAVSInfoByTaskAddress(ctx, taskAddr)
			if avsInfo.AvsAddress == "" {
				ctx.Logger().Error("Failed to update task result statistics, no AVS address found for task address!", "task address", taskAddr, "task id", taskID)
				continue
			}
			avsAddr := avsInfo.AvsAddress
			taskInfo, err := wrapper.keeper.GetTaskInfo(ctx, strconv.FormatUint(taskID, 10), taskAddr)
			if err != nil {
				// Log the error and continue to the next task, and this should be an 'impossible' case, since we retrieved the taskID and taskAddr from the taskResList
				ctx.Logger().Error("Failed to update task result statistics, GetTaskInfo call failed!", "task address", taskAddr, "task id", taskID, "error", err)
				continue
			}
			taskPowerTotal, err := wrapper.keeper.operatorKeeper.GetAVSUSDValue(ctx, avsAddr)
			if err != nil || taskPowerTotal.IsZero() {
				// Log the error and continue to the next task, and this is also an 'impossible' case, since a valid task must have a non-zero total power
				ctx.Logger().Error("Failed to update task result statistics, GetAVSUSDValue call failed!", "avs address", avsAddr, "error", err)
				continue
			}

			var signedOperatorList []string
			var operatorPowers []*types.OperatorActivePowerInfo
			operatorPowerTotal := sdkmath.LegacyZeroDec()
			for _, res := range value {
				// Find signed operators
				if res.BlsSignature != nil && res.TaskResponseHash != "" || res.TaskResponse != nil {
					signedOperatorList = append(signedOperatorList, res.OperatorAddress)
					power, err := wrapper.keeper.operatorKeeper.GetOperatorOptedUSDValue(ctx, avsAddr, res.OperatorAddress)
					activePower := sdkmath.LegacyZeroDec()
					if err != nil || power.ActiveUSDValue.IsNegative() {
						// Log the error and and use 0 as the active power for this operator
						ctx.Logger().Error("Failed to get optedUSDValue for operator, skip this one", "operator", res.OperatorAddress, "avsAddr", avsAddr, "error", err)
					} else {
						activePower = power.ActiveUSDValue
					}
					operatorSelfPower := &types.OperatorActivePowerInfo{
						OperatorAddress: res.OperatorAddress,
						SelfActivePower: activePower,
					}
					operatorPowers = append(operatorPowers, operatorSelfPower)
					operatorPowerTotal = operatorPowerTotal.Add(activePower)
				}
			}

			diff := types.Difference(taskInfo.OptInOperators, signedOperatorList)
			taskInfo.SignedOperators = signedOperatorList
			// If a signature is submitted only once, it is counted as NoSignedOperators
			taskInfo.NoSignedOperators = diff
			taskInfo.OperatorActivePower = &types.OperatorActivePowerList{OperatorPowerList: operatorPowers}

			taskInfo.TaskTotalPower = taskPowerTotal
			// Update the taskInfo in the state
			err = wrapper.keeper.SetTaskInfo(ctx, taskInfo)
			if err != nil {
				ctx.Logger().Error("Failed to update task result statistics,SetTaskInfo call failed!", "task result", taskAddr, "error", err)
			}
		}
	}
}

// BeforeEpochStart is called before an epoch starts.
func (wrapper EpochsHooksWrapper) BeforeEpochStart(
	sdk.Context, string, int64,
) {
}
