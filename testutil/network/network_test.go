package network_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/ExocoreNetwork/exocore/testutil/network"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/evmos/evmos/v16/server/config"

	exocorenetwork "github.com/ExocoreNetwork/exocore/testutil/network"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

type IntegrationTestSuite struct {
	suite.Suite

	network *network.Network
}

func (s *IntegrationTestSuite) SetupSuite() {
	s.T().Log("setting up integration test suite")

	var err error
	cfg := exocorenetwork.DefaultConfig()
	cfg.JSONRPCAddress = config.DefaultJSONRPCAddress
	cfg.NumValidators = 3
	cfg.CleanupDir = true
	cfg.EnableTMLogging = false

	s.network, err = network.New(s.T(), s.T().TempDir(), cfg)
	s.Require().NoError(err)
	s.Require().NotNil(s.network)

	_, err = s.network.WaitForHeightWithTimeout(2, 300*time.Second)
	s.Require().NoError(err)

	if s.network.Validators[0].JSONRPCClient == nil {
		address := fmt.Sprintf("http://%s", s.network.Validators[0].AppConfig.JSONRPC.Address)
		s.network.Validators[0].JSONRPCClient, err = ethclient.Dial(address)
		s.Require().NoError(err)
	}
}

func (s *IntegrationTestSuite) TearDownSuite() {
	s.T().Log("tearing down integration test suite")
	s.network.Cleanup()
}

func (s *IntegrationTestSuite) TestNetwork_Liveness() {
	h, err := s.network.WaitForHeightWithTimeout(10, time.Minute)
	s.Require().NoError(err, "expected to reach 10 blocks; got %d", h)

	latestHeight, err := s.network.LatestHeight()
	s.Require().NoError(err, "latest height failed")
	s.Require().GreaterOrEqual(latestHeight, h)

	res, err := s.network.QueryBank().Balance(context.Background(), &banktypes.QueryBalanceRequest{
		Address: s.network.Validators[0].Address.String(),
		Denom:   s.network.Config.NativeDenom,
	})
	s.Require().NoError(err, "query for validator's balance fail")

	s.Require().Equal(sdk.NewCoin(s.network.Config.NativeDenom, s.network.Config.AccountTokens), *res.Balance)
}

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}
