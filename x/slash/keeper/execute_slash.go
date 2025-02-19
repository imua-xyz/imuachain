package keeper

import (
	sdkmath "cosmossdk.io/math"

	//	"github.com/imua-xyz/imuachain/x/assets/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	assetstypes "github.com/imua-xyz/imuachain/x/assets/types"
)

type SlashParams struct {
	ClientChainLzID           uint64
	Action                    assetstypes.CrossChainOpType
	AssetsAddress             []byte
	OperatorAddress           sdk.AccAddress
	StakerAddress             []byte
	MiddlewareContractAddress []byte
	Proportion                sdkmath.LegacyDec
	OpAmount                  sdkmath.Int
	Proof                     []byte
}

// nolint: unused // This is to be implemented.
type OperatorFrozenStatus struct {
	// nolint: unused // This is to be implemented.
	operatorAddress sdk.AccAddress
	// nolint: unused // This is to be implemented.
	status bool
}

// func (k Keeper) OptIntoSlashing(ctx sdk.Context, event *SlashParams) error {
// 	//TODO implement me
// 	panic("implement me")
// }

// Slash this function might be deprecated, now we use the `slash.go` in the operator module to address the slashing.
// These interfaces are kept here for future refactoring because they are currently called by the slash-related precompile.
func (k Keeper) Slash(_ sdk.Context, _ *SlashParams) error {
	/*	// TODO the stakes are frozen for the impacted middleware, and deposits and withdrawals are disabled as well. All pending deposits and withdrawals for the current epoch will be invalidated.
		//	_ = k.SetFrozenStatus(ctx, string(event.OperatorAddress), true)

		// check event parameter then execute slash operation
		if event.OpAmount.IsNegative() {
			return errorsmod.Wrap(rtypes.ErrSlashAmountIsNegative, fmt.Sprintf("the amount is:%s", event.OpAmount))
		}
		stakeID, assetID := getStakeIDAndAssetID(event)
		// check if asset exists
		if !k.assetsKeeper.IsStakingAsset(ctx, assetID) {
			return errorsmod.Wrap(rtypes.ErrSlashAssetNotExist, fmt.Sprintf("the assetID is:%s", assetID))
		}

		// dont't create stakerasset info for native token.
		// TODO: do we need to do any other process for native token 'else{}' ?
		if assetID != assetstypes.NativeAssetID {
			changeAmount := assetstypes.DeltaStakerSingleAsset{
				TotalDepositAmount: event.OpAmount.Neg(),
				WithdrawableAmount: event.OpAmount.Neg(),
			}

			err := k.assetsKeeper.UpdateStakerAssetState(ctx, stakeID, assetID, changeAmount)
			if err != nil {
				return err
			}
			if err = k.assetsKeeper.UpdateStakingAssetTotalAmount(ctx, assetID, event.OpAmount.Neg()); err != nil {
				return err
			}
		}*/
	return nil
}

// func (k Keeper) FreezeOperator(ctx sdk.Context, event *SlashParams) error {
// 	k.SetFrozenStatus(ctx, string(event.OperatorAddress), true)
// 	return nil
// }

//	func (k Keeper) ResetFrozenStatus(ctx sdk.Context, event *SlashParams) error {
//		k.SetFrozenStatus(ctx, string(event.OperatorAddress), true)
//		return nil
//	}
// func (k Keeper) IsOperatorFrozen(ctx sdk.Context, event *SlashParams) (bool, error) {
// 	return k.GetFrozenStatus(ctx, string(event.OperatorAddress))

// }
// func (k Keeper) OperatorAssetSlashedProportion(ctx sdk.Context, opAddr sdk.AccAddress, assetID string, startHeight, endHeight uint64) sdkmath.LegacyDec {
// 	//TODO
// 	return sdkmath.LegacyNewDec(3)
// }
