package keeper

import (
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/imua-xyz/imuachain/x/oracle/types"
)

func (k Keeper) BeginBlock(ctx sdk.Context, _ abci.RequestBeginBlock) {
	defer telemetry.ModuleMeasureSince(types.ModuleName, time.Now(), telemetry.MetricKeyBeginBlocker)
	// Bond handlers for token feeders.
	// This is kept idempotent so new token feeders (added via params update) can be bonded without relying
	// on keeper re-initialization.
	p := k.GetParams(ctx)
	for tfID, tf := range p.TokenFeeders {
		if _, exists := k.postHandlers[int64(tfID)]; exists {
			continue
		}
		// #nosec G115 - safe conversion since tokenId is set from slice index
		if p.IsNST(int(tf.TokenID)) {
			k.BondPostAggregation(int64(tfID), UpdateNSTBalanceChange)
		}
		// #nosec G115 - safe conversion since tokenId is set from slice index
		if p.IsXChain(int(tf.TokenID)) {
			k.BondPostAggregation(int64(tfID), UpdateXChainMsgs)
		}
	}
	k.FeederManager.BeginBlock(ctx)
}
