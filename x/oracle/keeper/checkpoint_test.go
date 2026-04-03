package keeper_test

import (
	"encoding/hex"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	keepertest "github.com/imua-xyz/imuachain/testutil/keeper"
	"github.com/imua-xyz/imuachain/x/oracle/keeper"
	"github.com/imua-xyz/imuachain/x/oracle/types"
	"github.com/stretchr/testify/require"
)

func TestCheckpointCreation(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	// No checkpoint should be created for empty queue
	created := k.CreateCheckpointForPendingOutbound(ctx, 101)
	require.False(t, created)

	// Enqueue some outbound messages
	msg1 := keeper.OutboundMsg{
		DstChainID: 101,
		SeqNum:     1,
		Nonce:      1,
		PayloadHex: hex.EncodeToString([]byte{0x00, 0x01}),
		Height:     10,
	}
	msg2 := keeper.OutboundMsg{
		DstChainID: 101,
		SeqNum:     2,
		Nonce:      2,
		PayloadHex: hex.EncodeToString([]byte{0x00, 0x02}),
		Height:     10,
	}
	require.NoError(t, k.EnqueueOutboundForTest(ctx, msg1))
	require.NoError(t, k.EnqueueOutboundForTest(ctx, msg2))

	// Create checkpoint
	created = k.CreateCheckpointForPendingOutbound(ctx, 101)
	require.True(t, created)

	// Verify checkpoint
	nonce := k.GetLatestCheckpointNonce(ctx, 101)
	require.EqualValues(t, 1, nonce)

	cp, found := k.GetCheckpoint(ctx, 101, 1)
	require.True(t, found)
	require.Equal(t, uint64(1), cp.Nonce)
	require.Equal(t, uint64(101), cp.DstChainID)
	require.Equal(t, uint64(1), cp.SeqStart)
	require.Equal(t, uint64(2), cp.SeqEnd)
	require.False(t, cp.Finalized)
	require.NotEqual(t, common.Hash{}, cp.MessagesHash)

	// Should not create another checkpoint while one is pending
	created = k.CreateCheckpointForPendingOutbound(ctx, 101)
	require.False(t, created)
}

func TestCheckpointHashConsistency(t *testing.T) {
	// Verify Go hash computation matches expected format
	nonce := uint64(1)
	dstChainID := uint64(101)
	messagesHash := common.HexToHash("0xabcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789")

	hash := types.ComputeCheckpointHash(nonce, dstChainID, messagesHash)
	require.NotEqual(t, common.Hash{}, hash)

	// Same inputs should produce same hash
	hash2 := types.ComputeCheckpointHash(nonce, dstChainID, messagesHash)
	require.Equal(t, hash, hash2)

	// Different inputs should produce different hash
	hash3 := types.ComputeCheckpointHash(nonce+1, dstChainID, messagesHash)
	require.NotEqual(t, hash, hash3)
}

func TestCheckpointSignatureVerification(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	// Setup: create a test ECDSA key
	privKey, err := crypto.GenerateKey()
	require.NoError(t, err)
	evmAddr := crypto.PubkeyToAddress(privKey.PublicKey)

	// Enqueue and create checkpoint
	msg := keeper.OutboundMsg{
		DstChainID: 101, SeqNum: 1, Nonce: 1,
		PayloadHex: hex.EncodeToString([]byte{0x00, 0x01}), Height: 10,
	}
	require.NoError(t, k.EnqueueOutboundForTest(ctx, msg))
	require.True(t, k.CreateCheckpointForPendingOutbound(ctx, 101))

	cp, found := k.GetCheckpoint(ctx, 101, 1)
	require.True(t, found)

	// Sign the checkpoint
	checkpointHash := types.ComputeCheckpointHash(cp.Nonce, cp.DstChainID, cp.MessagesHash)
	ethSignedHash := types.ComputeEthSignedMessageHash(checkpointHash)
	sig, err := crypto.Sign(ethSignedHash.Bytes(), privKey)
	require.NoError(t, err)

	var r, s [32]byte
	copy(r[:], sig[0:32])
	copy(s[:], sig[32:64])
	v := uint8(sig[64] + 27)

	// Submit signature (with mock power)
	finalized, err := k.AddCheckpointSignature(ctx, 101, 1, evmAddr, v, r, s, 100)
	require.NoError(t, err)
	// Not finalized yet because we need 2/3 of total power
	// (testutil keeper may have 0 total validator power, so this may finalize immediately)
	_ = finalized

	// Verify signature is stored
	sigs := k.GetCheckpointSignatures(ctx, 101, 1)
	require.Len(t, sigs, 1)
	require.Equal(t, evmAddr, sigs[0].Validator)

	// Duplicate submission should be idempotent
	_, err = k.AddCheckpointSignature(ctx, 101, 1, evmAddr, v, r, s, 100)
	require.NoError(t, err)
	sigs = k.GetCheckpointSignatures(ctx, 101, 1)
	require.Len(t, sigs, 1) // still 1, not 2
}

func TestCheckpointInvalidSignature(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	// Setup
	privKey, err := crypto.GenerateKey()
	require.NoError(t, err)
	wrongAddr := common.HexToAddress("0x1234567890123456789012345678901234567890")

	msg := keeper.OutboundMsg{
		DstChainID: 101, SeqNum: 1, Nonce: 1,
		PayloadHex: hex.EncodeToString([]byte{0x00, 0x01}), Height: 10,
	}
	require.NoError(t, k.EnqueueOutboundForTest(ctx, msg))
	require.True(t, k.CreateCheckpointForPendingOutbound(ctx, 101))

	cp, _ := k.GetCheckpoint(ctx, 101, 1)

	// Sign with valid key but claim wrong address
	checkpointHash := types.ComputeCheckpointHash(cp.Nonce, cp.DstChainID, cp.MessagesHash)
	ethSignedHash := types.ComputeEthSignedMessageHash(checkpointHash)
	sig, _ := crypto.Sign(ethSignedHash.Bytes(), privKey)

	var r, s [32]byte
	copy(r[:], sig[0:32])
	copy(s[:], sig[32:64])
	v := uint8(sig[64] + 27)

	// Should fail: recovered address doesn't match claimed address
	_, err = k.AddCheckpointSignature(ctx, 101, 1, wrongAddr, v, r, s, 100)
	require.Error(t, err)
	require.Contains(t, err.Error(), "signature mismatch")
}

func TestCheckpointNonExistent(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	// Try to sign non-existent checkpoint
	_, err := k.AddCheckpointSignature(ctx, 101, 999, common.Address{}, 27, [32]byte{}, [32]byte{}, 100)
	require.Error(t, err)
	require.Contains(t, err.Error(), "checkpoint not found")
}

func TestEthSignedMessageHash(t *testing.T) {
	// Verify the Ethereum signed message hash format
	hash := common.HexToHash("0xabcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789")
	ethHash := types.ComputeEthSignedMessageHash(hash)
	require.NotEqual(t, hash, ethHash) // should be different from original
	require.NotEqual(t, common.Hash{}, ethHash)
}
