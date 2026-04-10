package oracle

import (
	"testing"

	"github.com/imua-xyz/imuachain/testutil/network"
	"github.com/stretchr/testify/suite"
)

func TestCreatePriceSuite(t *testing.T) {
	cfg := network.DefaultConfig()
	cfg.NumValidators = 4
	cfg.CleanupDir = true
	cfg.EnableTMLogging = true
	suite.Run(t, NewCreatePriceSuite(cfg))
}
