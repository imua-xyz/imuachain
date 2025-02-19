package keeper_test

import (
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	blscommon "github.com/prysmaticlabs/prysm/v4/crypto/bls/common"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	utiltx "github.com/evmos/evmos/v16/testutil/tx"
	"github.com/imua-xyz/imuachain/testutil"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/suite"
)

type AVSTestSuite struct {
	testutil.BaseTestSuite

	// needed by test
	operatorAddr          sdk.AccAddress
	avsAddr               string
	assetID               string
	stakerID              string
	assetAddr             common.Address
	assetDecimal          uint32
	clientChainLzID       uint64
	depositAmount         sdkmath.Int
	delegationAmount      sdkmath.Int
	updatedAmountForOptIn sdkmath.Int

	avsAddress        common.Address
	taskAddress       common.Address
	taskId            uint64
	blsKey            blscommon.SecretKey
	EpochDuration     time.Duration
	operatorAddresses []string
	blsKeys           []blscommon.SecretKey
}

var s *AVSTestSuite

func TestKeeperTestSuite(t *testing.T) {
	s = new(AVSTestSuite)
	suite.Run(t, s)

	// Run Ginkgo integration tests
	RegisterFailHandler(Fail)
	RunSpecs(t, "Keeper Suite")
}

func (suite *AVSTestSuite) SetupTest() {
	suite.DoSetupTest()
	suite.avsAddress = utiltx.GenerateAddress()
	suite.taskAddress = utiltx.GenerateAddress()
	epochID := suite.App.StakingKeeper.GetEpochIdentifier(suite.Ctx)
	epochInfo, _ := suite.App.EpochsKeeper.GetEpochInfo(suite.Ctx, epochID)
	suite.EpochDuration = epochInfo.Duration + time.Nanosecond // extra buffer
	suite.operatorAddresses = []string{
		sdk.AccAddress(utiltx.GenerateAddress().Bytes()).String(),
		sdk.AccAddress(utiltx.GenerateAddress().Bytes()).String(),
		sdk.AccAddress(utiltx.GenerateAddress().Bytes()).String(),
		sdk.AccAddress(utiltx.GenerateAddress().Bytes()).String(),
		sdk.AccAddress(utiltx.GenerateAddress().Bytes()).String(),
	}
}
