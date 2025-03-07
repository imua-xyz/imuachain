package keeper

import (
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) BeginBlock(ctx sdk.Context, _ abci.RequestBeginBlock) {
	if len(k.postHandlers) == 0 {
		// bond handlers for custom pre defined token feeders
		k.RegisterPostAggregation()
		// bond handlers for nst token feeders
		p := k.GetParams(ctx)
		// it's safe to iterate over the map, the order of the elements is not important
		for tfID, tf := range p.TokenFeeders {
			// #nosec G115 - safe conversion since tokenId is set from slice index
			if p.IsNST(int(tf.TokenID)) {
				k.BondPostAggregation(int64(tfID), UpdateNSTBalanceChange)
			}
		}
	}
	k.FeederManager.BeginBlock(ctx)
}
