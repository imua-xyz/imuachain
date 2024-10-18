package utils

import (
	"bytes"
	"sort"
	"strings"

	ibcclienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"

	abci "github.com/cometbft/cometbft/abci/types"

	"github.com/evmos/evmos/v16/crypto/ethsecp256k1"
	"golang.org/x/exp/constraints"
	"golang.org/x/xerrors"

	keytypes "github.com/ExocoreNetwork/exocore/types/keys"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/crypto/types/multisig"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	// MainnetChainID defines the Evmos EIP155 chain ID for mainnet
	// TODO: the mainnet chainid is still under consideration and need to be finalized later
	MainnetChainID = "exocore_233"
	// TestnetChainID defines the Evmos EIP155 chain ID for testnet
	// TODO: the testnet chainid is still under consideration and need to be finalized later
	TestnetChainID = "exocoretestnet_233"
	// DefaultChainID is the standard chain id used for testing purposes
	DefaultChainID = MainnetChainID + "-1"
	// BaseDenom defines the Evmos mainnet denomination
	BaseDenom = "hua"

	// DelimiterForCombinedKey is the delimiter used for constructing the combined key.
	DelimiterForCombinedKey = "/"

	// DelimiterForID Delimiter used for constructing the stakerID and assetID.
	DelimiterForID = "_"
)

// IsMainnet returns true if the chain-id has the Evmos mainnet EIP155 chain prefix.
func IsMainnet(chainID string) bool {
	return strings.HasPrefix(chainID, MainnetChainID)
}

// IsTestnet returns true if the chain-id has the Evmos testnet EIP155 chain prefix.
func IsTestnet(chainID string) bool {
	return strings.HasPrefix(chainID, TestnetChainID)
}

func IsValidRevisionChainID(chainID string) bool {
	if strings.Contains(chainID, DelimiterForCombinedKey) {
		return false
	}
	return ibcclienttypes.IsRevisionFormat(chainID)
}

func IsValidChainIDWithoutRevision(chainID string) bool {
	if strings.Contains(chainID, DelimiterForCombinedKey) {
		return false
	}
	return !ibcclienttypes.IsRevisionFormat(chainID)
}

// IsSupportedKey returns true if the pubkey type is supported by the chain
// (i.e eth_secp256k1, amino multisig, ed25519).
// NOTE: Nested multisigs are not supported.
func IsSupportedKey(pubkey cryptotypes.PubKey) bool {
	switch pubkey := pubkey.(type) {
	case *ethsecp256k1.PubKey, *ed25519.PubKey:
		return true
	case multisig.PubKey:
		if len(pubkey.GetPubKeys()) == 0 {
			return false
		}

		for _, pk := range pubkey.GetPubKeys() {
			switch pk.(type) {
			case *ethsecp256k1.PubKey, *ed25519.PubKey:
				continue
			default:
				// Nested multisigs are unsupported
				return false
			}
		}

		return true
	default:
		return false
	}
}

// CommonValidation is used to check for duplicates in the input list
// and validate the input information simultaneously.
// It might be used for validating most genesis states.
// slice is the input list
// seenFieldValue return the key used to check for duplicates and the
// value stored for the other validations
// validation is a function to execute customized check for the object
func CommonValidation[T any, V constraints.Ordered, D any](
	slice []T,
	seenFieldValue func(T) (V, D),
	validation func(int, T) error,
) (map[V]D, error) {
	seen := make(map[V]D)
	for i := range slice {
		v := slice[i]
		field, value := seenFieldValue(v)
		// check for no duplicated element
		if _, ok := seen[field]; ok {
			return nil, xerrors.Errorf(
				"duplicate element: %v",
				field,
			)
		}
		// perform the validation
		if err := validation(i, v); err != nil {
			return nil, err
		}
		seen[field] = value
	}
	return seen, nil
}

// // GetExocoreAddressFromBech32 returns the sdk.Account address of given address,
// // while also changing bech32 human readable prefix (HRP) to the value set on
// // the global sdk.Config (eg: `evmos`).
// // The function fails if the provided bech32 address is invalid.
// func GetExocoreAddressFromBech32(address string) (sdk.AccAddress, error) {
// 	bech32Prefix := strings.SplitN(address, "1", 2)[0]
// 	if bech32Prefix == address {
// 		return nil, errorsmod.Wrapf(errortypes.ErrInvalidAddress, "invalid bech32 address: %s", address)
// 	}

// 	addressBz, err := sdk.GetFromBech32(address, bech32Prefix)
// 	if err != nil {
// 		return nil, errorsmod.Wrapf(errortypes.ErrInvalidAddress, "invalid address %s, %s", address, err.Error())
// 	}

// 	// safety check: shouldn't happen
// 	if err := sdk.VerifyAddressFormat(addressBz); err != nil {
// 		return nil, err
// 	}

// 	return sdk.AccAddress(addressBz), nil
// }

// func DecodeHexString(hexString string) ([]byte, error) {
// 	if strings.HasPrefix(hexString, "0x") || strings.HasPrefix(hexString, "0X") {
// 		hexString = hexString[2:]
// 	}
// 	if len(hexString)%2 != 0 {
// 		hexString = "0" + hexString
// 	}
// 	return hex.DecodeString(hexString)
// }

// // ProcessAddress converts a hex address into the bech32 address format.
// // If the input address is already in bech32 format, it returns the same address.
// func ProcessAddress(address string) (string, error) {
// 	switch {
// 	case common.IsHexAddress(address):
// 		b := common.FromHex(address)
// 		encodedAddress, err := bech32.EncodeFromBase256(config.Bech32Prefix, b)
// 		if err != nil {
// 			return "", err
// 		}
// 		return encodedAddress, nil
// 	case strings.HasPrefix(address, config.Bech32Prefix):
// 		return address, nil
// 	default:
// 		return "", errorsmod.Wrapf(errortypes.ErrInvalidAddress, "invalid input address: %s", address)
// 	}
// }

// SortByPower sorts operators, their pubkeys, and their powers by the powers.
// the sorting is descending, so the highest power is first. If the powers are equal,
// the operator address (bytes, not string!) is used as a tiebreaker. The bytes
// are preferred since that is how the operator module stores them, indexed by
// the bytes. The caller must ensure that the slices are of the same length.
func SortByPower(
	operatorAddrs []sdk.AccAddress,
	pubKeys []keytypes.WrappedConsKey,
	powers []int64,
) ([]sdk.AccAddress, []keytypes.WrappedConsKey, []int64) {
	// Create a slice of indices
	indices := make([]int, len(powers))
	for i := range indices {
		indices[i] = i
	}

	// Sort the indices slice based on the powers slice
	// Since the operator address is unique, SliceStable is not needed
	sort.Slice(indices, func(i, j int) bool {
		if powers[indices[i]] == powers[indices[j]] {
			// ascending order of operator address as tiebreaker
			return bytes.Compare(operatorAddrs[indices[i]], operatorAddrs[indices[j]]) < 0
		}
		// descending order of power
		return powers[indices[i]] > powers[indices[j]]
	})

	// Reorder all slices using the sorted indices
	sortedOperatorAddrs := make([]sdk.AccAddress, len(operatorAddrs))
	sortedPubKeys := make([]keytypes.WrappedConsKey, len(pubKeys))
	sortedPowers := make([]int64, len(powers))
	for i, idx := range indices {
		sortedOperatorAddrs[i] = operatorAddrs[idx]
		sortedPubKeys[i] = pubKeys[idx]
		sortedPowers[i] = powers[idx]
	}
	return sortedOperatorAddrs, sortedPubKeys, sortedPowers
}

// AccumulateChanges accumulates the current and new validator updates and returns
// a list of unique validator updates. The list is sorted by power in descending order.
func AccumulateChanges(
	currentChanges, newChanges []abci.ValidatorUpdate,
) []abci.ValidatorUpdate {
	// get only unique updates
	m := make(map[string]abci.ValidatorUpdate)
	for _, change := range currentChanges {
		m[change.PubKey.String()] = change
	}
	for _, change := range newChanges {
		m[change.PubKey.String()] = change
	}

	// convert to list
	out := make([]abci.ValidatorUpdate, 0, len(m))
	for _, update := range m {
		out = append(out, update)
	}

	// The list of tendermint updates should hash the same across all consensus nodes
	// that means it is necessary to sort for determinism.
	sort.Slice(out, func(i, j int) bool {
		if out[i].Power != out[j].Power {
			return out[i].Power > out[j].Power
		}
		return out[i].PubKey.String() > out[j].PubKey.String()
	})

	return out
}

// AppendMany appends a variable number of byte slices together
func AppendMany(byteses ...[]byte) (out []byte) {
	for _, bytes := range byteses {
		out = append(out, bytes...)
	}
	return out
}

// ChainIDWithoutRevision returns the chainID without the revision number.
// For example, "exocoretestnet_233-1" returns "exocoretestnet_233".
// In the case of app chains, it is not used because upgrading the subscriber
// isn't handled yet, because, during an upgrade, it is not safe to assume
// that the same set of operators will continue with the new chainID.
// The coordinator upgrade also similarly needs to be
// designed and implemented, but that it should be a trivial fix like
// deploying a new IBC client.
func ChainIDWithoutRevision(chainID string) string {
	if !ibcclienttypes.IsRevisionFormat(chainID) {
		return chainID
	}
	splitStr := strings.Split(chainID, "-")
	return splitStr[0]
}

// ChainIDWithLenKey returns the key with the following format:
// bytePrefix | len(chainId) | chainId
// This is similar to Solidity's ABI encoding.
// The caller should typically append a constant length byte array to this and use it as a key.
func ChainIDWithLenKey(chainID string) []byte {
	chainIDL := len(chainID)
	return AppendMany(
		// Append the chainID length
		// #nosec G701
		sdk.Uint64ToBigEndian(uint64(chainIDL)),
		// Append the chainID
		[]byte(chainID),
	)
}

// PanicIfNil panics if the input is nil with the given message.
func PanicIfNil(x interface{}, msg string) {
	if x == nil {
		panic("zero or nil value for " + msg)
	}
}
