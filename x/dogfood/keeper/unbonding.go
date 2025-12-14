package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// GetUnbondingCompletionEpoch returns the epoch at the end of which
// an unbonding triggered in this epoch will be completed.
func (k Keeper) GetUnbondingCompletionEpoch(
	ctx sdk.Context,
) int64 {
	params := k.GetDogfoodParams(ctx)
	epochInfo, _ := k.epochsKeeper.GetEpochInfo(
		ctx, params.EpochIdentifier,
	)
	// if i execute the transaction at epoch 5, the vote power change
	// goes into effect at the beginning of epoch 6. the information
	// should be held for 7 epochs, so it should be deleted at the
	// beginning of epoch 13 or the end of epoch 12.
	return epochInfo.CurrentEpoch + int64(params.EpochsUntilUnbonded) // #nosec G701
}
