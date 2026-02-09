package keeper

import (
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/imua-xyz/imuachain/utils"
	assetstype "github.com/imua-xyz/imuachain/x/assets/types"
	feedistributiontypes "github.com/imua-xyz/imuachain/x/feedistribution/types"
)

func (k Keeper) generalWithdrawFromAVS(ctx sdk.Context, avs, assetID string, withdrawAmount sdkmath.Int,
	imuaReceiptAddr sdk.AccAddress, rewards sdk.DecCoins,
) (sdkmath.Int, sdkmath.Int, sdk.DecCoins, sdk.DecCoins, error) {
	if withdrawAmount.IsNil() || withdrawAmount.IsNegative() {
		return sdkmath.Int{}, sdkmath.Int{}, rewards, nil, feedistributiontypes.ErrInvalidInputParameter.Wrapf(
			"generalWithdrawFromAVS, the withdraw amount is nil or negative, amount:%s", withdrawAmount)
	}
	// check and calculate the actual amount withdrawable for an AVS
	rewardAssetInfo, err := k.GetAVSRewardAsset(ctx, avs, assetID)
	if err != nil {
		return sdkmath.Int{}, sdkmath.Int{}, rewards, nil, err
	}
	rewardDecimalAmount := rewards.AmountOf(rewardAssetInfo.RewardAssetInfo.RewardDenomination)
	if rewardDecimalAmount.IsZero() {
		// do nothing if there isn't this asset in the input rewards
		return sdkmath.ZeroInt(), sdkmath.ZeroInt(), rewards, sdk.DecCoins{}, nil
	}

	// withdraw all rewards if the input amount is 0
	withdrawAmountPerAVSDec := rewardDecimalAmount
	if !withdrawAmount.IsZero() {
		withdrawAmountPerAVSDec = feedistributiontypes.ScaleIntByDecimals(withdrawAmount, rewardAssetInfo.RewardAssetInfo.DenominationExponent)
	}

	if withdrawAmountPerAVSDec.LT(sdkmath.LegacyZeroDec()) {
		// stop withdrawing the reward
		return sdkmath.Int{}, sdkmath.Int{}, rewards, nil,
			feedistributiontypes.ErrInvalidInputParameter.Wrapf("generalWithdrawFromAVS: negative withdraw amount,withdrawAmountPerAVSDec:%s", withdrawAmountPerAVSDec)
	} else if withdrawAmountPerAVSDec.IsZero() {
		// do nothing if the withdraw amount is zero
		return sdkmath.ZeroInt(), sdkmath.ZeroInt(), rewards, sdk.DecCoins{}, nil
	}

	// the actual amount is the minimum of the reward pool balance, the reward amount,
	// and the requested withdraw amount.
	actualWithdrawAmountDec := sdkmath.LegacyMinDec(rewardAssetInfo.RewardAssetState.RewardPoolBalance,
		sdkmath.LegacyMinDec(rewardDecimalAmount, withdrawAmountPerAVSDec))
	if actualWithdrawAmountDec.IsZero() {
		// do nothing if the actual withdraw amount is zero
		return sdkmath.ZeroInt(), sdkmath.ZeroInt(), rewards, sdk.DecCoins{}, nil
	}
	// decrease the withdrawing amount from the outstanding reward
	subReward := sdk.DecCoins{
		sdk.NewDecCoinFromDec(rewardAssetInfo.RewardAssetInfo.RewardDenomination, actualWithdrawAmountDec),
	}
	rewardsAfterSub, hasNegative := rewards.SafeSub(subReward)
	if hasNegative {
		return sdkmath.Int{}, sdkmath.Int{}, rewards, nil, feedistributiontypes.ErrNegativeCoinAmount.Wrapf("WithdrawStakerRewards: avs:%s, assetID:%s,denomination:%s", avs, assetID, rewardAssetInfo.RewardAssetInfo.RewardDenomination)
	}

	// TruncateInt in `UnscaleDecToInt` to ensure the vault has enough fund
	actualWithdrawAmountInt := feedistributiontypes.UnscaleDecToInt(actualWithdrawAmountDec, rewardAssetInfo.RewardAssetInfo.DenominationExponent)
	// update the state of AVS reward asset.
	err = k.UpdateAVSRewardAssetState(ctx, avs, assetID, &feedistributiontypes.DeltaAVSRewardAssetState{
		RewardPoolBalance: actualWithdrawAmountDec.Neg(),
	})
	if err != nil {
		return sdkmath.Int{}, sdkmath.Int{}, rewards, nil, err
	}

	// send the rewards token for dogfood AVS
	// check if the avs is dogfood
	chainIDWithoutRevision := utils.ChainIDWithoutRevision(ctx.ChainID())
	dogfoodAVSAddr := utils.GenerateAVSAddress(chainIDWithoutRevision)
	var withdrawAmountFromDogfood sdkmath.Int
	if dogfoodAVSAddr == avs {
		withdrawAmountFromDogfood = actualWithdrawAmountInt
		if len(imuaReceiptAddr) == 0 {
			return sdkmath.Int{}, sdkmath.Int{}, rewards, nil, feedistributiontypes.ErrInvalidImuaReceiptAddr
		}
		// transfer the imua token to the receipt address
		// truncate reward dec coins, return remainder to community pool
		finalRewards, remainder := subReward.TruncateDecimal()
		err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, feedistributiontypes.ModuleName, imuaReceiptAddr, finalRewards)
		if err != nil {
			return sdkmath.Int{}, sdkmath.Int{}, rewards, nil, err
		}
		if !remainder.IsZero() {
			err = k.UpdateAVSCommunityPool(ctx, avs, true, remainder)
			if err != nil {
				return sdkmath.Int{}, sdkmath.Int{}, rewards, nil, err
			}
		}
	}

	return actualWithdrawAmountInt, withdrawAmountFromDogfood, rewardsAfterSub, subReward, nil
}

// WithdrawStakerRewards withdraws the specified rewards for a staker.
// This function is exposed via a precompile contract interface.
// Only rewards from the "dogfood" AVS are sent directly to the staker's
// receipt address, as the reward vault is managed by this module.
// For rewards from other AVSs, where the reward vaults may reside on different blockchains,
// the gateway contract is responsible for handling subsequent withdrawals from the corresponding vaults.
// This module does not perform actual transfers; it only updates the reward accounting
// and returns the withdrawal amount to the gateway contract for subsequent withdrawals.
func (k Keeper) WithdrawStakerRewards(ctx sdk.Context, stakerID, assetID string,
	amount sdkmath.Int, imuaReceiptAddr sdk.AccAddress,
) (sdkmath.Int, sdkmath.Int, error) {
	if amount.IsNil() || amount.IsNegative() {
		return sdkmath.Int{}, sdkmath.Int{}, feedistributiontypes.ErrInvalidInputParameter.Wrapf(
			"WithdrawStakerRewards, the withdraw amount is nil or negative, amount:%s", amount)
	}
	// withdraw all rewards if the input amount is 0.
	isWithdrawAllReward := false
	if amount.IsZero() {
		isWithdrawAllReward = true
	}

	withdrawAmountPerAVS := sdkmath.NewIntFromBigInt(amount.BigInt())
	actualTotalWithdrawAmount := sdkmath.ZeroInt()
	withdrawAmountFromDogfood := sdkmath.ZeroInt()
	opFunc := func(avs string, rewards *feedistributiontypes.StakerClaimedRewards) (bool, bool, error) {
		if !isWithdrawAllReward && !withdrawAmountPerAVS.IsPositive() {
			// the expected amount has been withdrawn, stop the iteration.
			// the state doesn't need to be updated because there was no change
			// during this iteration.
			return true, false, nil
		}

		var err error
		// withdraw from outstanding rewards
		actualWithdrawAmountInt, amountFromDogfood, endOutstandingRewards, subOutstandingRewards, err := k.generalWithdrawFromAVS(
			ctx, avs, assetID, withdrawAmountPerAVS, imuaReceiptAddr, rewards.OutstandingRewards)
		if err != nil {
			return true, false, err
		} else if len(subOutstandingRewards) != 0 {
			actualTotalWithdrawAmount = actualTotalWithdrawAmount.Add(actualWithdrawAmountInt)
			if !isWithdrawAllReward {
				withdrawAmountPerAVS = withdrawAmountPerAVS.Sub(actualWithdrawAmountInt)
			}

			// Update the input rewards; they will be saved to the KV store if the withdrawal is successful.
			rewards.OutstandingRewards = endOutstandingRewards
			rewards.WithdrawnRewards = rewards.WithdrawnRewards.Add(subOutstandingRewards...)
			if !amountFromDogfood.IsNil() {
				withdrawAmountFromDogfood = withdrawAmountFromDogfood.Add(amountFromDogfood)
			}
		}

		if !isWithdrawAllReward && !withdrawAmountPerAVS.IsPositive() {
			// the expected amount has been withdrawn, return from the current iteration and set the flag
			// to update the state related to outstanding rewards. This check can prevent withdrawing more
			// rewards than the expected amount from the subsequent withdrawable rewards.
			ctx.EventManager().EmitEvent(
				sdk.NewEvent(
					feedistributiontypes.EventTypeWithdrawRewardFromAVS,
					sdk.NewAttribute(feedistributiontypes.AttributeKeyStakerID, stakerID),
					sdk.NewAttribute(feedistributiontypes.AttributeKeyAvsAddress, avs),
					sdk.NewAttribute(feedistributiontypes.AttributeKeyWithdrawDecCoinsFromAVS, subOutstandingRewards.String()),
					sdk.NewAttribute(feedistributiontypes.AttributeKeyStakerOutstandingRewards, endOutstandingRewards.String()),
				),
			)
			return false, true, nil
		}

		// withdraw from withdrawable rewards
		actualWithdrawAmountInt, amountFromDogfood, endWithdrawableRewards, subWithdrawableRewards, err := k.generalWithdrawFromAVS(
			ctx, avs, assetID, withdrawAmountPerAVS, imuaReceiptAddr, rewards.WithdrawableRewards)
		if err != nil {
			return true, false, err
		} else if len(subWithdrawableRewards) != 0 {
			actualTotalWithdrawAmount = actualTotalWithdrawAmount.Add(actualWithdrawAmountInt)
			if !isWithdrawAllReward {
				withdrawAmountPerAVS = withdrawAmountPerAVS.Sub(actualWithdrawAmountInt)
			}

			// Update the input rewards; they will be saved to the KV store if the withdrawal is successful.
			rewards.WithdrawableRewards = endWithdrawableRewards
			rewards.WithdrawnRewards = rewards.WithdrawnRewards.Add(subWithdrawableRewards...)
			if !amountFromDogfood.IsNil() {
				withdrawAmountFromDogfood = withdrawAmountFromDogfood.Add(amountFromDogfood)
			}
		}
		totalSubRewards := subOutstandingRewards.Add(subWithdrawableRewards...)
		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				feedistributiontypes.EventTypeWithdrawRewardFromAVS,
				sdk.NewAttribute(feedistributiontypes.AttributeKeyStakerID, stakerID),
				sdk.NewAttribute(feedistributiontypes.AttributeKeyAvsAddress, avs),
				sdk.NewAttribute(feedistributiontypes.AttributeKeyWithdrawDecCoinsFromAVS, totalSubRewards.String()),
				sdk.NewAttribute(feedistributiontypes.AttributeKeyStakerOutstandingRewards, endOutstandingRewards.String()),
				sdk.NewAttribute(feedistributiontypes.AttributeKeyStakerWithdrawableRewards, endWithdrawableRewards.String()),
			),
		)
		return false, true, nil
	}
	// iterate to withdraw rewards from multiple AVSs, because different AVSs might
	// use the same asset as reward.
	err := k.IterateStakerClaimedRewards(ctx, stakerID, true, opFunc)
	if err != nil {
		return sdkmath.Int{}, sdkmath.Int{}, err
	}
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			feedistributiontypes.EventTypeWithdrawRewards,
			sdk.NewAttribute(feedistributiontypes.AttributeKeyStakerID, stakerID),
			sdk.NewAttribute(feedistributiontypes.AttributeKeyAssetID, assetID),
			sdk.NewAttribute(feedistributiontypes.AttributeKeyTotalWithdrawAmount, actualTotalWithdrawAmount.String()),
			sdk.NewAttribute(feedistributiontypes.AttributeKeyWithdrawAmountFromDogfood, withdrawAmountFromDogfood.String()),
		),
	)
	return actualTotalWithdrawAmount, withdrawAmountFromDogfood, nil
}

func (k Keeper) WithdrawRewardFromDogfood(ctx sdk.Context, stakerID string,
	amount sdkmath.Int, imuaReceiptAddr sdk.AccAddress,
) (sdk.Coins, error) {
	if amount.IsNil() || amount.IsNegative() {
		return nil, feedistributiontypes.ErrInvalidInputParameter.Wrapf(
			"WithdrawRewardFromDogfood, the withdraw amount is nil or negative, amount:%s", amount)
	}
	// withdraw all rewards if the input amount is 0.
	isWithdrawAllReward := false
	if amount.IsZero() {
		isWithdrawAllReward = true
	}

	chainIDWithoutRevision := utils.ChainIDWithoutRevision(ctx.ChainID())
	dogfoodAVSAddr := utils.GenerateAVSAddress(chainIDWithoutRevision)
	stakerClaimedRewards, err := k.GetStakerClaimedRewards(ctx, stakerID, dogfoodAVSAddr)
	if err != nil {
		return nil, err
	}

	expectedWithdrawalAmount := sdkmath.NewIntFromBigInt(amount.BigInt())
	// withdraw from outstanding rewards
	actualWithdrawAmountInt, _, endOutstandingRewards, subOutstandingRewards, err := k.generalWithdrawFromAVS(
		ctx, dogfoodAVSAddr, assetstype.ImuachainAssetID, expectedWithdrawalAmount, imuaReceiptAddr, stakerClaimedRewards.OutstandingRewards)
	if err != nil {
		return nil, err
	}
	stakerClaimedRewards.OutstandingRewards = endOutstandingRewards
	stakerClaimedRewards.WithdrawnRewards = stakerClaimedRewards.WithdrawnRewards.Add(subOutstandingRewards...)

	totalSubRewards := sdk.NewDecCoins(subOutstandingRewards...)
	if isWithdrawAllReward || (!isWithdrawAllReward && actualWithdrawAmountInt.LT(expectedWithdrawalAmount)) {
		if !isWithdrawAllReward {
			expectedWithdrawalAmount = expectedWithdrawalAmount.Sub(actualWithdrawAmountInt)
		}
		// withdraw from withdrawable rewards
		_, _, endWithdrawableRewards, subWithdrawableRewards, err := k.generalWithdrawFromAVS(
			ctx, dogfoodAVSAddr, assetstype.ImuachainAssetID, expectedWithdrawalAmount, imuaReceiptAddr, stakerClaimedRewards.WithdrawableRewards)
		if err != nil {
			return nil, err
		}
		stakerClaimedRewards.WithdrawableRewards = endWithdrawableRewards
		stakerClaimedRewards.WithdrawnRewards = stakerClaimedRewards.WithdrawnRewards.Add(subWithdrawableRewards...)
		totalSubRewards = totalSubRewards.Add(subWithdrawableRewards...)
	}
	err = k.SetStakerClaimedRewards(ctx, stakerID, dogfoodAVSAddr, stakerClaimedRewards)
	if err != nil {
		return nil, err
	}

	subRewardCoins, _ := totalSubRewards.TruncateDecimal()
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			feedistributiontypes.EventTypeWithdrawRewardFromAVS,
			sdk.NewAttribute(feedistributiontypes.AttributeKeyStakerID, stakerID),
			sdk.NewAttribute(feedistributiontypes.AttributeKeyAvsAddress, dogfoodAVSAddr),
			sdk.NewAttribute(feedistributiontypes.AttributeKeyWithdrawDecCoinsFromAVS, totalSubRewards.String()),
			sdk.NewAttribute(feedistributiontypes.AttributeKeyStakerOutstandingRewards, endOutstandingRewards.String()),
			sdk.NewAttribute(feedistributiontypes.AttributeKeyStakerWithdrawableRewards, stakerClaimedRewards.WithdrawableRewards.String()),
		),
	)

	return subRewardCoins, nil
}

// WithdrawOperatorCommission : withdraw operator commission
// It's same as WithdrawStakerRewards, it will also be exposed via precompile.
// So the operators will use their evm addresses to withdraw the commission
// through an evm transaction.
func (k Keeper) WithdrawOperatorCommission(ctx sdk.Context, assetID string,
	amount sdkmath.Int, operator, imuaReceiptAddr sdk.AccAddress,
) (sdkmath.Int, sdkmath.Int, error) {
	// withdraw all commissions if the input amount is 0.
	isWithdrawAllCommission := false
	if amount.IsZero() {
		isWithdrawAllCommission = true
	}
	withdrawAmountPerAVS := amount
	actualTotalWithdrawAmount := sdkmath.ZeroInt()
	withdrawAmountFromDogfood := sdkmath.ZeroInt()
	opFunc := func(avs string, commissions *feedistributiontypes.OperatorCommission) (bool, bool, error) {
		if !isWithdrawAllCommission && !withdrawAmountPerAVS.IsPositive() {
			// the expected amount has been withdrawn, stop the iteration.
			return true, false, nil
		}

		var err error
		actualWithdrawAmountInt, amountFromDogfood, endCommissions, subCommissions, err := k.generalWithdrawFromAVS(
			ctx, avs, assetID, withdrawAmountPerAVS, imuaReceiptAddr, commissions.UnwithdrawnCommission)
		if err != nil {
			return false, false, err
		} else if len(subCommissions) == 0 {
			// withdraw nothing from this AVS, continue iterating the other AVSs
			return false, false, nil
		}

		actualTotalWithdrawAmount = actualTotalWithdrawAmount.Add(actualWithdrawAmountInt)
		if !isWithdrawAllCommission {
			withdrawAmountPerAVS = withdrawAmountPerAVS.Sub(actualWithdrawAmountInt)
		}

		// Update the input commission; they will be saved to the KV store if the withdrawal is successful.
		commissions.UnwithdrawnCommission = endCommissions
		commissions.WithdrawnCommission = commissions.WithdrawnCommission.Add(subCommissions...)
		if !amountFromDogfood.IsNil() {
			withdrawAmountFromDogfood = amountFromDogfood
		}

		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				feedistributiontypes.EventTypeWithdrawCommissionFromAVS,
				sdk.NewAttribute(feedistributiontypes.AttributeKeyOperator, operator.String()),
				sdk.NewAttribute(feedistributiontypes.AttributeKeyAvsAddress, avs),
				sdk.NewAttribute(feedistributiontypes.AttributeKeyWithdrawDecCoinsFromAVS, subCommissions.String()),
			),
		)
		return false, true, nil
	}
	// iterate to withdraw rewards from multiple AVSs, because different AVSs might
	// use the same asset as reward.
	err := k.IterateOperatorCommissions(ctx, operator.String(), true, opFunc)
	if err != nil {
		return sdkmath.Int{}, sdkmath.Int{}, err
	}
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			feedistributiontypes.EventTypeWithdrawCommission,
			sdk.NewAttribute(feedistributiontypes.AttributeKeyOperator, operator.String()),
			sdk.NewAttribute(feedistributiontypes.AttributeKeyAssetID, assetID),
			sdk.NewAttribute(feedistributiontypes.AttributeKeyTotalWithdrawAmount, actualTotalWithdrawAmount.String()),
			sdk.NewAttribute(feedistributiontypes.AttributeKeyWithdrawAmountFromDogfood, withdrawAmountFromDogfood.String()),
		),
	)
	return actualTotalWithdrawAmount, withdrawAmountFromDogfood, nil
}

// WithdrawCommissionFromDogfood : withdraw operator commission from dogfood.
// Unlike WithdrawOperatorCommission, it only withdraws the IMUA commission from the dogfood AVS.
// So it can be provided through a cosmos transaction.
func (k Keeper) WithdrawCommissionFromDogfood(ctx sdk.Context, operator sdk.AccAddress) (sdk.Coins, error) {
	// check if the avs is dogfood
	chainIDWithoutRevision := utils.ChainIDWithoutRevision(ctx.ChainID())
	dogfoodAVSAddr := utils.GenerateAVSAddress(chainIDWithoutRevision)
	operatorCommission, err := k.GetOperatorCommission(ctx, operator.String(), dogfoodAVSAddr)
	if err != nil {
		return nil, err
	}

	// withdraw all commissions
	// use 0 as the input amount to withdraw all commissions.
	_, _, endCommissions, subCommissions, err := k.generalWithdrawFromAVS(
		ctx, dogfoodAVSAddr, assetstype.ImuachainAssetID, sdk.ZeroInt(), operator, operatorCommission.UnwithdrawnCommission)
	if err != nil {
		return nil, err
	}

	operatorCommission.UnwithdrawnCommission = endCommissions
	operatorCommission.WithdrawnCommission = operatorCommission.WithdrawnCommission.Add(subCommissions...)
	err = k.SetOperatorCommission(ctx, operator.String(), dogfoodAVSAddr, operatorCommission)
	if err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			feedistributiontypes.EventTypeWithdrawCommissionFromAVS,
			sdk.NewAttribute(feedistributiontypes.AttributeKeyOperator, operator.String()),
			sdk.NewAttribute(feedistributiontypes.AttributeKeyAvsAddress, dogfoodAVSAddr),
			sdk.NewAttribute(feedistributiontypes.AttributeKeyWithdrawDecCoinsFromAVS, subCommissions.String()),
		),
	)

	subCommissionCoins, _ := subCommissions.TruncateDecimal()
	return subCommissionCoins, nil
}

// WithdrawDelegationRewards is an interface used by the ante handler to withdraw rewards for gas fees.
// This function is disabled because some stakers from other incompatible client chains don't have an address on
// the Imua chain. Additionally, the `IterateDelegations` interface in dogfood also has no actual implementation,
// which means this interface will never be called by the ante handler.
func (k Keeper) WithdrawDelegationRewards(_ sdk.Context, _ sdk.AccAddress, _ sdk.ValAddress) (sdk.Coins, error) {
	return sdk.Coins{}, nil
}
