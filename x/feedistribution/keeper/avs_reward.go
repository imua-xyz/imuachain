package keeper

import (
	"strconv"

	"github.com/imua-xyz/imuachain/utils"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"

	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/imua-xyz/imuachain/types/keys"
	feedistributiontypes "github.com/imua-xyz/imuachain/x/feedistribution/types"
	operatortypes "github.com/imua-xyz/imuachain/x/operator/types"
)

type (
	// AVSEpochRewardFn is a function that retrieves the AVS reward for the current epoch.
	// There are three possible implementation methods:
	// 1. CustomizedEpochRewardFnForAVSs: Retrieves the reward information through the function
	//    `GetAVSRewardDistribution`.  This method is used when the AVS customizes reward inflation via a precompile
	//	   contract.
	// 2. EpochRewardFnForDogfood: Used for the dogfood, where reward inflation is determined by the mint module.
	// 3. DefaultEpochRewardFnForAVSs: The default function for reward inflation.
	//    The AVS can configure parameters to adjust reward inflation, but this method provides less flexibility than
	//    the first one.
	AVSEpochRewardFn func(ctx sdk.Context, avsAddr string) (sdk.DecCoins, error)
	// OperatorRewardProportionsFn is a function that retrieves the reward proportions of multiple operators for the
	// current epoch. There are three possible implementation methods, similar to the `AVSEpochRewardFn`.
	OperatorRewardProportionsFn func(ctx sdk.Context, avsAddr string) ([]feedistributiontypes.OperatorRewardProportion, error)
)

// SetAVSRewardDistribution : This function can be called by the reward inflation and allocation mechanisms of AVSs.
// Since different AVSs may have distinct reward models, they can customize this logic through an AVS reward contract.
// In such cases, we need to provide a precompiled interface for this function to facilitate reward contract
// development. Additionally, AVSs might require a keeper to periodically call the customized reward contract and set
// the reward distribution information. This process can be managed by an external service, such as the Chainlink keeper.
// In this case, the reward contract only needs to periodically update the corresponding parameters through the
// precompiled interface. All reward distributions, including distributions to operators and stakers,
// will be automatically executed on the Imua chain through the F1 distribution mechanism.
// Alternatively, we may provide a default inflation and allocation mechanism within the native modules of the Imua
// chain, similar to the `DefaultMintFn` in Cosmos SDK. In this case, AVSs only need to configure the inflation and
// allocation parameters, and no keeper is required. The Imua chain will automatically execute the logic based on the
// parameters. However, this approach lacks flexibility for customized requirements.
// AVSs can choose between these two methods based on their specific needs.
func (k Keeper) SetAVSRewardDistribution(ctx sdk.Context, avsAddr string, distribution feedistributiontypes.AVSRewardDistribution) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), feedistributiontypes.KeyPrefixAVSRewardDistribution)
	// check if the reward asset has been registered by the AVS
	for _, rewardCoin := range distribution.Rewards {
		if !k.IsAVSRewardAssetBySymbol(ctx, avsAddr, rewardCoin.Denom) {
			return feedistributiontypes.ErrAVSRewardAssetNotFound.Wrapf("the reward coin isn't registered, AvsAddr:%s denomination:%s", avsAddr, rewardCoin.Denom)
		}
	}
	// Check if the operator has opted into the AVS or just opted out
	// of it before the end of the current epoch.
	for _, operatorProportion := range distribution.OperatorRewardProportions {
		// We don't check if the operator is jailed here because there might
		// still be partial rewards for jailed operators.
		if k.operatorKeeper.IsOptedOutAndEffective(ctx, operatorProportion.OperatorAddr, avsAddr) {
			return feedistributiontypes.ErrInvalidRewardDistribution.Wrapf("invalid operator for reward distribution, operator:%s", operatorProportion.OperatorAddr)
		}
		// check if the operator has active USD value
		optedUSDValue, err := k.operatorKeeper.GetOperatorOptedUSDValue(ctx, avsAddr, operatorProportion.OperatorAddr)
		if err != nil {
			return err
		}
		if !optedUSDValue.ActiveUSDValue.IsPositive() {
			return feedistributiontypes.ErrInvalidRewardDistribution.Wrapf("invalid operator for reward distribution, operator:%s,ActiveUSDValue:%s", operatorProportion.OperatorAddr, optedUSDValue.ActiveUSDValue)
		}
	}

	epochInfo, err := k.avsKeeper.GetAVSEpochInfo(ctx, avsAddr)
	if err != nil {
		return err
	}
	distribution.RewardsEpochNumber = epochInfo.CurrentEpoch
	distribution.ProportionsEpochNumber = epochInfo.CurrentEpoch
	bz := k.cdc.MustMarshal(&distribution)
	store.Set(common.HexToAddress(avsAddr).Bytes(), bz)

	// emit event for indexers
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			feedistributiontypes.EventTypeAVSRewardDistributionSet,
			sdk.NewAttribute(feedistributiontypes.AttributeKeyAvsAddress, avsAddr),
			sdk.NewAttribute(feedistributiontypes.AttributeKeyEpochIdentifier, epochInfo.Identifier),
			sdk.NewAttribute(feedistributiontypes.AttributeKeyEpochNumber, strconv.FormatInt(epochInfo.CurrentEpoch, 10)),
			sdk.NewAttribute(feedistributiontypes.AttributeKeyEpochRewards, distribution.Rewards.String()),
			sdk.NewAttribute(
				feedistributiontypes.AttributeKeyOperatorProportions,
				feedistributiontypes.OperatorRewardProportions(distribution.OperatorRewardProportions).String()),
		),
	)
	return nil
}

// SetAVSEpochRewardExclusive sets the epoch rewards exclusively for an AVS.
// It is also provided to the AVS through a precompile contract.
// This interface allows the AVS to customize the reward inflation logic per epoch,
// providing greater flexibility for the AVS.
// Setting null rewards is allowed, enabling the AVS to disable reward distribution.
func (k Keeper) SetAVSEpochRewardExclusive(ctx sdk.Context, avsAddr string, rewards sdk.DecCoins) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), feedistributiontypes.KeyPrefixAVSRewardDistribution)
	// check if the reward asset has been registered by the AVS
	for _, rewardCoin := range rewards {
		if !k.IsAVSRewardAssetBySymbol(ctx, avsAddr, rewardCoin.Denom) {
			return feedistributiontypes.ErrAVSRewardAssetNotFound.Wrapf("the reward coin isn't registered, AvsAddr:%s denomination:%s", avsAddr, rewardCoin.Denom)
		}
	}
	rewardDistribution := feedistributiontypes.AVSRewardDistribution{}
	key := common.HexToAddress(avsAddr).Bytes()
	value := store.Get(key)
	if value != nil {
		k.cdc.MustUnmarshal(value, &rewardDistribution)
	}

	epochInfo, err := k.avsKeeper.GetAVSEpochInfo(ctx, avsAddr)
	if err != nil {
		return err
	}
	rewardDistribution.RewardsEpochNumber = epochInfo.CurrentEpoch
	rewardDistribution.Rewards = rewards
	bz := k.cdc.MustMarshal(&rewardDistribution)
	store.Set(key, bz)
	// emit event for indexers
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			feedistributiontypes.EventTypeAVSEpochRewardSet,
			sdk.NewAttribute(feedistributiontypes.AttributeKeyAvsAddress, avsAddr),
			sdk.NewAttribute(feedistributiontypes.AttributeKeyEpochIdentifier, epochInfo.Identifier),
			sdk.NewAttribute(feedistributiontypes.AttributeKeyEpochNumber, strconv.FormatInt(epochInfo.CurrentEpoch, 10)),
			sdk.NewAttribute(feedistributiontypes.AttributeKeyEpochRewards, rewards.String()),
		),
	)
	return nil
}

// SetAVSRewardProportionsExclusive sets the operator reward proportions exclusively for an AVS.
// It is also provided to the AVS through a precompile contract.
// This interface allows the AVS to customize the reward proportion of each operator per epoch,
// providing greater flexibility for the AVS.
func (k Keeper) SetAVSRewardProportionsExclusive(ctx sdk.Context, avsAddr string, rewardProportions []feedistributiontypes.OperatorRewardProportion) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), feedistributiontypes.KeyPrefixAVSRewardDistribution)
	// Check if the operator has opted into the AVS or just opted out
	// of it before the end of the current epoch.
	for _, operatorProportion := range rewardProportions {
		// We don't check if the operator is jailed here because there might
		// still be partial rewards for jailed operators.
		if k.operatorKeeper.IsOptedOutAndEffective(ctx, operatorProportion.OperatorAddr, avsAddr) {
			return feedistributiontypes.ErrInvalidRewardDistribution.Wrapf("invalid operator for reward distribution, operator:%s", operatorProportion.OperatorAddr)
		}
	}
	rewardDistribution := feedistributiontypes.AVSRewardDistribution{}
	key := common.HexToAddress(avsAddr).Bytes()
	value := store.Get(key)
	if value != nil {
		k.cdc.MustUnmarshal(value, &rewardDistribution)
	}

	epochInfo, err := k.avsKeeper.GetAVSEpochInfo(ctx, avsAddr)
	if err != nil {
		return err
	}
	rewardDistribution.ProportionsEpochNumber = epochInfo.CurrentEpoch
	rewardDistribution.OperatorRewardProportions = rewardProportions
	bz := k.cdc.MustMarshal(&rewardDistribution)
	store.Set(key, bz)
	// emit event for indexers
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			feedistributiontypes.EventTypeAVSRewardProportionsSet,
			sdk.NewAttribute(feedistributiontypes.AttributeKeyAvsAddress, avsAddr),
			sdk.NewAttribute(feedistributiontypes.AttributeKeyEpochIdentifier, epochInfo.Identifier),
			sdk.NewAttribute(feedistributiontypes.AttributeKeyEpochNumber, strconv.FormatInt(epochInfo.CurrentEpoch, 10)),
			sdk.NewAttribute(
				feedistributiontypes.AttributeKeyOperatorProportions,
				feedistributiontypes.OperatorRewardProportions(rewardProportions).String()),
		),
	)
	return nil
}

func (k Keeper) GetAVSRewardDistribution(ctx sdk.Context, avsAddr string) (*feedistributiontypes.AVSRewardDistribution, error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), feedistributiontypes.KeyPrefixAVSRewardDistribution)
	value := store.Get(common.HexToAddress(avsAddr).Bytes())
	if value == nil {
		return nil, feedistributiontypes.ErrNotAVSRewardDistribution
	}

	ret := feedistributiontypes.AVSRewardDistribution{}
	k.cdc.MustUnmarshal(value, &ret)
	return &ret, nil
}

func (k Keeper) DeleteAVSRewardDistribution(ctx sdk.Context, avsAddr string) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), feedistributiontypes.KeyPrefixAVSRewardDistribution)
	store.Delete(common.HexToAddress(avsAddr).Bytes())
	return nil
}

func (k Keeper) SetAVSRewardParam(ctx sdk.Context, avsAddr string, param feedistributiontypes.AVSRewardParam) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), feedistributiontypes.KeyPrefixAVSRewardParam)
	bz := k.cdc.MustMarshal(&param)
	store.Set(common.HexToAddress(avsAddr).Bytes(), bz)

	// emit event for indexers
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			feedistributiontypes.EventTypeAVSRewardParamSet,
			sdk.NewAttribute(feedistributiontypes.AttributeKeyAvsAddress, avsAddr),
			sdk.NewAttribute(feedistributiontypes.AttributeKeyAVSRewardParam, param.ToEventString()),
		),
	)
	return nil
}

func (k Keeper) GetAVSRewardParam(ctx sdk.Context, avsAddr string) (*feedistributiontypes.AVSRewardParam, error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), feedistributiontypes.KeyPrefixAVSRewardParam)
	value := store.Get(common.HexToAddress(avsAddr).Bytes())
	if value == nil {
		return nil, feedistributiontypes.ErrNoKeyInTheStore.Wrapf("GetAVSRewardParam, AvsAddr:%s", avsAddr)
	}

	ret := feedistributiontypes.AVSRewardParam{}
	k.cdc.MustUnmarshal(value, &ret)
	return &ret, nil
}

func (k Keeper) EpochRewardFnForDogfood() AVSEpochRewardFn {
	return func(ctx sdk.Context, _ string) (sdk.DecCoins, error) {
		feeCollector := k.authKeeper.GetModuleAccount(ctx, k.feeCollectorName)
		feesCollectedInt := k.bankKeeper.GetAllBalances(ctx, feeCollector.GetAddress())
		feesCollected := sdk.NewDecCoinsFromCoins(feesCollectedInt...)
		// transfer collected fees to the distribution module account
		err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, k.feeCollectorName, feedistributiontypes.ModuleName, feesCollectedInt)
		if err != nil {
			return nil, err
		}

		chainIDWithoutRevision := utils.ChainIDWithoutRevision(ctx.ChainID())
		dogfoodAVSAddr := utils.GenerateAVSAddress(chainIDWithoutRevision)
		// fund the reward pool of dogfood AVS
		validRewards := make(sdk.DecCoins, 0)
		for _, singleReward := range feesCollected {
			assetID, _, err := k.GetAVSRewardAssetByDenomination(ctx, dogfoodAVSAddr, singleReward.Denom)
			if err != nil {
				ctx.Logger().Error("can't get the dogfood reward asset by the denomination", "denomination", singleReward.Denom)
				// An invalid reward shouldn't affect valid rewards.
				continue
			}
			if singleReward.Amount.IsNil() || singleReward.Amount.IsNegative() {
				ctx.Logger().Error("invalid amount when funding reward pool of dogfood", "reward", singleReward)
				// An invalid reward shouldn't affect valid rewards.
				continue
			}
			err = k.UpdateAVSRewardAssetState(ctx, dogfoodAVSAddr, assetID,
				&feedistributiontypes.DeltaAVSRewardAssetState{
					RewardPoolBalance: singleReward.Amount,
					RewardPoolTotal:   singleReward.Amount,
				})
			if err != nil {
				// continue to skip the invalid reward asset
				ctx.Logger().Error("can't update the asset state for dogfood reward", "reward", singleReward)
				continue
			}
			validRewards = append(validRewards, singleReward)
		}

		return validRewards, nil
	}
}

// getOverlap returns the number of blocks in the intersection of [start, end] and [epochStart, epochEnd].
func getOverlap(start, end, epochStart, epochEnd uint64) uint64 {
	if end < epochStart || start > epochEnd {
		return 0
	}
	s := max(start, epochStart)
	e := min(end, epochEnd)
	return e - s + 1
}

// CalcJailedBlocksInEpoch calculates the number of jailed blocks within a given epoch.
// It traverses the jailToggleHeights slice in reverse to skip toggle events that are completely before the epoch.
func CalcJailedBlocksInEpoch(jailToggleHeights []uint64, epochStart, epochEnd uint64, jailed bool) (uint64, error) {
	n := len(jailToggleHeights)
	totalJailed := uint64(0)

	isOdd := n%2 == 1
	i := n - 1

	if (isOdd && !jailed) || (!isOdd && jailed) {
		return 0, operatortypes.ErrInvalidJailStatusOrHeights.Wrapf("CalcJailedBlocksInEpoch,jailed:%v,jailToggleHeights:%d", jailed, n)
	}

	for i > 0 {
		var jailStart, jailEnd uint64

		// If toggle list has odd length, the operator is still jailed.
		// This means there's no recorded unjail height yet, so we treat the current epoch end
		// as a "virtual" unjail height for the purpose of calculating overlap.
		// This allows us to reuse the same (jail+1, unjail] pattern as in the paired case.
		if isOdd && i == n-1 {
			jailStart = jailToggleHeights[i] + 1
			jailEnd = epochEnd
			totalJailed += getOverlap(jailStart, jailEnd, epochStart, epochEnd)
			i--
			continue
		}

		// Handle jail-unjail pairs: (jail+1, unjail]
		jailEnd = jailToggleHeights[i]
		jailStart = jailToggleHeights[i-1] + 1

		// Skip if the whole jail interval is before the epoch
		if jailEnd < epochStart {
			break
		}

		totalJailed += getOverlap(jailStart, jailEnd, epochStart, epochEnd)
		i -= 2
	}

	return totalJailed, nil
}

// VotingPowerRatioAfterJail returns the voting power adjustment ratio considering jail.
// In the IMUA protocol, rewards are distributed on an epoch basis, while jail and unjail actions occur
// on a block basis. Therefore, when distributing rewards for an epoch, if an operator is jailed, they
// should not receive rewards for the duration they were jailed.
// Our approach to handling this is as follows:
// We calculate the proportion of time within the epoch during which the operator was not jailed, relative
// to the entire epoch. This proportion is then used to adjust the operator’s effective voting power,
// thereby reducing the rewards they receive.
// The advantage of applying this adjustment to voting power instead of directly reducing the operator’s
// reward for the epoch is that the total rewards for the epoch will still be fully distributed among all
// operators. This prevents the portion of rewards lost by a jailed operator from being redirected to the
// community pool.
func (k Keeper) VotingPowerRatioAfterJail(ctx sdk.Context, operator, avsAddr string) (math.LegacyDec, error) {
	optedInfo, err := k.operatorKeeper.GetOptedInfo(ctx, operator, avsAddr)
	if err != nil {
		return math.LegacyDec{}, err
	}
	epochInfo, err := k.avsKeeper.GetAVSEpochInfo(ctx, avsAddr)
	if err != nil {
		return math.LegacyDec{}, err
	}

	currentHeight := uint64(ctx.BlockHeight())
	currentEpochStartHeight := uint64(epochInfo.CurrentEpochStartHeight)
	currentEpochBlockNumber := currentHeight - currentEpochStartHeight + 1
	totalJailed, err := CalcJailedBlocksInEpoch(optedInfo.JailToggleHeights, currentEpochStartHeight, currentHeight, optedInfo.Jailed)
	if err != nil {
		return math.LegacyDec{}, err
	}
	if totalJailed > currentEpochBlockNumber {
		return math.LegacyDec{}, operatortypes.ErrInvalidJailedBlockNumber.Wrapf("totalJailed:%d,currentEpochBlockNumber:%d", totalJailed, currentEpochBlockNumber)
	}

	effectiveBlockNumber := currentEpochBlockNumber - totalJailed
	ratio := math.LegacyNewDec(int64(effectiveBlockNumber)).QuoInt64(int64(currentEpochBlockNumber)) // #nosec G115
	return ratio, nil
}

type operatorVotingPowerCallback func(operatorAddr string, votingPower math.LegacyDec) bool

// CommonRewardProportion : Abstracts the common logic for calculating operator reward distribution
// for Dogfood and other AVSs. The logic differences will be handled in the caller function.
// The main difference is that Dogfood retrieves the voting power from the Dogfood module,
// whereas other AVSs retrieve the voting power from the Operator module.
func (k Keeper) CommonRewardProportion(
	ctx sdk.Context,
	avsAddr string,
	totalVotingPower math.LegacyDec,
	iterateOperators func(operatorVotingPowerCallback) error,
) ([]feedistributiontypes.OperatorRewardProportion, error) {
	operatorRewardProportions := make([]feedistributiontypes.OperatorRewardProportion, 0)
	operatorVotingPowersAfterJail := make([]operatortypes.OperatorVotingPower, 0)
	totalPowerAfterJail := totalVotingPower
	isHandleJail := false
	if !totalVotingPower.IsPositive() {
		// return null reward proportions, because the rewards should be allocated to
		// the community pool.
		return operatorRewardProportions, nil
	}

	callBackFn := func(operatorAddr string, votingPowerDec math.LegacyDec) bool {
		effectiveRatio, err := k.VotingPowerRatioAfterJail(ctx, operatorAddr, avsAddr)
		if err != nil {
			ctx.Logger().Error("err when getting the effective voting power ratio after jail; skipping", "operator", operatorAddr, "err", err)
			// return false to continue handling the other operators
			return false
		}
		if !isHandleJail {
			rewardProportion := votingPowerDec.QuoTruncate(totalVotingPower)
			operatorRewardProportions = append(operatorRewardProportions,
				feedistributiontypes.OperatorRewardProportion{
					OperatorAddr:     operatorAddr,
					RewardProportion: rewardProportion,
				})
		}
		if effectiveRatio.LT(math.LegacyNewDec(1)) {
			// handle the exceptional case where the effective ratio is less than zero.
			if effectiveRatio.LT(math.LegacyZeroDec()) {
				effectiveRatio = math.LegacyZeroDec()
			}
			effectiveVotingPowerDec := votingPowerDec.Mul(effectiveRatio)
			totalPowerAfterJail.SubMut(votingPowerDec.Sub(effectiveVotingPowerDec))
			votingPowerDec = effectiveVotingPowerDec
			if !isHandleJail {
				// set the flag when any operator needs to handle a jail event.
				isHandleJail = true
			}
		}
		// It will be used to recalculate the reward proportion when the jail or unjail
		// events need to be handled.
		if votingPowerDec.IsPositive() {
			operatorVotingPowersAfterJail = append(
				operatorVotingPowersAfterJail,
				operatortypes.OperatorVotingPower{
					OperatorAddr: operatorAddr,
					VotingPower:  votingPowerDec,
				})
		}
		return false
	}

	err := iterateOperators(callBackFn)
	if err != nil {
		return nil, err
	}
	if isHandleJail {
		// recalculate the reward proportion
		operatorRewardProportions = make([]feedistributiontypes.OperatorRewardProportion, 0)
		for _, operatorVotingPower := range operatorVotingPowersAfterJail {
			effectiveProportion := operatorVotingPower.VotingPower.QuoTruncate(totalPowerAfterJail)
			operatorRewardProportions = append(operatorRewardProportions,
				feedistributiontypes.OperatorRewardProportion{
					OperatorAddr:     operatorVotingPower.OperatorAddr,
					RewardProportion: effectiveProportion,
				})
		}
	}
	return operatorRewardProportions, nil
}

func (k Keeper) RewardProportionsFnForDogfood() OperatorRewardProportionsFn {
	return func(ctx sdk.Context, avsAddr string) ([]feedistributiontypes.OperatorRewardProportion, error) {
		previousTotalPower := k.StakingKeeper.GetLastTotalPower(ctx).Int64()
		totalVotingPower := math.LegacyNewDec(previousTotalPower)

		iterateOperators := func(callback operatorVotingPowerCallback) error {
			var err error
			var consensusKey cryptotypes.PubKey
			var wrappedKey keys.WrappedConsKey
			var found bool
			var accAddress sdk.AccAddress

			allValidators := k.StakingKeeper.GetAllImuachainValidators(ctx)
			for i, val := range allValidators {
				consensusKey, err = val.ConsPubKey()
				if err != nil {
					ctx.Logger().Error("Failed to deserialize public key; skipping", "error", err, "i", i)
					continue
				}
				wrappedKey = keys.NewWrappedConsKeyFromSdkKey(consensusKey)
				found, accAddress = k.operatorKeeper.GetOperatorAddressForChainIDAndConsAddr(
					ctx, utils.ChainIDWithoutRevision(ctx.ChainID()), wrappedKey.ToConsAddr(),
				)
				if !found {
					ctx.Logger().Error("Operator address not found; skipping", "consAddress", wrappedKey.ToConsAddr(), "i", i)
					continue
				}
				isBreak := callback(accAddress.String(), math.LegacyNewDec(val.Power))
				if isBreak {
					break
				}
			}
			return nil
		}

		return k.CommonRewardProportion(ctx, avsAddr, totalVotingPower, iterateOperators)
	}
}

func (k Keeper) CustomizedEpochRewardFnForAVSs() AVSEpochRewardFn {
	return func(ctx sdk.Context, avsAddr string) (sdk.DecCoins, error) {
		rewardDistribution, err := k.GetAVSRewardDistribution(ctx, avsAddr)
		if err != nil {
			return nil, err
		}
		return rewardDistribution.Rewards, nil
	}
}

func (k Keeper) CustomizedRewardProportionsFnForAVSs() OperatorRewardProportionsFn {
	return func(ctx sdk.Context, avsAddr string) ([]feedistributiontypes.OperatorRewardProportion, error) {
		rewardDistribution, err := k.GetAVSRewardDistribution(ctx, avsAddr)
		if err != nil {
			return nil, err
		}
		return rewardDistribution.OperatorRewardProportions, nil
	}
}

// DefaultEpochRewardFnForAVSs : The current implementation is the same as the `CustomizedEpochRewardFnForAVSs`,
// because we haven't determined a general reward inflation curve for multiple AVSs, and the dogfood also mints
// a fixed amount of reward each epoch. In this case, the AVS can set a fixed reward through the precompile
// interface via `SetAVSEpochRewardExclusive`. This will be the same as the current implementation of dogfood.
// TODO: This function should be modified once we determine a general reward inflation mechanism for multiple AVSs.
func (k Keeper) DefaultEpochRewardFnForAVSs() AVSEpochRewardFn {
	return func(ctx sdk.Context, avsAddr string) (sdk.DecCoins, error) {
		rewardDistribution, err := k.GetAVSRewardDistribution(ctx, avsAddr)
		if err != nil {
			return nil, err
		}
		return rewardDistribution.Rewards, nil
	}
}

func (k Keeper) DefaultRewardProportionsFnForAVSs() OperatorRewardProportionsFn {
	return func(ctx sdk.Context, avsAddr string) ([]feedistributiontypes.OperatorRewardProportion, error) {
		totalVotingPower, err := k.operatorKeeper.GetAVSUSDValue(ctx, avsAddr)
		if err != nil {
			return nil, err
		}
		iterateOperators := func(callback operatorVotingPowerCallback) error {
			opFunc := func(operator string, optedUSDValues *operatortypes.OperatorOptedUSDValue) error {
				if optedUSDValues.ActiveUSDValue.IsPositive() {
					callback(operator, optedUSDValues.ActiveUSDValue)
				}
				return nil
			}
			return k.operatorKeeper.IterateOperatorUSDValuesForAVS(ctx, avsAddr, false, opFunc)
		}
		return k.CommonRewardProportion(ctx, avsAddr, totalVotingPower, iterateOperators)
	}
}

func (k Keeper) AVSRewardAndProportionsByParam(ctx sdk.Context, avsAddr string) (bool, feedistributiontypes.EpochRewardsAndProportions, error) {
	var avsEpochRewardFn AVSEpochRewardFn
	var operatorRewardProportionsFn OperatorRewardProportionsFn
	var isDogfood bool
	// check if the avs is dogfood
	chainIDWithoutRevision := utils.ChainIDWithoutRevision(ctx.ChainID())
	dogfoodAVSAddr := utils.GenerateAVSAddress(chainIDWithoutRevision)
	if dogfoodAVSAddr == avsAddr {
		isDogfood = true
		avsEpochRewardFn = k.EpochRewardFnForDogfood()
		operatorRewardProportionsFn = k.RewardProportionsFnForDogfood()
	} else {
		param, err := k.GetAVSRewardParam(ctx, avsAddr)
		if err != nil {
			return false, feedistributiontypes.EpochRewardsAndProportions{}, err
		}
		// check the reward parameter of AVS
		if param.CustomRewardInflation {
			avsEpochRewardFn = k.CustomizedEpochRewardFnForAVSs()
		} else {
			avsEpochRewardFn = k.DefaultEpochRewardFnForAVSs()
		}
		if param.CustomOperatorRatio {
			operatorRewardProportionsFn = k.CustomizedRewardProportionsFnForAVSs()
		} else {
			operatorRewardProportionsFn = k.DefaultRewardProportionsFnForAVSs()
		}
	}
	avsEpochReward, err := avsEpochRewardFn(ctx, avsAddr)
	if err != nil {
		return isDogfood, feedistributiontypes.EpochRewardsAndProportions{}, err
	}
	// don't calculate the operator reward proportions if the epoch rewards is null
	if len(avsEpochReward) == 0 {
		return isDogfood, feedistributiontypes.EpochRewardsAndProportions{}, nil
	}
	operatorRewardProportions, err := operatorRewardProportionsFn(ctx, avsAddr)
	if err != nil {
		return isDogfood, feedistributiontypes.EpochRewardsAndProportions{}, err
	}
	return isDogfood, feedistributiontypes.EpochRewardsAndProportions{
		Rewards:                   avsEpochReward,
		OperatorRewardProportions: operatorRewardProportions,
	}, nil
}
