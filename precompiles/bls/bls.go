package bls

import (
	"bytes"
	"embed"
	"fmt"

	"github.com/cometbft/cometbft/libs/log"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
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
	baseGas uint64
}

// NewPrecompile creates a new BLS Precompile instance as a
// PrecompiledContract interface.
func NewPrecompile(baseGas uint64) (*Precompile, error) {
	// short-circuit if baseGas is zero
	if baseGas == 0 {
		return nil, fmt.Errorf("baseGas cannot be zero")
	}

	abiBz, err := f.ReadFile("abi.json")
	if err != nil {
		return nil, fmt.Errorf("error loading the deposit ABI %s", err)
	}

	newABI, err := abi.JSON(bytes.NewReader(abiBz))
	if err != nil {
		return nil, fmt.Errorf(cmn.ErrInvalidABI, err)
	}

	p := &Precompile{
		Precompile: cmn.Precompile{
			ABI:                  newABI,
			KvGasConfig:          storetypes.KVGasConfig(),
			TransientKVGasConfig: storetypes.TransientGasConfig(),
			ApprovalExpiration:   cmn.DefaultExpirationDuration,
		},
		baseGas: baseGas,
	}

	p.SetAddress(common.HexToAddress("0x0000000000000000000000000000000000000809"))

	return p, nil
}

// RequiredGas calculates the precompiled contract's base gas rate.
func (p Precompile) RequiredGas(_ []byte) uint64 {
	// TODO: @devin: add gas calculation depending on the method
	return p.baseGas
}

// Run executes the precompiled contract deposit methods defined in the ABI.
func (p Precompile) Run(evm *vm.EVM, contract *vm.Contract, readOnly bool) (bz []byte, err error) {
	ctx, stateDB, snapshot, method, initialGas, args, err := p.RunSetup(evm, contract, readOnly, p.IsTransaction)
	if err != nil {
		return nil, err
	}

	// This handles any out of gas errors that may occur during the execution of a precompile tx or query.
	// It avoids panics and returns the out of gas error so the EVM can continue gracefully.
	defer cmn.HandleGasError(ctx, contract, initialGas, &err)()

	switch method.Name {
	case MethodVerify:
		bz, err = p.Verify(method, args)
	case MethodFastAggregateVerify:
		bz, err = p.FastAggregateVerify(method, args)
	case MethodAggregatePubkeys:
		bz, err = p.AggregatePubkeys(method, args)
	case MethodAggregateSignatures:
		bz, err = p.AggregateSignatures(method, args)
	case MethodAddTwoPubkeys:
		bz, err = p.AddTwoPubkeys(method, args)
	default:
		return nil, fmt.Errorf("invalid method")
	}

	if err != nil {
		return nil, err
	}

	if p.IsTransaction(method.Name) {
		// only add journal entries for non-query methods
		if err := p.AddJournalEntries(stateDB, snapshot); err != nil {
			return nil, err
		}
	}

	return bz, nil
}

// IsTransaction checks if the given methodID corresponds to a transaction or query.
//
// Available bls transactions are:
//   - MethodVerify
func (Precompile) IsTransaction(methodID string) bool {
	switch methodID {
	case MethodVerify,
		MethodFastAggregateVerify,
		MethodAggregatePubkeys,
		MethodAggregateSignatures,
		MethodAddTwoPubkeys:
		// TODO: @devin: shouldn't this be false?
		return true
	default:
		return false
	}
}

// Logger returns a precompile-specific logger.
func (p Precompile) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("Imuachain module", "bls")
}
