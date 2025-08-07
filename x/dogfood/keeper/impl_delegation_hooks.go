package keeper

import (
	sdkmath "cosmossdk.io/math"

	assetstype "github.com/imua-xyz/imuachain/x/assets/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	delegationtypes "github.com/imua-xyz/imuachain/x/delegation/types"
)

// DelegationHooksWrapper is the wrapper structure that implements the delegation hooks for the
// dogfood keeper.
type DelegationHooksWrapper struct {
	keeper *Keeper
}

// Interface guard
var _ delegationtypes.DelegationHooks = DelegationHooksWrapper{}

// DelegationHooks returns the delegation hooks wrapper. It follows the "accept interfaces,
// return concretes" pattern.
func (k *Keeper) DelegationHooks() DelegationHooksWrapper {
	return DelegationHooksWrapper{k}
}

// AfterDelegation is called after a delegation is made.
func (wrapper DelegationHooksWrapper) AfterDelegation(
	_ sdk.Context, _, _ string, _ sdk.AccAddress, _ sdkmath.Int, _ assetstype.OperatorAssetInfo,
) error {
	// we do nothing here, since the vote power for all operators is calculated
	// in the end separately. even if we knew the amount of the delegation, the
	// exchange rate at the end of the epoch is unknown.
	return nil
}

// AfterUndelegationStarted is called after an undelegation is started.
func (wrapper DelegationHooksWrapper) AfterUndelegationStarted(
	_ sdk.Context, _, _ string, _ sdk.AccAddress, _ []byte,
	_ sdkmath.Int, _ assetstype.OperatorAssetInfo,
) error {
	// Do nothing here because the `GetUnbondingExpiration` function can now
	// calculate the correct unbonding duration, even for opt-out cases;
	// therefore, the dogfood module doesn't need to manage the completion of undelegations.
	// todo: remove the whole hook file and the related code in the future?
	return nil
}
