package keeper

import (
	"fmt"
	"math"
	"strings"

	epochtypes "github.com/imua-xyz/imuachain/x/epochs/types"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/imua-xyz/imuachain/utils"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/imua-xyz/imuachain/x/delegation/types"
)

// AllUndelegations function returns all the undelegation records in the module.
// It is used during `ExportGenesis` to export the undelegation records.
func (k Keeper) AllUndelegations(ctx sdk.Context) (undelegations []types.UndelegationAndHoldCount, err error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixUndelegationInfo)
	iterator := sdk.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	ret := make([]types.UndelegationAndHoldCount, 0)
	for ; iterator.Valid(); iterator.Next() {
		var undelegation types.UndelegationRecord
		k.cdc.MustUnmarshal(iterator.Value(), &undelegation)
		holdCount := k.GetUndelegationHoldCount(ctx, iterator.Key())
		ret = append(ret, types.UndelegationAndHoldCount{
			Undelegation: &undelegation,
			HoldCount:    holdCount,
		})
	}
	return ret, nil
}

// SetUndelegationRecords stores the provided undelegation records.
// The records are stored with 3 different keys:
// (1) recordKey == blockNumber + undelegationID + txHash + operatorAddress => record
// (2) operatorAccAddr + stakerID + assetID + undelegationID => recordKey
// (3) epochIdentifierLength + completedEpochIdentifier + completedEpochNumber + UndelegationID => recordKey
// If a record exists with the same key, it will be overwritten; however, that is not a big
// concern since the lzNonce and txHash are unique for each record.
func (k *Keeper) SetUndelegationRecords(ctx sdk.Context, isGenesis bool, records []types.UndelegationAndHoldCount) error {
	singleRecordStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixUndelegationInfo)
	stakerUndelegationStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixStakerUndelegationInfo)
	pendingUndelegationStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixPendingUndelegations)
	store := ctx.KVStore(k.storeKey)

	for i := range records {
		undelegation := records[i].Undelegation
		if undelegation.CompletedEpochIdentifier != epochtypes.NullEpochIdentifier {
			epochInfo, exist := k.epochsKeeper.GetEpochInfo(ctx, undelegation.CompletedEpochIdentifier)
			if !exist {
				return errorsmod.Wrapf(types.ErrEpochIdentifierNotExist, "identifier:%s", undelegation.CompletedEpochIdentifier)
			}
			if undelegation.CompletedEpochNumber < epochInfo.CurrentEpoch {
				return errorsmod.Wrapf(types.ErrInvalidCompletionEpoch, "epochIdentifier:%s,currentEpochNumber:%d,CompleteEpochNumber:%d", undelegation.CompletedEpochIdentifier, epochInfo.CurrentEpoch, undelegation.CompletedEpochNumber)
			}
		}
		bz := k.cdc.MustMarshal(undelegation)
		// todo: check if the following state can only be set once?
		singleRecKey := types.GetUndelegationRecordKey(undelegation.BlockNumber, undelegation.UndelegationId, undelegation.TxHash, undelegation.OperatorAddr)
		singleRecordStore.Set(singleRecKey, bz)

		stakerKey := types.GetStakerUndelegationRecordKey(undelegation.StakerId, undelegation.AssetId, undelegation.UndelegationId)
		stakerUndelegationStore.Set(stakerKey, singleRecKey)

		pendingUndelegationKey := types.GetPendingUndelegationRecordKey(undelegation.CompletedEpochIdentifier, undelegation.CompletedEpochNumber, undelegation.UndelegationId)
		pendingUndelegationStore.Set(pendingUndelegationKey, singleRecKey)

		if isGenesis {
			// set on-hold count
			store.Set(types.GetUndelegationOnHoldKey(singleRecKey), sdk.Uint64ToBigEndian(records[i].HoldCount))
		}
	}
	return nil
}

// DeleteUndelegationRecord deletes the undelegation record from the module.
// The deletion is performed from all the 3 stores.
func (k *Keeper) DeleteUndelegationRecord(ctx sdk.Context, record *types.UndelegationRecord) error {
	singleRecordStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixUndelegationInfo)
	stakerUndelegationStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixStakerUndelegationInfo)
	pendingUndelegationStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixPendingUndelegations)

	singleRecKey := types.GetUndelegationRecordKey(record.BlockNumber, record.UndelegationId, record.TxHash, record.OperatorAddr)
	singleRecordStore.Delete(singleRecKey)

	stakerKey := types.GetStakerUndelegationRecordKey(record.StakerId, record.AssetId, record.UndelegationId)
	stakerUndelegationStore.Delete(stakerKey)

	pendingUndelegationKey := types.GetPendingUndelegationRecordKey(record.CompletedEpochIdentifier, record.CompletedEpochNumber, record.UndelegationId)
	pendingUndelegationStore.Delete(pendingUndelegationKey)

	store := ctx.KVStore(k.storeKey)
	// delegate on-hold record for the undelegation
	store.Delete(types.GetUndelegationOnHoldKey(singleRecKey))

	// emit an event to track the undelegation record identifiers.
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeUndelegationMatured,
			// the amount is the only thing that changes from the original record creation time
			// technically we are tracking this number via slashing, but best to include it here
			// as well.
			sdk.NewAttribute(types.AttributeKeyAmount, record.ActualCompletedAmount.String()),
			// everything else can be looked up from the original record identifier
			sdk.NewAttribute(types.AttributeKeyRecordID, hexutil.Encode(record.GetKey())),
		),
	)

	return nil
}

// GetUndelegationRecords returns the undelegation records for the provided record keys.
func (k *Keeper) GetUndelegationRecords(ctx sdk.Context, singleRecordKeys [][]byte) (record []*types.UndelegationAndHoldCount, err error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixUndelegationInfo)
	ret := make([]*types.UndelegationAndHoldCount, 0)
	for _, singleRecordKey := range singleRecordKeys {
		value := store.Get(singleRecordKey)
		if value == nil {
			return nil, errorsmod.Wrap(types.ErrNoKeyInTheStore, fmt.Sprintf("undelegation record key doesn't exist: key is %s", singleRecordKey))
		}
		undelegationRecord := types.UndelegationRecord{}
		k.cdc.MustUnmarshal(value, &undelegationRecord)
		holdCount := k.GetUndelegationHoldCount(ctx, singleRecordKey)
		ret = append(ret, &types.UndelegationAndHoldCount{
			Undelegation: &undelegationRecord,
			HoldCount:    holdCount,
		})
	}
	return ret, nil
}

// IterateUndelegationsByOperator iterates over the undelegation records belonging to the
// provided operator and filter. If the filter is non-nil, it will only iterate over the
// records for which the block height is greater than or equal to the filter.
func (k *Keeper) IterateUndelegationsByOperator(
	ctx sdk.Context, operator string, heightFilter *uint64, isUpdate bool,
	opFunc func(undelegation *types.UndelegationRecord) error,
) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixUndelegationInfo)
	operatorAccAddress := sdk.MustAccAddressFromBech32(operator)
	iterator := sdk.KVStorePrefixIterator(store, operatorAccAddress)
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

// GetStakerUndelegationRecKeys returns the undelegation record keys corresponding to the provided
// staker and asset.
func (k *Keeper) GetStakerUndelegationRecKeys(ctx sdk.Context, stakerID, assetID string) (recordKeyList [][]byte, err error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixStakerUndelegationInfo)
	iterator := sdk.KVStorePrefixIterator(store, []byte(strings.Join([]string{stakerID, assetID}, "/")))
	defer iterator.Close()

	ret := make([][]byte, 0)
	for ; iterator.Valid(); iterator.Next() {
		ret = append(ret, iterator.Value())
	}
	return ret, nil
}

func (k *Keeper) GetUndelegationRecKey(ctx sdk.Context, stakerID, assetID string, undelegationID uint64) (recordKey []byte, err error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixStakerUndelegationInfo)
	recordKey = store.Get(types.GetStakerUndelegationRecordKey(stakerID, assetID, undelegationID))
	return recordKey, nil
}

// GetStakerUndelegationRecords returns the undelegation records for the provided staker and asset.
func (k *Keeper) GetStakerUndelegationRecords(ctx sdk.Context, stakerID, assetID string) (records []*types.UndelegationAndHoldCount, err error) {
	recordKeys, err := k.GetStakerUndelegationRecKeys(ctx, stakerID, assetID)
	if err != nil {
		return nil, err
	}

	return k.GetUndelegationRecords(ctx, recordKeys)
}

// IterateUndelegationsByStakerAndAsset iterates over the undelegation records belonging to the provided
// stakerID and assetID. If the isUpdate is true, the undelegation record will be updated after the
// operation is performed.
func (k *Keeper) IterateUndelegationsByStakerAndAsset(
	ctx sdk.Context, stakerID, assetID string, isUpdate bool,
	opFunc func(undelegationKey []byte, undelegation *types.UndelegationRecord) (bool, error),
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
		isBreak, err := opFunc(iterator.Value(), &undelegation)
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

func (k *Keeper) GetUnCompletableUndelegations(ctx sdk.Context, epochIdentifier string, epochNumber int64) ([]*types.UndelegationAndHoldCount, error) {
	records := make([]*types.UndelegationAndHoldCount, 0)
	expiredUndelegationOpFunc := func(recordKey []byte, record *types.UndelegationRecord) error {
		holdCount := k.GetUndelegationHoldCount(ctx, recordKey)
		records = append(records, &types.UndelegationAndHoldCount{
			Undelegation: record,
			HoldCount:    holdCount,
		})
		return nil
	}
	err := k.IteratePendingUndelegations(ctx, false, epochIdentifier, epochNumber, expiredUndelegationOpFunc)
	if err != nil {
		return nil, err
	}
	return records, nil
}

// GetCompletableUndelegations returns the undelegation records scheduled to completed at the end
// of the block. The pending undelegations should be expired and aren't held
func (k *Keeper) GetCompletableUndelegations(ctx sdk.Context) ([]*types.UndelegationRecord, error) {
	records := make([]*types.UndelegationRecord, 0)
	expiredUndelegationOpFunc := func(_ []byte, record *types.UndelegationRecord) error {
		records = append(records, record)
		return nil
	}

	// For the null epoch, we set `types.NullEpochNumber + 1` as the virtual current epoch number,
	// allowing the related undelegations to be completed at the end of the block.
	err := k.IteratePendingUndelegations(ctx, true, epochtypes.NullEpochIdentifier, epochtypes.NullEpochNumber+1, expiredUndelegationOpFunc)
	if err != nil {
		return nil, err
	}
	// iterate all pending undelegations across multiple epochs.
	allEpochs := k.epochsKeeper.AllEpochInfos(ctx)
	for _, epochInfo := range allEpochs {
		err := k.IteratePendingUndelegations(ctx, true, epochInfo.Identifier, epochInfo.CurrentEpoch, expiredUndelegationOpFunc)
		if err != nil {
			return nil, err
		}
	}
	return records, nil
}

// IteratePendingUndelegations : This function iterates through all undelegations.
// If the `isCompletable` flag is true, it retrieves all completable undelegations,
// including the undelegations that are expired and not held.
// If the `isCompletable` flag is false, it retrieves all undelegations that aren't completable,
// including the undelegations that are unexpired and expired but held by the other processes.
// The iteration leverages ascending or descending order to quickly fetch results
// because undelegations are stored in the order of their epoch numbers.
func (k *Keeper) IteratePendingUndelegations(
	ctx sdk.Context, isCompletable bool, epochIdentifier string, currentEpoch int64,
	opFunc func(recordKey []byte, undelegationRecord *types.UndelegationRecord) error,
) error {
	pendingUndelegationStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixPendingUndelegations)
	undelegationStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixUndelegationInfo)

	prefix := utils.AppendMany(
		sdk.Uint64ToBigEndian(uint64(len(epochIdentifier))),
		[]byte(epochIdentifier))
	iterator := sdk.KVStorePrefixIterator(pendingUndelegationStore, prefix)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		pendingUndelegationKeys, err := types.ParsePendingUndelegationKey(iterator.Key())
		if err != nil {
			return err
		}
		if isCompletable {
			// Due to the current implementation, the completion of undelegation is triggered
			// by per-block checks rather than using epoch hooks. As a result, when the epoch
			// number to be completed equals the current epoch number, the undelegation remains
			// in a pending state. It will only be completed after the current epoch ends, specifically
			// in the first block of the next epoch.
			// This logic might be changed if we chose epochHook to complete the undelgation in the future.
			if pendingUndelegationKeys.EpochNumber >= uint64(currentEpoch) {
				// These pending undelegations aren't expired, break the iteration
				break
			}
			if k.GetUndelegationHoldCount(ctx, iterator.Value()) > 0 {
				// The expired pending undelegation is held, so do not complete it;
				// then, continue addressing the other expired pending undelegations.
				k.Logger(ctx).Info("IteratePendingUndelegations: the expired pending undelegation is held",
					"recordKey", hexutil.Encode(iterator.Value()))
				continue
			}
		} else if pendingUndelegationKeys.EpochNumber < uint64(currentEpoch) &&
			k.GetUndelegationHoldCount(ctx, iterator.Value()) == 0 {
			// These pending undelegations are expired and not held
			k.Logger(ctx).Info("IteratePendingUndelegations: the this undelegation is expired but not held",
				"recordKey", hexutil.Encode(iterator.Value()))
			continue
		}

		// call opFunc to execute some operations for the expired pending undelegations
		value := undelegationStore.Get(iterator.Value())
		if value == nil {
			return errorsmod.Wrap(types.ErrNoKeyInTheStore, fmt.Sprintf("undelegation record key doesn't exist: key is %x", iterator.Value()))
		}
		undelegation := types.UndelegationRecord{}
		k.cdc.MustUnmarshal(value, &undelegation)
		err = opFunc(iterator.Value(), &undelegation)
		if err != nil {
			return err
		}
	}
	return nil
}

// IncrementUndelegationHoldCount increments the hold count for the undelegation record key.
func (k Keeper) IncrementUndelegationHoldCount(ctx sdk.Context, recordKey []byte) error {
	prev := k.GetUndelegationHoldCount(ctx, recordKey)
	if prev == math.MaxUint64 {
		return types.ErrCannotIncHoldCount
	}
	now := prev + 1
	store := ctx.KVStore(k.storeKey)
	store.Set(types.GetUndelegationOnHoldKey(recordKey), sdk.Uint64ToBigEndian(now))
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeUndelegationHoldCountChanged,
			sdk.NewAttribute(types.AttributeKeyRecordID, hexutil.Encode(recordKey)),
			sdk.NewAttribute(types.AttributeKeyHoldCount, fmt.Sprintf("%d", now)),
		),
	)
	return nil
}

// GetUndelegationHoldCount returns the hold count for the undelegation record key.
func (k *Keeper) GetUndelegationHoldCount(ctx sdk.Context, recordKey []byte) uint64 {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.GetUndelegationOnHoldKey(recordKey))
	return sdk.BigEndianToUint64(bz)
}

// DecrementUndelegationHoldCount decrements the hold count for the undelegation record key.
func (k Keeper) DecrementUndelegationHoldCount(ctx sdk.Context, recordKey []byte) error {
	prev := k.GetUndelegationHoldCount(ctx, recordKey)
	if prev == 0 {
		return types.ErrCannotDecHoldCount
	}
	now := prev - 1
	store := ctx.KVStore(k.storeKey)
	store.Set(types.GetUndelegationOnHoldKey(recordKey), sdk.Uint64ToBigEndian(now))
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeUndelegationHoldCountChanged,
			sdk.NewAttribute(types.AttributeKeyRecordID, hexutil.Encode(recordKey)),
			sdk.NewAttribute(types.AttributeKeyHoldCount, fmt.Sprintf("%d", now)),
		),
	)
	return nil
}

// IncrementLastUndelegationID increments the global undelegation ID.
func (k Keeper) IncrementLastUndelegationID(ctx sdk.Context) error {
	prev := k.GetLastUndelegationID(ctx)
	if prev == math.MaxUint64 {
		return types.ErrCannotIncUndelegationID
	}
	now := prev + 1
	store := ctx.KVStore(k.storeKey)
	store.Set(types.KeyPrefixLastUndelegationID, sdk.Uint64ToBigEndian(now))
	return nil
}

// SetLastUndelegationID sets the global undelegation ID.
func (k Keeper) SetLastUndelegationID(ctx sdk.Context, lastUndelegationID uint64) error {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.KeyPrefixLastUndelegationID, sdk.Uint64ToBigEndian(lastUndelegationID))
	return nil
}

// GetLastUndelegationID returns the global undelegation ID.
func (k *Keeper) GetLastUndelegationID(ctx sdk.Context) uint64 {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.KeyPrefixLastUndelegationID)
	if bz == nil {
		// use 0 as the initial undelegation ID
		return 0
	}
	return sdk.BigEndianToUint64(bz)
}
