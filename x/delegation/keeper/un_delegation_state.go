package keeper

import (
	"fmt"
	"math"
	"strings"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/imua-xyz/imuachain/x/delegation/types"
)

func (k Keeper) AllUndelegations(ctx sdk.Context) (undelegations []types.UndelegationRecord, err error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixUndelegationInfo)
	iterator := sdk.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	ret := make([]types.UndelegationRecord, 0)
	for ; iterator.Valid(); iterator.Next() {
		var undelegation types.UndelegationRecord
		k.cdc.MustUnmarshal(iterator.Value(), &undelegation)
		ret = append(ret, undelegation)
	}
	return ret, nil
}

// SetUndelegationRecords function saves the undelegation records to be handled when the handle time expires.
// When we save the undelegation records, we save them in three kv stores which are `KeyPrefixUndelegationInfo` `KeyPrefixStakerUndelegationInfo` and `KeyPrefixPendingUndelegations`
func (k *Keeper) SetUndelegationRecords(ctx sdk.Context, records []types.UndelegationRecord) error {
	singleRecordStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixUndelegationInfo)
	stakerUndelegationStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixStakerUndelegationInfo)
	pendingUndelegationStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixPendingUndelegations)
	currentHeight := ctx.BlockHeight()
	for i := range records {
		record := records[i]
		if record.CompleteBlockNumber < uint64(currentHeight) {
			return errorsmod.Wrapf(types.ErrInvalidCompletedHeight, "currentHeight:%d,CompleteBlockNumber:%d", currentHeight, record.CompleteBlockNumber)
		}
		bz := k.cdc.MustMarshal(&record)
		// todo: check if the following state can only be set once?
		singleRecKey := types.GetUndelegationRecordKey(record.BlockNumber, record.LzTxNonce, record.TxHash, record.OperatorAddr)
		singleRecordStore.Set(singleRecKey, bz)

		stakerKey := types.GetStakerUndelegationRecordKey(record.StakerID, record.AssetID, record.LzTxNonce)
		stakerUndelegationStore.Set(stakerKey, singleRecKey)

		pendingUndelegationKey := types.GetPendingUndelegationRecordKey(record.CompleteBlockNumber, record.LzTxNonce)
		pendingUndelegationStore.Set(pendingUndelegationKey, singleRecKey)
	}
	return nil
}

func (k *Keeper) DeleteUndelegationRecord(ctx sdk.Context, record *types.UndelegationRecord) error {
	singleRecordStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixUndelegationInfo)
	stakerUndelegationStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixStakerUndelegationInfo)
	pendingUndelegationStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixPendingUndelegations)

	singleRecKey := types.GetUndelegationRecordKey(record.BlockNumber, record.LzTxNonce, record.TxHash, record.OperatorAddr)
	singleRecordStore.Delete(singleRecKey)

	stakerKey := types.GetStakerUndelegationRecordKey(record.StakerID, record.AssetID, record.LzTxNonce)
	stakerUndelegationStore.Delete(stakerKey)

	pendingUndelegationKey := types.GetPendingUndelegationRecordKey(record.CompleteBlockNumber, record.LzTxNonce)
	pendingUndelegationStore.Delete(pendingUndelegationKey)
	return nil
}

func (k *Keeper) SetSingleUndelegationRecord(ctx sdk.Context, record *types.UndelegationRecord) (recordKey []byte, err error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixUndelegationInfo)
	bz := k.cdc.MustMarshal(record)
	key := types.GetUndelegationRecordKey(record.BlockNumber, record.LzTxNonce, record.TxHash, record.OperatorAddr)
	store.Set(key, bz)
	return key, nil
}

// StorePendingUndelegationRecord add it to handle the delay of completing undelegation caused by onHoldCount
// In the event that the undelegation is held by another module, this function is used within the EndBlocker to increment the scheduled completion block number by 1.
// Then the completion time of the undelegation will be delayed to the next block.
func (k *Keeper) StorePendingUndelegationRecord(ctx sdk.Context, singleRecKey []byte, record *types.UndelegationRecord) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixPendingUndelegations)
	pendingUndelegationKey := types.GetPendingUndelegationRecordKey(record.CompleteBlockNumber, record.LzTxNonce)
	store.Set(pendingUndelegationKey, singleRecKey)
	return nil
}

func (k *Keeper) GetUndelegationRecords(ctx sdk.Context, singleRecordKeys []string) (record []*types.UndelegationRecord, err error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixUndelegationInfo)
	ret := make([]*types.UndelegationRecord, 0)
	for _, singleRecordKey := range singleRecordKeys {
		keyBytes := []byte(singleRecordKey)
		value := store.Get(keyBytes)
		if value == nil {
			return nil, errorsmod.Wrap(types.ErrNoKeyInTheStore, fmt.Sprintf("undelegation record key doesn't exist: key is %s", singleRecordKey))
		}
		undelegationRecord := types.UndelegationRecord{}
		k.cdc.MustUnmarshal(value, &undelegationRecord)
		ret = append(ret, &undelegationRecord)
	}
	return ret, nil
}

// IterateUndelegationsByOperator iterate the undelegation records according to the operator
// and height filter. If the heightFilter isn't nil, only return the undelegations that the
// created height is greater than or equal to the filter height.
func (k *Keeper) IterateUndelegationsByOperator(
	ctx sdk.Context, operator string, heightFilter *uint64, isUpdate bool,
	opFunc func(undelegation *types.UndelegationRecord) error,
) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixUndelegationInfo)
	iterator := sdk.KVStorePrefixIterator(store, []byte(operator))
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		if heightFilter != nil {
			keyFields, err := types.ParseUndelegationRecordKey(iterator.Key())
			if err != nil {
				return err
			}
			if keyFields.BlockHeight < *heightFilter {
				continue
			}
		}
		undelegation := types.UndelegationRecord{}
		k.cdc.MustUnmarshal(iterator.Value(), &undelegation)
		err := opFunc(&undelegation)
		if err != nil {
			return err
		}

		if isUpdate {
			bz := k.cdc.MustMarshal(&undelegation)
			store.Set(iterator.Key(), bz)
		}
	}
	return nil
}

func (k *Keeper) SetStakerUndelegationInfo(ctx sdk.Context, stakerID, assetID string, recordKey []byte, lzNonce uint64) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixStakerUndelegationInfo)
	key := types.GetStakerUndelegationRecordKey(stakerID, assetID, lzNonce)
	store.Set(key, recordKey)
	return nil
}

func (k *Keeper) GetStakerUndelegationRecKeys(ctx sdk.Context, stakerID, assetID string) (recordKeyList []string, err error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixStakerUndelegationInfo)
	iterator := sdk.KVStorePrefixIterator(store, []byte(strings.Join([]string{stakerID, assetID}, "/")))
	defer iterator.Close()

	ret := make([]string, 0)
	for ; iterator.Valid(); iterator.Next() {
		ret = append(ret, string(iterator.Value()))
	}
	return ret, nil
}

func (k *Keeper) GetStakerUndelegationRecords(ctx sdk.Context, stakerID, assetID string) (records []*types.UndelegationRecord, err error) {
	recordKeys, err := k.GetStakerUndelegationRecKeys(ctx, stakerID, assetID)
	if err != nil {
		return nil, err
	}

	return k.GetUndelegationRecords(ctx, recordKeys)
}

// IterateUndelegationsByStakerAndAsset iterate the undelegation records according to the stakerID and assetID.
func (k *Keeper) IterateUndelegationsByStakerAndAsset(
	ctx sdk.Context, stakerID, assetID string, isUpdate bool,
	opFunc func(undelegationKey string, undelegation *types.UndelegationRecord) (bool, error),
) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixStakerUndelegationInfo)
	iterator := sdk.KVStorePrefixIterator(store, types.IteratorPrefixForStakerAsset(stakerID, assetID))
	defer iterator.Close()
	undelegationInfoStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixUndelegationInfo)
	for ; iterator.Valid(); iterator.Next() {
		infoValue := undelegationInfoStore.Get(iterator.Value())
		if infoValue == nil {
			return errorsmod.Wrap(types.ErrNoKeyInTheStore, fmt.Sprintf("undelegation record key doesn't exist: key is %s", string(iterator.Value())))
		}
		undelegation := types.UndelegationRecord{}
		k.cdc.MustUnmarshal(infoValue, &undelegation)
		isBreak, err := opFunc(string(iterator.Value()), &undelegation)
		if err != nil {
			return err
		}
		if isUpdate {
			bz := k.cdc.MustMarshal(&undelegation)
			undelegationInfoStore.Set(iterator.Value(), bz)
		}
		if isBreak {
			break
		}
	}
	return nil
}

func (k *Keeper) SetPendingUndelegationInfo(ctx sdk.Context, height, lzNonce uint64, recordKey string) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixPendingUndelegations)
	key := types.GetPendingUndelegationRecordKey(height, lzNonce)
	store.Set(key, []byte(recordKey))
	return nil
}

func (k *Keeper) GetPendingUndelegationRecKeys(ctx sdk.Context, height uint64) (recordKeyList []string, err error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixPendingUndelegations)
	iterator := sdk.KVStorePrefixIterator(store, []byte(hexutil.EncodeUint64(height)))
	defer iterator.Close()

	ret := make([]string, 0)
	for ; iterator.Valid(); iterator.Next() {
		ret = append(ret, string(iterator.Value()))
	}
	return ret, nil
}

func (k *Keeper) GetPendingUndelegationRecords(ctx sdk.Context, height uint64) (records []*types.UndelegationRecord, err error) {
	recordKeys, err := k.GetPendingUndelegationRecKeys(ctx, height)
	if err != nil {
		return nil, err
	}
	if len(recordKeys) == 0 {
		records = make([]*types.UndelegationRecord, 0)
		return records, nil
	}
	// The states of records stored by KeyPrefixPendingUndelegations kvStore should always be IsPending,so using AllRecords as getType here is ok.
	return k.GetUndelegationRecords(ctx, recordKeys)
}

func (k Keeper) IncrementUndelegationHoldCount(ctx sdk.Context, recordKey []byte) error {
	prev := k.GetUndelegationHoldCount(ctx, recordKey)
	if prev == math.MaxUint64 {
		return types.ErrCannotIncHoldCount
	}
	now := prev + 1
	store := ctx.KVStore(k.storeKey)
	store.Set(types.GetUndelegationOnHoldKey(recordKey), sdk.Uint64ToBigEndian(now))
	return nil
}

func (k *Keeper) GetUndelegationHoldCount(ctx sdk.Context, recordKey []byte) uint64 {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.GetUndelegationOnHoldKey(recordKey))
	return sdk.BigEndianToUint64(bz)
}

func (k Keeper) DecrementUndelegationHoldCount(ctx sdk.Context, recordKey []byte) error {
	prev := k.GetUndelegationHoldCount(ctx, recordKey)
	if prev == 0 {
		return types.ErrCannotDecHoldCount
	}
	now := prev - 1
	store := ctx.KVStore(k.storeKey)
	store.Set(types.GetUndelegationOnHoldKey(recordKey), sdk.Uint64ToBigEndian(now))
	return nil
}
