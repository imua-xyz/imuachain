package bech32

import (
	"bytes"
	"embed"
	"fmt"

	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	cmn "github.com/evmos/evmos/v16/precompiles/common"
)

const (
	gasPerCall = 6_000
)

var _ vm.PrecompiledContract = &Precompile{}

// Embed abi json file to the executable binary. Needed when importing as dependency.
//
//go:embed abi.json
var f embed.FS

// Precompile defines the precompiled contract for deposit.
type Precompile struct {
	cmn.Precompile
}

// NewPrecompile instantiates a new IBech32 precompile.
func NewPrecompile(authzKeeper authzkeeper.Keeper) (*Precompile, error) {
	abiBz, err := f.ReadFile("abi.json")
	if err != nil {
		return nil, fmt.Errorf("error loading the deposit ABI %s", err)
	}

	newAbi, err := abi.JSON(bytes.NewReader(abiBz))
	if err != nil {
		return nil, fmt.Errorf(cmn.ErrInvalidABI, err)
	}

	p := &Precompile{
		Precompile: cmn.Precompile{
			ABI:                  newAbi,
			AuthzKeeper:          authzKeeper,
			KvGasConfig:          storetypes.KVGasConfig(),
			TransientKVGasConfig: storetypes.TransientGasConfig(),
			// should be configurable in the future.
			ApprovalExpiration: cmn.DefaultExpirationDuration,
			Addr:               common.HexToAddress("0x0000000000000000000000000000000000000400"),
		},
	}

	return p, nil
}

// RequiredGas returns the gas required to execute the bech32 precompile.
func (p Precompile) RequiredGas([]byte) uint64 {
	return gasPerCall
}

// Run performs the bech32 precompile.
func (p Precompile) Run(evm *vm.EVM, contract *vm.Contract, readOnly bool) (bz []byte, err error) {
	ctx, stateDB, snapshot, method, initialGas, args, err := p.RunSetup(evm, contract, readOnly, p.IsTransaction)
	if err != nil {
		return nil, err
	}
	defer cmn.HandleGasError(ctx, contract, initialGas, &err)()

	switch method.Name {
	case MethodHexToBech32:
		return p.HexToBech32(method, args)
	case MethodBech32ToHex:
		return p.Bech32ToHex(method, args)
	}

	cost := ctx.GasMeter().GasConsumed() - initialGas
	if !contract.UseGas(cost) {
		return nil, vm.ErrOutOfGas
	}

	if p.IsTransaction(method.Name) {
		// only add journal entries for non-query methods
		if err := p.AddJournalEntries(stateDB, snapshot); err != nil {
			return nil, err
		}
	}

	return nil, nil
}

// IsTransaction reports whether a precompile is write (true) or read-only (false).
func (Precompile) IsTransaction(_ string) bool {
	// bech32 precompile is read-only and/or stateless
	return false
}
