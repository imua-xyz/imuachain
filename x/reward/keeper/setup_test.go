package keeper_test

import (
	"testing"

	"github.com/imua-xyz/imuachain/testutil"

	"github.com/stretchr/testify/suite"
)

type RewardTestSuite struct {
	testutil.BaseTestSuite
}

var s *RewardTestSuite

func TestKeeperTestSuite(t *testing.T) {
	s = new(RewardTestSuite)
	suite.Run(t, s)
}

// SetupTest setup test environment, it uses`require.TestingT` to support both `testing.T` and `testing.B`.
func (suite *RewardTestSuite) SetupTest() {
	suite.DoSetupTest()
}
