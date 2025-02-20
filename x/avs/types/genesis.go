package types

import (
	"fmt"
	"strconv"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	assetstype "github.com/imua-xyz/imuachain/x/assets/types"
)

// this line is used by starport scaffolding # genesis/types/import

// DefaultIndex is the default global index
const DefaultIndex uint64 = 1

func NewGenesisState(
	avsInfos []AVSInfo,
	taskInfos []TaskInfo,
	blsPubKeys []BlsPubKeyInfo,
	taskResultInfos []TaskResultInfo,
	challengeInfos []ChallengeInfo,
	taskNums []TaskID,
	chainIDInfos []ChainIDInfo,
) *GenesisState {
	// Ensure slices are never nil
	if avsInfos == nil {
		avsInfos = []AVSInfo{}
	}
	if taskInfos == nil {
		taskInfos = []TaskInfo{}
	}
	if blsPubKeys == nil {
		blsPubKeys = []BlsPubKeyInfo{}
	}
	if taskResultInfos == nil {
		taskResultInfos = []TaskResultInfo{}
	}
	if challengeInfos == nil {
		challengeInfos = []ChallengeInfo{}
	}
	if taskNums == nil {
		taskNums = []TaskID{}
	}
	if chainIDInfos == nil {
		chainIDInfos = []ChainIDInfo{}
	}
	return &GenesisState{
		AvsInfos:        avsInfos,
		TaskInfos:       taskInfos,
		BlsPubKeys:      blsPubKeys,
		TaskResultInfos: taskResultInfos,
		ChallengeInfos:  challengeInfos,
		TaskNums:        taskNums,
		ChainIdInfos:    chainIDInfos,
	}
}

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return NewGenesisState(nil, nil, nil, nil, nil, nil, nil)
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
// Explanation: The existence of the operator was not checked as it depends on the operator module that needs to be loaded first
func (gs GenesisState) Validate() error {
	// Check for duplicated avs address
	avsAddresses := make(map[string]bool)
	for _, info := range gs.AvsInfos {
		if !common.IsHexAddress(info.AvsAddress) {
			return fmt.Errorf("invalid AVS address: %s", info.AvsAddress)
		}
		if avsAddresses[info.AvsAddress] {
			return fmt.Errorf("duplicate AVS address: %s", info.AvsAddress)
		}
		avsAddresses[info.AvsAddress] = true
	}

	// Check for duplicated task address
	taskInfoMap := make(map[string]bool)
	for _, info := range gs.TaskInfos {
		if !common.IsHexAddress(info.TaskContractAddress) {
			return fmt.Errorf("invalid hex address: %s", info.TaskContractAddress)
		}
		infoKey := assetstype.GetJoinedStoreKey(strings.ToLower(info.TaskContractAddress), strconv.FormatUint(info.TaskId, 10))

		if taskInfoMap[string(infoKey)] {
			return fmt.Errorf("duplicate task address: %s", info.TaskContractAddress)
		}
		taskInfoMap[string(infoKey)] = true
	}

	// Check for duplicated taskID
	taskNumMap := make(map[string]bool)
	for _, taskNum := range gs.TaskNums {
		if !common.IsHexAddress(taskNum.TaskAddress) {
			return fmt.Errorf("invalid hex address: %s", taskNum.TaskAddress)
		}
		taskNumKey := assetstype.GetJoinedStoreKey(strings.ToLower(taskNum.TaskAddress),
			strconv.FormatUint(taskNum.TaskId, 10))
		if taskNumMap[string(taskNumKey)] {
			return fmt.Errorf("duplicate task ID %v ", taskNum)
		}
		taskNumMap[string(taskNumKey)] = true
	}

	// Check for duplicated pubKey
	pubKeyMap := make(map[string]bool)
	for _, key := range gs.BlsPubKeys {
		_, err := sdk.AccAddressFromBech32(key.OperatorAddress)
		if err != nil {
			return fmt.Errorf("invalid operatorAddress address: %s", key.OperatorAddress)
		}
		if key.PubKey == nil {
			return fmt.Errorf("pubKey is nil: %v", key)
		}
		if pubKeyMap[string(key.PubKey)] {
			return fmt.Errorf("duplicate pubKey %v", key)
		}
		pubKeyMap[string(key.PubKey)] = true
	}

	// Check for duplicated task result
	taskResultMap := make(map[string]bool)
	for _, result := range gs.TaskResultInfos {

		if !common.IsHexAddress(result.TaskContractAddress) {
			return fmt.Errorf("invalid hex address: %s", result.TaskContractAddress)
		}
		_, err := sdk.AccAddressFromBech32(result.OperatorAddress)
		if err != nil {
			return fmt.Errorf("invalid operatorAddress address: %s", result.OperatorAddress)
		}
		resultKey := assetstype.GetJoinedStoreKey(result.OperatorAddress, strings.ToLower(result.TaskContractAddress),
			strconv.FormatUint(result.TaskId, 10))

		if taskResultMap[string(resultKey)] {
			return fmt.Errorf("duplicate Task Result %v", result)
		}
		taskResultMap[string(resultKey)] = true
	}

	// Check for duplicated challenge
	challengeMap := make(map[string]bool)
	for _, result := range gs.ChallengeInfos {
		if challengeMap[result.Key] {
			return fmt.Errorf("duplicate challenge %v", result)
		}
		challengeMap[result.Key] = true
	}

	// Check for duplicated chainId Info
	chainIDMap := make(map[string]bool)
	for _, result := range gs.ChainIdInfos {
		if !common.IsHexAddress(result.AvsAddress) {
			return fmt.Errorf("invalid hex address: %s", result.AvsAddress)
		}
		if chainIDMap[result.AvsAddress] {
			return fmt.Errorf("duplicate chainId %v", result)
		}
		chainIDMap[result.AvsAddress] = true
	}
	return nil
}
