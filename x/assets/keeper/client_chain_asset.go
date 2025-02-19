package keeper

import (
	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	assetstype "github.com/imua-xyz/imuachain/x/assets/types"
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
	return nil
}

// SetStakingAssetInfo todo: Temporarily use clientChainAssetAddr+'_'+LayerZeroChainID as the key.
// It provides a function to register the client chain assets supported by exoCore.It's called by genesis configuration now,however it will be called by the governance in the future
func (k Keeper) SetStakingAssetInfo(ctx sdk.Context, info *assetstype.StakingAssetInfo) (err error) {
	if info.AssetBasicInfo.Decimals > assetstype.MaxDecimal {
		return errorsmod.Wrapf(assetstype.ErrInvalidInputParameter, "the decimal is greater than the MaxDecimal,decimal:%v,MaxDecimal:%v", info.AssetBasicInfo.Decimals, assetstype.MaxDecimal)
	}
	if info.StakingTotalAmount.IsNegative() {
		return errorsmod.Wrapf(assetstype.ErrInvalidInputParameter, "the total staking amount is negative, StakingTotalAmount:%v", info.StakingTotalAmount)
	}
	store := prefix.NewStore(ctx.KVStore(k.storeKey), assetstype.KeyPrefixReStakingAssetInfo)
	// key := common.HexToAddress(incentive.Contract)
	bz := k.cdc.MustMarshal(info)

	_, assetID := assetstype.GetStakerIDAndAssetIDFromStr(info.AssetBasicInfo.LayerZeroChainID, "", info.AssetBasicInfo.Address)
	store.Set([]byte(assetID), bz)
	return nil
}

func (k Keeper) IsStakingAsset(ctx sdk.Context, assetID string) bool {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), assetstype.KeyPrefixReStakingAssetInfo)
	return store.Has([]byte(assetID))
}

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
