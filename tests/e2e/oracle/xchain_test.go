package oracle

import (
	"testing"

	"github.com/imua-xyz/imuachain/testutil/network"
	"github.com/stretchr/testify/suite"
)

func TestXChainSuite(t *testing.T) {
	ensureXChainGatewayGenesis()
	cfg := network.DefaultConfig()
	cfg.NumValidators = 3
	cfg.CleanupDir = true
	cfg.EnableTMLogging = true
	oracleGen := PrepareXChainOracleGenesis(&cfg)
	cfg.OracleGenesisState = &oracleGen
	suite.Run(t, NewXChainTestSuite(cfg))
}
