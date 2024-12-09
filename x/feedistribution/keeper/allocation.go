package keeper

import (
	"errors"
	"sort"

	"cosmossdk.io/math"
	avstypes "github.com/ExocoreNetwork/exocore/x/avs/types"
	"github.com/ExocoreNetwork/exocore/x/feedistribution/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// AllocateTokens performs reward and fee distribution to all validators.
//  1. afterSlash distributed accumlated fees till now in current epoch and the portion of minted coins
//     corresponding to the time passed in current epoch
//  2. afterEpoch distributes all left coins including both fees and minted coins
//
// CONTRACT: before we adopt f1 like mechnisam to deal with precisely distribution,
// we need to set the epochIdentify the same to both dogfood and exomint
func (k Keeper) AllocateTokens(ctx sdk.Context, isSlash bool) error {
	logger := k.Logger()
	feeCollector := k.authKeeper.GetModuleAccount(ctx, k.feeCollectorName)
	feesCollectedInt := k.bankKeeper.GetAllBalances(ctx, feeCollector.GetAddress())
	feesCollected := sdk.NewDecCoinsFromCoins(feesCollectedInt...)
	// if this is triggered by slash instead of epochEnd, we need to calculated the amount of minted coins
	// corresponding to passed time of current epoch
	if isSlash {
		mintParams := k.mintKeeper.GetParams(ctx)
		mintedCoin := sdk.NewCoin(
			mintParams.MintDenom, mintParams.EpochReward,
		)

		mintedCoinDec := sdk.NewDecCoinFromCoin(mintedCoin)
		// we only distribute fees (excluding minted coins) from current epoch
		// if the minted coins of current epoch had been allocated, we calculate the corresponding portion of minted tokens
		if feesCollectedInt.AmountOf(mintParams.MintDenom).GTE(mintParams.EpochReward) {
			epochInfo, found := k.epochsKeeper.GetEpochInfo(ctx, mintParams.EpochIdentifier)
			if !found {
				// skip the calculation and distribute no minted coins out, the remaining will be handled at the end of the epoch
				feesCollected.Sub(sdk.DecCoins{mintedCoinDec})
				logger.Error("Failed to find epoch info, no minted coins will be distributed")
				if feesCollected.Empty() {
					return errors.New("failed to AllocateTokens on slash event for minted coins calculation fail")
				}
			} else {
				passedDuration := math.LegacyNewDec(int64(ctx.BlockTime().Sub(epochInfo.StartTime)))
				epochDuration := math.LegacyNewDec(int64(epochInfo.Duration))
				mintedCoinDec.Amount.MulMut(math.LegacyOneDec().Sub(passedDuration.QuoTruncate(epochDuration)))
				feesCollected.Sub(sdk.DecCoins{mintedCoinDec})
			}
		}
	}

	// transfer collected fees including minted coins to the distribution module account
	if err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, k.feeCollectorName, types.ModuleName, feesCollectedInt); err != nil {
		return err
	}

	totalPreviousPower := k.StakingKeeper.GetLastTotalPower(ctx).Int64()
	feePool := k.GetFeePool(ctx)
	if totalPreviousPower == 0 {
		feePool.CommunityPool = feePool.CommunityPool.Add(feesCollected...)
		k.SetFeePool(ctx, feePool)
		return nil
	}
	logger.Info("Allocate tokens to all validators", "feesCollected amount is ", feesCollected, feesCollected.Empty())
	// calculate fraction allocated to exocore validators
	remaining := feesCollected
	communityTax, err := k.GetCommunityTax(ctx)
	if err != nil {
		return err
	}
	feeMultiplier := feesCollected.MulDecTruncate(math.LegacyOneDec().Sub(communityTax))

	// allocate tokens proportionally to voting power of different validators
	// TODO: Consider parallelizing later
	allValidators := k.StakingKeeper.GetAllExocoreValidators(ctx)
	for i, val := range allValidators {
		pk, err := val.ConsPubKey()
		if err != nil {
			logger.Error("Failed to deserialize public key; skipping", "error", err, "i", i)
			continue
		}
		validatorDetail, found := k.StakingKeeper.ValidatorByConsAddrForChainID(
			ctx, sdk.GetConsAddress(pk), avstypes.ChainIDWithoutRevision(ctx.ChainID()),
		)
		if !found {
			logger.Error("Operator address not found; skipping", "consAddress", sdk.GetConsAddress(pk), "i", i)
			continue
		}
		if totalPreviousPower == 0 {
			return nil
		}
		powerFraction := math.LegacyNewDec(val.Power).QuoTruncate(math.LegacyNewDec(totalPreviousPower))
		reward := feeMultiplier.MulDecTruncate(powerFraction)

		k.AllocateTokensToValidator(ctx, validatorDetail, reward, feePool)
		remaining = remaining.Sub(reward)
	}

	// allocate community funding
	feePool.CommunityPool = feePool.CommunityPool.Add(remaining...)
	k.SetFeePool(ctx, feePool)

	return nil
}

// AllocateTokensToValidator allocate tokens to a particular validator,
// splitting according to commission.
func (k Keeper) AllocateTokensToValidator(ctx sdk.Context, val stakingtypes.ValidatorI, tokens sdk.DecCoins, feePool *types.FeePool) {
	logger := k.Logger()
	valBz := val.GetOperator()
	accAddr := sdk.AccAddress(valBz)
	ops, err := k.StakingKeeper.OperatorInfo(ctx, accAddr.String())
	if err != nil {
		ctx.Logger().Error("Failed to get operator info", "error", err)
	}
	commission := tokens.MulDec(ops.GetCommission().Rate)
	shared := tokens.Sub(commission)
	// update current commission
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeCommission,
		sdk.NewAttribute(sdk.AttributeKeyAmount, commission.String()),
		sdk.NewAttribute(types.AttributeKeyValidator, val.GetOperator().String()),
	))
	currentCommission := k.GetValidatorAccumulatedCommission(ctx, valBz)
	currentCommission.Commission = currentCommission.Commission.Add(commission...)
	k.SetValidatorAccumulatedCommission(ctx, valBz, currentCommission)
	// update current rewards, i.e. the rewards to stakers
	// if the rewards do not exist it's fine, we will just add to zero.
	// allocate share tokens to all stakers of this operator.
	operatorAccAddress := sdk.AccAddress(valBz)
	k.AllocateTokensToStakers(ctx, operatorAccAddress, shared, feePool)

	// update outstanding rewards
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeRewards,
		sdk.NewAttribute(sdk.AttributeKeyAmount, commission.String()),
		sdk.NewAttribute(types.AttributeKeyValidator, val.GetOperator().String()),
	))

	// ValidatorOutstandingRewards is the rewards of a validator address.
	outstanding := k.GetValidatorOutstandingRewards(ctx, valBz)
	outstanding.Rewards = outstanding.Rewards.Add(tokens...)
	k.SetValidatorOutstandingRewards(ctx, valBz, outstanding)
	logger.Info("Allocate tokens to validator successfully", "allocated amount is", tokens, "accumulated allocated amount is", outstanding.Rewards.String())
}

func (k Keeper) AllocateTokensToStakers(ctx sdk.Context, operatorAddress sdk.AccAddress, rewardToAllStakers sdk.DecCoins, feePool *types.FeePool) {
	logger := k.Logger()
	logger.Info("AllocateTokensToStakers", "operatorAddress", operatorAddress.String())
	stakersPowerMap, curTotalStakersPowers := make(map[string]math.LegacyDec), math.LegacyNewDec(0)
	globalStakerAddressList := make([]string, 0)

	chainIDWithoutRevision := avstypes.ChainIDWithoutRevision(ctx.ChainID())
	isAVS, avsAddress := k.avsKeeper.IsAVSByChainID(ctx, chainIDWithoutRevision)
	if !isAVS {
		logger.Error("Skipping distribution for due to fail to generate avsAddr from chainID", "chainID", ctx.ChainID())
		return
	}

	assetIDs := k.StakingKeeper.GetAssetIDs(ctx)
	for _, assetID := range assetIDs {
		stakerList, err := k.StakingKeeper.GetStakersByOperator(ctx, operatorAddress.String(), assetID)
		if err != nil {
			logger.Debug("staker lists not found; skipping")
			continue
		}
		for _, staker := range stakerList.Stakers {
			if curStakerPower, err := k.StakingKeeper.CalculateUSDValueForStaker(ctx, staker, avsAddress, operatorAddress.Bytes()); err != nil {
				logger.Error("curStakerPower error", "error", err)
			} else {
				stakersPowerMap[staker] = curStakerPower
				globalStakerAddressList = append(globalStakerAddressList, staker)
				curTotalStakersPowers = curTotalStakersPowers.Add(curStakerPower)
			}
		}
	}
	sort.Slice(globalStakerAddressList, func(i, j int) bool {
		return stakersPowerMap[globalStakerAddressList[i]].GT(stakersPowerMap[globalStakerAddressList[j]])
	})
	remaining := rewardToAllStakers
	// allocate to stakers in voting power descending order if the curTotalStakersPower is positive
	if curTotalStakersPowers.IsPositive() {
		for _, staker := range globalStakerAddressList {
			stakerPower := stakersPowerMap[staker]
			powerFraction := stakerPower.QuoTruncate(curTotalStakersPowers)
			rewardToSingleStaker := rewardToAllStakers.MulDecTruncate(powerFraction)
			k.AllocateTokensToSingleStaker(ctx, staker, rewardToSingleStaker)
			remaining = remaining.Sub(rewardToSingleStaker)
		}
	}
	feePool.CommunityPool = feePool.CommunityPool.Add(rewardToAllStakers...)
	logger.Info("allocate tokens to stakers successfully", "allocated amount is", rewardToAllStakers.String())
}

func (k Keeper) AllocateTokensToSingleStaker(ctx sdk.Context, stakerAddress string, reward sdk.DecCoins) {
	logger := k.Logger()
	currentStakerRewards := k.GetStakerRewards(ctx, stakerAddress)
	currentStakerRewards.Rewards = currentStakerRewards.Rewards.Add(reward...)
	k.SetStakerRewards(ctx, stakerAddress, currentStakerRewards)
	logger.Info("allocate tokens to single staker successfully", "allocated amount is", reward, "accumulated allocated amount is", currentStakerRewards.Rewards.String())
}
