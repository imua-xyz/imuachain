package feedermanagement

import (
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strconv"
	"strings"

	sdkerrors "cosmossdk.io/errors"
	"github.com/ExocoreNetwork/exocore/x/oracle/keeper/common"
	oracletypes "github.com/ExocoreNetwork/exocore/x/oracle/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func NewFeederManager(k common.KeeperOracle) *FeederManager {
	return &FeederManager{
		k:               k,
		sortedFeederIDs: make([]int64, 0),
		rounds:          make(map[int64]*round),
		cs:              nil,
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

// BeginBlock initializes the caches and slashing records, and setup the rounds
func (f *FeederManager) BeginBlock(ctx sdk.Context) (recovered bool) {
	// if the cache is nil and we are not in recovery mode, init the caches
	if f.cs == nil {
		var err error
		recovered, err = f.recovery(ctx)
		// it's safe to panic since this will only happen when the node is starting with something wrong in the store
		if err != nil {
			panic(err)
		}
		// init feederManager if failed to recovery, this should only happened on block_height==1
		if !recovered {
			f.initCaches(ctx)
			f.SetParamsUpdated()
			f.SetValidatorsUpdated()
		}
		f.initBehaviorRecords(ctx, ctx.BlockHeight())
		// in recovery mode, snapshot of feederManager is set in the beginblock instead of in the process of replaying endblockInrecovery
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
	// handleQuoteBehavior for ending quotes of rounds
	// commit state of mature rounds
	f.updateAndCommitRounds(ctx)

	// set status to open for rounds before their quoting window
	feederIDs := f.prepareRounds(ctx)
	// remove nonces for closing quoting-window and set nonces for opening quoting-window
	f.setupNonces(ctx, feederIDs)

	f.ResetFlags()

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
			logger.Debug("clear nonces for closing quoting window or forceSeal", "feederID", r.feederID, "roundID", r.roundID, "basedBlock", r.roundBaseBlock, "height", height, "forceSeal", f.forceSeal)
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
		logger.Debug("init nonces for new quoting window", "feederID", feederID, "roundID", r.roundID, "basedBlock", r.roundBaseBlock, "height", height)
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

// praepareRounds prepares the rounds for the next block, and returns the feederIDs of the rounds that are open on next block
func (f *FeederManager) prepareRounds(ctx sdk.Context) []int64 {
	logger := f.k.Logger(ctx)
	feederIDs := make([]int64, 0)
	height := ctx.BlockHeight()
	// it's safe to range map directly, this is just used to update memory state
	for _, r := range f.rounds {
		if open := r.PrepareForNextBlock(ctx.BlockHeight()); open {
			feederIDs = append(feederIDs, r.feederID)
			// logs might not be displayed in order, it's marked with [mem] to indicate that this is a memory state update
			logger.Info("[mem] open quoting window for round", "feederID", r.feederID, "roundID", r.roundID, "basedBlock", r.roundBaseBlock, "height", height)
		}
	}
	return feederIDs
}

// 1. update and commit Params if updated
// 2. update and commit validatorPowers if updated
// forceSeal: 1. params has some modifications related to quoting. 2.validatorSet changed
// resetSlashing: params has some modifications related to oracle_slashing
// func (f *FeederManager) updateAndCommitCaches(ctx sdk.Context) (forceSeal, resetSlashing bool, prevValidators, addedValidators []string) {
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
			f.SetResetSlasing()
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

func (f *FeederManager) commitRounds(ctx sdk.Context) {
	logger := f.k.Logger(ctx)
	height := ctx.BlockHeight()
	successFeederIDs := make([]string, 0)
	// it's safe to range map directly since the sate update is independent for each feederID, however we use sortedFeederIDs to keep the order of logs
	// this can be replaced by map iteration directly when better performance is needed
	for _, feederID := range f.sortedFeederIDs {
		r := f.rounds[feederID]
		if r.Committable() {
			finalPrice, ok := r.FinalPrice()
			if !ok {
				logger.Info("commit round with price from previous", "feederID", r.feederID, "roundID", r.roundID, "baseBlock", r.roundBaseBlock, "height", height)
				// #nosec G115  // tokenID is index of slice
				f.k.GrowRoundID(ctx, uint64(r.tokenID))
			} else {
				if f.cs.IsRuleV1(r.feederID) {
					priceCommit := finalPrice.ProtoPriceTimeRound(r.roundID, ctx.BlockTime().Format(oracletypes.TimeLayout))
					logger.Info("commit round with aggregated price", "feederID", r.feederID, "roundID", r.roundID, "baseBlock", r.roundBaseBlock, "price", priceCommit, "height", height)

					// #nosec G115  // tokenID is index of slice
					f.k.AppendPriceTR(ctx, uint64(r.tokenID), *priceCommit, finalPrice.DetID)
					// f.k.AppendPriceTR(ctx, uint64(r.tokenID), *priceCommit)

					fstr := strconv.FormatInt(feederID, 10)
					successFeederIDs = append(successFeederIDs, fstr) // there's no valid price for any round yet
				} else {
					logger.Error("We currently only support rules under oracle V1: only allow price from source Chainlink", "feederID", r.feederID)
				}
			}
			// keep aggregator for possible 'handlQuotingMisBehavior' at quotingWindowEnd
			r.status = roundStatusClosed
		}
		// close all quotingWindow to skip current rounds' 'handlQuotingMisBehavior'
		if f.forceSeal {
			r.closeQuotingWindow()
		}
	}
	if len(successFeederIDs) > 0 {
		feederIDsStr := strings.Join(successFeederIDs, "_")
		ctx.EventManager().EmitEvent(sdk.NewEvent(
			oracletypes.EventTypeCreatePrice,
			sdk.NewAttribute(oracletypes.AttributeKeyPriceUpdated, oracletypes.AttributeValuePriceUpdatedSuccess),
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

func (f *FeederManager) handleQuotingMisBehavior(ctx sdk.Context) {
	height := ctx.BlockHeight()
	logger := f.k.Logger(ctx)

	// it's safe to range map directly, each state update is independent for each feederID
	// state to be updated: {validatorReportInfo, validatorMissedRoundBitArray, signInfo, assets} of individual validator
	// we use sortedFeederIDs to keep the order of logs
	// this can be replaced by map iteration directly when better performance is needed
	for _, feederID := range f.sortedFeederIDs {
		r := f.rounds[feederID]
		if r.IsQuotingWindowEnd(height) {
			if _, found := r.FinalPrice(); !found {
				r.closeQuotingWindow()
				continue
			}
			validators := f.cs.GetValidators()
			for _, validator := range validators {
				reportedInfo, found := f.k.GetValidatorReportInfo(ctx, validator)
				if !found {
					logger.Error(fmt.Sprintf("Expected report info for validator %s but not found", validator))
					continue
				}
				miss, malicious := r.PerformanceReview(validator)
				if malicious {
					detID := r.getFinalDetIDForSourceID(oracletypes.SourceChainlinkID)
					finalPrice, _ := r.FinalPrice()
					logger.Info(
						"confirmed malicious price",
						"validator", validator,
						"infraction_height", height,
						"infraction_time", ctx.BlockTime(),
						"feederID", r.feederID,
						"detID", detID,
						"sourceID", oracletypes.SourceChainlinkID,
						"finalPrice", finalPrice,
					)
					consAddr, err := sdk.ConsAddressFromBech32(validator)
					if err != nil {
						f.k.Logger(ctx).Error("when do orale_performance_review, got invalid consAddr string. This should never happen", "validatorStr", validator)
						continue
					}

					operator := f.k.ValidatorByConsAddr(ctx, consAddr)
					if operator != nil && !operator.IsJailed() {
						power, _ := f.cs.GetPowerForValidator(validator)
						coinsBurned := f.k.SlashWithInfractionReason(ctx, consAddr, height, power.Int64(), f.k.GetSlashFractionMalicious(ctx), stakingtypes.Infraction_INFRACTION_UNSPECIFIED)
						ctx.EventManager().EmitEvent(
							sdk.NewEvent(
								oracletypes.EventTypeOracleSlash,
								sdk.NewAttribute(oracletypes.AttributeKeyValidatorKey, validator),
								sdk.NewAttribute(oracletypes.AttributeKeyPower, fmt.Sprintf("%d", power)),
								sdk.NewAttribute(oracletypes.AttributeKeyReason, oracletypes.AttributeValueMaliciousReportPrice),
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
					}
					continue
				}
				reportedRoundsWindow := f.k.GetReportedRoundsWindow(ctx)
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

				minReportedPerWindow := f.k.GetMinReportedPerWindow(ctx)

				if miss {
					ctx.EventManager().EmitEvent(
						sdk.NewEvent(
							oracletypes.EventTypeOracleLiveness,
							sdk.NewAttribute(oracletypes.AttributeKeyValidatorKey, validator),
							sdk.NewAttribute(oracletypes.AttributeKeyMissedRounds, fmt.Sprintf("%d", reportedInfo.MissedRoundsCounter)),
							sdk.NewAttribute(oracletypes.AttributeKeyHeight, fmt.Sprintf("%d", height)),
						),
					)

					logger.Info(
						"oracle_absent validator",
						"height", ctx.BlockHeight(),
						"validator", validator,
						"missed", reportedInfo.MissedRoundsCounter,
						"threshold", minReportedPerWindow,
					)
				}

				minHeight := reportedInfo.StartHeight + reportedRoundsWindow
				maxMissed := reportedRoundsWindow - minReportedPerWindow
				// if we are past the minimum height and the validator has missed too many rounds reporting prices, punish them
				if height > minHeight && reportedInfo.MissedRoundsCounter > maxMissed {
					consAddr, err := sdk.ConsAddressFromBech32(validator)
					if err != nil {
						f.k.Logger(ctx).Error("when do orale_performance_review, got invalid consAddr string. This should never happen", "validatorStr", validator)
						continue
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
							"height", height,
							"validator", consAddr.String(),
							"min_height", minHeight,
							"threshold", minReportedPerWindow,
							"jailed_until", jailUntil,
						)
					} else {
						// validator was (a) not found or (b) already jailed so we do not slash
						logger.Info(
							"validator would have been slashed for too many missed reporting price, but was either not found in store or already jailed",
							"validator", validator,
						)
					}
				}
				// Set the updated reportInfo
				f.k.SetValidatorReportInfo(ctx, validator, reportedInfo)
			}
			r.closeQuotingWindow()
		}
	}
}

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
				f.rounds[feederID] = newRound(feederID, tokenFeeder, int64(params.MaxNonce), f.cs, NewAggMedian())
			}
		}
		f.sortedFeederIDs.sort()
	}
}

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

func (f *FeederManager) updateAndCommitRoundsInRecovery(ctx sdk.Context) {
	f.setCommittableState(ctx)
	f.commitRoundsInRecovery()
	f.handleQuotingMisBehaviorInRecovery(ctx)
	f.updateRoundsParamsAndAddNewRounds(ctx)
	f.removeExpiredRounds(ctx)
}

func (f *FeederManager) updateAndCommitRounds(ctx sdk.Context) {
	f.setCommittableState(ctx)
	f.commitRounds(ctx)
	// behaviors review and close quotingWindow
	f.handleQuotingMisBehavior(ctx)
	f.updateRoundsParamsAndAddNewRounds(ctx)
	f.removeExpiredRounds(ctx)
}

func (f *FeederManager) ResetFlags() {
	f.paramsUpdated = false
	f.validatorsUpdated = false
	f.forceSeal = false
	f.resetSlashing = false
}

func (f *FeederManager) SetParamsUpdated() {
	f.paramsUpdated = true
}

func (f *FeederManager) SetValidatorsUpdated() {
	f.validatorsUpdated = true
}

func (f *FeederManager) SetResetSlasing() {
	f.resetSlashing = true
}

func (f *FeederManager) SetForceSeal() {
	f.forceSeal = true
}

func (f *FeederManager) ValidateMsg(msg *oracletypes.MsgCreatePrice) error {
	// nonce, feederID, creator has been verified by anteHandler
	// baseBlock is going to be verified by its corresponding round
	decimal, err := f.cs.GetDecimalFromFeederID(msg.FeederID)
	if err != nil {
		return err
	}
	for _, ps := range msg.Prices {
		// #nosec G115
		deterministic, err := f.cs.IsDeterministic(int64(ps.SourceID))
		if err != nil {
			return err
		}
		l := len(ps.Prices)
		if deterministic {
			if l == 0 {
				return fmt.Errorf("source:id_%d has no valid price, empty list", ps.SourceID)
			}
			if l > int(f.cs.GetMaxNonce()) {
				return fmt.Errorf("deterministic source:id_%d must provide no more than %d prices from different DetIDs, got:%d", ps.SourceID, f.cs.GetMaxNonce(), l)
			}
			seenDetIDs := make(map[string]struct{})
			for _, p := range ps.Prices {
				if _, ok := seenDetIDs[p.DetID]; ok {
					return errors.New("duplicated detIDs")
				}
				if len(p.Price) == 0 {
					return errors.New("price must not be empty")
				}
				if len(p.DetID) == 0 {
					return errors.New("detID of deteministic price must not be empty")
				}
				if p.Decimal != decimal {
					return fmt.Errorf("decimal not match for feederID:%d, expect:%d, got:%d", msg.FeederID, decimal, p.Decimal)
				}
				seenDetIDs[p.DetID] = struct{}{}
			}
		} else {
			// NOTE: v1 does not actually have this type of sources
			if l != 1 {
				return fmt.Errorf("non-deteministic sources should provide exactly one valid price, got:%d", len(ps.Prices))
			}
			p := ps.Prices[0]
			if len(p.Price) == 0 {
				return errors.New("price must not be empty")
			}
			if p.Decimal != decimal {
				return fmt.Errorf("decimal not match for feederID:%d, expect:%d, got:%d", msg.FeederID, decimal, p.Decimal)
			}
			if len(p.DetID) > 0 {
				return errors.New("price from non-deterministic should not have detID")
			}
			if len(p.Timestamp) == 0 {
				return errors.New("price from non-deterministic must have timestamp")
			}
		}
	}
	return nil
}

func (f *FeederManager) ProcessQuote(ctx sdk.Context, msg *oracletypes.MsgCreatePrice, isCheckTx bool) (*oracletypes.PriceTimeRound, error) {
	if isCheckTx {
		f = f.getCheckTx()
	}
	if err := f.ValidateMsg(msg); err != nil {
		return nil, oracletypes.ErrInvalidMsg.Wrap(err.Error())
	}
	msgItem := getProtoMsgItemFromQuote(msg)

	// #nosec G115  // feederID is index of slice
	r, ok := f.rounds[int64(msgItem.FeederID)]
	if !ok {
		// This should not happened since we do check the nonce in anthHandle
		return nil, fmt.Errorf("round not exists for feederID:%d, proposer:%s", msgItem.FeederID, msgItem.Validator)
	}

	// #nosec G115  // baseBlock is block height which is not negative
	if valid := r.ValidQuotingBaseBlock(int64(msg.BasedBlock)); !valid {
		return nil, fmt.Errorf("failed to process price-feed msg for feederID:%d, round is quoting:%t,quotingWindow is open:%t, expected baseBlock:%d, got baseBlock:%d", msgItem.FeederID, r.IsQuoting(), r.IsQuotingWindowOpen(), r.roundBaseBlock, msg.BasedBlock)
	}

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

	return &ret
}

func (f *FeederManager) updateCheckTx() {
	// flgas are taken care of
	// sortedFeederIDs will not be modified except in abci.EndBlock
	// successFeedereIDs will not be modifed except in abci.EndBlock
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
	f.fCheckTx = &ret
}

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

// initCaches initializes the caches of the FeederManager with keeper, params, validatorPowers
func (f *FeederManager) initCaches(ctx sdk.Context) {
	f.cs = newCaches()
	params := f.k.GetParams(ctx)
	validatorSet := f.k.GetAllExocoreValidators(ctx)
	validatorPowers := make(map[string]*big.Int)
	for _, v := range validatorSet {
		validatorPowers[sdk.ConsAddress(v.Address).String()] = big.NewInt(v.Power)
	}
	f.cs.Init(f.k, &params, validatorPowers)
}

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

	validatorSet := f.k.GetAllExocoreValidators(ctx)
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
		f.rounds[tfID] = newRound(tfID, tf, int64(params.MaxNonce), f.cs, NewAggMedian())
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

	return true, nil
}

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

// recoveryStartPoint returns the height to start the recovery process
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
