package avs

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/core/vm"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	cmn "github.com/evmos/evmos/v16/precompiles/common"
	imuacmn "github.com/imua-xyz/imuachain/precompiles/common"
	avstypes "github.com/imua-xyz/imuachain/x/avs/types"
)

func (p Precompile) GetAVSParamsFromInputs(contract *vm.Contract, origin common.Address, method *abi.Method, args []interface{}) (*avstypes.AVSRegisterOrDeregisterParams, error) {
	if len(args) != len(p.ABI.Methods[MethodRegisterAVS].Inputs) {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, len(p.ABI.Methods[MethodRegisterAVS].Inputs), len(args))
	}
	var avsPayload Payload
	if err := method.Inputs.Copy(&avsPayload, args); err != nil {
		return nil, fmt.Errorf("error while unpacking args to Payload struct: %s", err)
	}
	avsParams := &avstypes.AVSRegisterOrDeregisterParams{}
	//	we'd better not use evm.Origin but let the precompile caller pass in the sender address,
	//	since tx.origin has some security issue and might not be supported
	//	in a long term: https://docs.soliditylang.org/en/latest/security-considerations.html#tx-origin
	if !common.IsHexAddress(avsPayload.AVSParams.Sender.String()) {
		return nil, fmt.Errorf("the contract input parameter sender error,value:%v", avsPayload.AVSParams.Sender)
	}
	// The provided sender address should always be equal to the origin address.
	// In case the contract caller address is the same as the sender address provided,
	// update the sender address to be equal to the origin address.
	// Otherwise, if the provided sender address is different from the origin address,
	// return an error because is a forbidden operation
	_, err := CheckOriginAndSender(contract, origin, avsPayload.AVSParams.Sender)
	if err != nil {
		return nil, err
	}

	avsParams.CallerAddress = avsPayload.AVSParams.Sender[:]
	avsParams.AvsName = avsPayload.AVSParams.AvsName
	// When creating tasks in AVS, check the minimum requirements,minStakeAmount at least greater than 0
	if avsPayload.AVSParams.MinStakeAmount == 0 {
		return nil, fmt.Errorf("the contract input parameter MinStakeAmount error,value:%v", avsPayload.AVSParams.MinStakeAmount)
	}
	avsParams.MinStakeAmount = avsPayload.AVSParams.MinStakeAmount

	if !common.IsHexAddress(avsPayload.AVSParams.TaskAddress.String()) {
		return nil, fmt.Errorf("the contract input parameter TaskAddr error,value:%v", avsPayload.AVSParams.TaskAddress)
	}
	avsParams.TaskAddress = avsPayload.AVSParams.TaskAddress

	if !common.IsHexAddress(avsPayload.AVSParams.SlashAddress.String()) {
		return nil, fmt.Errorf("the contract input parameter SlashAddress error,value:%v", avsPayload.AVSParams.SlashAddress)
	}
	avsParams.SlashContractAddress = avsPayload.AVSParams.SlashAddress

	if !common.IsHexAddress(avsPayload.AVSParams.RewardAddress.String()) {
		return nil, fmt.Errorf("the contract input parameter RewardAddress error,value:%v", avsPayload.AVSParams.RewardAddress)
	}
	avsParams.RewardContractAddress = avsPayload.AVSParams.RewardAddress

	if avsPayload.AVSParams.AvsOwnerAddresses == nil {
		return nil, fmt.Errorf("the contract input parameter AvsOwnerAddresses error,value:%v", avsPayload.AVSParams.AvsOwnerAddresses)
	}
	exoAddresses := make([]string, len(avsPayload.AVSParams.AvsOwnerAddresses))
	for i, address := range avsPayload.AVSParams.AvsOwnerAddresses {
		var accAddress sdk.AccAddress = address[:]
		exoAddresses[i] = accAddress.String()
	}
	avsParams.AvsOwnerAddresses = exoAddresses

	if avsPayload.AVSParams.WhitelistAddresses == nil {
		return nil, fmt.Errorf("the contract input parameter WhitelistAddresses error,value:%v", avsPayload.AVSParams.WhitelistAddresses)
	}
	exoWhiteAddresses := make([]string, len(avsPayload.AVSParams.WhitelistAddresses))
	for i, address := range avsPayload.AVSParams.WhitelistAddresses {
		var accAddress sdk.AccAddress = address[:]
		exoWhiteAddresses[i] = accAddress.String()
	}
	avsParams.WhitelistAddresses = exoWhiteAddresses
	// string, since it is the address_id representation
	if avsPayload.AVSParams.AssetIDs == nil {
		return nil, fmt.Errorf("the contract input parameter AssetIds error,value:%v", avsPayload.AVSParams.AssetIDs)
	}
	avsParams.AssetIDs = avsPayload.AVSParams.AssetIDs

	avsParams.UnbondingPeriod = avsPayload.AVSParams.AvsUnbondingPeriod

	avsParams.MinSelfDelegation = avsPayload.AVSParams.MinSelfDelegation

	if avsPayload.AVSParams.EpochIdentifier == "" {
		return nil, fmt.Errorf("the contract input parameter EpochIdentifier error,value:%v", avsPayload.AVSParams.EpochIdentifier)
	}
	avsParams.EpochIdentifier = avsPayload.AVSParams.EpochIdentifier

	avsParams.MinOptInOperators = avsPayload.AVSParams.MiniOptInOperators

	avsParams.MinTotalStakeAmount = avsPayload.AVSParams.MinTotalStakeAmount

	avsParams.AvsReward = avsPayload.AVSParams.AvsRewardProportion

	avsParams.AvsSlash = avsPayload.AVSParams.AvsSlashProportion

	return avsParams, nil
}

func (p Precompile) GetTaskParamsFromInputs(_ sdk.Context, args []interface{}) (*avstypes.TaskInfoParams, error) {
	if len(args) != len(p.ABI.Methods[MethodCreateAVSTask].Inputs) {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, len(p.ABI.Methods[MethodCreateAVSTask].Inputs), len(args))
	}
	taskParams := &avstypes.TaskInfoParams{}
	callerAddress, ok := args[0].(common.Address)
	if !ok || (callerAddress == common.Address{}) {
		return nil, fmt.Errorf(imuacmn.ErrContractInputParamOrType, 0, "common.Address", callerAddress)
	}
	taskParams.CallerAddress = callerAddress[:]
	name, ok := args[1].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf(imuacmn.ErrContractInputParamOrType, 1, "string", name)
	}
	taskParams.TaskName = name

	hash, ok := args[2].([]byte)
	if !ok {
		return nil, fmt.Errorf(imuacmn.ErrContractInputParamOrType, 2, "[]byte", hash)
	}
	taskParams.Hash = hash

	taskResponsePeriod, ok := args[3].(uint64)
	if !ok {
		return nil, fmt.Errorf(imuacmn.ErrContractInputParamOrType, 3, "uint64", taskResponsePeriod)
	}
	taskParams.TaskResponsePeriod = taskResponsePeriod

	taskChallengePeriod, ok := args[4].(uint64)
	if !ok {
		return nil, fmt.Errorf(imuacmn.ErrContractInputParamOrType, 4, "uint64", taskChallengePeriod)
	}
	taskParams.TaskChallengePeriod = taskChallengePeriod

	thresholdPercentage, ok := args[5].(uint8)
	if !ok || thresholdPercentage > 100 {
		return nil, fmt.Errorf(imuacmn.ErrContractInputParamOrType, 5, "uint64", thresholdPercentage)
	}
	taskParams.ThresholdPercentage = thresholdPercentage

	taskStatisticalPeriod, ok := args[6].(uint64)
	if !ok {
		return nil, fmt.Errorf(imuacmn.ErrContractInputParamOrType, 6, "uint64", taskStatisticalPeriod)
	}
	taskParams.TaskStatisticalPeriod = taskStatisticalPeriod
	return taskParams, nil
}

// Params is a utility structure used to wrap args received by the
// Solidity interface of the avs function.
type Params struct {
	Sender              common.Address   `abi:"sender"`              // the sender of the  transaction
	AvsName             string           `abi:"avsName"`             // the name of AVS
	MinStakeAmount      uint64           `abi:"minStakeAmount"`      // the minimum amount of funds staked by each operator
	TaskAddress         common.Address   `abi:"taskAddress"`         // the task address of AVS
	SlashAddress        common.Address   `abi:"slashAddress"`        // the slash address of AVS
	RewardAddress       common.Address   `abi:"rewardAddress"`       // the reward address of AVS
	AvsOwnerAddresses   []common.Address `abi:"avsOwnerAddresses"`   // the owners who have permission for AVS
	WhitelistAddresses  []common.Address `abi:"whitelistAddresses"`  // the whitelist address of the operator
	AssetIDs            []string         `abi:"assetIDs"`            // the basic asset information of AVS
	AvsUnbondingPeriod  uint64           `abi:"avsUnbondingPeriod"`  // the unbonding duration of AVS
	MinSelfDelegation   uint64           `abi:"minSelfDelegation"`   // the minimum delegation amount for an operator
	EpochIdentifier     string           `abi:"epochIdentifier"`     // the AVS epoch identifier
	MiniOptInOperators  uint64           `abi:"miniOptInOperators"`  // the minimum number of opt-in operators
	MinTotalStakeAmount uint64           `abi:"minTotalStakeAmount"` // the minimum total amount of stake by all operators
	AvsRewardProportion uint64           `abi:"avsRewardProportion"` // the proportion of reward for AVS
	AvsSlashProportion  uint64           `abi:"avsSlashProportion"`  // the proportion of slash for AVS
}

// Payload is the same as the expected input of the avs function in the Solidity interface.
type Payload struct {
	AVSParams Params
}

// ParseAVStData parses the packet data from the outpost precompiled contract.
func ParseAVStData(method *abi.Method, args []interface{}) (Params, error) {
	if len(args) != 1 {
		return Params{}, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 1, len(args))
	}

	var avsPayload Payload
	if err := method.Inputs.Copy(&avsPayload, args); err != nil {
		return Params{}, fmt.Errorf("error while unpacking args to Payload struct: %s", err)
	}
	return avsPayload.AVSParams, nil
}

// CheckOriginAndSender ensures the correct sender is being used.
func CheckOriginAndSender(contract *vm.Contract, origin common.Address, sender common.Address) (common.Address, error) {
	if contract.CallerAddress == sender {
		return origin, nil
	} else if origin != sender {
		return common.Address{}, fmt.Errorf(imuacmn.ErrDifferentOriginFromSender, origin.String(), sender.String())
	}
	return sender, nil
}

type TaskInfo struct {
	TaskContractAddress     common.Address
	Name                    string
	Hash                    []byte
	TaskID                  uint64
	TaskResponsePeriod      uint64
	TaskStatisticalPeriod   uint64
	TaskChallengePeriod     uint64
	ThresholdPercentage     uint8
	StartingEpoch           uint64
	ActualThreshold         string
	OptInOperators          []common.Address
	SignedOperators         []common.Address
	NoSignedOperators       []common.Address
	ErrSignedOperators      []common.Address
	TaskTotalPower          string
	OperatorActivePower     []OperatorActivePower
	IsExpected              bool
	EligibleRewardOperators []common.Address
	EligibleSlashOperators  []common.Address
}
type OperatorActivePower struct {
	Operator common.Address
	Power    *big.Int
}

type TaskResultInfo struct {
	OperatorAddress     common.Address
	TaskResponseHash    string
	TaskResponse        []byte
	BlsSignature        []byte
	TaskContractAddress common.Address
	TaskID              uint64
	Phase               uint8
}

type OperatorResInfo struct {
	TaskContractAddress common.Address
	TaskID              uint64
	OperatorAddress     common.Address
	TaskResponseHash    string
	TaskResponse        []byte
	BlsSignature        []byte
	Power               *big.Int
	Phase               uint8
}
