package keeper_test

import (
	"bytes"
	"encoding/hex"
	"reflect"
	"sort"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	keytypes "github.com/imua-xyz/imuachain/types/keys"
	keepertest "github.com/imua-xyz/imuachain/testutil/keeper"
	dogfoodkeeper "github.com/imua-xyz/imuachain/x/dogfood/keeper"
	dogfoodtypes "github.com/imua-xyz/imuachain/x/dogfood/types"
	"github.com/imua-xyz/imuachain/x/oracle/keeper"
	"github.com/imua-xyz/imuachain/x/oracle/types"
	"github.com/stretchr/testify/require"
)

// patchDogfoodValidators stubs GetAllImuachainValidators against the zero-value
// dogfood keeper in keepertest.OracleKeeper. Caller defers the returned Reset.
func patchDogfoodValidators(vals []dogfoodtypes.ImuachainValidator) *gomonkey.Patches {
	return gomonkey.ApplyMethod(
		reflect.TypeOf(dogfoodkeeper.Keeper{}),
		"GetAllImuachainValidators",
		func(_ dogfoodkeeper.Keeper, _ sdk.Context) []dogfoodtypes.ImuachainValidator {
			return vals
		},
	)
}

// stubOperatorKeeper satisfies types.OperatorKeeper for tests by mapping
// consensus addresses to operator AccAddresses via an in-memory table.
type stubOperatorKeeper struct {
	consToOp map[string]sdk.AccAddress
}

func (s *stubOperatorKeeper) GetOperatorConsKeyForChainID(_ sdk.Context, _ sdk.AccAddress, _ string) (bool, keytypes.WrappedConsKey, error) {
	return false, nil, nil
}

func (s *stubOperatorKeeper) GetOperatorAddressForChainIDAndConsAddr(_ sdk.Context, _ string, consAddr sdk.ConsAddress) (bool, sdk.AccAddress) {
	if a, ok := s.consToOp[consAddr.String()]; ok {
		return true, a
	}
	return false, nil
}

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

	// Stub a 3-validator set (total=300). One sig (power=100) stays under the
	// strict 2/3 threshold so the checkpoint remains pending for idempotency assertion.
	patcher := patchDogfoodValidators([]dogfoodtypes.ImuachainValidator{
		{Address: evmAddr.Bytes(), Power: 100},
		{Address: []byte{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff, 0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb, 0xcc, 0xdd}, Power: 100},
		{Address: []byte{0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa, 0x99, 0x88, 0x77, 0x66, 0x55, 0x44, 0x33, 0x22, 0x11, 0x00, 0xff, 0xee, 0xdd, 0xcc}, Power: 100},
	})
	defer patcher.Reset()

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

	finalized, err := k.AddCheckpointSignature(ctx, 101, 1, evmAddr, v, r, s, 100)
	require.NoError(t, err)
	require.False(t, finalized, "single sig of 100/300 power must not cross strict 2/3")

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

// TestValsetCheckpointAddressNamespace asserts that the valset checkpoint stores
// **operator EVM addresses** (matching the namespace that ecrecover yields on
// signed checkpoints), not raw dogfood consensus addresses. This is the bug that
// would otherwise prevent BridgeVerifier on the destination chain from
// associating signatures with valset entries.
func TestValsetCheckpointAddressNamespace(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	// Build 3 validators with DISTINCT consensus addresses and operator EVM
	// addresses, so we can detect any reuse of the wrong namespace.
	type val struct {
		cons sdk.ConsAddress
		op   sdk.AccAddress
		pow  int64
	}
	vals := []val{
		{cons: sdk.ConsAddress(bytes.Repeat([]byte{0x11}, 20)), op: sdk.AccAddress(common.HexToAddress("0xaa00000000000000000000000000000000000001").Bytes()), pow: 100},
		{cons: sdk.ConsAddress(bytes.Repeat([]byte{0x22}, 20)), op: sdk.AccAddress(common.HexToAddress("0xcc00000000000000000000000000000000000002").Bytes()), pow: 200},
		{cons: sdk.ConsAddress(bytes.Repeat([]byte{0x33}, 20)), op: sdk.AccAddress(common.HexToAddress("0xbb00000000000000000000000000000000000003").Bytes()), pow: 150},
	}
	dogVals := make([]dogfoodtypes.ImuachainValidator, len(vals))
	consToOp := make(map[string]sdk.AccAddress, len(vals))
	for i, v := range vals {
		dogVals[i] = dogfoodtypes.ImuachainValidator{Address: v.cons.Bytes(), Power: v.pow}
		consToOp[v.cons.String()] = v.op
	}
	patcher := patchDogfoodValidators(dogVals)
	defer patcher.Reset()

	k.SetOperatorKeeper(&stubOperatorKeeper{consToOp: consToOp})

	// Run the under-test routine.
	k.CreateValidatorSetCheckpointIfChanged(ctx)

	nonce := uint64(1)
	cp, found := k.GetValsetCheckpoint(ctx, nonce)
	require.True(t, found, "valset checkpoint should be created on first call")
	require.Equal(t, nonce, cp.Nonce)
	require.Len(t, cp.Validators, 3)
	require.Len(t, cp.Powers, 3)
	require.EqualValues(t, 450, cp.TotalPower)

	// Assert the stored addresses are operator EVM addresses, NOT consensus addrs,
	// sorted strictly ascending.
	expected := []common.Address{
		common.HexToAddress("0xaa00000000000000000000000000000000000001"),
		common.HexToAddress("0xbb00000000000000000000000000000000000003"),
		common.HexToAddress("0xcc00000000000000000000000000000000000002"),
	}
	require.True(t, sort.SliceIsSorted(cp.Validators, func(i, j int) bool {
		return bytes.Compare(cp.Validators[i].Bytes(), cp.Validators[j].Bytes()) < 0
	}), "validators must be sorted ascending for BridgeVerifier")
	require.Equal(t, expected, cp.Validators)

	// Powers must follow the address permutation.
	require.Equal(t, []int64{100, 150, 200}, cp.Powers)

	// None of the consensus addresses should appear in the stored set —
	// regression guard against the prior bug.
	for _, v := range vals {
		consAsAddr := common.BytesToAddress(v.cons.Bytes())
		for _, stored := range cp.Validators {
			require.NotEqual(t, consAsAddr, stored,
				"consensus address %s leaked into valset (namespace bug)", consAsAddr.Hex())
		}
	}
}

// TestValsetCheckpointIfChangedSkipsOnNoChange asserts that calling
// CreateValidatorSetCheckpointIfChanged twice with the same valset produces a
// single checkpoint (the second call is a no-op).
func TestValsetCheckpointIfChangedSkipsOnNoChange(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	cons := sdk.ConsAddress(bytes.Repeat([]byte{0x11}, 20))
	op := sdk.AccAddress(common.HexToAddress("0xaa00000000000000000000000000000000000001").Bytes())

	patcher := patchDogfoodValidators([]dogfoodtypes.ImuachainValidator{
		{Address: cons.Bytes(), Power: 100},
	})
	defer patcher.Reset()
	k.SetOperatorKeeper(&stubOperatorKeeper{consToOp: map[string]sdk.AccAddress{cons.String(): op}})

	k.CreateValidatorSetCheckpointIfChanged(ctx)
	_, found := k.GetValsetCheckpoint(ctx, 1)
	require.True(t, found)

	// Second call: identical valset → no new checkpoint.
	k.CreateValidatorSetCheckpointIfChanged(ctx)
	_, found = k.GetValsetCheckpoint(ctx, 2)
	require.False(t, found, "identical valset must not produce a new checkpoint")
}

// TestValsetCheckpointSkipsWhenOperatorBindingMissing asserts that validators
// without an operator binding for this chain are skipped (logged, not stored),
// preventing leakage of unmappable identities into the destination contract.
func TestValsetCheckpointSkipsWhenOperatorBindingMissing(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	cons1 := sdk.ConsAddress(bytes.Repeat([]byte{0x11}, 20))
	op1 := sdk.AccAddress(common.HexToAddress("0xaa00000000000000000000000000000000000001").Bytes())
	cons2 := sdk.ConsAddress(bytes.Repeat([]byte{0x22}, 20))
	// cons2 deliberately has no mapping in the stub.

	patcher := patchDogfoodValidators([]dogfoodtypes.ImuachainValidator{
		{Address: cons1.Bytes(), Power: 100},
		{Address: cons2.Bytes(), Power: 200},
	})
	defer patcher.Reset()
	k.SetOperatorKeeper(&stubOperatorKeeper{consToOp: map[string]sdk.AccAddress{cons1.String(): op1}})

	k.CreateValidatorSetCheckpointIfChanged(ctx)
	cp, found := k.GetValsetCheckpoint(ctx, 1)
	require.True(t, found)
	require.Len(t, cp.Validators, 1, "unmapped validator must be skipped")
	require.EqualValues(t, 100, cp.TotalPower)
}
