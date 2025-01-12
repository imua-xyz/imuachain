package keeper_test

import (
	"fmt"
	"time"

	"github.com/ExocoreNetwork/exocore/testutil"

	"cosmossdk.io/math"

	delegationtypes "github.com/ExocoreNetwork/exocore/x/delegation/types"
	epochsTypes "github.com/ExocoreNetwork/exocore/x/epochs/types"
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
	suite.Equal(delegationtypes.AccAddressLength, len(suite.AccAddress))
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

func (suite *OperatorTestSuite) TestGetUnbondingExpiration() {
	suite.prepare()
	epochIdentifier, epochNumber, err := suite.App.OperatorKeeper.GetUnbondingExpiration(suite.Ctx, suite.operatorAddr)
	suite.NoError(err)
	suite.Equal(epochsTypes.NullEpochIdentifier, epochIdentifier)
	suite.Equal(epochsTypes.NullEpochNumber, epochNumber)

	// opts into multiple AVSs
	testAVSNumber := 4
	for i := 0; i < testAVSNumber; i++ {
		avsName := fmt.Sprintf("avsTestAddr_%d", i)
		suite.prepareAvs(avsName, []string{usdtAssetID}, testutil.EpochsForTest[i], defaultUnbondingPeriod)
		err = suite.App.OperatorKeeper.OptIn(suite.Ctx, suite.operatorAddr, suite.avsAddr)
		suite.NoError(err)
	}
	weekEpochInfo, found := suite.App.EpochsKeeper.GetEpochInfo(suite.Ctx, epochsTypes.WeekEpochID)
	suite.True(found)

	epochIdentifier, epochNumber, err = suite.App.OperatorKeeper.GetUnbondingExpiration(suite.Ctx, suite.operatorAddr)
	suite.NoError(err)
	suite.Equal(epochsTypes.WeekEpochID, epochIdentifier)
	suite.Equal(uint64(weekEpochInfo.CurrentEpoch)+defaultUnbondingPeriod, uint64(epochNumber))

	// register an AVS where the epoch identifier is by minute, but the unbonding duration is greater
	// than all the above test AVSs.
	minuteEpochInfo, found := suite.App.EpochsKeeper.GetEpochInfo(suite.Ctx, epochsTypes.MinuteEpochID)
	suite.True(found)
	avsName := fmt.Sprintf("avsTestAddr_%d", testAVSNumber+1)
	minuteUnbondingPeriod := defaultUnbondingPeriod*uint64(weekEpochInfo.Duration.Milliseconds()/minuteEpochInfo.Duration.Milliseconds()) + 1
	suite.prepareAvs(avsName, []string{usdtAssetID}, epochsTypes.MinuteEpochID, minuteUnbondingPeriod)
	err = suite.App.OperatorKeeper.OptIn(suite.Ctx, suite.operatorAddr, suite.avsAddr)
	suite.NoError(err)
	epochIdentifier, epochNumber, err = suite.App.OperatorKeeper.GetUnbondingExpiration(suite.Ctx, suite.operatorAddr)
	suite.NoError(err)
	suite.Equal(epochsTypes.MinuteEpochID, epochIdentifier)
	suite.Equal(uint64(minuteEpochInfo.CurrentEpoch)+minuteUnbondingPeriod, uint64(epochNumber))

	// test the case where the operator opts in and out at same epoch
	err = suite.App.OperatorKeeper.OptOut(suite.Ctx, suite.operatorAddr, suite.avsAddr)
	suite.NoError(err)
	epochIdentifier, epochNumber, err = suite.App.OperatorKeeper.GetUnbondingExpiration(suite.Ctx, suite.operatorAddr)
	suite.NoError(err)
	suite.Equal(epochsTypes.WeekEpochID, epochIdentifier)
	suite.Equal(uint64(weekEpochInfo.CurrentEpoch)+defaultUnbondingPeriod, uint64(epochNumber))

	// test the case where the operator opts out but is still within the unbonding duration
	err = suite.App.OperatorKeeper.OptIn(suite.Ctx, suite.operatorAddr, suite.avsAddr)
	suite.NoError(err)
	suite.CommitAfter(time.Minute + time.Nanosecond)
	err = suite.App.OperatorKeeper.OptOut(suite.Ctx, suite.operatorAddr, suite.avsAddr)
	suite.NoError(err)
	epochIdentifier, epochNumber, err = suite.App.OperatorKeeper.GetUnbondingExpiration(suite.Ctx, suite.operatorAddr)
	suite.NoError(err)
	suite.Equal(epochsTypes.MinuteEpochID, epochIdentifier)
	minuteEpochInfo, found = suite.App.EpochsKeeper.GetEpochInfo(suite.Ctx, epochsTypes.MinuteEpochID)
	suite.True(found)
	suite.Equal(uint64(minuteEpochInfo.CurrentEpoch)+minuteUnbondingPeriod, uint64(epochNumber))

	// test the case where the operator has opted out for a period, then it's unbonding duration won't be
	// the maximum value.
	err = suite.App.OperatorKeeper.OptIn(suite.Ctx, suite.operatorAddr, suite.avsAddr)
	suite.NoError(err)
	suite.CommitAfter(time.Minute + time.Nanosecond)
	err = suite.App.OperatorKeeper.OptOut(suite.Ctx, suite.operatorAddr, suite.avsAddr)
	suite.NoError(err)
	suite.CommitAfter(2*time.Minute + time.Nanosecond)
	epochIdentifier, epochNumber, err = suite.App.OperatorKeeper.GetUnbondingExpiration(suite.Ctx, suite.operatorAddr)
	suite.NoError(err)
	suite.Equal(epochsTypes.WeekEpochID, epochIdentifier)
	weekEpochInfo, found = suite.App.EpochsKeeper.GetEpochInfo(suite.Ctx, epochsTypes.WeekEpochID)
	suite.True(found)
	suite.Equal(uint64(weekEpochInfo.CurrentEpoch)+defaultUnbondingPeriod, uint64(epochNumber))
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
