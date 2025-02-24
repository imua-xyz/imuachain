package keeper_test

import (
	"testing"
	"time"

	"github.com/imua-xyz/imuachain/testutil"
	"github.com/stretchr/testify/suite"
)

var s *KeeperTestSuite

type KeeperTestSuite struct {
	testutil.BaseTestSuite
	EpochDuration time.Duration
}

func TestKeeperTestSuite(t *testing.T) {
	s = new(KeeperTestSuite)
	suite.Run(t, s)
}

func (suite *KeeperTestSuite) SetupTest() {
	suite.DoSetupTest()
	epochID := suite.App.StakingKeeper.GetEpochIdentifier(suite.Ctx)
	epochInfo, _ := suite.App.EpochsKeeper.GetEpochInfo(suite.Ctx, epochID)
	suite.EpochDuration = epochInfo.Duration + time.Nanosecond // extra buffer
}
