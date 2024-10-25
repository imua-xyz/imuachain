package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
)

// CommitPhase represents the phases of the Two-Phase Commit protocol
type CommitPhase uint32

const (
	PreparePhase CommitPhase = iota
	DoCommitPhase
)

type OperatorOptParams struct {
	Name            string         `json:"name"`
	BlsPublicKey    string         `json:"bls_public_key"`
	IsRegistered    bool           `json:"is_registered"`
	Action          uint64         `json:"action"`
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
	ThresholdPercentage   uint64         `json:"threshold_percentage"`
	StartingEpoch         uint64         `json:"starting_epoch"`
	OperatorAddress       sdk.AccAddress `json:"operator_address"`
	TaskResponseHash      string         `json:"task_response_hash"`
	TaskResponse          []byte         `json:"task_response"`
	BlsSignature          []byte         `json:"bls_signature"`
	Phase                 string         `json:"phase"`
	ActualThreshold       uint64         `json:"actual_threshold"`
	OptInCount            uint64         `json:"opt_in_count"`
	SignedCount           uint64         `json:"signed_count"`
	NoSignedCount         uint64         `json:"no_signed_count"`
	ErrSignedCount        uint64         `json:"err_signed_count"`
	CallerAddress         sdk.AccAddress `json:"caller_address"`
}
type BlsParams struct {
	Operator                      sdk.AccAddress
	Name                          string
	PubKey                        []byte
	PubkeyRegistrationSignature   []byte
	PubkeyRegistrationMessageHash []byte
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

const (
	RegisterAction   = 1
	DeRegisterAction = 2
	UpdateAction     = 3
)

type ChallengeParams struct {
	TaskContractAddress common.Address `json:"task_contract_address"`
	TaskHash            []byte         `json:"hash"`
	TaskID              uint64         `json:"task_id"`
	OperatorAddress     sdk.AccAddress `json:"operator_address"`
	TaskResponseHash    []byte         `json:"task_response_hash"`
	CallerAddress       sdk.AccAddress `json:"caller_address"`
}

type TaskResultParams struct {
	OperatorAddress     sdk.AccAddress `json:"operator_address"`
	TaskResponseHash    string         `json:"task_response_hash"`
	TaskResponse        []byte         `json:"task_response"`
	BlsSignature        []byte         `json:"bls_signature"`
	TaskContractAddress common.Address `json:"task_contract_address"`
	TaskID              uint64         `json:"task_id"`
	Phase               uint8          `json:"phase"`
	CallerAddress       sdk.AccAddress `json:"caller_address"`
}
