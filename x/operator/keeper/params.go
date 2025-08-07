package keeper

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	operatortypes "github.com/imua-xyz/imuachain/x/operator/types"
)

// GetParams returns the params for the operator module.
func (k Keeper) GetParams(ctx sdk.Context) operatortypes.Params {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(operatortypes.KeyForParams())
	if bz == nil {
		k.Logger(ctx).Info("params not found, using default params")
		return operatortypes.DefaultParams()
	}
	var params operatortypes.Params
	k.cdc.MustUnmarshal(bz, &params)
	return params
}

// SetParams sets the params for the operator module.
// Calling functions must ensure that the params are valid.
func (k Keeper) SetParams(ctx sdk.Context, params operatortypes.Params) {
	store := ctx.KVStore(k.storeKey)
	bz := k.cdc.MustMarshal(&params)
	store.Set(operatortypes.KeyForParams(), bz)
}

// GetMinCommissionUpdateInterval returns the minimum interval between
// commission updates.
func (k Keeper) GetMinCommissionUpdateInterval(ctx sdk.Context) time.Duration {
	return k.GetParams(ctx).MinCommissionUpdateInterval
}

// GetMinCommissionRate returns the minimum commission rate.
func (k Keeper) GetMinCommissionRate(ctx sdk.Context) sdk.Dec {
	return k.GetParams(ctx).MinCommissionRate
}
