package keeper_test

import (
	"testing"

	"github.com/imua-xyz/imuachain/testutil"
	"github.com/stretchr/testify/suite"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/imua-xyz/imuachain/x/exomint/types"
)

var s *KeeperTestSuite

type KeeperTestSuite struct {
	testutil.BaseTestSuite
	queryClient types.QueryClient
}

func TestKeeperTestSuite(t *testing.T) {
	s = new(KeeperTestSuite)
	suite.Run(t, s)
}

func (suite *KeeperTestSuite) SetupTest() {
	suite.DoSetupTest()
	queryHelper := baseapp.NewQueryServerTestHelper(suite.Ctx, suite.App.InterfaceRegistry())
	types.RegisterQueryServer(queryHelper, suite.App.ExomintKeeper)
	suite.queryClient = types.NewQueryClient(queryHelper)
}
