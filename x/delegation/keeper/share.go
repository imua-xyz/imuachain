package keeper

import (
	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	assetstype "github.com/ExocoreNetwork/exocore/x/assets/types"
	delegationtypes "github.com/ExocoreNetwork/exocore/x/delegation/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// TokensFromShares calculate the token amount of provided shares, then truncated to Int
// It uses `LegacyDec.Quo` to calculate the quotient, `LegacyDec.Quo` perform a bankers
// rounding quotient, so the calculated token amount may be either larger or smaller
// due to precision rounding issues. But it's acceptable, because bankers rounding balances the
// deviation caused by precision, especially when a large number of restakers undertake
// undelegation. Additionally, the last undelegation from an operator will undelegate all
// remaining token to avoid the calculated token amount is bigger than the remaining token
// caused by the bankers rounding.
func TokensFromShares(stakerShare, totalShare sdkmath.LegacyDec, totalAmount sdkmath.Int) (sdkmath.Int, error) {
	if stakerShare.GT(totalShare) {
		return sdkmath.ZeroInt(), errorsmod.Wrapf(delegationtypes.ErrInsufficientShares, "the stakerShare is:%v the totalShare is:%v", stakerShare, totalShare)
	}
	if totalShare.IsZero() {
		if totalAmount.IsZero() {
			// this can happen if everyone exits.
			return sdkmath.ZeroInt(), nil
		}
		return sdkmath.ZeroInt(), delegationtypes.ErrDivisorIsZero
	}
	return (stakerShare.MulInt(totalAmount)).Quo(totalShare).TruncateInt(), nil
}

// SharesFromTokens returns the shares of a delegation given a delegated amount. It
// returns an error if the validator has no tokens.
// It uses `LegacyDec.QuoInt` to calculate the quotient, the calculated result will
// be truncated through the truncation implemented by golang's standard big.Int.
// So the calculated share might tend to be smaller, but it seems acceptable, because
// we need to make sure the staker can't get a bigger share than they should get.
func SharesFromTokens(totalShare sdkmath.LegacyDec, stakerAmount, totalAmount sdkmath.Int) (sdkmath.LegacyDec, error) {
	if totalAmount.IsZero() {
		if totalShare.IsZero() {
			// this can happen if everyone exits.
			return sdkmath.LegacyZeroDec(), nil
		}
		return sdkmath.LegacyZeroDec(), delegationtypes.ErrDivisorIsZero
	}
	return totalShare.MulInt(stakerAmount).QuoInt(totalAmount), nil
}

// CalculateShare calculates the S_j
// S_j = S * T_j / T, `S` and `T` is the current asset share and amount of operator,
// and the T_j represents the change in staker's asset amount when some external
// operations occur, such as: delegation, undelegation and non-instantaneous slashing.
// A special case is the initial delegation, when T = 0 and S = 0, so T_j / T is undefined.
// For the initial delegation, delegator j who delegates T_j tokens receive S_j = T_j shares.
func (k Keeper) CalculateShare(ctx sdk.Context, operator sdk.AccAddress, assetID string, amount sdkmath.Int) (share sdkmath.LegacyDec, err error) {
	// get the total share of operator
	isExist := k.assetsKeeper.IsOperatorAssetExist(ctx, operator, assetID)
	var info *assetstype.OperatorAssetInfo
	if isExist {
		info, err = k.assetsKeeper.GetOperatorSpecifiedAssetInfo(ctx, operator, assetID)
		if err != nil {
			return share, err
		}
	}
	if !isExist || info.TotalShare.IsZero() {
		// the first delegation to a validator sets the exchange rate to one
		share = sdk.NewDecFromInt(amount)
	} else {
		share, err = SharesFromTokens(info.TotalShare, amount, info.TotalAmount)
		if err != nil {
			return share, err
		}
	}
	return share, nil
}

// ValidateUndelegationAmount validates that a given undelegation amount is
// valid based on upon the converted shares. If the amount is valid, the total
// amount of respective shares is returned, otherwise an error is returned.
func (k Keeper) ValidateUndelegationAmount(
	ctx sdk.Context, operator sdk.AccAddress, stakerID, assetID string, amount sdkmath.Int,
) (share sdkmath.LegacyDec, err error) {
	if !amount.IsPositive() {
		return share, delegationtypes.ErrAmountIsNotPositive
	}

	delegationInfo, err := k.GetSingleDelegationInfo(ctx, stakerID, assetID, operator.String())
	if err != nil {
		return share, err
	}

	info, err := k.assetsKeeper.GetOperatorSpecifiedAssetInfo(ctx, operator, assetID)
	if err != nil {
		return share, err
	}

	share, err = SharesFromTokens(info.TotalShare, amount, info.TotalAmount)
	if err != nil {
		return share, err
	}

	if share.GT(delegationInfo.UndelegatableShare) {
		return share, delegationtypes.ErrInsufficientShares
	}

	// Depending on the share, amount can be smaller than unit amount(1stake).
	// If the remain amount after unbonding is smaller than the minimum share,
	// it's completely unbonded to avoid leaving dust shares.
	tolerance, err := SharesFromTokens(info.TotalShare, sdkmath.OneInt(), info.TotalAmount)
	if err != nil {
		return share, err
	}

	if delegationInfo.UndelegatableShare.Sub(share).LT(tolerance) {
		share = delegationInfo.UndelegatableShare
	}

	return share, nil
}

// CalculateSlashShare calculates the actual slash share according to the slash amount,
// it will be used when the slash needs to be executed from the share of the staker.
func (k Keeper) CalculateSlashShare(
	ctx sdk.Context, operator sdk.AccAddress, stakerID, assetID string, slashAmount sdkmath.Int,
) (share sdkmath.LegacyDec, err error) {
	if !slashAmount.IsPositive() {
		return share, delegationtypes.ErrAmountIsNotPositive
	}
	delegationInfo, err := k.GetSingleDelegationInfo(ctx, stakerID, assetID, operator.String())
	if err != nil {
		return share, err
	}
	info, err := k.assetsKeeper.GetOperatorSpecifiedAssetInfo(ctx, operator, assetID)
	if err != nil {
		return share, err
	}
	if slashAmount.GT(info.TotalAmount) {
		slashAmount = info.TotalAmount
	}
	shouldSlashShare, err := SharesFromTokens(info.TotalShare, slashAmount, info.TotalAmount)
	if err != nil {
		return share, err
	}
	if shouldSlashShare.GT(delegationInfo.UndelegatableShare) {
		shouldSlashShare = delegationInfo.UndelegatableShare
	}
	return shouldSlashShare, nil
}

// RemoveShareFromOperator is used to remove the share from an operator when an undelegation
// is submitted, it will return the token amount that should be removed.
func (k Keeper) RemoveShareFromOperator(
	ctx sdk.Context, isUndelegation bool, operator sdk.AccAddress, stakerID, assetID string, share sdkmath.LegacyDec,
) (token sdkmath.Int, err error) {
	if !share.IsPositive() {
		return token, delegationtypes.ErrAmountIsNotPositive
	}
	operatorAssetState, err := k.assetsKeeper.GetOperatorSpecifiedAssetInfo(ctx, operator, assetID)
	if err != nil {
		return token, err
	}
	if share.GT(operatorAssetState.TotalShare) {
		return token, delegationtypes.ErrInsufficientShares
	}

	var removedToken sdkmath.Int
	if operatorAssetState.TotalShare.Equal(share) {
		// last delegation share gets any trimmings
		removedToken = operatorAssetState.TotalAmount
	} else {
		// leave excess tokens in the validator
		// however fully use all the delegator shares
		removedToken, err = TokensFromShares(share, operatorAssetState.TotalShare, operatorAssetState.TotalAmount)
		if err != nil {
			return token, err
		}
	}

	delta := assetstype.DeltaOperatorSingleAsset{
		TotalAmount: removedToken.Neg(),
		TotalShare:  share.Neg(),
	}
	// Check if the staker belongs to the delegated operator. Decrease the operator's share if yes.
	getOperator, err := k.GetAssociatedOperator(ctx, stakerID)
	if err != nil {
		return token, err
	}
	if getOperator != "" && getOperator == operator.String() {
		delta.OperatorShare = share.Neg()
	}
	if isUndelegation {
		delta.PendingUndelegationAmount = removedToken
	}
	err = k.assetsKeeper.UpdateOperatorAssetState(ctx, operator, assetID, delta)
	if err != nil {
		return token, err
	}
	return removedToken, nil
}

// RemoveShare updates all states regarding staker and operator when removing share.
// It might be used for undelegation, slash and native token. For the native token,
// it will be considered a slash operation in exocore when the asset amount is reduced
// by the client chain slash.
func (k Keeper) RemoveShare(
	ctx sdk.Context, isUndelegation bool, operator sdk.AccAddress, stakerID, assetID string, share sdkmath.LegacyDec,
) (removeToken sdkmath.Int, err error) {
	if !share.IsPositive() {
		return removeToken, delegationtypes.ErrAmountIsNotPositive
	}
	// remove share from operator
	removeToken, err = k.RemoveShareFromOperator(ctx, isUndelegation, operator, stakerID, assetID, share)
	if err != nil {
		return removeToken, err
	}

	// update delegation state
	deltaAmount := &delegationtypes.DeltaDelegationAmounts{
		UndelegatableShare: share.Neg(),
	}
	if isUndelegation {
		deltaAmount.WaitUndelegationAmount = removeToken
		// don't update staker asset info for exo-native-token
		if assetID != assetstype.ExocoreAssetID {
			// todo: TotalDepositAmount might be influenced by slash and precision loss,
			// consider removing it, it can be recalculated from the share for RPC query.
			err = k.assetsKeeper.UpdateStakerAssetState(ctx, stakerID, assetID, assetstype.DeltaStakerSingleAsset{
				PendingUndelegationAmount: removeToken,
			})
			if err != nil {
				return removeToken, err
			}
		}
	}
	shareIsZero, err := k.UpdateDelegationState(ctx, stakerID, assetID, operator.String(), deltaAmount)
	if err != nil {
		return removeToken, err
	}
	// if the share is zero, delete the staker from the map to ensure the stakers stored in the map
	// always own assets from the operator.
	if shareIsZero {
		err = k.DeleteStakerForOperator(ctx, operator.String(), assetID, stakerID)
		if err != nil {
			return removeToken, err
		}
	}
	return removeToken, nil
}
