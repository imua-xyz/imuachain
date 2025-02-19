package keeper_test

import (
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/imua-xyz/imuachain/x/epochs/types"
)

// Test helpers
func (suite *KeeperTestSuite) PostSetup() {
	// setup query helpers
	queryHelper := baseapp.NewQueryServerTestHelper(suite.Ctx, suite.App.InterfaceRegistry())
	types.RegisterQueryServer(queryHelper, suite.App.EpochsKeeper)
	suite.queryClient = types.NewQueryClient(queryHelper)
}
