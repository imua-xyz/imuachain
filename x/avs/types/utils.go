package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
)

type OperatorOptParams struct {
	Name            string         `json:"name"`
	BlsPublicKey    string         `json:"bls_public_key"`
	IsRegistered    bool           `json:"is_registered"`
	Action          OperatorAction `json:"action"`
	OperatorAddress sdk.AccAddress `json:"operator_address"`
	Status          string         `json:"status"`
	AvsAddress      common.Address `json:"avs_address"`
}

type TaskInfoParams struct {
	TaskContractAddress   common.Address `json:"task_contract_address"`
	TaskName              string         `json:"name"`
	Hash                  []byte         `json:"hash"`
	TaskID                uint64         `json:"task_id"`
	TaskResponsePeriod    uint64         `json:"task_response_period"`
	TaskStatisticalPeriod uint64         `json:"task_statistical_period"`
	TaskChallengePeriod   uint64         `json:"task_challenge_period"`
	ThresholdPercentage   uint8          `json:"threshold_percentage"`
	StartingEpoch         uint64         `json:"starting_epoch"`
	OperatorAddress       sdk.AccAddress `json:"operator_address"`
	TaskResponseHash      string         `json:"task_response_hash"`
	TaskResponse          []byte         `json:"task_response"`
	BlsSignature          []byte         `json:"bls_signature"`
	ActualThreshold       uint64         `json:"actual_threshold"`
	OptInCount            uint64         `json:"opt_in_count"`
	SignedCount           uint64         `json:"signed_count"`
	NoSignedCount         uint64         `json:"no_signed_count"`
	ErrSignedCount        uint64         `json:"err_signed_count"`
	CallerAddress         sdk.AccAddress `json:"caller_address"`
}
type BlsParams struct {
	OperatorAddress             sdk.AccAddress
	AvsAddress                  common.Address
	PubKey                      []byte
	PubKeyRegistrationSignature []byte
}

type ProofParams struct {
	TaskID              string
	TaskContractAddress common.Address
	AvsAddress          common.Address
	Aggregator          string
	OperatorStatus      []OperatorStatusParams
	CallerAddress       sdk.AccAddress
}
type OperatorStatusParams struct {
	OperatorAddress sdk.AccAddress
	Status          string
	ProofData       string
}

// OperatorAction represents the type of action an operator can perform
type OperatorAction uint64

const (
	RegisterAction   OperatorAction = 1
	DeRegisterAction OperatorAction = 2
	UpdateAction     OperatorAction = 3
)

type ChallengeParams struct {
	TaskContractAddress     common.Address   `json:"task_contract_address"`
	TaskID                  uint64           `json:"task_id"`
	CallerAddress           sdk.AccAddress   `json:"caller_address"`
	ActualThreshold         uint8            `json:"actual_threshold"`
	IsExpected              bool             `json:"is_expected"`
	EligibleRewardOperators []common.Address `json:"eligible_reward_operators"`
	EligibleSlashOperators  []common.Address `json:"eligible_slash_operators"`
}

type TaskResultParams struct {
	OperatorAddress     sdk.AccAddress `json:"operator_address"`
	TaskResponseHash    string         `json:"task_response_hash"`
	TaskResponse        []byte         `json:"task_response"`
	BlsSignature        []byte         `json:"bls_signature"`
	TaskContractAddress common.Address `json:"task_contract_address"`
	TaskID              uint64         `json:"task_id"`
	Phase               Phase          `json:"phase"`
	CallerAddress       sdk.AccAddress `json:"caller_address"`
}

func ValidatePhase(phase Phase) error {
	switch phase {
	case PhasePrepare, PhaseDoCommit:
		return nil
	default:
		return fmt.Errorf("invalid phase value: %d", phase)
	}
}

func AddressToString(addresses []common.Address) []string {
	if len(addresses) == 0 {
		return nil
	}
	result := make([]string, len(addresses))
	for i, address := range addresses {
		result[i] = address.String()
	}
	return result
}
