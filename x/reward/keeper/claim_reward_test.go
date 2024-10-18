package keeper_test

import (
	sdkmath "cosmossdk.io/math"
	"github.com/ExocoreNetwork/exocore/x/assets/types"
	"github.com/ExocoreNetwork/exocore/x/reward/keeper"
	rewardtype "github.com/ExocoreNetwork/exocore/x/reward/types"
	"github.com/ethereum/go-ethereum/common"
)

func (suite *RewardTestSuite) TestClaimWithdrawRequest() {
	_, err := suite.App.AssetsKeeper.GetAllStakingAssetsInfo(suite.Ctx)
	suite.NoError(err)

	usdtAddress := common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7")
	usdcAddress := common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48")
	event := &keeper.RewardParams{
		ClientChainLzID:       101,
		Action:                types.WithdrawReward,
		WithdrawRewardAddress: suite.Address[:],
		OpAmount:              sdkmath.NewInt(10),
	}

	// test the case that the deposit asset hasn't registered
	event.AssetsAddress = usdcAddress[:]
	err = suite.App.RewardKeeper.RewardForWithdraw(suite.Ctx, event)
	// suite.ErrorContains(err, rewardtype.ErrRewardAssetNotExist.Error())
	suite.ErrorContains(err, rewardtype.ErrNotSupportYet.Error())

	// test the normal case
	event.AssetsAddress = usdtAddress[:]
	err = suite.App.RewardKeeper.RewardForWithdraw(suite.Ctx, event)
	// suite.NoError(err)
	suite.ErrorContains(err, rewardtype.ErrNotSupportYet.Error())

	// check state after reward
	// stakerID, assetID := types.GetStakerIDAndAssetID(event.ClientChainLzID, event.WithdrawRewardAddress, event.AssetsAddress)
	// info, err := suite.App.AssetsKeeper.GetStakerSpecifiedAssetInfo(suite.Ctx, stakerID, assetID)
	// suite.NoError(err)
	// suite.Equal(types.StakerAssetInfo{
	// 	TotalDepositAmount:        sdkmath.NewInt(10),
	// 	WithdrawableAmount:        sdkmath.NewInt(10),
	// 	PendingUndelegationAmount: sdkmath.NewInt(0),
	// }, *info)

	// assetInfo, err := suite.App.AssetsKeeper.GetStakingAssetInfo(suite.Ctx, assetID)
	// suite.NoError(err)
	// suite.Equal(sdkmath.NewInt(10).Add(assets[0].StakingTotalAmount), assetInfo.StakingTotalAmount)
}
