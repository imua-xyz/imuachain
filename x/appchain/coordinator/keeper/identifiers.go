package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/imua-xyz/imuachain/x/appchain/coordinator/types"
)

// SetClientForChain sets the ibc client id for a given chain id.
func (k Keeper) SetClientForChain(
	ctx sdk.Context, chainID string, clientID string,
) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.ClientForChainKey(chainID), []byte(clientID))
}

// GetClientForChain gets the ibc client id for a given chain id.
func (k Keeper) GetClientForChain(
	ctx sdk.Context, chainID string,
) (string, bool) {
	store := ctx.KVStore(k.storeKey)
	bytes := store.Get(types.ClientForChainKey(chainID))
	if bytes == nil {
		return "", false
	}
	return string(bytes), true
}

// DeleteClientForChain deletes the ibc client id for a given chain id.
func (k Keeper) DeleteClientForChain(ctx sdk.Context, chainID string) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(types.ClientForChainKey(chainID))
}
