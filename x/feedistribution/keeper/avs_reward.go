package keeper

import (
	"cosmossdk.io/math"
	"github.com/ExocoreNetwork/exocore/types/keys"
	avstypes "github.com/ExocoreNetwork/exocore/x/avs/types"
	feedistributiontypes "github.com/ExocoreNetwork/exocore/x/feedistribution/types"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
)

// SetAVSRewardDistribution : This function can be called by the reward inflation and allocation mechanisms of AVSs.
// Since different AVSs may have distinct reward models, they can customize this logic through an AVS reward contract.
// In such cases, we need to provide a precompiled interface for this function to facilitate reward contract
// development. Additionally, AVSs require a keeper to periodically call the customized reward contract and set
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
			feedistributiontypes.ErrAVSRewardAssetNotFound.Wrapf("the reward coin isn't registered, avsAddr:%s denomination:%s", avsAddr, rewardCoin.Denom)
		}
	}
	// Check if the operator has opted into the AVS or just opted out of it before the end of the current epoch.
	for _, operator := range distribution.OperatorRewardProportions {

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
				feedistributiontypes.AttributeKeyRewardPoolBalance,
				feedistributiontypes.OperatorRewardProportions(distribution.OperatorRewardProportions).String()),
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

// DeleteRewardDistributionsByEpoch : The reward distributions of AVSs with the same epoch configuration
// should be deleted after completing the reward distribution for the current epoch.
// This allows them to be reset for the next epoch.
func (k Keeper) DeleteRewardDistributionsByEpoch(ctx sdk.Context, epochIdentifier string, epochNumber int64) error {
	avsList := k.avsKeeper.GetEpochEndAVSs(ctx, epochIdentifier, epochNumber)
	store := prefix.NewStore(ctx.KVStore(k.storeKey), feedistributiontypes.KeyPrefixAVSRewardDistribution)
	for _, avs := range avsList {
		store.Delete(common.HexToAddress(avs).Bytes())
	}
	return nil
}

func (k Keeper) RewardDistributionForDogfood(ctx sdk.Context) (*feedistributiontypes.AVSRewardDistribution, error) {
	feeCollector := k.authKeeper.GetModuleAccount(ctx, k.feeCollectorName)
	feesCollectedInt := k.bankKeeper.GetAllBalances(ctx, feeCollector.GetAddress())
	feesCollected := sdk.NewDecCoinsFromCoins(feesCollectedInt...)
	allValidators := k.StakingKeeper.GetAllExocoreValidators(ctx)
	previousTotalPower := k.StakingKeeper.GetLastTotalPower(ctx).Int64()
	ret := &feedistributiontypes.AVSRewardDistribution{
		Rewards:                   feesCollected,
		OperatorRewardProportions: make([]*feedistributiontypes.OperatorRewardProportion, 0),
	}
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
		rewardProportion := math.LegacyNewDec(val.Power).QuoTruncate(math.LegacyNewDec(previousTotalPower))
		ret.OperatorRewardProportions = append(ret.OperatorRewardProportions, &feedistributiontypes.OperatorRewardProportion{
			OperatorAddr:     accAddress.String(),
			RewardProportion: rewardProportion,
		})
	}
	return ret, nil
}

func (k Keeper) DefaultRewardDistributionForAVSs(ctx sdk.Context, avsAddr string) (*feedistributiontypes.AVSRewardDistribution, error) {
	avsRewardAssets, err := k.GetAllAVSRewardAssetSymbols(ctx, avsAddr)
	if err != nil {
		return nil, err
	}
	if len(avsRewardAssets) == 0 {

	}
}
