package keeper

import (
	"strconv"
	"time"

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

// LoadVotingPowerSnapshot loads the voting power snapshot information for the provided height,
// returning the height of the first block in the epoch the snapshot serves, along with the specific
// voting power data. The start height will be used to filter pending undelegations during slashing.
func (k *Keeper) LoadVotingPowerSnapshot(ctx sdk.Context, avsAddr string, height int64) (int64, *types.VotingPowerSnapshot, error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixVotingPowerSnapshot)
	var ret types.VotingPowerSnapshot
	// If there is no snapshot for the input height, we need to find the correct key.
	// The snapshot closest to the input height is the one used for its voting
	// power information. The correct snapshot key can be found by taking advantage
	// of the ascending order of data returned when using an iterator range.
	keyWithHeight := assetstype.GetJoinedStoreKey(avsAddr, strconv.FormatInt(height, 10))
	findKey := keyWithHeight
	if !store.Has(keyWithHeight) {
		iterator := sdk.KVStorePrefixIterator(store, []byte(avsAddr))
		defer iterator.Close()
		for ; iterator.Valid(); iterator.Next() {
			if string(iterator.Key()) <= string(keyWithHeight) {
				findKey = iterator.Key()
			} else {
				break
			}
		}
	}

	value := store.Get(findKey)
	if value == nil {
		return 0, nil, types.ErrNoKeyInTheStore.Wrapf("LoadVotingPowerSnapshot: key is %s", findKey)
	}
	k.cdc.MustUnmarshal(value, &ret)

	// fall back to the last snapshot if the voting power set is nil
	if ret.VotingPowerSet == nil {
		value = store.Get(assetstype.GetJoinedStoreKey(avsAddr, strconv.FormatInt(ret.LastChangedHeight, 10)))
		if value == nil {
			return 0, nil, types.ErrNoKeyInTheStore.Wrapf("LoadVotingPowerSnapshot: fall back to the height %v", ret.LastChangedHeight)
		}
		k.cdc.MustUnmarshal(value, &ret)
	}

	keysList, err := assetstype.ParseJoinedKey(findKey)
	if value == nil {
		return 0, nil, err
	}
	findHeight, err := strconv.ParseInt(keysList[1], 10, 64)
	if value == nil {
		return 0, nil, err
	}
	return findHeight, &ret, nil
}

// RemoveVotingPowerSnapshot remove all snapshots older than the input time.
func (k *Keeper) RemoveVotingPowerSnapshot(ctx sdk.Context, avsAddr string, time time.Time) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixVotingPowerSnapshot)
	iterator := sdk.KVStorePrefixIterator(store, []byte(avsAddr))
	defer iterator.Close()
	// the retained key is used to record the snapshot that will be fallen back to
	// by snapshots earlier than the input time.
	var retainedKey []byte
	var snapshot types.VotingPowerSnapshot
	for ; iterator.Valid(); iterator.Next() {
		k.cdc.MustUnmarshal(iterator.Value(), &snapshot)
		if snapshot.BlockTime.After(time) {
			// delete the retained key, because the snapshots that is earlier than the input time
			// don't need to retain any old snapshot key.
			if snapshot.VotingPowerSet != nil && retainedKey != nil {
				store.Delete(retainedKey)
			}
			break
		}
		if snapshot.VotingPowerSet != nil {
			// delete the old retained key, because the key currently holding the voting power set
			// will become the latest retained key.
			if retainedKey != nil {
				store.Delete(retainedKey)
			}
			retainedKey = iterator.Key()
		} else {
			store.Delete(iterator.Key())
		}
	}
	return nil
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

func (k *Keeper) SetLastChangedHeight(ctx sdk.Context, avsAddr string, lastChangeHeight int64) error {
	opFunc := func(helper *types.SnapshotHelper) error {
		helper.LastChangedHeight = lastChangeHeight
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
