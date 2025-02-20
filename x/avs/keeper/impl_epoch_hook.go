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
		for _, value := range groupedTasks {
			var signedOperatorList []string
			var taskID uint64
			var taskAddr string
			var avsAddr string
			var operatorPowers []*types.OperatorActivePowerInfo
			operatorPowerTotal := sdkmath.LegacyZeroDec()
			for _, res := range value {
				// Find signed operators
				if res.BlsSignature != nil && res.TaskResponseHash != "" || res.TaskResponse != nil {
					signedOperatorList = append(signedOperatorList, res.OperatorAddress)
					if avsAddr == "" {
						avsInfo := wrapper.keeper.GetAVSInfoByTaskAddress(ctx, res.TaskContractAddress)
						avsAddr = avsInfo.AvsAddress
					}
					if taskID == 0 {
						taskID = res.TaskId
					}
					if taskAddr == "" {
						taskAddr = res.TaskContractAddress
					}
					power, err := wrapper.keeper.operatorKeeper.GetOperatorOptedUSDValue(ctx, avsAddr, res.OperatorAddress)
					if err != nil || power.ActiveUSDValue.IsNegative() {
						ctx.Logger().Error("Failed to update task result statistics,GetOperatorOptedUSDValue call failed!", "task result", taskAddr, "error", err)
						// Handle the error gracefully, continue to the next
						// continue
					}

					operatorSelfPower := &types.OperatorActivePowerInfo{
						OperatorAddress: res.OperatorAddress,
						SelfActivePower: power.ActiveUSDValue,
					}
					operatorPowers = append(operatorPowers, operatorSelfPower)
					operatorPowerTotal = operatorPowerTotal.Add(power.ActiveUSDValue)
				}
			}
			taskInfo, err := wrapper.keeper.GetTaskInfo(ctx, strconv.FormatUint(taskID, 10), taskAddr)
			if err != nil {
				ctx.Logger().Error("Failed to update task result statistics,GetTaskInfo call failed!", "task result", taskAddr, "error", err)
				// Handle the error gracefully, continue to the next
				// continue
			}
			diff := types.Difference(taskInfo.OptInOperators, signedOperatorList)
			taskInfo.SignedOperators = signedOperatorList
			// If a signature is submitted only once, it is counted as NoSignedOperators
			taskInfo.NoSignedOperators = diff
			taskInfo.OperatorActivePower = &types.OperatorActivePowerList{OperatorPowerList: operatorPowers}
			// Calculate actual threshold
			taskPowerTotal, err := wrapper.keeper.operatorKeeper.GetAVSUSDValue(ctx, avsAddr)

			if err != nil || taskPowerTotal.IsZero() || operatorPowerTotal.IsZero() {
				ctx.Logger().Error("Failed to update task result statistics,GetAVSUSDValue call failed!", "task result", taskAddr, "error", err)
				// Handle the error gracefully, continue to the next
				// continue
			}
			taskInfo.TaskTotalPower = taskPowerTotal
			// Update the taskInfo in the state
			err = wrapper.keeper.SetTaskInfo(ctx, taskInfo)
			if err != nil {
				ctx.Logger().Error("Failed to update task result statistics,SetTaskInfo call failed!", "task result", taskAddr, "error", err)
				// Handle the error gracefully, continue to the next
				// continue
			}
		}
	}
}

// BeforeEpochStart is called before an epoch starts.
func (wrapper EpochsHooksWrapper) BeforeEpochStart(
	sdk.Context, string, int64,
) {
}
