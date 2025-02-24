package keeper

import (
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/imua-xyz/imuachain/x/oracle/keeper/common"
)

func (k Keeper) BeginBlock(ctx sdk.Context, _ abci.RequestBeginBlock) {
	if k.postHandlers == nil {
		k.postHandlers = make(map[int64]common.PostAggregationHandler)
		// bond handlers for custom pre defined token feeders
		k.RegisterPostAggregation()
		// bond handlers for nst token feeders
		p := k.GetParams(ctx)
		for tfID, tf := range p.TokenFeeders {
			// #nosec G115 - safe conversion since tokenId is set from slice index
			if p.IsNST(int(tf.TokenID)) {
				k.BondPostAggregation(int64(tfID), UpdateNSTBalanceChange)
			}
		}
	}
	k.FeederManager.BeginBlock(ctx)
}
