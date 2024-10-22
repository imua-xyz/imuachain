package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/ExocoreNetwork/exocore/testutil/keeper"
	conntypes "github.com/cosmos/ibc-go/v7/modules/core/03-connection/types"
)

func TestSetGetCoordinatorClientID(t *testing.T) {
	keeper, ctx, _ := keepertest.NewSubscriberKeeper(t)

	clientID := "07-tendermint-0"
	keeper.SetCoordinatorClientID(ctx, clientID)

	retrievedID, found := keeper.GetCoordinatorClientID(ctx)
	require.True(t, found)
	require.Equal(t, clientID, retrievedID)
}

func TestSetGetDeleteCoordinatorChannel(t *testing.T) {
	keeper, ctx, _ := keepertest.NewSubscriberKeeper(t)

	channelID := "channel-0"
	keeper.SetCoordinatorChannel(ctx, channelID)

	retrievedID, found := keeper.GetCoordinatorChannel(ctx)
	require.True(t, found)
	require.Equal(t, channelID, retrievedID)

	keeper.DeleteCoordinatorChannel(ctx)

	_, found = keeper.GetCoordinatorChannel(ctx)
	require.False(t, found)
}

func TestVerifyCoordinatorChain(t *testing.T) {
	keeper, ctx, mocks := keepertest.NewSubscriberKeeper(t)

	clientID := "07-tendermint-0"
	connectionID := "connection-0"
	keeper.SetCoordinatorClientID(ctx, clientID)

	// Test with valid connection
	mocks.ConnectionKeeper.EXPECT().GetConnection(ctx, connectionID).Return(
		conntypes.ConnectionEnd{ClientId: clientID},
		true,
	)

	err := keeper.VerifyCoordinatorChain(ctx, []string{connectionID})
	require.NoError(t, err)

	// Test with too many connection hops
	err = keeper.VerifyCoordinatorChain(ctx, []string{connectionID, "connection-1"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "must have direct connection to coordinator chain")

	// Test with non-existent connection
	mocks.ConnectionKeeper.EXPECT().GetConnection(ctx, "non-existent").Return(
		conntypes.ConnectionEnd{},
		false,
	)

	err = keeper.VerifyCoordinatorChain(ctx, []string{"non-existent"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "connection not found")

	// Test with mismatched client ID
	mocks.ConnectionKeeper.EXPECT().GetConnection(ctx, connectionID).Return(
		conntypes.ConnectionEnd{ClientId: "wrong-client-id"},
		true,
	)

	err = keeper.VerifyCoordinatorChain(ctx, []string{connectionID})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid client")
}
