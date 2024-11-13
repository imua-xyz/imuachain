package oracle

import (
	"testing"

	"github.com/ExocoreNetwork/exocore/testutil/network"
	"github.com/stretchr/testify/suite"
)

func TestE2ESuite(t *testing.T) {
	cfg := network.DefaultConfig()
	cfg.NumValidators = 4
	cfg.CleanupDir = true
	cfg.EnableTMLogging = true
	suite.Run(t, NewE2ETestSuite(cfg))
}
