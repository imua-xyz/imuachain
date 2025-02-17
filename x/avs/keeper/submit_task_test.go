package keeper_test

import (
	"math/big"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common/math"

	sdkmath "cosmossdk.io/math"
	assetskeeper "github.com/ExocoreNetwork/exocore/x/assets/keeper"
	assetstypes "github.com/ExocoreNetwork/exocore/x/assets/types"
	avstypes "github.com/ExocoreNetwork/exocore/x/avs/types"
	delegationtype "github.com/ExocoreNetwork/exocore/x/delegation/types"
	epochstypes "github.com/ExocoreNetwork/exocore/x/epochs/types"
	operatorTypes "github.com/ExocoreNetwork/exocore/x/operator/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/prysmaticlabs/prysm/v4/crypto/bls/blst"
)

func (suite *AVSTestSuite) prepareOperator() {
	opAccAddr, err := sdk.AccAddressFromBech32("exo13h6xg79g82e2g2vhjwg7j4r2z2hlncelwutkjr")
	suite.operatorAddr = opAccAddr
	suite.NoError(err)
	// register operator
	registerReq := &operatorTypes.RegisterOperatorReq{
		FromAddress: suite.operatorAddr.String(),
		Info: &operatorTypes.OperatorInfo{
			EarningsAddr: suite.operatorAddr.String(),
			ApproveAddr:  suite.operatorAddr.String(),
		},
	}
	_, err = s.OperatorMsgServer.RegisterOperator(s.Ctx, registerReq)
	suite.NoError(err)
}

func (suite *AVSTestSuite) prepareDeposit(assetAddr common.Address, amount sdkmath.Int) {
	clientChainLzID := uint64(101)
	suite.avsAddr = common.BytesToAddress([]byte("avsTestAddr")).String()
	suite.assetAddr = assetAddr
	suite.assetDecimal = 6
	suite.clientChainLzID = clientChainLzID
	suite.depositAmount = amount
	suite.updatedAmountForOptIn = sdkmath.NewInt(20)
	suite.stakerID, suite.assetID = assetstypes.GetStakerIDAndAssetID(suite.clientChainLzID, suite.Address[:], suite.assetAddr[:])
	// staking assets
	depositParam := &assetskeeper.DepositWithdrawParams{
		ClientChainLzID: suite.clientChainLzID,
		Action:          assetstypes.DepositLST,
		StakerAddress:   suite.Address[:],
		OpAmount:        suite.depositAmount,
		AssetsAddress:   assetAddr[:],
	}
	_, err := suite.App.AssetsKeeper.PerformDepositOrWithdraw(suite.Ctx, depositParam)
	suite.NoError(err)
}

func (suite *AVSTestSuite) prepareDelegation(isDelegation bool, assetAddr common.Address, amount sdkmath.Int) {
	suite.delegationAmount = amount
	param := &delegationtype.DelegationOrUndelegationParams{
		ClientChainID:   suite.clientChainLzID,
		AssetsAddress:   assetAddr[:],
		OperatorAddress: suite.operatorAddr,
		StakerAddress:   suite.Address[:],
		OpAmount:        amount,
		TxHash:          common.HexToHash("0x24c4a315d757249c12a7a1d7b6fb96261d49deee26f06a3e1787d008b445c3ac"),
	}
	var err error
	if isDelegation {
		err = suite.App.DelegationKeeper.DelegateTo(suite.Ctx, param)
	} else {
		err = suite.App.DelegationKeeper.UndelegateFrom(suite.Ctx, param)
	}
	suite.NoError(err)
}

func (suite *AVSTestSuite) prepareAvs(assetIDs []string) {
	err := suite.App.AVSManagerKeeper.UpdateAVSInfo(suite.Ctx, &avstypes.AVSRegisterOrDeregisterParams{
		AvsName:             "avs01",
		Action:              avstypes.RegisterAction,
		EpochIdentifier:     epochstypes.HourEpochID,
		AvsAddress:          common.HexToAddress(suite.avsAddr),
		AssetID:             assetIDs,
		TaskAddr:            suite.taskAddress,
		SlashContractAddr:   common.Address{},
		RewardContractAddr:  common.Address{},
		MinSelfDelegation:   3,
		AvsOwnerAddress:     nil,
		UnbondingPeriod:     7,
		MinOptInOperators:   0,
		MinStakeAmount:      2,
		MinTotalStakeAmount: 2,
		AvsSlash:            2,
		AvsReward:           3,
	})

	suite.NoError(err)
}

func (suite *AVSTestSuite) prepareOptIn() {
	err := suite.App.OperatorKeeper.OptIn(suite.Ctx, suite.operatorAddr, suite.avsAddr)
	suite.NoError(err)
	suite.CommitAfter(time.Hour*1 + time.Nanosecond)
	suite.CommitAfter(time.Hour*1 + time.Nanosecond)
	suite.CommitAfter(time.Hour*1 + time.Nanosecond)
}

func (suite *AVSTestSuite) prepareOperatorubkey() {
	privateKey, err := blst.RandKey()
	suite.blsKey = privateKey
	publicKey := privateKey.PublicKey()
	blsPub := &avstypes.BlsPubKeyInfo{
		Operator: suite.operatorAddr.String(),
		PubKey:   publicKey.Marshal(),
		Name:     "",
	}

	err = suite.App.AVSManagerKeeper.SetOperatorPubKey(suite.Ctx, blsPub)
	suite.NoError(err)
}

func (suite *AVSTestSuite) prepareTaskInfo() {
	suite.taskId = suite.App.AVSManagerKeeper.GetTaskID(suite.Ctx, suite.taskAddress)
	epoch, _ := suite.App.EpochsKeeper.GetEpochInfo(suite.Ctx, epochstypes.HourEpochID)
	operatorList, err := suite.App.OperatorKeeper.GetOptedInOperatorListByAVS(suite.Ctx, suite.avsAddr)

	info := &avstypes.TaskInfo{
		TaskContractAddress:   suite.taskAddress.String(),
		Name:                  "test-avsTask",
		TaskId:                suite.taskId,
		Hash:                  []byte("req-struct"),
		TaskResponsePeriod:    2,
		TaskStatisticalPeriod: 1,
		TaskChallengePeriod:   2,
		ThresholdPercentage:   60,
		StartingEpoch:         uint64(epoch.CurrentEpoch + 1),
		ActualThreshold:       0,
		OptInOperators:        operatorList,
		TaskTotalPower:        sdk.Dec(sdkmath.ZeroInt()),
	}
	err = suite.App.AVSManagerKeeper.SetTaskInfo(suite.Ctx, info)
	suite.NoError(err)

	getTaskInfo, err := suite.App.AVSManagerKeeper.GetTaskInfo(suite.Ctx, strconv.Itoa(int(suite.taskId)), common.Address(suite.taskAddress.Bytes()).String())
	suite.NoError(err)
	suite.Equal(*info, *getTaskInfo)
}

func (suite *AVSTestSuite) prepare() {
	usdtAddress := common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7")
	decimal := int64(6)
	depositAmount := sdkmath.NewInt(100).Mul(sdkmath.NewIntFromBigInt(math.BigPow(10, decimal)))
	delegationAmount := sdkmath.NewInt(50).Mul(sdkmath.NewIntFromBigInt(math.BigPow(10, decimal)))
	suite.prepareOperator()
	suite.prepareDeposit(usdtAddress, depositAmount)
	err := suite.App.DelegationKeeper.AssociateOperatorWithStaker(suite.Ctx, suite.clientChainLzID, suite.operatorAddr, suite.Address[:])
	suite.NoError(err)
	suite.prepareDelegation(true, usdtAddress, delegationAmount)
	suite.prepareAvs([]string{"0xdac17f958d2ee523a2206206994597c13d831ec7_0x65"})
	suite.prepareOptIn()
	suite.prepareOperatorubkey()
	suite.prepareTaskInfo()
	suite.CommitAfter(time.Hour*1 + time.Nanosecond)
	suite.CommitAfter(time.Hour*1 + time.Nanosecond)
	suite.CommitAfter(time.Hour*1 + time.Nanosecond)
}

func (suite *AVSTestSuite) TestSubmitTask_OnlyPhaseOne() {
	suite.prepare()
	taskRes := avstypes.TaskResponse{TaskID: 1, NumberSum: big.NewInt(100)}
	jsonData, err := avstypes.MarshalTaskResponse(taskRes)
	suite.NoError(err)
	_ = crypto.Keccak256Hash(jsonData)

	// pub, err := suite.App.AVSManagerKeeper.GetOperatorPubKey(suite.Ctx, suite.operatorAddr.String())
	suite.NoError(err)

	msg, _ := avstypes.GetTaskResponseDigestEncodeByjson(taskRes)
	msgBytes := msg[:]
	sig := suite.blsKey.Sign(msgBytes)

	info := &avstypes.TaskResultInfo{
		TaskContractAddress: suite.taskAddress.String(),
		OperatorAddress:     suite.operatorAddr.String(),
		TaskId:              suite.taskId,
		TaskResponseHash:    "",
		TaskResponse:        nil,
		BlsSignature:        sig.Marshal(),
		Phase:               avstypes.Phase(avstypes.PhasePrepare),
	}
	err = suite.App.AVSManagerKeeper.SubmitTaskResult(suite.Ctx, suite.operatorAddr.String(), info)
	suite.NoError(err)
}

func (suite *AVSTestSuite) TestSubmitTask_OnlyPhaseTwo() {
	suite.TestSubmitTask_OnlyPhaseOne()
	suite.CommitAfter(suite.EpochDuration)

	taskRes := avstypes.TaskResponse{TaskID: 1, NumberSum: big.NewInt(100)}
	jsonData, err := avstypes.MarshalTaskResponse(taskRes)
	suite.NoError(err)
	hash := crypto.Keccak256Hash(jsonData)

	// pub, err := suite.App.AVSManagerKeeper.GetOperatorPubKey(suite.Ctx, suite.operatorAddr.String())
	suite.NoError(err)

	msg, _ := avstypes.GetTaskResponseDigestEncodeByjson(taskRes)
	msgBytes := msg[:]
	sig := suite.blsKey.Sign(msgBytes)

	info := &avstypes.TaskResultInfo{
		TaskContractAddress: suite.taskAddress.String(),
		OperatorAddress:     suite.operatorAddr.String(),
		TaskId:              suite.taskId,
		TaskResponseHash:    hash.String(),
		TaskResponse:        jsonData,
		BlsSignature:        sig.Marshal(),
		Phase:               avstypes.Phase(avstypes.PhaseDoCommit),
	}
	err = suite.App.AVSManagerKeeper.SubmitTaskResult(suite.Ctx, suite.operatorAddr.String(), info)
	suite.NoError(err)
}
