package keeper

import (
	keytypes "github.com/ExocoreNetwork/exocore/types/keys"
	operatortypes "github.com/ExocoreNetwork/exocore/x/operator/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type OperatorHooksWrapper struct {
	keeper *Keeper
}

var _ operatortypes.OperatorHooks = OperatorHooksWrapper{}

func (k *Keeper) OperatorHooks() OperatorHooksWrapper {
	return OperatorHooksWrapper{k}
}

// AfterOperatorKeySet is the implementation of the operator hooks.
// CONTRACT: an operator cannot set their key if they are already in the process of removing it.
func (wrapper OperatorHooksWrapper) AfterOperatorKeySet(
	sdk.Context, sdk.AccAddress, string, keytypes.WrappedConsKey,
) {
}

// AfterOperatorKeyReplaced is the implementation of the operator hooks.
// CONTRACT: key replacement is not allowed if the operator is in the process of removing their
// key.
// CONTRACT: key replacement from newKey to oldKey is not allowed, after a replacement from
// oldKey to newKey.
func (wrapper OperatorHooksWrapper) AfterOperatorKeyReplaced(
	_ sdk.Context, _ sdk.AccAddress, _ keytypes.WrappedConsKey,
	_ keytypes.WrappedConsKey, _ string,
) {
}

// AfterOperatorKeyRemovalInitiated is the implementation of the operator hooks.
func (wrapper OperatorHooksWrapper) AfterOperatorKeyRemovalInitiated(
	_ sdk.Context, _ sdk.AccAddress, _ string, _ keytypes.WrappedConsKey,
) {
}

func (wrapper OperatorHooksWrapper) AfterSlash(
	ctx sdk.Context, _ sdk.AccAddress, _ []string,
) {
	logger := wrapper.keeper.Logger()
	logger.Info(
		"AfterSlash of distribution",
	)
	// When distribution triggered by slash, we only distribute fee collected until now from last distribution
	if err := wrapper.keeper.AllocateTokens(ctx, true); err != nil {
		logger.Error("failed to allocate tokens", "err", err)
	}
}
