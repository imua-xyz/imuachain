package keeper

import (
	"github.com/ExocoreNetwork/exocore/x/oracle/types"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) SetValidatorUpdateForCache(ctx sdk.Context, validatorUpdateBlock types.ValidatorUpdateBlock) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ValidatorUpdateBlockKey))
	b := k.cdc.MustMarshal(&validatorUpdateBlock)
	store.Set(types.BlockKey, b)
}

// GetValidatorUpdateBlock returns validatorUpdateBlock
func (k Keeper) GetValidatorUpdateBlock(ctx sdk.Context) (val types.ValidatorUpdateBlock, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ValidatorUpdateBlockKey))

	b := store.Get(types.BlockKey)
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveValidatorUpdateBlock removes validatorUpdateBlock from the store
func (k Keeper) RemoveValidatorUpdateBlock(ctx sdk.Context) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ValidatorUpdateBlockKey))
	store.Delete(types.BlockKey)
}
