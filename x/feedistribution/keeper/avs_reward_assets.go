package keeper

import (
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"

	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	assetstype "github.com/imua-xyz/imuachain/x/assets/types"
	"github.com/imua-xyz/imuachain/x/feedistribution/types"
)

// UpdateAVSRewardAssetState updates the reward asset state of a specified AVS.
// This function is called when the AVS funds the reward pool, when the reward distribution is executed,
// and when stakers or operators claim rewards.
// There will be a precompiled contract interface regarding it.
func (k Keeper) UpdateAVSRewardAssetState(ctx sdk.Context, avsAddr, assetID string,
	delta *types.DeltaAVSRewardAssetState,
) (err error) {
	if delta == nil {
		return types.ErrInvalidRewardAssetParameter.Wrapf("UpdateAVSRewardAssetState: the input delta is nil,AvsAddr:%s,assetID:%s", avsAddr, assetID)
	}
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixAVSRewardAssets)
	key := assetstype.GetJoinedStoreKey(avsAddr, assetID)
	value := store.Get(key)
	if value == nil {
		return types.ErrAVSRewardAssetNotFound.Wrapf("avs:%s,assetID:%s", avsAddr, assetID)
	}

	ret := types.AVSRewardAsset{}
	k.cdc.MustUnmarshal(value, &ret)

	// update all states of the reward asset
	err = assetstype.UpdateAssetDecValue(&ret.RewardAssetState.RewardPoolBalance, &delta.RewardPoolBalance)
	if err != nil {
		return err
	}
	err = assetstype.UpdateAssetDecValue(&ret.RewardAssetState.RewardPoolTotal, &delta.RewardPoolTotal)
	if err != nil {
		return err
	}
	err = assetstype.UpdateAssetDecValue(&ret.RewardAssetState.RewardAllocationTotal, &delta.RewardAllocationTotal)
	if err != nil {
		return err
	}
	bz := k.cdc.MustMarshal(&ret)
	store.Set(key, bz)

	// emit event for indexers
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeUpdatedAVSRewardAsset,
			sdk.NewAttribute(types.AttributeKeyAvsAddress, avsAddr),
			sdk.NewAttribute(types.AttributeKeyAssetID, assetID),
			sdk.NewAttribute(types.AttributeKeyRewardPoolBalance, ret.RewardAssetState.RewardPoolBalance.String()),
			sdk.NewAttribute(types.AttributeKeyRewardPoolTotal, ret.RewardAssetState.RewardPoolTotal.String()),
			sdk.NewAttribute(types.AttributeKeyRewardAllocationTotal, ret.RewardAssetState.RewardAllocationTotal.String()),
		),
	)
	return nil
}

// SetAVSRewardAssets
// It provides a function to register the reward assets by AVS. It will be provided to the AVS by the precompile
// interface. If an asset with the provided assetID already exists, it will return an error.
func (k Keeper) SetAVSRewardAssets(ctx sdk.Context, avsAddr string, assets []assetstype.AssetInfo) (err error) {
	assetStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixAVSRewardAssets)
	symbolStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixAVSRewardAssetBySymbol)
	symbolMap := make(map[string]interface{}, len(assets))
	// check if the AVS is registered
	if isAVS, _ := k.avsKeeper.IsAVS(ctx, avsAddr); !isAVS {
		return types.ErrInvalidRewardAssetParameter.Wrapf("AVS not found %s", avsAddr)
	}
	for _, assetInfo := range assets {
		if assetInfo.Decimals > assetstype.MaxDecimal {
			return types.ErrInvalidRewardAssetParameter.Wrapf("the decimal is greater than the MaxDecimal,decimal:%v,MaxDecimal:%v", assetInfo.Decimals, assetstype.MaxDecimal)
		}
		// check for symbol duplication
		if _, ok := symbolMap[assetInfo.Symbol]; ok {
			return types.ErrInvalidRewardAssetParameter.Wrapf("duplicated symbol: %s", assetInfo.Symbol)
		}
		symbolMap[assetInfo.Symbol] = nil
		_, assetID := assetstype.GetStakerIDAndAssetIDFromStr(assetInfo.LayerZeroChainID, "", assetInfo.Address)
		assetKey := assetstype.GetJoinedStoreKey(avsAddr, assetID)
		symbolKey := assetstype.GetJoinedStoreKey(avsAddr, assetInfo.Symbol)
		if assetStore.Has(assetKey) {
			return types.ErrInvalidRewardAssetParameter.Wrapf("the reward asset is already stored,AvsAddr:%s,assetID:%s", avsAddr, assetID)
		}
		if symbolStore.Has(symbolKey) {
			return types.ErrInvalidRewardAssetParameter.Wrapf("the symbol key of reward asset is already stored,AvsAddr:%s,assetID:%s,symbol:%s", avsAddr, assetID, assetInfo.Symbol)
		}
		// set the reward asset info
		avsRewardAsset := &types.AVSRewardAsset{
			AssetBasicInfo: assetInfo,
			RewardAssetState: types.AVSRewardAssetState{
				RewardPoolBalance:     sdkmath.LegacyZeroDec(),
				RewardPoolTotal:       sdkmath.LegacyZeroDec(),
				RewardAllocationTotal: sdkmath.LegacyZeroDec(),
			},
		}
		bz := k.cdc.MustMarshal(avsRewardAsset)
		assetStore.Set(assetKey, bz)
		symbolStore.Set(symbolKey, []byte(assetID))
		// emit event for indexers
		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				types.EventTypeNewAVSRewardAsset,
				sdk.NewAttribute(types.AttributeKeyAvsAddress, avsAddr),
				sdk.NewAttribute(assetstype.AttributeKeyAssetID, assetID),
				sdk.NewAttribute(assetstype.AttributeKeyName, assetInfo.Name),
				sdk.NewAttribute(assetstype.AttributeKeySymbol, assetInfo.Symbol),
				sdk.NewAttribute(assetstype.AttributeKeyAddress, assetInfo.Address),
				sdk.NewAttribute(assetstype.AttributeKeyDecimals, fmt.Sprintf("%d", assetInfo.Decimals)),
				sdk.NewAttribute(assetstype.AttributeKeyLZID, fmt.Sprintf("%d", assetInfo.LayerZeroChainID)),
				sdk.NewAttribute(assetstype.AttributeKeyMetaInfo, assetInfo.MetaInfo),
				sdk.NewAttribute(assetstype.AttributeKeyImuachainIndex, fmt.Sprintf("%d", assetInfo.ImuaChainIndex)),
			),
		)
	}
	return nil
}

// IsAVSAllRewardsClaimed checks whether all rewards for the given AVS have been distributed and claimed.
// This imposes a restriction on AVS deregistration to prevent AVSs from escaping rewards.
// TODO: This restriction may introduce an issue where a staker can block an AVS from being removed from the
// protocol by refusing to claim rewards. In the future, we may need to design a mechanism to support
// forced deregistration to handle this case.
func (k Keeper) IsAVSAllRewardsClaimed(ctx sdk.Context, avsAddr string) bool {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixAVSRewardAssets)
	iterator := sdk.KVStorePrefixIterator(store, []byte(strings.ToLower(avsAddr)))
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var avsRewardAsset types.AVSRewardAsset
		k.cdc.MustUnmarshal(iterator.Value(), &avsRewardAsset)
		// check if the reward has been distributed and claimed
		claimedAmount := avsRewardAsset.RewardAssetState.RewardPoolTotal.Sub(avsRewardAsset.RewardAssetState.RewardPoolBalance)
		if !avsRewardAsset.RewardAssetState.RewardAllocationTotal.Equal(claimedAmount) {
			return false
		}
	}
	return true
}

// IsAVSRewardAssetByAssetID checks if the assetID is a reward asset of specified AVS.
func (k Keeper) IsAVSRewardAssetByAssetID(ctx sdk.Context, avsAddr, assetID string) bool {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixAVSRewardAssets)
	key := assetstype.GetJoinedStoreKey(avsAddr, assetID)
	return store.Has(key)
}

// GetAVSRewardAssetInfo returns the avs reward asset information stored against the  provided AvsAddr and assetID.
func (k Keeper) GetAVSRewardAssetInfo(ctx sdk.Context, avsAddr, assetID string) (info *types.AVSRewardAsset, err error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixAVSRewardAssets)
	key := assetstype.GetJoinedStoreKey(avsAddr, assetID)
	value := store.Get(key)
	if value == nil {
		return nil, types.ErrAVSRewardAssetNotFound.Wrapf("avs:%s,assetID:%s", avsAddr, assetID)
	}

	ret := types.AVSRewardAsset{}
	k.cdc.MustUnmarshal(value, &ret)
	return &ret, nil
}

// IsAVSRewardAssetBySymbol checks if the symbol is a reward asset of specified AVS.
func (k Keeper) IsAVSRewardAssetBySymbol(ctx sdk.Context, avsAddr, symbol string) bool {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixAVSRewardAssetBySymbol)
	key := assetstype.GetJoinedStoreKey(avsAddr, symbol)
	return store.Has(key)
}

// GetAVSRewardAssetIDBySymbol returns the avs reward assetID stored against the  provided AvsAddr and symbol.
func (k Keeper) GetAVSRewardAssetIDBySymbol(ctx sdk.Context, avsAddr, symbol string) (assetID string, err error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixAVSRewardAssetBySymbol)
	key := assetstype.GetJoinedStoreKey(avsAddr, symbol)
	value := store.Get(key)
	if value == nil {
		return assetID, types.ErrAVSRewardAssetNotFound.Wrapf("avs:%s,symbol:%s", avsAddr, symbol)
	}
	return string(value), nil
}

// GetAVSRewardAssetBySymbol returns the avs reward asset information stored against the  provided AvsAddr and symbol.
func (k Keeper) GetAVSRewardAssetBySymbol(ctx sdk.Context, avsAddr, symbol string) (assetID string,
	info *types.AVSRewardAsset, err error,
) {
	assetID, err = k.GetAVSRewardAssetIDBySymbol(ctx, avsAddr, symbol)
	if err != nil {
		return "", nil, err
	}
	assetInfo, err := k.GetAVSRewardAssetInfo(ctx, avsAddr, assetID)
	if err != nil {
		return assetID, nil, err
	}
	return assetID, assetInfo, nil
}

// UpdateAVSRewardAssetMetaInfo updates the meta information stored against the AvsAddr and assetID.
// If the key does not exist, it returns an error.
func (k Keeper) UpdateAVSRewardAssetMetaInfo(ctx sdk.Context, avsAddr, assetID string, metainfo string) error {
	info, err := k.GetAVSRewardAssetInfo(ctx, avsAddr, assetID)
	if err != nil {
		return err
	}
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixAVSRewardAssets)
	key := assetstype.GetJoinedStoreKey(avsAddr, assetID)
	info.AssetBasicInfo.MetaInfo = metainfo
	bz := k.cdc.MustMarshal(info)
	store.Set(key, bz)
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeUpdatedRewardAssetMetaInfo,
			sdk.NewAttribute(types.AttributeKeyAvsAddress, avsAddr),
			sdk.NewAttribute(assetstype.AttributeKeyAssetID, assetID),
			sdk.NewAttribute(assetstype.AttributeKeyMetaInfo, metainfo),
		),
	)
	return nil
}

func (k Keeper) GetAllRewardAssetsByAVS(ctx sdk.Context, avsAddr string) (allRewardAssets types.AVSRewardAssets, err error) {
	if !common.IsHexAddress(avsAddr) {
		return types.AVSRewardAssets{}, types.ErrInvalidInputParameter.Wrapf("Invalid AVS address: must be a valid EVM hexadecimal address, avsAddr:%s", avsAddr)
	}
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixAVSRewardAssets)
	iterator := sdk.KVStorePrefixIterator(store, []byte(avsAddr))
	defer iterator.Close()

	ret := make([]types.AVSRewardAsset, 0)
	for ; iterator.Valid(); iterator.Next() {
		var avsRewardAsset types.AVSRewardAsset
		k.cdc.MustUnmarshal(iterator.Value(), &avsRewardAsset)
		ret = append(ret, avsRewardAsset)
	}
	return types.AVSRewardAssets{
		AvsRewardAssets: ret,
	}, nil
}

func (k Keeper) GetAllAVSRewardAssetSymbols(ctx sdk.Context, avsAddr string) (symbols []string, err error) {
	if !common.IsHexAddress(avsAddr) {
		return nil, types.ErrInvalidInputParameter.Wrapf("Invalid AVS address: must be a valid EVM hexadecimal address, avsAddr:%s", avsAddr)
	}
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixAVSRewardAssets)
	iterator := sdk.KVStorePrefixIterator(store, []byte(avsAddr))
	defer iterator.Close()

	ret := make([]string, 0)
	for ; iterator.Valid(); iterator.Next() {
		var avsRewardAsset types.AVSRewardAsset
		k.cdc.MustUnmarshal(iterator.Value(), &avsRewardAsset)
		ret = append(ret, avsRewardAsset.AssetBasicInfo.Symbol)
	}
	return ret, nil
}

func (k Keeper) SetAllAVSRewardAssets(ctx sdk.Context, allAVSRewardAssets []types.AVSAddrAndRewardAssets) error {
	assetStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixAVSRewardAssets)
	assetSymbolStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixAVSRewardAssetBySymbol)

	for _, avsRewardAsset := range allAVSRewardAssets {
		for i := range avsRewardAsset.AvsRewardAssets {
			rewardAsset := avsRewardAsset.AvsRewardAssets[i]

			// check the decimal
			if rewardAsset.AssetBasicInfo.Decimals > assetstype.MaxDecimal {
				return types.ErrInvalidInputParameter.Wrapf("the decimal is greater than the MaxDecimal,decimal:%v,MaxDecimal:%v", rewardAsset.AssetBasicInfo.Decimals, assetstype.MaxDecimal)
			}

			bz := k.cdc.MustMarshal(&rewardAsset)
			_, assetID := assetstype.GetStakerIDAndAssetIDFromStr(rewardAsset.AssetBasicInfo.LayerZeroChainID,
				"", rewardAsset.AssetBasicInfo.Address)
			assetKey := assetstype.GetJoinedStoreKey(avsRewardAsset.Avs, assetID)
			assetStore.Set(assetKey, bz)
			symbolKey := assetstype.GetJoinedStoreKey(avsRewardAsset.Avs,
				rewardAsset.AssetBasicInfo.Symbol)
			assetSymbolStore.Set(symbolKey, []byte(assetID))
		}
	}
	return nil
}

func (k Keeper) GetAllAVSRewardAssets(ctx sdk.Context) ([]types.AVSAddrAndRewardAssets, error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixAVSRewardAssets)
	iterator := sdk.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	ret := make([]types.AVSAddrAndRewardAssets, 0)
	avs := ""
	for ; iterator.Valid(); iterator.Next() {
		keys, err := assetstype.ParseJoinedStoreKey(iterator.Key(), 2)
		if err != nil {
			return nil, err
		}
		// the iterator is ordered, allowing all assets of the same AVS to be processed together.
		// initialize the slice when meeting the new avs.
		if avs != keys[0] {
			ret = append(ret, types.AVSAddrAndRewardAssets{
				Avs:             keys[0],
				AvsRewardAssets: make([]types.AVSRewardAsset, 0),
			})
			avs = keys[0]
		}
		var rewardAsset types.AVSRewardAsset
		k.cdc.MustUnmarshal(iterator.Value(), &rewardAsset)
		avsNumber := len(ret)
		ret[avsNumber-1].AvsRewardAssets = append(ret[avsNumber-1].AvsRewardAssets, rewardAsset)
	}
	return ret, nil
}

func (k Keeper) IsRegisteredRewardAsset(ctx sdk.Context, assetID string) bool {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixAVSRewardAssets)
	iterator := sdk.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		keys, err := assetstype.ParseJoinedStoreKey(iterator.Key(), 2)
		if err != nil {
			return false
		}
		if keys[1] == assetID {
			return true
		}
	}
	return false
}
