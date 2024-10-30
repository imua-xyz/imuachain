package keeper

import (
	"fmt"

	commontypes "github.com/ExocoreNetwork/exocore/x/appchain/common/types"
	"github.com/ExocoreNetwork/exocore/x/appchain/subscriber/types"
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// InitGenesis initializes the subscriber module's state from a genesis state.
// This state is typically obtained from a coordinator chain, however, it
// may be exported from a previous state of the subscriber chain.
func (k Keeper) InitGenesis(ctx sdk.Context, gs types.GenesisState) []abci.ValidatorUpdate {
	// do not support switchover use case yet.
	if ctx.BlockHeight() > 0 {
		// this is not supported not because of any technical limitations
		// but rather because the business logic and the security logic
		// around switchover is not yet fully designed.
		panic("switchover use case not supported yet")
	}
	k.SetSubscriberParams(ctx, gs.Params)
	k.SetPort(ctx, commontypes.SubscriberPortID)
	// only bind to the port if the capability keeper hasn't done so already
	if !k.IsBound(ctx, commontypes.SubscriberPortID) {
		k.Logger(ctx).Info("binding port", "port", commontypes.SubscriberPortID)
		// does not return an error on failure, instead it panics
		k.portKeeper.BindPort(ctx, commontypes.SubscriberPortID)
	}
	// the client state and the consensus state are provided by the coordinator.
	clientID, err := k.clientKeeper.CreateClient(
		ctx, gs.Coordinator.ClientState, gs.Coordinator.ConsensusState,
	)
	if err != nil {
		panic(fmt.Sprintf("could not create client for coordinator chain: %v", err))
	}
	k.SetCoordinatorClientID(ctx, clientID)
	// TODO: in the case of switchover, this number may be a different value
	k.SetValsetUpdateIDForHeight(ctx, ctx.BlockHeight(), types.FirstValsetUpdateID)
	return k.ApplyValidatorChanges(ctx, gs.Coordinator.InitialValSet)
}

func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	return &types.GenesisState{
		Params: k.GetSubscriberParams(ctx),
	}
}
