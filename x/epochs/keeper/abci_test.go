package keeper_test

import (
	"time"

	"github.com/imua-xyz/imuachain/x/epochs/types"
)

// test situations for block production
// 1. one minute into the future and see if the minute epoch has incremented
// 2. one hour into the future and see if the hour epoch has incremented
// 3. one day into the future and see if the day epoch has incremented
// 4. one week into the future and see if the week epoch has incremented
// 5. two minutes, one-by-one, and see if the minute epoch has incremented twice.
//    and then less than one minute, so see if the minute epoch has not incremented.
// 6. directly three minutes into the future, the minute epoch should increase by 1.
//    then, add more blocks at one second each, and see the minute epoch increases
//    by 2 at the first two blocks, and then at 60 seconds therafter.

// test cases 1 to 4
func (suite *KeeperTestSuite) TestEpochsIncreaseByOne() {
	allEpochs := []string{
		types.DayEpochID, types.HourEpochID, types.MinuteEpochID, types.WeekEpochID,
	}
	durations := []time.Duration{
		time.Hour * 24, time.Hour, time.Minute, time.Hour * 24 * 7,
	}
	for i := range allEpochs {
		epochIdentifier := allEpochs[i]
		suite.Run(epochIdentifier, func() {
			suite.SetupTest() // reset
			duration := durations[i]
			prevEpoch, found := suite.App.EpochsKeeper.GetEpochInfo(suite.Ctx, epochIdentifier)
			suite.Require().True(found)
			// negligible amount of buffer duration added
			suite.CommitAfter(duration + time.Nanosecond)
			// commit will run the EndBlockers for the current block, call app.Commit
			// and then run the BeginBlockers for the next block with the new time.
			// during the BeginBlocker, the epoch will be incremented.
			epoch, found := suite.App.EpochsKeeper.GetEpochInfo(suite.Ctx, epochIdentifier)
			suite.Require().True(found)
			suite.Require().Equal(prevEpoch.CurrentEpoch+1, epoch.CurrentEpoch, "block %d prev %d next %d", suite.Ctx.BlockHeight(), prevEpoch.CurrentEpoch, epoch.CurrentEpoch)
		})
	}
}

// test case 5
func (suite *KeeperTestSuite) TestTwoEpochIncreases() {
	epsilon := time.Nanosecond // negligible amount of buffer duration
	epochID := types.MinuteEpochID
	duration := time.Minute + epsilon
	suite.SetupTest()
	prevEpoch, found := suite.App.EpochsKeeper.GetEpochInfo(suite.Ctx, epochID)
	suite.Require().True(found)
	suite.CommitAfter(duration)
	epoch, found := suite.App.EpochsKeeper.GetEpochInfo(suite.Ctx, epochID)
	suite.Require().True(found)
	suite.Require().Equal(prevEpoch.CurrentEpoch+1, epoch.CurrentEpoch)
	suite.CommitAfter(duration)
	epoch, found = suite.App.EpochsKeeper.GetEpochInfo(suite.Ctx, epochID)
	suite.Require().True(found)
	suite.Require().Equal(prevEpoch.CurrentEpoch+2, epoch.CurrentEpoch)
	suite.CommitAfter(time.Second * 30)
	epoch, found = suite.App.EpochsKeeper.GetEpochInfo(suite.Ctx, epochID)
	suite.Require().True(found)
	suite.Require().Equal(prevEpoch.CurrentEpoch+2, epoch.CurrentEpoch)
}

// test case 6
func (suite *KeeperTestSuite) TestChainDowntime() {
	epochID := types.MinuteEpochID
	suite.SetupTest()
	prevEpoch, found := suite.App.EpochsKeeper.GetEpochInfo(suite.Ctx, epochID)
	suite.Require().True(found)
	// we skip 3 increments, of which #1 is applied here
	suite.CommitAfter(time.Minute*3 + time.Nanosecond)
	epoch, found := suite.App.EpochsKeeper.GetEpochInfo(suite.Ctx, epochID)
	suite.Require().True(found)
	suite.Require().Equal(prevEpoch.CurrentEpoch+1, epoch.CurrentEpoch)
	// #2 is applied here
	suite.CommitAfter(time.Second)
	epoch, found = suite.App.EpochsKeeper.GetEpochInfo(suite.Ctx, epochID)
	suite.Require().True(found)
	suite.Require().Equal(prevEpoch.CurrentEpoch+2, epoch.CurrentEpoch)
	// #3 is applied here
	suite.CommitAfter(time.Second)
	epoch, found = suite.App.EpochsKeeper.GetEpochInfo(suite.Ctx, epochID)
	suite.Require().True(found)
	suite.Require().Equal(prevEpoch.CurrentEpoch+3, epoch.CurrentEpoch)
	// #4 is not applied
	suite.CommitAfter(time.Second)
	epoch, found = suite.App.EpochsKeeper.GetEpochInfo(suite.Ctx, epochID)
	suite.Require().True(found)
	suite.Require().Equal(prevEpoch.CurrentEpoch+3, epoch.CurrentEpoch)
	// #5 is applied after a minute from #3
	suite.CommitAfter(time.Minute - time.Second + time.Nanosecond)
	epoch, found = suite.App.EpochsKeeper.GetEpochInfo(suite.Ctx, epochID)
	suite.Require().True(found)
	suite.Require().Equal(prevEpoch.CurrentEpoch+4, epoch.CurrentEpoch)
}
