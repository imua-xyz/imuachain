package keeper_test

import (
	"cosmossdk.io/math"

	operatortype "github.com/ExocoreNetwork/exocore/x/operator/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func (suite *OperatorTestSuite) TestOperatorInfo() {
	info := &operatortype.OperatorInfo{
		EarningsAddr:     suite.AccAddress.String(),
		ApproveAddr:      "",
		OperatorMetaInfo: "test operator",
		ClientChainEarningsAddr: &operatortype.ClientChainEarningAddrList{
			EarningInfoList: []*operatortype.ClientChainEarningAddrInfo{
				{defaultClientChainID, "0x1f9840a85d5af5bf1d1762f925bdaddc4201f984"},
			},
		},
		Commission: stakingtypes.NewCommission(math.LegacyZeroDec(), math.LegacyZeroDec(), math.LegacyZeroDec()),
	}
	suite.Equal(operatortype.AccAddressLength, len(suite.AccAddress))
	err := suite.App.OperatorKeeper.SetOperatorInfo(suite.Ctx, suite.AccAddress.String(), info)
	suite.NoError(err)

	getOperatorInfo, err := suite.App.OperatorKeeper.QueryOperatorInfo(suite.Ctx, &operatortype.GetOperatorInfoReq{OperatorAddr: suite.AccAddress.String()})
	suite.NoError(err)
	suite.Equal(*info, *getOperatorInfo)
}

func (suite *OperatorTestSuite) TestAllOperators() {
	suite.prepare()
	operatorDetail := operatortype.OperatorDetail{
		OperatorAddress: suite.AccAddress.String(),
		OperatorInfo: operatortype.OperatorInfo{
			EarningsAddr:     suite.AccAddress.String(),
			OperatorMetaInfo: "testOperator",
			Commission:       stakingtypes.NewCommission(sdk.ZeroDec(), sdk.ZeroDec(), sdk.ZeroDec()),
		},
	}
	err := suite.App.OperatorKeeper.SetOperatorInfo(suite.Ctx, suite.AccAddress.String(), &operatorDetail.OperatorInfo)
	suite.NoError(err)

	getOperators := suite.App.OperatorKeeper.AllOperators(suite.Ctx)
	suite.Contains(getOperators, operatorDetail)
}

// TODO: enable this test when editing operator is implemented. allow for querying
// of the old commission against the new one.
// func (suite *OperatorTestSuite) TestHistoricalOperatorInfo() {
// 	height := suite.Ctx.BlockHeight()
// 	info := &operatortype.OperatorInfo{
// 		EarningsAddr:     suite.AccAddress.String(),
// 		ApproveAddr:      "",
// 		OperatorMetaInfo: "test operator",
// 		ClientChainEarningsAddr: &operatortype.ClientChainEarningAddrList{
// 			EarningInfoList: nil,
// 		},
// 	}
// 	err := suite.App.OperatorKeeper.SetOperatorInfo(suite.Ctx, suite.AccAddress.String(), info)
// 	suite.NoError(err)
// 	suite.NextBlock()
// 	suite.Equal(height+1, suite.Ctx.BlockHeight(), "nexBlock failed")

// 	newInfo := *info
// 	newInfo.OperatorMetaInfo = "new operator"
// 	err = suite.App.OperatorKeeper.SetOperatorInfo(suite.Ctx, suite.AccAddress.String(), &newInfo)
// 	suite.NoError(err)

// 	for i := 0; i < 10; i++ {
// 		suite.NextBlock()
// 	}
// 	// get historical operator info
// 	historicalQueryCtx, err := suite.App.CreateQueryContext(height, false)
// 	suite.NoError(err)
// 	getInfo, err := suite.App.OperatorKeeper.QueryOperatorInfo(historicalQueryCtx, &operatortype.GetOperatorInfoReq{
// 		OperatorAddr: suite.AccAddress.String(),
// 	})
// 	suite.NoError(err)
// 	suite.Equal(info.OperatorMetaInfo, getInfo.OperatorMetaInfo)

// 	getInfo, err = suite.App.OperatorKeeper.QueryOperatorInfo(suite.Ctx, &operatortype.GetOperatorInfoReq{
// 		OperatorAddr: suite.AccAddress.String(),
// 	})
// 	suite.NoError(err)
// 	suite.Equal(newInfo.OperatorMetaInfo, getInfo.OperatorMetaInfo)
// }
