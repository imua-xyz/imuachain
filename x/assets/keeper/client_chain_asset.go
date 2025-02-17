package keeper

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	assetstype "github.com/ExocoreNetwork/exocore/x/assets/types"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// UpdateStakingAssetTotalAmount updating the total deposited amount of a specified asset in exoCore chain
// The function will be called when stakers deposit and withdraw their assets
func (k Keeper) UpdateStakingAssetTotalAmount(ctx sdk.Context, assetID string, changeAmount sdkmath.Int) (err error) {
	if changeAmount.IsNil() {
		return assetstype.ErrInputPointerIsNil
	}
	store := prefix.NewStore(ctx.KVStore(k.storeKey), assetstype.KeyPrefixReStakingAssetInfo)
	key := []byte(assetID)
	value := store.Get(key)
	if value == nil {
		return assetstype.ErrNoClientChainAssetKey
	}

	ret := assetstype.StakingAssetInfo{}
	k.cdc.MustUnmarshal(value, &ret)

	// calculate and set new amount
	err = assetstype.UpdateAssetValue(&ret.StakingTotalAmount, &changeAmount)
	if err != nil {
		return err
	}
	bz := k.cdc.MustMarshal(&ret)
	store.Set(key, bz)

	// emit event for indexers
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			assetstype.EventTypeUpdatedStakingTotalAmount,
			sdk.NewAttribute(assetstype.AttributeKeyAssetID, assetID),
			sdk.NewAttribute(assetstype.AttributeKeyTotalAmount, ret.StakingTotalAmount.String()),
		),
	)

	return nil
}

// SetStakingAssetInfo todo: Temporarily use clientChainAssetAddr+'_'+LayerZeroChainID as the key.
// It provides a function to register the client chain assets supported by Exocore.
// New assets may be registered via ExocoreGateway, which uses precompiles to call this function.
// If an asset with the provided assetID already exists, it will return an error.
func (k Keeper) SetStakingAssetInfo(ctx sdk.Context, info *assetstype.StakingAssetInfo) (err error) {
	if info.AssetBasicInfo.Decimals > assetstype.MaxDecimal {
		return errorsmod.Wrapf(assetstype.ErrInvalidInputParameter, "the decimal is greater than the MaxDecimal,decimal:%v,MaxDecimal:%v", info.AssetBasicInfo.Decimals, assetstype.MaxDecimal)
	}
	if info.StakingTotalAmount.IsNegative() {
		return errorsmod.Wrapf(assetstype.ErrInvalidInputParameter, "the total staking amount is negative, StakingTotalAmount:%v", info.StakingTotalAmount)
	}
	store := prefix.NewStore(ctx.KVStore(k.storeKey), assetstype.KeyPrefixReStakingAssetInfo)
	_, assetID := assetstype.GetStakerIDAndAssetIDFromStr(info.AssetBasicInfo.LayerZeroChainID, "", info.AssetBasicInfo.Address)
	if store.Has([]byte(assetID)) {
		return assetstype.ErrRegisterDuplicateAssetID.Wrapf(
			"the asset has already been registered,assetID:%v,LayerZeroChainID:%v,ClientChainAssetAddr:%v",
			assetID, info.AssetBasicInfo.LayerZeroChainID, info.AssetBasicInfo.Address,
		)
	}
	bz := k.cdc.MustMarshal(info)
	store.Set([]byte(assetID), bz)
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			assetstype.EventTypeNewToken,
			sdk.NewAttribute(assetstype.AttributeKeyAssetID, assetID),
			sdk.NewAttribute(assetstype.AttributeKeyName, info.AssetBasicInfo.Name),
			sdk.NewAttribute(assetstype.AttributeKeySymbol, info.AssetBasicInfo.Symbol),
			sdk.NewAttribute(assetstype.AttributeKeyAddress, info.AssetBasicInfo.Address),
			sdk.NewAttribute(assetstype.AttributeKeyDecimals, fmt.Sprintf("%d", info.AssetBasicInfo.Decimals)),
			sdk.NewAttribute(assetstype.AttributeKeyLZID, fmt.Sprintf("%d", info.AssetBasicInfo.LayerZeroChainID)),
			sdk.NewAttribute(assetstype.AttributeKeyMetaInfo, info.AssetBasicInfo.MetaInfo),
			sdk.NewAttribute(assetstype.AttributeKeyExocoreChainIdx, fmt.Sprintf("%d", info.AssetBasicInfo.ExocoreChainIndex)),
			sdk.NewAttribute(assetstype.AttributeKeyTotalAmount, info.StakingTotalAmount.String()),
		),
	)
	return nil
}

// IsStakingAsset checks if the assetID is a staking asset.
func (k Keeper) IsStakingAsset(ctx sdk.Context, assetID string) bool {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), assetstype.KeyPrefixReStakingAssetInfo)
	return store.Has([]byte(assetID))
}

// UpdateStakingAssetMetaInfo updates the meta information stored against a provided assetID.
// If the assetID does not exist, it returns an error.
func (k Keeper) UpdateStakingAssetMetaInfo(ctx sdk.Context, assetID string, metainfo string) error {
	info, err := k.GetStakingAssetInfo(ctx, assetID)
	if err != nil {
		return err
	}
	store := prefix.NewStore(ctx.KVStore(k.storeKey), assetstype.KeyPrefixReStakingAssetInfo)
	info.AssetBasicInfo.MetaInfo = metainfo
	bz := k.cdc.MustMarshal(info)
	store.Set([]byte(assetID), bz)
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			assetstype.EventTypeUpdatedToken,
			sdk.NewAttribute(assetstype.AttributeKeyAssetID, assetID),
			sdk.NewAttribute(assetstype.AttributeKeyMetaInfo, metainfo),
		),
	)
	return nil
}

// GetStakingAssetInfo returns the asset information stored against a provided assetID.
func (k Keeper) GetStakingAssetInfo(ctx sdk.Context, assetID string) (info *assetstype.StakingAssetInfo, err error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), assetstype.KeyPrefixReStakingAssetInfo)
	value := store.Get([]byte(assetID))
	if value == nil {
		return nil, assetstype.ErrNoClientChainAssetKey
	}

	ret := assetstype.StakingAssetInfo{}
	k.cdc.MustUnmarshal(value, &ret)
	return &ret, nil
}

// GetAssetsDecimal returns the decimal of all the provided assets.
func (k Keeper) GetAssetsDecimal(ctx sdk.Context, assets map[string]interface{}) (decimals map[string]uint32, err error) {
	if assets == nil {
		return nil, errorsmod.Wrap(assetstype.ErrInputPointerIsNil, "assets is nil")
	}
	store := prefix.NewStore(ctx.KVStore(k.storeKey), assetstype.KeyPrefixReStakingAssetInfo)
	decimals = make(map[string]uint32, 0)
	for assetID := range assets {
		value := store.Get([]byte(assetID))
		if value == nil {
			return nil, assetstype.ErrNoClientChainAssetKey
		}
		ret := assetstype.StakingAssetInfo{}
		k.cdc.MustUnmarshal(value, &ret)
		decimals[assetID] = ret.AssetBasicInfo.Decimals
	}

	return decimals, nil
}

func (k Keeper) GetAllStakingAssetsInfo(ctx sdk.Context) (allAssets []assetstype.StakingAssetInfo, err error) {
	store := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStorePrefixIterator(store, assetstype.KeyPrefixReStakingAssetInfo)
	defer iterator.Close()

	ret := make([]assetstype.StakingAssetInfo, 0)
	for ; iterator.Valid(); iterator.Next() {
		var assetInfo assetstype.StakingAssetInfo
		k.cdc.MustUnmarshal(iterator.Value(), &assetInfo)
		ret = append(ret, assetInfo)
	}
	return ret, nil
}
