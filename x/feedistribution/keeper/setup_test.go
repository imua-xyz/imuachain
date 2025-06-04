package keeper_test

import (
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/imua-xyz/imuachain/testutil"
	"github.com/stretchr/testify/suite"
)

var s *KeeperTestSuite

type KeeperTestSuite struct {
	testutil.BaseTestSuite
	EpochDuration time.Duration

	// Used for test
	testOperators []types.AccAddress
	testAVSs      []common.Address
	// Use the default EVM-compatible client chain for testing stakers.
	testStakers       []common.Address
	testStakerIDs     []string
	testClientChainID uint64
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
