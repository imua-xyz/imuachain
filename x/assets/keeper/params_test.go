package keeper_test

import (
	assetstype "github.com/ExocoreNetwork/exocore/x/assets/types"
)

func (suite *StakingAssetsTestSuite) TestParams() {
	params := &assetstype.Params{
		Gateways: []string{"0x3fC91A3afd70395Cd496C647d5a6CC9D4B2b7FAD"},
	}
	err := suite.App.AssetsKeeper.SetParams(suite.Ctx, params)
	suite.NoError(err)

	getParams, err := suite.App.AssetsKeeper.GetParams(suite.Ctx)
	suite.NoError(err)
	suite.Equal(*params, *getParams)
}
