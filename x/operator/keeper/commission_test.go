package keeper_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/imua-xyz/imuachain/testutil"
	operatortypes "github.com/imua-xyz/imuachain/x/operator/types"
	"github.com/stretchr/testify/suite"
)

type CommissionTestSuite struct {
	testutil.BaseTestSuite
}

func (suite *CommissionTestSuite) SetupTest() {
}

func TestCommissionTestSuite(t *testing.T) {
	suite.Run(t, new(CommissionTestSuite))
}

func (suite *CommissionTestSuite) TestCommissionRateTimeBound() {
	suite.DoSetupTest()
	// register operator
	suite.RegisterOperator(suite.AccAddress.String(), stakingtypes.NewCommission(sdk.ZeroDec(), sdk.OneDec(), sdk.OneDec()), true)
	suite.Commit()
	// change to 6% immediately
	targetCommissionRate := sdk.NewDecWithPrec(6, 2)
	updateCommissionReq := &operatortypes.UpdateCommissionRateReq{
		Address:        suite.AccAddress.String(),
		CommissionRate: targetCommissionRate,
	}
	_, err := suite.OperatorMsgServer.UpdateCommissionRate(suite.Ctx, updateCommissionReq)
	suite.Require().ErrorAs(err, &stakingtypes.ErrCommissionUpdateTime)
	// wait for the minimum update interval
	duration := suite.App.OperatorKeeper.GetMinCommissionUpdateInterval(suite.Ctx)
	suite.CommitAfter(duration + time.Nanosecond)
	// change to 6%
	_, err = suite.OperatorMsgServer.UpdateCommissionRate(suite.Ctx, updateCommissionReq)
	suite.Require().NoError(err)
	suite.Commit()
	// check the info
	operatorInfo, err := suite.App.OperatorKeeper.OperatorInfo(
		suite.Ctx, suite.AccAddress.String(),
	)
	suite.Require().NoError(err)
	suite.Require().Equal(targetCommissionRate, operatorInfo.Commission.CommissionRates.Rate)
	suite.Require().Equal(sdk.OneDec(), operatorInfo.Commission.CommissionRates.MaxRate)
	suite.Require().Equal(sdk.OneDec(), operatorInfo.Commission.CommissionRates.MaxChangeRate)
	// now try to change at exactly the minimum rate
	suite.CommitAfter(duration + time.Nanosecond)
	updateCommissionReq.CommissionRate = sdk.NewDecWithPrec(5, 2)
	_, err = suite.OperatorMsgServer.UpdateCommissionRate(suite.Ctx, updateCommissionReq)
	suite.Require().NoError(err)
	suite.Commit()
	// now try to change below the minimum rate
	suite.CommitAfter(duration + time.Nanosecond)
	updateCommissionReq.CommissionRate = sdk.NewDecWithPrec(4, 2)
	_, err = suite.OperatorMsgServer.UpdateCommissionRate(suite.Ctx, updateCommissionReq)
	suite.Require().ErrorAs(err, &stakingtypes.ErrCommissionLTMinRate)
}

func (suite *CommissionTestSuite) TestCommissionRateChange() {
	suite.DoSetupTest()
	// register operator
	suite.RegisterOperator(suite.AccAddress.String(), stakingtypes.Commission{
		CommissionRates: stakingtypes.CommissionRates{
			Rate:          sdk.ZeroDec(),
			MaxRate:       sdk.NewDecWithPrec(75, 2),
			MaxChangeRate: sdk.NewDecWithPrec(5, 2),
		},
	}, true)
	duration := suite.App.OperatorKeeper.GetMinCommissionUpdateInterval(suite.Ctx)
	// from 0 we can go to 5% but not higher
	// let's try 5.1%
	suite.CommitAfter(duration + time.Nanosecond)
	updateCommissionReq := &operatortypes.UpdateCommissionRateReq{
		Address:        suite.AccAddress.String(),
		CommissionRate: sdk.NewDecWithPrec(51, 2),
	}
	_, err := suite.OperatorMsgServer.UpdateCommissionRate(suite.Ctx, updateCommissionReq)
	suite.Require().ErrorAs(err, &stakingtypes.ErrCommissionGTMaxChangeRate)
	// now try 5%
	updateCommissionReq.CommissionRate = sdk.NewDecWithPrec(5, 2)
	_, err = suite.OperatorMsgServer.UpdateCommissionRate(suite.Ctx, updateCommissionReq)
	suite.Require().NoError(err)
	suite.Commit()
	info, err := suite.App.OperatorKeeper.OperatorInfo(
		suite.Ctx, suite.AccAddress.String(),
	)
	suite.Require().NoError(err)
	suite.Require().Equal(sdk.NewDecWithPrec(5, 2), info.Commission.CommissionRates.Rate)
	for i := 0; i <= (75-5)/5; i++ {
		suite.CommitAfter(duration + time.Nanosecond)
		updateCommissionReq.CommissionRate = sdk.NewDecWithPrec(int64(5+5*i), 2)
		_, err = suite.OperatorMsgServer.UpdateCommissionRate(suite.Ctx, updateCommissionReq)
		suite.Require().NoError(err)
		suite.Commit()
	}
	// check value is 75% now
	info, err = suite.App.OperatorKeeper.OperatorInfo(
		suite.Ctx, suite.AccAddress.String(),
	)
	suite.Require().NoError(err)
	suite.Require().Equal(sdk.NewDecWithPrec(75, 2), info.Commission.CommissionRates.Rate)
	suite.CommitAfter(duration + time.Nanosecond)
	// now try to change to 76%
	updateCommissionReq.CommissionRate = sdk.NewDecWithPrec(76, 2)
	_, err = suite.OperatorMsgServer.UpdateCommissionRate(suite.Ctx, updateCommissionReq)
	suite.Require().ErrorAs(err, &stakingtypes.ErrCommissionGTMaxRate)
}
