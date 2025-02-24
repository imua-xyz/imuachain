package avs_test

import (
	"fmt"
	"strings"
	"time"

	utiltx "github.com/imua-xyz/imuachain/testutil/tx"

	assetskeeper "github.com/imua-xyz/imuachain/x/assets/keeper"
	avstypes "github.com/imua-xyz/imuachain/x/avs/types"
	epochstypes "github.com/imua-xyz/imuachain/x/epochs/types"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	assetstypes "github.com/imua-xyz/imuachain/x/assets/types"
	delegationtype "github.com/imua-xyz/imuachain/x/delegation/types"
	operatorKeeper "github.com/imua-xyz/imuachain/x/operator/keeper"
	operatorTypes "github.com/imua-xyz/imuachain/x/operator/types"
)

type StateForCheck struct {
	OptedInfo        *operatorTypes.OptedInfo
	AVSTotalShare    sdkmath.LegacyDec
	AVSOperatorShare sdkmath.LegacyDec
	AssetState       *operatorTypes.OptedInAssetState
	OperatorShare    sdkmath.LegacyDec
	StakerShare      sdkmath.LegacyDec
}

func (suite *AVSManagerPrecompileSuite) prepareOperator(address string) {
	opAccAddress, err := sdk.AccAddressFromBech32(address)
	suite.operatorAddress = opAccAddress
	suite.NoError(err)
	// register operator
	registerReq := &operatorTypes.RegisterOperatorReq{
		FromAddress: suite.operatorAddress.String(),
		Info: &operatorTypes.OperatorInfo{
			EarningsAddr: suite.operatorAddress.String(),
			ApproveAddr:  suite.operatorAddress.String(),
		},
	}
	_, err = s.OperatorMsgServer.RegisterOperator(s.Ctx, registerReq)
	suite.NoError(err)
}

func (suite *AVSManagerPrecompileSuite) prepareDeposit(assetAddress common.Address, amount sdkmath.Int) {
	clientChainLzID := uint64(101)
	suite.avsAddress = common.BytesToAddress([]byte("avsTestAddress")).String()
	suite.assetAddress = assetAddress
	suite.assetDecimal = 6
	suite.clientChainLzID = clientChainLzID
	suite.depositAmount = amount
	suite.updatedAmountForOptIn = sdkmath.NewInt(20)
	suite.stakerID, suite.assetID = assetstypes.GetStakerIDAndAssetID(suite.clientChainLzID, suite.Address[:], suite.assetAddress[:])
	// staking assets
	depositParam := &assetskeeper.DepositWithdrawParams{
		ClientChainLzID: suite.clientChainLzID,
		Action:          assetstypes.DepositLST,
		StakerAddress:   suite.Address[:],
		OpAmount:        suite.depositAmount,
		AssetsAddress:   assetAddress[:],
	}
	_, err := suite.App.AssetsKeeper.PerformDepositOrWithdraw(suite.Ctx, depositParam)
	suite.NoError(err)
}

func (suite *AVSManagerPrecompileSuite) prepareDelegation(isDelegation bool, assetAddress common.Address, amount sdkmath.Int) {
	suite.delegationAmount = amount
	param := &delegationtype.DelegationOrUndelegationParams{
		ClientChainID:   suite.clientChainLzID,
		AssetsAddress:   assetAddress[:],
		OperatorAddress: suite.operatorAddress,
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

func (suite *AVSManagerPrecompileSuite) prepare() {
	usdtAddress := common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7")
	depositAmount := sdkmath.NewInt(100)
	delegationAmount := sdkmath.NewInt(50)
	suite.prepareOperator("im18cggcpvwspnd5c6ny8wrqxpffj5zmhkl3agtrj")
	suite.prepareDeposit(usdtAddress, depositAmount)
	suite.prepareDelegation(true, usdtAddress, delegationAmount)
}

func (suite *AVSManagerPrecompileSuite) prepareAvs(assetIDs []string, task string) {
	avsOwnerAddresses := []string{
		sdk.AccAddress(suite.Address.Bytes()).String(),
		sdk.AccAddress(utiltx.GenerateAddress().Bytes()).String(),
		sdk.AccAddress(utiltx.GenerateAddress().Bytes()).String(),
	}
	err := suite.App.AVSManagerKeeper.UpdateAVSInfo(suite.Ctx, &avstypes.AVSRegisterOrDeregisterParams{
		Action:            avstypes.RegisterAction,
		EpochIdentifier:   epochstypes.HourEpochID,
		AvsAddress:        common.HexToAddress(suite.avsAddress),
		AssetIDs:          assetIDs,
		TaskAddress:       common.HexToAddress(task),
		AvsOwnerAddresses: avsOwnerAddresses,
	})
	suite.NoError(err)
}

func (suite *AVSManagerPrecompileSuite) CheckState(expectedState *StateForCheck) {
	// check opted info
	optInfo, err := suite.App.OperatorKeeper.GetOptedInfo(suite.Ctx, suite.operatorAddress.String(), suite.avsAddress)
	if expectedState.OptedInfo == nil {
		suite.True(strings.Contains(err.Error(), operatorTypes.ErrNoKeyInTheStore.Error()))
	} else {
		suite.NoError(err)
		suite.Equal(*expectedState.OptedInfo, *optInfo)
	}
	// check total USD value for AVS and operator
	value, err := suite.App.OperatorKeeper.GetAVSUSDValue(suite.Ctx, suite.avsAddress)
	if expectedState.AVSTotalShare.IsNil() {
		suite.True(strings.Contains(err.Error(), operatorTypes.ErrNoKeyInTheStore.Error()))
	} else {
		suite.NoError(err)
		suite.Equal(expectedState.AVSTotalShare, value)
	}

	optedUSDValues, err := suite.App.OperatorKeeper.GetOperatorOptedUSDValue(suite.Ctx, suite.avsAddress, suite.operatorAddress.String())
	if expectedState.AVSOperatorShare.IsNil() {
		fmt.Println("the err is:", err)
		suite.True(strings.Contains(err.Error(), operatorTypes.ErrNoKeyInTheStore.Error()))
	} else {
		suite.NoError(err)
		suite.Equal(expectedState.AVSOperatorShare, optedUSDValues.TotalUSDValue)
	}
}

func (suite *AVSManagerPrecompileSuite) TestOptIn() {
	suite.prepare()
	suite.prepareAvs([]string{"0xdac17f958d2ee523a2206206994597c13d831ec7_0x65"}, utiltx.GenerateAddress().String())
	err := suite.App.OperatorKeeper.OptIn(suite.Ctx, suite.operatorAddress, suite.avsAddress)
	suite.NoError(err)
	// check if the related state is correct
	price, err := suite.App.OperatorKeeper.OracleInterface().GetSpecifiedAssetsPrice(suite.Ctx, suite.assetID)
	suite.NoError(err)
	usdValue := operatorKeeper.CalculateUSDValue(suite.delegationAmount, price.Value, suite.assetDecimal, price.Decimal)
	expectedState := &StateForCheck{
		OptedInfo: &operatorTypes.OptedInfo{
			OptedInHeight:  uint64(suite.Ctx.BlockHeight()),
			OptedOutHeight: operatorTypes.DefaultOptedOutHeight,
		},
		AVSTotalShare:    usdValue,
		AVSOperatorShare: usdValue,
		AssetState: &operatorTypes.OptedInAssetState{
			Amount: suite.delegationAmount,
			Value:  usdValue,
		},
		OperatorShare: sdkmath.LegacyDec{},
		StakerShare:   usdValue,
	}
	suite.CommitAfter(time.Hour*1 + time.Nanosecond)
	suite.CommitAfter(time.Hour*2 + time.Nanosecond)
	suite.CheckState(expectedState)
}

func (suite *AVSManagerPrecompileSuite) TestOptInList() {
	suite.prepare()
	suite.prepareAvs([]string{"0xdac17f958d2ee523a2206206994597c13d831ec7_0x65"}, utiltx.GenerateAddress().String())
	err := suite.App.OperatorKeeper.OptIn(suite.Ctx, suite.operatorAddress, suite.avsAddress)
	suite.NoError(err)
	// check if the related state is correct
	operatorList, err := suite.App.OperatorKeeper.GetOptedInOperatorListByAVS(suite.Ctx, suite.avsAddress)
	suite.NoError(err)
	suite.Contains(operatorList, suite.operatorAddress.String())

	avsList, err := suite.App.OperatorKeeper.GetOptedInAVSForOperator(suite.Ctx, suite.operatorAddress.String())
	suite.NoError(err)

	suite.Contains(avsList, suite.avsAddress)
}

func (suite *AVSManagerPrecompileSuite) TestOptOut() {
	suite.prepare()
	suite.prepareAvs([]string{"0xdac17f958d2ee523a2206206994597c13d831ec7_0x65"}, utiltx.GenerateAddress().String())
	err := suite.App.OperatorKeeper.OptOut(suite.Ctx, suite.operatorAddress, suite.avsAddress)
	suite.EqualError(err, operatorTypes.ErrNotOptedIn.Error())

	err = suite.App.OperatorKeeper.OptIn(suite.Ctx, suite.operatorAddress, suite.avsAddress)
	suite.NoError(err)
	optInHeight := suite.Ctx.BlockHeight()
	suite.NextBlock()

	err = suite.App.OperatorKeeper.OptOut(suite.Ctx, suite.operatorAddress, suite.avsAddress)
	suite.NoError(err)

	expectedState := &StateForCheck{
		OptedInfo: &operatorTypes.OptedInfo{
			OptedInHeight:  uint64(optInHeight),
			OptedOutHeight: uint64(suite.Ctx.BlockHeight()),
		},
		AVSTotalShare:    sdkmath.LegacyZeroDec(),
		AVSOperatorShare: sdkmath.LegacyZeroDec(),
		AssetState:       nil,
		OperatorShare:    sdkmath.LegacyDec{},
		StakerShare:      sdkmath.LegacyDec{},
	}
	suite.CommitAfter(time.Hour*1 + time.Nanosecond)
	suite.CommitAfter(time.Hour*2 + time.Nanosecond)
	suite.CheckState(expectedState)
}
