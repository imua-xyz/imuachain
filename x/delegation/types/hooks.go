package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ DelegationHooks = &MultiDelegationHooks{}

type MultiDelegationHooks []DelegationHooks

func NewMultiDelegationHooks(hooks ...DelegationHooks) MultiDelegationHooks {
	return hooks
}

func (hooks MultiDelegationHooks) AfterDelegation(ctx sdk.Context, stakerID, assetID string, operator sdk.AccAddress) error {
	for _, hook := range hooks {
		err := hook.AfterDelegation(ctx, stakerID, assetID, operator)
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
) error {
	for _, hook := range hooks {
		err := hook.AfterUndelegationStarted(ctx, stakerID, assetID, addr, recordKey)
		if err != nil {
			return err
		}
	}
	return nil
}
