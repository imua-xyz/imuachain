package feedermanagement

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"slices"
	"sort"
	"strconv"
	"strings"

	sdkerrors "cosmossdk.io/errors"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/imua-xyz/imuachain/x/oracle/keeper/common"
	oracletypes "github.com/imua-xyz/imuachain/x/oracle/types"

	"github.com/cometbft/cometbft/libs/log"
)

func NewFeederManager(k common.KeeperOracle) *FeederManager {
	return &FeederManager{
		k:                           k,
		sortedFeederIDs:             make([]int64, 0),
		rounds:                      make(map[int64]*round),
		cs:                          nil,
		phaseTwoCollectingFeederIDs: make(map[uint64]uint64),
	}
}

//nolint:revive
func (f *FeederManager) GetCaches() *caches {
	return f.cs
}

func (f *FeederManager) InitCachesForTest(k Submitter, params *oracletypes.Params, validators map[string]*big.Int) {
	f.cs = newCaches()
	f.cs.Init(k, params, validators)
}

func (f *FeederManager) GetParamsFromCache() *oracletypes.Params {
	return f.cs.params.params
}

func (f *FeederManager) GetMaxNonceFromCache() int32 {
	return f.cs.GetMaxNonce()
}

func (f *FeederManager) GetMaxSizePricesFromCache() int32 {
	return f.cs.GetMaxSizePrices()
}

func (f *FeederManager) GetTokenIDForFeederID(feederID int64) (int64, bool) {
	return f.cs.GetTokenIDForFeederID(feederID)
}

func (f *FeederManager) SetKeeper(k common.KeeperOracle) {
	f.k = k
}

func (f *FeederManager) SetNilCaches() {
	f.cs = nil
}

// BeginBlock initializes the caches and slashing records, and set up the rounds
func (f *FeederManager) BeginBlock(ctx sdk.Context) (recovered bool) {
	// if the cache is nil and we are not in recovery mode, init the caches
	if f.cs == nil {
		var err error
		recovered, err = f.recovery(ctx) // it's safe to panic since this will only happen when the node is starting with something wrong in the store
		if err != nil {
			panic(err)
		}
		// init feederManager if recovery failed, this should only happen on block_height==1
		if !recovered {
			f.initCaches(ctx)
			f.SetParamsUpdated()
			f.SetValidatorsUpdated()
		}
		f.initBehaviorRecords(ctx, ctx.BlockHeight())
		// in recovery mode, snapshot of feederManager is set in the beginblock instead of in the process of replaying endblockInrecovery
		// TODO: remove this into recovery, and call separately for init mode, that would lead to write updateCheckTx twice, but more clear
		f.updateCheckTx()
	}
	return
}

func (f *FeederManager) EndBlock(ctx sdk.Context) {
	// update params and validator set if necessary in caches and commit all updated information
	addedValidators := f.updateAndCommitCaches(ctx)

	// update Slashing related records (reportInfo, missCountBitArray), handle case for 1. resetSlashing, 2. new validators added for validatorset change
	f.updateBehaviorRecordsForNextBlock(ctx, addedValidators)

	// update rounds including create new rounds based on params change, remove expired rounds
	// handleQuotingBehavior for ending quotes of rounds
	// commit state of mature rounds
	f.updateAndCommitRounds(ctx)

	// set status to open for rounds before their quoting window
	feederIDs := f.prepareRounds(ctx)
	// remove nonces for closing quoting-window and set nonces for opening quoting-window
	f.setupNonces(ctx, feederIDs)

	f.ResetFlags()

	f.resetPhaseTwoCollectingFeederIDs()

	f.resetPhaseTwoMaliciousTx()

	f.updateCheckTx()
}

func (f *FeederManager) EndBlockInRecovery(ctx sdk.Context, params *oracletypes.Params) {
	if params != nil {
		f.SetParamsUpdated()
		_ = f.cs.AddCache(params)
	}
	f.updateAndCommitRoundsInRecovery(ctx)
	f.prepareRounds(ctx)
	f.ResetFlags()
	f.resetPhaseTwoCollectingFeederIDs()
	// updateCheckTx() is invoked in BeginBlock either in recovery or init mode, so we skip that in EndBlockRecovery
}

func (f *FeederManager) setupNonces(ctx sdk.Context, feederIDs []int64) {
	logger := f.k.Logger(ctx)
	height := ctx.BlockHeight()
	// the order does not matter, it's safe to update independent state in non-deterministic order
	// no need to go through all 'hash' process to range sorted key slice
	feederIDsUint64 := make([]uint64, 0, len(f.rounds))
	for _, r := range f.rounds {
		// remove nonces for closed quoting windows or when forceSeal is marked
		if r.IsQuotingWindowEnd(height) || f.forceSeal {
			logger.Debug("clear nonces for closing quoting window or forceSeal",
				"feederID", r.feederID, "roundID", r.roundID, "basedBlock", r.roundBaseBlock, "height", height, "forceSeal", f.forceSeal)
			// items will be removed from slice and keep the order, so it's safe to delete items in different order
			// #nosec G115  // feederID is index of slice
			feederIDsUint64 = append(feederIDsUint64, uint64(r.feederID))
		}
	}

	if len(feederIDsUint64) > 0 {
		if f.forceSeal {
			f.k.RemoveNonceWithFeederIDsForAll(ctx, feederIDsUint64)
		} else {
			f.k.RemoveNonceWithFeederIDsForValidators(ctx, feederIDsUint64, f.cs.GetValidators())
		}
	}

	if len(feederIDs) == 0 {
		return
	}
	// setup nonces for opening quoting windows
	// items need to be insert into slice in order, so feederIDs is sorted
	sort.Slice(feederIDs, func(i, j int) bool { return feederIDs[i] < feederIDs[j] })
	validators := f.cs.GetValidators()
	feederIDsUint64 = make([]uint64, 0, len(feederIDs))
	for _, feederID := range feederIDs {
		r := f.rounds[feederID]
		logger.Debug("init nonces for new quoting window",
			"feederID", feederID, "roundID", r.roundID, "basedBlock", r.roundBaseBlock, "height", height)
		// #nosec G115 -- feederID is index of slice
		feederIDsUint64 = append(feederIDsUint64, uint64(feederID))
	}
	f.k.AddZeroNonceItemWithFeederIDsForValidators(ctx, feederIDsUint64, validators)
}

func (f *FeederManager) initBehaviorRecords(ctx sdk.Context, height int64) {
	if !f.validatorsUpdated {
		return
	}
	validators := f.cs.GetValidators()
	for _, validator := range validators {
		f.k.InitValidatorReportInfo(ctx, validator, height)
	}
}

func (f *FeederManager) updateBehaviorRecordsForNextBlock(ctx sdk.Context, addedValidators []string) {
	height := ctx.BlockHeight() + 1
	if f.resetSlashing {
		// reset all validators' reportInfo
		f.k.ClearAllValidatorReportInfo(ctx)
		f.k.ClearAllValidatorMissedRoundBitArray(ctx)
		validators := f.cs.GetValidators()
		// order does not matter for independent state update
		for _, validator := range validators {
			f.k.InitValidatorReportInfo(ctx, validator, height)
		}
	} else if f.validatorsUpdated {
		// order does not matter for independent state update
		for _, validator := range addedValidators {
			// add possible new added validator info for slashing tracking
			f.k.InitValidatorReportInfo(ctx, validator, height)
		}
	}
}

// prepareRounds prepares the rounds for the next block, and returns the feederIDs of the rounds that are open on next block
func (f *FeederManager) prepareRounds(ctx sdk.Context) []int64 {
	logger := f.k.Logger(ctx)
	feederIDs := make([]int64, 0)
	height := ctx.BlockHeight()
	// it's safe to range map directly, this is just used to update memory state
	for _, r := range f.rounds {
		if open := r.PrepareForNextBlock(ctx.BlockHeight()); open {
			feederIDs = append(feederIDs, r.feederID)
			// logs might not be displayed in order, it's marked with [mem] to indicate that this is a memory state update
			logger.Info("[mem] open quoting window for round",
				"feederID", r.feederID, "roundID", r.roundID, "basedBlock", r.roundBaseBlock, "height", height)
		}
	}
	return feederIDs
}

// 1. update and commit Params if updated
// 2. update and commit validatorPowers if updated
// forceSeal: 1. params has some modifications related to quoting. 2.validatorSet changed
// resetSlashing: params has some modifications related to oracle_slashing
func (f *FeederManager) updateAndCommitCaches(ctx sdk.Context) (activeValidators []string) {
	// update params in caches
	if f.paramsUpdated {
		paramsOld := &oracletypes.Params{}
		f.cs.Read(paramsOld)
		params := f.k.GetParams(ctx)
		if paramsOld.IsForceSealingUpdate(&params) {
			f.SetForceSeal()
		}
		if paramsOld.IsSlashingResetUpdate(&params) {
			f.SetResetSlashing()
		}
		_ = f.cs.AddCache(&params)
	}

	// update validators
	validatorUpdates := f.k.GetValidatorUpdates(ctx)
	if len(validatorUpdates) > 0 {
		f.SetValidatorsUpdated()
		f.SetForceSeal()
		activeValidators = make([]string, 0)
		validatorMap := make(map[string]*big.Int)
		for _, vu := range validatorUpdates {
			pubKey, _ := cryptocodec.FromTmProtoPublicKey(vu.PubKey)
			validatorStr := sdk.ConsAddress(pubKey.Address()).String()
			validatorMap[validatorStr] = big.NewInt(vu.Power)
			if vu.Power > 0 {
				activeValidators = append(activeValidators, validatorStr)
			}
		}
		// update validator set information in cache
		_ = f.cs.AddCache(ItemV(validatorMap))
	}

	// commit caches: msgs is exists, params if updated, validatorPowers is updated
	_, vUpdated, pUpdated := f.cs.Commit(ctx, false)
	if vUpdated || pUpdated {
		f.k.Logger(ctx).Info("update caches", "validatorUpdated", vUpdated, "paramsUpdated", pUpdated)
		if pUpdated {
			ctx.EventManager().EmitEvent(sdk.NewEvent(
				oracletypes.EventTypeOracleUpdateParams,
				sdk.NewAttribute(oracletypes.AttributeKeyParamsUpdated, oracletypes.AttributeValueSuccess),
			))
		}
	}
	return activeValidators
}

func (f *FeederManager) commitRoundsInRecovery() {
	// safe to range map directly, this is just used to update memory state, we don't update state in recovery mode
	for _, r := range f.rounds {
		if r.Committable() {
			r.FinalPrice()
			r.status = roundStatusClosed
		}
		// close all quotingWindow to skip current rounds' 'handleQuotingMisBehavior'
		if f.forceSeal {
			r.closeQuotingWindow()
		}
	}
}

func (f *FeederManager) processRound(ctx sdk.Context, feederID, height int64, logger log.Logger) (success bool) {
	if feederID == 0 || feederID > int64(len(f.rounds)) {
		logger.Error("invalid feederID", "feederID", feederID)
		return success
	}

	r := f.rounds[feederID]
	if f.forceSeal {
		defer func() {
			// close all quotingWindow to skip current rounds' 'handlQuotingMisBehavior'
			r.closeQuotingWindow()
			if r.twoPhases && r.m != nil {
				// #nosec G115
				f.k.Clear2ndPhase(ctx, uint64(feederID), r.m.RootIndex())
				r.m = nil
			}
		}()
	}

	if r.Committable() {
		// just set status to close, and keep aggregator for possible 'handleQuotingMisBehavior' at quotingWindowEnd
		r.status = roundStatusClosed
		finalPrice, ok := r.FinalPrice()
		if !ok {
			logger.Info("commit round with price from previous",
				"feederID", r.feederID, "roundID", r.roundID, "baseBlock", r.roundBaseBlock, "height", height)
			// #nosec G115  // tokenID is index of slice
			f.k.GrowRoundID(ctx, uint64(r.tokenID), uint64(r.roundID))
			return success
		}
		if !f.cs.IsRuleV1(r.feederID) {
			logger.Error("We currently only support rules under oracle V1", "feederID", r.feederID)
			return success
		}
		priceCommit := finalPrice.ProtoPriceTimeRound(r.roundID, ctx.BlockTime().Format(oracletypes.TimeLayout))
		logger.Info("commit round with aggregated price",
			"feederID", r.feederID, "roundID", r.roundID, "baseBlock", r.roundBaseBlock, "price", priceCommit, "height", height)

		// #nosec G115  // tokenID is index of slice
		if updated := f.k.AppendPriceTR(ctx, uint64(r.tokenID), *priceCommit); !updated {
			// this is an 'impossible' case, we should not reach here
			latestPrice, latestRoundID := f.k.GrowRoundID(ctx, uint64(r.tokenID), uint64(r.roundID))
			logger.Error("failed to append price due to roundID gap and update this round with GrowRoundID",
				"feederID", r.feederID, "try-to-update-roundID", r.roundID, "try-to-update-price", priceCommit,
				"result-latestPrice", latestPrice, "result-latestRoundID", latestRoundID)
		} else {
			success = true
			// set up for 2-phases aggregation
			if r.twoPhases {
				rootHash := []byte(finalPrice.Price[:32])
				tmp := finalPrice.Price[32:]
				leafCount, err := strconv.ParseUint(tmp, 10, 32)
				// this should not happen, the format is guarded by anteHandler
				if err != nil {
					logger.Error("failed to parse leafCount from finalPrice", "feederID", r.feederID, "error", err)
					return success
				}
				// set up mem-round for 2nd phase aggregation
				r.m, err = oracletypes.NewMT(f.cs.RawDataPieceSize(), uint32(leafCount), rootHash)
				if err != nil {
					logger.Error("failed to create merkle tree", "feederID", r.feederID, "error", err)
					return success
				}
				// set up state for 2nd phase aggregation
				// #nosec G115
				logger.Info("set up 2ndPhase on successful 1stPhase aggregation",
					"feederID", r.feederID, "rootHash", hex.EncodeToString([]byte(finalPrice.Price)), "leafCount", finalPrice.DetID)
				if err := f.k.Setup2ndPhase(ctx, uint64(r.feederID), f.cs.GetValidators(), uint32(leafCount), rootHash); err != nil {
					logger.Error("failed to setup 2ndPhase on successful 1stPhase aggregation", "feederID", r.feederID, "error", err)
				}
			}
		}
		return success
	}
	if r.twoPhases {
		// check if r is 2-phases and rawData is completed, for 2nd-phase, the status of round must be closed
		if r.m.CollectingRawData() {
			if len(r.cachedProofForBlock) > 0 {
				// #nosec G115
				f.k.AddNodesToMerkleTree(ctx, uint64(r.feederID), r.cachedProofForBlock)
				// reset cachedProofForBlock after commit to state
				r.cachedProofForBlock = nil
			}
			if LatestLeafIndex, ok := r.m.LatestLeafIndex(); ok {
				// #nosec G115
				f.k.SetNextPieceIndexForFeeder(ctx, uint64(r.feederID), LatestLeafIndex+1)
			}
			return success
		}
		if rawData, ok := r.m.CompleteRawData(); ok {
			rootHash := r.m.RootHash()
			logger.Info("execute postHandler after 2ndPhase completed collecting rawData",
				"feederID", r.feederID, "rootHash", base64.StdEncoding.EncodeToString(rootHash), "leafCount", r.m.LeafCount())
			// execute pootHandler with rawData
			// #nosec G115
			if err := r.h(ctx, rootHash, rawData, uint64(r.feederID), uint64(r.roundID), f.k); err != nil {
				// just log the error and wait for next round to update
				// TODO(leonz): this suites for NST, we can just wait for next round to update, but does it suites for commmon case ? should we do some other postHandling for this fail when it's not of NST case?
				logger.Error("failed to execute postHandler for 2phases aggregation on consensus price",
					"feederID", r.feederID, "roundID", r.roundID, "consensus_1st-phase-hash", hex.EncodeToString(r.m.RootHash()), "error", err)
			}
			// reset related cache from state
			// #nosec G115
			f.k.Clear2ndPhase(ctx, uint64(r.feederID), r.m.RootIndex())
			r.m = nil
		}
	}
	return success
}

func (f *FeederManager) commitRounds(ctx sdk.Context) {
	logger := f.k.Logger(ctx)
	height := ctx.BlockHeight()
	successFeederIDs := make([]string, 0)
	// it's safe to range map directly since the sate update is independent for each feederID, however we use sortedFeederIDs to keep the order of logs
	// this can be replaced by map iteration directly when better performance is needed
	for _, feederID := range f.sortedFeederIDs {
		if f.processRound(ctx, feederID, height, logger) {
			successFeederIDs = append(successFeederIDs, strconv.FormatInt(feederID, 10))
		}
	}
	if len(successFeederIDs) > 0 {
		feederIDsStr := strings.Join(successFeederIDs, "_")
		ctx.EventManager().EmitEvent(sdk.NewEvent(
			oracletypes.EventTypeCreatePrice,
			sdk.NewAttribute(oracletypes.AttributeKeyPriceUpdated, oracletypes.AttributeValueTrue),
			sdk.NewAttribute(oracletypes.AttributeKeyFeederIDs, feederIDsStr),
		))
	}
}

func (f *FeederManager) handleQuotingMisBehaviorInRecovery(ctx sdk.Context) {
	height := ctx.BlockHeight()
	logger := f.k.Logger(ctx)
	// it's safe to range map directly, no state in kvStore will be updated in recovery mode, only memory state is updated
	for _, r := range f.rounds {
		if r.IsQuotingWindowEnd(height) && r.a != nil {
			validators := f.cs.GetValidators()
			for _, validator := range validators {
				_, found := f.k.GetValidatorReportInfo(ctx, validator)
				if !found {
					logger.Error(fmt.Sprintf("Expected report info for validator %s but not found", validator))
					continue
				}
				_, malicious := r.PerformanceReview(validator)
				if malicious {
					r.getFinalDetIDForSourceID(oracletypes.SourceChainlinkID)
					r.FinalPrice()
				}
			}
			r.closeQuotingWindow()
		}
	}
}

func (f *FeederManager) handleMalicious(ctx sdk.Context, logger log.Logger, validator string, logInfo []any, rawData bool) {
	height := ctx.BlockHeight()
	logger.Info(
		"confirmed malicious",
		append(
			logInfo,
			"validator", validator,
			"infraction_height", height,
			"infraction_time", ctx.BlockTime(),
		)...,
	)
	consAddr, err := sdk.ConsAddressFromBech32(validator)
	if err != nil {
		logger.Error("when performing oracle_performance_review, got invalid consAddr string. This should never happen", "validatorStr", validator)
	}
	operator := f.k.ValidatorByConsAddr(ctx, consAddr)
	if operator != nil && !operator.IsJailed() {
		reportedInfo, found := f.k.GetValidatorReportInfo(ctx, validator)
		if !found {
			logger.Error(fmt.Sprintf("Expected report info for validator %s but not found", validator))
			return
		}
		power, _ := f.cs.GetPowerForValidator(validator)
		coinsBurned := f.k.SlashWithInfractionReason(ctx, consAddr, height, power.Int64(), f.k.GetSlashFractionMalicious(ctx), stakingtypes.Infraction_INFRACTION_UNSPECIFIED)
		var reason string
		if rawData {
			reason = oracletypes.AttributeValueMaliciousReportPiece
		} else {
			reason = oracletypes.AttributeValueMaliciousReportPrice
		}
		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				oracletypes.EventTypeOracleSlash,
				sdk.NewAttribute(oracletypes.AttributeKeyValidatorKey, validator),
				sdk.NewAttribute(oracletypes.AttributeKeyPower, fmt.Sprintf("%d", power)),
				sdk.NewAttribute(oracletypes.AttributeKeyReason, reason),
				sdk.NewAttribute(oracletypes.AttributeKeyJailed, validator),
				sdk.NewAttribute(oracletypes.AttributeKeyBurnedCoins, coinsBurned.String()),
			),
		)
		f.k.Jail(ctx, consAddr)
		jailUntil := ctx.BlockHeader().Time.Add(f.k.GetMaliciousJailDuration(ctx))
		f.k.JailUntil(ctx, consAddr, jailUntil)

		reportedInfo.MissedRoundsCounter = 0
		reportedInfo.IndexOffset = 0
		f.k.ClearValidatorMissedRoundBitArray(ctx, validator)
		f.k.SetValidatorReportInfo(ctx, validator, reportedInfo)
	}
}

func (f *FeederManager) handleMissCount(ctx sdk.Context, logger log.Logger, validator string, minReportedPerWindow, reportedRoundsWindow int64, logInfo []any, miss, rawData bool) {
	height := ctx.BlockHeight()

	reportedInfo, found := f.k.GetValidatorReportInfo(ctx, validator)
	if !found {
		logger.Error(fmt.Sprintf("Expected report info for validator %s but not found", validator))
		return
	}

	// #nosec G115
	index := uint64(reportedInfo.IndexOffset % reportedRoundsWindow)
	reportedInfo.IndexOffset++
	// Update reported round bit array & counter
	// This counter just tracks the sum of the bit array
	// That way we avoid needing to read/write the whole array each time
	previous := f.k.GetValidatorMissedRoundBitArray(ctx, validator, index)
	switch {
	case !previous && miss:
		// Array value has changed from not missed to missed, increment counter
		f.k.SetValidatorMissedRoundBitArray(ctx, validator, index, true)
		reportedInfo.MissedRoundsCounter++
	case previous && !miss:
		// Array value has changed from missed to not missed, decrement counter
		f.k.SetValidatorMissedRoundBitArray(ctx, validator, index, false)
		reportedInfo.MissedRoundsCounter--
	default:
		// Array value at this index has not changed, no need to update counter
	}

	if miss {
		proposer := ""
		if rawData {
			proposer = validator
		}
		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				oracletypes.EventTypeOracleLiveness,
				sdk.NewAttribute(oracletypes.AttributeKeyValidatorKey, validator),
				sdk.NewAttribute(oracletypes.AttributeKeyMissedRounds, fmt.Sprintf("%d", reportedInfo.MissedRoundsCounter)),
				sdk.NewAttribute(oracletypes.AttributeKeyHeight, fmt.Sprintf("%d", height)),
				sdk.NewAttribute(oracletypes.AttributeKeyProposer, proposer),
			),
		)

		logger.Info(
			"oracle_absent validator",
			append(
				logInfo,
				"height", height,
				"validator", validator,
				"missed", reportedInfo.MissedRoundsCounter,
				"threshold", minReportedPerWindow,
			)...,
		)
	}

	minHeight := reportedInfo.StartHeight + reportedRoundsWindow
	maxMissed := reportedRoundsWindow - minReportedPerWindow
	// if we are past the minimum height and the validator has missed too many rounds reporting prices, punish them
	if height > minHeight && reportedInfo.MissedRoundsCounter > maxMissed {
		consAddr, err := sdk.ConsAddressFromBech32(validator)
		if err != nil {
			f.k.Logger(ctx).Error("when performing oracle_performance_review, got invalid consAddr string. This should never happen", "validatorStr", validator)
			return
		}
		operator := f.k.ValidatorByConsAddr(ctx, consAddr)
		if operator != nil && !operator.IsJailed() {
			// missing rounds confirmed: just jail the validator
			f.k.Jail(ctx, consAddr)
			jailUntil := ctx.BlockHeader().Time.Add(f.k.GetMissJailDuration(ctx))
			f.k.JailUntil(ctx, consAddr, jailUntil)

			// We need to reset the counter & array so that the validator won't be immediately slashed for miss report info upon rebonding.
			reportedInfo.MissedRoundsCounter = 0
			reportedInfo.IndexOffset = 0
			f.k.ClearValidatorMissedRoundBitArray(ctx, validator)

			logger.Info(
				"jailing validator due to oracle_liveness fault",
				append(
					logInfo,
					"height", height,
					"validator", consAddr.String(),
					"min_height", minHeight,
					"threshold", minReportedPerWindow,
					"jailed_until", jailUntil,
				)...,
			)
		} else {
			// validator was (a) not found or (b) already jailed so we do not slash
			logger.Info(
				"validator would have been jailed for too many missed reporting price, but was either not found in store or already jailed",
				"validator", validator,
			)
		}
	}
	// Set the updated reportInfo
	f.k.SetValidatorReportInfo(ctx, validator, reportedInfo)
}

func (f *FeederManager) handleQuotingMisBehavior(ctx sdk.Context) {
	height := ctx.BlockHeight()
	logger := f.k.Logger(ctx)

	// it's safe to range map directly, each state update is independent for each feederID
	// state to be updated: {validatorReportInfo, validatorMissedRoundBitArray, signInfo, assets} of individual validator
	// we use sortedFeederIDs to keep the order of logs
	// this can be replaced by map iteration directly when better performance is needed
	minReportedPerWindow := f.k.GetMinReportedPerWindow(ctx)
	reportedRoundsWindow := f.k.GetReportedRoundsWindow(ctx)

	// handle malicious tx for phase-2 of 2-phases aggregation
	if len(f.phaseTwoMaliciousTx) > 0 {
		keysMaliciousTx := make([]uint64, 0, len(f.phaseTwoMaliciousTx))
		for fID := range f.phaseTwoMaliciousTx {
			keysMaliciousTx = append(keysMaliciousTx, fID)
		}
		slices.Sort(keysMaliciousTx)
		// we use sorted keys to handle the malicious slash&jail though we don't see any dependency on the order
		for _, fID := range keysMaliciousTx {
			validator := f.phaseTwoMaliciousTx[fID]
			// #nosec G115
			logInfo := []any{"validator submit malicious piece of rawData", validator, "feederID", fID, "roundID", f.rounds[int64(fID)].roundID}
			f.handleMalicious(ctx, logger, validator, logInfo, true)
		}
	}

	for _, feederID := range f.sortedFeederIDs {
		// #nosec G115
		if _, ok := f.phaseTwoCollectingFeederIDs[uint64(feederID)]; ok {
			if ctx.BlockHeight() < int64(f.rounds[feederID].roundPhaseTwoCheckingBlock) {
				continue
			}
			if _, ok := f.phaseTwoCollectingFeederIDs[uint64(feederID)]; ok {
				r := f.rounds[feederID]
				consAddrStr := sdk.ConsAddress(ctx.BlockHeader().ProposerAddress).String()
				logInfo := []any{"proposer", consAddrStr, "missed_rawData_feederID", feederID, "roundID", r.roundID}
				f.handleMissCount(ctx, logger, consAddrStr, minReportedPerWindow, reportedRoundsWindow, logInfo, true, true)
			}
			// this feederID is collecting piece and there's no tx included by the proposer for current block
			continue
		}
		r := f.rounds[feederID]
		if r.IsQuotingWindowEnd(height) {
			if _, found := r.FinalPrice(); !found {
				r.closeQuotingWindow()
				continue
			}
			validators := f.cs.GetValidators()
			for _, validator := range validators {
				miss, malicious := r.PerformanceReview(validator)
				if malicious {
					finalPrice, _ := r.FinalPrice()
					logInfo := []any{"feederID", feederID, "detID", r.getFinalDetIDForSourceID(oracletypes.SourceChainlinkID), "roundID", r.roundID, "finalPrice", finalPrice}
					f.handleMalicious(ctx, logger, validator, logInfo, false)
					continue
				}
				logInfo := []any{}
				f.handleMissCount(ctx, logger, validator, minReportedPerWindow, reportedRoundsWindow, logInfo, miss, false)
			}
			r.closeQuotingWindow()
		}
	}
}

// setCommittableState sets the status of rounds to committable if their quoting window has ended or if forceSeal is set.
func (f *FeederManager) setCommittableState(ctx sdk.Context) {
	if f.forceSeal {
		// safe to range map. update memory state only, the result would be the same in any order
		for _, r := range f.rounds {
			if r.status == roundStatusOpen {
				r.status = roundStatusCommittable
			}
		}
	} else {
		height := ctx.BlockHeight()
		// safe to range map. update memory state only, the result would be the same in any order
		for _, r := range f.rounds {
			if r.IsQuotingWindowEnd(height) && r.status == roundStatusOpen {
				r.status = roundStatusCommittable
			}
		}
	}
}

// updateRoundsParamsAndAddNewRounds updates round parameters and adds new rounds if params have changed.
func (f *FeederManager) updateRoundsParamsAndAddNewRounds(ctx sdk.Context) {
	height := ctx.BlockHeight()
	logger := f.k.Logger(ctx)

	if f.paramsUpdated {
		params := &oracletypes.Params{}
		f.cs.Read(params)
		existsFeederIDs := make(map[int64]struct{})
		// safe to range map. update memory state only, the result would be the same in any order
		for _, r := range f.rounds {
			r.UpdateParams(params.TokenFeeders[r.feederID], int64(params.MaxNonce))
			existsFeederIDs[r.feederID] = struct{}{}
		}
		// add new rounds
		for feederID, tokenFeeder := range params.TokenFeeders {
			if feederID == 0 {
				continue
			}
			feederID := int64(feederID)
			// #nosec G115
			if _, ok := existsFeederIDs[feederID]; !ok && (tokenFeeder.EndBlock == 0 || tokenFeeder.EndBlock > uint64(height)) {
				logger.Info("[mem] add new round", "feederID", feederID, "height", height)
				f.sortedFeederIDs = append(f.sortedFeederIDs, feederID)
				twoPhases := f.cs.IsRule2PhasesByFeederID(uint64(feederID))
				ph, _ := f.k.GetPostAggregation(feederID)
				f.rounds[feederID] = newRound(feederID, tokenFeeder, int64(params.MaxNonce), f.cs, NewAggMedian(), twoPhases, ph)
			}
		}
		f.sortedFeederIDs.sort()
	}
}

// removeExpiredRounds removes rounds that have expired based on the current block height.
func (f *FeederManager) removeExpiredRounds(ctx sdk.Context) {
	height := ctx.BlockHeight()
	expiredFeederIDs := make([]int64, 0)
	// safe to range map, we generate the slice, the content of elements would be the same, order does not matter
	for _, r := range f.rounds {
		if r.endBlock > 0 && r.endBlock <= height {
			expiredFeederIDs = append(expiredFeederIDs, r.feederID)
		}
	}
	// the order does not matter when remove item from slice as RemoveNonceWithFeederIDForAll does
	expiredFeederIDsToRemoveUint64 := make([]uint64, 0)
	for _, feederID := range expiredFeederIDs {
		if r := f.rounds[feederID]; r.status != roundStatusClosed {
			r.closeQuotingWindow()
			// #nosec G115
			expiredFeederIDsToRemoveUint64 = append(expiredFeederIDsToRemoveUint64, uint64(feederID))
		}
		delete(f.rounds, feederID)
		f.sortedFeederIDs.remove(feederID)
	}
	if len(expiredFeederIDsToRemoveUint64) > 0 {
		f.k.RemoveNonceWithFeederIDsForValidators(ctx, expiredFeederIDsToRemoveUint64, f.cs.GetValidators())
	}
}

// updateAndCommitRoundsInRecovery updates and commits rounds during recovery mode.
func (f *FeederManager) updateAndCommitRoundsInRecovery(ctx sdk.Context) {
	f.setCommittableState(ctx)
	f.commitRoundsInRecovery()
	f.handleQuotingMisBehaviorInRecovery(ctx)
	f.updateRoundsParamsAndAddNewRounds(ctx)
	f.removeExpiredRounds(ctx)
}

// updateAndCommitRounds updates and commits rounds during normal operation.
func (f *FeederManager) updateAndCommitRounds(ctx sdk.Context) {
	f.setCommittableState(ctx)
	f.commitRounds(ctx)
	// behaviors review and close quotingWindow
	f.handleQuotingMisBehavior(ctx)
	f.updateRoundsParamsAndAddNewRounds(ctx)
	f.removeExpiredRounds(ctx)
}

// ResetFlags resets the update flags for params, validators, forceSeal, and resetSlashing.
func (f *FeederManager) ResetFlags() {
	f.paramsUpdated = false
	f.validatorsUpdated = false
	f.forceSeal = false
	f.resetSlashing = false
}

// SetParamsUpdated marks that params have been updated in the current block.
func (f *FeederManager) SetParamsUpdated() {
	f.paramsUpdated = true
}

// SetValidatorsUpdated marks that validators have been updated in the current block.
func (f *FeederManager) SetValidatorsUpdated() {
	f.validatorsUpdated = true
}

// SetResetSlashing marks that slashing should be reset due to param changes.
func (f *FeederManager) SetResetSlashing() {
	f.resetSlashing = true
}

// SetForceSeal marks that all rounds should be force sealed.
func (f *FeederManager) SetForceSeal() {
	f.forceSeal = true
}

// validateMsg validates a MsgCreatePrice against the current state and round configuration.
func (f *FeederManager) validateMsg(ctx sdk.Context, msg *oracletypes.MsgCreatePrice) (*round, error) {
	// TODO:(leonz) ? this validation is not suitable for validateBasic, it need state information, but maybe move them into anteHandler ?
	// nonce, feederID, creator has been verified by anteHandler
	// baseBlock is going to be verified by its corresponding round
	decimal, err := f.cs.GetDecimalFromFeederID(msg.FeederID)
	if err != nil {
		return nil, err
	}
	for _, ps := range msg.Prices {
		// #nosec G115
		deterministic, err := f.cs.IsDeterministic(int64(ps.SourceID))
		if err != nil {
			return nil, err
		}
		l := len(ps.Prices)
		if deterministic {
			if l > int(f.cs.GetMaxNonce()) {
				return nil, fmt.Errorf("deterministic source:id_%d must provide no more than %d prices from different DetIDs, got:%d", ps.SourceID, f.cs.GetMaxNonce(), l)
			}
			for _, p := range ps.Prices {
				if len(p.DetID) == 0 {
					return nil, errors.New("detID of deterministic price must not be empty")
				}
				if p.Decimal != decimal {
					return nil, fmt.Errorf("decimal does not match for feederID:%d, expect:%d, got:%d", msg.FeederID, decimal, p.Decimal)
				}
			}
		} else {
			// NOTE: v1 does not actually have this type of sources
			if l != 1 {
				return nil, fmt.Errorf("non-deterministic sources should provide exactly one valid price, got:%d", len(ps.Prices))
			}
			p := ps.Prices[0]
			if p.Decimal != decimal {
				return nil, fmt.Errorf("decimal does not match for feederID:%d, expect:%d, got:%d", msg.FeederID, decimal, p.Decimal)
			}
			if len(p.DetID) > 0 {
				return nil, errors.New("price from non-deterministic should not have detID")
			}
		}
	}

	if f.cs.IsRule2PhasesByFeederID(msg.FeederID) && msg.IsSinglePhase() {
		return nil, fmt.Errorf("feederID:%d is configured for 2-phases aggregation, but the message is not of 2-phases", msg.FeederID)
	}
	// extra check for message as 1st phase for 2-phases aggregation
	if msg.IsPhaseTwo() {
		lPrice := len(msg.Prices[0].Prices[0].Price)
		if lPrice > int(f.cs.RawDataPieceSize()) {
			return nil, fmt.Errorf("message for 2nd-phase aggregation should have exactly one price with length between 1 and %d", f.cs.RawDataPieceSize())
		}
	}

	if msg.IsPhaseOne() {
		// validation had been done by msg.ValidateBasic
		leafCount, _ := strconv.ParseUint(msg.Prices[0].Prices[0].Price[32:], 10, 32)

		// we wait one more maxNonce blocks to make sure proposer getting expected txs in their mempool
		// we don't use the last block of current round(which is the baseBlock of the next round), so the quotingWindow for 2nd-phase message is from [baseBlock+2*maxNonce, nextBaseBlock-1]
		// #nosec G115  // maxNonce is positive
		interval, found := f.cs.IntervalForFeederID(msg.FeederID)
		if !found {
			return nil, fmt.Errorf("2-phases aggregation for feederID:%d, interval not found", msg.FeederID)
		}
		// #nosec G115  // maxNonce is positive
		windowForPhaseTwo := interval - uint64(f.cs.GetMaxNonce())*2
		if leafCount > windowForPhaseTwo {
			return nil, fmt.Errorf("2-phases aggregation for feederID:%d, should have detID less than or equal to %d and be at least 1, got%d",
				msg.FeederID, windowForPhaseTwo, leafCount)
		}
	}

	// stateful verify against round
	// #nosec G115 - TODO: use uint64 for rounds
	r, ok := f.rounds[int64(msg.FeederID)]
	if !ok {
		// This should not happened since we do check the nonce in anteHandler
		vAddr, _ := oracletypes.ConsAddrStrFromCreator(msg.Creator)
		return nil, fmt.Errorf("round not exists for feederID:%d, proposer:%s", msg.FeederID, vAddr)
	}

	// #nosec -G115
	if valid := r.ValidQuotingBaseBlock(int64(msg.BasedBlock), msg.IsSinglePhase()); !valid {
		return nil, fmt.Errorf("failed to process price-feed msg for feederID:%d, round is quoting:%t,quotingWindow is open:%t, expected baseBlock:%d, got baseBlock:%d, currentHeight:%d",
			msg.FeederID, r.IsQuoting(), r.IsQuotingWindowOpen(), r.roundBaseBlock, msg.BasedBlock, ctx.BlockHeight())
	}

	if r.twoPhases == msg.IsSinglePhase() {
		// this should not happen, since message itself had been checked in 'validateMsg', when came to here it means there' something wrong with mem-round initialization
		return nil, fmt.Errorf("the 2phases status of round and message is mismatched, there's something wrong with mem-round initialization, feederID:%d, r.IsTwoPhases:%t, msg.IsTwoPhases:%t",
			msg.FeederID, r.twoPhases, !msg.IsSinglePhase())
	}

	if msg.IsPhaseTwo() && (r.m == nil || r.m.Completed()) {
		return nil, fmt.Errorf("message with 2-nd phase for feederID:%d of round_%d is reject since that round is not collecting raw data", msg.FeederID, r.roundID)
	}

	return r, nil
}

// ProcessQuote processes a price quote message, validates it, tallies the result, and updates caches if needed.
func (f *FeederManager) ProcessQuote(ctx sdk.Context, msg *oracletypes.MsgCreatePrice, isCheckTx bool) (*oracletypes.PriceTimeRound, error) {
	if isCheckTx {
		f = f.getCheckTx()
	}
	var r *round
	var err error
	if r, err = f.validateMsg(ctx, msg); err != nil {
		return nil, oracletypes.ErrInvalidMsg.Wrap(err.Error())
	}

	msgItem := getProtoMsgItemFromQuote(msg)

	// tally msgItem
	finalPrice, validMsgItem, err := r.Tally(msgItem)

	// record msgItem in caches if needed
	defer func() {
		if !isCheckTx &&
			validMsgItem != nil &&
			(err == nil || sdkerrors.IsOf(err, oracletypes.ErrQuoteRecorded)) {
			_ = f.cs.AddCache(validMsgItem)
		}
	}()

	if err != nil {
		return nil, err
	}

	if finalPrice == nil {
		return nil, nil
	}
	return finalPrice.ProtoPriceTimeRound(r.roundID, ctx.BlockTime().Format(oracletypes.TimeLayout)), nil
}

// getCheckTx returns a copy of the FeederManager for use in CheckTx mode.
func (f *FeederManager) getCheckTx() *FeederManager {
	fCheckTx := f.fCheckTx
	ret := *fCheckTx
	ret.fCheckTx = nil

	// rounds
	rounds := make(map[int64]*round)
	// safe to range map, map copy
	for id, r := range fCheckTx.rounds {
		rounds[id] = r.CopyForCheckTx()
	}
	ret.rounds = rounds

	ret.phaseTwoCollectingFeederIDs = make(map[uint64]uint64)
	for feederID, startBaseBlock := range fCheckTx.phaseTwoCollectingFeederIDs {
		ret.phaseTwoCollectingFeederIDs[feederID] = startBaseBlock
	}

	// this remains empty all the process during checkTx
	ret.phaseTwoMaliciousTx = make(map[uint64]string)

	return &ret
}

// updateCheckTx updates the fCheckTx field with a shallow copy of the current FeederManager state for CheckTx/simulation.
func (f *FeederManager) updateCheckTx() {
	// flgas are taken care of
	// sortedFeederIDs will not be modified except in abci.EndBlock
	// successFeederIDs will not be modifed except in abci.EndBlock
	// caches will not be modifed except in abci.EndBlock, abci.DeliverTx (in abci.Query_simulate, or abci.CheckTx the update in ProcessQuote is forbided)
	// shallow copy is good enough for these fields

	ret := *f
	ret.fCheckTx = nil

	rounds := make(map[int64]*round)

	// safe to range map, map copy
	for id, r := range f.rounds {
		rounds[id] = r.CopyForCheckTx()
	}
	ret.rounds = rounds

	ret.phaseTwoCollectingFeederIDs = make(map[uint64]uint64)
	for feederID, startBaseBlock := range f.phaseTwoCollectingFeederIDs {
		ret.phaseTwoCollectingFeederIDs[feederID] = startBaseBlock
	}

	// phaseTwoMaliciousTx must be empty
	// the verification for simulation is skipped, so it's safe to ignore this, however we new a map for possible future update
	ret.phaseTwoMaliciousTx = make(map[uint64]string)

	f.fCheckTx = &ret
}

// ProcessQuoteInRecovery processes a batch of MsgItems during recovery mode, updating rounds as needed.
func (f *FeederManager) ProcessQuoteInRecovery(msgItems []*oracletypes.MsgItem) {
	for _, msgItem := range msgItems {
		// #nosec G115  // feederID is index of slice
		r, ok := f.rounds[int64(msgItem.FeederID)]
		if !ok {
			continue
		}
		// error deos not need to be handled in recovery mode
		//nolint:all
		r.Tally(msgItem)
	}
}

// initCaches initializes the caches of the FeederManager with keeper, params, and validator powers.
func (f *FeederManager) initCaches(ctx sdk.Context) {
	f.cs = newCaches()
	params := f.k.GetParams(ctx)
	validatorSet := f.k.GetAllImuachainValidators(ctx)
	validatorPowers := make(map[string]*big.Int)
	for _, v := range validatorSet {
		validatorPowers[sdk.ConsAddress(v.Address).String()] = big.NewInt(v.Power)
	}
	f.cs.Init(f.k, &params, validatorPowers)
}

// recovery attempts to recover the FeederManager state from recent params and validator updates.
func (f *FeederManager) recovery(ctx sdk.Context) (bool, error) {
	height := ctx.BlockHeight()
	recentParamsList, prevRecentParams, latestRecentParams := f.k.GetRecentParamsWithinMaxNonce(ctx)
	if latestRecentParams.Block == 0 {
		return false, nil
	}
	validatorUpdateBlock, found := f.k.GetValidatorUpdateBlock(ctx)
	if !found {
		// on recovery mode, the validator update block must be found
		return false, errors.New("validator update block not found in recovery mode for feeder manager")
	}
	// #nosec G115  // validatorUpdateBlock.Block represents blockheight
	startHeight, replayRecentParamsList := getRecoveryStartPoint(height, recentParamsList, &prevRecentParams, &latestRecentParams, int64(validatorUpdateBlock.Block))

	f.cs = newCaches()
	params := replayRecentParamsList[0].Params
	replayRecentParamsList = replayRecentParamsList[1:]

	validatorSet := f.k.GetAllImuachainValidators(ctx)
	validatorPowers := make(map[string]*big.Int)
	for _, v := range validatorSet {
		validatorPowers[sdk.ConsAddress(v.Address).String()] = big.NewInt(v.Power)
	}

	f.cs.Init(f.k, params, validatorPowers)

	replayHeight := startHeight - 1

	ctxReplay := ctx.WithBlockHeight(replayHeight)
	for tfID, tf := range params.TokenFeeders {
		if tfID == 0 {
			continue
		}
		// #nosec G115  // safe conversion
		if tf.EndBlock > 0 && int64(tf.EndBlock) <= replayHeight {
			continue
		}
		tfID := int64(tfID)
		// #nosec G115  // safe conversion
		twoPhases := f.cs.IsRule2PhasesByFeederID(uint64(tfID))
		postHandler, _ := f.k.GetPostAggregation(tfID)
		f.rounds[tfID] = newRound(tfID, tf, int64(params.MaxNonce), f.cs, NewAggMedian(), twoPhases, postHandler)
		f.sortedFeederIDs.add(tfID)
	}
	f.prepareRounds(ctxReplay)

	params = nil
	recentMsgs := f.k.GetAllRecentMsg(ctxReplay)
	for ; startHeight < height; startHeight++ {
		ctxReplay = ctxReplay.WithBlockHeight(startHeight)
		// only execute msgItems corresponding to rounds opened on or after replayHeight, since any rounds opened before replay height must be closed on or before height-1
		// which means no memory state need to be updated for thoes rounds
		// and we don't need to take care of 'close quoting-window' since the size of replay window t most equals to maxNonce
		// #nosec G115  // block is not negative
		if len(recentMsgs) > 0 && int64(recentMsgs[0].Block) <= startHeight {
			i := 0
			for idx, recentMsg := range recentMsgs {
				// #nosec G115  // block height is defined as int64 in cosmossdk
				if int64(recentMsg.Block) > startHeight {
					break
				}
				i = idx
				if int64(recentMsg.Block) == startHeight {
					f.ProcessQuoteInRecovery(recentMsg.Msgs)
					break
				}
			}
			recentMsgs = recentMsgs[i+1:]
		}
		// #nosec G115
		if len(replayRecentParamsList) > 0 && int64(replayRecentParamsList[0].Block) == startHeight {
			params = replayRecentParamsList[0].Params
			replayRecentParamsList = replayRecentParamsList[1:]
		}
		f.EndBlockInRecovery(ctxReplay, params)
	}

	f.cs.SkipCommit()

	pieceSize := f.cs.RawDataPieceSize()
	// recovery for 2nd-phase state
	for _, r := range f.rounds {
		if r.twoPhases {
			// reset r.m from state
			// #nosec G115
			feederID := uint64(r.feederID)
			// #nosec G115 - uint64 is more reasonable
			leafCount, rootHash := f.k.GetFeederTreeInfo(ctx, uint64(r.feederID))
			if leafCount == 0 {
				continue
			}
			r.m, _ = oracletypes.NewMT(pieceSize, leafCount, rootHash)
			// rawdata
			rawDataPieces, err := f.k.GetRawDataPieces(ctx, feederID)
			if err != nil {
				return false, err
			}
			r.m.SetRawDataPieces(rawDataPieces)
			// proof nodes
			// #nosec G115
			nodes := f.k.GetNodesFromMerkleTree(ctx, uint64(r.feederID))
			r.m.SetProofNodes(nodes)
		}
	}

	return true, nil
}

// RoundIDToBaseBlock returns the base block for a given feederID and roundID, if found.
func (f *FeederManager) RoundIDToBaseBlock(feederID, roundID uint64) (uint64, bool) {
	// #nosec G115
	r, ok := f.rounds[int64(feederID)]
	if !ok {
		return 0, false
	}
	return r.baseBlockFromRoundID(roundID)
}

// BaseBlockToNextRoundID returns the roundID for the given feederID and base block, if found.
func (f *FeederManager) BaseBlockToNextRoundID(feederID, baseBlock uint64) (uint64, bool) {
	// TODO(leonz): use uint64 as f.rounds key
	// #nosec G115
	r, ok := f.rounds[int64(feederID)]
	if !ok {
		return 0, false
	}
	// TODO(leonz): use uint64 for getPosition
	// #nosec G115
	b, rID, _, _ := r.getPosition(int64(baseBlock))
	// #nosec G115
	if uint64(b) != baseBlock {
		return 0, false
	}
	// #nosec G115
	return uint64(rID), true
}

// Equals compares two FeederManager instances for equality.
func (f *FeederManager) Equals(fm *FeederManager) bool {
	if f == nil || fm == nil {
		return f == fm
	}
	if f.fCheckTx == nil && fm.fCheckTx != nil {
		return false
	}
	if f.fCheckTx != nil && fm.fCheckTx == nil {
		return false
	}
	if !f.fCheckTx.Equals(fm.fCheckTx) {
		return false
	}
	if f.paramsUpdated != fm.paramsUpdated ||
		f.validatorsUpdated != fm.validatorsUpdated ||
		f.resetSlashing != fm.resetSlashing ||
		f.forceSeal != fm.forceSeal {
		return false
	}
	if !f.sortedFeederIDs.Equals(fm.sortedFeederIDs) {
		return false
	}
	if !f.cs.Equals(fm.cs) {
		return false
	}
	if len(f.rounds) != len(fm.rounds) {
		return false
	}
	// safe to range map, compare map
	for id, r := range f.rounds {
		if r2, ok := fm.rounds[id]; !ok {
			return false
		} else if !r.Equals(r2) {
			return false
		}
	}
	return true
}

// LatestRoundBaseBlock returns the base block of the latest round for a given feederID.
func (f *FeederManager) LatestRoundBaseBlock(feederID uint64) (uint64, bool) {
	// #nosec G115
	r, ok := f.rounds[int64(feederID)]
	if !ok {
		return 0, false
	}

	// #nosec G115
	return uint64(r.roundBaseBlock), true
}

func (f *FeederManager) GetNSTFeederIDFromClientChainID(clientChainID uint64) (uint64, bool) {
	return f.cs.GetNSTFeederIDFromClientChainID(clientChainID)
}

// getRecoveryStartPoint returns the height to start the recovery process
func getRecoveryStartPoint(currentHeight int64, recentParamsList []*oracletypes.RecentParams, prevRecentParams, latestRecentParams *oracletypes.RecentParams, validatorUpdateHeight int64) (height int64, replayRecentParamsList []*oracletypes.RecentParams) {
	if currentHeight > int64(latestRecentParams.Params.MaxNonce) {
		height = currentHeight - int64(latestRecentParams.Params.MaxNonce)
	}
	// there is no params updated in the recentParamsList, we can start from the validator update block if it's not too old(out of the distance of maxNonce from current height)
	if len(recentParamsList) == 0 {
		if height < validatorUpdateHeight {
			height = validatorUpdateHeight
		}
		// for empty recetParamsList, use latestrecentParams as the start point
		replayRecentParamsList = append(replayRecentParamsList, latestRecentParams)
		height++
		return height, replayRecentParamsList
	}

	if prevRecentParams.Block > 0 && prevRecentParams.Params.IsForceSealingUpdate(recentParamsList[0].Params) {
		// #nosec G115
		height = int64(recentParamsList[0].Block)
	}
	idx := 0
	for i := 1; i < len(recentParamsList); i++ {
		if recentParamsList[i-1].Params.IsForceSealingUpdate(recentParamsList[i].Params) {
			// #nosec G115
			height = int64(recentParamsList[i].Block)
			idx = i
		}
	}
	replayRecentParamsList = recentParamsList[idx:]

	if height < validatorUpdateHeight {
		height = validatorUpdateHeight
	}
	height++
	return height, replayRecentParamsList
}

// getProtoMsgItemFromQuote converts a MsgCreatePrice to a MsgItem for processing.
func getProtoMsgItemFromQuote(msg *oracletypes.MsgCreatePrice) *oracletypes.MsgItem {
	// address has been valid before
	validator, _ := oracletypes.ConsAddrStrFromCreator(msg.Creator)

	return &oracletypes.MsgItem{
		FeederID: msg.FeederID,
		// validator's consAddr
		Validator: validator,
		PSources:  msg.Prices,
	}
}
