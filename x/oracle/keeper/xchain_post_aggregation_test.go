package keeper_test

import (
	"encoding/base64"
	"encoding/json"
	"strconv"
	"testing"

	keepertest "github.com/imua-xyz/imuachain/testutil/keeper"
	"github.com/imua-xyz/imuachain/x/oracle/keeper"
	"github.com/stretchr/testify/require"
)

func TestUpdateXChainMsgs_Basic(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	rootHash := make([]byte, 32)
	for i := 0; i < 32; i++ {
		rootHash[i] = byte(i)
	}
	payload := base64.StdEncoding.EncodeToString([]byte{0x01})

	batch := keeper.RawDataXChainBatch{
		SrcChainID: 123,
		BatchSeq:   1,
		Messages: []keeper.RawDataXChainMsg{
			{ID: "tx1:0", Nonce: 1, Type: "evm", PayloadB64: payload},
			{ID: "tx1:0", Nonce: 1, Type: "evm", PayloadB64: payload}, // duplicate in same batch
		},
	}
	raw, err := json.Marshal(batch)
	require.NoError(t, err)

	err = keeper.UpdateXChainMsgs(ctx, rootHash, raw, 99, 7, k)
	require.NoError(t, err)

	lastSeq, found := k.GetXChainLastSeq(ctx, 123)
	require.True(t, found)
	require.EqualValues(t, 1, lastSeq)
	require.False(t, k.HasXChainMsgProcessed(ctx, 123, "tx1:0"))

	// replay same batch should be idempotent
	err = keeper.UpdateXChainMsgs(ctx, rootHash, raw, 99, 7, k)
	require.NoError(t, err)

	// Execute queued delivery (budgeted EndBlock consumer).
	k.ProcessXChainQueue(ctx, 123)
	require.True(t, k.HasXChainMsgProcessed(ctx, 123, "tx1:0"))
	execSeq, found := k.GetXChainLastExecutedSeq(ctx, 123)
	require.True(t, found)
	require.EqualValues(t, 1, execSeq)

	// Queue should be empty after processing.
	require.False(t, k.HasXChainQueue(ctx, 123))
	head, tail := k.GetXChainQueueHeadTail(ctx, 123)
	require.EqualValues(t, 0, head)
	require.EqualValues(t, 0, tail)
}

func TestUpdateXChainMsgs_SeqGap(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	rootHash := make([]byte, 32)
	payload := base64.StdEncoding.EncodeToString([]byte{0x01})

	batch1 := keeper.RawDataXChainBatch{
		SrcChainID: 1,
		BatchSeq:   1,
		Messages:   []keeper.RawDataXChainMsg{{ID: "a", Nonce: 1, Type: "evm", PayloadB64: payload}},
	}
	raw1, _ := json.Marshal(batch1)
	require.NoError(t, keeper.UpdateXChainMsgs(ctx, rootHash, raw1, 1, 1, k))

	// gap: try to apply seq=3 while last=1
	batch3 := keeper.RawDataXChainBatch{
		SrcChainID: 1,
		BatchSeq:   3,
		Messages:   []keeper.RawDataXChainMsg{{ID: "b", Nonce: 3, Type: "evm", PayloadB64: payload}},
	}
	raw3, _ := json.Marshal(batch3)
	err := keeper.UpdateXChainMsgs(ctx, rootHash, raw3, 1, 2, k)
	require.Error(t, err)
}

func TestUpdateXChainMsgs_BudgetedDelivery(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	rootHash := make([]byte, 32)

	msgs := make([]keeper.RawDataXChainMsg, 0, 60)
	payload := base64.StdEncoding.EncodeToString([]byte{0x01})
	for i := 0; i < 60; i++ {
		msgs = append(msgs, keeper.RawDataXChainMsg{
			ID:         "m" + strconv.Itoa(i),
			Nonce:      uint64(i + 1),
			Type:       "evm",
			PayloadB64: payload,
		})
	}
	batch := keeper.RawDataXChainBatch{SrcChainID: 9, BatchSeq: 1, Messages: msgs}
	raw, err := json.Marshal(batch)
	require.NoError(t, err)

	require.NoError(t, keeper.UpdateXChainMsgs(ctx, rootHash, raw, 1, 1, k))

	// One pass should not finish 60 messages if the budget is < 60.
	k.ProcessXChainQueue(ctx, 9)

	// At least the first one should be processed, but not all.
	require.True(t, k.HasXChainMsgProcessed(ctx, 9, msgs[0].ID))
	execSeq, found := k.GetXChainLastExecutedSeq(ctx, 9)
	require.False(t, found)
	require.EqualValues(t, 0, execSeq)

	// Drain fully.
	for i := 0; i < 10; i++ {
		k.ProcessXChainQueue(ctx, 9)
	}
	execSeq, found = k.GetXChainLastExecutedSeq(ctx, 9)
	require.True(t, found)
	require.EqualValues(t, 1, execSeq)
}

func TestUpdateXChainMsgs_InvalidPayloadB64(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	rootHash := make([]byte, 32)
	batch := keeper.RawDataXChainBatch{
		SrcChainID: 2,
		BatchSeq:   1,
		Messages: []keeper.RawDataXChainMsg{
			{ID: "badpayload", Nonce: 1, Type: "evm", PayloadB64: "%%%"},
		},
	}
	raw, err := json.Marshal(batch)
	require.NoError(t, err)

	err = keeper.UpdateXChainMsgs(ctx, rootHash, raw, 1, 1, k)
	require.Error(t, err)

	_, found := k.GetXChainLastSeq(ctx, 2)
	require.False(t, found)
	require.False(t, k.HasXChainQueue(ctx, 2))
}

func TestUpdateXChainMsgs_EmptyMsgID(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	rootHash := make([]byte, 32)
	batch := keeper.RawDataXChainBatch{
		SrcChainID: 3,
		BatchSeq:   1,
		Messages: []keeper.RawDataXChainMsg{
			{ID: "", Nonce: 1, Type: "evm"},
		},
	}
	raw, err := json.Marshal(batch)
	require.NoError(t, err)

	err = keeper.UpdateXChainMsgs(ctx, rootHash, raw, 1, 1, k)
	require.Error(t, err)

	_, found := k.GetXChainLastSeq(ctx, 3)
	require.False(t, found)
	require.False(t, k.HasXChainQueue(ctx, 3))
}
