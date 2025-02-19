package bech32

import (
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	cmn "github.com/evmos/evmos/v16/precompiles/common"
	imuacmn "github.com/imua-xyz/imuachain/precompiles/common"
)

const (
	// bech32 separator
	separator = "1"
)

// method IDs (without the type)
const (
	MethodHexToBech32 = "hexToBech32"
	MethodBech32ToHex = "bech32ToHex"
)

// HexToBech32 converts a hex address to a bech32 address with the provided human readable prefix.
func (p Precompile) HexToBech32(
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	if len(args) != len(method.Inputs) {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, len(method.Inputs), len(args))
	}

	address, ok := args[0].(common.Address)
	if !ok {
		return nil, fmt.Errorf("invalid hex address")
	}

	// validate that the prefix is non-empty
	prefix, _ := args[1].(string)
	if strings.TrimSpace(prefix) == "" {
		return nil, fmt.Errorf(
			"empty bech32 prefix provided, expected a non-empty string",
		)
	}

	// superfluous check; should never fail given len(common.Address) == 20
	if err := sdk.VerifyAddressFormat(address.Bytes()); err != nil {
		return nil, err
	}

	bech32Str, err := sdk.Bech32ifyAddressBytes(prefix, address.Bytes())
	if err != nil {
		return nil, err
	}

	return method.Outputs.Pack(bech32Str)
}

// Bech32ToHex converts a bech32 address to a hex address. The prefix is automatically detected.
func (p Precompile) Bech32ToHex(
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	// validate the number of arguments
	if len(args) != len(method.Inputs) {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, len(method.Inputs), len(args))
	}

	address, ok := args[0].(string)
	if !ok || address == "" {
		return nil, fmt.Errorf("invalid bech32 address")
	}

	// find the prefix
	prefix := strings.SplitN(address, separator, 2)[0]
	if prefix == address {
		return nil, fmt.Errorf("invalid bech32 address (no separator): %s", address)
	}

	// get the address bytes
	addressBz, err := sdk.GetFromBech32(address, prefix)
	if err != nil {
		return nil, err
	}

	// validate the address bytes: call the verifier if set and check length is range-bound
	if err := sdk.VerifyAddressFormat(addressBz); err != nil {
		return nil, err
	}

	// check the bytes length, since BytesToAddress silently crops
	if len(addressBz) != common.AddressLength {
		return nil, fmt.Errorf(imuacmn.ErrInvalidAddrLength, len(addressBz), common.AddressLength)
	}

	// pack the address bytes
	return method.Outputs.Pack(common.BytesToAddress(addressBz))
}
