package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/imua-xyz/imuachain/x/delegation/types"
)

// GetParams get all parameters as types.Params
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	store := ctx.KVStore(k.storeKey)
	key := types.KeyPrefixParams
	bz := store.Get(key)
	var params types.Params
	k.cdc.MustUnmarshal(bz, &params)
	return params
}

// SetParams set the Undelegation Penalty.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	store := ctx.KVStore(k.storeKey)
	key := types.KeyPrefixParams
	bz := k.cdc.MustMarshal(&params)
	store.Set(key, bz)
}

// Get Instant Undelegation Penalty.
func (k Keeper) GetInstantUndelegationPenalty(ctx sdk.Context) uint32 {
	params := k.GetParams(ctx)
	return params.InstantUndelegationPenalty
}
