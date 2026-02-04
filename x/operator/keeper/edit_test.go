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
			OperatorAddr: suite.AccAddress.String(),
			Description:  stakingtypes.NewDescription("operator1", "", "", "", ""),
			Commission:   stakingtypes.NewCommission(sdk.ZeroDec(), sdk.OneDec(), sdk.OneDec()),
		},
	}
	_, err := suite.OperatorMsgServer.RegisterOperator(suite.Ctx, registerReq)
	suite.Require().ErrorAs(err, &operatortypes.ErrOperatorNameAlreadyExists)
	suite.Commit()
	// next name
	registerReq.Info.Description.Moniker = "operator3"
	_, err = suite.OperatorMsgServer.RegisterOperator(suite.Ctx, registerReq)
	suite.Require().NoError(err)
	suite.Commit()
	// now edit it but keep the name same
	editReq := &operatortypes.EditOperatorReq{
		Address:     suite.AccAddress.String(),
		Description: stakingtypes.NewDescription("operator3", "", "", "", ""),
	}
	_, err = suite.OperatorMsgServer.EditOperator(suite.Ctx, editReq)
	suite.Require().ErrorAs(err, &operatortypes.ErrOperatorNameAlreadyExists)
	suite.Commit()
	// now a new name
	editReq.Description.Moniker = "operator4"
	_, err = suite.OperatorMsgServer.EditOperator(suite.Ctx, editReq)
	suite.Require().NoError(err)
	suite.Commit()
	// check the info
	operatorInfo, err := suite.App.OperatorKeeper.OperatorInfo(
		suite.Ctx, suite.AccAddress.String(),
	)
	suite.Require().NoError(err)
	suite.Require().Equal("operator4", operatorInfo.Description.Moniker)
	// change to a large name
	editReq.Description.Moniker = strings.Repeat("a", stakingtypes.MaxMonikerLength+1)
	err = editReq.ValidateBasic()
	suite.Require().ErrorAs(err, &operatortypes.ErrParameterInvalid)
	suite.Require().Contains(err.Error(), "invalid moniker length")
	// change to a nil name
	editReq.Description.Moniker = ""
	err = editReq.ValidateBasic()
	suite.Require().ErrorAs(err, &operatortypes.ErrParameterInvalid)
	suite.Require().Contains(err.Error(), "empty description")
}

func (suite *EditOperatorTestSuite) TestRegisterOperator() {
	// large name
	suite.DoSetupTest()
	registerReq := &operatortypes.RegisterOperatorReq{
		FromAddress: suite.AccAddress.String(),
		Info: &operatortypes.OperatorInfo{
			OperatorAddr: suite.AccAddress.String(),
			Description:  stakingtypes.NewDescription(strings.Repeat("a", stakingtypes.MaxMonikerLength+1), "", "", "", ""),
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
	suite.Require().Contains(err.Error(), "invalid moniker length")
	// nil name
	registerReq.Info.Description.Moniker = ""
	_, err = suite.OperatorMsgServer.RegisterOperator(suite.Ctx, registerReq)
	suite.Require().ErrorAs(err, &operatortypes.ErrParameterInvalid)
	suite.Require().Contains(err.Error(), "operator moniker is empty")
	// real name
	registerReq.Info.Description.Moniker = "operator3"
	_, err = suite.OperatorMsgServer.RegisterOperator(suite.Ctx, registerReq)
	suite.Require().NoError(err)
	suite.Commit()
}
