package keeper

import (
	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	"github.com/ethereum/go-ethereum/common/hexutil"

	sdk "github.com/cosmos/cosmos-sdk/types"
	assetstypes "github.com/imua-xyz/imuachain/x/assets/types"
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
// when there is a deposit or withdraw. It returns the final deposit amount, post completion of the deposit
// or withdraw operation.
func (k Keeper) PerformDepositOrWithdraw(
	ctx sdk.Context, params *DepositWithdrawParams,
) (sdkmath.Int, error) {
	// check params parameter before executing operation
	if !params.OpAmount.IsPositive() {
		return sdkmath.ZeroInt(), assetstypes.ErrInvalidAmount.Wrapf(
			"non-positive amount:%s", params.OpAmount,
		)
	}

	// check if staking asset exists
	stakerID, assetID := assetstypes.GetStakerIDAndAssetID(params.ClientChainLzID, params.StakerAddress, params.AssetsAddress)
	if !k.IsStakingAsset(ctx, assetID) {
		return sdkmath.ZeroInt(), assetstypes.ErrNoClientChainAssetKey.Wrapf(
			"assetAddr:%s clientChainID:%v",
			hexutil.Encode(params.AssetsAddress), params.ClientChainLzID,
		)
	}

	// even though this is unlikely to be true, guard against it.
	if assetID == assetstypes.ImuachainAssetID {
		return sdkmath.ZeroInt(), assetstypes.ErrNoClientChainAssetKey.Wrapf(
			"cannot deposit imua native assetID:%s", assetID,
		)
	}

	// add the sign to the (previously positive) amount
	actualOpAmount := params.OpAmount
	switch params.Action {
	case assetstypes.DepositLST, assetstypes.DepositNST:
	case assetstypes.WithdrawLST, assetstypes.WithdrawNST:
		actualOpAmount = actualOpAmount.Neg()
	default:
		return sdkmath.ZeroInt(), assetstypes.ErrInvalidOperationType.Wrapf(
			"the operation type is: %v", params.Action,
		)
	}

	changeAmount := assetstypes.DeltaStakerSingleAsset{
		TotalDepositAmount: actualOpAmount,
		WithdrawableAmount: actualOpAmount,
	}
	// update asset state of the specified staker
	info, err := k.UpdateStakerAssetState(ctx, stakerID, assetID, changeAmount)
	if err != nil {
		return sdkmath.ZeroInt(), errorsmod.Wrapf(
			err, "stakerID:%s assetID:%s", stakerID, assetID,
		)
	}

	// update total amount of the deposited asset
	err = k.UpdateStakingAssetTotalAmount(ctx, assetID, actualOpAmount)
	if err != nil {
		return sdkmath.ZeroInt(), errorsmod.Wrapf(err, "assetID:%s", assetID)
	}

	// TODO: consider emitting EVM event?
	// currently such events are emitted by the HomeChainGateway so this may not be
	// necessary. however, there is no large downside in emitting equivalent EVM
	// events here.

	// return the final deposit amount
	return info.TotalDepositAmount, nil
}
