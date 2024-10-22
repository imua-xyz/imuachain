package keeper

import (
	"fmt"

	"github.com/ExocoreNetwork/exocore/x/appchain/coordinator/types"
	epochstypes "github.com/ExocoreNetwork/exocore/x/epochs/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
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
	if len(chains.List) == 0 {
		store.Delete(types.InitTimeoutEpochKey(epoch))
		return
	}
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

// SetChainInitTimeout stores a lookup from chain to the epoch by the end of which
// it must be initialized.
func (k Keeper) SetChainInitTimeout(
	ctx sdk.Context, chainID string, epoch epochstypes.Epoch,
) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.ChainInitTimeoutKey(chainID), k.cdc.MustMarshal(&epoch))
}

// GetChainInitTimeout returns the epoch by the end of which the chain must be initialized.
func (k Keeper) GetChainInitTimeout(
	ctx sdk.Context, chainID string,
) (epoch epochstypes.Epoch, found bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ChainInitTimeoutKey(chainID))
	if bz == nil {
		return epoch, false
	}
	k.cdc.MustUnmarshal(bz, &epoch)
	return epoch, true
}

// DeleteChainInitTimeout deletes the lookup from chain to the epoch by the end of which
// it must be initialized.
func (k Keeper) DeleteChainInitTimeout(ctx sdk.Context, chainID string) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(types.ChainInitTimeoutKey(chainID))
}

// SetVscTimeout stores the epoch by the end of which a response to a VSC must be received.
func (k Keeper) SetVscTimeout(
	ctx sdk.Context, chainID string, vscID uint64, timeout epochstypes.Epoch,
) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.VscTimeoutKey(chainID, vscID), k.cdc.MustMarshal(&timeout))
}

// GetVscTimeout returns the epoch by the end of which a response to a VSC must be received.
func (k Keeper) GetVscTimeout(
	ctx sdk.Context, chainID string, vscID uint64,
) (timeout epochstypes.Epoch, found bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.VscTimeoutKey(chainID, vscID))
	if bz == nil {
		return timeout, false
	}
	k.cdc.MustUnmarshal(bz, &timeout)
	return timeout, true
}

// GetFirstVscTimeout returns the first epoch by the end of which a response to a VSC must be received.
func (k Keeper) GetFirstVscTimeout(
	ctx sdk.Context, chainID string,
) (timeout epochstypes.Epoch, vscID uint64, found bool) {
	store := ctx.KVStore(k.storeKey)
	partialKey := append(
		[]byte{types.VscTimeoutBytePrefix},
		[]byte(chainID)...,
	)
	iterator := sdk.KVStorePrefixIterator(store, partialKey)
	defer iterator.Close()

	if iterator.Valid() {
		_, vscID, err := types.ParseVscTimeoutKey(iterator.Key())
		if err != nil {
			return timeout, 0, false
		}
		bz := iterator.Value()
		k.cdc.MustUnmarshal(bz, &timeout)
		return timeout, vscID, true
	}
	return timeout, 0, false
}

// DeleteVscTimeout deletes the epoch by the end of which a response to a VSC must be received.
func (k Keeper) DeleteVscTimeout(ctx sdk.Context, chainID string, vscID uint64) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(types.VscTimeoutKey(chainID, vscID))
}

// RemoveTimedoutSubscribers removes the subscribers that are timed out at the end of the current epoch.
// epochNumber passed is the current epoch number, which is ending.
func (k Keeper) RemoveTimedoutSubscribers(ctx sdk.Context, epochIdentifier string, epochNumber int64) {
	// init timeout chains
	epoch := epochstypes.NewEpoch(uint64(epochNumber), epochIdentifier)
	chains := k.GetChainsToInitTimeout(ctx, epoch)
	for _, chainID := range chains.List {
		if err := k.StopSubscriberChain(ctx, chainID, true); err != nil {
			k.Logger(ctx).Error("failed to stop subscriber chain", "chainID", chainID, "error", err)
			continue
		}
		k.DeleteChainInitTimeout(ctx, chainID) // prune
	}
	// vsc timeout chains
	vscChains := k.GetAllChainsWithChannels(ctx)
	for _, chainID := range vscChains {
		fmt.Println("chainID", chainID)
		timeout, vscID, found := k.GetFirstVscTimeout(ctx, chainID)
		if !found {
			continue
		}
		fmt.Println("timeout", timeout)
		if timeout.EpochIdentifier == epochIdentifier && timeout.EpochNumber <= uint64(epochNumber) {
			k.Logger(ctx).Info(
				"VSC timed out, removing subscriber",
				"chainID", chainID,
				"epochIdentifier", timeout.EpochIdentifier,
				"epochNumber", timeout.EpochNumber,
			)
			if err := k.StopSubscriberChain(ctx, chainID, true); err != nil {
				k.Logger(ctx).Error("failed to stop subscriber chain", "chainID", chainID, "error", err)
				continue
			}
			k.DeleteVscTimeout(ctx, chainID, vscID) // prune
		}
	}
}
