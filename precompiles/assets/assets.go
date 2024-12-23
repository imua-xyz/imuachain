package assets

import (
	"bytes"
	"embed"
	"fmt"
	"math/big"

	assetskeeper "github.com/ExocoreNetwork/exocore/x/assets/keeper"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	"github.com/ethereum/go-ethereum/accounts/abi"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	cmn "github.com/evmos/evmos/v16/precompiles/common"
)

var _ vm.PrecompiledContract = &Precompile{}

// Embed abi json file to the executable binary. Needed when importing as dependency.
//
//go:embed abi.json
var f embed.FS

// Precompile defines the precompiled contract for deposit.
type Precompile struct {
	cmn.Precompile
	assetsKeeper assetskeeper.Keeper
}

// NewPrecompile creates a new deposit Precompile instance as a
// PrecompiledContract interface.
func NewPrecompile(
	assetsKeeper assetskeeper.Keeper,
	authzKeeper authzkeeper.Keeper,
) (*Precompile, error) {
	abiBz, err := f.ReadFile("abi.json")
	if err != nil {
		return nil, fmt.Errorf("error loading the deposit ABI %s", err)
	}

	newAbi, err := abi.JSON(bytes.NewReader(abiBz))
	if err != nil {
		return nil, fmt.Errorf(cmn.ErrInvalidABI, err)
	}

	return &Precompile{
		Precompile: cmn.Precompile{
			ABI:                  newAbi,
			AuthzKeeper:          authzKeeper,
			KvGasConfig:          storetypes.KVGasConfig(),
			TransientKVGasConfig: storetypes.TransientGasConfig(),
			ApprovalExpiration:   cmn.DefaultExpirationDuration, // should be configurable in the future.
		},
		assetsKeeper: assetsKeeper,
	}, nil
}

// Address defines the address of the deposit compile contract.
// address: 0x0000000000000000000000000000000000000804
func (p Precompile) Address() common.Address {
	return common.HexToAddress("0x0000000000000000000000000000000000000804")
}

// RequiredGas calculates the precompiled contract's base gas rate.
func (p Precompile) RequiredGas(input []byte) uint64 {
	methodID := input[:4]

	method, err := p.MethodById(methodID)
	if err != nil {
		// This should never happen since this method is going to fail during Run
		return 0
	}
	return p.Precompile.RequiredGas(input, p.IsTransaction(method.Name))
}

// Run executes the precompiled contract methods defined in the ABI.
func (p Precompile) Run(evm *vm.EVM, contract *vm.Contract, readOnly bool) (bz []byte, err error) {
	ctx, stateDB, method, initialGas, args, err := p.RunSetup(evm, contract, readOnly, p.IsTransaction)
	if err != nil {
		return nil, err
	}

	// This handles any out of gas errors that may occur during the execution of a precompile tx or query.
	// It avoids panics and returns the out of gas error so the EVM can continue gracefully.
	defer cmn.HandleGasError(ctx, contract, initialGas, &err)()

	if err := stateDB.Commit(); err != nil {
		return nil, err
	}
	cc, writeFunc := ctx.CacheContext()
	switch method.Name {
	// transactions
	case MethodDepositLST, MethodWithdrawLST,
		MethodDepositNST, MethodWithdrawNST:
		bz, err = p.DepositOrWithdraw(cc, evm.Origin, contract, stateDB, method, args)
		if err != nil {
			ctx.Logger().Error("internal error when calling assets precompile", "module", "assets precompile", "method", method.Name, "err", err)
			// for failed cases we expect it returns bool value instead of error
			// this is a workaround because the error returned by precompile can not be caught in EVM
			// see https://github.com/ExocoreNetwork/exocore/issues/70
			// TODO: we should figure out root cause and fix this issue to make precompiles work normally
			bz, err = method.Outputs.Pack(false, new(big.Int))
		} else {
			writeFunc()
		}
	case MethodRegisterOrUpdateClientChain:
		bz, err = p.RegisterOrUpdateClientChain(cc, contract, method, args)
		if err != nil {
			ctx.Logger().Error("internal error when calling assets precompile", "module", "assets precompile", "method", method.Name, "err", err)
			bz, err = method.Outputs.Pack(false, false)
		} else {
			writeFunc()
		}
	case MethodRegisterToken:
		bz, err = p.RegisterToken(cc, contract, method, args)
		if err != nil {
			ctx.Logger().Error("internal error when calling assets precompile", "module", "assets precompile", "method", method.Name, "err", err)
			bz, err = method.Outputs.Pack(false)
		} else {
			writeFunc()
		}
	case MethodUpdateToken:
		bz, err = p.UpdateToken(cc, contract, method, args)
		if err != nil {
			ctx.Logger().Error("internal error when calling assets precompile", "module", "assets precompile", "method", method.Name, "err", err)
			bz, err = method.Outputs.Pack(false)
		} else {
			writeFunc()
		}
	case MethodUpdateAuthorizedGateways:
		bz, err = p.UpdateAuthorizedGateways(ctx, contract, method, args)
		if err != nil {
			ctx.Logger().Error("internal error when calling assets precompile", "module", "assets precompile", "method", method.Name, "err", err)
			bz, err = method.Outputs.Pack(false)
		} else {
			writeFunc()
		}
	// queries
	case MethodGetClientChains:
		bz, err = p.GetClientChains(ctx, method, args)
		if err != nil {
			ctx.Logger().Error("internal error when calling assets precompile", "module", "assets precompile", "method", method.Name, "err", err)
			bz, err = method.Outputs.Pack(false, []uint32{})
		}
	case MethodIsRegisteredClientChain:
		bz, err = p.IsRegisteredClientChain(ctx, method, args)
		if err != nil {
			ctx.Logger().Error("internal error when calling assets precompile", "module", "assets precompile", "method", method.Name, "err", err)
			bz, err = method.Outputs.Pack(false, false)
		}
	case MethodIsAuthorizedGateway:
		bz, err = p.IsAuthorizedGateway(ctx, method, args)
		if err != nil {
			ctx.Logger().Error("internal error when calling assets precompile", "module", "assets precompile", "method", method.Name, "err", err)
			bz, err = method.Outputs.Pack(false, false)
		}
	case MethodGetTokenInfo:
		bz, err = p.GetTokenInfo(ctx, method, args)
		if err != nil {
			ctx.Logger().Error("internal error when calling assets precompile", "module", "assets precompile", "method", method.Name, "err", err)
			bz, err = method.Outputs.Pack(false, NewEmptyTokenInfo())
		}
	case MethodGetStakerBalanceByToken:
		bz, err = p.GetStakerBalanceByToken(ctx, method, args)
		if err != nil {
			ctx.Logger().Error("internal error when calling assets precompile", "module", "assets precompile", "method", method.Name, "err", err)
			bz, err = method.Outputs.Pack(false, NewEmptyStakerBalance())
		}
	default:
		return nil, fmt.Errorf(cmn.ErrUnknownMethod, method.Name)
	}

	if err != nil {
		ctx.Logger().Error("return error when calling assets precompile", "module", "assets precompile", "method", method.Name, "err", err)
		return nil, err
	}

	cost := ctx.GasMeter().GasConsumed() - initialGas

	if !contract.UseGas(cost) {
		return nil, vm.ErrOutOfGas
	}

	return bz, nil
}

// IsTransaction checks if the given methodID corresponds to a transaction or query.
func (Precompile) IsTransaction(methodID string) bool {
	switch methodID {
	case MethodDepositLST, MethodWithdrawLST,
		MethodDepositNST, MethodWithdrawNST,
		MethodRegisterOrUpdateClientChain,
		MethodRegisterToken, MethodUpdateToken, MethodUpdateAuthorizedGateways:
		return true
	case MethodGetClientChains, MethodIsRegisteredClientChain, MethodIsAuthorizedGateway, MethodGetTokenInfo, MethodGetStakerBalanceByToken:
		return false
	default:
		return false
	}
}
