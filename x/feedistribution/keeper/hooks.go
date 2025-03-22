package keeper

import (
	assetstype "github.com/ExocoreNetwork/exocore/x/assets/types"
	delegationtypes "github.com/ExocoreNetwork/exocore/x/delegation/types"
	"strings"

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
	expEpochID := wrapper.keeper.GetParams(ctx).EpochIdentifier
	if strings.Compare(epochIdentifier, expEpochID) == 0 {
		// distribute the rewards to operators
		err := wrapper.keeper.AllocateRewardsByEpoch(ctx, epochIdentifier, epochNumber)
		if err != nil {
			ctx.Logger().Error("failed to allocate the rewards by epoch", "err", err, "epochIdentifier", epochIdentifier, "epochNumber", epochNumber)
			return
		}
		// handle delegations whose stake has changed.
		err = wrapper.keeper.HandleChangedDelegations(ctx, epochIdentifier)
		if err != nil {
			ctx.Logger().Error("failed to handle the delegations with changed stakes by epoch", "err", err, "epochIdentifier", epochIdentifier, "epochNumber", epochNumber)
			return
		}
	}
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
	ctx sdk.Context, stakerID, assetID string, operator sdk.AccAddress, prevAssetState assetstype.OperatorAssetInfo,
) error {
	return wrapper.keeper.MarkStakeChangedDelegations(ctx, stakerID, assetID, operator, prevAssetState)
}

// AfterUndelegationStarted is called after an undelegation is started.
func (wrapper DelegationHooksWrapper) AfterUndelegationStarted(
	ctx sdk.Context, stakerID, assetID string, operator sdk.AccAddress, _ []byte, prevAssetState assetstype.OperatorAssetInfo,
) error {
	return wrapper.keeper.MarkStakeChangedDelegations(ctx, stakerID, assetID, operator, prevAssetState)
}
