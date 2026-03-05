package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	operatortypes "github.com/imua-xyz/imuachain/x/operator/types"
)

// FreezeOperator permanently freezes an operator.
func (k *Keeper) FreezeOperator(ctx sdk.Context, addr sdk.AccAddress) error {
	if !k.IsOperator(ctx, addr) {
		return operatortypes.ErrNoSuchOperator
	}
	if k.IsOperatorFrozen(ctx, addr) {
		return operatortypes.ErrOperatorAlreadyFrozen
	}
	store := ctx.KVStore(k.storeKey)
	store.Set(operatortypes.KeyForOperatorFrozen(addr), []byte{1})
	return nil
}

// IsOperatorFrozen checks if the operator is permanently frozen.
func (k *Keeper) IsOperatorFrozen(ctx sdk.Context, addr sdk.AccAddress) bool {
	store := ctx.KVStore(k.storeKey)
	return store.Has(operatortypes.KeyForOperatorFrozen(addr))
}

// GetAllFrozenOperators returns all frozen operators.
func (k Keeper) GetAllFrozenOperators(ctx sdk.Context) []string {
	store := ctx.KVStore(k.storeKey)
	ret := []string{}
	iterator := sdk.KVStorePrefixIterator(store, operatortypes.KeyPrefixOperatorFrozen)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		// strip the prefix
		accAddr := sdk.AccAddress(iterator.Key()[1:])
		ret = append(ret, accAddr.String())
	}
	return ret
}
