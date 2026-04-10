package types

import "crypto/sha256"

const (
	XChainPrefix = "XChain/"
	// XChainLastAcceptedSeqPrefix stores the last accepted (enqueued) batch seq per srcChainID.
	XChainLastSeqPrefix = XChainPrefix + "lastSeq/"
	// XChainLastExecutedSeqPrefix stores the last fully executed batch seq per srcChainID.
	XChainLastExecutedSeqPrefix = XChainPrefix + "lastExecSeq/"
	XChainMsgPrefix             = XChainPrefix + "msg/"

	// XChainQueue prefixes for queued batches (budgeted EndBlock delivery).
	XChainQueuePrefix     = XChainPrefix + "queue/"
	XChainQueueHeadPrefix = XChainQueuePrefix + "head/"
	XChainQueueTailPrefix = XChainQueuePrefix + "tail/"
	XChainQueueItemPrefix = XChainQueuePrefix + "item/"
	XChainMsgRetryPrefix  = XChainPrefix + "retry/"
)

func XChainLastSeqKey(srcChainID uint64) []byte {
	return append([]byte(XChainLastSeqPrefix), Uint64Bytes(srcChainID)...)
}

func XChainLastExecutedSeqKey(srcChainID uint64) []byte {
	return append([]byte(XChainLastExecutedSeqPrefix), Uint64Bytes(srcChainID)...)
}

func XChainQueueHeadKey(srcChainID uint64) []byte {
	return append([]byte(XChainQueueHeadPrefix), Uint64Bytes(srcChainID)...)
}

func XChainQueueTailKey(srcChainID uint64) []byte {
	return append([]byte(XChainQueueTailPrefix), Uint64Bytes(srcChainID)...)
}

func XChainQueueItemKey(srcChainID, idx uint64) []byte {
	key := make([]byte, 0, len(XChainQueueItemPrefix)+8+8)
	key = append(key, []byte(XChainQueueItemPrefix)...)
	key = append(key, Uint64Bytes(srcChainID)...)
	key = append(key, Uint64Bytes(idx)...)
	return key
}

// XChainMsgProcessedKey builds a stable key for a processed message under a source chain.
// We hash msgID to keep keys short and avoid key-space abuse.
func XChainMsgProcessedKey(srcChainID uint64, msgID string) []byte {
	sum := sha256.Sum256([]byte(msgID))
	key := make([]byte, 0, len(XChainMsgPrefix)+8+len(sum))
	key = append(key, []byte(XChainMsgPrefix)...)
	key = append(key, Uint64Bytes(srcChainID)...)
	key = append(key, sum[:]...)
	return key
}

// XChainMsgRetryKey stores per-message retry counts.
func XChainMsgRetryKey(srcChainID uint64, msgID string) []byte {
	sum := sha256.Sum256([]byte(msgID))
	key := make([]byte, 0, len(XChainMsgRetryPrefix)+8+len(sum))
	key = append(key, []byte(XChainMsgRetryPrefix)...)
	key = append(key, Uint64Bytes(srcChainID)...)
	key = append(key, sum[:]...)
	return key
}
