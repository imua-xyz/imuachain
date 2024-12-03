package types

import (
	keytypes "github.com/ExocoreNetwork/exocore/types/keys"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ OperatorHooks = &MultiOperatorHooks{}

type MultiOperatorHooks []OperatorHooks

func NewMultiOperatorHooks(hooks ...OperatorHooks) MultiOperatorHooks {
	return hooks
}

func (hooks MultiOperatorHooks) AfterOperatorKeySet(
	ctx sdk.Context,
	addr sdk.AccAddress,
	chainID string,
	pubKey keytypes.WrappedConsKey,
) {
	for _, hook := range hooks {
		hook.AfterOperatorKeySet(ctx, addr, chainID, pubKey)
	}
}

func (hooks MultiOperatorHooks) AfterOperatorKeyReplaced(
	ctx sdk.Context,
	addr sdk.AccAddress,
	oldKey keytypes.WrappedConsKey,
	newAddr keytypes.WrappedConsKey,
	chainID string,
) {
	for _, hook := range hooks {
		hook.AfterOperatorKeyReplaced(ctx, addr, oldKey, newAddr, chainID)
	}
}

func (hooks MultiOperatorHooks) AfterOperatorKeyRemovalInitiated(
	ctx sdk.Context, addr sdk.AccAddress, chainID string, key keytypes.WrappedConsKey,
) {
	for _, hook := range hooks {
		hook.AfterOperatorKeyRemovalInitiated(ctx, addr, chainID, key)
	}
}

func (hooks MultiOperatorHooks) AfterSlash(
	ctx sdk.Context, addr sdk.AccAddress, affectedAVSList []string,
) {
	for _, hook := range hooks {
		hook.AfterSlash(ctx, addr, affectedAVSList)
	}
}
