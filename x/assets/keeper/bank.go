package keeper

import (
	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	"github.com/ethereum/go-ethereum/common/hexutil"

	assetstypes "github.com/ExocoreNetwork/exocore/x/assets/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type DepositWithdrawParams struct {
	ClientChainLzID uint64
	Action          assetstypes.CrossChainOpType
	AssetsAddress   []byte
	StakerAddress   []byte
	OpAmount        sdkmath.Int
	ValidatorID     []byte
}

// PerformDepositOrWithdraw the assets precompile contract will call this function to update asset state
// when there is a deposit or withdraw.
func (k Keeper) PerformDepositOrWithdraw(ctx sdk.Context, params *DepositWithdrawParams) error {
	// check params parameter before executing operation
	if params.OpAmount.IsNegative() {
		return assetstypes.ErrInvalidAmount.Wrapf(
			"negative amount:%s", params.OpAmount,
		)
	}
	stakerID, assetID := assetstypes.GetStakerIDAndAssetID(params.ClientChainLzID, params.StakerAddress, params.AssetsAddress)
	if !k.IsStakingAsset(ctx, assetID) {
		return errorsmod.Wrapf(assetstypes.ErrNoClientChainAssetKey, "assetAddr:%s clientChainID:%v", hexutil.Encode(params.AssetsAddress), params.ClientChainLzID)
	}

	actualOpAmount := params.OpAmount
	switch params.Action {
	case assetstypes.DepositLST, assetstypes.DepositNST:
	case assetstypes.WithdrawLST, assetstypes.WithdrawNST:
		actualOpAmount = actualOpAmount.Neg()
	default:
		return errorsmod.Wrapf(assetstypes.ErrInvalidOperationType, "the operation type is: %v", params.Action)
	}

	changeAmount := assetstypes.DeltaStakerSingleAsset{
		TotalDepositAmount: actualOpAmount,
		WithdrawableAmount: actualOpAmount,
	}
	// don't update staker info for exo-native-token
	// TODO: do we need additional process for exo-native-token ?
	if assetID != assetstypes.ExocoreAssetID {
		// update asset state of the specified staker
		err := k.UpdateStakerAssetState(ctx, stakerID, assetID, changeAmount)
		if err != nil {
			return errorsmod.Wrapf(err, "stakerID:%s assetID:%s", stakerID, assetID)
		}

		// update total amount of the deposited asset
		err = k.UpdateStakingAssetTotalAmount(ctx, assetID, actualOpAmount)
		if err != nil {
			return errorsmod.Wrapf(err, "assetID:%s", assetID)
		}
	}
	return nil
}
