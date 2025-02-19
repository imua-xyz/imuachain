package types

// AssetID obtains the asset ID from the staking asset info
func (info StakingAssetInfo) AssetID() string {
	return info.AssetBasicInfo.AssetID()
}

// AssetID obtains the asset ID from the asset info
func (info AssetInfo) AssetID() string {
	_, assetID := GetStakerIDAndAssetIDFromStr(
		info.LayerZeroChainID, "", info.Address,
	)
	return assetID
}
