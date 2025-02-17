package keeper

import (
	"strconv"

	"github.com/ExocoreNetwork/exocore/x/dogfood/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// GetUnbondingCompletionEpoch returns the epoch at the end of which
// an unbonding triggered in this epoch will be completed.
func (k Keeper) GetUnbondingCompletionEpoch(
	ctx sdk.Context,
) int64 {
	params := k.GetDogfoodParams(ctx)
	epochInfo, _ := k.epochsKeeper.GetEpochInfo(
		ctx, params.EpochIdentifier,
	)
	// if i execute the transaction at epoch 5, the vote power change
	// goes into effect at the beginning of epoch 6. the information
	// should be held for 7 epochs, so it should be deleted at the
	// beginning of epoch 13 or the end of epoch 12.
	return epochInfo.CurrentEpoch + int64(params.EpochsUntilUnbonded) // #nosec G701
}

// AppendUndelegationsToMature stores that the undelegation with recordKey should be
// released at the end of the provided epoch.
func (k Keeper) AppendUndelegationToMature(
	ctx sdk.Context, epoch int64, recordKey []byte,
) {
	prev := k.GetUndelegationsToMature(ctx, epoch)
	next := types.UndelegationRecordKeys{
		List: append(prev, recordKey),
	}
	k.setUndelegationsToMature(ctx, epoch, next)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeUndelegationMaturityScheduled,
			sdk.NewAttribute(types.AttributeKeyEpoch, strconv.FormatInt(epoch, 10)),
			sdk.NewAttribute(types.AttributeKeyRecordID, hexutil.Encode(recordKey)),
		),
	)
}

// GetUndelegationsToMature returns all undelegation entries that should be released
// at the end of the provided epoch.
func (k Keeper) GetUndelegationsToMature(
	ctx sdk.Context, epoch int64,
) [][]byte {
	store := ctx.KVStore(k.storeKey)
	key, _ := types.UnbondingReleaseMaturityKey(epoch)
	bz := store.Get(key)
	if bz == nil {
		return [][]byte{}
	}
	var res types.UndelegationRecordKeys
	if err := res.Unmarshal(bz); err != nil {
		// should never happen
		panic(err)
	}
	return res.GetList()
}

// ClearUndelegationsToMature is a pruning method which is called after we mature
// the undelegation entries.
func (k Keeper) ClearUndelegationsToMature(
	ctx sdk.Context, epoch int64,
) {
	store := ctx.KVStore(k.storeKey)
	key, _ := types.UnbondingReleaseMaturityKey(epoch)
	store.Delete(key)
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeUndelegationsMatured,
			sdk.NewAttribute(types.AttributeKeyEpoch, strconv.FormatInt(epoch, 10)),
		),
	)
}

// setUndelegationsToMature sets all undelegation entries that should be released
// at the end of the provided epoch.
func (k Keeper) setUndelegationsToMature(
	ctx sdk.Context, epoch int64, undelegationRecords types.UndelegationRecordKeys,
) {
	store := ctx.KVStore(k.storeKey)
	key, _ := types.UnbondingReleaseMaturityKey(epoch)
	val, err := undelegationRecords.Marshal()
	if err != nil {
		panic(err)
	}
	store.Set(key, val)
}

// GetAllUndelegationsToMature gets a list of epochs and the corresponding undelegation record
// keys which are scheduled to mature at the end of that epoch. It is ordered, first by the
// epoch and then by the record key's bytes.
func (k Keeper) GetAllUndelegationsToMature(ctx sdk.Context) []types.EpochToUndelegationRecordKeys {
	store := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStorePrefixIterator(store, []byte{types.OptOutsToFinishBytePrefix})
	defer iterator.Close()

	res := []types.EpochToUndelegationRecordKeys{}

	for ; iterator.Valid(); iterator.Next() {
		epoch, _ := types.SafeUint64ToInt64(sdk.BigEndianToUint64(iterator.Key()[1:]))
		var recordKeys types.UndelegationRecordKeys
		k.cdc.MustUnmarshal(iterator.Value(), &recordKeys)
		subRes := []string{}
		for _, recordKey := range recordKeys.GetList() {
			subRes = append(subRes, hexutil.Encode(recordKey))
		}
		res = append(res, types.EpochToUndelegationRecordKeys{
			Epoch:                  epoch,
			UndelegationRecordKeys: subRes,
		})
	}

	return res
}

// GetUndelegationMaturityEpoch gets the maturity epoch for the undelegation record.
func (k Keeper) GetUndelegationMaturityEpoch(
	ctx sdk.Context, recordKey []byte,
) (int64, bool) {
	store := ctx.KVStore(k.storeKey)
	key := types.UndelegationMaturityEpochKey(recordKey)
	bz := store.Get(key)
	if bz == nil {
		return 0, false
	}
	epoch := sdk.BigEndianToUint64(bz)
	return types.SafeUint64ToInt64(epoch)
}

// SetUndelegationMaturityEpoch sets the maturity epoch for the undelegation record.
func (k Keeper) SetUndelegationMaturityEpoch(
	ctx sdk.Context, recordKey []byte, epoch int64,
) {
	store := ctx.KVStore(k.storeKey)
	key := types.UndelegationMaturityEpochKey(recordKey)
	uepoch, _ := types.SafeInt64ToUint64(epoch)
	bz := sdk.Uint64ToBigEndian(uepoch)
	store.Set(key, bz)
}

// ClearUndelegationMaturityEpoch clears the maturity epoch for the undelegation record.
func (k Keeper) ClearUndelegationMaturityEpoch(
	ctx sdk.Context, recordKey []byte,
) {
	store := ctx.KVStore(k.storeKey)
	key := types.UndelegationMaturityEpochKey(recordKey)
	store.Delete(key)
}
