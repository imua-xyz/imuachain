package keeper_test

import (
	"fmt"
	"time"

	avstypes "github.com/ExocoreNetwork/exocore/x/avs/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (suite *KeeperTestSuite) TestExportGenesis() {
	suite.SetupTest()
	suite.prepare()
	epoch, _ := suite.App.EpochsKeeper.GetEpochInfo(suite.Ctx, suite.App.StakingKeeper.GetEpochIdentifier(suite.Ctx))
	currentEpoch := epoch.CurrentEpoch
	suite.Assert().Equal(currentEpoch, epoch.CurrentEpoch)

	epsilon := time.Nanosecond // negligible amount of buffer duration
	suite.Commit()
	suite.CommitAfter(time.Hour*24 + epsilon - time.Minute)
	allValidators := suite.App.StakingKeeper.GetAllExocoreValidators(suite.Ctx) // GetAllValidators(suite.Ctx)
	for i, val := range allValidators {
		pk, err := val.ConsPubKey()
		if err != nil {
			suite.Ctx.Logger().Error("Failed to deserialize public key; skipping", "error", err, "i", i)
			continue
		}
		validatorDetail, found := suite.App.StakingKeeper.ValidatorByConsAddrForChainID(
			suite.Ctx, sdk.GetConsAddress(pk), avstypes.ChainIDWithoutRevision(suite.Ctx.ChainID()),
		)
		if !found {
			suite.Ctx.Logger().Error("Operator address not found; skipping", "consAddress", sdk.GetConsAddress(pk), "i", i)
			continue
		}
		valBz := validatorDetail.GetOperator()
		currentRewards := suite.App.DistrKeeper.GetValidatorOutstandingRewards(suite.Ctx, valBz)
		suite.Require().NotNil(currentRewards)
	}
	genesis := suite.App.DistrKeeper.ExportGenesis(suite.Ctx)
	suite.App.DistrKeeper.InitGenesis(suite.Ctx, *genesis)
	suite.Require().NotNil(genesis, "Exported genesis should not be nil")
	fmt.Println(genesis)
}
