package keeper_test

import (
	"strings"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/imua-xyz/imuachain/testutil"
	operatortypes "github.com/imua-xyz/imuachain/x/operator/types"
	"github.com/stretchr/testify/suite"
)

type EditOperatorTestSuite struct {
	testutil.BaseTestSuite
}

func (suite *EditOperatorTestSuite) SetupTest() {
}

func TestEditOperatorTestSuite(t *testing.T) {
	suite.Run(t, new(EditOperatorTestSuite))
}

func (suite *EditOperatorTestSuite) TestEditOperator() {
	// same name again
	suite.DoSetupTest()
	registerReq := &operatortypes.RegisterOperatorReq{
		FromAddress: suite.AccAddress.String(),
		Info: &operatortypes.OperatorInfo{
			EarningsAddr:     suite.AccAddress.String(),
			ApproveAddr:      suite.AccAddress.String(),
			OperatorMetaInfo: "operator1",
			Commission: stakingtypes.Commission{
				CommissionRates: stakingtypes.CommissionRates{
					Rate:          sdk.ZeroDec(),
					MaxRate:       sdk.OneDec(),
					MaxChangeRate: sdk.OneDec(),
				},
			},
		},
	}
	_, err := suite.OperatorMsgServer.RegisterOperator(suite.Ctx, registerReq)
	suite.Require().ErrorAs(err, &operatortypes.ErrOperatorNameAlreadyExists)
	suite.Commit()
	// next name
	registerReq.Info.OperatorMetaInfo = "operator3"
	_, err = suite.OperatorMsgServer.RegisterOperator(suite.Ctx, registerReq)
	suite.Require().NoError(err)
	suite.Commit()
	// now edit it but keep the name same
	editReq := &operatortypes.EditOperatorReq{
		Address:          suite.AccAddress.String(),
		OperatorMetaInfo: "operator3",
	}
	_, err = suite.OperatorMsgServer.EditOperator(suite.Ctx, editReq)
	suite.Require().ErrorAs(err, &operatortypes.ErrOperatorNameAlreadyExists)
	suite.Commit()
	// now a new name
	editReq.OperatorMetaInfo = "operator4"
	_, err = suite.OperatorMsgServer.EditOperator(suite.Ctx, editReq)
	suite.Require().NoError(err)
	suite.Commit()
	// check the info
	operatorInfo, err := suite.App.OperatorKeeper.OperatorInfo(
		suite.Ctx, suite.AccAddress.String(),
	)
	suite.Require().NoError(err)
	suite.Require().Equal("operator4", operatorInfo.OperatorMetaInfo)
	// change to a large name
	editReq.OperatorMetaInfo = strings.Repeat("a", stakingtypes.MaxMonikerLength+1)
	_, err = suite.OperatorMsgServer.EditOperator(suite.Ctx, editReq)
	suite.Require().ErrorAs(err, &operatortypes.ErrParameterInvalid)
	suite.Require().Contains(err.Error(), "info length exceeds")
	// change to a nil name
	editReq.OperatorMetaInfo = ""
	_, err = suite.OperatorMsgServer.EditOperator(suite.Ctx, editReq)
	suite.Require().ErrorAs(err, &operatortypes.ErrParameterInvalid)
	suite.Require().Contains(err.Error(), "operator meta info is empty")
}

func (suite *EditOperatorTestSuite) TestRegisterOperator() {
	// large name
	suite.DoSetupTest()
	registerReq := &operatortypes.RegisterOperatorReq{
		FromAddress: suite.AccAddress.String(),
		Info: &operatortypes.OperatorInfo{
			EarningsAddr:     suite.AccAddress.String(),
			ApproveAddr:      suite.AccAddress.String(),
			OperatorMetaInfo: strings.Repeat("a", stakingtypes.MaxMonikerLength+1),
			Commission: stakingtypes.Commission{
				CommissionRates: stakingtypes.CommissionRates{
					Rate:          sdk.ZeroDec(),
					MaxRate:       sdk.OneDec(),
					MaxChangeRate: sdk.OneDec(),
				},
			},
		},
	}
	_, err := suite.OperatorMsgServer.RegisterOperator(suite.Ctx, registerReq)
	suite.Require().ErrorAs(err, &operatortypes.ErrParameterInvalid)
	suite.Require().Contains(err.Error(), "info length exceeds")
	// nil name
	registerReq.Info.OperatorMetaInfo = ""
	_, err = suite.OperatorMsgServer.RegisterOperator(suite.Ctx, registerReq)
	suite.Require().ErrorAs(err, &operatortypes.ErrParameterInvalid)
	suite.Require().Contains(err.Error(), "operator meta info is empty")
	// real name
	registerReq.Info.OperatorMetaInfo = "operator3"
	_, err = suite.OperatorMsgServer.RegisterOperator(suite.Ctx, registerReq)
	suite.Require().NoError(err)
	suite.Commit()
}
