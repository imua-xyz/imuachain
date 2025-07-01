package keeper

import (
	"cosmossdk.io/math"
	keytypes "github.com/imua-xyz/imuachain/types/keys"
	assetstype "github.com/imua-xyz/imuachain/x/assets/types"
	delegationtypes "github.com/imua-xyz/imuachain/x/delegation/types"
	operatortypes "github.com/imua-xyz/imuachain/x/operator/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	epochstypes "github.com/imua-xyz/imuachain/x/epochs/types"
)

// EpochsHooksWrapper is the wrapper structure that implements the epochs hooks for the distribution
// keeper.
type EpochsHooksWrapper struct {
	keeper *Keeper
}

// Interface guard
var _ epochstypes.EpochHooks = EpochsHooksWrapper{}

// EpochsHooks returns the epochs hooks wrapper.
func (k *Keeper) EpochsHooks() EpochsHooksWrapper {
	return EpochsHooksWrapper{k}
}

// BeforeEpochStart : noop, We don't need to do anything here
func (wrapper EpochsHooksWrapper) BeforeEpochStart(_ sdk.Context, _ string, _ int64) {
}

// AfterEpochEnd mints and allocates coins at the end of each epoch end
func (wrapper EpochsHooksWrapper) AfterEpochEnd(ctx sdk.Context, epochIdentifier string, epochNumber int64) {
	// distribute the rewards to operators
	err := wrapper.keeper.AllocateRewardsByEpoch(ctx, epochIdentifier, epochNumber)
	if err != nil {
		ctx.Logger().Error("failed to allocate the rewards by epoch", "err", err, "EpochIdentifier", epochIdentifier, "epochNumber", epochNumber)
		// Do not return, as the reward distribution for stakers should not be affected
		// by the AVS rewards distribution of the current epoch.
		// If the function returns here, the cumulative rewards for the staker will not be
		// distributed correctly.
	}
	// handle delegations whose stake has changed.
	err = wrapper.keeper.HandleChangedDelegations(ctx, epochIdentifier)
	if err != nil {
		ctx.Logger().Error("failed to handle the delegations with changed stakes by epoch", "err", err, "EpochIdentifier", epochIdentifier, "epochNumber", epochNumber)
		return
	}
	// clear the delegation change information
	// this function will be called by the epoch hook, so using cache context
	// to ensure the state atomicity.
	cc, writeFunc := ctx.CacheContext()
	err = wrapper.keeper.DeleteStakeChangedDelegationsByEpoch(cc, epochIdentifier)
	if err != nil {
		ctx.Logger().Error("failed to delete the delegation change information by epoch", "err", err, "EpochIdentifier", epochIdentifier, "epochNumber", epochNumber)
		return
	}
	writeFunc()
}

// DelegationHooksWrapper is the wrapper structure that implements the delegation hooks for the
// distribution keeper.
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
	ctx sdk.Context, stakerID, assetID string, operator sdk.AccAddress,
	preDelegatedAmount math.Int, prevAssetState assetstype.OperatorAssetInfo,
) error {
	return wrapper.keeper.MarkChangedDelegations(ctx, stakerID, assetID, operator, preDelegatedAmount, prevAssetState)
}

// AfterUndelegationStarted is called after an undelegation is started.
func (wrapper DelegationHooksWrapper) AfterUndelegationStarted(
	ctx sdk.Context, stakerID, assetID string, operator sdk.AccAddress, _ []byte,
	preDelegatedAmount math.Int, prevAssetState assetstype.OperatorAssetInfo,
) error {
	return wrapper.keeper.MarkChangedDelegations(ctx, stakerID, assetID, operator, preDelegatedAmount, prevAssetState)
}

// OperatorHooksWrapper is the wrapper structure that implements the operator hooks for the
// distribution keeper.
type OperatorHooksWrapper struct {
	keeper *Keeper
}

// Interface guards
var _ operatortypes.OperatorHooks = OperatorHooksWrapper{}

func (k *Keeper) OperatorHooks() OperatorHooksWrapper {
	return OperatorHooksWrapper{k}
}

// AfterOperatorKeySet is the implementation of the operator hooks.
func (h OperatorHooksWrapper) AfterOperatorKeySet(
	sdk.Context, sdk.AccAddress, string, keytypes.WrappedConsKey,
) {
	// No operation needed here.
}

// AfterOperatorKeyReplaced is the implementation of the operator hooks.
func (h OperatorHooksWrapper) AfterOperatorKeyReplaced(
	sdk.Context, sdk.AccAddress, keytypes.WrappedConsKey,
	keytypes.WrappedConsKey, string,
) {
	// No operation needed here.
}

// AfterOperatorKeyRemovalInitiated is the implementation of the operator hooks.
func (h OperatorHooksWrapper) AfterOperatorKeyRemovalInitiated(
	_ sdk.Context, _ sdk.AccAddress, _ string, _ keytypes.WrappedConsKey,
) {
}

// AfterSlash is the implementation of the operator hooks.
func (h OperatorHooksWrapper) AfterSlash(
	ctx sdk.Context, operator sdk.AccAddress, slashProportion sdk.Dec, _ []string,
	slashAssetsPool []operatortypes.SlashFromAssetsPool,
) {
	cc, writeFunc := ctx.CacheContext()
	err := h.keeper.HandleOperatorSlashEvent(cc, operator, slashProportion, slashAssetsPool)
	if err != nil {
		ctx.Logger().Error("AfterSlash: failed to handle the slash event", "err", err,
			"operator", operator, "slashProportion", slashProportion, "slashAssetsPool", slashAssetsPool)
		return
	}
	writeFunc()
}

func (h OperatorHooksWrapper) AfterJail(_ sdk.Context, _ sdk.AccAddress, _ bool, _ []string) {
	// do nothing
}
