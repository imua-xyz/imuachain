package keeper

import (
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	assetstype "github.com/ExocoreNetwork/exocore/x/assets/types"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// This file provides all functions about operator assets state management.

// AllOperatorAssets
func (k Keeper) AllOperatorAssets(ctx sdk.Context) (operatorAssets []assetstype.AssetsByOperator, err error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), assetstype.KeyPrefixOperatorAssetInfos)
	iterator := sdk.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	ret := make([]assetstype.AssetsByOperator, 0)
	var previousOperator string
	for ; iterator.Valid(); iterator.Next() {
		keyList, err := assetstype.ParseJoinedStoreKey(iterator.Key(), 2)
		if err != nil {
			return nil, err
		}
		operator, assetID := keyList[0], keyList[1]
		if previousOperator != operator {
			assetsByOperator := assetstype.AssetsByOperator{
				Operator:    operator,
				AssetsState: make([]assetstype.AssetByID, 0),
			}
			ret = append(ret, assetsByOperator)
		}
		var assetInfo assetstype.OperatorAssetInfo
		k.cdc.MustUnmarshal(iterator.Value(), &assetInfo)
		index := len(ret) - 1
		ret[index].AssetsState = append(ret[index].AssetsState, assetstype.AssetByID{
			AssetID: assetID,
			Info:    assetInfo,
		})
		previousOperator = operator
	}
	return ret, nil
}

func (k Keeper) GetOperatorAssetInfos(ctx sdk.Context, operatorAddr sdk.Address, assetsFilter map[string]interface{}) (assetsInfo []assetstype.AssetByID, err error) {
	ret := make([]assetstype.AssetByID, 0)
	opFunc := func(assetID string, state *assetstype.OperatorAssetInfo) error {
		ret = append(ret, assetstype.AssetByID{
			AssetID: assetID,
			Info:    *state,
		})
		return nil
	}
	err = k.IterateAssetsForOperator(ctx, false, operatorAddr.String(), assetsFilter, opFunc)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (k Keeper) IsOperatorAssetExist(ctx sdk.Context, operatorAddr sdk.Address, assetID string) bool {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), assetstype.KeyPrefixOperatorAssetInfos)
	key := assetstype.GetJoinedStoreKey(operatorAddr.String(), assetID)
	return store.Has(key)
}

func (k Keeper) GetOperatorSpecifiedAssetInfo(ctx sdk.Context, operatorAddr sdk.Address, assetID string) (info *assetstype.OperatorAssetInfo, err error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), assetstype.KeyPrefixOperatorAssetInfos)
	key := assetstype.GetJoinedStoreKey(operatorAddr.String(), assetID)
	value := store.Get(key)
	if value == nil {
		return nil, assetstype.ErrNoOperatorAssetKey
	}
	ret := assetstype.OperatorAssetInfo{}
	k.cdc.MustUnmarshal(value, &ret)
	return &ret, nil
}

// UpdateOperatorAssetState is used to update the operator states that include TotalAmount OperatorAmount and WaitUndelegationAmount
// The input `changeAmount` represents the values that you want to add or decrease,using positive or negative values for increasing and decreasing,respectively. The function will calculate and update new state after a successful check.
// The function will be called when there is delegation or undelegation related to the operator. In the future,it will also be called when the operator deposit their own assets.
func (k Keeper) UpdateOperatorAssetState(ctx sdk.Context, operatorAddr sdk.Address, assetID string, changeAmount assetstype.DeltaOperatorSingleAsset) (err error) {
	// get the latest state,use the default initial state if the state hasn't been stored
	store := prefix.NewStore(ctx.KVStore(k.storeKey), assetstype.KeyPrefixOperatorAssetInfos)
	key := assetstype.GetJoinedStoreKey(operatorAddr.String(), assetID)
	assetState := assetstype.OperatorAssetInfo{
		TotalAmount:               math.NewInt(0),
		PendingUndelegationAmount: math.NewInt(0),
		TotalShare:                math.LegacyNewDec(0),
		OperatorShare:             math.LegacyNewDec(0),
	}
	value := store.Get(key)
	if value != nil {
		k.cdc.MustUnmarshal(value, &assetState)
	}

	// update all states of the specified operator asset
	err = assetstype.UpdateAssetValue(&assetState.TotalAmount, &changeAmount.TotalAmount)
	if err != nil {
		return errorsmod.Wrap(err, "UpdateOperatorAssetState TotalAmountOrWantChangeValue error")
	}
	err = assetstype.UpdateAssetValue(&assetState.PendingUndelegationAmount, &changeAmount.PendingUndelegationAmount)
	if err != nil {
		return errorsmod.Wrap(err, "UpdateOperatorAssetState WaitUndelegationAmountOrWantChangeValue error")
	}
	err = assetstype.UpdateAssetDecValue(&assetState.TotalShare, &changeAmount.TotalShare)
	if err != nil {
		return errorsmod.Wrap(err, "UpdateOperatorAssetState TotalShare error")
	}
	err = assetstype.UpdateAssetDecValue(&assetState.OperatorShare, &changeAmount.OperatorShare)
	if err != nil {
		return errorsmod.Wrap(err, "UpdateOperatorAssetState OperatorShare error")
	}

	// store the updated state
	bz := k.cdc.MustMarshal(&assetState)
	store.Set(key, bz)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			assetstype.EventTypeUpdatedOperatorAsset,
			sdk.NewAttribute(assetstype.AttributeKeyOperatorAddress, operatorAddr.String()),
			sdk.NewAttribute(assetstype.AttributeKeyAssetID, assetID),
			sdk.NewAttribute(assetstype.AttributeKeyTotalAmount, assetState.TotalAmount.String()),
			sdk.NewAttribute(assetstype.AttributeKeyPendingUndelegationAmount, assetState.PendingUndelegationAmount.String()),
			sdk.NewAttribute(assetstype.AttributeKeyTotalShare, assetState.TotalShare.String()),
			sdk.NewAttribute(assetstype.AttributeKeyOperatorShare, assetState.OperatorShare.String()),
		),
	)

	return nil
}

// IteratorAssetsForOperator iterates all assets for the specified operator
// if `assetsFilter` is nil, the `opFunc` will handle all assets, it equals to an iterator without filter
// if `assetsFilter` isn't nil, the `opFunc` will only handle the assets that is in the filter map.
func (k Keeper) IterateAssetsForOperator(ctx sdk.Context, isUpdate bool, operator string, assetsFilter map[string]interface{}, opFunc func(assetID string, state *assetstype.OperatorAssetInfo) error) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), assetstype.KeyPrefixOperatorAssetInfos)
	iterator := sdk.KVStorePrefixIterator(store, []byte(operator))
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		var amounts assetstype.OperatorAssetInfo
		k.cdc.MustUnmarshal(iterator.Value(), &amounts)
		keys, err := assetstype.ParseJoinedKey(iterator.Key())
		if err != nil {
			return err
		}
		assetID := keys[1]
		if assetsFilter != nil {
			if _, ok := assetsFilter[assetID]; !ok {
				continue
			}
		}
		err = opFunc(assetID, &amounts)
		if err != nil {
			return err
		}
		if isUpdate {
			// store the updated state
			bz := k.cdc.MustMarshal(&amounts)
			store.Set(iterator.Key(), bz)
			ctx.EventManager().EmitEvent(
				sdk.NewEvent(
					assetstype.EventTypeUpdatedOperatorAsset,
					sdk.NewAttribute(assetstype.AttributeKeyOperatorAddress, operator),
					sdk.NewAttribute(assetstype.AttributeKeyAssetID, assetID),
					sdk.NewAttribute(assetstype.AttributeKeyTotalAmount, amounts.TotalAmount.String()),
					sdk.NewAttribute(assetstype.AttributeKeyPendingUndelegationAmount, amounts.PendingUndelegationAmount.String()),
					sdk.NewAttribute(assetstype.AttributeKeyTotalShare, amounts.TotalShare.String()),
					sdk.NewAttribute(assetstype.AttributeKeyOperatorShare, amounts.OperatorShare.String()),
				),
			)
		}
	}
	return nil
}
