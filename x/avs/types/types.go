package types

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"sort"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/ethereum/go-ethereum/accounts/abi"

	ibcclienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

// contract code used to generate ChainIDCode
// It is a contract that supports IERC165 interface and does nothing else.
// This way, contracts that follow the standard will not send ERC20/ERC721 transactions to the AVS.
// 0.8.24+commit.e11b9ed9
// 200 optimizer runs

/*
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

interface IERC165 {
    function supportsInterface(bytes4 interfaceId) external view returns (bool);
}

contract SupportsInterfaceExample is IERC165 {
    function supportsInterface(bytes4 interfaceId) public pure override returns (bool) {
        return interfaceId == type(IERC165).interfaceId;
    }
}
*/

var (
	// ChainIDCode is the "fake" code used to mark a generated AVS address as occupied by a contract.
	ChainIDCode = hexutil.MustDecode("0x608060405234801561001057600080fd5b5060c78061001f6000396000f3fe6080604052348015600f57600080fd5b506004361060285760003560e01c806301ffc9a714602d575b600080fd5b604e60383660046062565b6001600160e01b0319166301ffc9a760e01b1490565b604051901515815260200160405180910390f35b600060208284031215607357600080fd5b81356001600160e01b031981168114608a57600080fd5b939250505056fea2646970667358221220b872b230d6a37b4ce12f24d5127759bc0451696f0186fabee8c3e9abe32c462c64736f6c63430008180033")
	// ChainIDCodeHash is the hash of the ChainIDCode.
	ChainIDCodeHash = crypto.Keccak256Hash(ChainIDCode)
	// ChainIDPrefix is pre-pended to the chainID, and the combination hashed to generate the AVS address.
	ChainIDPrefix = []byte("chain-id-prefix")
)

const (
	InvalidTaskID    = 0
	BLSMessageToSign = "BLS12-381 Signed Message\nChainIDWithoutRevision: %s\nAccAddressBech32: %s"
)

type AVSRegisterOrDeregisterParams struct {
	// AvsName is the name of the AVS as an arbitrary string.
	AvsName string
	// AvsAddress is the hex address of the AVS.
	AvsAddress common.Address
	// MinStakeAmount is the minimum amount of stake for a task to be considered valid.
	MinStakeAmount uint64
	// TaskAddress is the hex address of the task contract.
	TaskAddress common.Address
	// SlashContractAddress is the hex address of the slash contract.
	SlashContractAddress common.Address
	// RewardContractAddress is the hex address of the reward contract.
	RewardContractAddress common.Address
	// AvsOwnerAddresses is the list of bech32 addresses of the AVS owners.
	AvsOwnerAddresses []string
	// AssetIDs is the list of asset IDs that the AVS is allowed to use.
	AssetIDs            []string
	Params              []string
	UnbondingPeriod     uint64
	MinSelfDelegation   uint64
	EpochIdentifier     string
	MinOptInOperators   uint64
	MinTotalStakeAmount uint64
	CallerAddress       sdk.AccAddress
	ChainID             string
	AvsReward           uint64
	AvsSlash            uint64
	Action              OperatorAction
	// WhitelistAddresses is the list of bech32 address of the operators.
	WhitelistAddresses []string
}

var (
	taskResponseType, _ = abi.NewType("tuple", "struct", []abi.ArgumentMarshaling{
		{Name: "TaskID", Type: "uint64"},
		{Name: "NumberSum", Type: "uint256"},
	})

	Args = abi.Arguments{
		{Type: taskResponseType, Name: "TaskResponse"},
	}
)

// ChainIDWithoutRevision returns the chainID without the revision number.
// For example, "imuachaintestnet_233-1" returns "imuachaintestnet_233".
func ChainIDWithoutRevision(chainID string) string {
	if !ibcclienttypes.IsRevisionFormat(chainID) {
		return chainID
	}
	splitStr := strings.Split(chainID, "-")
	return splitStr[0]
}

// GenerateAVSAddress generates a hex AVS address based on the chainID.
// It returns a hex address as a string.
func GenerateAVSAddress(chainID string) string {
	avsAddress := common.BytesToAddress(
		crypto.Keccak256(
			append(
				ChainIDPrefix,
				[]byte(chainID)...,
			),
		),
	)
	return strings.ToLower(avsAddress.String())
}

type TaskResponse struct {
	TaskID    uint64
	NumberSum *big.Int
}

// GetTaskResponseDigestEncodeByjson returns the hash of the TaskResponse, which is what operators sign over
// MarshalTaskResponse marshals the TaskResponse struct into JSON bytes.
func MarshalTaskResponse(h TaskResponse) ([]byte, error) {
	return json.Marshal(h)
}

// UnmarshalTaskResponse unmarshals the JSON bytes into a TaskResponse struct.
func UnmarshalTaskResponse(jsonData []byte) (TaskResponse, error) {
	var taskResponse TaskResponse
	err := json.Unmarshal(jsonData, &taskResponse)
	return taskResponse, err
}

// GetTaskResponseDigestEncodeByjson returns the hash of the TaskResponse, which is what operators sign over.
func GetTaskResponseDigestEncodeByjson(h TaskResponse) ([32]byte, error) {
	jsonData, err := MarshalTaskResponse(h)
	if err != nil {
		return [32]byte{}, err
	}
	taskResponseDigest := crypto.Keccak256Hash(jsonData)
	return taskResponseDigest, nil
}

// GetTaskResponseDigestEncodeByAbi returns the hash of the TaskResponse, which is what operators sign over.
func GetTaskResponseDigestEncodeByAbi(h TaskResponse) ([32]byte, error) {
	packed, err := Args.Pack(&h)
	if err != nil {
		return [32]byte{}, err
	}
	fmt.Println("Res:", hex.EncodeToString(packed))
	hashAbi := crypto.Keccak256Hash(packed)
	return hashAbi, nil
}

func Difference(a, b []string) []string {
	var different []string //nolint:prealloc

	diffMap := make(map[string]bool)

	// Add all elements of a to the map
	for _, item := range a {
		diffMap[item] = true
	}

	// Remove elements found in b from the map and collect differences
	for _, item := range b {
		if diffMap[item] {
			delete(diffMap, item)
		} else {
			different = append(different, item)
		}
	}

	// Add remaining elements from the map to different
	for item := range diffMap {
		different = append(different, item)
	}

	// Sort the different slice
	sort.Strings(different)

	return different
}
