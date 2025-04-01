package keeper_test

import (
	"github.com/imua-xyz/imuachain/testutil"
	assetstype "github.com/imua-xyz/imuachain/x/assets/types"
)

func (suite *StakingAssetsTestSuite) TestGenesisClientChainAndAssetInfo() {
	// test the client chains getting
	clientChains, err := suite.App.AssetsKeeper.GetAllClientChainInfo(suite.Ctx)
	suite.NoError(err)
	for _, testClientChain := range testutil.DefaultTestClientChain {
		suite.Contains(clientChains, testClientChain)
	}

	chainInfo, err := suite.App.AssetsKeeper.GetClientChainInfoByIndex(suite.Ctx, 101)
	suite.NoError(err)
	suite.Equal(testutil.DefaultTestClientChain[0], *chainInfo)

	// test the client chain assets getting
	assets, err := suite.App.AssetsKeeper.GetAllStakingAssetsInfo(suite.Ctx)
	suite.NoError(err)
	for _, assetInfo := range assets {
		suite.Contains(testutil.DefaultTestStakingAssets, assetInfo.AssetBasicInfo)
	}

	usdtAsset := testutil.DefaultTestStakingAssets[0]
	_, assetID := assetstype.GetStakerIDAndAssetIDFromStr(usdtAsset.LayerZeroChainID, "", usdtAsset.Address)
	assetInfo, err := suite.App.AssetsKeeper.GetStakingAssetInfo(suite.Ctx, assetID)
	suite.NoError(err)
	suite.Equal(usdtAsset, assetInfo.AssetBasicInfo)
}
