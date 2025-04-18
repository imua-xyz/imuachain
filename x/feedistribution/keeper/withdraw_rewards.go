package keeper

import (
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	assetstype "github.com/imua-xyz/imuachain/x/assets/types"
	avstypes "github.com/imua-xyz/imuachain/x/avs/types"
	feedistributiontypes "github.com/imua-xyz/imuachain/x/feedistribution/types"
)

func (k Keeper) generalWithdrawFromAVS(ctx sdk.Context, avs, assetID string, withdrawAmount sdkmath.Int,
	imuaReceiptAddr sdk.AccAddress, rewards sdk.DecCoins,
) (sdkmath.Int, sdkmath.Int, sdk.DecCoins, error) {
	if withdrawAmount.IsNil() || !withdrawAmount.IsPositive() {
		return sdkmath.Int{}, sdkmath.Int{}, nil, feedistributiontypes.ErrInvalidInputParameter.Wrapf(
			"generalWithdrawFromAVS, the withdraw amount is nil or not positive, amount:%s", withdrawAmount)
	}
	// check and calculate the actual amount withdrawable for an AVS
	rewardAssetInfo, err := k.GetAVSRewardAssetInfo(ctx, avs, assetID)
	if err != nil {
		return sdkmath.Int{}, sdkmath.Int{}, nil, err
	}
	rewardDecimalAmount := rewards.AmountOf(rewardAssetInfo.AssetBasicInfo.Symbol)
	decimalFactor := sdkmath.NewIntWithDecimal(1, int(rewardAssetInfo.AssetBasicInfo.Decimals)) // #nosec G115
	withdrawAmountPerAVSDec := sdk.NewDecFromInt(withdrawAmount).QuoInt(decimalFactor)
	if withdrawAmountPerAVSDec.LT(sdkmath.LegacyZeroDec()) {
		// stop withdrawing the reward
		return sdkmath.Int{}, sdkmath.Int{}, nil,
			feedistributiontypes.ErrInvalidInputParameter.Wrapf("generalWithdrawFromAVS: negative withdraw amount,withdrawAmountPerAVSDec:%s", withdrawAmountPerAVSDec)
	} else if withdrawAmountPerAVSDec.IsZero() {
		// do nothing if the withdraw amount is zero
		return sdkmath.ZeroInt(), sdkmath.ZeroInt(), rewards, nil
	}

	// the actual amount is the minimum of the reward pool balance, the reward amount,
	// and the requested withdraw amount.
	actualWithdrawAmountDec := sdkmath.LegacyMinDec(rewardAssetInfo.RewardAssetState.RewardPoolBalance,
		sdkmath.LegacyMinDec(rewardDecimalAmount, withdrawAmountPerAVSDec))
	// decrease the withdrawing amount from the outstanding reward
	subReward := sdk.DecCoins{
		sdk.NewDecCoinFromDec(rewardAssetInfo.AssetBasicInfo.Symbol, actualWithdrawAmountDec),
	}
	rewardsAfterSub, hasNegative := rewards.SafeSub(subReward)
	if hasNegative {
		return sdkmath.Int{}, sdkmath.Int{}, nil, feedistributiontypes.ErrNegativeCoinAmount.Wrapf("WithdrawStakerRewards: avs:%s, assetID:%s,symbol:%s", avs, assetID, rewardAssetInfo.AssetBasicInfo.Symbol)
	}

	// use TruncateInt to ensure the vault has enough fund
	actualWithdrawAmountInt := actualWithdrawAmountDec.MulInt(decimalFactor).TruncateInt()

	// send the imua token for dogfood AVS
	// check if the avs is dogfood and the reward asset is imua token
	chainIDWithoutRevision := avstypes.ChainIDWithoutRevision(ctx.ChainID())
	dogfoodAVSAddr := avstypes.GenerateAVSAddress(chainIDWithoutRevision)
	var withdrawAmountFromDogfood sdkmath.Int
	if dogfoodAVSAddr == avs && assetID == assetstype.ImuachainAssetID {
		withdrawAmountFromDogfood = actualWithdrawAmountInt
		if len(imuaReceiptAddr) == 0 {
			return sdkmath.Int{}, sdkmath.Int{}, nil, feedistributiontypes.ErrInvalidImuaReceiptAddr
		}
		// transfer the imua token to the receipt address
		// truncate reward dec coins, return remainder to community pool
		finalRewards, remainder := subReward.TruncateDecimal()
		err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, feedistributiontypes.ModuleName, imuaReceiptAddr, finalRewards)
		if err != nil {
			return sdkmath.Int{}, sdkmath.Int{}, nil, err
		}
		err = k.UpdateAVSCommunityPool(ctx, avs, true, remainder)
		if err != nil {
			return sdkmath.Int{}, sdkmath.Int{}, nil, err
		}
	}

	return actualWithdrawAmountInt, withdrawAmountFromDogfood, rewardsAfterSub, nil
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
	withdrawAmountPerAVS := amount
	actualTotalWithdrawAmount := sdkmath.ZeroInt()
	withdrawAmountFromDogfood := sdkmath.ZeroInt()
	allAVSActualWithdrawAmounts := feedistributiontypes.AllAVSActualWithdrawAmount(
		make([]feedistributiontypes.ActualWithdrawAmountPerAVS, 0))
	opFunc := func(avs string, rewards *feedistributiontypes.StakerOutstandingRewards) (bool, error) {
		var err error
		actualWithdrawAmountInt, amountFromDogfood, endRewards, err := k.generalWithdrawFromAVS(
			ctx, avs, assetID, withdrawAmountPerAVS, imuaReceiptAddr, rewards.Rewards)
		if err != nil {
			return false, err
		}
		actualTotalWithdrawAmount = actualTotalWithdrawAmount.Add(actualWithdrawAmountInt)
		withdrawAmountPerAVS = withdrawAmountPerAVS.Sub(actualWithdrawAmountInt)
		allAVSActualWithdrawAmounts = append(allAVSActualWithdrawAmounts, feedistributiontypes.ActualWithdrawAmountPerAVS{
			Avs:                  avs,
			ActualWithdrawAmount: actualTotalWithdrawAmount,
		})
		// Update the input rewards; they will be saved to the KV store if the withdrawal is successful.
		rewards.Rewards = endRewards
		if !amountFromDogfood.IsNil() {
			withdrawAmountFromDogfood = amountFromDogfood
		}
		return false, nil
	}
	// iterate to withdraw rewards from multiple AVSs, because different AVSs might
	// use the same asset as reward.
	err := k.IterateStakerOutstandingRewards(ctx, stakerID, true, opFunc)
	if err != nil {
		return sdkmath.Int{}, sdkmath.Int{}, err
	}
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			feedistributiontypes.EventTypeWithdrawRewards,
			sdk.NewAttribute(feedistributiontypes.AttributeKeyStakerID, stakerID),
			sdk.NewAttribute(feedistributiontypes.AttributeKeyAssetID, assetID),
			sdk.NewAttribute(feedistributiontypes.AttributeKeyAllAVSActualWithdrawAmounts, allAVSActualWithdrawAmounts.String()),
			sdk.NewAttribute(feedistributiontypes.AttributeKeyTotalWithdrawAmount, actualTotalWithdrawAmount.String()),
			sdk.NewAttribute(feedistributiontypes.AttributeKeyWithdrawAmountFromDogfood, withdrawAmountFromDogfood.String()),
		),
	)
	return actualTotalWithdrawAmount, withdrawAmountFromDogfood, nil
}

// WithdrawOperatorCommission : withdraw operator commission
// It's same as WithdrawStakerRewards, it will also be exposed via precompile.
// So the operators will use their evm addresses to withdraw the commission
// through an evm transaction.
func (k Keeper) WithdrawOperatorCommission(ctx sdk.Context, assetID string,
	amount sdkmath.Int, operator, imuaReceiptAddr sdk.AccAddress,
) (sdkmath.Int, sdkmath.Int, error) {
	withdrawAmountPerAVS := amount
	actualTotalWithdrawAmount := sdkmath.ZeroInt()
	withdrawAmountFromDogfood := sdkmath.ZeroInt()
	allAVSActualWithdrawAmounts := feedistributiontypes.AllAVSActualWithdrawAmount(
		make([]feedistributiontypes.ActualWithdrawAmountPerAVS, 0))
	opFunc := func(avs string, commissions *feedistributiontypes.OperatorAccumulatedCommission) (bool, error) {
		var err error
		actualWithdrawAmountInt, amountFromDogfood, endCommissions, err := k.generalWithdrawFromAVS(
			ctx, avs, assetID, withdrawAmountPerAVS, imuaReceiptAddr, commissions.Commission)
		if err != nil {
			return false, err
		}
		actualTotalWithdrawAmount = actualTotalWithdrawAmount.Add(actualWithdrawAmountInt)
		withdrawAmountPerAVS = withdrawAmountPerAVS.Sub(actualWithdrawAmountInt)
		allAVSActualWithdrawAmounts = append(allAVSActualWithdrawAmounts, feedistributiontypes.ActualWithdrawAmountPerAVS{
			Avs:                  avs,
			ActualWithdrawAmount: actualTotalWithdrawAmount,
		})
		// Update the input commission; they will be saved to the KV store if the withdrawal is successful.
		commissions.Commission = endCommissions
		if !amountFromDogfood.IsNil() {
			withdrawAmountFromDogfood = amountFromDogfood
		}
		return false, nil
	}
	// iterate to withdraw rewards from multiple AVSs, because different AVSs might
	// use the same asset as reward.
	err := k.IterateOperatorAccumulatedCommissions(ctx, operator.String(), true, opFunc)
	if err != nil {
		return sdkmath.Int{}, sdkmath.Int{}, err
	}
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			feedistributiontypes.EventTypeWithdrawCommission,
			sdk.NewAttribute(feedistributiontypes.AttributeKeyOperator, operator.String()),
			sdk.NewAttribute(feedistributiontypes.AttributeKeyAssetID, assetID),
			sdk.NewAttribute(feedistributiontypes.AttributeKeyAllAVSActualWithdrawAmounts, allAVSActualWithdrawAmounts.String()),
			sdk.NewAttribute(feedistributiontypes.AttributeKeyTotalWithdrawAmount, actualTotalWithdrawAmount.String()),
			sdk.NewAttribute(feedistributiontypes.AttributeKeyWithdrawAmountFromDogfood, withdrawAmountFromDogfood.String()),
		),
	)
	return actualTotalWithdrawAmount, withdrawAmountFromDogfood, nil
}

// WithdrawDelegationRewards is an interface used by the ante handler to withdraw rewards for gas fees.
// This function is disabled because some stakers from other incompatible client chains don't have an address on
// the Imua chain. Additionally, the `IterateDelegations` interface in dogfood also has no actual implementation,
// which means this interface will never be called by the ante handler.
func (k Keeper) WithdrawDelegationRewards(_ sdk.Context, _ sdk.AccAddress, _ sdk.ValAddress) (sdk.Coins, error) {
	return sdk.Coins{}, nil
}
