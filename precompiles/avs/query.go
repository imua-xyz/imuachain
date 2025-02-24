package avs

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	cmn "github.com/evmos/evmos/v16/precompiles/common"
	imuacmn "github.com/imua-xyz/imuachain/precompiles/common"
	avstype "github.com/imua-xyz/imuachain/x/avs/types"
)

const (
	MethodGetRegisteredPubKey      = "getRegisteredPubkey"
	MethodGetOptInOperators        = "getOptInOperators"
	MethodGetAVSUSDValue           = "getAVSUSDValue"
	MethodGetOperatorOptedUSDValue = "getOperatorOptedUSDValue"

	MethodGetAVSEpochIdentifier       = "getAVSEpochIdentifier"
	MethodGetTaskInfo                 = "getTaskInfo"
	MethodIsOperator                  = "isOperator"
	MethodGetCurrentEpoch             = "getCurrentEpoch"
	MethodGetOperatorTaskResponseList = "getOperatorTaskResponseList"
	MethodGetOperatorTaskResponse     = "getOperatorTaskResponse"
	MethodGetChallengeInfo            = "getChallengeInfo"
)

func (p Precompile) GetRegisteredPubKey(
	ctx sdk.Context,
	_ *vm.Contract,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	if len(args) != len(p.ABI.Methods[MethodGetRegisteredPubKey].Inputs) {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, len(p.ABI.Methods[MethodGetRegisteredPubKey].Inputs), len(args))
	}
	operatorAddress, ok := args[0].(common.Address)
	if !ok || operatorAddress == (common.Address{}) {
		return nil, fmt.Errorf(imuacmn.ErrContractInputParamOrType, 0, "common.Address", operatorAddress)
	}
	avsAddress, ok := args[1].(common.Address)
	if !ok {
		return nil, fmt.Errorf(imuacmn.ErrContractInputParamOrType, 1, "common.Address", avsAddress)
	}
	var accAddress sdk.AccAddress = operatorAddress[:]
	blsPubKeyInfo, err := p.avsKeeper.GetOperatorPubKey(ctx, accAddress.String(), avsAddress.String())
	if err != nil {
		if errors.Is(err, avstype.ErrNoKeyInTheStore) {
			return method.Outputs.Pack([]byte{})
		}
		return nil, err
	}
	return method.Outputs.Pack(blsPubKeyInfo.PubKey)
}

func (p Precompile) GetOptedInOperatorAccAddresses(
	ctx sdk.Context,
	_ *vm.Contract,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	if len(args) != len(p.ABI.Methods[MethodGetOptInOperators].Inputs) {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, len(p.ABI.Methods[MethodGetOptInOperators].Inputs), len(args))
	}

	avsAddress, ok := args[0].(common.Address)
	if !ok || avsAddress == (common.Address{}) {
		return nil, fmt.Errorf(imuacmn.ErrContractInputParamOrType, 0, "common.Address", avsAddress)
	}

	list, err := p.avsKeeper.GetOperatorKeeper().GetOptedInOperatorListByAVS(ctx, strings.ToLower(avsAddress.String()))
	if err != nil {
		return nil, err
	}
	commonAddressList := make([]common.Address, 0)
	for _, operatorAddressStr := range list {
		acc, err := sdk.AccAddressFromBech32(operatorAddressStr)
		if err != nil {
			return nil, err
		}
		operatorAddress := common.BytesToAddress(acc)
		commonAddressList = append(commonAddressList, operatorAddress)
	}
	return method.Outputs.Pack(commonAddressList)
}

// GetAVSUSDValue is a function to retrieve the USD share of specified Avs,
func (p Precompile) GetAVSUSDValue(
	ctx sdk.Context,
	_ *vm.Contract,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	if len(args) != len(p.ABI.Methods[MethodGetAVSUSDValue].Inputs) {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, len(p.ABI.Methods[MethodGetAVSUSDValue].Inputs), len(args))
	}
	avsAddress, ok := args[0].(common.Address)
	if !ok {
		return nil, fmt.Errorf(imuacmn.ErrContractInputParamOrType, 0, "common.Address", avsAddress)
	}
	amount, err := p.avsKeeper.GetOperatorKeeper().GetAVSUSDValue(ctx, avsAddress.String())
	if err != nil {
		if errors.Is(err, avstype.ErrNoKeyInTheStore) {
			return method.Outputs.Pack(common.Big0)
		}
		return nil, err
	}
	return method.Outputs.Pack(amount.BigInt())
}

// GetOperatorOptedUSDValue is a function to retrieve the USD share of specified operator and Avs,
func (p Precompile) GetOperatorOptedUSDValue(
	ctx sdk.Context,
	_ *vm.Contract,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	if len(args) != len(p.ABI.Methods[MethodGetOperatorOptedUSDValue].Inputs) {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, len(p.ABI.Methods[MethodGetOperatorOptedUSDValue].Inputs), len(args))
	}
	avsAddress, ok := args[0].(common.Address)
	if !ok {
		return nil, fmt.Errorf(imuacmn.ErrContractInputParamOrType, 0, "common.Address", avsAddress)
	}
	operatorAddress, ok := args[1].(common.Address)
	if !ok {
		return nil, fmt.Errorf(imuacmn.ErrContractInputParamOrType, 1, "common.Address", operatorAddress)
	}
	var accAddress sdk.AccAddress = operatorAddress[:]
	amount, err := p.avsKeeper.GetOperatorKeeper().GetOperatorOptedUSDValue(ctx, strings.ToLower(avsAddress.String()), accAddress.String())
	if err != nil {
		if errors.Is(err, avstype.ErrNoKeyInTheStore) {
			return method.Outputs.Pack(common.Big0)
		}
		return nil, err
	}
	return method.Outputs.Pack(amount.ActiveUSDValue.BigInt())
}

func (p Precompile) GetAVSEpochIdentifier(
	ctx sdk.Context,
	_ *vm.Contract,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	if len(args) != len(p.ABI.Methods[MethodGetAVSEpochIdentifier].Inputs) {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, len(p.ABI.Methods[MethodGetAVSEpochIdentifier].Inputs), len(args))
	}
	avsAddress, ok := args[0].(common.Address)
	if !ok {
		return nil, fmt.Errorf(imuacmn.ErrContractInputParamOrType, 0, "common.Address", avsAddress)
	}

	avs, err := p.avsKeeper.GetAVSInfo(ctx, avsAddress.String())
	if err != nil {
		// if the avs does not exist, return empty array
		if errors.Is(err, avstype.ErrNoKeyInTheStore) {
			return method.Outputs.Pack("")
		}
		return nil, err
	}

	return method.Outputs.Pack(avs.GetInfo().EpochIdentifier)
}

func (p Precompile) IsOperator(
	ctx sdk.Context,
	_ *vm.Contract,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	if len(args) != len(p.ABI.Methods[MethodIsOperator].Inputs) {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, len(p.ABI.Methods[MethodIsOperator].Inputs), len(args))
	}
	operatorAddress, ok := args[0].(common.Address)
	if !ok {
		return nil, fmt.Errorf(imuacmn.ErrContractInputParamOrType, 0, "common.Address", operatorAddress)
	}

	param := operatorAddress[:]
	flag := p.avsKeeper.GetOperatorKeeper().IsOperator(ctx, param)

	return method.Outputs.Pack(flag)
}

func (p Precompile) GetTaskInfo(
	ctx sdk.Context,
	_ *vm.Contract,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	if len(args) != len(p.ABI.Methods[MethodGetTaskInfo].Inputs) {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, len(p.ABI.Methods[MethodGetTaskInfo].Inputs), len(args))
	}
	taskAddress, ok := args[0].(common.Address)
	if !ok {
		return nil, fmt.Errorf(imuacmn.ErrContractInputParamOrType, 0, "common.Address", taskAddress)
	}
	taskID, ok := args[1].(uint64)
	if !ok {
		return nil, fmt.Errorf(imuacmn.ErrContractInputParamOrType, 1, "uint64", taskID)
	}

	task, err := p.avsKeeper.GetTaskInfo(ctx, strconv.FormatUint(taskID, 10), taskAddress.String())
	if err != nil {
		return nil, err
	}
	var param []*avstype.OperatorActivePowerInfo
	if task.OperatorActivePower != nil {
		param = task.OperatorActivePower.OperatorPowerList
	}
	// Pack the values into the struct
	result := TaskInfo{
		TaskContractAddress:     common.HexToAddress(task.TaskContractAddress),
		Name:                    task.Name,
		Hash:                    task.Hash,
		TaskID:                  task.TaskId,
		TaskResponsePeriod:      task.TaskResponsePeriod,
		TaskStatisticalPeriod:   task.TaskStatisticalPeriod,
		TaskChallengePeriod:     task.TaskChallengePeriod,
		ThresholdPercentage:     uint8(task.ThresholdPercentage),
		StartingEpoch:           task.StartingEpoch,
		ActualThreshold:         task.ActualThreshold,
		OptInOperators:          p.stringToAddress(task.OptInOperators),
		SignedOperators:         p.stringToAddress(task.SignedOperators),
		NoSignedOperators:       p.stringToAddress(task.NoSignedOperators),
		ErrSignedOperators:      p.stringToAddress(task.ErrSignedOperators),
		TaskTotalPower:          task.TaskTotalPower.String(),
		OperatorActivePower:     ParseActivePower(param),
		IsExpected:              task.IsExpected,
		EligibleRewardOperators: p.stringToAddress(task.EligibleRewardOperators),
		EligibleSlashOperators:  p.stringToAddress(task.EligibleSlashOperators),
	}
	return method.Outputs.Pack(result)
}

func ParseActivePower(list []*avstype.OperatorActivePowerInfo) []OperatorActivePower {
	if len(list) == 0 {
		return nil
	}
	result := make([]OperatorActivePower, len(list))
	for i, info := range list {
		result[i] = OperatorActivePower{
			Operator: common.HexToAddress(info.OperatorAddress),
			Power:    info.SelfActivePower.BigInt(),
		}
	}
	return result
}

// stringToAddress is a helper function to convert a slice of strings to a slice of common.Address.
func (p Precompile) stringToAddress(addresses []string) []common.Address {
	if len(addresses) == 0 {
		return nil
	}
	result := make([]common.Address, len(addresses))
	for i, address := range addresses {
		accAddress, _ := sdk.AccAddressFromBech32(address)
		result[i] = common.BytesToAddress(accAddress)
	}
	return result
}

// GetCurrentEpoch obtain the specified current epoch based on epochIdentifier.
func (p Precompile) GetCurrentEpoch(
	ctx sdk.Context,
	_ *vm.Contract,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	if len(args) != len(p.ABI.Methods[MethodGetCurrentEpoch].Inputs) {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, len(p.ABI.Methods[MethodGetCurrentEpoch].Inputs), len(args))
	}
	epochIdentifier, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf(imuacmn.ErrContractInputParamOrType, 0, "string", epochIdentifier)
	}
	epoch, flag := p.avsKeeper.GetEpochKeeper().GetEpochInfo(ctx, epochIdentifier)
	if !flag {
		return nil, errorsmod.Wrap(avstype.ErrNoKeyInTheStore, fmt.Sprintf("GetCurrentEpoch: epochIdentifier is %s", epochIdentifier))
	}
	return method.Outputs.Pack(epoch.CurrentEpoch)
}

func (p Precompile) GetOperatorTaskResponseList(
	ctx sdk.Context,
	_ *vm.Contract,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	if len(args) != len(p.ABI.Methods[MethodGetOperatorTaskResponseList].Inputs) {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, len(p.ABI.Methods[MethodGetOperatorTaskResponseList].Inputs), len(args))
	}
	taskAddress, ok := args[0].(common.Address)
	if !ok {
		return nil, fmt.Errorf(imuacmn.ErrContractInputParamOrType, 0, "common.Address", taskAddress)
	}
	taskID, ok := args[1].(uint64)
	if !ok {
		return nil, fmt.Errorf(imuacmn.ErrContractInputParamOrType, 1, "uint64", taskID)
	}

	task, err := p.avsKeeper.GetTaskInfo(ctx, strconv.FormatUint(taskID, 10), taskAddress.String())
	if err != nil {
		return nil, err
	}

	resList := make([]OperatorResInfo, 0)

	for _, accAddress := range task.SignedOperators {
		tempAddress, err := sdk.AccAddressFromBech32(accAddress)
		if err != nil {
			return nil, err
		}
		res, err := p.avsKeeper.GetTaskResultInfo(ctx, accAddress, taskAddress.String(), taskID)
		if err != nil {
			return nil, err
		}
		power := common.Big0
		for _, a := range task.OperatorActivePower.OperatorPowerList {
			if a.OperatorAddress == accAddress {
				power = a.SelfActivePower.BigInt()
				break
			}
		}
		// Pack the values into the struct
		result := OperatorResInfo{
			TaskContractAddress: taskAddress,
			TaskID:              taskID,
			OperatorAddress:     common.BytesToAddress(tempAddress),
			TaskResponseHash:    res.TaskResponseHash,
			TaskResponse:        res.TaskResponse,
			BlsSignature:        res.BlsSignature,
			Power:               power,
			Phase:               uint8(res.Phase),
		}

		resList = append(resList, result)
	}
	return method.Outputs.Pack(resList)
}

func (p Precompile) GetOperatorTaskResponse(
	ctx sdk.Context,
	_ *vm.Contract,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	if len(args) != len(p.ABI.Methods[MethodGetOperatorTaskResponse].Inputs) {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, len(p.ABI.Methods[MethodGetOperatorTaskResponse].Inputs), len(args))
	}
	taskAddress, ok := args[0].(common.Address)
	if !ok {
		return nil, fmt.Errorf(imuacmn.ErrContractInputParamOrType, 0, "common.Address", taskAddress)
	}
	operatorAddress, ok := args[1].(common.Address)
	if !ok {
		return nil, fmt.Errorf(imuacmn.ErrContractInputParamOrType, 1, "common.Address", operatorAddress)
	}
	taskID, ok := args[2].(uint64)
	if !ok {
		return nil, fmt.Errorf(imuacmn.ErrContractInputParamOrType, 2, "uint64", taskID)
	}
	var accAddress sdk.AccAddress = operatorAddress[:]

	res, err := p.avsKeeper.GetTaskResultInfo(ctx, accAddress.String(), taskAddress.String(), taskID)
	if err != nil {
		return nil, err
	}
	// Pack the values into the struct
	result := TaskResultInfo{
		OperatorAddress:     operatorAddress,
		TaskResponseHash:    res.TaskResponseHash,
		TaskResponse:        res.TaskResponse,
		BlsSignature:        res.BlsSignature,
		TaskContractAddress: taskAddress,
		TaskID:              taskID,
		Phase:               uint8(res.Phase),
	}
	return method.Outputs.Pack(result)
}

func (p Precompile) GetChallengeInfo(
	ctx sdk.Context,
	_ *vm.Contract,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	if len(args) != len(p.ABI.Methods[MethodGetChallengeInfo].Inputs) {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, len(p.ABI.Methods[MethodGetChallengeInfo].Inputs), len(args))
	}
	taskAddress, ok := args[0].(common.Address)
	if !ok {
		return nil, fmt.Errorf(imuacmn.ErrContractInputParamOrType, 0, "common.Address", taskAddress)
	}
	taskID, ok := args[1].(uint64)
	if !ok {
		return nil, fmt.Errorf(imuacmn.ErrContractInputParamOrType, 1, "uint64", taskID)
	}

	addr, err := p.avsKeeper.GetTaskChallengedInfo(ctx, taskAddress.String(), taskID)
	if err != nil {
		return nil, err
	}

	return method.Outputs.Pack(common.HexToAddress(addr))
}
