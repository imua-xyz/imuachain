package keeper

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/imua-xyz/imuachain/x/oracle/types"
)

// NOTE: Version 1 budget is a consensus constant. Later this should become an on-chain param.
const (
	xchainMaxDeliveriesPerEndBlock = 50
	xchainMaxMsgRetries            = 2
)

type xchainQueuedBatch struct {
	RootHashB64 string             `json:"root_hash_b64"`
	Batch       RawDataXChainBatch `json:"batch"`
	NextIndex   uint64             `json:"next_index"`
}

// encodeGatewayOracleReceive builds calldata for:
//
//	oracleReceive(uint32 srcChainId, uint64 nonce, bytes message)
//
// where `message` is the original LayerZero message bytes (act + args).
func encodeGatewayOracleReceive(srcChainID uint32, nonce uint64, message []byte) ([]byte, error) {
	uint32Ty, err := abi.NewType("uint32", "", nil)
	if err != nil {
		return nil, err
	}
	uint64Ty, err := abi.NewType("uint64", "", nil)
	if err != nil {
		return nil, err
	}
	bytesTy, err := abi.NewType("bytes", "", nil)
	if err != nil {
		return nil, err
	}
	args := abi.Arguments{
		{Type: uint32Ty},
		{Type: uint64Ty},
		{Type: bytesTy},
	}
	packed, err := args.Pack(srcChainID, nonce, message)
	if err != nil {
		return nil, err
	}
	methodID := crypto.Keccak256([]byte("oracleReceive(uint32,uint64,bytes)"))[:4]
	return append(methodID, packed...), nil
}

func (k Keeper) getXChainQueueHead(ctx sdk.Context, srcChainID uint64) uint64 {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.XChainQueueHeadKey(srcChainID))
	if bz == nil {
		return 0
	}
	v, err := types.BytesToUint64(bz)
	if err != nil {
		return 0
	}
	return v
}

func (k Keeper) getXChainQueueTail(ctx sdk.Context, srcChainID uint64) uint64 {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.XChainQueueTailKey(srcChainID))
	if bz == nil {
		return 0
	}
	v, err := types.BytesToUint64(bz)
	if err != nil {
		return 0
	}
	return v
}

// HasXChainQueue reports whether a queue has been created for srcChainID.
// This is primarily for observability/tests.
func (k Keeper) HasXChainQueue(ctx sdk.Context, srcChainID uint64) bool {
	return ctx.KVStore(k.storeKey).Has(types.XChainQueueHeadKey(srcChainID))
}

// GetXChainQueueHeadTail returns the current (head, tail) indices for srcChainID.
// Missing keys are treated as 0.
func (k Keeper) GetXChainQueueHeadTail(ctx sdk.Context, srcChainID uint64) (uint64, uint64) {
	return k.getXChainQueueHead(ctx, srcChainID), k.getXChainQueueTail(ctx, srcChainID)
}

func (k Keeper) setXChainQueueHead(ctx sdk.Context, srcChainID, head uint64) {
	ctx.KVStore(k.storeKey).Set(types.XChainQueueHeadKey(srcChainID), types.Uint64Bytes(head))
}

func (k Keeper) setXChainQueueTail(ctx sdk.Context, srcChainID, tail uint64) {
	ctx.KVStore(k.storeKey).Set(types.XChainQueueTailKey(srcChainID), types.Uint64Bytes(tail))
}

func (k Keeper) enqueueXChainBatch(ctx sdk.Context, srcChainID uint64, qb xchainQueuedBatch) error {
	bz, err := json.Marshal(qb)
	if err != nil {
		return err
	}

	store := ctx.KVStore(k.storeKey)
	head := k.getXChainQueueHead(ctx, srcChainID)
	tail := k.getXChainQueueTail(ctx, srcChainID)
	var reset bool
	if tail < head {
		// corrupted; reset to empty to avoid panic / infinite loop
		head, tail = 0, 0
		reset = true
	}

	// Ensure the head key exists so EndBlock can discover this srcChainID queue via prefix iteration.
	if store.Get(types.XChainQueueHeadKey(srcChainID)) == nil || reset {
		k.setXChainQueueHead(ctx, srcChainID, head)
	}

	store.Set(types.XChainQueueItemKey(srcChainID, tail), bz)
	k.setXChainQueueTail(ctx, srcChainID, tail+1)
	return nil
}

func (k Keeper) peekXChainBatch(ctx sdk.Context, srcChainID uint64) (idx uint64, qb xchainQueuedBatch, found bool, err error) {
	store := ctx.KVStore(k.storeKey)
	head := k.getXChainQueueHead(ctx, srcChainID)
	tail := k.getXChainQueueTail(ctx, srcChainID)
	if tail <= head {
		return 0, xchainQueuedBatch{}, false, nil
	}
	bz := store.Get(types.XChainQueueItemKey(srcChainID, head))
	if bz == nil {
		store.Delete(types.XChainQueueHeadKey(srcChainID))
		store.Delete(types.XChainQueueTailKey(srcChainID))
		return 0, xchainQueuedBatch{}, false, nil
	}
	var out xchainQueuedBatch
	if err := json.Unmarshal(bz, &out); err != nil {
		return 0, xchainQueuedBatch{}, false, err
	}
	return head, out, true, nil
}

func (k Keeper) setXChainBatch(ctx sdk.Context, srcChainID, idx uint64, qb xchainQueuedBatch) error {
	bz, err := json.Marshal(qb)
	if err != nil {
		return err
	}
	ctx.KVStore(k.storeKey).Set(types.XChainQueueItemKey(srcChainID, idx), bz)
	return nil
}

func (k Keeper) dequeueXChainBatch(ctx sdk.Context, srcChainID, idx uint64) {
	store := ctx.KVStore(k.storeKey)
	head := k.getXChainQueueHead(ctx, srcChainID)
	tail := k.getXChainQueueTail(ctx, srcChainID)
	if idx != head {
		// only support FIFO pop
		return
	}

	// FIFO pop: delete head item then advance head.
	store.Delete(types.XChainQueueItemKey(srcChainID, idx))

	head++
	if head >= tail {
		// reset to keep keyspace compact
		store.Delete(types.XChainQueueHeadKey(srcChainID))
		store.Delete(types.XChainQueueTailKey(srcChainID))
		return
	}
	k.setXChainQueueHead(ctx, srcChainID, head)
}

// ProcessXChainQueue drains queued batches with a fixed per-EndBlock budget.
// It is called in EndBlock and MUST be deterministic.
func (k Keeper) ProcessXChainQueue(ctx sdk.Context, srcChainID uint64) {
	delivered := 0
	for delivered < xchainMaxDeliveriesPerEndBlock {
		idx, qb, found, err := k.peekXChainBatch(ctx, srcChainID)
		if err != nil || !found {
			return
		}
		if qb.Batch.SrcChainID != srcChainID {
			// corrupted; drop to unblock
			k.dequeueXChainBatch(ctx, srcChainID, idx)
			continue
		}

		msgs := qb.Batch.Messages
		originalNextIndex := qb.NextIndex

		// Process as many messages as possible from the current head batch within the budget.
		for delivered < xchainMaxDeliveriesPerEndBlock && qb.NextIndex < uint64(len(msgs)) {
			m := msgs[qb.NextIndex]
			// invalid payload.
			if m.ID == "" || m.PayloadB64 == "" ||
				// Idempotency at execution time
				k.HasXChainMsgProcessed(ctx, srcChainID, m.ID) {
				// invalid; drop message and continue
				qb.NextIndex++
				continue
			}

			// Decode payload (kept as base64 to preserve "gateway interface unchanged").
			var payload []byte
			b, err := base64.StdEncoding.DecodeString(m.PayloadB64)
			if err != nil || len(b) == 0 {
				// invalid payload; skip this message (do not mark processed)
				qb.NextIndex++
				continue
			}
			payload = b

			// Deliver to gateway (budgeted). This call is the L0-replacement execution step.
			if err := k.deliverXChainToGateway(ctx, srcChainID, qb.Batch.BatchSeq, m.ID, m.Nonce, payload); err != nil {
				retries := k.GetXChainMsgRetryCount(ctx, srcChainID, m.ID) + 1
				if retries > xchainMaxMsgRetries {
					// Drop the message to unblock the queue.
					k.SetXChainMsgProcessed(ctx, srcChainID, m.ID)
					k.ClearXChainMsgRetryCount(ctx, srcChainID, m.ID)
					ctx.EventManager().EmitEvent(sdk.NewEvent(
						types.EventTypeXChainDelivery,
						sdk.NewAttribute(types.AttributeKeyXChainSrcChainID, strconv.FormatUint(srcChainID, 10)),
						sdk.NewAttribute(types.AttributeKeyXChainBatchSeq, strconv.FormatUint(qb.Batch.BatchSeq, 10)),
						sdk.NewAttribute(types.AttributeKeyXChainMsgID, m.ID),
						sdk.NewAttribute(types.AttributeKeyXChainRetryCount, strconv.FormatUint(retries, 10)),
						sdk.NewAttribute(types.AttributeKeyReason, "xchain_delivery_failed"),
					))
					delivered++
					qb.NextIndex++
					continue
				}
				k.SetXChainMsgRetryCount(ctx, srcChainID, m.ID, retries)
				// Persist progress so far, then stop and retry the failing message next EndBlock.
				if qb.NextIndex != originalNextIndex {
					// TODO: consider separating the nextIndex as a separate indexed value to avoid the overhead of serializing the whole batch for each message.
					_ = k.setXChainBatch(ctx, srcChainID, idx, qb)
				}
				return
			}

			k.SetXChainMsgProcessed(ctx, srcChainID, m.ID)
			k.ClearXChainMsgRetryCount(ctx, srcChainID, m.ID)
			delivered++
			qb.NextIndex++
		}

		// Batch complete: advance lastExecutedSeq and pop.
		if qb.NextIndex >= uint64(len(msgs)) {
			k.SetXChainLastExecutedSeq(ctx, srcChainID, qb.Batch.BatchSeq)
			k.dequeueXChainBatch(ctx, srcChainID, idx)
			continue
		}

		// Budget exhausted (or no progress possible): persist updated nextIndex and stop this EndBlock.
		if qb.NextIndex != originalNextIndex {
			// TODO: NestIndex as a separate indexed value to avoid the overhead of serializing the whole batch for each message.
			_ = k.setXChainBatch(ctx, srcChainID, idx, qb)
		}
		return
	}
}

func (k Keeper) deliverXChainToGateway(ctx sdk.Context, srcChainID, batchSeq uint64, msgID string, nonce uint64, payload []byte) error {
	if msgID == "" {
		return errors.New("empty msgID")
	}

	// Fallback mode (unit tests / wiring not complete): emit event only.
	if k.evmKeeper == nil {
		ctx.EventManager().EmitEvent(sdk.NewEvent(
			types.EventTypeXChainDelivery,
			sdk.NewAttribute(types.AttributeKeyXChainSrcChainID, strconv.FormatUint(srcChainID, 10)),
			sdk.NewAttribute(types.AttributeKeyXChainBatchSeq, strconv.FormatUint(batchSeq, 10)),
			sdk.NewAttribute(types.AttributeKeyXChainMsgID, msgID),
			sdk.NewAttribute(types.AttributeKeyXChainPayloadBytes, fmt.Sprintf("%d", len(payload))),
		))
		return nil
	}

	gateways, err := k.assetsKeeper.GetGatewayAddresses(ctx)
	if err != nil {
		return fmt.Errorf("failed to get gateway addresses: %w", err)
	}
	if len(gateways) == 0 {
		return fmt.Errorf("no gateway configured for xchain delivery (srcChainID=%d)", srcChainID)
	}
	to := gateways[0]

	// Use a deterministic module account address as the EVM sender.
	from := common.BytesToAddress(authtypes.NewModuleAddress(types.ModuleName))

	// Call the gateway's oracle receiver entrypoint (to mimic LayerZero inbound execution).
	//
	// Required gateway addition (imua-contracts):
	//   function oracleReceive(uint32 srcChainId, uint64 nonce, bytes calldata message) external;
	//
	// Where `message` is the original LZ payload (act + args).
	// srcChainID is a LayerZero Endpoint ID (EID), not a native chain ID.
	// LayerZero V2 EIDs are uint32 values (30xxx for mainnet, 40xxx for testnet).
	calldata, err := encodeGatewayOracleReceive(uint32(srcChainID), nonce, payload)
	if err != nil {
		return fmt.Errorf("failed to encode gateway oracleReceive calldata: %w", err)
	}

	// TODO: set an estimate of gas cost as an upper bound.
	const gasLimit = uint64(3_000_000)
	msg := ethtypes.NewMessage(
		from, &to,
		0,             // nonce (not used for calls)
		big.NewInt(0), // value
		gasLimit,
		big.NewInt(0), // gasPrice (fee deduction is not performed in ApplyMessage)
		nil, nil,
		calldata,
		nil,
		true,
	)

	rsp, err := k.evmKeeper.ApplyMessage(ctx, msg, nil, true)
	if err != nil {
		return fmt.Errorf("gateway call failed: %w", err)
	}
	if rsp != nil && rsp.Failed() {
		return fmt.Errorf("gateway call vm failed: %s", rsp.VmError)
	}

	// Parse outbound events (OutboundResponse / OutboundMessage) from EVM logs
	// and enqueue them for relay to the destination client chain.
	if rsp != nil && len(rsp.Logs) > 0 {
		k.parseAndEnqueueOutbound(ctx, rsp.Logs, srcChainID)
	}

	// Observability: emit event on successful delivery (payload bytes only).
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeXChainDelivery,
		sdk.NewAttribute(types.AttributeKeyXChainSrcChainID, strconv.FormatUint(srcChainID, 10)),
		sdk.NewAttribute(types.AttributeKeyXChainBatchSeq, strconv.FormatUint(batchSeq, 10)),
		sdk.NewAttribute(types.AttributeKeyXChainMsgID, msgID),
		sdk.NewAttribute(types.AttributeKeyXChainPayloadBytes, fmt.Sprintf("%d", len(payload))),
	))
	return nil
}
