package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/imua-xyz/imuachain/x/avs/types"
)

// GetParams get all parameters as types.Params
func (k Keeper) GetParams(ctx sdk.Context) *types.Params {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixParams)
	value := store.Get(types.ParamsKey)
	ret := &types.Params{}
	k.cdc.MustUnmarshal(value, ret)
	return ret
}

// SetParams set the params
func (k Keeper) SetParams(ctx sdk.Context, params *types.Params) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixParams)
	bz := k.cdc.MustMarshal(params)
	store.Set(types.ParamsKey, bz)
}
