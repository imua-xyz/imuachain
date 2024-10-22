package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	testkeeper "github.com/ExocoreNetwork/exocore/testutil/keeper"
	epochstypes "github.com/ExocoreNetwork/exocore/x/epochs/types"
)

func TestAppendAndGetChainsToInitTimeout(t *testing.T) {
	k, ctx, _ := testkeeper.NewCoordinatorKeeper(t)
	epoch := epochstypes.NewEpoch(1, "test")
	chainID := "test-chain"

	// Append chain to init timeout
	k.AppendChainToInitTimeout(ctx, epoch, chainID)
	k.SetChainInitTimeout(ctx, chainID, epoch)

	// Get chains to init timeout
	chains := k.GetChainsToInitTimeout(ctx, epoch)
	require.Equal(t, 1, len(chains.List))
	require.Equal(t, chainID, chains.List[0])
	epoch, found := k.GetChainInitTimeout(ctx, chainID)
	require.True(t, found)
	require.Equal(t, epoch, epoch)
}

func TestRemoveChainFromInitTimeout(t *testing.T) {
	k, ctx, _ := testkeeper.NewCoordinatorKeeper(t)
	epoch := epochstypes.NewEpoch(1, "test")
	chainID1 := "test-chain-1"
	chainID2 := "test-chain-2"

	// Append chains to init timeout
	k.AppendChainToInitTimeout(ctx, epoch, chainID1)
	k.SetChainInitTimeout(ctx, chainID1, epoch)
	k.AppendChainToInitTimeout(ctx, epoch, chainID2)
	k.SetChainInitTimeout(ctx, chainID2, epoch)

	// Remove one chain
	k.RemoveChainFromInitTimeout(ctx, epoch, chainID1)
	k.DeleteChainInitTimeout(ctx, chainID1)

	// Check remaining chains
	chains := k.GetChainsToInitTimeout(ctx, epoch)
	require.Equal(t, 1, len(chains.List))
	require.Equal(t, chainID2, chains.List[0])
	_, found := k.GetChainInitTimeout(ctx, chainID1)
	require.False(t, found)
	epoch, found = k.GetChainInitTimeout(ctx, chainID2)
	require.True(t, found)
	require.Equal(t, epoch, epoch)
}

func TestSetAndGetVscTimeout(t *testing.T) {
	k, ctx, _ := testkeeper.NewCoordinatorKeeper(t)
	chainID := "test-chain"
	vscID := uint64(1)
	timeout := epochstypes.NewEpoch(2, "test")

	// Set VSC timeout
	k.SetVscTimeout(ctx, chainID, vscID, timeout)

	// Get VSC timeout
	storedTimeout, found := k.GetVscTimeout(ctx, chainID, vscID)
	require.True(t, found)
	require.Equal(t, timeout, storedTimeout)

	// Delete VSC timeout
	k.DeleteVscTimeout(ctx, chainID, vscID)

	// Check if deleted
	_, found = k.GetVscTimeout(ctx, chainID, vscID)
	require.False(t, found)
}

func TestGetFirstVscTimeout(t *testing.T) {
	k, ctx, _ := testkeeper.NewCoordinatorKeeper(t)
	chainID := "test-chain"
	timeout1 := epochstypes.NewEpoch(2, "test")
	timeout2 := epochstypes.NewEpoch(3, "test")

	// Set multiple VSC timeouts
	k.SetVscTimeout(ctx, chainID, 1, timeout1)
	k.SetVscTimeout(ctx, chainID, 2, timeout2)

	// Get first VSC timeout
	firstTimeout, vscID, found := k.GetFirstVscTimeout(ctx, chainID)
	require.True(t, found)
	require.Equal(t, timeout1, firstTimeout)
	require.Equal(t, uint64(1), vscID)
}

func TestRemoveTimedoutSubscribers(t *testing.T) {
	k, ctx, _ := testkeeper.NewCoordinatorKeeper(t)
	epochIdentifier := "test"
	epochNumber := int64(5)

	chainID1 := "test-chain-1"
	chainID2 := "test-chain-2"
	channelID := "channel-0"

	// Set up test data
	epoch := epochstypes.NewEpoch(uint64(epochNumber), epochIdentifier)
	k.AppendChainToInitTimeout(ctx, epoch, chainID1)

	// channel must be set for vsc timeout to occur, since packets are only sent over channels
	k.SetChannelForChain(ctx, chainID2, channelID)
	k.SetChainForChannel(ctx, channelID, chainID2)
	k.SetVscTimeout(ctx, chainID2, 1, epoch)

	// Call RemoveTimedoutSubscribers
	k.RemoveTimedoutSubscribers(ctx, epochIdentifier, epochNumber)

	// Verify results
	_, found := k.GetChainInitTimeout(ctx, chainID1)
	require.False(t, found)

	_, found = k.GetVscTimeout(ctx, chainID2, 1)
	require.False(t, found)
}
