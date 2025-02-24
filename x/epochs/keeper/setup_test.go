package keeper_test

import (
	"testing"

	"github.com/imua-xyz/imuachain/testutil"
	"github.com/stretchr/testify/suite"

	"github.com/imua-xyz/imuachain/x/epochs/types"
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
	identifiers := []string{
		types.DayEpochID, types.HourEpochID, types.MinuteEpochID, types.WeekEpochID,
	}
	for _, identifier := range identifiers {
		epoch, found := suite.App.EpochsKeeper.GetEpochInfo(suite.Ctx, identifier)
		suite.Require().True(found)
		suite.Require().NotZero(epoch.CurrentEpochStartHeight)
	}
	suite.PostSetup()
}
