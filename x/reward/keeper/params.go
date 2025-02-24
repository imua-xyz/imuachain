package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/imua-xyz/imuachain/x/reward/types"
)

// SetParams The function related to module parameter should be deleted
// if no parameters need to be stored in the future.
func (k Keeper) SetParams(ctx sdk.Context, params *types.Params) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixParams)
	bz := k.cdc.MustMarshal(params)
	store.Set(types.ParamsKey, bz)
	return nil
}

func (k Keeper) GetParams(ctx sdk.Context) (*types.Params, error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixParams)
	ifExist := store.Has(types.ParamsKey)
	if !ifExist {
		return nil, types.ErrNoParamsKey
	}

	value := store.Get(types.ParamsKey)

	ret := &types.Params{}
	k.cdc.MustUnmarshal(value, ret)
	return ret, nil
}
