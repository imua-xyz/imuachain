package keeper

import "github.com/imua-xyz/imuachain/x/oracle/keeper/common"

// RegisterPostAggregation registers handler for tokenfeeder set with deterministic source which need to do some process with the deterministic aggregated result
// this is used to register the post handlers served for some customer defined deterministic source oracle requirement
func (k *Keeper) RegisterPostAggregation() {
	// k.BondPostAggregation(1, UpdateNSTBalanceChange)
}

func (k *Keeper) BondPostAggregation(feederID int64, postHandler common.PostAggregationHandler) {
	k.postHandlers[feederID] = postHandler
}

func (k *Keeper) GetPostAggregation(feederID int64) (handler common.PostAggregationHandler, found bool) {
	if k.postHandlers == nil {
		return nil, false
	}
	handler, found = k.postHandlers[feederID]
	return
}
