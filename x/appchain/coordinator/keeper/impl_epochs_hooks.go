package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	avstypes "github.com/imua-xyz/imuachain/x/avs/types"
	epochstypes "github.com/imua-xyz/imuachain/x/epochs/types"
)

// EpochsHooksWrapper is the wrapper structure that implements the epochs hooks for the dogfood
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
	ctx sdk.Context, identifier string, epoch int64,
) {
	// whenever an epoch ends, we should iterate over the list of pending subscriber chains
	// to be activated, and then activate them. once activated, we should move them from
	// the pending list to the active list.
	executable := wrapper.keeper.GetPendingSubChains(ctx, identifier, uint64(epoch))
	for _, subscriber := range executable.List {
		cctx, writeFn, err := wrapper.keeper.CreateClientForSubscriberInCachedCtx(ctx, subscriber)
		if err != nil {
			// within this block, we use the ctx and not the cctx, since the cctx's job is solely
			// to guard the client creation.
			// no re-attempts will be made for this subscriber
			ctx.Logger().Error(
				"subscriber client not created",
				"chainID", subscriber.ChainID,
				"error", err,
			)
			// clear the registered AVS. remember that this module stores
			// the chainID with the revision but the AVS module stores it without.
			chainID := avstypes.ChainIDWithoutRevision(subscriber.ChainID)
			// always guaranteed to exist
			_, addr := wrapper.keeper.avsKeeper.IsAVSByChainID(ctx, chainID)
			if err := wrapper.keeper.avsKeeper.DeleteAVSInfo(ctx, addr); err != nil {
				// should never happen
				ctx.Logger().Error(
					"subscriber AVS not deleted",
					"chainID", subscriber,
					"error", err,
				)
			}
			continue
		}
		// copy over the events from the cached ctx
		ctx.EventManager().EmitEvents(cctx.EventManager().Events())
		writeFn()
		wrapper.keeper.Logger(ctx).Info(
			"subscriber chain started",
			"chainID", subscriber,
			// we start at the current block and do not allow scheduling. this is the same
			// as any other AVS.
			"spawn time", ctx.BlockTime().UTC(),
		)
	}
	// delete those that were executed (including those that failed)
	wrapper.keeper.ClearPendingSubChains(ctx, identifier, uint64(epoch))
	// next, we iterate over the active list and queue the validator set update for them.
}

// BeforeEpochStart is called before an epoch starts.
func (wrapper EpochsHooksWrapper) BeforeEpochStart(
	sdk.Context, string, int64,
) {
	// no-op
}
