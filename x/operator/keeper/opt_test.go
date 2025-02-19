package keeper_test

import (
	"strings"
	"time"

	"github.com/ExocoreNetwork/exocore/x/epochs/types"

	sdkmath "cosmossdk.io/math"
	assetskeeper "github.com/ExocoreNetwork/exocore/x/assets/keeper"
	assetstypes "github.com/ExocoreNetwork/exocore/x/assets/types"
	avstypes "github.com/ExocoreNetwork/exocore/x/avs/types"
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

func (suite *OperatorTestSuite) registerOperator(operator string) {
	// register operator
	registerReq := &operatorTypes.RegisterOperatorReq{
		FromAddress: operator,
		Info: &operatorTypes.OperatorInfo{
			EarningsAddr: operator,
			ApproveAddr:  operator,
		},
	}
	_, err := s.OperatorMsgServer.RegisterOperator(s.Ctx, registerReq)
	suite.NoError(err)
}

func (suite *OperatorTestSuite) prepareOperator() {
	operator := "exo13h6xg79g82e2g2vhjwg7j4r2z2hlncelwutkjr"
	opAccAddr, err := sdk.AccAddressFromBech32(operator)
	suite.operatorAddr = opAccAddr
	suite.NoError(err)
	// register operator
	suite.registerOperator(operator)
}

func (suite *OperatorTestSuite) prepareDeposit(stakerAddr, assetAddr common.Address, amount sdkmath.Int) {
	suite.assetAddr = assetAddr
	suite.assetDecimal = uint32(assetDecimal)
	suite.clientChainLzID = defaultClientChainID
	suite.stakerID, suite.assetID = assetstypes.GetStakerIDAndAssetID(suite.clientChainLzID, stakerAddr[:], assetAddr[:])
	// staking assets
	depositParam := &assetskeeper.DepositWithdrawParams{
		ClientChainLzID: suite.clientChainLzID,
		Action:          assetstypes.DepositLST,
		StakerAddress:   stakerAddr[:],
		OpAmount:        amount,
		AssetsAddress:   assetAddr[:],
	}
	_, err := suite.App.AssetsKeeper.PerformDepositOrWithdraw(suite.Ctx, depositParam)
	suite.NoError(err)
}

func (suite *OperatorTestSuite) prepareDelegation(isDelegation bool, staker, assetAddr common.Address, operator sdk.AccAddress, amount sdkmath.Int) {
	suite.delegationAmount = amount
	param := &delegationtype.DelegationOrUndelegationParams{
		ClientChainID:   suite.clientChainLzID,
		AssetsAddress:   assetAddr[:],
		OperatorAddress: operator,
		StakerAddress:   staker[:],
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

func (suite *OperatorTestSuite) prepare() {
	depositAmount := sdkmath.NewInt(100)
	delegationAmount := sdkmath.NewInt(50)
	suite.prepareOperator()
	suite.prepareDeposit(suite.Address, usdtAddr, depositAmount)
	suite.prepareDelegation(true, suite.Address, usdtAddr, suite.operatorAddr, delegationAmount)
}

func (suite *OperatorTestSuite) prepareAvs(avsName string, assetIDs []string, epochIdentifier string, unbondingPeriod uint64) {
	suite.avsAddr = common.BytesToAddress([]byte(avsName)).String()
	err := suite.App.AVSManagerKeeper.UpdateAVSInfo(suite.Ctx, &avstypes.AVSRegisterOrDeregisterParams{
		Action:          avstypes.RegisterAction,
		EpochIdentifier: epochIdentifier,
		AvsAddress:      common.HexToAddress(suite.avsAddr),
		AssetIDs:        assetIDs,
		UnbondingPeriod: unbondingPeriod,
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
	suite.prepareAvs(defaultAVSName, []string{usdtAssetID}, types.HourEpochID, defaultUnbondingPeriod)
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
	suite.prepareAvs(defaultAVSName, []string{usdtAssetID}, types.HourEpochID, defaultUnbondingPeriod)
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
	suite.prepareAvs(defaultAVSName, []string{usdtAssetID}, types.HourEpochID, defaultUnbondingPeriod)
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
