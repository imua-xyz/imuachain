package keeper

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	types "github.com/imua-xyz/imuachain/x/appchain/coordinator/types"
)

// TODO: this file should be in the x/avs keeper instead.

// SetSubSlashFractionDowntime sets the sub slash fraction downtime for a chain
func (k Keeper) SetSubSlashFractionDowntime(ctx sdk.Context, chainID string, fraction string) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.SubSlashFractionDowntimeKey(chainID), []byte(fraction))
}

// GetSubSlashFractionDowntime gets the sub slash fraction downtime for a chain
func (k Keeper) GetSubSlashFractionDowntime(ctx sdk.Context, chainID string) string {
	store := ctx.KVStore(k.storeKey)
	key := types.SubSlashFractionDowntimeKey(chainID)
	return string(store.Get(key))
}

// SetSubSlashFractionDoubleSign sets the sub slash fraction double sign for a chain
func (k Keeper) SetSubSlashFractionDoubleSign(ctx sdk.Context, chainID string, fraction string) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.SubSlashFractionDoubleSignKey(chainID), []byte(fraction))
}

// GetSubSlashFractionDoubleSign gets the sub slash fraction double sign for a chain
func (k Keeper) GetSubSlashFractionDoubleSign(ctx sdk.Context, chainID string) string {
	store := ctx.KVStore(k.storeKey)
	key := types.SubSlashFractionDoubleSignKey(chainID)
	return string(store.Get(key))
}

// SetSubDowntimeJailDuration sets the sub downtime jail duration for a chain
func (k Keeper) SetSubDowntimeJailDuration(ctx sdk.Context, chainID string, duration time.Duration) {
	store := ctx.KVStore(k.storeKey)
	// duration is always positive
	store.Set(types.SubDowntimeJailDurationKey(chainID), sdk.Uint64ToBigEndian(uint64(duration)))
}

// GetSubDowntimeJailDuration gets the sub downtime jail duration for a chain
func (k Keeper) GetSubDowntimeJailDuration(ctx sdk.Context, chainID string) time.Duration {
	store := ctx.KVStore(k.storeKey)
	key := types.SubDowntimeJailDurationKey(chainID)
	return time.Duration(sdk.BigEndianToUint64(store.Get(key)))
}
