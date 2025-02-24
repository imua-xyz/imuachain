package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/imua-xyz/imuachain/x/epochs/types"
)

// AddEpochInfo adds a new epoch info to the store.
// It validates the epoch info being sent, and checks that an epoch with the same
// identifier does not already exist.
// Before saving, it sets the start time and current epoch start height if they are not set.
// Since it validates (and fills, where necessary) the input provided, it is a public function.
func (k Keeper) AddEpochInfo(ctx sdk.Context, epochInfo types.EpochInfo) error {
	if err := epochInfo.Validate(); err != nil {
		return err
	}
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixEpoch)
	if store.Has([]byte(epochInfo.Identifier)) {
		return types.ErrDuplicateEpochInfo
	}
	if epochInfo.StartTime.IsZero() {
		// this is from the block header, which has it in UTC
		epochInfo.StartTime = ctx.BlockTime()
	}
	if epochInfo.CurrentEpochStartHeight == 0 {
		// at genesis, this will still be 0.
		// then, begin blocker will set it to 1.
		epochInfo.CurrentEpochStartHeight = ctx.BlockHeight()
	}
	k.setEpochInfoUnchecked(ctx, epochInfo)
	return nil
}

// GetEpochInfo returns the epoch info for the given identifier.
// If the epoch info does not exist, it returns false.
func (k Keeper) GetEpochInfo(
	ctx sdk.Context, identifier string,
) (epoch types.EpochInfo, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixEpoch)
	bz := store.Get([]byte(identifier))
	if len(bz) == 0 {
		return epoch, false
	}
	k.cdc.MustUnmarshal(bz, &epoch)
	return epoch, true
}

// setEpochInfoUnchecked sets the provided epoch info in the store, indexed by the identifier.
// It performs no validation; the caller must ensure that it is valid and all the fields
// are populated correctly. This is why the function is private to this module.
func (k Keeper) setEpochInfoUnchecked(ctx sdk.Context, epoch types.EpochInfo) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixEpoch)
	bz := k.cdc.MustMarshal(&epoch)
	store.Set([]byte(epoch.Identifier), bz)
}

// IterateEpochInfos iterates through all the epochs.
func (k Keeper) IterateEpochInfos(
	ctx sdk.Context, fn func(
		index int64, epochInfo types.EpochInfo,
	) (stop bool),
) {
	store := ctx.KVStore(k.storeKey)

	iterator := sdk.KVStorePrefixIterator(store, types.KeyPrefixEpoch)
	defer iterator.Close()

	i := int64(0)

	for ; iterator.Valid(); iterator.Next() {
		epoch := types.EpochInfo{}
		k.cdc.MustUnmarshal(iterator.Value(), &epoch)

		if fn(i, epoch) { // stop
			break
		}

		i++
	}
}

// AllEpochInfos returns all the epoch infos.
func (k Keeper) AllEpochInfos(ctx sdk.Context) []types.EpochInfo {
	epochs := []types.EpochInfo{}
	k.IterateEpochInfos(ctx, func(_ int64, epochInfo types.EpochInfo) (stop bool) {
		epochs = append(epochs, epochInfo)
		return false
	})
	return epochs
}
