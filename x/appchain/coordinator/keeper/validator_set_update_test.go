package keeper_test

import (
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	testkeeper "github.com/ExocoreNetwork/exocore/testutil/keeper"
	testutiltx "github.com/ExocoreNetwork/exocore/testutil/tx"
	keytypes "github.com/ExocoreNetwork/exocore/types/keys"
	commontypes "github.com/ExocoreNetwork/exocore/x/appchain/common/types"
	"github.com/ExocoreNetwork/exocore/x/appchain/coordinator/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
)

func TestQueueValidatorUpdatesForEpochID(t *testing.T) {
	keeper, ctx, mocks := testkeeper.NewCoordinatorKeeper(t)

	epochID := "test-epoch"
	epochNumber := int64(1)
	chainID := "test-chain"

	mocks.AVSKeeper.EXPECT().GetEpochEndChainIDs(ctx, epochID, epochNumber).Return([]string{chainID})
	mocks.OperatorKeeper.EXPECT().GetActiveOperatorsForChainID(gomock.Any(), chainID).Return([]sdk.AccAddress{}, []keytypes.WrappedConsKey{})
	mocks.OperatorKeeper.EXPECT().GetVotePowerForChainID(gomock.Any(), gomock.Any(), chainID).Return([]int64{}, nil)

	keeper.QueueValidatorUpdatesForEpochID(ctx, epochID, epochNumber)

	// Verify that the validator updates were queued
	packets := keeper.GetPendingVscPackets(ctx, chainID)
	require.Equal(t, 1, len(packets.List))
}

func TestQueueValidatorUpdatesForChainID(t *testing.T) {
	keeper, ctx, mocks := testkeeper.NewCoordinatorKeeper(t)

	chainID := "test-chain"
	operator := sdk.AccAddress("testoperator")
	pubKey := ed25519.GenPrivKey().PubKey()
	wrappedKey := keytypes.NewWrappedConsKeyFromSdkKey(pubKey)

	// set up max validators for chain
	keeper.SetMaxValidatorsForChain(ctx, chainID, 100)

	mocks.OperatorKeeper.EXPECT().GetActiveOperatorsForChainID(ctx, chainID).Return([]sdk.AccAddress{operator}, []keytypes.WrappedConsKey{wrappedKey}).Times(1)
	mocks.OperatorKeeper.EXPECT().GetVotePowerForChainID(ctx, []sdk.AccAddress{operator}, chainID).Return([]int64{100}, nil)

	err := keeper.QueueValidatorUpdatesForChainID(ctx, chainID)
	require.NoError(t, err)

	// Verify that the validator updates were queued
	packets := keeper.GetPendingVscPackets(ctx, chainID)
	require.Equal(t, 1, len(packets.List))
	require.Equal(t, uint64(1), packets.List[0].ValsetUpdateID)
	require.Equal(t, 1, len(packets.List[0].ValidatorUpdates))
}

func TestSendQueuedValidatorUpdates(t *testing.T) {
	keeper, ctx, mocks := testkeeper.NewCoordinatorKeeper(t)

	chainID := "test-chain"
	channelID := "channel-0"
	epochNumber := int64(1)

	keeper.SetChannelForChain(ctx, chainID, channelID)

	key := testutiltx.GenerateConsensusKey()
	packet := commontypes.ValidatorSetChangePacketData{
		ValsetUpdateID: 1,
		ValidatorUpdates: []abci.ValidatorUpdate{
			{PubKey: *key.ToTmProtoKey(), Power: 100},
		},
	}
	keeper.AppendPendingVscPacket(ctx, chainID, packet)

	mocks.ScopedKeeper.EXPECT().GetCapability(gomock.Any(), gomock.Any()).Return(nil, true)
	mocks.ChannelKeeper.EXPECT().GetChannel(gomock.Any(), gomock.Any(), gomock.Any()).Return(channeltypes.Channel{}, true)
	mocks.ChannelKeeper.EXPECT().SendPacket(
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
	).Return(uint64(1), nil)

	keeper.SendQueuedValidatorUpdates(ctx, epochNumber)

	// Verify that the pending packets were sent and cleared
	packets := keeper.GetPendingVscPackets(ctx, chainID)
	require.Empty(t, packets.List)
}

func TestAppendGetSetPendingVscPacket(t *testing.T) {
	keeper, ctx, _ := testkeeper.NewCoordinatorKeeper(t)

	chainID := "test-chain"
	key := testutiltx.GenerateConsensusKey()
	packet := commontypes.ValidatorSetChangePacketData{
		ValsetUpdateID: 1,
		ValidatorUpdates: []abci.ValidatorUpdate{
			{PubKey: *key.ToTmProtoKey(), Power: 100},
		},
	}

	// Test appending
	keeper.AppendPendingVscPacket(ctx, chainID, packet)

	// Test getting
	packets := keeper.GetPendingVscPackets(ctx, chainID)
	require.Equal(t, 1, len(packets.List))
	require.Equal(t, packet, packets.List[0])

	// Test setting
	newPackets := types.ValidatorSetChangePackets{
		List: []commontypes.ValidatorSetChangePacketData{packet, packet},
	}
	keeper.SetPendingVscPackets(ctx, chainID, newPackets)

	gotPackets := keeper.GetPendingVscPackets(ctx, chainID)
	require.Equal(t, 2, len(gotPackets.List))

	// Test setting empty (should delete)
	keeper.SetPendingVscPackets(ctx, chainID, types.ValidatorSetChangePackets{})
	gotPackets = keeper.GetPendingVscPackets(ctx, chainID)
	require.Empty(t, gotPackets.List)
}
