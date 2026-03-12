package keeper

import (
	"fmt"

	"cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	keytypes "github.com/imua-xyz/imuachain/types/keys"
	"github.com/imua-xyz/imuachain/utils"
	"github.com/imua-xyz/imuachain/x/dogfood/types"
)

func (k Keeper) BeginBlock(ctx sdk.Context) {
	// for IBC, track historical validator set
	k.TrackHistoricalInfo(ctx)
	// check if event needs to be emitted
	if k.ShouldEmitAvsEvent(ctx) {
		defer k.ClearEmitAvsEventFlag(ctx)
		// emit the event
		chainIDWithoutRevision := utils.ChainIDWithoutRevision(ctx.ChainID())
		_, avsAddress := k.avsKeeper.IsAVSByChainID(ctx, chainIDWithoutRevision)
		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				types.EventTypeDogfoodAvsCreated,
				sdk.NewAttribute(types.AttributeKeyChainIDWithoutRev, chainIDWithoutRevision),
				sdk.NewAttribute(types.AttributeKeyAvsAddress, avsAddress),
			),
		)
	}
}

func (k Keeper) EndBlock(ctx sdk.Context) []abci.ValidatorUpdate {
	if !k.ShouldUpdateValidatorSet(ctx) {
		k.SetValidatorUpdates(ctx, []abci.ValidatorUpdate{})
		return []abci.ValidatorUpdate{}
	}
	defer k.ClearValidatorSetUpdateFlag(ctx)
	logger := k.Logger(ctx)
	chainIDWithoutRevision := utils.ChainIDWithoutRevision(ctx.ChainID())
	// recall that epoch hooks are called in BeginBlocker and we are in the EndBlocker.
	// it means that the epoch number reported by the epoch module is correct.
	epochID := k.GetDogfoodParams(ctx).EpochIdentifier
	// note: we can ignore the case wherein the epoch is not found or panic.
	// for complete code level clarity, i chose to panic. currently, we check
	// the existence of the epoch at genesis, and we do not permit changing the
	// epoch identifier in the MsgServer. thus, it is guaranteed to exist.
	epochInfo, found := k.epochsKeeper.GetEpochInfo(ctx, epochID)
	if !found {
		panic(fmt.Sprintf("epoch %s not found", epochID))
	}
	currentEpochNumber := epochInfo.CurrentEpoch
	nextEpochNumber := currentEpochNumber + 1
	// start by clearing the previous consensus keys for the chain.
	// each AVS can have a separate epoch and hence this function is a part of this module
	// and not the operator module.
	// as a reminder, this refers solely to the consensus key used in the previous epoch,
	// which is tracked by x/operator to tell us if it's a new opt-in or a key replacement.
	k.operatorKeeper.ClearPreviousConsensusKeys(ctx, chainIDWithoutRevision)
	// let the operator module know that the opt out has finished.
	optOuts := k.GetPendingOptOuts(ctx)
	failed := []sdk.AccAddress{}
	for _, addr := range optOuts.GetList() {
		addrAcc := sdk.AccAddress(addr)
		err := k.operatorKeeper.CompleteOperatorKeyRemovalForChainID(
			ctx, addrAcc, chainIDWithoutRevision,
		)
		if err != nil {
			// the errors returned by the function are, as of writing,
			// 1. ErrOperatorNotExist
			// 2. ErrUnknownChainID
			// 3. ErrOperatorNotRemovingKey
			// none of these errors, in a consistent state machine with
			// perfect logic, should happen. however, we guard against
			// them anyway
			logger.Error(
				"error completing operator key removal",
				"error", err,
				"addr", addrAcc.String(),
				"rescheduled to end of epoch", fmt.Sprintf("%d", nextEpochNumber),
				"identifier", epochID,
			)
			failed = append(failed, addrAcc)
		}
	}
	// reschedule these for the next epoch
	for _, addr := range failed {
		k.AppendOptOutToFinish(ctx, nextEpochNumber, addr)
		k.SetOperatorOptOutFinishEpoch(ctx, addr, nextEpochNumber)
	}
	k.ClearPendingOptOuts(ctx)
	// for slashing, the operator module is required to store a mapping of chain id + cons addr
	// to operator address. this information can now be pruned, since the opt out is considered
	// complete.
	consensusAddrs := k.GetPendingConsensusAddrs(ctx)
	failedConsAddrs := []sdk.ConsAddress{}
	for _, consensusAddr := range consensusAddrs.GetList() {
		consAddr := sdk.ConsAddress(consensusAddr)
		cc, writeFunc := ctx.CacheContext()
		// tell the slashing module to delete look up from consensus addr to pub key
		if err := k.Hooks().AfterValidatorRemoved(
			cc, consAddr, nil,
		); err != nil {
			logger.Error(
				"error in AfterValidatorRemoved hook",
				"err", err,
				"consensusAddr", consAddr.String(),
			)
			failedConsAddrs = append(failedConsAddrs, consensusAddr)
			continue
		}
		k.operatorKeeper.DeleteOperatorAddressForChainIDAndConsAddr(
			cc, chainIDWithoutRevision, consAddr,
		)
		// clear old signed vs missed blocks
		// signing info is never cleared since it contains tombstone status
		k.slashingKeeper.ClearValidatorMissedBlockBitArray(cc, consAddr)
		writeFunc()
	}
	// reschedule these for the next epoch
	for _, addr := range failedConsAddrs {
		k.AppendConsensusAddrToPrune(ctx, nextEpochNumber, addr)
	}
	k.ClearPendingConsensusAddrs(ctx)
	// finally, perform the actual operations of vote power changes.
	// 1. find all operator keys for the chain.
	// 2. find last stored operator keys + their powers.
	// 3. find newest vote power for the operator keys, and sort them.
	// 4. loop through #1 and see if anything has changed.
	//    if it hasn't, do nothing for that operator key.
	//    if it has, queue an update.
	// 5. keep in mind the total vote power.
	totalPower := math.ZeroInt()
	prevList := k.GetAllImuachainValidators(ctx)
	// prevMap is a map of the previous validators, indexed by the consensus address
	// and the value being the vote power.
	prevMap := make(map[string]int64, len(prevList))
	for _, validator := range prevList {
		pubKey, err := validator.ConsPubKey()
		if err != nil {
			// indicates an error in deserialization, and should never happen.
			logger.Error("error deserializing consensus public key", "error", err)
			continue
		}
		addressString := sdk.GetConsAddress(pubKey).String()
		prevMap[addressString] = validator.Power
	}
	operators, keys := k.operatorKeeper.GetActiveOperatorsForChainID(ctx, chainIDWithoutRevision)
	powers, err := k.operatorKeeper.GetVotePowerForChainID(
		ctx, operators, chainIDWithoutRevision,
	)
	if err != nil {
		logger.Error("error getting vote power for chain", "error", err)
		return []abci.ValidatorUpdate{}
	}
	operators, keys, powers = utils.SortByPower(operators, keys, powers)
	maxVals := k.GetMaxValidators(ctx)
	logger.Info("before loop", "maxVals", maxVals, "len(operators)", len(operators))
	// the capacity of this list is twice the maximum number of validators.
	// this is because we can have a maximum of maxVals validators, and we can also have
	// a maximum of maxVals validators that are removed.
	res := make([]keytypes.WrappedConsKeyWithPower, 0, maxVals*2)
	for i := range operators {
		logger.Debug("loop", i)
		// #nosec G701 // ok on 64-bit systems.
		if i >= int(maxVals) {
			// we have reached the maximum number of validators, amongst all the validators.
			// even if there are intersections with the previous validator set, this will
			// only be reached if we exceed the threshold.
			// if there are no intersections, this case is glaringly obvious.
			logger.Debug("max validators reached", "i", i)
			break
		}
		power := powers[i]
		if power < 1 {
			// we have reached the bottom of the rung.
			// assumption is that negative vote power isn't provided by the module.
			// the consensus engine will reject it anyway and panic.
			logger.Debug("power less than 1", "i", i)
			break
		}
		// find the previous power.
		wrappedKey := keys[i]
		addressString := wrappedKey.ToConsAddr().String()
		prevPower, found := prevMap[addressString]
		if found {
			// if the power has changed, queue an update. skip, otherwise.
			if prevPower != power {
				logger.Debug(
					"power changed",
					"i", i,
					"operator", operators[i].String(),
					"power", power,
					"prevPower", prevPower,
				)
				res = append(res, keytypes.WrappedConsKeyWithPower{
					Key:   wrappedKey,
					Power: power,
				})
			} else {
				logger.Debug(
					"power not changed",
					"i", i,
					"operator", operators[i].String(),
					"power", power,
				)
			}
			// remove the validator from the previous map, so that 0 power
			// is not queued for it.
			delete(prevMap, addressString)
		} else {
			// new consensus key, queue an update.
			res = append(res, keytypes.WrappedConsKeyWithPower{
				Key:   wrappedKey,
				Power: power,
			})
			logger.Debug(
				"new validator",
				"i", i,
				"operator", operators[i].String(),
				"power", power,
			)
		}
		// all powers, regardless of whether the key exists, are added to the total power.
		totalPower = totalPower.Add(sdk.NewInt(power))
	}
	logger.Info(
		"before removal",
		"totalPower", totalPower,
		"len(res)", len(res),
	)
	// the remaining validators in prevMap have been removed.
	// we need to queue a change in power to 0 for them.
	for _, validator := range prevList { // O(N)
		// #nosec G703 // already checked in the previous iteration over prevList.
		pubKey, _ := validator.ConsPubKey()
		addressString := sdk.GetConsAddress(pubKey).String()
		// Check if this validator is still in prevMap (i.e., hasn't been deleted)
		if _, exists := prevMap[addressString]; exists { // O(1) since hash map
			res = append(res, keytypes.WrappedConsKeyWithPower{
				Key:   keytypes.NewWrappedConsKeyFromSdkKey(pubKey),
				Power: 0,
			})
			// while calculating total power, we started with 0 and not previous power.
			// so the previous power of these validators does not need to be subtracted.
		}
	}
	logger.Info(
		"after removal",
		"len(res)", len(res),
	)
	// if there are any updates, set total power on lookup index.
	if len(res) > 0 {
		k.SetLastTotalPower(ctx, totalPower)
	}

	// call via wrapper function so that validator info is stored.
	return k.ApplyValidatorChanges(ctx, res)
}
