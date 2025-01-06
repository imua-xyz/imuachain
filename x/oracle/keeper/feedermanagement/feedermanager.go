package feedermanagement

import (
	"fmt"
	"math/big"

	//	oraclekeeper "github.com/ExocoreNetwork/exocore/x/oracle/keeper"
	"github.com/ExocoreNetwork/exocore/x/oracle/keeper/common"
	oracletypes "github.com/ExocoreNetwork/exocore/x/oracle/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func NewFeederManager(k common.KeeperOracle) *FeederManager {
	return &FeederManager{
		k:                k,
		rounds:           make(map[int64]*round),
		cs:               nil,
		successFeederIDs: make([]int64, 0),
	}
}

func (f *FeederManager) SetKeeper(k common.KeeperOracle) {
	f.k = k
}

// BeginBlock initializes the caches and slashing records, and setup the rounds
// if (isCheckTx||isSimulate||isRecover) is true, then commitState is set to be true
func (f *FeederManager) BeginBlock(ctx sdk.Context) {
	// if the cache is nil and we are not in recovery mode, init the caches
	if f.cs == nil {
		if !f.recovery(ctx) {
			f.initCaches(ctx)
		}
		f.initBehaviorRecords(ctx)
	}
}

func (f *FeederManager) EndBlock(ctx sdk.Context) {
	// update params and validator set if necessary in caches and commit all updated information
	addedValidators := f.updateAndCommitCaches(ctx)

	// update Slashing related records (reportInfo, missCountBitArray), handle case for 1. reseetSlashing, 2. new validators added for validatorset change
	f.updateBehaviorRecords(ctx, addedValidators)

	// update rounds including create new rounds based on params change, remove expired rounds
	// handleQuoteBehavior for ending quotes of rounds
	// commit state of mature rounds
	f.updateAndCommitRounds(ctx)

	// set status to open for rounds before their quoting window
	feederIDs := f.prepareRounds(ctx)
	// remove nocnes for closing quoting-window and set nonces for opening quoting-window
	f.setupNonces(ctx, feederIDs)

	f.ResetFlags()
}

func (f *FeederManager) EndBlockInRecovery(ctx sdk.Context, params *oracletypes.Params) {
	if params != nil {
		f.SetParamsUpdated()
		f.cs.AddCache(params)
	}
	f.updateAndCommitRoundsInRecovery(ctx)
	f.prepareRounds(ctx)
	f.ResetFlags()
}

func (f *FeederManager) setupNonces(ctx sdk.Context, feederIDs []int64) {
	// remove nonces for closed quoting windows
	height := ctx.BlockHeight()
	if f.forceSeal {
		for _, r := range f.rounds {
			f.k.RemoveNonceWithFeederIDForAll(ctx, uint64(r.feederID))
		}
	} else {
		for _, r := range f.rounds {
			if r.IsQuotingWindowEnd(height) {
				f.k.RemoveNonceWithFeederIDForAll(ctx, uint64(r.feederID))
			}
		}
	}
	// setup nonces for opening quoting windows
	if len(feederIDs) == 0 {
		return
	}
	validators := f.cs.GetValidators()
	for _, feederID := range feederIDs {
		f.k.AddZeroNonceItemWithFeederIDForValidators(ctx, uint64(feederID), validators)
	}
}

func (f *FeederManager) initBehaviorRecords(ctx sdk.Context) {
	validators := f.cs.GetValidators()
	for _, validator := range validators {
		f.k.InitValidatorReportInfo(ctx, validator, ctx.BlockHeight())
	}
}

func (f *FeederManager) updateBehaviorRecords(ctx sdk.Context, addedValidators []string) {
	height := ctx.BlockHeight()
	if f.resetSlashing {
		// reset all validators' reportInfo
		f.k.ClearAllValidatorReportInfo(ctx)
		f.k.ClearAllValidatorMissedRoundBitArray(ctx)
		validators := f.cs.GetValidators()
		for _, validator := range validators {
			f.k.InitValidatorReportInfo(ctx, validator, height)
		}
	} else if f.validatorsUpdated {
		for _, validator := range addedValidators {
			// add possible new added validator info for slashing tracking
			f.k.InitValidatorReportInfo(ctx, validator, height)
		}
	}
}

func (f *FeederManager) prepareRounds(ctx sdk.Context) []int64 {
	feederIDs := make([]int64, 0)
	for _, r := range f.rounds {
		if open := r.PrepareForNextBlock(ctx.BlockHeight()); open {
			feederIDs = append(feederIDs, r.feederID)
		}
	}
	return feederIDs
}

// 1. update and commit Params if updated
// 2. update and commit validatorPowers if updated
// forceSeal: 1. params has some modifications related to quoting. 2.validatorSet changed
// resetSlashing: params has some modifications related to oracle_slashing
// func (f *FeederManager) updateAndCommitCaches(ctx sdk.Context) (forceSeal, resetSlashing bool, prevValidators, addedValidators []string) {
func (f *FeederManager) updateAndCommitCaches(ctx sdk.Context) (addedValidators []string) {
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
		f.cs.AddCache(&params)
	}

	// update validators
	validatorUpdates := f.k.GetValidatorUpdates(ctx)
	if len(validatorUpdates) > 0 {
		f.SetValidatorsUpdated()
		f.SetForceSeal()
		addedValidators = make([]string, 0)
		validatorMap := make(map[string]*big.Int)
		for _, vu := range validatorUpdates {
			pubKey, _ := cryptocodec.FromTmProtoPublicKey(vu.PubKey)
			validatorStr := sdk.ConsAddress(pubKey.Address()).String()
			validatorMap[validatorStr] = big.NewInt(vu.Power)
			if vu.Power > 0 {
				addedValidators = append(addedValidators, validatorStr)
			}
		}
		// update validator set information in cache
		f.cs.AddCache(ItemV(validatorMap))
	}

	// commit caches: msgs is exists, params if updated, validatorPowers is updated
	f.cs.Commit(ctx, false)
	return addedValidators
}

func (f *FeederManager) commitRoundsInRecovery() {
	for _, r := range f.rounds {
		if r.Committable() {
			r.FinalPrice()
			r.status = roundStatusClosed
		}
		// close all quotingWindow to skip current rounds' 'handlQuotingMisBehavior'
		if f.forceSeal {
			r.closeQuotingWindow()
		}
	}
}

func (f *FeederManager) commitRounds(ctx sdk.Context) {
	for _, r := range f.rounds {
		if r.Committable() {
			finalPrice, ok := r.FinalPrice()
			if !ok {
				f.k.GrowRoundID(ctx, uint64(r.tokenID))
			} else {
				if f.cs.IsRuleV1(r.feederID) {
					priceCommit := finalPrice.PriceTimeRound(r.roundID, ctx.BlockTime().Format(oracletypes.TimeLayout))
					f.k.AppendPriceTR(ctx, uint64(r.tokenID), *priceCommit)
				} else {
					f.logger.Error("We currently only support rules under oracle V1: only allow price from source Chainlink", "feederID", r.feederID)
				}
			}
			// keep aggregator for possible 'handlQuotingMisBehavior' at quotingWindowEnd
			r.status = roundStatusClosed
		}
		// close all quotingWindow to skip current rounds' 'handlQuotingMisBehavior'
		if f.forceSeal {
			r.closeQuotingWindow()
			//			f.k.RemoveNonceWithFeederIDForAll(ctx, uint64(r.feederID))
		}
	}
}

func (f *FeederManager) handleQuotingMisBehaviorInRecovery(ctx sdk.Context) {
	height := ctx.BlockHeight()
	for _, r := range f.rounds {
		if r.IsQuotingWindowEnd(height) && r.a != nil {
			// TODO: slashing&jailing
			validators := f.cs.GetValidators()
			for _, validator := range validators {
				_, found := f.k.GetValidatorReportInfo(ctx, validator)
				if !found {
					f.logger.Error(fmt.Sprintf("Expected report info for validator %s but not found", validator))
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
	for _, r := range f.rounds {
		if r.IsQuotingWindowEnd(height) && r.a != nil {
			// TODO: slashing&jailing
			validators := f.cs.GetValidators()
			for _, validator := range validators {
				reportedInfo, found := f.k.GetValidatorReportInfo(ctx, validator)
				if !found {
					f.logger.Error(fmt.Sprintf("Expected report info for validator %s but not found", validator))
					continue
				}
				miss, malicious := r.PerformanceReview(validator)
				if malicious {
					detID := r.getFinalDetIDForSourceID(oracletypes.SourceChainlinkID)
					finalPrice, _ := r.FinalPrice()
					// TODO: malicious price, just slash&jail immediately
					f.logger.Info(
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
						panic("invalid consAddr string")
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

					f.logger.Debug(
						"absent validator",
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
						panic("invalid consAddr string")
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

						f.logger.Info(
							"jailing validator due to oracle_liveness fault",
							"height", height,
							"validator", consAddr.String(),
							"min_height", minHeight,
							"threshold", minReportedPerWindow,
							"jailed_until", jailUntil,
						)
					} else {
						// validator was (a) not found or (b) already jailed so we do not slash
						f.logger.Info(
							"validator would have been slashed for too many missed repoerting price, but was either not found in store or already jailed",
							"validator", validator,
						)
					}
				}
				// Set the updated reportInfo
				f.k.SetValidatorReportInfo(ctx, validator, reportedInfo)
			}
			r.closeQuotingWindow()
			//	f.k.RemoveNonceWithFeederIDForAll(ctx, uint64(r.feederID))
		}
	}
}

func (f *FeederManager) setCommittableState(ctx sdk.Context) {
	if f.forceSeal {
		for _, r := range f.rounds {
			if r.status == roundStatusOpen {
				r.status = roundStatusCommittable
			}
		}
	} else {
		height := ctx.BlockHeight()
		for _, r := range f.rounds {
			if r.IsQuotingWindowEnd(height) && r.status == roundStatusOpen {
				r.status = roundStatusCommittable
			}
		}
	}
}

func (f *FeederManager) updateRoundsParamsAndAddNewRounds(ctx sdk.Context) {
	height := ctx.BlockHeight()
	if f.paramsUpdated {
		params := &oracletypes.Params{}
		f.cs.Read(params)
		existsFeederIDs := make(map[int64]struct{})
		//	expiredFeederIDs := make([]int64, 0)
		for _, r := range f.rounds {
			r.UpdateParams(params.TokenFeeders[r.feederID], int64(params.MaxNonce))
			existsFeederIDs[r.feederID] = struct{}{}
		}
		// add new rounds
		for feederID, tokenFeeder := range params.TokenFeeders {
			feederID := int64(feederID)
			if _, ok := existsFeederIDs[feederID]; !ok && (tokenFeeder.EndBlock == 0 || tokenFeeder.EndBlock > uint64(height)) {
				f.rounds[feederID] = newRound(feederID, tokenFeeder, int64(params.MaxNonce), f.cs)
			}
		}
	}
}

func (f *FeederManager) removeExpiredRounds(ctx sdk.Context) {
	height := ctx.BlockHeight()
	expiredFeederIDs := make([]int64, 0)
	for _, r := range f.rounds {
		if r.endBlock <= height {
			expiredFeederIDs = append(expiredFeederIDs, r.feederID)
		}
	}
	for _, feederID := range expiredFeederIDs {
		if r := f.rounds[feederID]; r.status != roundStatusClosed {
			r.closeQuotingWindow()
			f.k.RemoveNonceWithFeederIDForAll(ctx, uint64(r.feederID))
		}
		delete(f.rounds, feederID)
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

func (f *FeederManager) ProcessQuote(msg *oracletypes.MsgCreatePrice, isCheckTx bool) (*PriceResult, error) {
	if isCheckTx {
		f = f.coppyForSimulation()
	}
	msgItem := f.getMsgItemFromQuote(msg)
	r, ok := f.rounds[int64(msgItem.FeederID)]
	if !ok {
		// This should not happend since we do check the nonce in anthHandle
		f.logger.Error("round not exists", "msgItem", msgItem)
		return nil, fmt.Errorf("round not exists for feederID:%d, porposer:%s", msgItem.FeederID, msgItem.Validator)
	}

	if valid := r.ValidQuotingBaseBlock(int64(msg.BasedBlock)); !valid {
		return nil, fmt.Errorf("failed to execute price-feed msg for feederID:%d, round is quoting:%t,quotingWindow is open:%t, expected baseBlock:%d, got baseBlock:%d", msgItem.FeederID, r.IsQuoting(), r.IsQuotingWindowOpen(), r.roundBaseBlock, msg.BasedBlock)
	}

	return r.Tally(msgItem)
}

func (f *FeederManager) coppyForSimulation() *FeederManager {
	return nil
}

func (f *FeederManager) ProcessQuoteInRecovery(msgItems []*oracletypes.MsgItem) {
	for _, msgItem := range msgItems {
		r, ok := f.rounds[int64(msgItem.FeederID)]
		if !ok {
			continue
		}
		r.Tally(msgItem)
	}
}

func (f *FeederManager) ProcessQuoteInSimulation(ctx sdk.Context, msg *oracletypes.MsgCreatePrice) {}

func (f *FeederManager) getMsgItemFromQuote(msg *oracletypes.MsgCreatePrice) *oracletypes.MsgItem {
	// address has been valid before
	accAddress, _ := sdk.AccAddressFromBech32(msg.Creator)
	validator := sdk.ConsAddress(accAddress).String()

	return &oracletypes.MsgItem{
		FeederID: msg.FeederID,
		// validator's consAddr
		Validator: validator,
		PSources:  msg.Prices,
	}
}

func (f *FeederManager) closeRounds(ctx sdk.Context, forceSeal bool) {}

func (f *FeederManager) commitCaches(ctx sdk.Context) {}

// initCaches initializes the caches of the FeederManager with keeper, params, validatorPowers
func (f *FeederManager) initCaches(ctx sdk.Context) {
	f.cs = newCaches()
	params := f.k.GetParams(ctx)
	validatorSet := f.k.GetAllExocoreValidators(ctx)
	validatorPowers := make(map[string]*big.Int)
	for _, v := range validatorSet {
		validatorPowers[sdk.ConsAddress(v.Address).String()] = big.NewInt(v.Power)
	}
	f.cs.Init(ctx, f.k, &params, validatorPowers)
}

func (f *FeederManager) recovery(ctx sdk.Context) bool {
	height := ctx.BlockHeight()
	recentParamsList, latestRecentParams, prevRecentParams := f.k.GetRecentParamsWithinMaxNonce(ctx)
	if latestRecentParams.Block == 0 {
		return false
	}
	validatorUpdateBlock, found := f.k.GetValidatorUpdateBlock(ctx)
	if !found {
		// on recovery mode, the validator update block must be found, otherwise we just panic to stop the node start
		// it's safe to panic since this will only happen when the node is starting with something wrong in the store
		panic("validator update block not found in recovery mode for feeder manager")
	}
	startHeight, replayRecentParamsList := getRecoveryStartPoint(height, recentParamsList, &prevRecentParams, &latestRecentParams, int64(validatorUpdateBlock.Block))

	f.cs = newCaches()
	params := replayRecentParamsList[0].Params
	replayRecentParamsList = replayRecentParamsList[1:]

	validatorSet := f.k.GetAllExocoreValidators(ctx)
	validatorPowers := make(map[string]*big.Int)
	for _, v := range validatorSet {
		validatorPowers[sdk.ConsAddress(v.Address).String()] = big.NewInt(v.Power)
	}

	f.cs.Init(ctx, f.k, params, validatorPowers)

	replayHeight := startHeight - 1

	ctxReplay := ctx.WithBlockHeight(replayHeight)
	for tfID, tf := range params.TokenFeeders {
		// safe conversion
		if tf.EndBlock > 0 && int64(tf.EndBlock) <= replayHeight {
			continue
		}
		tfID := int64(tfID)
		f.rounds[tfID] = newRound(tfID, tf, int64(params.MaxNonce), f.cs)
	}
	f.prepareRounds(ctxReplay)

	recentMsgs := f.k.GetAllRecentMsg(ctxReplay)

	for ; startHeight < height; startHeight++ {
		ctxReplay = ctxReplay.WithBlockHeight(startHeight)
		// only execute msgItems corresponding to rounds opened on or after replayHeight, since any rounds opened before replay height must be closed on or before height-1
		// which means no memory state need to be updated for thoes rounds
		// and we don't need to take care of 'close quoting-window' since the size of replay window t most equals to maxNonce
		i := 0
		for idx, recentMsg := range recentMsgs {
			i = idx
			if int64(recentMsg.Block) == startHeight {
				f.ProcessQuoteInRecovery(ctxReplay, recentMsg.Msgs)
				break
			}
		}
		recentMsgs = recentMsgs[i+1:]
		// var params *oracletypes.Params
		if len(replayRecentParamsList) > 0 && int64(replayRecentParamsList[0].Block) == startHeight {
			params = replayRecentParamsList[0].Params
			replayRecentParamsList = replayRecentParamsList[1:]
		}
		f.EndBlockInRecovery(ctxReplay, params)
	}

	return true
}

// recoveryStartPoint returns the height to start the recovery process
func getRecoveryStartPoint(currentHeight int64, recentParamsList []*oracletypes.RecentParams, prevRecentParams, latestRecentParams *oracletypes.RecentParams, validatorUpdateHeight int64) (height int64, replayRecentParamsList []*oracletypes.RecentParams) {
	height = currentHeight - int64(latestRecentParams.Params.MaxNonce)
	// there is no params updated in the recentParamsList, we can start from the validator update block if it's not too old(out of the distance of maxNonce from current height)
	if len(recentParamsList) == 0 {
		if height < validatorUpdateHeight {
			height = validatorUpdateHeight
		}
		// for empty recetParamsList, use latestrecentParams as the start point
		replayRecentParamsList = append(replayRecentParamsList, latestRecentParams)
		height++
		return
	}

	if prevRecentParams.Block > 0 && prevRecentParams.Params.IsForceSealingUpdate(recentParamsList[0].Params) {
		height = int64(recentParamsList[0].Block)
	}
	idx := 0
	for i := 1; i < len(recentParamsList); i++ {
		if recentParamsList[i-1].Params.IsForceSealingUpdate(recentParamsList[i].Params) {
			height = int64(recentParamsList[i].Block)
			idx = i
		}
	}
	replayRecentParamsList = recentParamsList[idx:]

	if height < validatorUpdateHeight {
		height = validatorUpdateHeight
	}
	height++
	return
}
