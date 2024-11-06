package keeper_test

import (
	"strings"
	"time"

	assetskeeper "github.com/ExocoreNetwork/exocore/x/assets/keeper"
	avstypes "github.com/ExocoreNetwork/exocore/x/avs/types"
	epochstypes "github.com/ExocoreNetwork/exocore/x/epochs/types"

	sdkmath "cosmossdk.io/math"
	assetstypes "github.com/ExocoreNetwork/exocore/x/assets/types"
	delegationtype "github.com/ExocoreNetwork/exocore/x/delegation/types"
	operatorKeeper "github.com/ExocoreNetwork/exocore/x/operator/keeper"
	operatorTypes "github.com/ExocoreNetwork/exocore/x/operator/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
)

type StateForCheck struct {
	OptedInfo        *operatorTypes.OptedInfo
	AVSTotalShare    sdkmath.LegacyDec
	AVSOperatorShare sdkmath.LegacyDec
	AssetState       *operatorTypes.OptedInAssetState
	OperatorShare    sdkmath.LegacyDec
	StakerShare      sdkmath.LegacyDec
}

func (suite *OperatorTestSuite) prepareOperator() {
	opAccAddr, err := sdk.AccAddressFromBech32("exo13h6xg79g82e2g2vhjwg7j4r2z2hlncelwutkjr")
	suite.operatorAddr = opAccAddr
	suite.NoError(err)
	// register operator
	registerReq := &operatorTypes.RegisterOperatorReq{
		FromAddress: suite.operatorAddr.String(),
		Info: &operatorTypes.OperatorInfo{
			EarningsAddr: suite.operatorAddr.String(),
		},
	}
	_, err = s.OperatorMsgServer.RegisterOperator(s.Ctx, registerReq)
	suite.NoError(err)
}

func (suite *OperatorTestSuite) prepareDeposit(assetAddr common.Address, amount sdkmath.Int) {
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
	err := suite.App.AssetsKeeper.PerformDepositOrWithdraw(suite.Ctx, depositParam)
	suite.NoError(err)
}

func (suite *OperatorTestSuite) prepareDelegation(isDelegation bool, assetAddr common.Address, amount sdkmath.Int) {
	suite.delegationAmount = amount
	param := &delegationtype.DelegationOrUndelegationParams{
		ClientChainID:   suite.clientChainLzID,
		AssetsAddress:   assetAddr[:],
		OperatorAddress: suite.operatorAddr,
		StakerAddress:   suite.Address[:],
		OpAmount:        amount,
		LzNonce:         0,
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

func (suite *OperatorTestSuite) prepare() {
	usdtAddress := common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7")
	depositAmount := sdkmath.NewInt(100)
	delegationAmount := sdkmath.NewInt(50)
	suite.prepareOperator()
	suite.prepareDeposit(usdtAddress, depositAmount)
	suite.prepareDelegation(true, usdtAddress, delegationAmount)
}

func (suite *OperatorTestSuite) prepareAvs(assetIDs []string) {
	err := suite.App.AVSManagerKeeper.UpdateAVSInfo(suite.Ctx, &avstypes.AVSRegisterOrDeregisterParams{
		Action:          avstypes.RegisterAction,
		EpochIdentifier: epochstypes.HourEpochID,
		AvsAddress:      common.HexToAddress(suite.avsAddr),
		AssetID:         assetIDs,
	})
	suite.NoError(err)
}

func (suite *OperatorTestSuite) CheckState(expectedState *StateForCheck) {
	// check opted info
	optInfo, err := suite.App.OperatorKeeper.GetOptedInfo(suite.Ctx, suite.operatorAddr.String(), suite.avsAddr)
	if expectedState.OptedInfo == nil {
		suite.True(strings.Contains(err.Error(), operatorTypes.ErrNoKeyInTheStore.Error()))
	} else {
		suite.NoError(err)
		suite.Equal(*expectedState.OptedInfo, *optInfo)
	}
	// check total USD value for AVS and operator
	value, err := suite.App.OperatorKeeper.GetAVSUSDValue(suite.Ctx, suite.avsAddr)
	if expectedState.AVSTotalShare.IsNil() {
		suite.True(strings.Contains(err.Error(), operatorTypes.ErrNoKeyInTheStore.Error()))
	} else {
		suite.NoError(err)
		suite.Equal(expectedState.AVSTotalShare, value)
	}

	optedUSDValues, err := suite.App.OperatorKeeper.GetOperatorOptedUSDValue(suite.Ctx, suite.avsAddr, suite.operatorAddr.String())
	if expectedState.AVSOperatorShare.IsNil() {
		suite.True(strings.Contains(err.Error(), operatorTypes.ErrNoKeyInTheStore.Error()))
	} else {
		suite.NoError(err)
		suite.Equal(expectedState.AVSOperatorShare, optedUSDValues.TotalUSDValue)
	}
}

func (suite *OperatorTestSuite) TestOptIn() {
	suite.prepare()
	suite.prepareAvs([]string{"0xdac17f958d2ee523a2206206994597c13d831ec7_0x65"})
	err := suite.App.OperatorKeeper.OptIn(suite.Ctx, suite.operatorAddr, suite.avsAddr)
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

func (suite *OperatorTestSuite) TestOptInList() {
	suite.prepare()
	suite.prepareAvs([]string{"0xdac17f958d2ee523a2206206994597c13d831ec7_0x65"})
	err := suite.App.OperatorKeeper.OptIn(suite.Ctx, suite.operatorAddr, suite.avsAddr)
	suite.NoError(err)
	// check if the related state is correct
	operatorList, err := suite.App.OperatorKeeper.GetOptedInOperatorListByAVS(suite.Ctx, suite.avsAddr)
	suite.NoError(err)
	suite.Contains(operatorList, suite.operatorAddr.String())

	avsList, err := suite.App.OperatorKeeper.GetOptedInAVSForOperator(suite.Ctx, suite.operatorAddr.String())
	suite.NoError(err)

	suite.Contains(avsList, suite.avsAddr)
}

func (suite *OperatorTestSuite) TestOptOut() {
	suite.prepare()
	suite.prepareAvs([]string{"0xdac17f958d2ee523a2206206994597c13d831ec7_0x65"})
	err := suite.App.OperatorKeeper.OptOut(suite.Ctx, suite.operatorAddr, suite.avsAddr)
	suite.EqualError(err, operatorTypes.ErrNotOptedIn.Error())

	err = suite.App.OperatorKeeper.OptIn(suite.Ctx, suite.operatorAddr, suite.avsAddr)
	suite.NoError(err)
	optInHeight := suite.Ctx.BlockHeight()
	suite.NextBlock()

	err = suite.App.OperatorKeeper.OptOut(suite.Ctx, suite.operatorAddr, suite.avsAddr)
	suite.NoError(err)

	expectedState := &StateForCheck{
		OptedInfo: &operatorTypes.OptedInfo{
			OptedInHeight:  uint64(optInHeight),
			OptedOutHeight: uint64(suite.Ctx.BlockHeight()),
		},
		AVSTotalShare:    sdkmath.LegacyNewDec(0),
		AVSOperatorShare: sdkmath.LegacyNewDec(0),
		AssetState:       nil,
		OperatorShare:    sdkmath.LegacyDec{},
		StakerShare:      sdkmath.LegacyDec{},
	}
	suite.CommitAfter(time.Hour*1 + time.Nanosecond)
	suite.CommitAfter(time.Hour*2 + time.Nanosecond)
	suite.CheckState(expectedState)
}
