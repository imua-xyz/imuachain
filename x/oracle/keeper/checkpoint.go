package keeper

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/imua-xyz/imuachain/x/oracle/types"
)

// --- Checkpoint nonce management ---

func (k Keeper) getCheckpointNonce(ctx sdk.Context, dstChainID uint64) uint64 {
	bz := ctx.KVStore(k.storeKey).Get(types.CheckpointNonceKey(dstChainID))
	if bz == nil {
		return 0
	}
	v, _ := types.BytesToUint64(bz)
	return v
}

func (k Keeper) setCheckpointNonce(ctx sdk.Context, dstChainID, nonce uint64) {
	ctx.KVStore(k.storeKey).Set(types.CheckpointNonceKey(dstChainID), types.Uint64Bytes(nonce))
}

// --- Checkpoint CRUD ---

func (k Keeper) setCheckpoint(ctx sdk.Context, dstChainID, nonce uint64, cp types.OutboundCheckpoint) error {
	bz, err := json.Marshal(cp)
	if err != nil {
		return err
	}
	ctx.KVStore(k.storeKey).Set(types.CheckpointDataKey(dstChainID, nonce), bz)
	return nil
}

// GetCheckpoint returns the checkpoint for a given dstChainID and nonce.
func (k Keeper) GetCheckpoint(ctx sdk.Context, dstChainID, nonce uint64) (types.OutboundCheckpoint, bool) {
	bz := ctx.KVStore(k.storeKey).Get(types.CheckpointDataKey(dstChainID, nonce))
	if bz == nil {
		return types.OutboundCheckpoint{}, false
	}
	var cp types.OutboundCheckpoint
	if err := json.Unmarshal(bz, &cp); err != nil {
		return types.OutboundCheckpoint{}, false
	}
	return cp, true
}

// GetLatestCheckpointNonce returns the latest checkpoint nonce for a destination chain.
func (k Keeper) GetLatestCheckpointNonce(ctx sdk.Context, dstChainID uint64) uint64 {
	return k.getCheckpointNonce(ctx, dstChainID)
}

// --- Checkpoint creation (called from EndBlock) ---

// CreateCheckpointForPendingOutbound creates a new checkpoint for any un-checkpointed
// outbound messages. Returns true if a new checkpoint was created.
func (k Keeper) CreateCheckpointForPendingOutbound(ctx sdk.Context, dstChainID uint64) bool {
	head := k.getOutboundHead(ctx, dstChainID)
	tail := k.getOutboundTail(ctx, dstChainID)
	if tail <= head {
		return false
	}

	// Collect all pending outbound message payloads
	store := ctx.KVStore(k.storeKey)
	var payloads [][]byte
	var seqStart, seqEnd uint64
	for idx := head; idx < tail; idx++ {
		bz := store.Get(types.OutboundItemKey(dstChainID, idx))
		if bz == nil {
			continue
		}
		var msg OutboundMsg
		if err := json.Unmarshal(bz, &msg); err != nil {
			continue
		}
		payload, err := hex.DecodeString(msg.PayloadHex)
		if err != nil {
			continue
		}
		if seqStart == 0 || msg.SeqNum < seqStart {
			seqStart = msg.SeqNum
		}
		if msg.SeqNum > seqEnd {
			seqEnd = msg.SeqNum
		}
		payloads = append(payloads, payload)
	}

	if len(payloads) == 0 {
		return false
	}

	// Check if a pending (non-finalized) checkpoint already exists for this range
	currentNonce := k.getCheckpointNonce(ctx, dstChainID)
	if currentNonce > 0 {
		existing, found := k.GetCheckpoint(ctx, dstChainID, currentNonce)
		if found && !existing.Finalized {
			// There's already a pending checkpoint, don't create another
			return false
		}
	}

	nonce := currentNonce + 1
	messagesHash := types.ComputeMessagesHash(payloads)

	cp := types.OutboundCheckpoint{
		Nonce:        nonce,
		DstChainID:   dstChainID,
		MessagesHash: messagesHash,
		SeqStart:     seqStart,
		SeqEnd:       seqEnd,
		Height:       ctx.BlockHeight(),
		Finalized:    false,
	}

	if err := k.setCheckpoint(ctx, dstChainID, nonce, cp); err != nil {
		ctx.Logger().Error("failed to create checkpoint", "err", err)
		return false
	}
	k.setCheckpointNonce(ctx, dstChainID, nonce)

	checkpointHash := types.ComputeCheckpointHash(nonce, dstChainID, messagesHash)
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeCheckpointCreated,
		sdk.NewAttribute(types.AttributeKeyOutboundDstChainID, fmt.Sprintf("%d", dstChainID)),
		sdk.NewAttribute(types.AttributeKeyCheckpointNonce, fmt.Sprintf("%d", nonce)),
		sdk.NewAttribute(types.AttributeKeyCheckpointHash, checkpointHash.Hex()),
		sdk.NewAttribute(types.AttributeKeyCheckpointSeqRange, fmt.Sprintf("%d-%d", seqStart, seqEnd)),
	))

	return true
}

// --- Signature submission and finalization ---

// AddCheckpointSignature adds a validator's ECDSA signature to a checkpoint.
// Returns (finalized, error).
func (k Keeper) AddCheckpointSignature(
	ctx sdk.Context,
	dstChainID, nonce uint64,
	validatorAddr common.Address,
	v uint8, r, s [32]byte,
	validatorPower int64,
) (bool, error) {
	cp, found := k.GetCheckpoint(ctx, dstChainID, nonce)
	if !found {
		return false, fmt.Errorf("checkpoint not found: dstChainID=%d nonce=%d", dstChainID, nonce)
	}
	if cp.Finalized {
		return true, nil // already finalized, no-op
	}

	// Verify the ECDSA signature
	checkpointHash := types.ComputeCheckpointHash(nonce, dstChainID, cp.MessagesHash)
	ethSignedHash := types.ComputeEthSignedMessageHash(checkpointHash)

	// Reconstruct the 65-byte signature [R || S || V]
	sig := make([]byte, 65)
	copy(sig[0:32], r[:])
	copy(sig[32:64], s[:])
	sig[64] = v - 27 // crypto.Ecrecover expects V in {0,1}

	recoveredPub, err := crypto.Ecrecover(ethSignedHash.Bytes(), sig)
	if err != nil {
		return false, fmt.Errorf("signature recovery failed: %w", err)
	}

	pubKey, err := crypto.UnmarshalPubkey(recoveredPub)
	if err != nil {
		return false, fmt.Errorf("unmarshal pubkey failed: %w", err)
	}

	recoveredAddr := crypto.PubkeyToAddress(*pubKey)
	if recoveredAddr != validatorAddr {
		return false, fmt.Errorf("signature mismatch: expected %s got %s", validatorAddr.Hex(), recoveredAddr.Hex())
	}

	// Store signature
	sigData := types.CheckpointSignature{
		Validator: validatorAddr,
		V:         v,
		R:         r,
		S:         s,
		Power:     validatorPower,
	}
	sigBz, err := json.Marshal(sigData)
	if err != nil {
		return false, err
	}
	store := ctx.KVStore(k.storeKey)

	// Check if this validator already signed (idempotent)
	sigKey := types.CheckpointSigKey(dstChainID, nonce, validatorAddr.Bytes())
	if store.Has(sigKey) {
		return cp.Finalized, nil
	}

	store.Set(sigKey, sigBz)

	// Update accumulated signed power
	signedPower := k.getCheckpointSignedPower(ctx, dstChainID, nonce)
	signedPower += validatorPower
	k.setCheckpointSignedPower(ctx, dstChainID, nonce, signedPower)

	// Guard against totalPower<=0 (empty valset): the strict 2/3 inequality
	// would otherwise degenerate to `signedPower*3 > 0` and finalize on a
	// single signature.
	totalPower := k.getTotalValidatorPower(ctx)
	if totalPower <= 0 {
		ctx.Logger().Error("checkpoint signature accepted but totalPower<=0",
			"dstChainID", dstChainID, "nonce", nonce, "signedPower", signedPower)
		return false, nil
	}
	if signedPower*3 > totalPower*2 {
		cp.Finalized = true
		_ = k.setCheckpoint(ctx, dstChainID, nonce, cp)

		ctx.EventManager().EmitEvent(sdk.NewEvent(
			types.EventTypeCheckpointFinalized,
			sdk.NewAttribute(types.AttributeKeyOutboundDstChainID, fmt.Sprintf("%d", dstChainID)),
			sdk.NewAttribute(types.AttributeKeyCheckpointNonce, fmt.Sprintf("%d", nonce)),
		))
		return true, nil
	}

	return false, nil
}

// GetCheckpointSignatures returns all signatures for a checkpoint.
func (k Keeper) GetCheckpointSignatures(ctx sdk.Context, dstChainID, nonce uint64) []types.CheckpointSignature {
	store := ctx.KVStore(k.storeKey)
	prefix := types.CheckpointSigKey(dstChainID, nonce, nil)
	it := sdk.KVStorePrefixIterator(store, prefix)
	defer it.Close()

	var sigs []types.CheckpointSignature
	for ; it.Valid(); it.Next() {
		var sig types.CheckpointSignature
		if err := json.Unmarshal(it.Value(), &sig); err != nil {
			continue
		}
		sigs = append(sigs, sig)
	}
	return sigs
}

// GetMinimalCheckpointSignatures returns the smallest set of signatures whose
// cumulative power exceeds 2/3 of totalPower. Signatures are selected by
// descending power (greedy), then sorted by ascending address (required by
// BridgeVerifier). This minimizes the number of ecrecover calls on Ethereum.
func (k Keeper) GetMinimalCheckpointSignatures(ctx sdk.Context, dstChainID, nonce uint64) []types.CheckpointSignature {
	all := k.GetCheckpointSignatures(ctx, dstChainID, nonce)
	if len(all) == 0 {
		return nil
	}

	totalPower := k.getTotalValidatorPower(ctx)
	if totalPower <= 0 {
		return all // can't determine threshold, return all
	}
	required := totalPower*2/3 + 1

	// Sort by power descending (greedy selection)
	sort.Slice(all, func(i, j int) bool {
		return all[i].Power > all[j].Power
	})

	// Pick just enough to exceed 2/3
	selected := make([]types.CheckpointSignature, 0, len(all))
	var accumulated int64
	for _, sig := range all {
		selected = append(selected, sig)
		accumulated += sig.Power
		if accumulated >= required {
			break
		}
	}

	// Sort selected by address ascending (BridgeVerifier requirement)
	sort.Slice(selected, func(i, j int) bool {
		return bytes.Compare(selected[i].Validator.Bytes(), selected[j].Validator.Bytes()) < 0
	})

	return selected
}

// --- Power tracking helpers ---

func (k Keeper) getCheckpointSignedPower(ctx sdk.Context, dstChainID, nonce uint64) int64 {
	bz := ctx.KVStore(k.storeKey).Get(types.CheckpointSignedPowerKey(dstChainID, nonce))
	if bz == nil {
		return 0
	}
	v, _ := types.BytesToUint64(bz)
	return int64(v)
}

func (k Keeper) setCheckpointSignedPower(ctx sdk.Context, dstChainID, nonce uint64, power int64) {
	ctx.KVStore(k.storeKey).Set(types.CheckpointSignedPowerKey(dstChainID, nonce), types.Uint64Bytes(uint64(power)))
}

func (k Keeper) getTotalValidatorPower(ctx sdk.Context) int64 {
	validators := k.GetAllImuachainValidators(ctx)
	var total int64
	for _, v := range validators {
		total += v.Power
	}
	return total
}

// --- Validator Set Checkpoints ---

// CreateValidatorSetCheckpointIfChanged creates a new valset checkpoint when the
// active validator set differs from the previously stored one.
//
// Address namespace: validators must be identified by the **operator EVM address**
// — the same namespace that ecrecover yields when validators sign checkpoints.
// The dogfood consensus address is in a different namespace and would break
// destination-chain signature lookup. We map via operatorKeeper.
//
// Output ordering: validators are sorted strictly ascending by EVM address to
// match BridgeVerifier._validateAndIngestValset on the destination side.
//
// Called from EpochsHooksWrapper.AfterEpochEnd (EndBlock chain). Recover is
// load-bearing here — a panic would halt consensus via the epochs hook.
func (k Keeper) CreateValidatorSetCheckpointIfChanged(ctx sdk.Context) {
	defer func() {
		if r := recover(); r != nil {
			ctx.Logger().Error("valset checkpoint creation panic recovered", "error", r)
		}
	}()

	validators := k.GetAllImuachainValidators(ctx)
	if len(validators) == 0 {
		return
	}

	if k.operatorKeeper == nil {
		// Without operator binding we cannot produce the correct address namespace.
		// Skip rather than write a checkpoint that destination contracts can't verify against.
		ctx.Logger().Info("valset checkpoint skipped: operatorKeeper not wired")
		return
	}

	chainID := ctx.ChainID()
	type valEntry struct {
		addr  common.Address
		power int64
	}
	entries := make([]valEntry, 0, len(validators))
	for _, v := range validators {
		if v.Power <= 0 {
			continue
		}
		consAddr := sdk.ConsAddress(v.Address)
		found, opAccAddr := k.operatorKeeper.GetOperatorAddressForChainIDAndConsAddr(ctx, chainID, consAddr)
		if !found {
			ctx.Logger().Info("valset checkpoint: validator has no operator binding for this chain",
				"consAddr", consAddr.String(), "chainID", chainID)
			continue
		}
		entries = append(entries, valEntry{
			addr:  common.BytesToAddress(opAccAddr.Bytes()),
			power: v.Power,
		})
	}
	if len(entries) == 0 {
		return
	}

	// BridgeVerifier requires strictly ascending addresses.
	sort.Slice(entries, func(i, j int) bool {
		return bytes.Compare(entries[i].addr.Bytes(), entries[j].addr.Bytes()) < 0
	})

	addrs := make([]common.Address, len(entries))
	powers := make([]int64, len(entries))
	var totalPower int64
	for i, e := range entries {
		addrs[i] = e.addr
		powers[i] = e.power
		totalPower += e.power
	}

	store := ctx.KVStore(k.storeKey)
	nonceBz := store.Get([]byte(types.ValsetCheckpointNonceKey))
	var currentNonce uint64
	if nonceBz != nil {
		currentNonce, _ = types.BytesToUint64(nonceBz)
	}

	// Honor "IfChanged": skip when previous valset matches.
	if currentNonce > 0 {
		if prev, ok := k.GetValsetCheckpoint(ctx, currentNonce); ok && valsetEqual(prev, addrs, powers) {
			return
		}
	}

	newNonce := currentNonce + 1
	cp := types.ValidatorSetCheckpoint{
		Nonce:      newNonce,
		Validators: addrs,
		Powers:     powers,
		TotalPower: totalPower,
		Height:     ctx.BlockHeight(),
		Finalized:  false,
	}

	cpBz, err := json.Marshal(cp)
	if err != nil {
		ctx.Logger().Error("failed to marshal valset checkpoint", "error", err)
		return
	}

	store.Set(types.ValsetCheckpointDataKey(newNonce), cpBz)
	store.Set([]byte(types.ValsetCheckpointNonceKey), types.Uint64Bytes(newNonce))

	valsetHash := types.ComputeValsetCheckpointHash(newNonce, addrs, powers)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeCheckpointCreated,
		sdk.NewAttribute("type", "valset"),
		sdk.NewAttribute(types.AttributeKeyCheckpointNonce, fmt.Sprintf("%d", newNonce)),
		sdk.NewAttribute(types.AttributeKeyCheckpointHash, valsetHash.Hex()),
		sdk.NewAttribute("validator_count", fmt.Sprintf("%d", len(addrs))),
		sdk.NewAttribute("total_power", fmt.Sprintf("%d", totalPower)),
	))
}

func valsetEqual(prev types.ValidatorSetCheckpoint, addrs []common.Address, powers []int64) bool {
	if len(prev.Validators) != len(addrs) || len(prev.Powers) != len(powers) {
		return false
	}
	for i := range addrs {
		if prev.Validators[i] != addrs[i] || prev.Powers[i] != powers[i] {
			return false
		}
	}
	return true
}

// GetValsetCheckpoint returns a validator set checkpoint by nonce.
func (k Keeper) GetValsetCheckpoint(ctx sdk.Context, nonce uint64) (types.ValidatorSetCheckpoint, bool) {
	bz := ctx.KVStore(k.storeKey).Get(types.ValsetCheckpointDataKey(nonce))
	if bz == nil {
		return types.ValidatorSetCheckpoint{}, false
	}
	var cp types.ValidatorSetCheckpoint
	if err := json.Unmarshal(bz, &cp); err != nil {
		return types.ValidatorSetCheckpoint{}, false
	}
	return cp, true
}
