package keeper

import (
	"encoding/base64"
	"strconv"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func TestProcessXChainQueue_SkipInvalidPayload(t *testing.T) {
	k, ctx := MockOracleKeeper(t)

	srcChainID := uint64(7)
	qb := xchainQueuedBatch{
		Batch: RawDataXChainBatch{
			SrcChainID: srcChainID,
			BatchSeq:   1,
			Messages: []RawDataXChainMsg{
				{ID: "bad", Nonce: 1, Type: "evm", PayloadB64: "%%%"},
				{ID: "good", Nonce: 2, Type: "evm", PayloadB64: base64.StdEncoding.EncodeToString([]byte{0x01})},
			},
		},
		NextIndex: 0,
	}

	require.NoError(t, k.enqueueXChainBatch(ctx, srcChainID, qb))
	k.ProcessXChainQueue(ctx, srcChainID)

	require.False(t, k.HasXChainMsgProcessed(ctx, srcChainID, "bad"))
	require.True(t, k.HasXChainMsgProcessed(ctx, srcChainID, "good"))

	execSeq, found := k.GetXChainLastExecutedSeq(ctx, srcChainID)
	require.True(t, found)
	require.EqualValues(t, 1, execSeq)
	require.False(t, k.HasXChainQueue(ctx, srcChainID))
}

func TestProcessXChainQueue_PersistsNextIndex(t *testing.T) {
	k, ctx := MockOracleKeeper(t)

	srcChainID := uint64(8)
	msgs := make([]RawDataXChainMsg, 0, 60)
	payload := base64.StdEncoding.EncodeToString([]byte{0x01})
	for i := 0; i < 60; i++ {
		msgs = append(msgs, RawDataXChainMsg{
			ID:         "m" + strconv.Itoa(i),
			Nonce:      uint64(i + 1),
			Type:       "evm",
			PayloadB64: payload,
		})
	}
	qb := xchainQueuedBatch{
		Batch: RawDataXChainBatch{
			SrcChainID: srcChainID,
			BatchSeq:   1,
			Messages:   msgs,
		},
		NextIndex: 0,
	}
	require.NoError(t, k.enqueueXChainBatch(ctx, srcChainID, qb))

	k.ProcessXChainQueue(ctx, srcChainID)

	_, found := k.GetXChainLastExecutedSeq(ctx, srcChainID)
	require.False(t, found)
	require.True(t, k.HasXChainQueue(ctx, srcChainID))
	require.True(t, k.HasXChainMsgProcessed(ctx, srcChainID, "m0"))
	require.False(t, k.HasXChainMsgProcessed(ctx, srcChainID, "m59"))

	_, got, found, err := k.peekXChainBatch(ctx, srcChainID)
	require.NoError(t, err)
	require.True(t, found)
	require.EqualValues(t, xchainMaxDeliveriesPerEndBlock, got.NextIndex)
}

func TestDequeueXChainBatch_FIFO(t *testing.T) {
	k, ctx := MockOracleKeeper(t)

	srcChainID := uint64(9)
	require.NoError(t, k.enqueueXChainBatch(ctx, srcChainID, xchainQueuedBatch{
		Batch: RawDataXChainBatch{
			SrcChainID: srcChainID,
			BatchSeq:   1,
			Messages:   []RawDataXChainMsg{{ID: "a", Nonce: 1, Type: "evm"}},
		},
	}))
	require.NoError(t, k.enqueueXChainBatch(ctx, srcChainID, xchainQueuedBatch{
		Batch: RawDataXChainBatch{
			SrcChainID: srcChainID,
			BatchSeq:   2,
			Messages:   []RawDataXChainMsg{{ID: "b", Nonce: 2, Type: "evm"}},
		},
	}))

	// Attempt to dequeue non-head should be ignored.
	k.dequeueXChainBatch(ctx, srcChainID, 1)
	head, tail := k.GetXChainQueueHeadTail(ctx, srcChainID)
	require.EqualValues(t, 0, head)
	require.EqualValues(t, 2, tail)

	idx, qb, found, err := k.peekXChainBatch(ctx, srcChainID)
	require.NoError(t, err)
	require.True(t, found)
	require.EqualValues(t, 0, idx)
	require.EqualValues(t, 1, qb.Batch.BatchSeq)

	// Dequeue head then the next batch should become head.
	k.dequeueXChainBatch(ctx, srcChainID, 0)
	idx, qb, found, err = k.peekXChainBatch(ctx, srcChainID)
	require.NoError(t, err)
	require.True(t, found)
	require.EqualValues(t, 1, idx)
	require.EqualValues(t, 2, qb.Batch.BatchSeq)

	k.dequeueXChainBatch(ctx, srcChainID, 1)
	require.False(t, k.HasXChainQueue(ctx, srcChainID))
}

func TestEncodeGatewayOracleReceive(t *testing.T) {
	calldata, err := encodeGatewayOracleReceive(1, 2, []byte{0xaa, 0xbb})
	require.NoError(t, err)
	require.Len(t, calldata, 4+32+32+96) // 4-byte selector + ABI-encoded args

	methodID := crypto.Keccak256([]byte("oracleReceive(uint32,uint64,bytes)"))[:4]
	require.Equal(t, methodID, calldata[:4])

	uint32Ty, err := abi.NewType("uint32", "", nil)
	require.NoError(t, err)
	uint64Ty, err := abi.NewType("uint64", "", nil)
	require.NoError(t, err)
	bytesTy, err := abi.NewType("bytes", "", nil)
	require.NoError(t, err)
	args := abi.Arguments{
		{Type: uint32Ty},
		{Type: uint64Ty},
		{Type: bytesTy},
	}
	packed, err := args.Pack(uint32(1), uint64(2), []byte{0xaa, 0xbb})
	require.NoError(t, err)
	require.Equal(t, append(methodID, packed...), calldata)
}
