package keeper

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/imua-xyz/imuachain/x/oracle/keeper/common"
	"github.com/imua-xyz/imuachain/x/oracle/types"
)

// RawDataXChainBatch is the rawData payload schema for cross-chain message batches.
// It is intentionally JSON to avoid proto regeneration while bootstrapping the cross-chain pipeline.
type RawDataXChainBatch struct {
	SrcChainID uint64             `json:"src_chain_id"`
	BatchSeq   uint64             `json:"batch_seq"`
	Messages   []RawDataXChainMsg `json:"messages"`
}

type RawDataXChainMsg struct {
	// ID must be globally unique for replay protection (txHash:logIndex).
	ID string `json:"id"`
	// Nonce is the LayerZero nonce on the source chain (lzNonce).
	// It is used by the gateway to maintain its inbound nonce tracking.
	Nonce uint64 `json:"nonce"`
	// Type is an application-level discriminator (e.g. "evm").
	Type string `json:"type"`
	// PayloadB64 carries the LZ message bytes (act + args). The handler only validates base64.
	PayloadB64 string `json:"payload_b64"`
}

var _ common.PostAggregationHandler = UpdateXChainMsgs

// UpdateXChainMsgs is a post-aggregation handler for oracle 2-phases "cross-chain message batch" feeders.
//
// Semantics (Version 1):
// - Enforces strict batch sequencing per srcChainID (batch_seq must equal lastAcceptedSeq+1).
// - Validates message IDs and payload encoding.
// - Enqueues the batch for budgeted EndBlock delivery (gateway delivery is executed later).
//
// NOTE: postHandler errors are only logged (they do not revert the block).
// We additionally recover panics here (e.g. from corrupted xchain_store decode
// in GetXChainLastSeq) and convert them to returned errors. Letting them
// propagate would halt consensus because the postHandler is invoked inside
// FeederManager.commitRounds without panic recovery upstream.
func UpdateXChainMsgs(
	ctx sdk.Context,
	rootHash []byte,
	rawData []byte,
	feederID, roundID uint64,
	kInf common.KeeperOracle,
) (retErr error) {
	defer func() {
		if r := recover(); r != nil {
			ctx.Logger().Error("UpdateXChainMsgs panic recovered — batch dropped, will retry next round",
				"feederID", feederID, "roundID", roundID, "panic", r)
			retErr = fmt.Errorf("xchain post-aggregation panic: %v", r)
		}
	}()
	// Queue length cap enforced in enqueueXChainBatch via xchainMaxPendingBatchesPerSrcChain.
	k, ok := kInf.(*Keeper)
	if !ok {
		return errors.New("input keeper interface type error")
	}

	var batch RawDataXChainBatch
	if err := json.Unmarshal(rawData, &batch); err != nil {
		return fmt.Errorf("invalid xchain rawData json: %w", err)
	}
	if batch.SrcChainID == 0 {
		return errors.New("invalid xchain rawData: src_chain_id must be > 0")
	}
	if batch.BatchSeq == 0 {
		return errors.New("invalid xchain rawData: batch_seq must be > 0")
	}

	lastSeq, found := k.GetXChainLastSeq(ctx, batch.SrcChainID)
	if !found {
		lastSeq = 0
	}
	// Idempotency: ignore already-processed (or older) batches.
	if batch.BatchSeq <= lastSeq {
		return nil
	}
	expected := lastSeq + 1
	if batch.BatchSeq != expected {
		return fmt.Errorf("xchain batch seq mismatch: srcChainID:%d expected:%d got:%d", batch.SrcChainID, expected, batch.BatchSeq)
	}

	// Validate and de-duplicate message IDs (stable: first occurrence wins).
	unique := make(map[string]struct{}, len(batch.Messages))
	msgs := make([]RawDataXChainMsg, 0, len(batch.Messages))
	for _, m := range batch.Messages {
		if m.ID == "" {
			return errors.New("invalid xchain message: empty id")
		}
		if _, seen := unique[m.ID]; seen {
			continue
		}
		unique[m.ID] = struct{}{}

		// Validate payload encoding (execution happens later in EndBlock queue consumer).
		if m.PayloadB64 == "" {
			return fmt.Errorf("invalid xchain message: empty payload_b64 for id:%s", m.ID)
		}
		if _, err := base64.StdEncoding.DecodeString(m.PayloadB64); err != nil {
			return fmt.Errorf("invalid xchain message payload_b64 for id:%s: %w", m.ID, err)
		}
		msgs = append(msgs, m)
	}

	batch.Messages = msgs

	if err := k.enqueueXChainBatch(ctx, batch.SrcChainID, xchainQueuedBatch{
		RootHashB64: base64.StdEncoding.EncodeToString(rootHash),
		Batch:       batch,
		NextIndex:   0,
	}); err != nil {
		return fmt.Errorf("failed to enqueue xchain batch: %w", err)
	}
	k.SetXChainLastSeq(ctx, batch.SrcChainID, batch.BatchSeq)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeXChainBatch,
		sdk.NewAttribute(types.AttributeKeyFeederID, strconv.FormatUint(feederID, 10)),
		sdk.NewAttribute(types.AttributeKeyRoundID, strconv.FormatUint(roundID, 10)),
		sdk.NewAttribute(types.AttributeKeyXChainSrcChainID, strconv.FormatUint(batch.SrcChainID, 10)),
		sdk.NewAttribute(types.AttributeKeyXChainBatchSeq, strconv.FormatUint(batch.BatchSeq, 10)),
		sdk.NewAttribute(types.AttributeKeyXChainMsgCount, strconv.Itoa(len(batch.Messages))),
		sdk.NewAttribute(types.AttributeKeyXChainRootHash, base64.StdEncoding.EncodeToString(rootHash)),
	))

	return nil
}
