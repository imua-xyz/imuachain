package keeper

import (
	sdkmath "cosmossdk.io/math"
	"github.com/ExocoreNetwork/exocore/x/assets/types"
	delegationtypes "github.com/ExocoreNetwork/exocore/x/delegation/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	MaxSlashProportion = 1
)

func (k Keeper) UpdateNSTBalance(
	ctx sdk.Context, stakerID, assetID string, amount sdkmath.Int,
) error {
	if amount.IsPositive() {
		// If the balance increases due to the client chain PoS staking reward, the increased
		// amount can be considered a virtual deposit event. However, the increased amount needs
		// to be manually delegated by the staker if they want it to contribute to voting power.
		// Of course, we can also treat it as both a virtual deposit and delegation event if we
		// think this approach is better. In that case, we would proportionally delegate the
		// increased amount to all operators to whom the related staker has already delegated
		// this native token.
		err := k.assetsKeeper.UpdateStakerAssetState(ctx, stakerID, assetID, types.DeltaStakerSingleAsset{
			TotalDepositAmount: amount,
			WithdrawableAmount: amount,
		})
		if err != nil {
			return err
		}
	} else if amount.IsNegative() {
		// If the balance decreases due to the client chain PoS slashing, the decreased amount
		// will be slashed from the withdrawable amount first, the pending undelegation second,
		// and the delegated share last if there is still a remaining amount that needs to be slashed.
		// When slash from the delegated share, we can proportionally decrease from all operators
		// to whom the related staker has already delegated.

		// slash from the withdrawable amount
		assetInfo, err := k.assetsKeeper.GetStakerSpecifiedAssetInfo(ctx, stakerID, assetID)
		if err != nil {
			return err
		}
		slashFromWithdrawable := amount.Neg()
		pendingSlashAmount := slashFromWithdrawable.Sub(assetInfo.WithdrawableAmount)
		if pendingSlashAmount.IsPositive() {
			slashFromWithdrawable = assetInfo.WithdrawableAmount
		} else {
			pendingSlashAmount = sdkmath.ZeroInt()
		}
		err = k.assetsKeeper.UpdateStakerAssetState(ctx, stakerID, assetID, types.DeltaStakerSingleAsset{
			TotalDepositAmount: slashFromWithdrawable.Neg(),
			WithdrawableAmount: slashFromWithdrawable.Neg(),
		})
		if err != nil {
			return err
		}
		ctx.Logger().Info("UpdateNSTBalance slash from withdrawable amount", "stakerID", stakerID, "assetID", assetID, "slashFromWithdrawable", slashFromWithdrawable, "pendingSlashAmount", pendingSlashAmount)

		// slash from pending undelegations
		if pendingSlashAmount.IsPositive() {
			opFunc := func(undelegationKey []byte, undelegation *delegationtypes.UndelegationRecord) (bool, error) {
				// slash from the single undelegation
				slashAmount := pendingSlashAmount
				pendingSlashAmount = slashAmount.Sub(undelegation.ActualCompletedAmount)
				if pendingSlashAmount.IsPositive() {
					slashAmount = undelegation.ActualCompletedAmount
				}
				undelegation.ActualCompletedAmount = undelegation.ActualCompletedAmount.Sub(slashAmount)
				err = k.assetsKeeper.UpdateStakerAssetState(ctx, stakerID, assetID, types.DeltaStakerSingleAsset{
					TotalDepositAmount: slashAmount.Neg(),
				})
				if err != nil {
					return true, err
				}
				ctx.Logger().Info("UpdateNSTBalance slash from undelegation", "stakerID", stakerID, "assetID", assetID, "operator", undelegation.OperatorAddr, "undelegationKey", undelegationKey, "slashAmount", slashAmount, "pendingSlashAmount", pendingSlashAmount)
				if !pendingSlashAmount.IsPositive() {
					// return ture to break the iteration if there isn't remaining amount to be slashed
					return true, nil
				}
				return false, nil
			}
			err = k.IterateUndelegationsByStakerAndAsset(ctx, stakerID, assetID, true, opFunc)
			if err != nil {
				return err
			}
		}

		// slash from the delegated share
		// the delegated share will be proportionally decreased from all operators to
		// whom the related staker has already delegated
		if pendingSlashAmount.IsPositive() {
			// calculate the slash proportion
			totalDelegatedAmount, err := k.TotalDelegatedAmountForStakerAsset(ctx, stakerID, assetID)
			if err != nil {
				return err
			}
			if !totalDelegatedAmount.IsZero() {
				slashProportion := sdkmath.LegacyNewDecFromBigInt(pendingSlashAmount.BigInt()).Quo(sdkmath.LegacyNewDecFromBigInt(totalDelegatedAmount.BigInt()))
				if slashProportion.GT(sdkmath.LegacyNewDec(MaxSlashProportion)) {
					slashProportion = sdkmath.LegacyNewDec(MaxSlashProportion)
				}
				opFunc := func(keys *delegationtypes.SingleDelegationInfoReq, delegationAmount *delegationtypes.DelegationAmounts) (bool, error) {
					opAccAddr, err := sdk.AccAddressFromBech32(keys.OperatorAddr)
					if err != nil {
						return true, err
					}
					slashShare := delegationAmount.UndelegatableShare.Mul(slashProportion)
					actualSlashAmount, err := k.RemoveShare(ctx, false, opAccAddr, stakerID, assetID, slashShare)
					if err != nil {
						return true, err
					}
					err = k.assetsKeeper.UpdateStakerAssetState(ctx, stakerID, assetID, types.DeltaStakerSingleAsset{
						TotalDepositAmount: actualSlashAmount.Neg(),
					})
					if err != nil {
						return true, err
					}
					pendingSlashAmount = pendingSlashAmount.Sub(actualSlashAmount)
					ctx.Logger().Info("UpdateNSTBalance slash from delegated share", "stakerID", stakerID, "assetID", assetID, "operator", keys.OperatorAddr, "slashProportion", slashProportion, "slashShare", slashShare, "actualSlashAmount", actualSlashAmount, "pendingSlashAmount", pendingSlashAmount)
					return false, nil
				}
				err = k.IterateDelegationsForStakerAndAsset(ctx, stakerID, assetID, opFunc)
				if err != nil {
					return err
				}
			}
		}
		// In this case, we only print a log as a reminder. This situation will only occur when the total slashing amount
		// from the client chain and Exocore chain is greater than the total staking amount.
		if pendingSlashAmount.IsPositive() {
			ctx.Logger().Info("UpdateNSTBalance all staking funds has been slashed, the remaining amount is:", "stakerID", stakerID, "assetID", assetID, "pendingSlashAmount", pendingSlashAmount)
		}
	}
	return nil
}
