package keeper

import (
	"time"

	"github.com/ExocoreNetwork/exocore/x/oracle/types"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gogotypes "github.com/cosmos/gogoproto/types"
)

// InitValidatorReportInfo creates a new item for a first-seen validator to track their performance
func (k Keeper) InitValidatorReportInfo(ctx sdk.Context, validator string, height int64) {
	store := ctx.KVStore(k.storeKey)
	key := types.SlashingValidatorReportInfoKey(validator)
	if !store.Has(key) {
		// set the record for validator to tracking performance of oracle service
		reportInfo := &types.ValidatorReportInfo{
			Address:     validator,
			StartHeight: height,
		}
		bz := k.cdc.MustMarshal(reportInfo)
		store.Set(key, bz)
	}
}

// SetValidatorReportInfo sets the reporting info for a validator
func (k Keeper) SetValidatorReportInfo(ctx sdk.Context, validator string, info types.ValidatorReportInfo) {
	store := ctx.KVStore(k.storeKey)
	bz := k.cdc.MustMarshal(&info)
	store.Set(types.SlashingValidatorReportInfoKey(validator), bz)
}

// GetValidatorReportInfo retruns the ValidatorReportInfo for a specific validator
func (k Keeper) GetValidatorReportInfo(ctx sdk.Context, validator string) (info types.ValidatorReportInfo, found bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.SlashingValidatorReportInfoKey(validator))
	if bz == nil {
		return
	}
	k.cdc.MustUnmarshal(bz, &info)
	found = true
	return
}

// SetValidatorMissedBlockBitArray sets the bit that checks if the validator has
// missed a round to report price in the current window
func (k Keeper) SetValidatorMissedRoundBitArray(ctx sdk.Context, validator string, index uint64, missed bool) {
	store := ctx.KVStore(k.storeKey)
	bz := k.cdc.MustMarshal(&gogotypes.BoolValue{Value: missed})
	store.Set(types.SlashingMissedBitArrayKey(validator, index), bz)
}

// GetValidatorMissedBlocks returns array of missed rounds for given validator
func (k Keeper) GetValidatorMissedRoundBitArray(ctx sdk.Context, validator string, index uint64) bool {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.SlashingMissedBitArrayKey(validator, index))
	if bz == nil {
		return false
	}
	var missed gogotypes.BoolValue
	k.cdc.MustUnmarshal(bz, &missed)
	return missed.Value
}

// GetReportedBlocksWindow sliding window for reporting slashing
func (k Keeper) GetReportedRoundsWindow(ctx sdk.Context) int64 {
	return k.GetParams(ctx).Slashing.ReportedRoundsWindow
}

// GetSlashFractionMiss fraction of power slashed for missed rounds
func (k Keeper) GetSlashFractionMiss(ctx sdk.Context) (res sdk.Dec) {
	return k.GetParams(ctx).Slashing.SlashFractionMiss
}

// GetSlashFractionMiss fraction of power slashed for missed rounds
func (k Keeper) GetSlashFractionMalicious(ctx sdk.Context) (res sdk.Dec) {
	return k.GetParams(ctx).Slashing.SlashFractionMalicious
}

// GetMinReportedPerWindow minimum blocks repored prices per window
func (k Keeper) GetMinReportedPerWindow(ctx sdk.Context) int64 {
	params := k.GetParams(ctx)
	reportedRoundsWindow := k.GetReportedRoundsWindow(ctx)

	// NOTE: RoundInt64 will never panic as minReportedPerWindow is
	//       less than 1.
	return params.Slashing.MinReportedPerWindow.MulInt64(reportedRoundsWindow).RoundInt64()
}

// MissJailDuration miss unbond duration
func (k Keeper) GetMissJailDuration(ctx sdk.Context) (res time.Duration) {
	return k.GetParams(ctx).Slashing.OracleMissJailDuration
}

// MissJailDuration miss unbond duration
func (k Keeper) GetMaliciousJailDuration(ctx sdk.Context) (res time.Duration) {
	return k.GetParams(ctx).Slashing.OracleMaliciousJailDuration
}

// IterateValidatorReportedInfos iterates over the stored reportInfo
// and performs a callback function
func (k Keeper) IterateValidatorReportInfos(ctx sdk.Context, handler func(address string, reportInfo types.ValidatorReportInfo) (stop bool)) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.ValidatorReportInfoPrefix)
	iterator := sdk.KVStorePrefixIterator(store, []byte{})
	for ; iterator.Valid(); iterator.Next() {
		address := string(iterator.Key())
		var info types.ValidatorReportInfo
		k.cdc.MustUnmarshal(iterator.Value(), &info)
		if handler(address, info) {
			break
		}
	}
	iterator.Close()
}

func (k Keeper) IterateValidatorMissedRoundBitArray(ctx sdk.Context, validator string, handler func(index int64, missed bool) (stop bool)) {
	//	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.SlashingMissedBitArrayPrefix(validator))
	store := ctx.KVStore(k.storeKey)
	index := int64(0)
	// Array may be sparse
	for ; index < k.GetReportedRoundsWindow(ctx); index++ {
		var missed gogotypes.BoolValue
		bz := store.Get(types.SlashingMissedBitArrayKey(validator, uint64(index)))
		if bz == nil {
			continue
		}

		k.cdc.MustUnmarshal(bz, &missed)
		if handler(index, missed.Value) {
			break
		}
	}
}

func (k Keeper) GetValidatorMissedRounds(ctx sdk.Context, address string) []*types.MissedRound {
	missedRounds := []*types.MissedRound{}
	k.IterateValidatorMissedRoundBitArray(ctx, address, func(index int64, missed bool) (stop bool) {
		missedRounds = append(missedRounds, types.NewMissedRound(index, missed))
		return false
	})
	return missedRounds
}

// clearValidatorMissedBlockBitArray deletes every instance of ValidatorMissedBlockBitArray in the store
func (k Keeper) ClearValidatorMissedRoundBitArray(ctx sdk.Context, validator string) {
	store := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStorePrefixIterator(store, types.SlashingMissedBitArrayPrefix(validator))
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		store.Delete(iterator.Key())
	}
}
