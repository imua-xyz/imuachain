package delegation

import (
	"fmt"
	"reflect"

	exocmn "github.com/ExocoreNetwork/exocore/precompiles/common"
	cmn "github.com/evmos/evmos/v16/precompiles/common"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

const (
	// MethodDelegate defines the ABI method name for the
	// Delegate transaction.
	MethodDelegate = "delegate"

	// MethodUndelegate defines the ABI method name for the
	// Undelegate transaction.
	MethodUndelegate = "undelegate"

	// MethodAssociateOperatorWithStaker defines the ABI method name for the
	// associateOperatorWithStaker transaction.
	MethodAssociateOperatorWithStaker = "associateOperatorWithStaker"

	// MethodDissociateOperatorFromStaker defines the ABI method name for the
	// dissociateOperatorFromStaker transaction.
	MethodDissociateOperatorFromStaker = "dissociateOperatorFromStaker"

	CtxKeyTxHash = "TxHash"
)

// Delegate the client chain assets to the operator through client chain, that will change the states in delegation and assets module
func (p Precompile) Delegate(
	ctx sdk.Context,
	_ common.Address,
	contract *vm.Contract,
	_ vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	// check the invalidation of caller contract
	authorized, err := p.assetsKeeper.IsAuthorizedGateway(ctx, contract.CallerAddress)
	if err != nil || !authorized {
		return nil, fmt.Errorf(exocmn.ErrContractCaller)
	}

	delegationParams, err := p.GetDelegationParamsFromInputs(ctx, args)
	if err != nil {
		return nil, err
	}

	err = p.delegationKeeper.DelegateTo(ctx, delegationParams)
	if err != nil {
		return nil, err
	}
	return method.Outputs.Pack(true)
}

// Undelegate the client chain assets from the operator through client chain, that will change the states in delegation and assets module
func (p Precompile) Undelegate(
	ctx sdk.Context,
	_ common.Address,
	contract *vm.Contract,
	_ vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	// check the invalidation of caller contract
	authorized, err := p.assetsKeeper.IsAuthorizedGateway(ctx, contract.CallerAddress)
	if err != nil || !authorized {
		return nil, fmt.Errorf(exocmn.ErrContractCaller)
	}

	undelegationParams, err := p.GetDelegationParamsFromInputs(ctx, args)
	if err != nil {
		return nil, err
	}

	txHash, ok := ctx.Value(CtxKeyTxHash).(common.Hash)
	if !ok || txHash.Bytes() == nil {
		return nil, fmt.Errorf(ErrCtxTxHash, reflect.TypeOf(ctx.Value(CtxKeyTxHash)), txHash)
	}
	undelegationParams.TxHash = txHash

	err = p.delegationKeeper.UndelegateFrom(ctx, undelegationParams)
	if err != nil {
		return nil, err
	}
	return method.Outputs.Pack(true)
}

func (p Precompile) AssociateOperatorWithStaker(
	ctx sdk.Context,
	_ common.Address,
	contract *vm.Contract,
	_ vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	// check the invalidation of caller contract
	authorized, err := p.assetsKeeper.IsAuthorizedGateway(ctx, contract.CallerAddress)
	if err != nil || !authorized {
		return nil, fmt.Errorf(exocmn.ErrContractCaller)
	}

	inputsLen := len(p.ABI.Methods[MethodAssociateOperatorWithStaker].Inputs)
	if len(args) != inputsLen {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, inputsLen, len(args))
	}
	clientChainID, ok := args[0].(uint32)
	if !ok {
		return nil, fmt.Errorf(exocmn.ErrContractInputParaOrType, 0, "uint32", args[0])
	}

	info, err := p.assetsKeeper.GetClientChainInfoByIndex(ctx, uint64(clientChainID))
	if err != nil {
		return nil, err
	}
	clientChainAddrLength := info.AddressLength
	staker, ok := args[1].([]byte)
	if !ok || staker == nil {
		return nil, fmt.Errorf(exocmn.ErrContractInputParaOrType, 1, "[]byte", args[1])
	}
	// #nosec G115
	if uint32(len(staker)) < clientChainAddrLength {
		return nil, fmt.Errorf(exocmn.ErrInvalidAddrLength, len(staker), clientChainAddrLength)
	}
	staker = staker[:clientChainAddrLength]

	operator, ok := args[2].([]byte)
	if !ok || operator == nil {
		return nil, fmt.Errorf(exocmn.ErrContractInputParaOrType, 2, "[]byte", args[2])
	}
	operatorAccAddr, err := sdk.AccAddressFromBech32(string(operator))
	if err != nil {
		return nil, fmt.Errorf("failed to parse the operator address from Bech32, operator:%s, error:%s ", operator, err.Error())
	}
	err = p.delegationKeeper.AssociateOperatorWithStaker(ctx, uint64(clientChainID), operatorAccAddr, staker)
	if err != nil {
		return nil, err
	}
	return method.Outputs.Pack(true)
}

func (p Precompile) DissociateOperatorFromStaker(
	ctx sdk.Context,
	_ common.Address,
	contract *vm.Contract,
	_ vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	// check the invalidation of caller contract
	authorized, err := p.assetsKeeper.IsAuthorizedGateway(ctx, contract.CallerAddress)
	if err != nil || !authorized {
		return nil, fmt.Errorf(exocmn.ErrContractCaller)
	}
	inputsLen := len(p.ABI.Methods[MethodDissociateOperatorFromStaker].Inputs)
	if len(args) != inputsLen {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, inputsLen, len(args))
	}
	clientChainID, ok := args[0].(uint32)
	if !ok {
		return nil, fmt.Errorf(exocmn.ErrContractInputParaOrType, 0, "uint32", args[0])
	}
	info, err := p.assetsKeeper.GetClientChainInfoByIndex(ctx, uint64(clientChainID))
	if err != nil {
		return nil, err
	}
	clientChainAddrLength := info.AddressLength

	staker, ok := args[1].([]byte)
	if !ok || staker == nil {
		return nil, fmt.Errorf(exocmn.ErrContractInputParaOrType, 1, "[]byte", args[1])
	}
	// #nosec G115
	if uint32(len(staker)) < clientChainAddrLength {
		return nil, fmt.Errorf(exocmn.ErrInvalidAddrLength, len(staker), clientChainAddrLength)
	}
	staker = staker[:clientChainAddrLength]

	err = p.delegationKeeper.DissociateOperatorFromStaker(ctx, uint64(clientChainID), staker)
	if err != nil {
		return nil, err
	}
	return method.Outputs.Pack(true)
}
