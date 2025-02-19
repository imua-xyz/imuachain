package keeper

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/imua-xyz/imuachain/x/avs/types"
)

// GetParams get all parameters as types.Params
func (k Keeper) GetParams(ctx sdk.Context) (*types.Params, error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixParams)
	ifExist := store.Has(types.ParamsKey)
	if !ifExist {
		return nil, fmt.Errorf("params %s not found", types.KeyPrefixParams)
	}

	value := store.Get(types.ParamsKey)

	ret := &types.Params{}
	k.cdc.MustUnmarshal(value, ret)
	return ret, nil
}

// SetParams set the params
func (k Keeper) SetParams(ctx sdk.Context, params *types.Params) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixParams)
	bz := k.cdc.MustMarshal(params)
	store.Set(types.ParamsKey, bz)
	return nil
}

type OperatorOptParams struct {
	Name            string
	BlsPublicKey    string
	IsRegistered    bool
	Action          uint64
	OperatorAddress string
	Status          string
	AvsAddress      string
}

type TaskInfoParams struct {
	TaskContractAddress   string `json:"task_contract_address"`
	TaskName              string `json:"name"`
	Hash                  []byte `json:"hash"`
	TaskID                uint64 `json:"task_id"`
	TaskResponsePeriod    uint64 `json:"task_response_period"`
	TaskStatisticalPeriod uint64 `json:"task_statistical_period"`
	TaskChallengePeriod   uint64 `json:"task_challenge_period"`
	ThresholdPercentage   uint64 `json:"threshold_percentage"`
	StartingEpoch         uint64 `json:"starting_epoch"`
	OperatorAddress       string `json:"operator_address"`
	TaskResponseHash      string `json:"task_response_hash"`
	TaskResponse          []byte `json:"task_response"`
	BlsSignature          []byte `json:"bls_signature"`
	Stage                 string `json:"stage"`
	ActualThreshold       uint64 `json:"actual_threshold"`
	OptInCount            uint64 `json:"opt_in_count"`
	SignedCount           uint64 `json:"signed_count"`
	NoSignedCount         uint64 `json:"no_signed_count"`
	ErrSignedCount        uint64 `json:"err_signed_count"`
	CallerAddress         string `json:"caller_address"`
}
type BlsParams struct {
	Operator                      string
	Name                          string
	PubKey                        []byte
	PubkeyRegistrationSignature   []byte
	PubkeyRegistrationMessageHash []byte
}

type ProofParams struct {
	TaskID              string
	TaskContractAddress string
	AvsAddress          string
	Aggregator          string
	OperatorStatus      []OperatorStatusParams
	CallerAddress       string
}
type OperatorStatusParams struct {
	OperatorAddress string
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
	CallerAddress       string         `json:"caller_address"`
}
