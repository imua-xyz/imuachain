package avs

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	errorsmod "cosmossdk.io/errors"

	exocmn "github.com/ExocoreNetwork/exocore/precompiles/common"
	avstype "github.com/ExocoreNetwork/exocore/x/avs/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	cmn "github.com/evmos/evmos/v16/precompiles/common"
)

const (
	MethodGetRegisteredPubkey      = "getRegisteredPubkey"
	MethodGetOptinOperators        = "getOptInOperators"
	MethodGetAVSUSDValue           = "getAVSUSDValue"
	MethodGetOperatorOptedUSDValue = "getOperatorOptedUSDValue"

	MethodGetAVSEpochIdentifier = "getAVSEpochIdentifier"
	MethodGetTaskInfo           = "getTaskInfo"
	MethodIsOperator            = "isOperator"
	MethodGetCurrentEpoch       = "getCurrentEpoch"
)

func (p Precompile) GetRegisteredPubkey(
	ctx sdk.Context,
	_ *vm.Contract,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	if len(args) != len(p.ABI.Methods[MethodGetRegisteredPubkey].Inputs) {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, len(p.ABI.Methods[MethodGetRegisteredPubkey].Inputs), len(args))
	}
	opratorAddress, ok := args[0].(common.Address)
	if !ok || opratorAddress == (common.Address{}) {
		return nil, fmt.Errorf(exocmn.ErrContractInputParamOrType, 0, "common.Address", opratorAddress)
	}
	avsAddress, ok := args[1].(common.Address)
	if !ok {
		return nil, fmt.Errorf(exocmn.ErrContractInputParamOrType, 1, "common.Address", avsAddress)
	}
	var accAddress sdk.AccAddress = opratorAddress[:]
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
	if len(args) != len(p.ABI.Methods[MethodGetOptinOperators].Inputs) {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, len(p.ABI.Methods[MethodGetOptinOperators].Inputs), len(args))
	}

	avsAddress, ok := args[0].(common.Address)
	if !ok || avsAddress == (common.Address{}) {
		return nil, fmt.Errorf(exocmn.ErrContractInputParamOrType, 0, "common.Address", avsAddress)
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
		return nil, fmt.Errorf(exocmn.ErrContractInputParamOrType, 0, "common.Address", avsAddress)
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
		return nil, fmt.Errorf(exocmn.ErrContractInputParamOrType, 0, "common.Address", avsAddress)
	}
	operatorAddress, ok := args[1].(common.Address)
	if !ok {
		return nil, fmt.Errorf(exocmn.ErrContractInputParamOrType, 1, "common.Address", operatorAddress)
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
		return nil, fmt.Errorf(exocmn.ErrContractInputParamOrType, 0, "common.Address", avsAddress)
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
		return nil, fmt.Errorf(exocmn.ErrContractInputParamOrType, 0, "common.Address", operatorAddress)
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
		return nil, fmt.Errorf(exocmn.ErrContractInputParamOrType, 0, "common.Address", taskAddress)
	}
	taskID, ok := args[1].(uint64)
	if !ok {
		return nil, fmt.Errorf(exocmn.ErrContractInputParamOrType, 1, "uint64", taskID)
	}

	task, err := p.avsKeeper.GetTaskInfo(ctx, strconv.FormatUint(taskID, 10), taskAddress.String())
	if err != nil {
		// if the avs does not exist, return empty array
		if errors.Is(err, avstype.ErrNoKeyInTheStore) {
			return method.Outputs.Pack("")
		}
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
		result[i] = common.HexToAddress(address)
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
		return nil, fmt.Errorf(exocmn.ErrContractInputParamOrType, 0, "string", epochIdentifier)
	}
	epoch, flag := p.avsKeeper.GetEpochKeeper().GetEpochInfo(ctx, epochIdentifier)
	if !flag {
		return nil, errorsmod.Wrap(avstype.ErrNoKeyInTheStore, fmt.Sprintf("GetCurrentEpoch: epochIdentifier is %s", epochIdentifier))
	}
	return method.Outputs.Pack(epoch.CurrentEpoch)
}
