package oracle

import (
	"time"

	"github.com/imua-xyz/imuachain/testutil/network"
	"github.com/stretchr/testify/suite"
)

type E2ETestSuite struct {
	suite.Suite

	cfg     network.Config
	network *network.Network
}

func NewE2ETestSuite(cfg network.Config) *E2ETestSuite {
	return &E2ETestSuite{cfg: cfg}
}

func (s *E2ETestSuite) SetupSuite() {
	s.T().Log("setting up e2e test suite")
	var err error
	s.network, err = network.New(s.T(), s.T().TempDir(), s.cfg)
	s.Require().NoError(err)
	_, err = s.network.WaitForHeightWithTimeout(2, 20*time.Second)
	s.Require().NoError(err)
}

func (s *E2ETestSuite) TearDownSuite() {
	s.network.Cleanup()
}
