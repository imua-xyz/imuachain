package keeper_test

import (
	"strconv"

	avstypes "github.com/ExocoreNetwork/exocore/x/avs/types"
	"github.com/ethereum/go-ethereum/common"
)

func (suite *AVSTestSuite) Test_GroupStatistics() {
	tasks := []avstypes.TaskResultInfo{
		{OperatorAddress: "addr1", TaskResponseHash: "hash1", TaskResponse: []byte("response1"), BlsSignature: []byte("sig1"), TaskContractAddress: "contract1", TaskId: 1},
		{OperatorAddress: "addr2", TaskResponseHash: "hash2", TaskResponse: []byte("response2"), BlsSignature: []byte("sig2"), TaskContractAddress: "contract2", TaskId: 2},
		{OperatorAddress: "addr3", TaskResponseHash: "hash3", TaskResponse: []byte("response3"), BlsSignature: []byte("sig3"), TaskContractAddress: "contract1", TaskId: 1},
	}

	groupedTasks := suite.App.AVSManagerKeeper.GroupTasksByIDAndAddress(tasks)
	suite.Equal(2, len(groupedTasks["contract1_1"]))
}

func (suite *AVSTestSuite) TestEpochEnd_TaskCalculation() {
	suite.TestSubmitTask_OnlyPhaseTwo_Mul()
	suite.CommitAfter(suite.EpochDuration)
	suite.CommitAfter(suite.EpochDuration)
	suite.CommitAfter(suite.EpochDuration)
	info, err := suite.App.AVSManagerKeeper.GetTaskInfo(suite.Ctx, strconv.Itoa(int(suite.taskId)), common.Address(suite.taskAddress.Bytes()).String())
	suite.NoError(err)
	expectInfo := &avstypes.TaskInfo{
		TaskContractAddress:   suite.taskAddress.String(),
		Name:                  "test-avsTask",
		TaskId:                suite.taskId,
		Hash:                  []byte("req-struct"),
		TaskResponsePeriod:    2,
		TaskStatisticalPeriod: 1,
		TaskChallengePeriod:   2,
		ThresholdPercentage:   90,
		StartingEpoch:         20,
		OptInOperators:        suite.operatorAddresses,
		NoSignedOperators:     nil,
		SignedOperators:       suite.operatorAddresses,
		ActualThreshold:       "33",
	}
	diff := avstypes.Difference(expectInfo.SignedOperators, info.SignedOperators)

	suite.Equal(0, len(diff))
	suite.Equal(expectInfo.NoSignedOperators, info.NoSignedOperators)
	suite.Equal(expectInfo.ActualThreshold, info.ActualThreshold)
}
