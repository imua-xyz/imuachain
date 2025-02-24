package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/imua-xyz/imuachain/x/appchain/coordinator/types"
	epochstypes "github.com/imua-xyz/imuachain/x/epochs/types"
)

// AppendChainToInitTimeout appends a chain to the list of chains which will timeout (if not
// initialized by then) at the end of the epoch.
func (k Keeper) AppendChainToInitTimeout(
	ctx sdk.Context, epoch epochstypes.Epoch, chainID string,
) {
	prev := k.GetChainsToInitTimeout(ctx, epoch)
	prev.List = append(prev.List, chainID)
	k.SetChainsToInitTimeout(ctx, epoch, prev)
}

// GetChainsToInitTimeout returns the list of chains which will timeout (if not initialized by then)
// at the end of the epoch.
func (k Keeper) GetChainsToInitTimeout(
	ctx sdk.Context, epoch epochstypes.Epoch,
) (res types.ChainIDs) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.InitTimeoutEpochKey(epoch))
	k.cdc.MustUnmarshal(bz, &res)
	return res
}

// SetChainsToInitTimeout sets the list of chains which will timeout (if not initialized by then)
// at the end of the epoch.
func (k Keeper) SetChainsToInitTimeout(
	ctx sdk.Context, epoch epochstypes.Epoch, chains types.ChainIDs,
) {
	store := ctx.KVStore(k.storeKey)
	bz := k.cdc.MustMarshal(&chains)
	store.Set(types.InitTimeoutEpochKey(epoch), bz)
}

// RemoveChainFromInitTimeout removes a chain from the list of chains which will timeout (if not
// initialized by then) at the end of the epoch.
func (k Keeper) RemoveChainFromInitTimeout(
	ctx sdk.Context, epoch epochstypes.Epoch, chainID string,
) {
	prev := k.GetChainsToInitTimeout(ctx, epoch)
	for i, id := range prev.List {
		if id == chainID {
			prev.List = append(prev.List[:i], prev.List[i+1:]...)
			break
		}
	}
	k.SetChainsToInitTimeout(ctx, epoch, prev)
}

// ClearChainsToInitTimeout clears the list of chains which will timeout (if not initialized by then)
// at the end of the epoch.
func (k Keeper) ClearChainsToInitTimeout(
	ctx sdk.Context, epoch epochstypes.Epoch,
) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(types.InitTimeoutEpochKey(epoch))
}
