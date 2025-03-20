package types

import (
	assetstype "github.com/ExocoreNetwork/exocore/x/assets/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ DelegationHooks = &MultiDelegationHooks{}

type MultiDelegationHooks []DelegationHooks

func NewMultiDelegationHooks(hooks ...DelegationHooks) MultiDelegationHooks {
	return hooks
}

func (hooks MultiDelegationHooks) AfterDelegation(ctx sdk.Context, stakerID, assetID string, operator sdk.AccAddress, prevAssetState assetstype.OperatorAssetInfo) error {
	for _, hook := range hooks {
		err := hook.AfterDelegation(ctx, stakerID, assetID, operator, prevAssetState)
		if err != nil {
			return err
		}
	}
	return nil
}

func (hooks MultiDelegationHooks) AfterUndelegationStarted(
	ctx sdk.Context,
	stakerID, assetID string,
	addr sdk.AccAddress,
	recordKey []byte,
	prevAssetState assetstype.OperatorAssetInfo,
) error {
	for _, hook := range hooks {
		err := hook.AfterUndelegationStarted(ctx, stakerID, assetID, addr, recordKey, prevAssetState)
		if err != nil {
			return err
		}
	}
	return nil
}
