package types

import (
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	assetstype "github.com/imua-xyz/imuachain/x/assets/types"
)

var _ DelegationHooks = &MultiDelegationHooks{}

type MultiDelegationHooks []DelegationHooks

func NewMultiDelegationHooks(hooks ...DelegationHooks) MultiDelegationHooks {
	return hooks
}

func (hooks MultiDelegationHooks) AfterDelegation(ctx sdk.Context, stakerID, assetID string, operator sdk.AccAddress,
	preDelegatedAmount sdkmath.Int, prevAssetState assetstype.OperatorAssetInfo,
) error {
	for _, hook := range hooks {
		err := hook.AfterDelegation(ctx, stakerID, assetID, operator, preDelegatedAmount, prevAssetState)
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
	preDelegatedAmount sdkmath.Int,
	prevAssetState assetstype.OperatorAssetInfo,
) error {
	for _, hook := range hooks {
		err := hook.AfterUndelegationStarted(ctx, stakerID, assetID, addr, recordKey, preDelegatedAmount, prevAssetState)
		if err != nil {
			return err
		}
	}
	return nil
}
