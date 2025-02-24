package keeper_test

import (
	"time"

	"github.com/imua-xyz/imuachain/x/epochs/types"
)

func (suite *KeeperTestSuite) TestEpochInfoAddition() {
	suite.SetupTest()

	epochInfo := types.NewGenesisEpochInfo("monthly", time.Hour*24*30)
	suite.App.EpochsKeeper.AddEpochInfo(suite.Ctx, epochInfo)
	epochInfoSaved, found := suite.App.EpochsKeeper.GetEpochInfo(suite.Ctx, "monthly")
	suite.Require().True(found)
	suite.Require().Equal(epochInfo.Identifier, epochInfoSaved.Identifier)
	suite.Require().Equal(epochInfo.Duration, epochInfoSaved.Duration)
	suite.Require().Equal(epochInfo.CurrentEpoch, epochInfoSaved.CurrentEpoch)
	suite.Require().Equal(epochInfo.EpochCountingStarted, epochInfoSaved.EpochCountingStarted)
	// AddEpochInfo sets these params by itself
	suite.Require().Equal(suite.Ctx.BlockTime(), epochInfoSaved.StartTime)
	suite.Require().Equal(suite.Ctx.BlockHeight(), epochInfoSaved.CurrentEpochStartHeight)
	// CurrentEpochStartTime is set in the BeginBlocker, so skip that check here.

	// verify that all of the epochs are set.
	allEpochs := suite.App.EpochsKeeper.AllEpochInfos(suite.Ctx)
	suite.Require().Len(allEpochs, 5)
	// alphabetical order
	suite.Require().Equal(allEpochs[0].Identifier, types.DayEpochID)
	suite.Require().Equal(allEpochs[1].Identifier, types.HourEpochID)
	suite.Require().Equal(allEpochs[2].Identifier, types.MinuteEpochID)
	suite.Require().Equal(allEpochs[3].Identifier, epochInfo.Identifier)
	suite.Require().Equal(allEpochs[4].Identifier, types.WeekEpochID)

	// Test retrieval of non-existent epoch info
	_, found = suite.App.EpochsKeeper.GetEpochInfo(suite.Ctx, "fake")
	suite.Require().False(found)
}
