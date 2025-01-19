package keeper

import (
	"github.com/ExocoreNetwork/exocore/x/oracle/types"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// SetMsgItemsForCache set a specific recentMsg with its height as index in the store
func (k Keeper) SetMsgItemsForCache(ctx sdk.Context, recentMsg types.RecentMsg) {
	index, found := k.GetIndexRecentMsg(ctx)
	block := uint64(ctx.BlockHeight())
	if found {
		i := 0
		maxNonce := k.GetParams(ctx).MaxNonce
		for ; i < len(index.Index); i++ {
			b := index.Index[i]
			// #nosec G115  // maxNonce is not negative
			if block < uint64(maxNonce) || b > block-uint64(maxNonce) {
				break
			}
			// remove old recentMsg
			k.RemoveRecentMsg(ctx, b)
		}
		index.Index = index.Index[i:]
	}
	index.Index = append(index.Index, block)
	k.SetIndexRecentMsg(ctx, index)
	k.SetRecentMsg(ctx, recentMsg)
}

// SetRecentMsg set a specific recentMsg in the store from its index
func (k Keeper) SetRecentMsg(ctx sdk.Context, recentMsg types.RecentMsg) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.RecentMsgKeyPrefix))
	b := k.cdc.MustMarshal(&recentMsg)
	store.Set(types.RecentMsgKey(
		recentMsg.Block,
	), b)
}

// GetRecentMsg returns a recentMsg from its index
func (k Keeper) GetRecentMsg(
	ctx sdk.Context,
	block uint64,
) (val types.RecentMsg, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.RecentMsgKeyPrefix))

	b := store.Get(types.RecentMsgKey(
		block,
	))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveRecentMsg removes a recentMsg from the store
func (k Keeper) RemoveRecentMsg(
	ctx sdk.Context,
	block uint64,
) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.RecentMsgKeyPrefix))
	store.Delete(types.RecentMsgKey(
		block,
	))
}

// GetAllRecentMsg returns all recentMsg
func (k Keeper) GetAllRecentMsg(ctx sdk.Context) (list []types.RecentMsg) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.RecentMsgKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.RecentMsg
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

func (k Keeper) GetAllRecentMsgAsMap(ctx sdk.Context) (result map[int64][]*types.MsgItem) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.RecentMsgKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()
	result = make(map[int64][]*types.MsgItem)

	for ; iterator.Valid(); iterator.Next() {
		var val types.RecentMsg
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		//		list = append(list, val)
		// #nosec G115
		result[int64(val.Block)] = val.Msgs
	}

	return
}
