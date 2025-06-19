package keeper

import (
	"errors"
	"fmt"
	"strings"

	"github.com/imua-xyz/imuachain/utils"

	assetstype "github.com/imua-xyz/imuachain/x/assets/types"
	delegationkeeper "github.com/imua-xyz/imuachain/x/delegation/keeper"
	delegationtype "github.com/imua-xyz/imuachain/x/delegation/types"
	oracletype "github.com/imua-xyz/imuachain/x/oracle/types"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	operatortypes "github.com/imua-xyz/imuachain/x/operator/types"
)

// UpdateOperatorUSDValue is a function to update the USD share for specified operator and Avs,
// The key and value that will be changed is:
// AVSAddr + '/' + operatorAddr -> types.OperatorOptedUSDValue (the total USD share of specified operator and Avs)
// This function will be called when some assets supported by Avs are delegated/undelegated or slashed.
// Currently this function is only called during tests.
func (k *Keeper) UpdateOperatorUSDValue(ctx sdk.Context, avsAddr, operatorAddr string, delta operatortypes.DeltaOperatorUSDInfo) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), operatortypes.KeyPrefixUSDValueForOperator)
	var key []byte
	if operatorAddr == "" {
		return errorsmod.Wrap(operatortypes.ErrParameterInvalid, "UpdateOperatorUSDValue the operatorAddr is empty")
	}
	key = assetstype.GetJoinedStoreKey(strings.ToLower(avsAddr), operatorAddr)

	usdInfo := operatortypes.OperatorOptedUSDValue{
		SelfUSDValue:   sdkmath.LegacyZeroDec(),
		TotalUSDValue:  sdkmath.LegacyZeroDec(),
		ActiveUSDValue: sdkmath.LegacyZeroDec(),
	}
	value := store.Get(key)
	if value != nil {
		k.cdc.MustUnmarshal(value, &usdInfo)
	}

	err := assetstype.UpdateAssetDecValue(&usdInfo.SelfUSDValue, &delta.SelfUSDValue)
	if err != nil {
		return err
	}
	err = assetstype.UpdateAssetDecValue(&usdInfo.TotalUSDValue, &delta.TotalUSDValue)
	if err != nil {
		return err
	}
	err = assetstype.UpdateAssetDecValue(&usdInfo.ActiveUSDValue, &delta.ActiveUSDValue)
	if err != nil {
		return err
	}
	bz := k.cdc.MustMarshal(&usdInfo)
	store.Set(key, bz)
	// emit an event even though this is only used for testing right now
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			operatortypes.EventTypeUpdateOperatorUSDValue,
			sdk.NewAttribute(operatortypes.AttributeKeyOperator, operatorAddr),
			sdk.NewAttribute(operatortypes.AttributeKeyAVSAddr, avsAddr),
			sdk.NewAttribute(operatortypes.AttributeKeySelfUSDValue, usdInfo.SelfUSDValue.String()),
			sdk.NewAttribute(operatortypes.AttributeKeyTotalUSDValue, usdInfo.TotalUSDValue.String()),
			sdk.NewAttribute(operatortypes.AttributeKeyActiveUSDValue, usdInfo.ActiveUSDValue.String()),
		),
	)
	return nil
}

func (k *Keeper) InitOperatorUSDValue(ctx sdk.Context, avsAddr, operatorAddr string) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), operatortypes.KeyPrefixUSDValueForOperator)
	var key []byte
	if operatorAddr == "" {
		return errorsmod.Wrap(operatortypes.ErrParameterInvalid, "InitOperatorUSDValue the operatorAddr is empty")
	}
	key = assetstype.GetJoinedStoreKey(strings.ToLower(avsAddr), operatorAddr)
	if store.Has(key) {
		// The operator's USD value won’t be deleted immediately when opting out,
		// so just return nil if it has already been initialized.
		return nil
	}
	initValue := operatortypes.OperatorOptedUSDValue{
		SelfUSDValue:   sdkmath.LegacyZeroDec(),
		TotalUSDValue:  sdkmath.LegacyZeroDec(),
		ActiveUSDValue: sdkmath.LegacyZeroDec(),
	}
	bz := k.cdc.MustMarshal(&initValue)
	store.Set(key, bz)
	// no need to emit event here because DEFAULT 0 in indexer
	return nil
}

// DeleteOperatorUSDValues is a function to delete the USD share related to some operators and Avs,
// The key and value that will be deleted is:
// AVSAddr + '/' + operatorAddr -> types.OperatorOptedUSDValue (the total USD share of specified operator and Avs)
// This function is called when handling opted-out operators during the voting power update, as the USD share
// does not need to be stored.
func (k *Keeper) DeleteOperatorUSDValues(ctx sdk.Context, avsAddr string, operators []string) error {
	if len(operators) == 0 {
		return nil
	}
	store := prefix.NewStore(ctx.KVStore(k.storeKey), operatortypes.KeyPrefixUSDValueForOperator)
	var key []byte
	operatorEvents := ""
	for _, operatorAddr := range operators {
		if operatorAddr == "" {
			return errorsmod.Wrap(operatortypes.ErrParameterInvalid, "DeleteOperatorUSDValue the operatorAddr is empty")
		}
		key = assetstype.GetJoinedStoreKey(strings.ToLower(avsAddr), operatorAddr)
		store.Delete(key)
		operatorEvents += fmt.Sprintf("%v,", operatorAddr)
	}
	if operatorEvents != "" {
		operatorEvents = operatorEvents[:len(operatorEvents)-1]
	}
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			operatortypes.EventTypeDeleteOperatorUSDValues,
			sdk.NewAttribute(operatortypes.AttributeKeyOperators, operatorEvents),
			sdk.NewAttribute(operatortypes.AttributeKeyAVSAddr, avsAddr),
		),
	)
	return nil
}

func (k *Keeper) DeleteAllOperatorsUSDValueForAVS(ctx sdk.Context, avsAddr string) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), operatortypes.KeyPrefixUSDValueForOperator)
	iterator := sdk.KVStorePrefixIterator(store, operatortypes.IterateOperatorsForAVSPrefix(strings.ToLower(avsAddr)))
	defer iterator.Close()

	hasOperator := false
	operatorEvents := ""
	for ; iterator.Valid(); iterator.Next() {
		parsed, err := assetstype.ParseJoinedStoreKey(iterator.Key(), 2)
		if err != nil {
			return err
		}
		store.Delete(iterator.Key())
		operatorEvents += fmt.Sprintf("%v,", parsed[1])
		hasOperator = true
	}
	if !hasOperator {
		return nil
	}
	if operatorEvents != "" {
		operatorEvents = operatorEvents[:len(operatorEvents)-1]
	}
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			operatortypes.EventTypeDeleteOperatorUSDValues,
			sdk.NewAttribute(operatortypes.AttributeKeyOperators, operatorEvents),
			sdk.NewAttribute(operatortypes.AttributeKeyAVSAddr, avsAddr),
		),
	)
	return nil
}

// GetOperatorOptedUSDValue is a function to retrieve the USD share of specified operator and Avs,
// The key and value to retrieve is:
// AVSAddr + '/' + operatorAddr -> types.OperatorOptedUSDValue (the total USD share of specified operator and Avs)
// This function will be called when the operator opts out of the AVS, because the total USD share
// of Avs should decrease the USD share of the opted-out operator
// This function can also serve as an RPC in the future.
func (k *Keeper) GetOperatorOptedUSDValue(ctx sdk.Context, avsAddr, operatorAddr string) (operatortypes.OperatorOptedUSDValue, error) {
	// return zero if the operator has opted-out of the AVS effectively
	if k.IsOptedOutAndEffective(ctx, operatorAddr, avsAddr) {
		return operatortypes.OperatorOptedUSDValue{
			SelfUSDValue:   sdkmath.LegacyZeroDec(),
			TotalUSDValue:  sdkmath.LegacyZeroDec(),
			ActiveUSDValue: sdkmath.LegacyZeroDec(),
		}, nil
	}

	store := prefix.NewStore(ctx.KVStore(k.storeKey), operatortypes.KeyPrefixUSDValueForOperator)
	var ret operatortypes.OperatorOptedUSDValue
	var key []byte
	if operatorAddr == "" {
		return operatortypes.OperatorOptedUSDValue{}, errorsmod.Wrap(operatortypes.ErrParameterInvalid, "GetOperatorOptedUSDValue the operatorAddr is empty")
	}
	key = assetstype.GetJoinedStoreKey(strings.ToLower(avsAddr), operatorAddr)
	value := store.Get(key)
	if value == nil {
		return operatortypes.OperatorOptedUSDValue{}, errorsmod.Wrap(operatortypes.ErrNoKeyInTheStore, fmt.Sprintf("GetOperatorOptedUSDValue: key is %s", key))
	}
	k.cdc.MustUnmarshal(value, &ret)

	return ret, nil
}

// UpdateAVSUSDValue is a function to update the total USD share of an Avs,
// The key and value that will be changed is:
// AVSAddr -> types.DecValueField（the total USD share of specified Avs）
// This function will be called when some assets of operator supported by the specified Avs
// are delegated/undelegated or slashed. Additionally, when an operator opts out of
// the Avs, this function also will be called.
// Currently not used.
func (k *Keeper) UpdateAVSUSDValue(ctx sdk.Context, avsAddr string, opAmount sdkmath.LegacyDec) error {
	if opAmount.IsNil() || opAmount.IsZero() {
		return errorsmod.Wrap(operatortypes.ErrValueIsNilOrZero, fmt.Sprintf("UpdateAVSUSDValue the opAmount is:%v", opAmount))
	}
	store := prefix.NewStore(ctx.KVStore(k.storeKey), operatortypes.KeyPrefixUSDValueForAVS)
	key := []byte(strings.ToLower(avsAddr))
	totalValue := operatortypes.DecValueField{Amount: sdkmath.LegacyZeroDec()}
	value := store.Get(key)
	if value != nil {
		k.cdc.MustUnmarshal(value, &totalValue)
	}

	err := assetstype.UpdateAssetDecValue(&totalValue.Amount, &opAmount)
	if err != nil {
		return err
	}
	bz := k.cdc.MustMarshal(&totalValue)
	store.Set(key, bz)
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			operatortypes.EventTypeUpdateAVSUSDValue,
			sdk.NewAttribute(operatortypes.AttributeKeyAVSAddr, avsAddr),
			sdk.NewAttribute(operatortypes.AttributeKeyTotalUSDValue, totalValue.Amount.String()),
		),
	)
	return nil
}

// SetAVSUSDValue is a function to set the total USD share of an Avs,
func (k *Keeper) SetAVSUSDValue(ctx sdk.Context, avsAddr string, amount sdkmath.LegacyDec) error {
	if amount.IsNil() {
		return errorsmod.Wrap(operatortypes.ErrValueIsNilOrZero, fmt.Sprintf("SetAVSUSDValue the amount is:%v", amount))
	}
	store := prefix.NewStore(ctx.KVStore(k.storeKey), operatortypes.KeyPrefixUSDValueForAVS)
	key := []byte(strings.ToLower(avsAddr))
	setValue := operatortypes.DecValueField{Amount: amount}
	bz := k.cdc.MustMarshal(&setValue)
	store.Set(key, bz)
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			operatortypes.EventTypeUpdateAVSUSDValue,
			sdk.NewAttribute(operatortypes.AttributeKeyAVSAddr, avsAddr),
			sdk.NewAttribute(operatortypes.AttributeKeyTotalUSDValue, amount.String()),
		),
	)
	return nil
}

func (k *Keeper) DeleteAVSUSDValue(ctx sdk.Context, avsAddr string) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), operatortypes.KeyPrefixUSDValueForAVS)
	key := []byte(strings.ToLower(avsAddr))
	store.Delete(key)
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			operatortypes.EventTypeDeleteAVSUSDValue,
			sdk.NewAttribute(operatortypes.AttributeKeyAVSAddr, avsAddr),
		),
	)
	return nil
}

// GetAVSUSDValue is a function to retrieve the USD share of specified Avs,
// The key and value to retrieve is:
// AVSAddr -> types.DecValueField（the total USD share of specified Avs）
func (k *Keeper) GetAVSUSDValue(ctx sdk.Context, avsAddr string) (sdkmath.LegacyDec, error) {
	store := prefix.NewStore(
		ctx.KVStore(k.storeKey),
		operatortypes.KeyPrefixUSDValueForAVS,
	)
	var ret operatortypes.DecValueField
	key := []byte(strings.ToLower(avsAddr))
	value := store.Get(key)
	if value == nil {
		return sdkmath.LegacyDec{}, errorsmod.Wrap(operatortypes.ErrNoKeyInTheStore, fmt.Sprintf("GetAVSUSDValue: key is %s", key))
	}
	k.cdc.MustUnmarshal(value, &ret)

	return ret.Amount, nil
}

// IterateOperatorUSDValuesForAVS is used to iterate the operator USD values of a specified AVS and
// do some external operations.
// `isUpdate` is a flag to indicate whether the change of the state should be set to the store.
func (k *Keeper) IterateOperatorUSDValuesForAVS(ctx sdk.Context, avsAddr string, isUpdate bool, opFunc func(operator string, optedUSDValues *operatortypes.OperatorOptedUSDValue) error) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), operatortypes.KeyPrefixUSDValueForOperator)
	iterator := sdk.KVStorePrefixIterator(store, operatortypes.IterateOperatorsForAVSPrefix(strings.ToLower(avsAddr)))
	defer iterator.Close()

	updatedKeyValues := make([]utils.KeyValue, 0)
	updatedOperators := make([]string, 0)
	for ; iterator.Valid(); iterator.Next() {
		keys, err := assetstype.ParseJoinedKey(iterator.Key())
		if err != nil {
			return err
		}
		var optedUSDValues operatortypes.OperatorOptedUSDValue
		k.cdc.MustUnmarshal(iterator.Value(), &optedUSDValues)
		err = opFunc(keys[1], &optedUSDValues)
		if err != nil {
			return err
		}
		if isUpdate {
			updatedKeyValues = append(updatedKeyValues, utils.KeyValue{
				Key:   append([]byte(nil), iterator.Key()...),
				Value: &optedUSDValues,
			})
			updatedOperators = append(updatedOperators, keys[1])
		}
	}

	for i, updatedKeyValue := range updatedKeyValues {
		bz := k.cdc.MustMarshal(updatedKeyValue.Value)
		store.Set(updatedKeyValue.Key, bz)
		optedUSDValues := updatedKeyValue.Value.(*operatortypes.OperatorOptedUSDValue)
		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				operatortypes.EventTypeUpdateOperatorUSDValue,
				sdk.NewAttribute(operatortypes.AttributeKeyOperator, updatedOperators[i]),
				sdk.NewAttribute(operatortypes.AttributeKeyAVSAddr, avsAddr),
				sdk.NewAttribute(operatortypes.AttributeKeySelfUSDValue, optedUSDValues.SelfUSDValue.String()),
				sdk.NewAttribute(operatortypes.AttributeKeyTotalUSDValue, optedUSDValues.TotalUSDValue.String()),
				sdk.NewAttribute(operatortypes.AttributeKeyActiveUSDValue, optedUSDValues.ActiveUSDValue.String()),
			),
		)
	}
	return nil
}

func (k Keeper) GetVotePowerForChainID(
	ctx sdk.Context, operators []sdk.AccAddress, chainIDWithoutRevision string,
) ([]int64, error) {
	isAVS, avsAddrString := k.avsKeeper.IsAVSByChainID(ctx, chainIDWithoutRevision)
	if !isAVS {
		return nil, operatortypes.ErrUnknownChainID.Wrapf(
			"GetVotePowerForChainID: chainIDWithoutRevision is %s", chainIDWithoutRevision,
		)
	}
	ret := make([]int64, 0)
	for _, operator := range operators {
		// this already filters by the required assetIDs
		optedUSDValues, err := k.GetOperatorOptedUSDValue(ctx, avsAddrString, operator.String())
		if err != nil {
			return nil, err
		}
		// truncate the USD value to int64, so if the usd value is smaller than 1U,
		// the returned value is 0.
		ret = append(ret, optedUSDValues.ActiveUSDValue.TruncateInt64())
	}
	return ret, nil
}

func (k *Keeper) GetOperatorAssetValue(ctx sdk.Context, operator sdk.AccAddress, chainIDWithoutRevision string) (int64, error) {
	isAVS, avsAddr := k.avsKeeper.IsAVSByChainID(ctx, chainIDWithoutRevision)
	if !isAVS {
		return 0, errorsmod.Wrap(operatortypes.ErrUnknownChainID, fmt.Sprintf("GetOperatorAssetValue: chainIDWithoutRevision is %s", chainIDWithoutRevision))
	}
	optedUSDValues, err := k.GetOperatorOptedUSDValue(ctx, operator.String(), avsAddr)
	if err != nil {
		return 0, err
	}
	// truncate the USD value to int64
	return optedUSDValues.ActiveUSDValue.TruncateInt64(), nil
}

func (k *Keeper) SetAllOperatorUSDValues(ctx sdk.Context, usdValues []operatortypes.OperatorUSDValue) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), operatortypes.KeyPrefixUSDValueForOperator)
	for i := range usdValues {
		usdValue := usdValues[i]
		bz := k.cdc.MustMarshal(&usdValue.OptedUSDValue)
		store.Set([]byte(usdValue.Key), bz)
	}
	return nil
}

func (k *Keeper) GetAllOperatorUSDValues(ctx sdk.Context) ([]operatortypes.OperatorUSDValue, error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), operatortypes.KeyPrefixUSDValueForOperator)
	iterator := sdk.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	ret := make([]operatortypes.OperatorUSDValue, 0)
	for ; iterator.Valid(); iterator.Next() {
		var usdValues operatortypes.OperatorOptedUSDValue
		k.cdc.MustUnmarshal(iterator.Value(), &usdValues)
		ret = append(ret, operatortypes.OperatorUSDValue{
			Key:           string(iterator.Key()),
			OptedUSDValue: usdValues,
		})
	}
	return ret, nil
}

func (k *Keeper) SetAllAVSUSDValues(ctx sdk.Context, usdValues []operatortypes.AVSUSDValue) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), operatortypes.KeyPrefixUSDValueForAVS)
	for i := range usdValues {
		usdValue := usdValues[i]
		bz := k.cdc.MustMarshal(&usdValue.Value)
		store.Set([]byte(strings.ToLower(usdValue.AVSAddr)), bz)
	}
	return nil
}

func (k *Keeper) IterateAVSUSDValues(ctx sdk.Context,
	opFunc func(avsAddr string, avsUSDValue *operatortypes.DecValueField) error,
) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), operatortypes.KeyPrefixUSDValueForAVS)
	iterator := sdk.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var usdValue operatortypes.DecValueField
		k.cdc.MustUnmarshal(iterator.Value(), &usdValue)
		err := opFunc(string(iterator.Key()), &usdValue)
		if err != nil {
			return err
		}
	}
	return nil
}

func (k *Keeper) GetAllAVSUSDValues(ctx sdk.Context) ([]operatortypes.AVSUSDValue, error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), operatortypes.KeyPrefixUSDValueForAVS)
	iterator := sdk.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	ret := make([]operatortypes.AVSUSDValue, 0)
	opFunc := func(avsAddr string, avsUSDValue *operatortypes.DecValueField) error {
		ret = append(ret, operatortypes.AVSUSDValue{
			AVSAddr: avsAddr,
			Value:   *avsUSDValue,
		})
		return nil
	}
	err := k.IterateAVSUSDValues(ctx, opFunc)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// CalculateRealTimeOperatorUSDValue calculates the total and self usd value for the
// operator according to the input assets filter and prices.
// This function will be used in slashing calculations and real time USD value calculation.
// The inputs/outputs and calculation logic for these two cases are different,
// so an `isForSlash` flag is used to distinguish between them.
// When it's called for real time USD value calculation, the needed outputs are the current total
// staking amount and the self-staking amount of the operator. The current total
// staking amount excludes the pending unbonding amount, so it's used to calculate the voting power.
// The self-staking amount is also needed to check if the operator's self-staking is sufficient.
// At the same time, the prices of all assets have been retrieved in the caller's function, so they
// are inputted as a parameter.
// When it's called by the slash execution, the needed output is the sum of the current total amount and
// the pending unbonding amount, because the undelegation also needs to be slashed. And the prices of
// all assets haven't been prepared by the caller, so the prices should be retrieved in this function.
func (k *Keeper) CalculateRealTimeOperatorUSDValue(
	ctx sdk.Context,
	isForSlash bool,
	operator string,
	assetsFilter map[string]interface{},
	decimals map[string]uint32,
	prices map[string]oracletype.Price,
) (operatortypes.OperatorStakingInfo, error) {
	var err error
	ret := operatortypes.OperatorStakingInfo{
		Staking:                 sdkmath.LegacyZeroDec(),
		SelfStaking:             sdkmath.LegacyZeroDec(),
		StakingAndWaitUnbonding: sdkmath.LegacyZeroDec(),
	}
	// iterate all assets owned by the operator to calculate its voting power
	opFuncToIterateAssets := func(assetID string, state *assetstype.OperatorAssetInfo) error {
		var price oracletype.Price
		var decimal uint32
		if isForSlash {
			// when calculated the USD value for slashing, the input prices map is null
			// so the price needs to be retrieved here
			price, err = k.oracleKeeper.GetSpecifiedAssetsPrice(ctx, assetID)
			if err != nil {
				// TODO: when assetID is not registered in oracle module, this error will finally lead to panic
				if !errors.Is(err, oracletype.ErrGetPriceRoundNotFound) {
					return err
				}
				// TODO: for now, we ignore the error when the price round is not found and set the price to 1 to avoid panic
			}
			assetInfo, err := k.assetsKeeper.GetStakingAssetInfo(ctx, assetID)
			if err != nil {
				return err
			}
			decimal = assetInfo.AssetBasicInfo.Decimals
			usdValue := CalculateUSDValue(state.TotalAmount.Add(state.PendingUndelegationAmount), price.Value, decimal, price.Decimal)
			ctx.Logger().Info("CalculateRealTimeOperatorUSDValue: get price for slash", "assetID", assetID, "assetDecimal", decimal, "price", price, "totalAmount", state.TotalAmount, "pendingUndelegationAmount", state.PendingUndelegationAmount, "StakingAndWaitUnbonding", ret.StakingAndWaitUnbonding, "addUSDValue", usdValue)
			ret.StakingAndWaitUnbonding.AddMut(usdValue)
		} else {
			if prices == nil {
				return errorsmod.Wrap(operatortypes.ErrValueIsNilOrZero, "CalculateRealTimeOperatorUSDValue prices map is nil")
			}
			price, ok := prices[assetID]
			if !ok {
				return errorsmod.Wrap(operatortypes.ErrKeyNotExistInMap, "CalculateRealTimeOperatorUSDValue map: prices, key: assetID")
			}
			decimal, ok := decimals[assetID]
			if !ok {
				return errorsmod.Wrap(operatortypes.ErrKeyNotExistInMap, "CalculateRealTimeOperatorUSDValue map: decimals, key: assetID")
			}
			ret.Staking.AddMut(CalculateUSDValue(state.TotalAmount, price.Value, decimal, price.Decimal))
			// calculate the token amount from the share for the operator
			selfAmount, err := delegationkeeper.TokensFromShares(state.OperatorShare, state.TotalShare, state.TotalAmount)
			if err != nil {
				return err
			}
			ret.SelfStaking.AddMut(CalculateUSDValue(selfAmount, price.Value, decimal, price.Decimal))
		}
		return nil
	}
	err = k.assetsKeeper.IterateAssetsForOperator(ctx, false, operator, assetsFilter, opFuncToIterateAssets)
	if err != nil {
		return ret, err
	}
	return ret, nil
}

// AggregateOperatorUSDValue computes the total USD value by summing the USD values of all assets.
// Unlike `CalculateRealTimeOperatorUSDValue`, this function may not reflect real-time values,
// as asset USD values are updated only at the end of an epoch or when a slash event occurs.
// It will be used for the voting power update.
func (k *Keeper) AggregateOperatorUSDValue(
	ctx sdk.Context,
	epochIdentifier, operator string,
	assetsList []string,
) (operatortypes.OperatorStakingInfo, error) {
	operatorAccAddr, err := sdk.AccAddressFromBech32(operator)
	if err != nil {
		return operatortypes.OperatorStakingInfo{}, delegationtype.ErrOperatorAddrIsNotAccAddr
	}
	ret := operatortypes.OperatorStakingInfo{
		Staking:                 sdkmath.LegacyZeroDec(),
		SelfStaking:             sdkmath.LegacyZeroDec(),
		StakingAndWaitUnbonding: sdkmath.LegacyZeroDec(),
	}
	for _, assetID := range assetsList {
		// get the total USD value of asset
		totalUSDValue, err := k.GetOperatorAssetUSDValue(ctx, epochIdentifier, operator, assetID)
		if err != nil {
			ctx.Logger().Error("AggregateOperatorUSDValue: failed to get the operator asset USD value", "err", err, "epochIdentifier", epochIdentifier, "operator", operator, "assetID", assetID)
			// continue handling the other assets
			continue
		}
		// calculate the self staking USD value for the operator
		operatorAssetInfo, err := k.assetsKeeper.GetOperatorSpecifiedAssetInfo(ctx, operatorAccAddr, assetID)
		if err != nil {
			ctx.Logger().Error("AggregateOperatorUSDValue: failed to get the operator asset info", "err", err, "epochIdentifier", epochIdentifier, "operator", operator, "assetID", assetID)
			// continue handling the other assets
			continue
		}
		ret.Staking.AddMut(totalUSDValue)
		selfAmount, err := delegationkeeper.TokenUSDValueFromShares(operatorAssetInfo.OperatorShare, operatorAssetInfo.TotalShare, totalUSDValue)
		if err != nil {
			ctx.Logger().Error("AggregateOperatorUSDValue: failed to calculate the self staking USD value for operator", "err", err, "epochIdentifier", epochIdentifier, "operator", operator, "assetID", assetID)
			// continue handling the other assets
			continue
		}
		ret.SelfStaking.AddMut(selfAmount)
	}
	return ret, nil
}

func (k Keeper) GetOrCalculateOperatorUSDValues(
	ctx sdk.Context,
	operator sdk.AccAddress,
	avsAddr string,
) (optedUSDValues operatortypes.OperatorOptedUSDValue, err error) {
	// the usd values will be deleted if the operator opts out effectively, so recalculate the
	// voting power to set the tokens and shares for this case.
	if k.IsOptedOutAndEffective(ctx, operator.String(), avsAddr) {
		// get assets supported by the AVS
		assets, err := k.avsKeeper.GetAVSSupportedAssets(ctx, avsAddr)
		if err != nil {
			return operatortypes.OperatorOptedUSDValue{}, err
		}
		if assets == nil {
			return operatortypes.OperatorOptedUSDValue{}, err
		}
		// get the prices and decimals of assets
		decimals, err := k.assetsKeeper.GetAssetsDecimal(ctx, assets)
		if err != nil {
			return operatortypes.OperatorOptedUSDValue{}, err
		}
		prices, err := k.oracleKeeper.GetMultipleAssetsPrices(ctx, assets)
		if err != nil {
			return operatortypes.OperatorOptedUSDValue{}, err
		}
		stakingInfo, err := k.CalculateRealTimeOperatorUSDValue(ctx, false, operator.String(), assets, decimals, prices)
		if err != nil {
			return operatortypes.OperatorOptedUSDValue{}, err
		}
		optedUSDValues.SelfUSDValue = stakingInfo.SelfStaking
		optedUSDValues.TotalUSDValue = stakingInfo.Staking
	} else {
		optedUSDValues, err = k.GetOperatorOptedUSDValue(ctx, avsAddr, operator.String())
		if err != nil {
			return operatortypes.OperatorOptedUSDValue{}, err
		}
	}
	return optedUSDValues, nil
}

func (k *Keeper) CalculateUSDValueForStaker(ctx sdk.Context, stakerID, avsAddr string, operator sdk.AccAddress) (sdkmath.LegacyDec, error) {
	if !k.IsOptedInAndNotJailed(ctx, operator.String(), avsAddr) {
		return sdkmath.LegacyZeroDec(), nil
	}
	optedUSDValues, err := k.GetOperatorOptedUSDValue(ctx, avsAddr, operator.String())
	if err != nil {
		return sdkmath.LegacyDec{}, err
	}
	if optedUSDValues.ActiveUSDValue.IsZero() {
		return sdkmath.LegacyZeroDec(), nil
	}

	// calculate the active voting power for staker
	assets, err := k.avsKeeper.GetAVSSupportedAssets(ctx, avsAddr)
	if err != nil {
		return sdkmath.LegacyDec{}, err
	}
	if assets == nil {
		return sdkmath.LegacyZeroDec(), nil
	}
	prices, err := k.oracleKeeper.GetMultipleAssetsPrices(ctx, assets)
	// we don't ignore the error regarding the price round not found here, because it's used to
	// distribute the reward.
	if err != nil {
		return sdkmath.LegacyDec{}, err
	}
	if prices == nil {
		return sdkmath.LegacyDec{}, errorsmod.Wrap(operatortypes.ErrValueIsNilOrZero, "CalculateUSDValueForStaker prices map is nil")
	}
	totalUSDValue := sdkmath.LegacyZeroDec()
	opFunc := func(keys *delegationtype.SingleDelegationInfoReq, amounts *delegationtype.DelegationAmounts) (bool, error) {
		// Return true to stop iteration, false to continue iterating
		if keys.OperatorAddr == operator.String() {
			if _, ok := assets[keys.AssetId]; ok {
				price, ok := prices[keys.AssetId]
				if !ok {
					return true, errorsmod.Wrapf(operatortypes.ErrKeyNotExistInMap, "CalculateUSDValueForStaker Price not found for assetID: %s", keys.AssetId)
				}
				operatorAsset, err := k.assetsKeeper.GetOperatorSpecifiedAssetInfo(ctx, operator, keys.AssetId)
				if err != nil {
					return true, err
				}
				amount, err := delegationkeeper.TokensFromShares(amounts.UndelegatableShare, operatorAsset.TotalShare, operatorAsset.TotalAmount)
				if err != nil {
					return true, err
				}
				assetInfo, err := k.assetsKeeper.GetStakingAssetInfo(ctx, keys.AssetId)
				if err != nil {
					return true, err
				}
				usdValue := CalculateUSDValue(amount, price.Value, assetInfo.AssetBasicInfo.Decimals, price.Decimal)
				totalUSDValue = totalUSDValue.Add(usdValue)
			}
		}
		return false, nil
	}
	err = k.delegationKeeper.IterateDelegationsForStaker(ctx, stakerID, opFunc)
	if err != nil {
		return sdkmath.LegacyDec{}, err
	}
	return totalUSDValue, nil
}

// SetOperatorAssetUSDValue is a function to set the operator asset USD value,
func (k *Keeper) SetOperatorAssetUSDValue(ctx sdk.Context, epochIdentifier, operator, assetID string, amount sdkmath.LegacyDec) error {
	if amount.IsNil() {
		return errorsmod.Wrap(operatortypes.ErrValueIsNilOrZero, fmt.Sprintf("SetOperatorAssetUSDValue the amount is:%v", amount))
	}
	store := prefix.NewStore(ctx.KVStore(k.storeKey), operatortypes.KeyPrefixOperatorAssetUSDValue)
	key := assetstype.GetJoinedStoreKey(epochIdentifier, operator, assetID)
	setValue := operatortypes.DecValueField{Amount: amount}
	bz := k.cdc.MustMarshal(&setValue)
	store.Set(key, bz)
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			operatortypes.EventTypeUpdateOperatorAssetUSDValue,
			sdk.NewAttribute(operatortypes.AttributeKeyEpochIdentifier, epochIdentifier),
			sdk.NewAttribute(operatortypes.AttributeKeyOperator, operator),
			sdk.NewAttribute(operatortypes.AttributeKeyAssetID, assetID),
			sdk.NewAttribute(operatortypes.AttributeKeyTotalUSDValue, amount.String()),
		),
	)
	return nil
}

func (k *Keeper) DeleteOperatorAssetUSDValueByEpoch(ctx sdk.Context, epochIdentifier, operator string) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), operatortypes.KeyPrefixOperatorAssetUSDValue)
	prefix := assetstype.GetJoinedStoreKey(epochIdentifier, operator)
	iterator := sdk.KVStorePrefixIterator(store, prefix)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		store.Delete(iterator.Key())
	}
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			operatortypes.EventTypeDeleteOperatorAssetUSDValueByEpoch,
			sdk.NewAttribute(operatortypes.AttributeKeyEpochIdentifier, epochIdentifier),
			sdk.NewAttribute(operatortypes.AttributeKeyOperator, operator),
		),
	)
	return nil
}

// GetOperatorAssetUSDValue is a function to retrieve the USD value of operator asset,
func (k *Keeper) GetOperatorAssetUSDValue(ctx sdk.Context, epochIdentifier, operator, assetID string) (sdkmath.LegacyDec, error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), operatortypes.KeyPrefixOperatorAssetUSDValue)
	key := assetstype.GetJoinedStoreKey(epochIdentifier, operator, assetID)
	var ret operatortypes.DecValueField
	value := store.Get(key)
	if value == nil {
		return sdkmath.LegacyDec{}, operatortypes.ErrNoKeyInTheStore.Wrapf("GetOperatorAssetUSDValue: key is %s", key)
	}
	k.cdc.MustUnmarshal(value, &ret)

	return ret.Amount, nil
}

// HasOperatorAssetUSDValue check whether the USD value of operator asset exists
func (k *Keeper) HasOperatorAssetUSDValue(ctx sdk.Context, epochIdentifier, operator, assetID string) bool {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), operatortypes.KeyPrefixOperatorAssetUSDValue)
	key := assetstype.GetJoinedStoreKey(epochIdentifier, operator, assetID)
	return store.Has(key)
}

func (k *Keeper) SetAllOperatorAssetUSDValues(ctx sdk.Context, usdValues []operatortypes.OperatorAssetUSDValue) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), operatortypes.KeyPrefixOperatorAssetUSDValue)
	for i := range usdValues {
		usdValue := usdValues[i]
		bz := k.cdc.MustMarshal(&usdValue.Value)
		store.Set([]byte(usdValue.Key), bz)
	}
	return nil
}

func (k *Keeper) GetAllOperatorAssetUSDValues(ctx sdk.Context) ([]operatortypes.OperatorAssetUSDValue, error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), operatortypes.KeyPrefixOperatorAssetUSDValue)
	iterator := sdk.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	ret := make([]operatortypes.OperatorAssetUSDValue, 0)
	for ; iterator.Valid(); iterator.Next() {
		var value operatortypes.DecValueField
		k.cdc.MustUnmarshal(iterator.Value(), &value)
		ret = append(ret, operatortypes.OperatorAssetUSDValue{
			Key:   string(iterator.Key()),
			Value: value,
		})
	}
	return ret, nil
}

// UpdateOperatorAssetUSDValue update the operator asset USD value by epoch.
func (k *Keeper) UpdateOperatorAssetUSDValue(ctx sdk.Context, epochIdentifiers []string, operator string) error {
	// check if the epoch is impactful
	impactfulEpochIdentifiers := make([]string, 0)
	for _, epochIdentifier := range epochIdentifiers {
		if !k.IsImpactfulEpochForOperator(ctx, epochIdentifier, operator) {
			// delete the operator asset USD Value
			err := k.DeleteOperatorAssetUSDValueByEpoch(ctx, epochIdentifier, operator)
			if err != nil {
				ctx.Logger().Error("UpdateOperatorAssetUSDValue: failed to delete the asset USD value", "epochIdentifier", epochIdentifier, "operator", operator, "err", err)
			}
			// don't handle the error, because failing to delete shouldn't influence the delete and update
			// for other epoch identifiers
		} else {
			impactfulEpochIdentifiers = append(impactfulEpochIdentifiers, epochIdentifier)
		}
	}
	if len(impactfulEpochIdentifiers) == 0 {
		return nil
	}

	// calculate and update the asset usd value
	// iterate all assets owned by the operator to calculate its voting power
	opFuncToIterateAssets := func(assetID string, state *assetstype.OperatorAssetInfo) error {
		var price oracletype.Price
		var decimal uint32
		price, err := k.oracleKeeper.GetSpecifiedAssetsPrice(ctx, assetID)
		if err != nil {
			// TODO: when assetID is not registered in oracle module, this error will finally lead to panic
			if !errors.Is(err, oracletype.ErrGetPriceRoundNotFound) {
				ctx.Logger().Error("UpdateOperatorAssetUSDValue: failed to get the asset price", "assetID", assetID, "err", err)
				// don't return error to continue handling the other assets.
				return nil
			}
			// TODO: for now, we ignore the error when the price round is not found and set the price to 1 to avoid panic
		}
		assetInfo, err := k.assetsKeeper.GetStakingAssetInfo(ctx, assetID)
		if err != nil {
			ctx.Logger().Error("UpdateOperatorAssetUSDValue: failed to get the asset info", "assetID", assetID, "err", err)
			// don't return error to continue handling the other assets.
			return nil
		}
		decimal = assetInfo.AssetBasicInfo.Decimals
		usdValue := CalculateUSDValue(state.TotalAmount, price.Value, decimal, price.Decimal)
		for _, epochIdentifier := range impactfulEpochIdentifiers {
			err = k.SetOperatorAssetUSDValue(ctx, epochIdentifier, operator, assetID, usdValue)
			if err != nil {
				ctx.Logger().Error("UpdateOperatorAssetUSDValue: failed to set the operator asset USD value", "epochIdentifier", epochIdentifier, "operator", operator, "assetID", assetID, "err", err)
				// don't return error to continue handling the other assets.
			}
		}
		return nil
	}
	err := k.assetsKeeper.IterateAssetsForOperator(ctx, false, operator, nil, opFuncToIterateAssets)
	if err != nil {
		return err
	}
	return nil
}

// UpdateAllOperatorAssetUSDValues update all operator asset USD values for the input epoch list.
// This function will be used for the voting power update. When the voting power update is caused
// by slash, it might not be called at the end of epoch. And all assets USD values of an operator
// should be same for the multiple epoch identifiers.
func (k *Keeper) UpdateAllOperatorAssetUSDValues(ctx sdk.Context, epochIdentifiers []string) error {
	opFunc := func(operatorAddr sdk.AccAddress, _ *operatortypes.OperatorInfo) (bool, error) {
		err := k.UpdateOperatorAssetUSDValue(ctx, epochIdentifiers, operatorAddr.String())
		if err != nil {
			ctx.Logger().Error("UpdateAllOperatorAssetUSDValues: error when updating the specific operator USD value", "err", err, "operator", operatorAddr.String())
			// Don't return an error to continue handling the other operators.
		}
		return false, nil
	}
	err := k.IterateOperators(ctx, opFunc)
	if err != nil {
		return err
	}
	return nil
}

func (k *Keeper) SetAVSAssetsPerEpoch(ctx sdk.Context, avsAddr string, assets []string) error {
	if len(assets) == 0 {
		return nil
	}
	store := prefix.NewStore(ctx.KVStore(k.storeKey), operatortypes.KeyPrefixAVSAssetListPerEpoch)
	key := []byte(strings.ToLower(avsAddr))
	bz := k.cdc.MustMarshal(&operatortypes.AVSAssetsPerEpoch{AssetIDs: assets})
	store.Set(key, bz)
	return nil
}

func (k *Keeper) DeleteAllAVSAssetsPerEpoch(ctx sdk.Context) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), operatortypes.KeyPrefixAVSAssetListPerEpoch)
	iterator := sdk.KVStorePrefixIterator(store, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		store.Delete(iterator.Key())
	}
	return nil
}

func (k *Keeper) GetAVSAssetsPerEpoch(ctx sdk.Context, avsAddr string) ([]string, error) {
	store := prefix.NewStore(
		ctx.KVStore(k.storeKey),
		operatortypes.KeyPrefixAVSAssetListPerEpoch,
	)
	var assets operatortypes.AVSAssetsPerEpoch
	key := []byte(strings.ToLower(avsAddr))
	value := store.Get(key)
	if value == nil {
		return nil, errorsmod.Wrap(operatortypes.ErrNoKeyInTheStore, fmt.Sprintf("GetAVSAssetsPerEpoch: key is %s", key))
	}
	k.cdc.MustUnmarshal(value, &assets)
	return assets.AssetIDs, nil
}

func (k *Keeper) HasAVSAssetsPerEpoch(ctx sdk.Context, avsAddr string) bool {
	store := prefix.NewStore(
		ctx.KVStore(k.storeKey),
		operatortypes.KeyPrefixAVSAssetListPerEpoch,
	)
	key := []byte(strings.ToLower(avsAddr))
	return store.Has(key)
}

func (k *Keeper) GetRecentEndedEpochAVSAssets(ctx sdk.Context, avsAddr string) ([]string, error) {
	if k.HasAVSAssetsPerEpoch(ctx, avsAddr) {
		// the avs assets have been changed, use a dedicated assets list.
		return k.GetAVSAssetsPerEpoch(ctx, avsAddr)
	}
	// the avs assets haven't been changed, so use the real-time assets list in
	// the avs information.
	avsAssetsList, err := k.avsKeeper.GetAVSAssetsList(ctx, avsAddr)
	if err != nil {
		return nil, err
	}
	return avsAssetsList, nil
}
