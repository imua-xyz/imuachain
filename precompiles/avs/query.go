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
	addr, ok := args[0].(common.Address)
	if !ok || addr == (common.Address{}) {
		return nil, fmt.Errorf(exocmn.ErrContractInputParamOrType, 0, "common.Address", addr)
	}
	var accAddr sdk.AccAddress = addr[:]
	blsPubKeyInfo, err := p.avsKeeper.GetOperatorPubKey(ctx, accAddr.String())
	if err != nil {
		if errors.Is(err, avstype.ErrNoKeyInTheStore) {
			return method.Outputs.Pack([]byte{})
		}
		return nil, err
	}
	return method.Outputs.Pack(blsPubKeyInfo.PubKey)
}

func (p Precompile) GetOptedInOperatorAccAddrs(
	ctx sdk.Context,
	_ *vm.Contract,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	if len(args) != len(p.ABI.Methods[MethodGetOptinOperators].Inputs) {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, len(p.ABI.Methods[MethodGetOptinOperators].Inputs), len(args))
	}

	addr, ok := args[0].(common.Address)
	if !ok || addr == (common.Address{}) {
		return nil, fmt.Errorf(exocmn.ErrContractInputParamOrType, 0, "common.Address", addr)
	}

	list, err := p.avsKeeper.GetOperatorKeeper().GetOptedInOperatorListByAVS(ctx, strings.ToLower(addr.String()))
	if err != nil {
		return nil, err
	}
	return method.Outputs.Pack(list)
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
	addr, ok := args[0].(common.Address)
	if !ok {
		return nil, fmt.Errorf(exocmn.ErrContractInputParamOrType, 0, "common.Address", addr)
	}
	amount, err := p.avsKeeper.GetOperatorKeeper().GetAVSUSDValue(ctx, addr.String())
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
	avsAddr, ok := args[0].(common.Address)
	if !ok {
		return nil, fmt.Errorf(exocmn.ErrContractInputParamOrType, 0, "common.Address", avsAddr)
	}
	addr, ok := args[1].(common.Address)
	if !ok {
		return nil, fmt.Errorf(exocmn.ErrContractInputParamOrType, 1, "common.Address", addr)
	}
	var operatorAddr sdk.AccAddress = addr[:]
	amount, err := p.avsKeeper.GetOperatorKeeper().GetOperatorOptedUSDValue(ctx, strings.ToLower(avsAddr.String()), operatorAddr.String())
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
	addr, ok := args[0].(common.Address)
	if !ok {
		return nil, fmt.Errorf(exocmn.ErrContractInputParamOrType, 0, "common.Address", addr)
	}

	avs, err := p.avsKeeper.GetAVSInfo(ctx, addr.String())
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
	operatorAddr, ok := args[0].(common.Address)
	if !ok {
		return nil, fmt.Errorf(exocmn.ErrContractInputParamOrType, 0, "common.Address", operatorAddr)
	}

	param := operatorAddr[:]
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
	addr, ok := args[0].(common.Address)
	if !ok {
		return nil, fmt.Errorf(exocmn.ErrContractInputParamOrType, 0, "common.Address", addr)
	}
	taskID, ok := args[1].(uint64)
	if !ok {
		return nil, fmt.Errorf(exocmn.ErrContractInputParamOrType, 1, "uint64", taskID)
	}

	task, err := p.avsKeeper.GetTaskInfo(ctx, strconv.FormatUint(taskID, 10), addr.String())
	if err != nil {
		// if the avs does not exist, return empty array
		if errors.Is(err, avstype.ErrNoKeyInTheStore) {
			return method.Outputs.Pack("")
		}
		return nil, err
	}
	info := []uint64{task.StartingEpoch, task.TaskResponsePeriod, task.TaskStatisticalPeriod}

	return method.Outputs.Pack(info)
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
