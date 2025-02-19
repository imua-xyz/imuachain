package keeper_test

import (
	rewardtype "github.com/imua-xyz/imuachain/x/reward/types"
)

func (suite *RewardTestSuite) TestParams() {
	params := &rewardtype.Params{}
	err := suite.App.RewardKeeper.SetParams(suite.Ctx, params)
	suite.NoError(err)

	getParams, err := suite.App.RewardKeeper.GetParams(suite.Ctx)
	suite.NoError(err)
	suite.Equal(*params, *getParams)
}
