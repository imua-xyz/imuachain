package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	commontypes "github.com/imua-xyz/imuachain/x/appchain/common/types"
	"github.com/imua-xyz/imuachain/x/appchain/subscriber/types"
)

// SetParams sets the appchain coordinator parameters.
// The caller must ensure that the params are valid.
func (k Keeper) SetParams(ctx sdk.Context, params commontypes.SubscriberParams) {
	store := ctx.KVStore(k.storeKey)
	bz := k.cdc.MustMarshal(&params)
	store.Set(types.ParamsKey(), bz)
}

// GetParams gets the appchain coordinator parameters.
func (k Keeper) GetParams(ctx sdk.Context) (res commontypes.SubscriberParams) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ParamsKey())
	k.cdc.MustUnmarshal(bz, &res)
	return res
}
