package keeper

import (
	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/core"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/imua-xyz/imuachain/x/assets/types"
	rtypes "github.com/imua-xyz/imuachain/x/reward/types"
)

type RewardParams struct {
	ClientChainLzID       uint64
	Action                types.CrossChainOpType
	AssetsAddress         []byte
	WithdrawRewardAddress []byte
	OpAmount              sdkmath.Int
}

func (k Keeper) PostTxProcessing(_ sdk.Context, _ core.Message, _ *ethtypes.Receipt) error {
	return errorsmod.Wrap(rtypes.ErrNotSupportYet, "reward module doesn't support PostTxProcessing")
}

func (k Keeper) RewardForWithdraw(sdk.Context, *RewardParams) error {
	// TODO: rewards aren't yet supported
	// it is safe to return an error, since the precompile call will prevent an error
	// if err != nil return false
	// the false will ensure no unnecessary LZ messages are sent by the gateway
	return rtypes.ErrNotSupportYet
	// // check event parameter then execute RewardForWithdraw operation
	// if event.OpAmount.IsNegative() {
	// 	return errorsmod.Wrap(rtypes.ErrRewardAmountIsNegative, fmt.Sprintf("the amount is:%s", event.OpAmount))
	// }
	// stakeID, assetID := getStakeIDAndAssetID(event)
	// // check is asset exist
	// if !k.assetsKeeper.IsStakingAsset(ctx, assetID) {
	// 	return errorsmod.Wrap(rtypes.ErrRewardAssetNotExist, fmt.Sprintf("the assetID is:%s", assetID))
	// }

	// // TODO verify the reward amount is valid
	// changeAmount := types.DeltaStakerSingleAsset{
	// 	TotalDepositAmount: event.OpAmount,
	// 	WithdrawableAmount: event.OpAmount,
	// }
	// // TODO: there should be a reward pool to be transferred from for native tokens' reward, don't update staker-asset-info,
	// just transfer im-native-token:pool->staker or handled by validators since the reward would be transferred to validators directly.
	// if assetID != types.ImuachainAssetID {
	// 	err := k.assetsKeeper.UpdateStakerAssetState(ctx, stakeID, assetID, changeAmount)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	if err = k.assetsKeeper.UpdateStakingAssetTotalAmount(ctx, assetID, event.OpAmount); err != nil {
	// 		return err
	// 	}
	// }
	// return nil
}

// WithdrawDelegationRewards is an implementation of a function in the distribution interface.
// Since this module acts as the distribution module for our network, this function is here.
// When implemented, this function should find the pending (native token) rewards for the
// specified delegator and validator address combination and send them to the delegator address.
func (Keeper) WithdrawDelegationRewards(
	sdk.Context, sdk.AccAddress, sdk.ValAddress,
) (sdk.Coins, error) {
	return nil, rtypes.ErrNotSupportYet
}
