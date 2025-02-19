package keeper_test

import (
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	blscommon "github.com/prysmaticlabs/prysm/v4/crypto/bls/common"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/evmos/evmos/v16/app"
	utiltx "github.com/evmos/evmos/v16/testutil/tx"
	evm "github.com/evmos/evmos/v16/x/evm/types"

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

	ctx               sdk.Context
	app               *app.Evmos
	queryClientEvm    evm.QueryClient
	consAddress       sdk.ConsAddress
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
		"exo1ve9s2u8c7u44la93pen79hwdd4zse2zku73cjp",
		"exo1edwpx7243z5ls7qehmzwwsnnvtm8ms0dgr6ukq",
		"exo1x28fd5v0mxjpevll60j5lf2jz4ksrpsdvck43r",
		"exo1pkeqsekm0wsu4d5wqntf32t9l0sn35xquk65kz",
		"exo1wsqzfdkmv5a4wu7788uw7zjaqfj6rcrm7q69dg",
	}
}
