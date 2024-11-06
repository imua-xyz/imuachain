package keeper

import (
	epochstypes "github.com/ExocoreNetwork/exocore/x/epochs/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// EpochsHooksWrapper is the wrapper structure that implements the epochs hooks for the operator
// keeper.
type EpochsHooksWrapper struct {
	keeper *Keeper
}

// Interface guard
var _ epochstypes.EpochHooks = EpochsHooksWrapper{}

// EpochsHooks returns the epochs hooks wrapper. It follows the "accept interfaces, return
// concretes" pattern.
func (k *Keeper) EpochsHooks() EpochsHooksWrapper {
	return EpochsHooksWrapper{k}
}

// AfterEpochEnd is called after an epoch ends. It is called during the BeginBlock function.
func (wrapper EpochsHooksWrapper) AfterEpochEnd(
	ctx sdk.Context, epochIdentifier string, epochNumber int64,
) {
	// get all the avs address bypass the epoch end
	// update the assets' share when their prices change
	// todo: need to consider the calling order
	avsList := wrapper.keeper.avsKeeper.GetEpochEndAVSs(ctx, epochIdentifier, epochNumber)
	for _, avs := range avsList {
		// avs address is checksummed hex, we should convert it to lowercase
		err := wrapper.keeper.UpdateVotingPower(ctx, avs)
		if err != nil {
			ctx.Logger().Error("Failed to update voting power", "avs", avs, "error", err)
			// Handle the error gracefully, continue to the next AVS
			continue
		}
	}
}

// BeforeEpochStart is called before an epoch starts.
func (wrapper EpochsHooksWrapper) BeforeEpochStart(
	sdk.Context, string, int64,
) {
}
