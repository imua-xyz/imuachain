package types

const (
	// OutboundPrefix is the top-level prefix for all outbound queue state.
	OutboundPrefix = "Outbound/"
	// OutboundSeqPrefix stores the next monotonic sequence number per dstChainID.
	OutboundSeqPrefix = OutboundPrefix + "seq/"
	// OutboundHeadPrefix stores the queue head index per dstChainID.
	OutboundHeadPrefix = OutboundPrefix + "head/"
	// OutboundTailPrefix stores the queue tail index per dstChainID.
	OutboundTailPrefix = OutboundPrefix + "tail/"
	// OutboundItemPrefix stores individual queued outbound messages.
	OutboundItemPrefix = OutboundPrefix + "item/"
)

func OutboundSeqKey(dstChainID uint64) []byte {
	return append([]byte(OutboundSeqPrefix), Uint64Bytes(dstChainID)...)
}

func OutboundHeadKey(dstChainID uint64) []byte {
	return append([]byte(OutboundHeadPrefix), Uint64Bytes(dstChainID)...)
}

func OutboundTailKey(dstChainID uint64) []byte {
	return append([]byte(OutboundTailPrefix), Uint64Bytes(dstChainID)...)
}

func OutboundItemKey(dstChainID, idx uint64) []byte {
	key := make([]byte, 0, len(OutboundItemPrefix)+8+8)
	key = append(key, []byte(OutboundItemPrefix)...)
	key = append(key, Uint64Bytes(dstChainID)...)
	key = append(key, Uint64Bytes(idx)...)
	return key
}
