package keeper_test

import (
	"cosmossdk.io/math"
	assetstype "github.com/imua-xyz/imuachain/x/assets/types"
)

func (suite *StakingAssetsTestSuite) TestGenesisClientChainAndAssetInfo() {
	ethClientChain := assetstype.ClientChainInfo{
		Name:               "ethereum",
		MetaInfo:           "ethereum blockchain",
		ChainId:            1,
		FinalizationBlocks: 10,
		LayerZeroChainID:   101,
		AddressLength:      20,
	}
	usdtClientChainAsset := assetstype.AssetInfo{
		Name:             "Tether USD",
		Symbol:           "USDT",
		Address:          "0xdAC17F958D2ee523a2206206994597C13D831ec7",
		Decimals:         6,
		LayerZeroChainID: ethClientChain.LayerZeroChainID,
		MetaInfo:         "Tether USD token",
	}
	stakingInfo := assetstype.StakingAssetInfo{
		AssetBasicInfo:     usdtClientChainAsset,
		StakingTotalAmount: math.NewIntWithDecimal(201, 6),
	}
	defaultGensisState := assetstype.NewGenesis(
		assetstype.DefaultParams(),
		[]assetstype.ClientChainInfo{ethClientChain},
		[]assetstype.StakingAssetInfo{stakingInfo},
		[]assetstype.DepositsByStaker{},
		nil,
	)

	// test the client chains getting
	clientChains, err := suite.App.AssetsKeeper.GetAllClientChainInfo(suite.Ctx)
	suite.NoError(err)
	suite.Contains(clientChains, defaultGensisState.ClientChains[0])

	chainInfo, err := suite.App.AssetsKeeper.GetClientChainInfoByIndex(suite.Ctx, 101)
	suite.NoError(err)
	suite.Equal(clientChains[0], *chainInfo)

	// test the client chain assets getting
	assets, err := suite.App.AssetsKeeper.GetAllStakingAssetsInfo(suite.Ctx)
	suite.NoError(err)
	suite.Contains(assets, defaultGensisState.Tokens[0])

	usdtAssetX := defaultGensisState.Tokens[0]
	usdtAsset := usdtAssetX.AssetBasicInfo
	_, assetID := assetstype.GetStakerIDAndAssetIDFromStr(usdtAsset.LayerZeroChainID, "", usdtAsset.Address)
	assetInfo, err := suite.App.AssetsKeeper.GetStakingAssetInfo(suite.Ctx, assetID)
	suite.NoError(err)
	suite.Equal(usdtAsset, assetInfo.AssetBasicInfo)
}
