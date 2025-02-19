package avs

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	avskeep "github.com/imua-xyz/imuachain/x/avs/keeper"
)

const (
	// EventTypeRegisterAVSTask defines the event type for the avs CreateAVSTask transaction.
	EventTypeRegisterAVSTask = "TaskCreated"
)

// EmitCreateAVSTaskEvent creates a new event emitted on a EmitCreateAVSTaskEvent transaction.
func (p Precompile) EmitCreateAVSTaskEvent(ctx sdk.Context, stateDB vm.StateDB, task *avskeep.TaskInfoParams) error {
	// Prepare the event topics
	event := p.ABI.Events[EventTypeRegisterAVSTask]

	topics := make([]common.Hash, 1)

	// The first topic is always the signature of the event.
	topics[0] = event.ID

	var err error

	// Pack the arguments to be used as the Data field
	arguments := event.Inputs[0:8]
	packed, err := arguments.Pack(task.TaskID, task.TaskContractAddress, task.TaskName, task.Hash, task.TaskResponsePeriod, task.TaskChallengePeriod, task.ThresholdPercentage, task.TaskStatisticalPeriod)
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
