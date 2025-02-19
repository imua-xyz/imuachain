// nolint
package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/imua-xyz/imuachain/x/oracle/types"
)

// SetIndexRecentParams set indexRecentParams in the store
func (k Keeper) SetIndexRecentParams(ctx sdk.Context, indexRecentParams types.IndexRecentParams) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.IndexRecentParamsKey))
	b := k.cdc.MustMarshal(&indexRecentParams)
	store.Set([]byte{0}, b)
}

// GetIndexRecentParams returns indexRecentParams
func (k Keeper) GetIndexRecentParams(ctx sdk.Context) (val types.IndexRecentParams, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.IndexRecentParamsKey))

	b := store.Get([]byte{0})
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveIndexRecentParams removes indexRecentParams from the store
func (k Keeper) RemoveIndexRecentParams(ctx sdk.Context) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.IndexRecentParamsKey))
	store.Delete([]byte{0})
}
