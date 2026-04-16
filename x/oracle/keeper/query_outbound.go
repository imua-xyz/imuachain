package keeper

import (
	"encoding/json"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// QueryOutboundMessagesResponse is the JSON response for the outbound messages query.
type QueryOutboundMessagesResponse struct {
	Messages []OutboundMsg `json:"messages"`
}

// QueryOutboundMessages returns pending outbound messages for a destination chain.
// This is called via the CLI or ABCI query path, not via proto-generated gRPC.
func (k Keeper) QueryOutboundMessages(ctx sdk.Context, dstChainID, afterSeq, limit uint64) *QueryOutboundMessagesResponse {
	msgs := k.GetOutboundMessages(ctx, dstChainID, afterSeq, limit)
	if msgs == nil {
		msgs = []OutboundMsg{}
	}
	return &QueryOutboundMessagesResponse{Messages: msgs}
}

// QueryOutboundMessagesJSON is a convenience wrapper that returns the JSON bytes.
func (k Keeper) QueryOutboundMessagesJSON(ctx sdk.Context, dstChainID, afterSeq, limit uint64) ([]byte, error) {
	resp := k.QueryOutboundMessages(ctx, dstChainID, afterSeq, limit)
	return json.Marshal(resp)
}
