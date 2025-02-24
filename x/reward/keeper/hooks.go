package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	epochstypes "github.com/imua-xyz/imuachain/x/epochs/types"
	"github.com/imua-xyz/imuachain/x/reward/types"
)

// EpochsHooksWrapper is the wrapper structure that implements the epochs hooks for the avs
// keeper.
type EpochsHooksWrapper struct {
	keeper *Keeper
}

// Interface guard
var _ epochstypes.EpochHooks = EpochsHooksWrapper{}

// EpochsHooks returns the epochs hooks wrapper. It follows the "accept interfaces, return
// concretes" pattern.
func (k *Keeper) EpochsHooks() EpochsHooksWrapper {
	return EpochsHooksWrapper{k}
}

// BeforeEpochStart: noop, We don't need to do anything here
func (wrapper EpochsHooksWrapper) BeforeEpochStart(_ sdk.Context, _ string, _ int64) {
}

// AfterEpochEnd distribute the reward at the end of each epoch end
func (wrapper EpochsHooksWrapper) AfterEpochEnd(
	ctx sdk.Context, epochIdentifier string, epochNumber int64,
) {
	expEpochIdentifier := wrapper.keeper.GetEpochIdentifier(ctx)
	if epochIdentifier != expEpochIdentifier {
		wrapper.keeper.Logger(ctx).Debug(
			"epochIdentifier didn't equal to expEpochIdentifier",
			"epochIdentifier", epochIdentifier,
		)
		return
	}

	// get all the avs address bypass the epoch end
	epochEndAVS := wrapper.keeper.avsKeeper.GetEpochEndAVSs(ctx, epochIdentifier, epochNumber)

	pool := wrapper.keeper.getPool(ctx, types.ModuleName)
	// distribute the reward to the avs accordingly
	ForEach(epochEndAVS, func(p string) {
		avsInfo, err := pool.k.avsKeeper.GetAVSInfo(ctx, p)
		if err != nil {
			wrapper.keeper.Logger(ctx).Error(
				"get avsInfo error",
				"avsInfo err", err,
			)
			return
		}
		assetID := avsInfo.Info.AssetIDs
		ownerAddress := avsInfo.Info.AvsOwnerAddresses

		for _, operator := range ownerAddress {
			opAddr, err := sdk.AccAddressFromBech32(operator)
			if err != nil {
				wrapper.keeper.Logger(ctx).Error(
					"get operatorInfo error",
					"operatorInfo err", err,
				)
				return
			}
			for _, asset := range assetID {
				assetInfo, err := wrapper.keeper.assetsKeeper.GetStakingAssetInfo(ctx, asset)
				if err != nil {
					wrapper.keeper.Logger(ctx).Error(
						"get assetInfo error",
						"assetInfo err", err,
					)
					return
				}
				if wrapper.keeper.assetsKeeper.IsOperatorAssetExist(ctx, opAddr, asset) {
					coin := sdk.Coin{
						Denom:  assetInfo.AssetBasicInfo.Symbol,
						Amount: sdk.NewInt(avsInfo.Info.AssetRewardAmountEpochBasis[asset]),
					}
					pool.AddReward(p, coin)
				}
			}
		}
	})
}

// ForEach apply the function on every element within the slice
func ForEach[T any](source []T, f func(T)) {
	for i := range source {
		f(source[i])
	}
}
