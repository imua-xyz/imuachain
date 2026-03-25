package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	keytypes "github.com/imua-xyz/imuachain/types/keys"
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
) error {
	for _, hook := range hooks {
		if err := hook.AfterOperatorKeySet(ctx, addr, chainID, pubKey); err != nil {
			return err
		}
	}
	return nil
}

func (hooks MultiOperatorHooks) AfterOperatorKeyReplaced(
	ctx sdk.Context,
	addr sdk.AccAddress,
	oldKey keytypes.WrappedConsKey,
	newKey keytypes.WrappedConsKey,
	chainID string,
) error {
	for _, hook := range hooks {
		if err := hook.AfterOperatorKeyReplaced(ctx, addr, oldKey, newKey, chainID); err != nil {
			return err
		}
	}
	return nil
}

func (hooks MultiOperatorHooks) AfterOperatorKeyRemovalInitiated(
	ctx sdk.Context, addr sdk.AccAddress, chainID string, key keytypes.WrappedConsKey,
) {
	for _, hook := range hooks {
		hook.AfterOperatorKeyRemovalInitiated(ctx, addr, chainID, key)
	}
}

func (hooks MultiOperatorHooks) AfterSlash(
	ctx sdk.Context, addr sdk.AccAddress, slashProportion sdk.Dec, affectedAVSList []string,
	slashAssetsPool []SlashAssetAmount, slashUnclaimedRewards []SlashFromUnclaimedRewards,
) {
	for _, hook := range hooks {
		hook.AfterSlash(ctx, addr, slashProportion, affectedAVSList, slashAssetsPool, slashUnclaimedRewards)
	}
}

func (hooks MultiOperatorHooks) AfterJail(
	ctx sdk.Context, addr sdk.AccAddress, isUnjail bool, affectedAVSList []string,
) {
	for _, hook := range hooks {
		hook.AfterJail(ctx, addr, isUnjail, affectedAVSList)
	}
}
