package keeper

import (
	"strconv"

	assetstype "github.com/ExocoreNetwork/exocore/x/assets/types"
	"github.com/ExocoreNetwork/exocore/x/operator/types"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k *Keeper) SetVotingPowerSnapshot(ctx sdk.Context, key []byte, snapshot *types.VotingPowerSnapshot) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixVotingPowerSnapshot)
	bz := k.cdc.MustMarshal(snapshot)
	store.Set(key, bz)
	return nil
}

func (k *Keeper) LoadVotingPowerSnapshot(ctx sdk.Context, avsAddr, epochIdentifier string, epochNumber int64, height *int64) (*types.VotingPowerSnapshot, error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixVotingPowerSnapshot)
	var ret types.VotingPowerSnapshot
	key := assetstype.GetJoinedStoreKey(avsAddr, epochIdentifier, strconv.FormatInt(epochNumber, 10))
	if height != nil && *height >= 0 {
		// When a snapshot caused by a slash exists for the specified epochNumber,
		// the snapshot closest to the input height is the one used for its voting
		// power information. The correct snapshot key can be found by taking advantage
		// of the ascending order of data returned when using an iterator range.
		keyWithHeight := assetstype.GetJoinedStoreKey(string(key), strconv.FormatInt(*height, 10))
		iterator := sdk.KVStorePrefixIterator(store, key)
		defer iterator.Close()
		var findKey []byte
		for ; iterator.Valid(); iterator.Next() {
			var amounts assetstype.OperatorAssetInfo
			k.cdc.MustUnmarshal(iterator.Value(), &amounts)
			if string(iterator.Key()) <= string(keyWithHeight) {
				findKey = iterator.Key()
			} else {
				break
			}
		}
		key = findKey
	}

	value := store.Get(key)
	if value == nil {
		return nil, types.ErrNoKeyInTheStore.Wrapf("LoadVotingPowerSnapshot: key is %s", key)
	}
	k.cdc.MustUnmarshal(value, &ret)

	// fall back to the last snapshot if the voting power set is nil
	if ret.VotingPowerSet == nil {
		value = store.Get([]byte(ret.LastChangedKey))
		if value == nil {
			return nil, types.ErrNoKeyInTheStore.Wrapf("LoadVotingPowerSnapshot: fall back to the key %s", ret.LastChangedKey)
		}
		k.cdc.MustUnmarshal(value, &ret)
	}
	return &ret, nil
}

func (k *Keeper) UpdateSnapshotHelper(ctx sdk.Context, avsAddr string, opFunc func(helper *types.SnapshotHelper) error) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixSnapshotHelper)
	var snapshotHelper types.SnapshotHelper
	value := store.Get([]byte(avsAddr))
	if value != nil {
		k.cdc.MustUnmarshal(value, &snapshotHelper)
	}
	err := opFunc(&snapshotHelper)
	if err != nil {
		return err
	}
	bz := k.cdc.MustMarshal(&snapshotHelper)
	store.Set([]byte(avsAddr), bz)
	return nil
}

func (k *Keeper) SetOptOutFlag(ctx sdk.Context, avsAddr string, hasOptOut bool) error {
	opFunc := func(helper *types.SnapshotHelper) error {
		helper.HasOptOut = hasOptOut
		return nil
	}
	return k.UpdateSnapshotHelper(ctx, avsAddr, opFunc)
}

func (k *Keeper) SetSlashFlag(ctx sdk.Context, avsAddr string, hasSlash bool) error {
	opFunc := func(helper *types.SnapshotHelper) error {
		helper.HasSlash = hasSlash
		return nil
	}
	return k.UpdateSnapshotHelper(ctx, avsAddr, opFunc)
}

func (k *Keeper) SetLastChangedKey(ctx sdk.Context, avsAddr, lastChangeKey string) error {
	opFunc := func(helper *types.SnapshotHelper) error {
		helper.LastChangedKey = lastChangeKey
		return nil
	}
	return k.UpdateSnapshotHelper(ctx, avsAddr, opFunc)
}

func (k *Keeper) SetSnapshotHelper(ctx sdk.Context, avsAddr string, snapshotHelper *types.SnapshotHelper) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixSnapshotHelper)
	bz := k.cdc.MustMarshal(snapshotHelper)
	store.Set([]byte(avsAddr), bz)
	return nil
}

func (k *Keeper) GetSnapshotHelper(ctx sdk.Context, avsAddr string) (types.SnapshotHelper, error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixSnapshotHelper)
	var ret types.SnapshotHelper
	value := store.Get([]byte(avsAddr))
	if value == nil {
		return ret, types.ErrNoKeyInTheStore.Wrapf("GetSnapshotHelper: the key is %s", avsAddr)
	}
	k.cdc.MustUnmarshal(value, &ret)
	return ret, nil
}

func (k *Keeper) HasSnapshotHelper(ctx sdk.Context, avsAddr string) bool {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixSnapshotHelper)
	return store.Has([]byte(avsAddr))
}

func (k *Keeper) HasSlash(ctx sdk.Context, avsAddr string) bool {
	helper, err := k.GetSnapshotHelper(ctx, avsAddr)
	if err != nil {
		return false
	}
	return helper.HasSlash
}
