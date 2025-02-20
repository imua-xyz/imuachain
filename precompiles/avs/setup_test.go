package avs_test

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/imua-xyz/imuachain/precompiles/avs"
	"github.com/imua-xyz/imuachain/testutil"
	"github.com/stretchr/testify/suite"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var s *AVSManagerPrecompileSuite

type AVSManagerPrecompileSuite struct {
	testutil.BaseTestSuite
	precompile *avs.Precompile
	// needed by test
	operatorAddress       sdk.AccAddress
	avsAddress            string
	assetID               string
	stakerID              string
	assetAddress          common.Address
	assetDecimal          uint32
	clientChainLzID       uint64
	depositAmount         sdkmath.Int
	delegationAmount      sdkmath.Int
	updatedAmountForOptIn sdkmath.Int
}

func TestPrecompileTestSuite(t *testing.T) {
	s = new(AVSManagerPrecompileSuite)
	suite.Run(t, s)

	// Run Ginkgo integration tests
	RegisterFailHandler(Fail)
	RunSpecs(t, "AVSManager Precompile Suite")
}

func (suite *AVSManagerPrecompileSuite) SetupTest() {
	suite.DoSetupTest()
	precompile, err := avs.NewPrecompile(suite.App.AVSManagerKeeper, suite.App.AuthzKeeper)
	suite.Require().NoError(err)
	suite.precompile = precompile
}
