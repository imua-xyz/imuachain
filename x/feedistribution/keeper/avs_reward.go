package keeper

import (
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
	bz := k.cdc.MustMarshal(&distribution)
	store.Set(common.HexToAddress(avsAddr).Bytes(), bz)
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

func (k Keeper) SetRewardDistributionForDogfood(ctx sdk.Context) error {

}
