package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/imua-xyz/imuachain/x/dogfood/types"
)

// SetPendingOptOuts sets the pending opt-outs to be applied at the end of the block.
func (k Keeper) SetPendingOptOuts(ctx sdk.Context, addrs types.AccountAddresses) {
	store := ctx.KVStore(k.storeKey)
	bz := k.cdc.MustMarshal(&addrs)
	store.Set(types.PendingOptOutsKey(), bz)
}

// GetPendingOptOuts returns the pending opt-outs to be applied at the end of the block.
func (k Keeper) GetPendingOptOuts(ctx sdk.Context) types.AccountAddresses {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.PendingOptOutsKey())
	if bz == nil {
		return types.AccountAddresses{}
	}
	var addrs types.AccountAddresses
	if err := addrs.Unmarshal(bz); err != nil {
		return types.AccountAddresses{}
	}
	return addrs
}

// ClearPendingOptOuts clears the pending opt-outs to be applied at the end of the block.
func (k Keeper) ClearPendingOptOuts(ctx sdk.Context) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(types.PendingOptOutsKey())
}

// SetPendingConsensusAddrs sets the pending consensus addresses to be pruned at the end of the
// block.
func (k Keeper) SetPendingConsensusAddrs(ctx sdk.Context, addrs types.ConsensusAddresses) {
	store := ctx.KVStore(k.storeKey)
	bz := k.cdc.MustMarshal(&addrs)
	store.Set(types.PendingConsensusAddrsKey(), bz)
}

// GetPendingConsensusAddrs returns the pending consensus addresses to be pruned at the end of
// the block.
func (k Keeper) GetPendingConsensusAddrs(ctx sdk.Context) types.ConsensusAddresses {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.PendingConsensusAddrsKey())
	if bz == nil {
		return types.ConsensusAddresses{}
	}
	var addrs types.ConsensusAddresses
	if err := addrs.Unmarshal(bz); err != nil {
		return types.ConsensusAddresses{}
	}
	return addrs
}

// ClearPendingConsensusAddrs clears the pending consensus addresses to be pruned at the end of
// the block.
func (k Keeper) ClearPendingConsensusAddrs(ctx sdk.Context) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(types.PendingConsensusAddrsKey())
}

// SetPendingUndelegations sets the pending undelegations to be released at the end of the
// block.
func (k Keeper) SetPendingUndelegations(
	ctx sdk.Context,
	undelegations types.UndelegationRecordKeys,
) {
	store := ctx.KVStore(k.storeKey)
	bz := k.cdc.MustMarshal(&undelegations)
	store.Set(types.PendingUndelegationsKey(), bz)
}

// GetPendingUndelegations returns the pending undelegations to be released at the end of the
// block.
func (k Keeper) GetPendingUndelegations(ctx sdk.Context) types.UndelegationRecordKeys {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.PendingUndelegationsKey())
	if bz == nil {
		return types.UndelegationRecordKeys{}
	}
	var undelegations types.UndelegationRecordKeys
	if err := undelegations.Unmarshal(bz); err != nil {
		return types.UndelegationRecordKeys{}
	}
	return undelegations
}

// ClearPendingUndelegations clears the pending undelegations to be released at the end of the
// block.
func (k Keeper) ClearPendingUndelegations(ctx sdk.Context) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(types.PendingUndelegationsKey())
}
