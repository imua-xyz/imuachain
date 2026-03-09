package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	operatortypes "github.com/imua-xyz/imuachain/x/operator/types"
)

// FreezeOperator permanently freezes an operator.
func (k *Keeper) FreezeOperator(ctx sdk.Context, addr sdk.AccAddress) error {
	if !k.IsOperator(ctx, addr) {
		return operatortypes.ErrNoSuchOperator
	}
	if k.IsOperatorFrozen(ctx, addr) {
		return operatortypes.ErrOperatorFrozenStateMismatch.Wrapf(
			"operator %s is already frozen", addr.String(),
		)
	}
	store := ctx.KVStore(k.storeKey)
	store.Set(operatortypes.KeyForOperatorFrozen(addr), []byte{1})
	k.emitFreezeEvent(ctx, addr, true)
	return nil
}

// emitFreezeEvent emits an event of name EventTypeFreezeOperator
// The attributes of the event are the operator address, and whether
// the operator is frozen (true) or unfrozen (false)
func (k Keeper) emitFreezeEvent(
	ctx sdk.Context, addr sdk.AccAddress, frozen bool,
) {
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			operatortypes.EventTypeFreezeOperator,
			sdk.NewAttribute(
				operatortypes.AttributeKeyOperator,
				addr.String(),
			),
			sdk.NewAttribute(
				operatortypes.AttributeKeyFrozenOrUnfrozen,
				fmt.Sprintf("%t", frozen),
			),
		),
	)
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
	prefix := operatortypes.KeyPrefixOperatorFrozen
	iterator := sdk.KVStorePrefixIterator(
		store, prefix,
	)
	prefixLen := len(prefix)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		// strip the prefix
		accAddr := sdk.AccAddress(iterator.Key()[prefixLen:])
		ret = append(ret, accAddr.String())
	}
	return ret
}

// UnfreezeOperator marks a previously frozen operator as unfrozen.
// It is ideally a governance gated operation.
func (k Keeper) UnfreezeOperator(
	ctx sdk.Context, addr sdk.AccAddress,
) error {
	if !k.IsOperator(ctx, addr) {
		return operatortypes.ErrNoSuchOperator
	}
	if !k.IsOperatorFrozen(ctx, addr) {
		return operatortypes.ErrOperatorFrozenStateMismatch.Wrapf(
			"operator %s is not frozen", addr.String(),
		)
	}
	store := ctx.KVStore(k.storeKey)
	store.Delete(operatortypes.KeyForOperatorFrozen(addr))
	k.emitFreezeEvent(ctx, addr, false)
	return nil
}
