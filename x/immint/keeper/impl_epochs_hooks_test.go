package keeper_test

import (
	"time"

	sdkmath "cosmossdk.io/math"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

func (suite *KeeperTestSuite) TestEpochHooks() {
	epsilon := time.Nanosecond // negligible amount of buffer duration
	params := suite.App.ImmintKeeper.GetParams(suite.Ctx)
	feeCollector := suite.App.AccountKeeper.GetModuleAddress(
		authtypes.FeeCollectorName,
	)
	// default is day, we start by committing after 1 minute
	suite.SetupTest() // reset
	suite.CommitAfter(time.Minute + epsilon)
	// check balance
	suite.Require().True(
		suite.App.BankKeeper.GetBalance(
			suite.Ctx,
			feeCollector,
			params.MintDenom,
		).Amount.Equal(sdkmath.ZeroInt()),
	)

	// now go to one day
	suite.CommitAfter(time.Hour*24 + epsilon - time.Minute)
	// check balance
	// suite.Require().True(
	//	suite.App.BankKeeper.GetBalance(
	//		suite.Ctx,
	//		feeCollector,
	//		params.MintDenom,
	//	).Amount.Equal(params.EpochReward),
	// )
}
