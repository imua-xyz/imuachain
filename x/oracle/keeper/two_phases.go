package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/imua-xyz/imuachain/x/oracle/types"
)

// SetNextPieceIndex sets the next-piece-index of feederID for 'node-recovery'
func (k Keeper) SetNextPieceIndex(ctx sdk.Context, feederID uint64, pieceIndex uint32) {
	store := ctx.KVStore(k.storeKey)
	key := types.TwoPhasesFeederKey(feederID)
	store.Set(key, types.Uint32Bytes(pieceIndex))
}

func (k Keeper) ClearNextPieceIndex(ctx sdk.Context, feederID uint64) {
	store := ctx.KVStore(k.storeKey)
	key := types.TwoPhasesFeederKey(feederID)
	store.Delete(key)
}

// NextPieceIndexByFeederID read directly from memory and return the next-piece-index of input feederID
func (k Keeper) NextPieceIndexByFeederID(ctx sdk.Context, feederID uint64) (uint32, bool) {
	return k.FeederManager.NextPieceIndexByFeederID(feederID)
}

// CheckAndIncreasePieceIndex checks and increase the 'nextPieceIndex' of a specific validator and feederID
// valid pieceIndex starts from 0
// returns (nextPieceIndexAfterIncreased, error)
func (k Keeper) CheckAndIncreaseNextPieceIndex(ctx sdk.Context, validator string, feederID uint64, nextPieceIndex uint32) (uint32, error) {
	maxPieceIndex, ok := k.FeederManager.MaxPieceIndexForTokenFeederID(feederID)
	if !ok {
		return 0, fmt.Errorf("max piece index not found for feederID: %d", feederID)
	}
	if nextPieceIndex > maxPieceIndex {
		return 0, fmt.Errorf("piece_index_check_failed: feederID:%d, max_piece_index:%d, got:%d", feederID, maxPieceIndex, nextPieceIndex)
	}
	store := ctx.KVStore(k.storeKey)
	key := types.TwoPhasesValidatorPieceKey(validator, feederID)
	bz := store.Get(key)
	if bz == nil {
		return 0, fmt.Errorf("piece_index_check_failed: validator_not_found: validator:%s, feeder_id:%d", validator, feederID)
	}
	expectedPieceIndex := types.BytesToUint32(bz)
	if nextPieceIndex != expectedPieceIndex {
		return 0, fmt.Errorf("piece_index_check_failed: non_conseecutive: expected:%d, recived:%d", expectedPieceIndex, nextPieceIndex)
	}
	store.Set(key, types.Uint32Bytes(nextPieceIndex+1))
	return nextPieceIndex + 1, nil
}

func (k Keeper) Setup2ndPhaseNextPieceIndex(ctx sdk.Context, feederID uint64, validators []string) {
	store := ctx.KVStore(k.storeKey)
	// 1. set nextPieceIndex for feederID, first piece index is 0
	store.Set(types.TwoPhasesFeederKey(feederID), types.Uint32Bytes(0))

	// 2. set nextPieceIndex for all activeValidators, fisr piece index is 0
	for _, validator := range validators {
		store.Set(types.TwoPhasesValidatorPieceKey(validator, feederID), types.Uint32Bytes(0))
	}
}

func (k Keeper) Clear2ndPhaseNextPieceIndex(ctx sdk.Context, feederID uint64, validators []string) {
	// TODO(leonz): implement me
	// 1. remove feederID->nextPieceIndex, 2. remove validator/feederID->nextPieceIndex
	store := ctx.KVStore(k.storeKey)
	store.Delete(types.TwoPhasesFeederKey(feederID))
	// 2. remove nextPieceIndex for validators
	for _, validator := range validators {
		store.Delete(types.TwoPhasesValidatorPieceKey(validator, feederID))
	}

}

// set feederID/pieceIndex -> rawData(]byte)

// read feederID/pieceIndex -> rawData([]byte)

// read all pieces of rawData from feederID/

// clear all rawData for feederID/

// update feederID -> proof([][]byte)

// clear the whole proof under feederID

// clear feederID/, clear:
// 1. rawData
// 2. proof
func (k Keeper) Clear2ndPhases(ctx sdk.Context, feederID uint64) {

}
