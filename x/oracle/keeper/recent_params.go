package keeper

import (
	"github.com/ExocoreNetwork/exocore/x/oracle/types"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) SetParamsForCache(ctx sdk.Context, params types.RecentParams) {
	block := uint64(ctx.BlockHeight())
	index, found := k.GetIndexRecentParams(ctx)
	if found {
		i := 0
		// if the maxNonce is changed in this block, all rounds would be force sealed, so it's ok to use either the old or new maxNonce
		maxNonce := k.GetParams(ctx).MaxNonce
		for ; i < len(index.Index); i++ {
			b := index.Index[i]
			if b > block-uint64(maxNonce) {
				break
			}
			// remove old recentParams
			k.RemoveRecentParams(ctx, b)
		}
		index.Index = index.Index[i:]
	}
	index.Index = append(index.Index, block)
	k.SetIndexRecentParams(ctx, index)
	k.SetRecentParams(ctx, params)
}

// SetRecentParams set a specific recentParams in the store from its index
func (k Keeper) SetRecentParams(ctx sdk.Context, recentParams types.RecentParams) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.RecentParamsKeyPrefix))
	b := k.cdc.MustMarshal(&recentParams)
	store.Set(types.RecentParamsKey(
		recentParams.Block,
	), b)
}

// GetRecentParams returns a recentParams from its index
func (k Keeper) GetRecentParams(
	ctx sdk.Context,
	block uint64,
) (val types.RecentParams, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.RecentParamsKeyPrefix))

	b := store.Get(types.RecentParamsKey(
		block,
	))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveRecentParams removes a recentParams from the store
func (k Keeper) RemoveRecentParams(
	ctx sdk.Context,
	block uint64,
) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.RecentParamsKeyPrefix))
	store.Delete(types.RecentParamsKey(
		block,
	))
}

// GetAllRecentParams returns all recentParams
func (k Keeper) GetAllRecentParams(ctx sdk.Context) (list []types.RecentParams) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.RecentParamsKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.RecentParams
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

func (k Keeper) GetAllRecentParamsAsMap(ctx sdk.Context) (result map[int64]*types.Params) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.RecentParamsKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	result = make(map[int64]*types.Params)

	for ; iterator.Valid(); iterator.Next() {
		var val types.RecentParams
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		// #nosec G115
		result[int64(val.Block)] = val.Params
	}

	return
}

// GetRecentParamsWithinMaxNonce returns all recentParams within the maxNonce and the latest recentParams separately
func (k Keeper) GetRecentParamsWithinMaxNonce(ctx sdk.Context) (recentParamsList []*types.RecentParams, prev, latest types.RecentParams) {
	maxNonce := k.GetParams(ctx).MaxNonce
	startHeight := uint64(ctx.BlockHeight()) - uint64(maxNonce)

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.RecentParamsKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	recentParamsList = make([]*types.RecentParams, 0, maxNonce)
	//	latest := types.RecentParams{}
	//	prev := types.RecentParams{}
	notFound := true
	for ; iterator.Valid(); iterator.Next() {
		var val types.RecentParams
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		latest = val
		if notFound {
			prev = val
		}
		if val.Block >= startHeight {
			if !notFound {
				notFound = true
			}
			recentParamsList = append(recentParamsList, &val)
		}
	}
	if len(recentParamsList) > 0 {
		if prev.Block == recentParamsList[0].Block {
			prev = types.RecentParams{}
		}
		if latest.Block == recentParamsList[len(recentParamsList)-1].Block {
			latest = types.RecentParams{}
		}
	}
	return
}
