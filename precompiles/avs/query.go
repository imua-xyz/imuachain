package avs

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	cmn "github.com/evmos/evmos/v16/precompiles/common"
	exocmn "github.com/imua-xyz/imuachain/precompiles/common"
)

const (
	MethodGetRegisteredPubkey      = "getRegisteredPubkey"
	MethodGetOptinOperators        = "getOptInOperators"
	MethodGetAVSUSDValue           = "getAVSUSDValue"
	MethodGetOperatorOptedUSDValue = "getOperatorOptedUSDValue"
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
	// the key is set using the operator's acc address so the same logic should apply here
	addr, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf(exocmn.ErrContractInputParaOrType, 0, "string", addr)
	}
	blsPubkeyInfo, err := p.avsKeeper.GetOperatorPubKey(ctx, addr)
	if err != nil {
		return nil, err
	}
	return method.Outputs.Pack(blsPubkeyInfo.PubKey)
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
		return nil, fmt.Errorf(exocmn.ErrContractInputParaOrType, 0, "string", addr)
	}

	list, err := p.avsKeeper.GetOptInOperators(ctx, addr.String())
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
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, len(p.ABI.Methods[MethodRegisterAVS].Inputs), len(args))
	}
	addr, ok := args[0].(common.Address)
	if !ok {
		return nil, fmt.Errorf(exocmn.ErrContractInputParaOrType, 0, "common.Address", addr)
	}
	amount, err := p.avsKeeper.GetOperatorKeeper().GetAVSUSDValue(ctx, addr.String())
	if err != nil {
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
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, len(p.ABI.Methods[MethodRegisterAVS].Inputs), len(args))
	}
	avsAddr, ok := args[0].(common.Address)
	if !ok {
		return nil, fmt.Errorf(exocmn.ErrContractInputParaOrType, 0, "common.Address", avsAddr)
	}
	operatorAddr, ok := args[1].(string)
	if !ok {
		return nil, fmt.Errorf(exocmn.ErrContractInputParaOrType, 1, "string", operatorAddr)
	}
	amount, err := p.avsKeeper.GetOperatorKeeper().GetOperatorOptedUSDValue(ctx, avsAddr.String(), operatorAddr)
	if err != nil {
		return nil, err
	}
	return method.Outputs.Pack(amount.ActiveUSDValue.BigInt())
}
