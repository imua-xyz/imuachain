package keeper

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/imua-xyz/imuachain/x/avs/types"
)

// PayRestakingFee when avs registers, this method is called to deduct the funds in the avs contract account
func (k Keeper) PayRestakingFee(ctx sdk.Context, avsAddr common.Address) error {
	params := k.GetParams(ctx)
	// Calculate the amount payable
	requiredFee := params.BaseRestakingFee

	discount, _ := sdk.NewDecFromStr(params.DiscountedRate.String())

	requiredFee.Amount = sdkmath.LegacyNewDecFromInt(requiredFee.Amount).Mul(sdk.OneDec().Sub(discount)).TruncateInt()

	// Collect a fee
	if err := k.bankKeeper.SendCoinsFromAccountToModule(
		ctx,
		avsAddr[:],
		types.ModuleName,
		sdk.NewCoins(*requiredFee)); err != nil {
		return err
	}

	record := types.AVSPaymentInfo{
		AvsAddress: avsAddr.String(),
		Fee:        requiredFee,
		StartTime:  ctx.BlockTime().Unix(),
		IsPaid:     true,
	}
	k.SetAVSPaymentInfo(ctx, &record)

	return nil
}

func (k Keeper) SetAVSPaymentInfo(ctx sdk.Context, record *types.AVSPaymentInfo) (err error) {
	if !common.IsHexAddress(record.AvsAddress) {
		return types.ErrInvalidAddr
	}
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixAVSPaymentInfo)
	bz := k.cdc.MustMarshal(record)
	store.Set(common.HexToAddress(record.AvsAddress).Bytes(), bz)
	return nil
}

func (k *Keeper) GetAVSPaymentInfo(ctx sdk.Context, avsAddress string) (info *types.AVSPaymentInfo, err error) {
	if !common.IsHexAddress(avsAddress) {
		return nil, types.ErrInvalidAddr
	}
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixAVSPaymentInfo)
	value := store.Get(common.HexToAddress(avsAddress).Bytes())
	if value == nil {
		return nil, errorsmod.Wrap(types.ErrNoKeyInTheStore,
			fmt.Sprintf("GetRestakingRecord: key not found for avs address %s", avsAddress))
	}

	ret := types.AVSPaymentInfo{}
	k.cdc.MustUnmarshal(value, &ret)
	return &ret, nil
}

func (k Keeper) DeleteAVSPaymentInfo(ctx sdk.Context, addr string) error {
	hexAddr := common.HexToAddress(addr)
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixAVSPaymentInfo)
	if !store.Has(hexAddr.Bytes()) {
		return errorsmod.Wrap(types.ErrNoKeyInTheStore, fmt.Sprintf("RestakingRecordInfo didn't exist: key is %s", addr))
	}
	store.Delete(hexAddr[:])
	return nil
}

func (k *Keeper) IsExistAVSPaymentInfo(ctx sdk.Context, addr string) bool {
	hexAddr := common.HexToAddress(addr)
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixAVSPaymentInfo)
	return store.Has(hexAddr.Bytes())
}

// IterateAVSPaymentInfo iterate through payment info
func (k Keeper) IterateAVSPaymentInfo(ctx sdk.Context, fn func(index int64, info types.AVSPaymentInfo) (stop bool)) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixAVSPaymentInfo)

	iterator := sdk.KVStorePrefixIterator(store, nil)
	defer iterator.Close()

	i := int64(0)

	for ; iterator.Valid(); iterator.Next() {
		info := types.AVSPaymentInfo{}
		k.cdc.MustUnmarshal(iterator.Value(), &info)

		stop := fn(i, info)

		if stop {
			break
		}
		i++
	}
}

// GetAllAVSPaymentInfos returns a slice containing all payment information.
func (k *Keeper) GetAllAVSPaymentInfos(ctx sdk.Context) ([]types.AVSPaymentInfo, error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixAVSPaymentInfo)
	iterator := sdk.KVStorePrefixIterator(store, nil)
	defer iterator.Close()

	ret := make([]types.AVSPaymentInfo, 0)
	for ; iterator.Valid(); iterator.Next() {
		task := types.AVSPaymentInfo{}
		err := k.cdc.Unmarshal(iterator.Value(), &task)
		if err != nil {
			return nil, errorsmod.Wrap(err, "GetAllAVSPaymentInfos: failed to unmarshal payment info")
		}
		ret = append(ret, task)
	}
	return ret, nil
}

// InstantUnbonding calculates the penalty and returns the remaining amount to the avs address
func (k Keeper) InstantUnbonding(ctx sdk.Context, avsAddr sdk.AccAddress) error {
	record, err := k.GetAVSPaymentInfo(ctx, avsAddr.String())
	if err != nil {
		return err
	}
	params := k.GetParams(ctx)
	// Calculate the remaining lock time
	elapsed := ctx.BlockTime().Unix() - record.StartTime
	if elapsed < int64(params.WithdrawalPeriod) {
		// CalculatePenalty
		penaltyRate, _ := sdk.NewDecFromStr(params.PenaltyRate.String())
		penaltyAmount := record.Fee.Amount.Mul(sdkmath.Int(penaltyRate))

		// Charge penalty (25%)
		penaltyCoin := sdk.NewCoin(record.Fee.Denom, penaltyAmount)
		if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, avsAddr, types.ModuleName, sdk.NewCoins(penaltyCoin)); err != nil {
			return err
		}

		// Return the remaining amount (75%)
		remaining := record.Fee.Amount.Sub(penaltyAmount)
		remainingCoin := sdk.NewCoin(record.Fee.Denom, remaining)
		if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, avsAddr, sdk.NewCoins(remainingCoin)); err != nil {
			return err
		}
	} else {
		if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, avsAddr, sdk.NewCoins(*record.Fee)); err != nil {
			return err
		}
	}

	return k.DeleteAVSPaymentInfo(ctx, avsAddr.String())
}
