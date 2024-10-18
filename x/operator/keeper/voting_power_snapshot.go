package keeper

import (
	assetstype "github.com/ExocoreNetwork/exocore/x/assets/types"
	"github.com/ExocoreNetwork/exocore/x/operator/types"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"strconv"
)

func (k *Keeper) StoreVotingPowerSnapshot(ctx sdk.Context, avsAddr, epochIdentifier string, epochNumber int64, height *int64, snapshot *types.VotingPowerSnapshot) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixVotingPowerSnapshot)
	key := assetstype.GetJoinedStoreKey(avsAddr, epochIdentifier, strconv.FormatInt(epochNumber, 10))
	if height != nil && *height >= 0 {
		key = assetstype.GetJoinedStoreKey(string(key), strconv.FormatInt(*height, 10))
	}
	bz := k.cdc.MustMarshal(snapshot)
	store.Set(key, bz)
	return nil
}

func (k *Keeper) LoadVotingPowerSnapshot(ctx sdk.Context, avsAddr, epochIdentifier string, epochNumber int64, height *int64) (*types.VotingPowerSnapshot, error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixVotingPowerSnapshot)
	var ret types.VotingPowerSnapshot
	key := assetstype.GetJoinedStoreKey(avsAddr, epochIdentifier, strconv.FormatInt(epochNumber, 10))
	if height != nil && *height >= 0 {
		key = assetstype.GetJoinedStoreKey(string(key), strconv.FormatInt(*height, 10))
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
