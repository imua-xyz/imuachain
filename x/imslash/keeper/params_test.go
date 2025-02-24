package keeper_test

import (
	slashtype "github.com/imua-xyz/imuachain/x/imslash/types"
)

func (suite *SlashTestSuite) TestParams() {
	params := &slashtype.Params{}
	err := suite.App.ImSlashKeeper.SetParams(suite.Ctx, params)
	suite.NoError(err)

	getParams, err := suite.App.ImSlashKeeper.GetParams(suite.Ctx)
	suite.NoError(err)
	suite.Equal(*params, *getParams)
}
