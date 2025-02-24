package keeper_test

import (
	"testing"

	"github.com/imua-xyz/imuachain/testutil"

	"github.com/stretchr/testify/suite"
)

type SlashTestSuite struct {
	testutil.BaseTestSuite
}

var s *SlashTestSuite

func TestKeeperTestSuite(t *testing.T) {
	s = new(SlashTestSuite)
	suite.Run(t, s)
}

// SetupTest setup test environment, it uses`require.TestingT` to support both `testing.T` and `testing.B`.
func (suite *SlashTestSuite) SetupTest() {
	suite.DoSetupTest()
}
