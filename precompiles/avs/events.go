package avs

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	cmn "github.com/evmos/evmos/v16/precompiles/common"

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
	EventTypeOperatorLeft            = "OperatorLeft"
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
	event := p.ABI.Events[EventTypeAVSRegistered]
	topics := make([]common.Hash, 2)

	// The first topic is always the signature of the event.
	topics[0] = event.ID

	var err error
	topics[1], err = cmn.MakeTopic(avs.AvsAddress)
	if err != nil {
		return err
	}

	// Prepare the event data: sender, avsName
	arguments := abi.Arguments{event.Inputs[1], event.Inputs[2]}
	packed, err := arguments.Pack(avs.CallerAddress.String(), avs.AvsName)
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

func (p Precompile) EmitAVSUpdated(ctx sdk.Context, stateDB vm.StateDB, avs *avstypes.AVSRegisterOrDeregisterParams) error {
	event := p.ABI.Events[EventTypeAVSUpdated]
	topics := make([]common.Hash, 2)

	// The first topic is always the signature of the event.
	topics[0] = event.ID

	var err error
	topics[1], err = cmn.MakeTopic(avs.AvsAddress)
	if err != nil {
		return err
	}

	// Prepare the event data: sender, avsName
	arguments := abi.Arguments{event.Inputs[1], event.Inputs[2]}
	packed, err := arguments.Pack(avs.CallerAddress.String(), avs.AvsName)
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

func (p Precompile) EmitAVSDeregistered(ctx sdk.Context, stateDB vm.StateDB, avs *avstypes.AVSRegisterOrDeregisterParams) error {
	event := p.ABI.Events[EventTypeAVSDeregistered]
	topics := make([]common.Hash, 2)

	// The first topic is always the signature of the event.
	topics[0] = event.ID

	var err error
	topics[1], err = cmn.MakeTopic(avs.AvsAddress)
	if err != nil {
		return err
	}

	// Prepare the event data: sender, avsName
	arguments := abi.Arguments{event.Inputs[1], event.Inputs[2]}
	packed, err := arguments.Pack(avs.CallerAddress.String(), avs.AvsName)
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

func (p Precompile) EmitOperatorJoined(ctx sdk.Context, stateDB vm.StateDB, params *avstypes.OperatorOptParams) error {
	event := p.ABI.Events[EventTypeOperatorJoined]
	topics := make([]common.Hash, 2)

	// The first topic is always the signature of the event.
	topics[0] = event.ID

	var err error
	topics[1], err = cmn.MakeTopic(params.AvsAddress)
	if err != nil {
		return err
	}

	// Prepare the event data: operatorAddress
	arguments := abi.Arguments{event.Inputs[1]}
	packed, err := arguments.Pack(params.OperatorAddress.String())
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

func (p Precompile) EmitOperatorOuted(ctx sdk.Context, stateDB vm.StateDB, params *avstypes.OperatorOptParams) error {
	event := p.ABI.Events[EventTypeOperatorLeft]
	topics := make([]common.Hash, 2)

	// The first topic is always the signature of the event.
	topics[0] = event.ID

	var err error
	topics[1], err = cmn.MakeTopic(params.AvsAddress)
	if err != nil {
		return err
	}

	// Prepare the event data: operatorAddress
	arguments := abi.Arguments{event.Inputs[1]}
	packed, err := arguments.Pack(params.OperatorAddress.String())
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

func (p Precompile) EmitTaskCreated(ctx sdk.Context, stateDB vm.StateDB, task *avstypes.TaskInfoParams) error {
	event := p.ABI.Events[EventTypeTaskCreated]
	topics := make([]common.Hash, 3)

	// The first topic is always the signature of the event.
	topics[0] = event.ID

	var err error
	topics[1], err = cmn.MakeTopic(task.TaskContractAddress)
	if err != nil {
		return err
	}

	topics[2], err = cmn.MakeTopic(task.TaskID)
	if err != nil {
		return err
	}
	// Prepare the event data:sender,name, hash, taskResponsePeriod,taskChallengePeriod,
	// thresholdPercentage,taskStatisticalPeriod
	arguments := abi.Arguments{
		event.Inputs[2], event.Inputs[3], event.Inputs[4],
		event.Inputs[5], event.Inputs[6], event.Inputs[7], event.Inputs[8],
	}
	packed, err := arguments.Pack(task.CallerAddress.String(), task.TaskName, task.Hash, task.TaskResponsePeriod, task.TaskChallengePeriod,
		task.ThresholdPercentage, task.TaskStatisticalPeriod)
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
		params.OperatorAddress.String(),
		params.Name)
}

func (p Precompile) EmitTaskSubmittedByOperator(ctx sdk.Context, stateDB vm.StateDB, params *avstypes.TaskResultParams) error {
	event := p.ABI.Events[EventTypeTaskSubmittedByOperator]
	topics := make([]common.Hash, 3)

	// The first topic is always the signature of the event.
	topics[0] = event.ID

	var err error
	topics[1], err = cmn.MakeTopic(params.TaskContractAddress)
	if err != nil {
		return err
	}

	topics[2], err = cmn.MakeTopic(params.TaskID)
	if err != nil {
		return err
	}
	// Prepare the event data:sender,TaskResponse, BlsSignature, Phase
	arguments := abi.Arguments{event.Inputs[2], event.Inputs[3], event.Inputs[4], event.Inputs[5]}
	// #nosec G115
	// TODO: consider modify define of Phase to uint8
	packed, err := arguments.Pack(params.CallerAddress.String(), params.TaskResponse, params.BlsSignature, uint8(params.Phase))
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
