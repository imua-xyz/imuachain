package assets

import (
	"bytes"
	"embed"
	"fmt"
	"math/big"

	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	"github.com/ethereum/go-ethereum/accounts/abi"
	assetskeeper "github.com/imua-xyz/imuachain/x/assets/keeper"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	cmn "github.com/evmos/evmos/v16/precompiles/common"
	imuacmn "github.com/imua-xyz/imuachain/precompiles/common"
)

var _ vm.PrecompiledContract = &Precompile{}

// Embed abi json file to the executable binary. Needed when importing as dependency.
//
//go:embed abi.json
var f embed.FS

// Precompile defines the precompiled contract for assets.
type Precompile struct {
	cmn.Precompile
	assetsKeeper assetskeeper.Keeper
}

// NewPrecompile creates a new assets Precompile instance as a
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
			Addr:                 common.HexToAddress("0x0000000000000000000000000000000000000804"),
		},
		assetsKeeper: assetsKeeper,
	}, nil
}

// RequiredGas calculates the precompiled contract's base gas rate.
func (p Precompile) RequiredGas(input []byte) uint64 {
	method, err := p.MethodById(input)
	if err != nil {
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

	// We have not determined the appropriate way to return an error from the precompile, that can be
	// caught in Solidity. Hence, we return a bool value instead. However, since a precompile performs
	// many stateful operations, we must wrap the overall execution in a cached context to guarantee
	// atomicity.
	cc := ctx
	writeFunc := func() {}
	if p.IsTransaction(method.Name) {
		cc, writeFunc = ctx.CacheContext()
	}

	var logError error

	switch method.Name {
	// transactions
	case MethodDepositLST, MethodWithdrawLST,
		MethodDepositNST, MethodWithdrawNST:
		bz, err = p.DepositOrWithdraw(cc, evm.Origin, contract, stateDB, method, args)
		if err != nil {
			logError = err
			bz, err = method.Outputs.Pack(false, new(big.Int))
		}
	case MethodRegisterOrUpdateClientChain:
		bz, err = p.RegisterOrUpdateClientChain(cc, contract, method, args)
		if err != nil {
			logError = err
			bz, err = method.Outputs.Pack(false, false)
		}
	case MethodRegisterToken:
		bz, err = p.RegisterToken(cc, contract, method, args)
		if err != nil {
			logError = err
			bz, err = method.Outputs.Pack(false)
		}
	case MethodUpdateToken:
		bz, err = p.UpdateToken(cc, contract, method, args)
		if err != nil {
			logError = err
			bz, err = method.Outputs.Pack(false)
		}
	case MethodUpdateAuthorizedGateways:
		bz, err = p.UpdateAuthorizedGateways(cc, contract, method, args)
		if err != nil {
			logError = err
			bz, err = method.Outputs.Pack(false)
		}
	// queries
	case MethodGetClientChains:
		bz, err = p.GetClientChains(cc, method, args)
		if err != nil {
			logError = err
			bz, err = method.Outputs.Pack(false, []uint32{})
		}
	case MethodIsRegisteredClientChain:
		bz, err = p.IsRegisteredClientChain(cc, method, args)
		if err != nil {
			logError = err
			bz, err = method.Outputs.Pack(false, false)
		}
	case MethodIsAuthorizedGateway:
		bz, err = p.IsAuthorizedGateway(cc, method, args)
		if err != nil {
			logError = err
			bz, err = method.Outputs.Pack(false, false)
		}
	case MethodGetTokenInfo:
		bz, err = p.GetTokenInfo(cc, method, args)
		if err != nil {
			logError = err
			bz, err = method.Outputs.Pack(false, NewEmptyTokenInfo())
		}
	case MethodGetStakerBalanceByToken:
		bz, err = p.GetStakerBalanceByToken(cc, method, args)
		if err != nil {
			logError = err
			bz, err = method.Outputs.Pack(false, NewEmptyStakerBalance())
		}
	default:
		// this will cause a vm.ErrExecutionReverted and kill the tx.
		// it is acceptable for now, given the same happens if you call
		// a contract with a non-existent method.
		// this will never happen because RunSetup will error out first.
		return nil, fmt.Errorf(cmn.ErrUnknownMethod, method.Name)
	}

	if logError != nil {
		// write on ctx and not cc to ensure it is written on the
		// original (uncached) context.
		ctx.Logger().Error(
			"return error when calling assets precompile",
			"module", "assets precompile",
			"method", method.Name,
			"err", logError,
		)
	} else {
		writeFunc()
	}

	cost := ctx.GasMeter().GasConsumed() - initialGas
	if !contract.UseGas(cost) {
		return nil, vm.ErrOutOfGas
	}

	return bz, nil
}

// IsTransaction checks if the given methodName corresponds to a transaction or query.
func (Precompile) IsTransaction(methodName string) bool {
	switch methodName {
	case MethodDepositLST, MethodWithdrawLST,
		MethodDepositNST, MethodWithdrawNST,
		MethodRegisterOrUpdateClientChain,
		MethodRegisterToken, MethodUpdateToken, MethodUpdateAuthorizedGateways:
		return true
	case MethodGetClientChains, MethodIsRegisteredClientChain,
		MethodIsAuthorizedGateway, MethodGetTokenInfo,
		MethodGetStakerBalanceByToken:
		return false
	default:
		// this panic is safe to perform because the `init` function
		// below forces developers to add all methods to the switch statement.
		panic(fmt.Sprintf("unknown method: %s", methodName))
	}
}

func init() {
	// dummy instance
	var p Precompile
	if err := imuacmn.ValidateIsTx(f, p.IsTransaction); err != nil {
		panic(err)
	}
}
