package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
	assetstype "github.com/imua-xyz/imuachain/x/assets/types"
)

// SetClientChainInfo todo: Temporarily use LayerZeroChainID as key.
// It provides a function to register the client chains supported by exoCore.It's called by genesis configuration now,however it will be called by the governance in the future
func (k Keeper) SetClientChainInfo(ctx sdk.Context, info *assetstype.ClientChainInfo) (err error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), assetstype.KeyPrefixClientChainInfo)

	bz := k.cdc.MustMarshal(info)

	store.Set([]byte(hexutil.EncodeUint64(info.LayerZeroChainID)), bz)
	return nil
}

func (k Keeper) ClientChainExists(ctx sdk.Context, index uint64) bool {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), assetstype.KeyPrefixClientChainInfo)
	return store.Has([]byte(hexutil.EncodeUint64(index)))
}

// GetClientChainInfoByIndex using LayerZeroChainID as the query index.
func (k Keeper) GetClientChainInfoByIndex(ctx sdk.Context, index uint64) (info *assetstype.ClientChainInfo, err error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), assetstype.KeyPrefixClientChainInfo)
	value := store.Get([]byte(hexutil.EncodeUint64(index)))
	if value == nil {
		return nil, assetstype.ErrNoClientChainKey
	}
	ret := assetstype.ClientChainInfo{}
	k.cdc.MustUnmarshal(value, &ret)
	return &ret, nil
}

// IterateAllClientChains iterates all client chains, and the `opFunc` will be called for
// each client chain. As for the `isUpdate`, it a flag to indicate if the client chain
// info handled by the `opFunc` will be restored.
func (k Keeper) IterateAllClientChains(ctx sdk.Context, isUpdate bool, opFunc func(clientChain *assetstype.ClientChainInfo) error) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), assetstype.KeyPrefixClientChainInfo)
	iterator := sdk.KVStorePrefixIterator(store, nil)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		var clientChain assetstype.ClientChainInfo
		k.cdc.MustUnmarshal(iterator.Value(), &clientChain)
		err := opFunc(&clientChain)
		if err != nil {
			return err
		}
		if isUpdate {
			// store the updated state
			bz := k.cdc.MustMarshal(&clientChain)
			store.Set(iterator.Key(), bz)
		}
	}
	return nil
}

func (k Keeper) GetAllClientChainInfo(ctx sdk.Context) (infos []assetstype.ClientChainInfo, err error) {
	ret := make([]assetstype.ClientChainInfo, 0)
	opFunc := func(clientChain *assetstype.ClientChainInfo) error {
		ret = append(ret, *clientChain)
		return nil
	}
	err = k.IterateAllClientChains(ctx, false, opFunc)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (k Keeper) GetAllClientChainID(ctx sdk.Context) ([]uint32, error) {
	ret := make([]uint32, 0)
	opFunc := func(clientChain *assetstype.ClientChainInfo) error {
		// #nosec G115
		ret = append(ret, uint32(clientChain.LayerZeroChainID))
		return nil
	}
	err := k.IterateAllClientChains(ctx, false, opFunc)
	if err != nil {
		return nil, err
	}
	return ret, nil
}
