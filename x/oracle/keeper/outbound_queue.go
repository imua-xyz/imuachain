package keeper

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	evmtypes "github.com/evmos/evmos/v16/x/evm/types"
	"github.com/imua-xyz/imuachain/x/oracle/types"
)

// OutboundMsg represents a message queued for relay from imuachain to a client chain.
type OutboundMsg struct {
	DstChainID uint64 `json:"dst_chain_id"`
	SeqNum     uint64 `json:"seq_num"`
	Nonce      uint64 `json:"nonce"`       // inbound nonce that triggered this response
	PayloadHex string `json:"payload_hex"` // hex-encoded response payload (action + args)
	Height     int64  `json:"height"`      // block height when created
}

// Event signatures for outbound messages emitted by the gateway contract.
var (
	// OutboundResponse(uint32 indexed dstChainId, uint64 indexed requestNonce, bytes payload)
	outboundResponseEventID = crypto.Keccak256Hash([]byte("OutboundResponse(uint32,uint64,bytes)"))
	// OutboundMessage(uint32 indexed dstChainId, bytes payload)
	outboundMessageEventID = crypto.Keccak256Hash([]byte("OutboundMessage(uint32,bytes)"))
)

// --- Outbound sequence ---

func (k Keeper) getOutboundNextSeq(ctx sdk.Context, dstChainID uint64) uint64 {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.OutboundSeqKey(dstChainID))
	if bz == nil {
		return 1
	}
	v, err := types.BytesToUint64(bz)
	if err != nil {
		return 1
	}
	return v
}

func (k Keeper) setOutboundNextSeq(ctx sdk.Context, dstChainID, seq uint64) {
	ctx.KVStore(k.storeKey).Set(types.OutboundSeqKey(dstChainID), types.Uint64Bytes(seq))
}

// --- Outbound queue CRUD ---

func (k Keeper) getOutboundHead(ctx sdk.Context, dstChainID uint64) uint64 {
	bz := ctx.KVStore(k.storeKey).Get(types.OutboundHeadKey(dstChainID))
	if bz == nil {
		return 0
	}
	v, _ := types.BytesToUint64(bz)
	return v
}

func (k Keeper) getOutboundTail(ctx sdk.Context, dstChainID uint64) uint64 {
	bz := ctx.KVStore(k.storeKey).Get(types.OutboundTailKey(dstChainID))
	if bz == nil {
		return 0
	}
	v, _ := types.BytesToUint64(bz)
	return v
}

func (k Keeper) setOutboundHead(ctx sdk.Context, dstChainID, head uint64) {
	ctx.KVStore(k.storeKey).Set(types.OutboundHeadKey(dstChainID), types.Uint64Bytes(head))
}

func (k Keeper) setOutboundTail(ctx sdk.Context, dstChainID, tail uint64) {
	ctx.KVStore(k.storeKey).Set(types.OutboundTailKey(dstChainID), types.Uint64Bytes(tail))
}

func (k Keeper) enqueueOutbound(ctx sdk.Context, msg OutboundMsg) error {
	bz, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	store := ctx.KVStore(k.storeKey)
	dstChainID := msg.DstChainID
	head := k.getOutboundHead(ctx, dstChainID)
	tail := k.getOutboundTail(ctx, dstChainID)
	if tail < head {
		head, tail = 0, 0
	}

	if store.Get(types.OutboundHeadKey(dstChainID)) == nil {
		k.setOutboundHead(ctx, dstChainID, head)
	}

	store.Set(types.OutboundItemKey(dstChainID, tail), bz)
	k.setOutboundTail(ctx, dstChainID, tail+1)
	return nil
}

// DequeueOutbound removes the head item from the outbound queue.
func (k Keeper) DequeueOutbound(ctx sdk.Context, dstChainID, idx uint64) {
	store := ctx.KVStore(k.storeKey)
	head := k.getOutboundHead(ctx, dstChainID)
	tail := k.getOutboundTail(ctx, dstChainID)
	if idx != head {
		return
	}
	store.Delete(types.OutboundItemKey(dstChainID, idx))
	head++
	if head >= tail {
		store.Delete(types.OutboundHeadKey(dstChainID))
		store.Delete(types.OutboundTailKey(dstChainID))
		return
	}
	k.setOutboundHead(ctx, dstChainID, head)
}

// GetOutboundMessages returns pending outbound messages for a destination chain,
// starting after afterSeq, up to limit items. Used by gRPC query for relayer polling.
func (k Keeper) GetOutboundMessages(ctx sdk.Context, dstChainID uint64, afterSeq uint64, limit uint64) []OutboundMsg {
	if limit == 0 {
		limit = 100
	}

	store := ctx.KVStore(k.storeKey)
	head := k.getOutboundHead(ctx, dstChainID)
	tail := k.getOutboundTail(ctx, dstChainID)
	if tail <= head {
		return nil
	}

	var result []OutboundMsg
	for idx := head; idx < tail && uint64(len(result)) < limit; idx++ {
		bz := store.Get(types.OutboundItemKey(dstChainID, idx))
		if bz == nil {
			continue
		}
		var msg OutboundMsg
		if err := json.Unmarshal(bz, &msg); err != nil {
			continue
		}
		if msg.SeqNum <= afterSeq {
			continue
		}
		result = append(result, msg)
	}
	return result
}

// EnqueueOutboundForTest is an exported wrapper for testing.
func (k Keeper) EnqueueOutboundForTest(ctx sdk.Context, msg OutboundMsg) error {
	return k.enqueueOutbound(ctx, msg)
}

// --- EVM log parsing ---

// parseAndEnqueueOutbound parses EVM logs from a gateway ApplyMessage call and
// enqueues any OutboundResponse or OutboundMessage events as outbound messages.
func (k Keeper) parseAndEnqueueOutbound(ctx sdk.Context, logs []*evmtypes.Log, srcChainID uint64) {
	for _, log := range logs {
		if len(log.Topics) == 0 {
			continue
		}
		topicHash := common.HexToHash(log.Topics[0])

		switch topicHash {
		case outboundResponseEventID:
			k.handleOutboundResponseLog(ctx, log, srcChainID)
		case outboundMessageEventID:
			k.handleOutboundMessageLog(ctx, log)
		}
	}
}

// handleOutboundResponseLog parses: OutboundResponse(uint32 indexed dstChainId, uint64 indexed requestNonce, bytes payload)
// Topics: [eventSig, dstChainId, requestNonce]
// Data: ABI-encoded (bytes payload)
func (k Keeper) handleOutboundResponseLog(ctx sdk.Context, log *evmtypes.Log, srcChainID uint64) {
	if len(log.Topics) < 3 {
		return
	}

	dstChainID := new(big.Int).SetBytes(common.HexToHash(log.Topics[1]).Bytes()).Uint64()
	nonce := new(big.Int).SetBytes(common.HexToHash(log.Topics[2]).Bytes()).Uint64()

	// Decode ABI-encoded (bytes payload) from data
	payload, err := decodeABIBytes(log.Data)
	if err != nil {
		ctx.Logger().Error("failed to decode OutboundResponse payload", "err", err)
		return
	}

	seq := k.getOutboundNextSeq(ctx, dstChainID)
	msg := OutboundMsg{
		DstChainID: dstChainID,
		SeqNum:     seq,
		Nonce:      nonce,
		PayloadHex: hex.EncodeToString(payload),
		Height:     ctx.BlockHeight(),
	}
	if err := k.enqueueOutbound(ctx, msg); err != nil {
		ctx.Logger().Error("failed to enqueue outbound response", "err", err, "dstChainID", dstChainID)
		return
	}
	k.setOutboundNextSeq(ctx, dstChainID, seq+1)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeOutbound,
		sdk.NewAttribute(types.AttributeKeyOutboundDstChainID, fmt.Sprintf("%d", dstChainID)),
		sdk.NewAttribute(types.AttributeKeyOutboundSeqNum, fmt.Sprintf("%d", seq)),
		sdk.NewAttribute(types.AttributeKeyOutboundNonce, fmt.Sprintf("%d", nonce)),
		sdk.NewAttribute(types.AttributeKeyOutboundPayloadBytes, fmt.Sprintf("%d", len(payload))),
	))
}

// handleOutboundMessageLog parses: OutboundMessage(uint32 indexed dstChainId, bytes payload)
// Topics: [eventSig, dstChainId]
// Data: ABI-encoded (bytes payload)
func (k Keeper) handleOutboundMessageLog(ctx sdk.Context, log *evmtypes.Log) {
	if len(log.Topics) < 2 {
		return
	}

	dstChainID := new(big.Int).SetBytes(common.HexToHash(log.Topics[1]).Bytes()).Uint64()

	payload, err := decodeABIBytes(log.Data)
	if err != nil {
		ctx.Logger().Error("failed to decode OutboundMessage payload", "err", err)
		return
	}

	seq := k.getOutboundNextSeq(ctx, dstChainID)
	msg := OutboundMsg{
		DstChainID: dstChainID,
		SeqNum:     seq,
		Nonce:      0, // admin-initiated, no inbound nonce
		PayloadHex: hex.EncodeToString(payload),
		Height:     ctx.BlockHeight(),
	}
	if err := k.enqueueOutbound(ctx, msg); err != nil {
		ctx.Logger().Error("failed to enqueue outbound message", "err", err, "dstChainID", dstChainID)
		return
	}
	k.setOutboundNextSeq(ctx, dstChainID, seq+1)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeOutbound,
		sdk.NewAttribute(types.AttributeKeyOutboundDstChainID, fmt.Sprintf("%d", dstChainID)),
		sdk.NewAttribute(types.AttributeKeyOutboundSeqNum, fmt.Sprintf("%d", seq)),
		sdk.NewAttribute(types.AttributeKeyOutboundPayloadBytes, fmt.Sprintf("%d", len(payload))),
	))
}

// decodeABIBytes decodes a single ABI-encoded `bytes` value from calldata.
func decodeABIBytes(data []byte) ([]byte, error) {
	bytesTy, err := abi.NewType("bytes", "", nil)
	if err != nil {
		return nil, err
	}
	args := abi.Arguments{{Type: bytesTy}}
	values, err := args.Unpack(data)
	if err != nil {
		return nil, fmt.Errorf("abi unpack bytes: %w", err)
	}
	if len(values) == 0 {
		return nil, fmt.Errorf("empty abi unpack result")
	}
	b, ok := values[0].([]byte)
	if !ok {
		return nil, fmt.Errorf("unexpected type from abi unpack: %T", values[0])
	}
	return b, nil
}

