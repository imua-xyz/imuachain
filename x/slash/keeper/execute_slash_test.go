package keeper_test

import (
	sdkmath "cosmossdk.io/math"
	"github.com/ethereum/go-ethereum/common"
	assetskeeper "github.com/imua-xyz/imuachain/x/assets/keeper"
	"github.com/imua-xyz/imuachain/x/assets/types"
	"github.com/imua-xyz/imuachain/x/slash/keeper"
	slashtype "github.com/imua-xyz/imuachain/x/slash/types"
)

func (suite *SlashTestSuite) TestSlash() {
	usdtAddress := common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7")
	usdcAddress := common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48")
	event := &keeper.SlashParams{
		ClientChainLzID: 101,
		Action:          types.Slash,
		StakerAddress:   suite.Address[:],
		OpAmount:        sdkmath.NewInt(90),
	}

	depositEvent := &assetskeeper.DepositWithdrawParams{
		ClientChainLzID: 101,
		Action:          types.DepositLST,
		StakerAddress:   suite.Address[:],
		OpAmount:        sdkmath.NewInt(100),
	}

	assets, err := suite.App.AssetsKeeper.GetAllStakingAssetsInfo(suite.Ctx)
	suite.NoError(err)
	suite.App.Logger().Info("the assets is:", "assets", assets)

	// deposit firstly
	depositEvent.AssetsAddress = usdtAddress[:]
	err = suite.App.AssetsKeeper.PerformDepositOrWithdraw(suite.Ctx, depositEvent)
	suite.NoError(err)

	// test the case that the slash  hasn't registered
	event.AssetsAddress = usdcAddress[:]
	err = suite.App.ExoSlashKeeper.Slash(suite.Ctx, event)
	suite.ErrorContains(err, slashtype.ErrSlashAssetNotExist.Error())

	stakerID, assetID := types.GetStakerIDAndAssetID(depositEvent.ClientChainLzID, depositEvent.StakerAddress, depositEvent.AssetsAddress)
	info, err := suite.App.AssetsKeeper.GetStakerSpecifiedAssetInfo(suite.Ctx, stakerID, assetID)
	suite.NoError(err)
	suite.Equal(types.StakerAssetInfo{
		TotalDepositAmount:        depositEvent.OpAmount,
		WithdrawableAmount:        depositEvent.OpAmount,
		PendingUndelegationAmount: sdkmath.NewInt(0),
	}, *info)

	// test the normal case
	event.AssetsAddress = usdtAddress[:]
	err = suite.App.ExoSlashKeeper.Slash(suite.Ctx, event)
	suite.NoError(err)

	// check state after slash
	stakerID, assetID = types.GetStakerIDAndAssetID(event.ClientChainLzID, event.StakerAddress, event.AssetsAddress)
	info, err = suite.App.AssetsKeeper.GetStakerSpecifiedAssetInfo(suite.Ctx, stakerID, assetID)
	suite.NoError(err)
	suite.Equal(types.StakerAssetInfo{
		TotalDepositAmount:        sdkmath.NewInt(10),
		WithdrawableAmount:        sdkmath.NewInt(10),
		PendingUndelegationAmount: sdkmath.NewInt(0),
	}, *info)

	assetInfo, err := suite.App.AssetsKeeper.GetStakingAssetInfo(suite.Ctx, assetID)
	suite.NoError(err)
	suite.Equal(assets[0].StakingTotalAmount.Add(depositEvent.OpAmount).Sub(event.OpAmount), assetInfo.StakingTotalAmount)
}
