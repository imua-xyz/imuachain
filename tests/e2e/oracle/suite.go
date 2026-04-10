package oracle

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/imua-xyz/imuachain/testutil/network"
	"github.com/stretchr/testify/suite"
)

type CreatePriceSuite struct {
	suite.Suite

	cfg     network.Config
	network *network.Network
}

func NewCreatePriceSuite(cfg network.Config) *CreatePriceSuite {
	return &CreatePriceSuite{cfg: cfg}
}

func (s *CreatePriceSuite) SetupSuite() {
	s.T().Log("setting up e2e test suite")
	var err error
	s.network, err = network.New(s.T(), s.T().TempDir(), s.cfg)
	s.Require().NoError(err)
	_, err = s.network.WaitForHeightWithTimeout(2, 20*time.Second)
	s.Require().NoError(err)
}

func (s *CreatePriceSuite) TearDownSuite() {
	s.network.Cleanup()
}

type XChainTestSuite struct {
	suite.Suite

	cfg         network.Config
	network     *network.Network
	gatewayAddr common.Address
}

func NewXChainTestSuite(cfg network.Config) *XChainTestSuite {
	return &XChainTestSuite{cfg: cfg}
}

func (s *XChainTestSuite) SetupSuite() {
	s.T().Log("setting up xchain e2e test suite")
	var err error
	s.network, err = network.New(s.T(), s.T().TempDir(), s.cfg)
	s.Require().NoError(err)
	_, err = s.network.WaitForHeightWithTimeout(2, 20*time.Second)
	s.Require().NoError(err)
}

func (s *XChainTestSuite) TearDownSuite() {
	s.network.Cleanup()
}
