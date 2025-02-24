package keeper_test

import (
	"fmt"

	assetstype "github.com/imua-xyz/imuachain/x/assets/types"

	"cosmossdk.io/math"
)

func (suite *StakingAssetsTestSuite) TestUpdateStakerAssetsState() {
	stakerID := fmt.Sprintf("%s_%s", suite.Address, "0")
	ethUniAssetID := fmt.Sprintf("%s_%s", "0x1f9840a85d5af5bf1d1762f925bdaddc4201f984", "101")
	ethUniInitialChangeValue := assetstype.DeltaStakerSingleAsset{
		TotalDepositAmount: math.NewInt(1000),
		WithdrawableAmount: math.NewInt(1000),
	}

	// test the initial storage of statker assets state
	_, err := suite.App.AssetsKeeper.UpdateStakerAssetState(suite.Ctx, stakerID, ethUniAssetID, ethUniInitialChangeValue)
	suite.Require().NoError(err)

	// test that the retrieved value is correct
	getInfo, err := suite.App.AssetsKeeper.GetStakerSpecifiedAssetInfo(suite.Ctx, stakerID, ethUniAssetID)
	suite.Require().NoError(err)
	suite.Require().True(ethUniInitialChangeValue.TotalDepositAmount.Equal(getInfo.TotalDepositAmount))
	suite.Require().True(ethUniInitialChangeValue.WithdrawableAmount.Equal(getInfo.WithdrawableAmount))

	// test valid increase of staker asset state
	ethUniInitialChangeValue.TotalDepositAmount = math.NewInt(500)
	ethUniInitialChangeValue.WithdrawableAmount = math.NewInt(500)
	_, err = suite.App.AssetsKeeper.UpdateStakerAssetState(suite.Ctx, stakerID, ethUniAssetID, ethUniInitialChangeValue)
	suite.Require().NoError(err)

	getInfo, err = suite.App.AssetsKeeper.GetStakerSpecifiedAssetInfo(suite.Ctx, stakerID, ethUniAssetID)
	suite.Require().NoError(err)
	suite.Require().True(getInfo.TotalDepositAmount.Equal(math.NewInt(1500)))
	suite.Require().True(getInfo.WithdrawableAmount.Equal(math.NewInt(1500)))

	// test valid decrease of staker asset state
	ethUniInitialChangeValue.TotalDepositAmount = math.NewInt(-500)
	ethUniInitialChangeValue.WithdrawableAmount = math.NewInt(-500)
	_, err = suite.App.AssetsKeeper.UpdateStakerAssetState(suite.Ctx, stakerID, ethUniAssetID, ethUniInitialChangeValue)
	suite.Require().NoError(err)
	getInfo, err = suite.App.AssetsKeeper.GetStakerSpecifiedAssetInfo(suite.Ctx, stakerID, ethUniAssetID)
	suite.Require().NoError(err)
	suite.Require().True(getInfo.TotalDepositAmount.Equal(math.NewInt(1000)))
	suite.Require().True(getInfo.WithdrawableAmount.Equal(math.NewInt(1000)))

	// test the decreased amount is bigger than original state
	ethUniInitialChangeValue.TotalDepositAmount = math.NewInt(-2000)
	ethUniInitialChangeValue.WithdrawableAmount = math.NewInt(-500)
	_, err = suite.App.AssetsKeeper.UpdateStakerAssetState(suite.Ctx, stakerID, ethUniAssetID, ethUniInitialChangeValue)
	suite.Require().Error(err, assetstype.ErrSubAmountIsMoreThanOrigin)
	getInfo, err = suite.App.AssetsKeeper.GetStakerSpecifiedAssetInfo(suite.Ctx, stakerID, ethUniAssetID)
	suite.Require().NoError(err)
	suite.Require().True(getInfo.TotalDepositAmount.Equal(math.NewInt(1000)))
	suite.Require().True(getInfo.WithdrawableAmount.Equal(math.NewInt(1000)))

	ethUniInitialChangeValue.TotalDepositAmount = math.NewInt(-500)
	ethUniInitialChangeValue.WithdrawableAmount = math.NewInt(-2000)
	_, err = suite.App.AssetsKeeper.UpdateStakerAssetState(suite.Ctx, stakerID, ethUniAssetID, ethUniInitialChangeValue)
	suite.Require().Error(err, assetstype.ErrSubAmountIsMoreThanOrigin)
	getInfo, err = suite.App.AssetsKeeper.GetStakerSpecifiedAssetInfo(suite.Ctx, stakerID, ethUniAssetID)
	suite.Require().NoError(err)
	suite.Require().True(getInfo.TotalDepositAmount.Equal(math.NewInt(1000)))
	suite.Require().True(getInfo.WithdrawableAmount.Equal(math.NewInt(1000)))

	// test the storage of multiple assets state
	ethUsdtAssetID := fmt.Sprintf("%s_%s", "0xdac17f958d2ee523a2206206994597c13d831ec7", "101")
	ethUsdtInitialChangeValue := assetstype.DeltaStakerSingleAsset{
		TotalDepositAmount: math.NewInt(2000),
		WithdrawableAmount: math.NewInt(2000),
	}
	_, err = suite.App.AssetsKeeper.UpdateStakerAssetState(suite.Ctx, stakerID, ethUsdtAssetID, ethUsdtInitialChangeValue)
	suite.Require().NoError(err)
	getInfo, err = suite.App.AssetsKeeper.GetStakerSpecifiedAssetInfo(suite.Ctx, stakerID, ethUsdtAssetID)
	suite.Require().NoError(err)
	suite.Require().True(getInfo.TotalDepositAmount.Equal(math.NewInt(2000)))
	suite.Require().True(getInfo.WithdrawableAmount.Equal(math.NewInt(2000)))
}

func (suite *StakingAssetsTestSuite) TestGetStakerAssetInfos() {
	stakerID := fmt.Sprintf("%s_%s", suite.Address, "0x0")
	ethUniAssetID := fmt.Sprintf("%s_%s", "0x1f9840a85d5af5bf1d1762f925bdaddc4201f984", "101")
	ethUsdtAssetID := fmt.Sprintf("%s_%s", "0xdac17f958d2ee523a2206206994597c13d831ec7", "101")
	ethUniInitialChangeValue := assetstype.DeltaStakerSingleAsset{
		TotalDepositAmount:        math.NewInt(1000),
		WithdrawableAmount:        math.NewInt(1000),
		PendingUndelegationAmount: math.NewInt(0),
	}
	ethUsdtInitialChangeValue := assetstype.DeltaStakerSingleAsset{
		TotalDepositAmount:        math.NewInt(2000),
		WithdrawableAmount:        math.NewInt(2000),
		PendingUndelegationAmount: math.NewInt(0),
	}
	assetsInfo := []assetstype.DepositByAsset{
		{
			AssetID: ethUniAssetID,
			Info:    assetstype.StakerAssetInfo(ethUniInitialChangeValue),
		},
		{
			AssetID: ethUsdtAssetID,
			Info:    assetstype.StakerAssetInfo(ethUsdtInitialChangeValue),
		},
	}
	_, err := suite.App.AssetsKeeper.UpdateStakerAssetState(suite.Ctx, stakerID, ethUniAssetID, ethUniInitialChangeValue)
	suite.Require().NoError(err)
	_, err = suite.App.AssetsKeeper.UpdateStakerAssetState(suite.Ctx, stakerID, ethUsdtAssetID, ethUsdtInitialChangeValue)
	suite.Require().NoError(err)

	// test get all assets state of staker
	getAssetsInfo, err := suite.App.AssetsKeeper.GetStakerAssetInfos(suite.Ctx, stakerID)
	suite.Require().NoError(err)
	suite.Contains(getAssetsInfo, assetsInfo[0])
	suite.Contains(getAssetsInfo, assetsInfo[1])
}
