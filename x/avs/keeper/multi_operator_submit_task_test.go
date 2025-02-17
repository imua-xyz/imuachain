package keeper_test

import (
	"math/big"
	"time"

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
	blscommon "github.com/prysmaticlabs/prysm/v4/crypto/bls/common"
)

func (suite *AVSTestSuite) prepareOperators() {
	for _, operatorAddress := range suite.operatorAddresses {
		opAccAddr, err := sdk.AccAddressFromBech32(operatorAddress)
		suite.Require().NoError(err)

		// register operator
		registerReq := &operatorTypes.RegisterOperatorReq{
			FromAddress: opAccAddr.String(),
			Info: &operatorTypes.OperatorInfo{
				EarningsAddr: opAccAddr.String(),
				ApproveAddr:  opAccAddr.String(),
			},
		}
		_, err = suite.OperatorMsgServer.RegisterOperator(suite.Ctx, registerReq)
		suite.Require().NoError(err)
	}
}

func (suite *AVSTestSuite) prepareMulDeposit(assetAddr common.Address, amount sdkmath.Int) {
	clientChainLzID := uint64(101)
	suite.avsAddr = common.BytesToAddress([]byte("avsTestAddr")).String()
	suite.assetAddr = assetAddr
	suite.assetDecimal = 6
	suite.clientChainLzID = clientChainLzID
	suite.depositAmount = amount
	suite.updatedAmountForOptIn = sdkmath.NewInt(2000)
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

func (suite *AVSTestSuite) prepareDelegations() {
	assetAddr := common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7")
	delegationAmount := sdkmath.NewInt(100)

	for _, operatorAddress := range suite.operatorAddresses {
		addr, err := sdk.AccAddressFromBech32(operatorAddress)
		suite.NoError(err)
		suite.prepareMulDelegation(addr, assetAddr, delegationAmount)
	}
}

func (suite *AVSTestSuite) prepareMulDelegation(operatorAddress sdk.AccAddress, assetAddr common.Address, amount sdkmath.Int) {
	param := &delegationtype.DelegationOrUndelegationParams{
		ClientChainID:   suite.clientChainLzID,
		AssetsAddress:   assetAddr[:],
		OperatorAddress: operatorAddress,
		StakerAddress:   suite.Address[:],
		OpAmount:        amount,
		TxHash:          common.HexToHash("0x24c4a315d757249c12a7a1d7b6fb96261d49deee26f06a3e1787d008b445c3ac"),
	}

	err := suite.App.DelegationKeeper.DelegateTo(suite.Ctx, param)
	suite.NoError(err)
}

func (suite *AVSTestSuite) prepareMulAvs(assetIDs []string) {
	err := suite.App.AVSManagerKeeper.UpdateAVSInfo(suite.Ctx, &avstypes.AVSRegisterOrDeregisterParams{
		AvsName:             "avs01",
		Action:              avstypes.RegisterAction,
		EpochIdentifier:     epochstypes.HourEpochID,
		AvsAddress:          common.HexToAddress(suite.avsAddr),
		AssetID:             assetIDs,
		TaskAddr:            suite.taskAddress,
		SlashContractAddr:   common.Address{},
		RewardContractAddr:  common.Address{},
		MinSelfDelegation:   0,
		AvsOwnerAddress:     nil,
		UnbondingPeriod:     7,
		MinOptInOperators:   3,
		MinStakeAmount:      2,
		MinTotalStakeAmount: 2,
		AvsSlash:            2,
		AvsReward:           3,
	})

	suite.NoError(err)
}

func (suite *AVSTestSuite) prepareMulOptIn() {
	for _, operatorAddress := range suite.operatorAddresses {
		addr, err := sdk.AccAddressFromBech32(operatorAddress)
		suite.NoError(err)
		err = suite.App.OperatorKeeper.OptIn(suite.Ctx, addr, suite.avsAddr)
		suite.NoError(err)
		suite.CommitAfter(time.Hour*1 + time.Nanosecond)
		suite.CommitAfter(time.Hour*1 + time.Nanosecond)
		suite.CommitAfter(time.Hour*1 + time.Nanosecond)
	}

	suite.CommitAfter(time.Hour*1 + time.Nanosecond)
	suite.CommitAfter(time.Hour*1 + time.Nanosecond)
	suite.CommitAfter(time.Hour*1 + time.Nanosecond)
}

func (suite *AVSTestSuite) prepareMulOperatorubkey() {
	suite.blsKeys = make([]blscommon.SecretKey, len(suite.operatorAddresses))
	for index, operatorAddress := range suite.operatorAddresses {
		privateKey, err := blst.RandKey()
		suite.blsKeys[index] = privateKey
		suite.Require().NoError(err)
		publicKey := privateKey.PublicKey()
		blsPub := &avstypes.BlsPubKeyInfo{
			Operator: operatorAddress,
			PubKey:   publicKey.Marshal(),
			Name:     "",
		}
		err = suite.App.AVSManagerKeeper.SetOperatorPubKey(suite.Ctx, blsPub)
		suite.Require().NoError(err)
	}
}

func (suite *AVSTestSuite) prepareMulTaskInfo() {
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
	}
	err = suite.App.AVSManagerKeeper.SetTaskInfo(suite.Ctx, info)
	suite.NoError(err)
}

func (suite *AVSTestSuite) prepareMul() {
	usdtAddress := common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7")
	depositAmount := sdkmath.NewInt(500)
	// delegationAmount := sdkmath.NewInt(100)
	suite.prepareOperators()
	suite.prepareMulDeposit(usdtAddress, depositAmount)
	suite.prepareDelegations()
	suite.prepareMulAvs([]string{"0xdac17f958d2ee523a2206206994597c13d831ec7_0x65"})
	suite.prepareMulOptIn()
	suite.prepareMulOperatorubkey()
	suite.prepareMulTaskInfo()
	suite.App.OperatorKeeper.SetAVSUSDValue(suite.Ctx, suite.avsAddr, sdkmath.LegacyNewDec(500))
	for _, operatorAddress := range suite.operatorAddresses {
		delta := operatorTypes.DeltaOperatorUSDInfo{
			SelfUSDValue:   sdkmath.LegacyNewDec(100),
			TotalUSDValue:  sdkmath.LegacyNewDec(100),
			ActiveUSDValue: sdkmath.LegacyNewDec(100),
		}
		suite.App.OperatorKeeper.UpdateOperatorUSDValue(suite.Ctx, suite.avsAddr, operatorAddress, delta)
	}

	suite.CommitAfter(time.Hour*1 + time.Nanosecond)
	suite.CommitAfter(time.Hour*1 + time.Nanosecond)
	suite.CommitAfter(time.Hour*1 + time.Nanosecond)
}

func (suite *AVSTestSuite) TestSubmitTask_OnlyPhaseOne_Mul() {
	suite.prepareMul()
	for index, operatorAddress := range suite.operatorAddresses {
		taskRes := avstypes.TaskResponse{TaskID: 1, NumberSum: big.NewInt(100)}
		msg, _ := avstypes.GetTaskResponseDigestEncodeByjson(taskRes)
		msgBytes := msg[:]
		sig := suite.blsKeys[index].Sign(msgBytes)

		info := &avstypes.TaskResultInfo{
			TaskContractAddress: suite.taskAddress.String(),
			OperatorAddress:     operatorAddress,
			TaskId:              suite.taskId,
			TaskResponseHash:    "",
			TaskResponse:        nil,
			BlsSignature:        sig.Marshal(),
			Phase:               avstypes.Phase(avstypes.PhasePrepare),
		}
		err := suite.App.AVSManagerKeeper.SubmitTaskResult(suite.Ctx, operatorAddress, info)
		suite.Require().NoError(err)
	}
}

func (suite *AVSTestSuite) TestSubmitTask_OnlyPhaseTwo_Mul() {
	suite.TestSubmitTask_OnlyPhaseOne_Mul()
	suite.CommitAfter(suite.EpochDuration)
	for index, operatorAddress := range suite.operatorAddresses {
		taskRes := avstypes.TaskResponse{TaskID: 1, NumberSum: big.NewInt(100)}
		jsonData, err := avstypes.MarshalTaskResponse(taskRes)
		suite.NoError(err)
		hash := crypto.Keccak256Hash(jsonData)
		// pub, err := suite.App.AVSManagerKeeper.GetOperatorPubKey(suite.Ctx, suite.operatorAddr.String())
		suite.NoError(err)
		msg, _ := avstypes.GetTaskResponseDigestEncodeByjson(taskRes)
		msgBytes := msg[:]
		sig := suite.blsKeys[index].Sign(msgBytes)

		info := &avstypes.TaskResultInfo{
			TaskContractAddress: suite.taskAddress.String(),
			OperatorAddress:     operatorAddress,
			TaskId:              suite.taskId,
			TaskResponseHash:    hash.String(),
			TaskResponse:        jsonData,
			BlsSignature:        sig.Marshal(),
			Phase:               avstypes.Phase(avstypes.PhaseDoCommit),
		}
		err = suite.App.AVSManagerKeeper.SubmitTaskResult(suite.Ctx, operatorAddress, info)
		suite.NoError(err)
	}
}
