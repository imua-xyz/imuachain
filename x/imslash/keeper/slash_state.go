package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/imua-xyz/imuachain/x/imslash/types"
)

func (k Keeper) SetFrozenStatus(ctx sdk.Context, operatorAddr string, status bool) (err error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixOperatorInfo)
	if status {
		store.Set([]byte(operatorAddr), []byte("1"))
		return nil
	}
	store.Set([]byte(operatorAddr), []byte("0"))
	return nil
}

func (k Keeper) GetFrozenStatus(ctx sdk.Context, operatorAddr string) (bool, error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixOperatorInfo)
	ifExist := store.Has(types.ParamsKey)

	if !ifExist {
		return false, types.ErrNoOperatorStatusKey
	}
	value := store.Get([]byte(operatorAddr))
	if string(value) == "1" {
		return true, nil
	}

	return false, nil
}
