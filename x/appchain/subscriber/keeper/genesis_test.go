package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ExocoreNetwork/exocore/testutil/keeper"
	commontypes "github.com/ExocoreNetwork/exocore/x/appchain/common/types"
	"github.com/ExocoreNetwork/exocore/x/appchain/subscriber/types"
	abci "github.com/cometbft/cometbft/abci/types"
	_07_tendermint "github.com/cosmos/ibc-go/v7/modules/light-clients/07-tendermint"
)

func TestInitGenesis(t *testing.T) {
	k, ctx, mocks := keeper.NewSubscriberKeeper(t)

	// Mock the necessary function calls
	mocks.ScopedKeeper.EXPECT().GetCapability(ctx, gomock.Any()).Return(nil, false)
	mocks.PortKeeper.EXPECT().BindPort(ctx, commontypes.SubscriberPortID).Return(nil)
	mocks.ClientKeeper.EXPECT().CreateClient(ctx, gomock.Any(), gomock.Any()).Return("test-client-id", nil)

	genesisState := types.GenesisState{
		Params: commontypes.DefaultSubscriberParams(),
		Coordinator: commontypes.CoordinatorInfo{
			ClientState:    &_07_tendermint.ClientState{},
			ConsensusState: &_07_tendermint.ConsensusState{},
			InitialValSet:  []abci.ValidatorUpdate{},
		},
	}

	validatorUpdates := k.InitGenesis(ctx, genesisState)

	// Verify that the genesis state was properly initialized
	require.Equal(t, genesisState.Params, k.GetSubscriberParams(ctx))
	require.Equal(t, commontypes.SubscriberPortID, k.GetPort(ctx))
	clientID, ok := k.GetCoordinatorClientID(ctx)
	require.True(t, ok)
	require.Equal(t, "test-client-id", clientID)
	require.Equal(t, types.FirstValsetUpdateID, k.GetValsetUpdateIDForHeight(ctx, ctx.BlockHeight()))
	require.NotNil(t, validatorUpdates)
}

func TestExportGenesis(t *testing.T) {
	k, ctx, _ := keeper.NewSubscriberKeeper(t)

	// Set up some state
	k.SetSubscriberParams(ctx, commontypes.DefaultSubscriberParams())

	genesisState := k.ExportGenesis(ctx)

	// Verify that the genesis state was properly exported
	require.Equal(t, commontypes.DefaultSubscriberParams(), genesisState.Params)
}

func TestInitGenesisNonZeroHeight(t *testing.T) {
	k, ctx, _ := keeper.NewSubscriberKeeper(t)

	// Set a non-zero block height
	header := ctx.BlockHeader()
	header.Height = 10
	ctx = ctx.WithBlockHeader(header)

	genesisState := types.GenesisState{
		Params: commontypes.DefaultSubscriberParams(),
	}

	require.Panics(t, func() {
		k.InitGenesis(ctx, genesisState)
	}, "InitGenesis should panic when block height is non-zero")
}
