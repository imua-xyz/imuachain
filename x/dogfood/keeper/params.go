package keeper

import (
	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/imua-xyz/imuachain/x/dogfood/types"
)

// GetEpochsUntilUnbonded returns the number of epochs after which an unbonding that is made
// during the current epoch will be released. It is a parameter of the dogfood module.
func (k Keeper) GetEpochsUntilUnbonded(ctx sdk.Context) uint32 {
	return k.GetDogfoodParams(ctx).EpochsUntilUnbonded
}

// GetEpochIdentifier returns the epoch identifier used to measure an epoch. It is a parameter
// of the dogfood module.
func (k Keeper) GetEpochIdentifier(ctx sdk.Context) string {
	return k.GetDogfoodParams(ctx).EpochIdentifier
}

// GetMaxValidators returns the maximum number of validators that can be asked to validate for
// the chain. It is a parameter of the dogfood module.
func (k Keeper) GetMaxValidators(ctx sdk.Context) uint32 {
	return k.GetDogfoodParams(ctx).MaxValidators
}

// GetHistorialEntries is the number of historical info entries to persist in the store. These
// entries are used by the IBC module. The return value is a parameter of the dogfood module.
func (k Keeper) GetHistoricalEntries(ctx sdk.Context) uint32 {
	return k.GetDogfoodParams(ctx).HistoricalEntries
}

// GetAssetIDs returns the asset IDs that are accepted by the dogfood module. It is a parameter
// of the dogfood module.
func (k Keeper) GetAssetIDs(ctx sdk.Context) []string {
	return k.GetDogfoodParams(ctx).AssetIDs
}

// GetMinSelfDelegation returns the minimum self-delegation amount for a validator. It is a
// parameter of the dogfood module.
func (k Keeper) GetMinSelfDelegation(ctx sdk.Context) sdkmath.Int {
	return k.GetDogfoodParams(ctx).MinSelfDelegation
}

// SetParams sets the params for the dogfood module.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	store := ctx.KVStore(k.storeKey)
	key := types.ParamsKey()
	bz := k.cdc.MustMarshal(&params)
	store.Set(key, bz)
}

// GetDogfoodParams returns the parameters for the dogfood module. Note that this function is
// intentionally called GetDogfoodParams and not GetParams, since the GetParams function is used
// to implement the slashingtypes.StakingKeeper interface `GetParams(sdk.Context)
// stakingtypes.Params`.
func (k Keeper) GetDogfoodParams(ctx sdk.Context) types.Params {
	store := ctx.KVStore(k.storeKey)
	key := types.ParamsKey()
	bz := store.Get(key)
	var params types.Params
	k.cdc.MustUnmarshal(bz, &params)
	return params
}
