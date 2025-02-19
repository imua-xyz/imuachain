//nolint:dupl
package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/imua-xyz/imuachain/x/oracle/types"
)

// SetValidatorUpdateBlock set validatorUpdateBlock in the store
func (k Keeper) SetValidatorUpdateBlock(ctx sdk.Context, validatorUpdateBlock types.ValidatorUpdateBlock) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ValidatorUpdateBlockKey))
	b := k.cdc.MustMarshal(&validatorUpdateBlock)
	store.Set([]byte{0}, b)
}

// GetValidatorUpdateBlock returns validatorUpdateBlock
func (k Keeper) GetValidatorUpdateBlock(ctx sdk.Context) (val types.ValidatorUpdateBlock, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ValidatorUpdateBlockKey))

	b := store.Get([]byte{0})
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveValidatorUpdateBlock removes validatorUpdateBlock from the store
func (k Keeper) RemoveValidatorUpdateBlock(ctx sdk.Context) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ValidatorUpdateBlockKey))
	store.Delete([]byte{0})
}
