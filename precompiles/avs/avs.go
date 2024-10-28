package avs

import (
	"bytes"
	"embed"
	"fmt"

	avsKeeper "github.com/ExocoreNetwork/exocore/x/avs/keeper"
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

// Precompile defines the precompiled contract for avs.
type Precompile struct {
	cmn.Precompile
	avsKeeper avsKeeper.Keeper
}

// NewPrecompile creates a new avs Precompile instance as a
// PrecompiledContract interface.
func NewPrecompile(
	avsKeeper avsKeeper.Keeper,
	authzKeeper authzkeeper.Keeper,
) (*Precompile, error) {
	abiBz, err := f.ReadFile("abi.json")
	if err != nil {
		return nil, fmt.Errorf("error loading the avs ABI %s", err)
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
			ApprovalExpiration:   cmn.DefaultExpirationDuration,
		},
		avsKeeper: avsKeeper,
	}, nil
}

// Address defines the address of the avs compile contract.
// address: 0x0000000000000000000000000000000000000901
func (p Precompile) Address() common.Address {
	return common.HexToAddress("0x0000000000000000000000000000000000000901")
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

// Run executes the precompiled contract RegisterOrDeregisterAVSInfo methods defined in the ABI.
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

	switch method.Name {
	case MethodRegisterAVS:
		bz, err = p.RegisterAVS(ctx, evm.Origin, contract, stateDB, method, args)
		if err != nil {
			ctx.Logger().Error("internal error when calling avs precompile", "module", "avs precompile", "method", method.Name, "err", err)
			bz, err = method.Outputs.Pack(false)
		}
	case MethodDeregisterAVS:
		bz, err = p.DeregisterAVS(ctx, evm.Origin, contract, stateDB, method, args)
		if err != nil {
			ctx.Logger().Error("internal error when calling avs precompile", "module", "avs precompile", "method", method.Name, "err", err)
			bz, err = method.Outputs.Pack(false)
		}
	case MethodUpdateAVS:
		bz, err = p.UpdateAVS(ctx, evm.Origin, contract, stateDB, method, args)
		if err != nil {
			ctx.Logger().Error("internal error when calling avs precompile", "module", "avs precompile", "method", method.Name, "err", err)
			bz, err = method.Outputs.Pack(false)
		}
	case MethodRegisterOperatorToAVS:
		bz, err = p.BindOperatorToAVS(ctx, evm.Origin, contract, stateDB, method, args)
		if err != nil {
			ctx.Logger().Error("internal error when calling avs precompile", "module", "avs precompile", "method", method.Name, "err", err)
			bz, err = method.Outputs.Pack(false)
		}
	case MethodDeregisterOperatorFromAVS:
		bz, err = p.UnbindOperatorToAVS(ctx, evm.Origin, contract, stateDB, method, args)
		if err != nil {
			ctx.Logger().Error("internal error when calling avs precompile", "module", "avs precompile", "method", method.Name, "err", err)
			bz, err = method.Outputs.Pack(false)
		}
	case MethodCreateAVSTask:
		bz, err = p.CreateAVSTask(ctx, evm.Origin, contract, stateDB, method, args)
		if err != nil {
			ctx.Logger().Error("internal error when calling avs precompile", "module", "avs precompile", "method", method.Name, "err", err)
			bz, err = method.Outputs.Pack(false)
		}
	case MethodRegisterBLSPublicKey:
		bz, err = p.RegisterBLSPublicKey(ctx, evm.Origin, contract, stateDB, method, args)
		if err != nil {
			ctx.Logger().Error("internal error when calling avs precompile", "module", "avs precompile", "method", method.Name, "err", err)
			bz, err = method.Outputs.Pack(false)
		}
	case MethodGetOptinOperators:
		bz, err = p.GetOptedInOperatorAccAddrs(ctx, contract, method, args)
	case MethodGetRegisteredPubkey:
		bz, err = p.GetRegisteredPubkey(ctx, contract, method, args)
	case MethodGetAVSUSDValue:
		bz, err = p.GetAVSUSDValue(ctx, contract, method, args)
		if err != nil {
			ctx.Logger().Error("internal error when calling avs precompile", "module", "avs precompile", "method", method.Name, "err", err)
			bz, err = method.Outputs.Pack(common.Big0)
		}
	case MethodGetOperatorOptedUSDValue:
		bz, err = p.GetOperatorOptedUSDValue(ctx, contract, method, args)
		if err != nil {
			ctx.Logger().Error("internal error when calling avs precompile", "module", "avs precompile", "method", method.Name, "err", err)
			bz, err = method.Outputs.Pack(common.Big0)
		}
	case MethodChallenge:
		bz, err = p.Challenge(ctx, evm.Origin, contract, stateDB, method, args)
		if err != nil {
			ctx.Logger().Error("internal error when calling avs precompile", "module", "avs precompile", "method", method.Name, "err", err)
			bz, err = method.Outputs.Pack(false)
		}
	}

	if err != nil {
		ctx.Logger().Error("call avs precompile error", "module", "avs precompile", "err", err)
		return nil, err
	}

	cost := ctx.GasMeter().GasConsumed() - initialGas

	if !contract.UseGas(cost) {
		return nil, vm.ErrOutOfGas
	}
	return bz, nil
}

// IsTransaction checks if the given methodID corresponds to a transaction or query.
//
// Available avs transactions are:
//   - AVSRegister
func (Precompile) IsTransaction(methodID string) bool {
	switch methodID {
	case MethodRegisterAVS, MethodDeregisterAVS, MethodUpdateAVS, MethodRegisterOperatorToAVS,
		MethodDeregisterOperatorFromAVS, MethodCreateAVSTask, MethodRegisterBLSPublicKey, MethodChallenge:
		return true
	case MethodGetRegisteredPubkey, MethodGetOptinOperators, MethodGetAVSUSDValue, MethodGetOperatorOptedUSDValue:
		return false
	default:
		return false
	}
}
