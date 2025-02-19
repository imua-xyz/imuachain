//nolint:dupl
package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/imua-xyz/imuachain/x/oracle/types"
)

// SetIndexRecentMsg set indexRecentMsg in the store
func (k Keeper) SetIndexRecentMsg(ctx sdk.Context, indexRecentMsg types.IndexRecentMsg) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.IndexRecentMsgKey))
	b := k.cdc.MustMarshal(&indexRecentMsg)
	store.Set([]byte{0}, b)
}

// GetIndexRecentMsg returns indexRecentMsg
func (k Keeper) GetIndexRecentMsg(ctx sdk.Context) (val types.IndexRecentMsg, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.IndexRecentMsgKey))

	b := store.Get([]byte{0})
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveIndexRecentMsg removes indexRecentMsg from the store
func (k Keeper) RemoveIndexRecentMsg(ctx sdk.Context) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.IndexRecentMsgKey))
	store.Delete([]byte{0})
}
