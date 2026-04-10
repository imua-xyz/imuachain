package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/imua-xyz/imuachain/x/oracle/types"
)

// EndBlock contains the end-of-block logic for the oracle module.
// It runs FeederManager endblock first (which may enqueue xchain batches via postHandlers),
// then runs budgeted xchain queue delivery.
func (k Keeper) EndBlock(ctx sdk.Context) {
	k.FeederManager.EndBlock(ctx)
	k.processAllXChainQueues(ctx)
	k.createOutboundCheckpoints(ctx)
}

// createOutboundCheckpoints iterates all outbound queues and creates checkpoints
// for any pending messages that haven't been checkpointed yet.
func (k Keeper) createOutboundCheckpoints(ctx sdk.Context) {
	store := ctx.KVStore(k.storeKey)
	prefix := []byte(types.OutboundHeadPrefix)
	it := sdk.KVStorePrefixIterator(store, prefix)
	defer it.Close()

	for ; it.Valid(); it.Next() {
		key := it.Key()
		if len(key) != len(prefix)+8 {
			continue
		}
		dstChainID, err := types.BytesToUint64(key[len(prefix):])
		if err != nil {
			continue
		}
		k.CreateCheckpointForPendingOutbound(ctx, dstChainID)
	}
}

func (k Keeper) processAllXChainQueues(ctx sdk.Context) {
	store := ctx.KVStore(k.storeKey)
	prefix := []byte(types.XChainQueueHeadPrefix)
	it := sdk.KVStorePrefixIterator(store, prefix)
	defer it.Close()

	for ; it.Valid(); it.Next() {
		key := it.Key()
		if len(key) != len(prefix)+8 {
			continue
		}
		srcChainID, err := types.BytesToUint64(key[len(prefix):])
		if err != nil {
			continue
		}
		k.ProcessXChainQueue(ctx, srcChainID)
	}
}
