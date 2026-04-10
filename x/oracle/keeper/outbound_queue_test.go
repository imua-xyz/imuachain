package keeper_test

import (
	"encoding/hex"
	"testing"

	keepertest "github.com/imua-xyz/imuachain/testutil/keeper"
	"github.com/imua-xyz/imuachain/x/oracle/keeper"
	"github.com/stretchr/testify/require"
)

func TestOutboundQueue_EnqueueAndQuery(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	// Initially empty
	msgs := k.GetOutboundMessages(ctx, 101, 0, 100)
	require.Empty(t, msgs)

	// Enqueue two messages
	msg1 := keeper.OutboundMsg{
		DstChainID: 101,
		SeqNum:     1,
		Nonce:      42,
		PayloadHex: hex.EncodeToString([]byte{0x00, 0x01, 0x02}),
		Height:     100,
	}
	msg2 := keeper.OutboundMsg{
		DstChainID: 101,
		SeqNum:     2,
		Nonce:      43,
		PayloadHex: hex.EncodeToString([]byte{0x00, 0x03, 0x04}),
		Height:     101,
	}

	require.NoError(t, k.EnqueueOutboundForTest(ctx, msg1))
	require.NoError(t, k.EnqueueOutboundForTest(ctx, msg2))

	// Query all
	msgs = k.GetOutboundMessages(ctx, 101, 0, 100)
	require.Len(t, msgs, 2)
	require.Equal(t, uint64(1), msgs[0].SeqNum)
	require.Equal(t, uint64(2), msgs[1].SeqNum)

	// Query with afterSeq filter
	msgs = k.GetOutboundMessages(ctx, 101, 1, 100)
	require.Len(t, msgs, 1)
	require.Equal(t, uint64(2), msgs[0].SeqNum)

	// Query with limit
	msgs = k.GetOutboundMessages(ctx, 101, 0, 1)
	require.Len(t, msgs, 1)

	// Dequeue first message
	k.DequeueOutbound(ctx, 101, 0)
	msgs = k.GetOutboundMessages(ctx, 101, 0, 100)
	require.Len(t, msgs, 1)
	require.Equal(t, uint64(2), msgs[0].SeqNum)

	// Dequeue second message (queue becomes empty, head/tail reset)
	k.DequeueOutbound(ctx, 101, 1)
	msgs = k.GetOutboundMessages(ctx, 101, 0, 100)
	require.Empty(t, msgs)
}

func TestOutboundQueue_DifferentChains(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	msg101 := keeper.OutboundMsg{DstChainID: 101, SeqNum: 1, PayloadHex: "0001"}
	msg102 := keeper.OutboundMsg{DstChainID: 102, SeqNum: 1, PayloadHex: "0002"}

	require.NoError(t, k.EnqueueOutboundForTest(ctx, msg101))
	require.NoError(t, k.EnqueueOutboundForTest(ctx, msg102))

	msgs101 := k.GetOutboundMessages(ctx, 101, 0, 100)
	msgs102 := k.GetOutboundMessages(ctx, 102, 0, 100)

	require.Len(t, msgs101, 1)
	require.Len(t, msgs102, 1)
	require.Equal(t, "0001", msgs101[0].PayloadHex)
	require.Equal(t, "0002", msgs102[0].PayloadHex)
}

func TestOutboundQueue_EmptyChain(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	msgs := k.GetOutboundMessages(ctx, 999, 0, 100)
	require.Empty(t, msgs)

	// DequeueOutbound on empty queue should be no-op
	k.DequeueOutbound(ctx, 999, 0)
	msgs = k.GetOutboundMessages(ctx, 999, 0, 100)
	require.Empty(t, msgs)
}
