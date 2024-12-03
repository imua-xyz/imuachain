package e2e

import (
	"github.com/ExocoreNetwork/exocore/testutil/network"
	"github.com/stretchr/testify/suite"
)

type BaseSuite struct {
	suite.Suite

	Cfg     network.Config
	Network *network.Network
}

func NewBaseSuite(cfg network.Config) *BaseSuite {
	return &BaseSuite{Cfg: cfg}
}

func (s *BaseSuite) SetupSuite() {
	s.T().Log("setting up e2e test suite")
	var err error
	s.Network, err = network.New(s.T(), s.T().TempDir(), s.Cfg)
	s.Require().NoError(err)
	_, err = s.Network.WaitForHeight(2)
	s.Require().NoError(err)
}
