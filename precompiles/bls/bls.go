package bls

import (
	"bytes"
	"embed"
	"fmt"
	"math/big"

	"github.com/cometbft/cometbft/libs/log"
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
	abi.ABI
}

// NewPrecompile creates a new BLS Precompile instance as a
// PrecompiledContract interface.
func NewPrecompile() (*Precompile, error) {
	abiBz, err := f.ReadFile("abi.json")
	if err != nil {
		return nil, fmt.Errorf("error loading the deposit ABI %s", err)
	}

	newABI, err := abi.JSON(bytes.NewReader(abiBz))
	if err != nil {
		return nil, fmt.Errorf(cmn.ErrInvalidABI, err)
	}

	return &Precompile{
		ABI: newABI,
	}, nil
}

// Address defines the address of the deposit compile contract.
// address: 0x0000000000000000000000000000000000000809
func (p Precompile) Address() common.Address {
	return common.HexToAddress("0x0000000000000000000000000000000000000809")
}

// RequiredGas calculates the precompiled contract's base gas rate.
func (p Precompile) RequiredGas(input []byte) uint64 {
	if len(input) < 4 {
		return 0
	}
	method, err := p.MethodById(input)
	if err != nil {
		return 0
	}
	switch method.Name {
	case MethodVerify:
		return Bls12381PairingBaseGas + Bls12381PairingPerPairGas

	case MethodFastAggregateVerify:

		return p.calculateFastAggregateVerifyGas(input)

	case MethodAggregatePubKeys:
		return p.calculateAggregationGas(input, Bls12381G1AddGas)

	case MethodAggregateSignatures:
		return p.calculateAggregationGas(input, Bls12381G2AddGas)

	case MethodAddTwoPubKeys:
		return Bls12381G1AddGas

	default:
		return 0
	}
}

// FastAggregateVerify gas calculation
func (p Precompile) calculateFastAggregateVerifyGas(input []byte) uint64 {
	if len(input) < 4+96 {
		return 0
	}

	pubKeysOffset := new(big.Int).SetBytes(input[4+64 : 4+96]).Uint64()
	if len(input) < int(4+pubKeysOffset+32) {
		return 0
	}

	m := new(big.Int).SetBytes(input[4+pubKeysOffset : 4+pubKeysOffset+32]).Uint64()
	if m < 1 {
		return 0
	}

	return (m-1)*Bls12381G1AddGas + (Bls12381PairingBaseGas + Bls12381PairingPerPairGas)
}

// Generic aggregation gas calculation
func (p Precompile) calculateAggregationGas(input []byte, gasPerOp uint64) uint64 {
	if len(input) < 4+32 {
		return 0
	}

	offset := new(big.Int).SetBytes(input[4 : 4+32]).Uint64()
	if len(input) < int(4+offset+32) {
		return 0
	}

	n := new(big.Int).SetBytes(input[4+offset : 4+offset+32]).Uint64()
	if n < 1 {
		return 0
	}

	return (n - 1) * gasPerOp
}

// Run executes the precompiled contract deposit methods defined in the ABI.
func (p Precompile) Run(_ *vm.EVM, contract *vm.Contract, _ bool) (bz []byte, err error) {
	methodID := contract.Input[:4]
	// NOTE: this function iterates over the method map and returns
	// the method with the given ID
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
	case MethodVerify:
		bz, err = p.Verify(method, args)
	case MethodFastAggregateVerify:
		bz, err = p.FastAggregateVerify(method, args)
	case MethodAggregatePubKeys:
		bz, err = p.AggregatePubKeys(method, args)
	case MethodAggregateSignatures:
		bz, err = p.AggregateSignatures(method, args)
	case MethodAddTwoPubKeys:
		bz, err = p.AddTwoPubKeys(method, args)
	default:
		return nil, fmt.Errorf("invalid method")
	}

	if err != nil {
		return nil, err
	}

	return bz, nil
}

// IsTransaction checks if the given methodID corresponds to a transaction or query.
//
// Available bls transactions are:
//   - MethodVerify
func (Precompile) IsTransaction() bool {
	return false
}

// Logger returns a precompile-specific logger.
func (p Precompile) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("Imuachain module", "bls")
}
