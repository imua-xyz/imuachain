package keeper_test

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/imua-xyz/imuachain/testutil"

	"github.com/stretchr/testify/suite"
)

type DelegationTestSuite struct {
	testutil.BaseTestSuite
	assetAddr        common.Address
	opAccAddr        types.AccAddress
	clientChainLzID  uint64
	depositAmount    sdkmath.Int
	delegationAmount sdkmath.Int
	accAddr          sdk.AccAddress
}

var s *DelegationTestSuite

func TestKeeperTestSuite(t *testing.T) {
	s = new(DelegationTestSuite)
	suite.Run(t, s)
}

func (suite *DelegationTestSuite) SetupTest() {
	suite.DoSetupTest()
}
