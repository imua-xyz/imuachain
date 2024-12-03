package keeper_test

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	"github.com/ExocoreNetwork/exocore/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/suite"
)

var s *OperatorTestSuite

type OperatorTestSuite struct {
	testutil.BaseTestSuite

	// needed by test
	operatorAddr     sdk.AccAddress
	avsAddr          string
	assetID          string
	stakerID         string
	assetAddr        common.Address
	assetDecimal     uint32
	clientChainLzID  uint64
	delegationAmount sdkmath.Int
}

func TestOperatorTestSuite(t *testing.T) {
	s = new(OperatorTestSuite)
	suite.Run(t, s)

	// Run Ginkgo integration tests
	RegisterFailHandler(Fail)
	RunSpecs(t, "operator module Suite")
}

func (suite *OperatorTestSuite) SetupTest() {
	suite.DoSetupTest()
}
