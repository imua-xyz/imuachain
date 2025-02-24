package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/imua-xyz/imuachain/x/immint/types"
)

func (suite *KeeperTestSuite) TestQueryParams() {
	defaultParams := types.DefaultParams()
	res, err := suite.queryClient.Params(sdk.WrapSDKContext(suite.Ctx), &types.QueryParamsRequest{})
	suite.Require().NoError(err)
	suite.Require().NotNil(res)
	suite.Require().Equal(defaultParams, res.Params)
}
