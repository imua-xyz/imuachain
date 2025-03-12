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

	return &Precompile{
		Precompile: cmn.Precompile{
			ABI:                  newAbi,
			AuthzKeeper:          authzKeeper,
			KvGasConfig:          storetypes.KVGasConfig(),
			TransientKVGasConfig: storetypes.TransientGasConfig(),
			// should be configurable in the future.
			ApprovalExpiration: cmn.DefaultExpirationDuration,
			Addr:               common.HexToAddress("0x0000000000000000000000000000000000000400"),
		},
	}, nil
}

// RequiredGas returns the gas required to execute the bech32 precompile.
func (p Precompile) RequiredGas([]byte) uint64 {
	return gasPerCall
}

// Run performs the bech32 precompile.
func (p Precompile) Run(_ *vm.EVM, contract *vm.Contract, _ bool) (bz []byte, err error) {
	// do not call RunSetup because this precompile is stateless
	if len(contract.Input) < 4 {
		return nil, vm.ErrExecutionReverted
	}

	methodID := contract.Input[:4]
	method, err := p.MethodById(methodID)
	if err != nil {
		return nil, err
	}

	argsBz := contract.Input[4:]
	args, err := method.Inputs.Unpack(argsBz)
	if err != nil {
		return nil, err
	}

	switch method.Name {
	case MethodHexToBech32:
		bz, err = p.HexToBech32(method, args)
	case MethodBech32ToHex:
		bz, err = p.Bech32ToHex(method, args)
	}

	if err != nil {
		return nil, err
	}

	return bz, nil
}

// IsTransaction reports whether a precompile is write (true) or read-only (false).
func (Precompile) IsTransaction(_ string) bool {
	// bech32 precompile is read-only and/or stateless
	return false
}
