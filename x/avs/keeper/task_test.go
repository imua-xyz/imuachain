package keeper_test

import (
	"strconv"

	types "github.com/ExocoreNetwork/exocore/x/avs/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
)

func (suite *AVSTestSuite) TestTaskInfo() {
	info := &types.TaskInfo{
		TaskContractAddress: common.Address(suite.AccAddress.Bytes()).String(),
		Name:                "test-avstask-01",
		TaskId:              3,
		Hash:                []byte("active"),
		TaskResponsePeriod:  10000,
		TaskChallengePeriod: 5000,
		ThresholdPercentage: 60,
		TaskTotalPower:      sdk.ZeroDec(),
	}
	err := suite.App.AVSManagerKeeper.SetTaskInfo(suite.Ctx, info)
	suite.NoError(err)

	getTaskInfo, err := suite.App.AVSManagerKeeper.GetTaskInfo(suite.Ctx, strconv.Itoa(3), common.Address(suite.AccAddress.Bytes()).String())
	suite.NoError(err)
	suite.Equal(*info, *getTaskInfo)
}

func (suite *AVSTestSuite) TestGetTaskId() {
	addr := common.Address(suite.AccAddress.Bytes())

	taskId := suite.App.AVSManagerKeeper.GetTaskID(suite.Ctx, addr)
	suite.Equal(uint64(1), taskId)

	taskId = suite.App.AVSManagerKeeper.GetTaskID(suite.Ctx, addr)
	suite.Equal(uint64(2), taskId)
	taskId = suite.App.AVSManagerKeeper.GetTaskID(suite.Ctx, addr)
	suite.Equal(uint64(3), taskId)

	addr = common.Address(suite.avsAddress.Bytes())

	taskId = suite.App.AVSManagerKeeper.GetTaskID(suite.Ctx, addr)
	suite.Equal(uint64(1), taskId)

	taskId = suite.App.AVSManagerKeeper.GetTaskID(suite.Ctx, addr)
	suite.Equal(uint64(2), taskId)
}

func (suite *AVSTestSuite) TestTaskChallengedInfo() {
	suite.TestEpochEnd_TaskCalculation()
	suite.CommitAfter(suite.EpochDuration)
}
