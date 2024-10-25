package avs

import (
	"github.com/ethereum/go-ethereum/accounts/abi"

	avstypes "github.com/ExocoreNetwork/exocore/x/avs/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
)

const (
	EventTypeAVSRegistered           = "AVSRegistered"
	EventTypeAVSUpdated              = "AVSUpdated"
	EventTypeAVSDeregistered         = "AVSDeregistered"
	EventTypeOperatorJoined          = "OperatorJoined"
	EventTypeOperatorOuted           = "OperatorOuted"
	EventTypeTaskCreated             = "TaskCreated"
	EventTypeChallengeInitiated      = "ChallengeInitiated"
	EventTypePublicKeyRegistered     = "PublicKeyRegistered"
	EventTypeTaskSubmittedByOperator = "TaskSubmittedByOperator"
)

func (p Precompile) emitEvent(ctx sdk.Context, stateDB vm.StateDB, eventName string, inputArgs abi.Arguments, args ...interface{}) error {
	event := p.ABI.Events[eventName]
	topics := []common.Hash{event.ID}

	packed, err := inputArgs.Pack(args...)
	if err != nil {
		return err
	}

	stateDB.AddLog(&ethtypes.Log{
		Address:     p.Address(),
		Topics:      topics,
		Data:        packed,
		BlockNumber: uint64(ctx.BlockHeight()),
	})

	return nil
}

// EmitAVSRegistered emits an Ethereum event when an AVS (Autonomous Verification Service) is registered.
//
// Parameters:
// - ctx: The SDK context containing information about the current state of the blockchain.
// - stateDB: The Ethereum state database where the event will be stored.
// - avs: A pointer to the AVSRegisterOrDeregisterParams struct containing the details of the AVS registration.
//
// Returns:
// - An error if there is an issue packing the event data or adding the log to the state database.
// - nil if the event is successfully emitted.
func (p Precompile) EmitAVSRegistered(ctx sdk.Context, stateDB vm.StateDB, avs *avstypes.AVSRegisterOrDeregisterParams) error {
	arguments := p.ABI.Events[EventTypeAVSRegistered].Inputs
	return p.emitEvent(ctx, stateDB, EventTypeAVSRegistered, arguments,
		avs.AvsAddress,
		avs.CallerAddress.String(),
		avs.AvsName)
}

func (p Precompile) EmitAVSUpdated(ctx sdk.Context, stateDB vm.StateDB, avs *avstypes.AVSRegisterOrDeregisterParams) error {
	arguments := p.ABI.Events[EventTypeAVSUpdated].Inputs
	return p.emitEvent(ctx, stateDB, EventTypeAVSUpdated, arguments,
		avs.AvsAddress,
		avs.CallerAddress.String(),
		avs.AvsName)
}

func (p Precompile) EmitAVSDeregistered(ctx sdk.Context, stateDB vm.StateDB, avs *avstypes.AVSRegisterOrDeregisterParams) error {
	arguments := p.ABI.Events[EventTypeAVSDeregistered].Inputs
	return p.emitEvent(ctx, stateDB, EventTypeAVSDeregistered, arguments,
		avs.AvsAddress,
		avs.CallerAddress.String(),
		avs.AvsName)
}

func (p Precompile) EmitOperatorJoined(ctx sdk.Context, stateDB vm.StateDB, params *avstypes.OperatorOptParams) error {
	arguments := p.ABI.Events[EventTypeOperatorJoined].Inputs
	return p.emitEvent(ctx, stateDB, EventTypeOperatorJoined, arguments,
		params.AvsAddress,
		params.OperatorAddress.String())
}

func (p Precompile) EmitOperatorOuted(ctx sdk.Context, stateDB vm.StateDB, params *avstypes.OperatorOptParams) error {
	arguments := p.ABI.Events[EventTypeOperatorOuted].Inputs
	return p.emitEvent(ctx, stateDB, EventTypeOperatorOuted, arguments,
		params.AvsAddress,
		params.OperatorAddress.String())
}

func (p Precompile) EmitTaskCreated(ctx sdk.Context, stateDB vm.StateDB, task *avstypes.TaskInfoParams) error {
	arguments := p.ABI.Events[EventTypeTaskCreated].Inputs
	return p.emitEvent(ctx, stateDB, EventTypeTaskCreated, arguments,
		task.CallerAddress.String(),
		task.TaskID,
		task.TaskContractAddress,
		task.TaskName,
		task.Hash,
		task.TaskResponsePeriod,
		task.TaskChallengePeriod,
		task.ThresholdPercentage,
		task.TaskStatisticalPeriod)
}

func (p Precompile) EmitChallengeInitiated(ctx sdk.Context, stateDB vm.StateDB, params *avstypes.ChallengeParams) error {
	arguments := p.ABI.Events[EventTypeChallengeInitiated].Inputs
	return p.emitEvent(ctx, stateDB, EventTypeChallengeInitiated, arguments,
		params.CallerAddress.String(),
		params.TaskHash,
		params.TaskID,
		params.TaskResponseHash,
		params.OperatorAddress.String())
}

func (p Precompile) EmitPublicKeyRegistered(ctx sdk.Context, stateDB vm.StateDB, params *avstypes.BlsParams) error {
	arguments := p.ABI.Events[EventTypePublicKeyRegistered].Inputs
	return p.emitEvent(ctx, stateDB, EventTypePublicKeyRegistered, arguments,
		params.Operator.String(),
		params.Name)
}

func (p Precompile) EmitTaskSubmittedByOperator(ctx sdk.Context, stateDB vm.StateDB, params *avstypes.TaskResultParams) error {
	arguments := p.ABI.Events[EventTypeTaskSubmittedByOperator].Inputs
	return p.emitEvent(ctx, stateDB, EventTypeTaskSubmittedByOperator, arguments,
		params.CallerAddress.String(),
		params.TaskID,
		params.TaskResponse,
		params.BlsSignature,
		params.TaskContractAddress,
		params.Phase)
}
