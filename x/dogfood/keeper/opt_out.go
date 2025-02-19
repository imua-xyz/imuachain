package keeper

import (
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/imua-xyz/imuachain/x/dogfood/types"
)

// SetOptOutInformation sets information related to an operator's opt out.
func (k Keeper) SetOptOutInformation(
	ctx sdk.Context, addr sdk.AccAddress,
) {
	unbondingCompletionEpoch := k.GetUnbondingCompletionEpoch(ctx)
	k.AppendOptOutToFinish(ctx, unbondingCompletionEpoch, addr)
	k.SetOperatorOptOutFinishEpoch(ctx, addr, unbondingCompletionEpoch)
	// CompleteOperatorKeyRemovalForChainID calls DeleteOperatorAddressForChainIDAndConsAddr,
	// so we do not need to save ConsensusAddrToPrune in the unbonding case.
}

// AppendOptOutToFinish appends an operator address to the list of operator addresses that have
// opted out and will be finished at the end of the provided epoch.
// The caller must ensure that the operator address is not already in the list.
func (k Keeper) AppendOptOutToFinish(
	ctx sdk.Context, epoch int64, operatorAddr sdk.AccAddress,
) {
	prev := k.GetOptOutsToFinish(ctx, epoch)
	next := types.AccountAddresses{List: append(prev, operatorAddr)}
	k.setOptOutsToFinish(ctx, epoch, next)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeOptOutBegan,
			sdk.NewAttribute(types.AttributeKeyEpoch, strconv.FormatInt(epoch, 10)),
			sdk.NewAttribute(types.AttributeKeyOperator, operatorAddr.String()),
		),
	)
}

// GetOptOutsToFinish returns the list of operator addresses that have opted out and will be
// finished at the end of the provided epoch.
func (k Keeper) GetOptOutsToFinish(
	ctx sdk.Context, epoch int64,
) [][]byte {
	store := ctx.KVStore(k.storeKey)
	// the epochs module validates at genesis that epoch is non-negative.
	key, _ := types.OptOutsToFinishKey(epoch)
	bz := store.Get(key)
	if bz == nil {
		return [][]byte{}
	}
	var res types.AccountAddresses
	if err := res.Unmarshal(bz); err != nil {
		panic(err)
	}
	return res.GetList()
}

// setOptOutsToFinish sets the list of operator addresses that have opted out and will be
// finished at the end of the provided epoch.
func (k Keeper) setOptOutsToFinish(
	ctx sdk.Context, epoch int64, addrs types.AccountAddresses,
) {
	store := ctx.KVStore(k.storeKey)
	key, _ := types.OptOutsToFinishKey(epoch)
	bz, err := addrs.Marshal()
	if err != nil {
		panic(err)
	}
	store.Set(key, bz)
}

// ClearOptOutsToFinish clears the list of operator addresses that have opted out and will be
// finished at the end of the provided epoch.
func (k Keeper) ClearOptOutsToFinish(ctx sdk.Context, epoch int64) {
	store := ctx.KVStore(k.storeKey)
	key, _ := types.OptOutsToFinishKey(epoch)
	store.Delete(key)
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeOptOutsFinished,
			sdk.NewAttribute(types.AttributeKeyEpoch, strconv.FormatInt(epoch, 10)),
		),
	)
}

// GetAllOptOutsToFinish gets a list of epochs and the corresponding operator addresses
// whose opt outs are scheduled to finish at the end of those epochs. It is ordered,
// first by the epoch and then by the operator address bytes.
func (k Keeper) GetAllOptOutsToFinish(ctx sdk.Context) []types.EpochToOperatorAddrs {
	store := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStorePrefixIterator(store, []byte{types.OptOutsToFinishBytePrefix})
	defer iterator.Close()

	res := []types.EpochToOperatorAddrs{}

	for ; iterator.Valid(); iterator.Next() {
		epoch, _ := types.SafeUint64ToInt64(sdk.BigEndianToUint64(iterator.Key()[1:]))
		var addrs types.AccountAddresses
		k.cdc.MustUnmarshal(iterator.Value(), &addrs)
		subRes := []string{}
		for _, addr := range addrs.GetList() {
			subRes = append(subRes, sdk.AccAddress(addr).String())
		}
		res = append(res, types.EpochToOperatorAddrs{
			Epoch:            epoch,
			OperatorAccAddrs: subRes,
		})
	}

	return res
}

// SetOperatorOptOutFinishEpoch sets the epoch at which an operator's opt out will be finished.
func (k Keeper) SetOperatorOptOutFinishEpoch(
	ctx sdk.Context, operatorAddr sdk.AccAddress, epoch int64,
) {
	store := ctx.KVStore(k.storeKey)
	key := types.OperatorOptOutFinishEpochKey(operatorAddr)
	uepoch, _ := types.SafeInt64ToUint64(epoch)
	bz := sdk.Uint64ToBigEndian(uepoch)
	store.Set(key, bz)
}

// GetOperatorOptOutFinishEpoch returns the epoch at which an operator's opt out will be
// finished.
func (k Keeper) GetOperatorOptOutFinishEpoch(
	ctx sdk.Context, operatorAddr sdk.AccAddress,
) int64 {
	store := ctx.KVStore(k.storeKey)
	key := types.OperatorOptOutFinishEpochKey(operatorAddr)
	bz := store.Get(key)
	if bz == nil {
		return -1
	}
	// max int64 is 9 quintillion, and max uint64 is double of that.
	// it is too far in the future to be a concern.
	return int64(sdk.BigEndianToUint64(bz)) // #nosec G115
}

// DeleteOperatorOptOutFinishEpoch deletes the epoch at which an operator's opt out will be
// finished.
func (k Keeper) DeleteOperatorOptOutFinishEpoch(
	ctx sdk.Context, operatorAddr sdk.AccAddress,
) {
	store := ctx.KVStore(k.storeKey)
	key := types.OperatorOptOutFinishEpochKey(operatorAddr)
	store.Delete(key)
}

// AppendConsensusAddrToPrune appends a consensus address to the list of consensus addresses to
// prune at the end of the epoch.
func (k Keeper) AppendConsensusAddrToPrune(
	ctx sdk.Context, epoch int64, operatorAddr sdk.ConsAddress,
) {
	prev := k.GetConsensusAddrsToPrune(ctx, epoch)
	next := types.ConsensusAddresses{List: append(prev, operatorAddr)}
	k.setConsensusAddrsToPrune(ctx, epoch, next)
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeConsAddrPruningScheduled,
			sdk.NewAttribute(types.AttributeKeyEpoch, strconv.FormatInt(epoch, 10)),
			sdk.NewAttribute(types.AttributeKeyConsAddr, operatorAddr.String()),
		),
	)
}

// GetConsensusAddrsToPrune returns the list of consensus addresses to prune at the end of the
// epoch.
func (k Keeper) GetConsensusAddrsToPrune(
	ctx sdk.Context, epoch int64,
) [][]byte {
	store := ctx.KVStore(k.storeKey)
	key, _ := types.ConsensusAddrsToPruneKey(epoch)
	bz := store.Get(key)
	if bz == nil {
		return [][]byte{}
	}
	var res types.ConsensusAddresses
	if err := res.Unmarshal(bz); err != nil {
		panic(err)
	}
	return res.GetList()
}

// ClearConsensusAddrsToPrune clears the list of consensus addresses to prune at the end of the
// epoch.
func (k Keeper) ClearConsensusAddrsToPrune(ctx sdk.Context, epoch int64) {
	store := ctx.KVStore(k.storeKey)
	key, _ := types.ConsensusAddrsToPruneKey(epoch)
	store.Delete(key)
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeConsAddrsPruned,
			sdk.NewAttribute(types.AttributeKeyEpoch, strconv.FormatInt(epoch, 10)),
		),
	)
}

// setConsensusAddrsToPrune sets the list of consensus addresses to prune at the end of the
// epoch.
func (k Keeper) setConsensusAddrsToPrune(
	ctx sdk.Context, epoch int64, addrs types.ConsensusAddresses,
) {
	store := ctx.KVStore(k.storeKey)
	key, _ := types.ConsensusAddrsToPruneKey(epoch)
	bz, err := addrs.Marshal()
	if err != nil {
		panic(err)
	}
	store.Set(key, bz)
}

// GetAllConsAddrsToPrune gets a list of epochs and the corresponding consensus addresses
// which should be pruned from the operator module at the end of that epoch. It is ordered,
// first by the epoch and then by the consensus address bytes.
func (k Keeper) GetAllConsAddrsToPrune(ctx sdk.Context) []types.EpochToConsensusAddrs {
	store := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStorePrefixIterator(store, []byte{types.OptOutsToFinishBytePrefix})
	defer iterator.Close()

	res := []types.EpochToConsensusAddrs{}

	for ; iterator.Valid(); iterator.Next() {
		epoch, _ := types.SafeUint64ToInt64(sdk.BigEndianToUint64(iterator.Key()[1:]))
		var addrs types.ConsensusAddresses
		k.cdc.MustUnmarshal(iterator.Value(), &addrs)
		subRes := []string{}
		for _, addr := range addrs.GetList() {
			subRes = append(subRes, sdk.ConsAddress(addr).String())
		}
		res = append(res, types.EpochToConsensusAddrs{
			Epoch:     epoch,
			ConsAddrs: subRes,
		})
	}

	return res
}
