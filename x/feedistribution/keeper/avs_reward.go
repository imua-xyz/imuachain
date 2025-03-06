package keeper

import (
	"cosmossdk.io/math"
	"github.com/ExocoreNetwork/exocore/types/keys"
	avstypes "github.com/ExocoreNetwork/exocore/x/avs/types"
	feedistributiontypes "github.com/ExocoreNetwork/exocore/x/feedistribution/types"
	"github.com/ExocoreNetwork/exocore/x/operator/types"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
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
	OperatorRewardProportionsFn func(ctx sdk.Context, avsAddr string) ([]*feedistributiontypes.OperatorRewardProportion, error)
)

// SetAVSRewardDistribution : This function can be called by the reward inflation and allocation mechanisms of AVSs.
// Since different AVSs may have distinct reward models, they can customize this logic through an AVS reward contract.
// In such cases, we need to provide a precompiled interface for this function to facilitate reward contract
// development. Additionally, AVSs might require a keeper to periodically call the customized reward contract and set
// the reward distribution information. This process can be managed by an external service, such as the Chainlink  keeper.
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
	if len(distribution.Rewards) == 0 {
		// don't set if the rewards are null
		return nil
	}
	// check if the reward asset has been registered by the AVS
	for _, rewardCoin := range distribution.Rewards {
		if !k.IsAVSRewardAssetBySymbol(ctx, avsAddr, rewardCoin.Denom) {
			return feedistributiontypes.ErrAVSRewardAssetNotFound.Wrapf("the reward coin isn't registered, avsAddr:%s denomination:%s", avsAddr, rewardCoin.Denom)
		}
	}
	// Check if the operator has opted into the AVS or just opted out
	// of it before the end of the current epoch.
	for _, operator := range distribution.OperatorRewardProportions {
		// We don't check if the operator is jailed here because there might
		// still be partial rewards for jailed operators.
		if !k.operatorKeeper.IsOptedOutAndEffective(ctx, operator.String(), avsAddr) {
			return feedistributiontypes.ErrInvalidRewardDistribution.Wrapf("invalid operator for reward distribution, operator:%s", operator)
		}
	}
	bz := k.cdc.MustMarshal(&distribution)
	store.Set(common.HexToAddress(avsAddr).Bytes(), bz)

	// emit event for indexers
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			feedistributiontypes.EventTypeAVSRewardDistributionSet,
			sdk.NewAttribute(feedistributiontypes.AttributeKeyAvsAddress, avsAddr),
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
func (k Keeper) SetAVSEpochRewardExclusive(ctx sdk.Context, avsAddr string, rewards sdk.DecCoins) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), feedistributiontypes.KeyPrefixAVSRewardDistribution)
	if len(rewards) == 0 {
		// don't set if the rewards are null
		return nil
	}
	// check if the reward asset has been registered by the AVS
	for _, rewardCoin := range rewards {
		if !k.IsAVSRewardAssetBySymbol(ctx, avsAddr, rewardCoin.Denom) {
			return feedistributiontypes.ErrAVSRewardAssetNotFound.Wrapf("the reward coin isn't registered, avsAddr:%s denomination:%s", avsAddr, rewardCoin.Denom)
		}
	}
	rewardDistribution := feedistributiontypes.AVSRewardDistribution{}
	key := common.HexToAddress(avsAddr).Bytes()
	value := store.Get(key)
	if value != nil {
		k.cdc.MustUnmarshal(value, &rewardDistribution)
	}
	rewardDistribution.Rewards = rewards
	bz := k.cdc.MustMarshal(&rewardDistribution)
	store.Set(key, bz)
	// emit event for indexers
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			feedistributiontypes.EventTypeAVSEpochRewardSet,
			sdk.NewAttribute(feedistributiontypes.AttributeKeyAvsAddress, avsAddr),
			sdk.NewAttribute(feedistributiontypes.AttributeKeyEpochRewards, rewards.String()),
		),
	)
	return nil
}

// SetAVSRewardProportionsExclusive sets the operator reward proportions exclusively for an AVS.
// It is also provided to the AVS through a precompile contract.
// This interface allows the AVS to customize the reward proportion of each operator per epoch,
// providing greater flexibility for the AVS.
func (k Keeper) SetAVSRewardProportionsExclusive(ctx sdk.Context, avsAddr string, rewardProportions []*feedistributiontypes.OperatorRewardProportion) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), feedistributiontypes.KeyPrefixAVSRewardDistribution)
	// Check if the operator has opted into the AVS or just opted out
	// of it before the end of the current epoch.
	for _, operator := range rewardProportions {
		// We don't check if the operator is jailed here because there might
		// still be partial rewards for jailed operators.
		if !k.operatorKeeper.IsOptedOutAndEffective(ctx, operator.String(), avsAddr) {
			return feedistributiontypes.ErrInvalidRewardDistribution.Wrapf("invalid operator for reward distribution, operator:%s", operator)
		}
	}
	rewardDistribution := feedistributiontypes.AVSRewardDistribution{}
	key := common.HexToAddress(avsAddr).Bytes()
	value := store.Get(key)
	if value != nil {
		k.cdc.MustUnmarshal(value, &rewardDistribution)
	}
	rewardDistribution.OperatorRewardProportions = rewardProportions
	bz := k.cdc.MustMarshal(&rewardDistribution)
	store.Set(key, bz)
	// emit event for indexers
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			feedistributiontypes.EventTypeAVSRewardProportionsSet,
			sdk.NewAttribute(feedistributiontypes.AttributeKeyAvsAddress, avsAddr),
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

func (k Keeper) EpochRewardFnForDogfood() AVSEpochRewardFn {
	return func(ctx sdk.Context, avsAddr string) (sdk.DecCoins, error) {
		feeCollector := k.authKeeper.GetModuleAccount(ctx, k.feeCollectorName)
		feesCollectedInt := k.bankKeeper.GetAllBalances(ctx, feeCollector.GetAddress())
		feesCollected := sdk.NewDecCoinsFromCoins(feesCollectedInt...)
		return feesCollected, nil
	}
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
	currentEpochBlockNumber := currentHeight - currentEpochStartHeight
	if optedInfo.UnJailedHeight > currentHeight ||
		optedInfo.JailedHeight > currentHeight ||
		optedInfo.UnJailedHeight < optedInfo.JailedHeight {
		return math.LegacyDec{}, feedistributiontypes.ErrInvalidJailOrUnJailHeight.Wrapf("jailed height:%v, unJailed height:%v", optedInfo.JailedHeight, optedInfo.UnJailedHeight)
	}
	var effectiveBlockNumber uint64
	if !optedInfo.Jailed {
		if optedInfo.UnJailedHeight == types.DefaultUnJailedHeight ||
			optedInfo.JailedHeight < currentEpochStartHeight {
			// The jail and unjail events occurred before the current epoch,
			// so they won't affect the reward calculation for the current epoch.
			return math.LegacyNewDec(1), nil
		} else {
			if optedInfo.JailedHeight < currentEpochStartHeight {
				// the jail event occurred before the current epoch but the unJail event occurred in
				// the current epoch
				effectiveBlockNumber = currentHeight - optedInfo.UnJailedHeight
			} else {
				// both the jail and unJail events occurred in the current epoch
				effectiveBlockNumber = currentEpochBlockNumber - (optedInfo.UnJailedHeight - optedInfo.JailedHeight)
			}
		}
	} else {
		if optedInfo.JailedHeight <= currentEpochStartHeight {
			// the jail event occurred before the current epoch, and the operator hasn't been unJailed.
			// so the ratio should be zero.
			return math.LegacyZeroDec(), nil
		} else {
			// the jail event occurred in the current epoch, so the operator can receive some rewards
			// for the period between the start height of the current epoch and the jailed height.
			effectiveBlockNumber = optedInfo.JailedHeight - currentEpochStartHeight
		}
	}
	ratio := math.LegacyNewDec(int64(effectiveBlockNumber)).QuoInt64(int64(currentEpochBlockNumber)) // #nosec G115
	return ratio, nil
}

func (k Keeper) RewardProportionsFnForDogfood() OperatorRewardProportionsFn {
	return func(ctx sdk.Context, avsAddr string) ([]*feedistributiontypes.OperatorRewardProportion, error) {
		allValidators := k.StakingKeeper.GetAllExocoreValidators(ctx)
		previousTotalPower := k.StakingKeeper.GetLastTotalPower(ctx).Int64()
		operatorRewardProportions := make([]*feedistributiontypes.OperatorRewardProportion, 0)
		operatorVotingPowerAfterJail := make([]math.LegacyDec, 0)
		totalPowerAfterJail := previousTotalPower
		isHandleJail := false
		for _, val := range allValidators {
			consensusKey, err := val.ConsPubKey()
			if err != nil {
				return nil, err
			}
			wrappedKey := keys.NewWrappedConsKeyFromSdkKey(consensusKey)
			found, accAddress := k.operatorKeeper.GetOperatorAddressForChainIDAndConsAddr(
				ctx, avstypes.ChainIDWithoutRevision(ctx.ChainID()), wrappedKey.ToConsAddr(),
			)
			if !found {
				return nil, feedistributiontypes.ErrOperatorNotFound
			}

			votingPowerDec := math.LegacyNewDec(val.Power)
			rewardProportion := votingPowerDec.QuoTruncate(math.LegacyNewDec(previousTotalPower))
			operatorRewardProportions = append(operatorRewardProportions,
				&feedistributiontypes.OperatorRewardProportion{
					OperatorAddr:     accAddress.String(),
					RewardProportion: rewardProportion,
				})
			operatorVotingPowers = append(operatorVotingPowers, votingPowerDec)
		}

		if !isHandleJail {
			return operatorRewardProportions, nil
		} else {

		}
		return operatorRewardProportions, nil
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
	return func(ctx sdk.Context, avsAddr string) ([]*feedistributiontypes.OperatorRewardProportion, error) {
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
	return func(ctx sdk.Context, avsAddr string) ([]*feedistributiontypes.OperatorRewardProportion, error) {

		return nil, nil
	}
}

func (k Keeper) DefaultRewardDistributionForAVSs(ctx sdk.Context, avsAddr string) (*feedistributiontypes.AVSRewardDistribution, error) {
	avsRewardAssets, err := k.GetAllAVSRewardAssetSymbols(ctx, avsAddr)
	if err != nil {
		return nil, err
	}
	if len(avsRewardAssets) == 0 {

	}
}
