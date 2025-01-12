package keeper_test

import (
	"fmt"
	epochtypes "github.com/ExocoreNetwork/exocore/x/epochs/types"
	"time"

	utiltx "github.com/ExocoreNetwork/exocore/testutil/tx"
	avstypes "github.com/ExocoreNetwork/exocore/x/avs/types"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	assetskeeper "github.com/ExocoreNetwork/exocore/x/assets/keeper"
	assetstypes "github.com/ExocoreNetwork/exocore/x/assets/types"

	"github.com/ExocoreNetwork/exocore/x/assets/types"
	delegationtype "github.com/ExocoreNetwork/exocore/x/delegation/types"
	operatortype "github.com/ExocoreNetwork/exocore/x/operator/types"
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
)

func (suite *DelegationTestSuite) basicPrepare() {
	suite.assetAddr = common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7")
	suite.clientChainLzID = uint64(101)
	opAccAddr, err := sdk.AccAddressFromBech32("exo13h6xg79g82e2g2vhjwg7j4r2z2hlncelwutkjr")
	suite.NoError(err)
	suite.opAccAddr = opAccAddr
	suite.depositAmount = sdkmath.NewInt(100)
	suite.delegationAmount = sdkmath.NewInt(50)
	suite.accAddr = suite.Address.Bytes()
}

func (suite *DelegationTestSuite) prepareDeposit(depositAmount sdkmath.Int) *assetskeeper.DepositWithdrawParams {
	depositEvent := &assetskeeper.DepositWithdrawParams{
		ClientChainLzID: suite.clientChainLzID,
		Action:          types.DepositLST,
		StakerAddress:   suite.Address[:],
		OpAmount:        depositAmount,
	}
	depositEvent.AssetsAddress = suite.assetAddr[:]
	err := suite.App.AssetsKeeper.PerformDepositOrWithdraw(suite.Ctx, depositEvent)
	suite.NoError(err)
	return depositEvent
}

func (suite *DelegationTestSuite) prepareDelegation(delegationAmount sdkmath.Int, operator sdk.AccAddress) *delegationtype.DelegationOrUndelegationParams {
	delegationEvent := &delegationtype.DelegationOrUndelegationParams{
		ClientChainID:   suite.clientChainLzID,
		Action:          types.DelegateTo,
		AssetsAddress:   suite.assetAddr.Bytes(),
		OperatorAddress: operator,
		StakerAddress:   suite.Address[:],
		OpAmount:        delegationAmount,
		TxHash:          common.HexToHash("0x24c4a315d757249c12a7a1d7b6fb96261d49deee26f06a3e1787d008b445c3ac"),
	}
	registerReq := &operatortype.RegisterOperatorReq{
		FromAddress: operator.String(),
		Info: &operatortype.OperatorInfo{
			EarningsAddr: operator.String(),
		},
	}
	_, err := s.OperatorMsgServer.RegisterOperator(s.Ctx, registerReq)
	suite.NoError(err)

	err = suite.App.DelegationKeeper.DelegateTo(suite.Ctx, delegationEvent)
	suite.NoError(err)
	return delegationEvent
}

func (suite *DelegationTestSuite) prepareOptingInDogfood(assetID string) (sdkmath.Int, *delegationtype.DelegationOrUndelegationParams) {
	assetInfo, err := suite.App.AssetsKeeper.GetStakingAssetInfo(suite.Ctx, assetID)
	suite.NoError(err)

	// use customized amount to meet the self-delegation requirement of dogfood AVS
	depositAmount := sdkmath.NewIntWithDecimal(1000, int(assetInfo.AssetBasicInfo.Decimals))
	suite.prepareDeposit(depositAmount)
	delegationAmount := sdkmath.NewIntWithDecimal(500, int(assetInfo.AssetBasicInfo.Decimals))
	delegationEvent := suite.prepareDelegation(delegationAmount, suite.opAccAddr)

	// mark it as self delegation
	err = suite.App.DelegationKeeper.AssociateOperatorWithStaker(
		suite.Ctx, suite.clientChainLzID, suite.opAccAddr, suite.Address[:],
	)
	suite.NoError(err)
	// opts into a test AVS
	chainIDWithoutRevision := avstypes.ChainIDWithoutRevision(suite.Ctx.ChainID())
	found, avsAddress := suite.App.AVSManagerKeeper.IsAVSByChainID(suite.Ctx, chainIDWithoutRevision)
	suite.True(found, "AVS not found")
	key := utiltx.GenerateConsensusKey()
	_, err = suite.OperatorMsgServer.OptIntoAVS(sdk.WrapSDKContext(suite.Ctx), &operatortype.OptIntoAVSReq{
		FromAddress:   suite.opAccAddr.String(),
		AvsAddress:    avsAddress,
		PublicKeyJSON: key.ToJSON(),
	})
	suite.NoError(err)
	return depositAmount, delegationEvent
}

func (suite *DelegationTestSuite) prepareDelegationNativeToken() *delegationtype.DelegationOrUndelegationParams {
	delegationEvent := &delegationtype.DelegationOrUndelegationParams{
		ClientChainID:   assetstypes.ExocoreChainLzID,
		Action:          types.DelegateTo,
		AssetsAddress:   common.HexToAddress(assetstypes.ExocoreAssetAddr).Bytes(),
		OperatorAddress: suite.opAccAddr,
		StakerAddress:   suite.accAddr[:],
		OpAmount:        suite.delegationAmount,
		TxHash:          common.HexToHash("0x24c4a315d757249c12a7a1d7b6fb96261d49deee26f06a3e1787d008b445c3ac"),
	}
	err := suite.App.DelegationKeeper.DelegateTo(suite.Ctx, delegationEvent)
	suite.NoError(err)
	return delegationEvent
}

func (suite *DelegationTestSuite) TestDelegateTo() {
	suite.basicPrepare()
	suite.prepareDeposit(suite.depositAmount)
	opAccAddr, err := sdk.AccAddressFromBech32("exo13h6xg79g82e2g2vhjwg7j4r2z2hlncelwutkjr")
	suite.NoError(err)
	delegationParams := &delegationtype.DelegationOrUndelegationParams{
		ClientChainID:   suite.clientChainLzID,
		Action:          types.DelegateTo,
		AssetsAddress:   suite.assetAddr.Bytes(),
		OperatorAddress: opAccAddr,
		StakerAddress:   suite.Address[:],
		OpAmount:        sdkmath.NewInt(50),
		TxHash:          common.HexToHash("0x24c4a315d757249c12a7a1d7b6fb96261d49deee26f06a3e1787d008b445c3ac"),
	}
	err = suite.App.DelegationKeeper.DelegateTo(suite.Ctx, delegationParams)
	suite.EqualError(err, errorsmod.Wrap(delegationtype.ErrOperatorNotExist, fmt.Sprintf("input operatorAddr is:%s", delegationParams.OperatorAddress)).Error())

	registerReq := &operatortype.RegisterOperatorReq{
		FromAddress: opAccAddr.String(),
		Info: &operatortype.OperatorInfo{
			EarningsAddr: opAccAddr.String(),
		},
	}
	_, err = s.OperatorMsgServer.RegisterOperator(s.Ctx, registerReq)
	suite.NoError(err)

	err = suite.App.DelegationKeeper.DelegateTo(suite.Ctx, delegationParams)
	suite.NoError(err)

	// check delegation states
	stakerID, assetID := types.GetStakerIDAndAssetID(delegationParams.ClientChainID, delegationParams.StakerAddress, delegationParams.AssetsAddress)
	restakerState, err := suite.App.AssetsKeeper.GetStakerSpecifiedAssetInfo(suite.Ctx, stakerID, assetID)
	suite.NoError(err)
	suite.Equal(types.StakerAssetInfo{
		TotalDepositAmount:        suite.depositAmount,
		WithdrawableAmount:        suite.depositAmount.Sub(delegationParams.OpAmount),
		PendingUndelegationAmount: sdkmath.ZeroInt(),
	}, *restakerState)

	operatorState, err := suite.App.AssetsKeeper.GetOperatorSpecifiedAssetInfo(suite.Ctx, opAccAddr, assetID)
	suite.NoError(err)
	suite.Equal(types.OperatorAssetInfo{
		TotalAmount:               delegationParams.OpAmount,
		PendingUndelegationAmount: sdkmath.ZeroInt(),
		TotalShare:                sdkmath.LegacyNewDecFromBigInt(delegationParams.OpAmount.BigInt()),
		OperatorShare:             sdkmath.LegacyZeroDec(),
	}, *operatorState)

	specifiedDelegationAmount, err := suite.App.DelegationKeeper.GetSingleDelegationInfo(suite.Ctx, stakerID, assetID, opAccAddr.String())
	suite.NoError(err)
	suite.Equal(delegationtype.DelegationAmounts{
		UndelegatableShare:     sdkmath.LegacyNewDecFromBigInt(delegationParams.OpAmount.BigInt()),
		WaitUndelegationAmount: sdkmath.ZeroInt(),
	}, *specifiedDelegationAmount)

	totalDelegationAmount, err := suite.App.DelegationKeeper.TotalDelegatedAmountForStakerAsset(suite.Ctx, stakerID, assetID)
	suite.NoError(err)
	suite.Equal(delegationParams.OpAmount, totalDelegationAmount)

	// delegate exocore-native-token
	delegationParams = &delegationtype.DelegationOrUndelegationParams{
		ClientChainID:   assetstypes.ExocoreChainLzID,
		Action:          types.DelegateTo,
		AssetsAddress:   common.HexToAddress(assetstypes.ExocoreAssetAddr).Bytes(),
		OperatorAddress: opAccAddr,
		StakerAddress:   suite.accAddr[:],
		OpAmount:        sdkmath.NewInt(50),
		TxHash:          common.HexToHash("0x24c4a315d757249c12a7a1d7b6fb96261d49deee26f06a3e1787d008b445c3ac"),
	}
	err = suite.App.DelegationKeeper.DelegateTo(suite.Ctx, delegationParams)
	suite.NoError(err)
	// check delegation states
	stakerID, assetID = types.GetStakerIDAndAssetID(delegationParams.ClientChainID, delegationParams.StakerAddress, delegationParams.AssetsAddress)
	restakerState, err = suite.App.AssetsKeeper.GetStakerSpecifiedAssetInfo(suite.Ctx, stakerID, assetID)
	suite.NoError(err)
	balance := suite.App.BankKeeper.GetBalance(suite.Ctx, suite.accAddr, assetstypes.ExocoreAssetDenom)
	suite.Equal(types.StakerAssetInfo{
		TotalDepositAmount:        balance.Amount.Add(delegationParams.OpAmount),
		WithdrawableAmount:        balance.Amount,
		PendingUndelegationAmount: sdkmath.ZeroInt(),
	}, *restakerState)
	operatorState, err = suite.App.AssetsKeeper.GetOperatorSpecifiedAssetInfo(suite.Ctx, opAccAddr, assetID)
	suite.NoError(err)
	suite.Equal(types.OperatorAssetInfo{
		TotalAmount:               delegationParams.OpAmount,
		PendingUndelegationAmount: sdkmath.ZeroInt(),
		TotalShare:                sdkmath.LegacyNewDecFromBigInt(delegationParams.OpAmount.BigInt()),
		OperatorShare:             sdkmath.LegacyZeroDec(),
	}, *operatorState)

	specifiedDelegationAmount, err = suite.App.DelegationKeeper.GetSingleDelegationInfo(suite.Ctx, stakerID, assetID, opAccAddr.String())
	suite.NoError(err)
	suite.Equal(delegationtype.DelegationAmounts{
		UndelegatableShare:     sdkmath.LegacyNewDecFromBigInt(delegationParams.OpAmount.BigInt()),
		WaitUndelegationAmount: sdkmath.ZeroInt(),
	}, *specifiedDelegationAmount)

	totalDelegationAmount, err = suite.App.DelegationKeeper.TotalDelegatedAmountForStakerAsset(suite.Ctx, stakerID, assetID)
	suite.NoError(err)
	suite.Equal(delegationParams.OpAmount, totalDelegationAmount)
}

func (suite *DelegationTestSuite) TestUndelegateFrom() {
	suite.basicPrepare()
	suite.prepareDeposit(suite.depositAmount)
	delegationEvent := suite.prepareDelegation(suite.delegationAmount, suite.opAccAddr)
	// test Undelegation
	initialUndelegationID := uint64(0)
	err := suite.App.DelegationKeeper.UndelegateFrom(suite.Ctx, delegationEvent)
	suite.NoError(err)

	// check state
	stakerID, assetID := types.GetStakerIDAndAssetID(delegationEvent.ClientChainID, delegationEvent.StakerAddress, delegationEvent.AssetsAddress)
	restakerState, err := suite.App.AssetsKeeper.GetStakerSpecifiedAssetInfo(suite.Ctx, stakerID, assetID)
	suite.NoError(err)
	suite.Equal(types.StakerAssetInfo{
		TotalDepositAmount:        suite.depositAmount,
		WithdrawableAmount:        suite.depositAmount.Sub(delegationEvent.OpAmount),
		PendingUndelegationAmount: delegationEvent.OpAmount,
	}, *restakerState)

	operatorState, err := suite.App.AssetsKeeper.GetOperatorSpecifiedAssetInfo(suite.Ctx, delegationEvent.OperatorAddress, assetID)
	suite.NoError(err)
	suite.Equal(types.OperatorAssetInfo{
		TotalAmount:               sdkmath.ZeroInt(),
		PendingUndelegationAmount: delegationEvent.OpAmount,
		TotalShare:                sdkmath.LegacyZeroDec(),
		OperatorShare:             sdkmath.LegacyZeroDec(),
	}, *operatorState)

	specifiedDelegationAmount, err := suite.App.DelegationKeeper.GetSingleDelegationInfo(suite.Ctx, stakerID, assetID, delegationEvent.OperatorAddress.String())
	suite.NoError(err)
	suite.Equal(delegationtype.DelegationAmounts{
		WaitUndelegationAmount: delegationEvent.OpAmount,
		UndelegatableShare:     sdkmath.LegacyZeroDec(),
	}, *specifiedDelegationAmount)

	totalDelegationAmount, err := suite.App.DelegationKeeper.TotalDelegatedAmountForStakerAsset(suite.Ctx, stakerID, assetID)
	suite.NoError(err)
	suite.Equal(sdkmath.ZeroInt(), totalDelegationAmount)

	records, err := suite.App.DelegationKeeper.GetStakerUndelegationRecords(suite.Ctx, stakerID, assetID)
	suite.NoError(err)
	suite.Equal(1, len(records))
	UndelegationRecord := &delegationtype.UndelegationRecord{
		StakerId:                 stakerID,
		AssetId:                  assetID,
		OperatorAddr:             delegationEvent.OperatorAddress.String(),
		TxHash:                   delegationEvent.TxHash.String(),
		BlockNumber:              uint64(suite.Ctx.BlockHeight()),
		Amount:                   delegationEvent.OpAmount,
		ActualCompletedAmount:    delegationEvent.OpAmount,
		UndelegationId:           initialUndelegationID,
		CompletedEpochIdentifier: epochtypes.NullEpochIdentifier,
		CompletedEpochNumber:     epochtypes.NullEpochNumber,
	}
	suite.Equal(UndelegationRecord, records[0].Undelegation)
	waitUndelegationRecords, err := suite.App.DelegationKeeper.GetUnCompletableUndelegations(suite.Ctx, epochtypes.NullEpochIdentifier, epochtypes.NullEpochNumber)
	suite.NoError(err)
	suite.Equal(1, len(waitUndelegationRecords))
	suite.Equal(UndelegationRecord, waitUndelegationRecords[0].Undelegation)

	// undelegate exocore-native-token
	delegationEvent = suite.prepareDelegationNativeToken()

	err = suite.App.DelegationKeeper.UndelegateFrom(suite.Ctx, delegationEvent)
	suite.NoError(err)

	stakerID, assetID = types.GetStakerIDAndAssetID(delegationEvent.ClientChainID, delegationEvent.StakerAddress, delegationEvent.AssetsAddress)
	restakerState, err = suite.App.AssetsKeeper.GetStakerSpecifiedAssetInfo(suite.Ctx, stakerID, assetID)
	suite.NoError(err)
	balance := suite.App.BankKeeper.GetBalance(suite.Ctx, suite.accAddr, assetstypes.ExocoreAssetDenom)
	suite.Equal(types.StakerAssetInfo{
		TotalDepositAmount:        balance.Amount.Add(delegationEvent.OpAmount),
		WithdrawableAmount:        balance.Amount,
		PendingUndelegationAmount: delegationEvent.OpAmount,
	}, *restakerState)

	operatorState, err = suite.App.AssetsKeeper.GetOperatorSpecifiedAssetInfo(suite.Ctx, delegationEvent.OperatorAddress, assetID)
	suite.NoError(err)
	suite.Equal(types.OperatorAssetInfo{
		TotalAmount:               sdkmath.ZeroInt(),
		PendingUndelegationAmount: delegationEvent.OpAmount,
		TotalShare:                sdkmath.LegacyZeroDec(),
		OperatorShare:             sdkmath.LegacyZeroDec(),
	}, *operatorState)

	specifiedDelegationAmount, err = suite.App.DelegationKeeper.GetSingleDelegationInfo(suite.Ctx, stakerID, assetID, delegationEvent.OperatorAddress.String())
	suite.NoError(err)
	suite.Equal(delegationtype.DelegationAmounts{
		WaitUndelegationAmount: delegationEvent.OpAmount,
		UndelegatableShare:     sdkmath.LegacyZeroDec(),
	}, *specifiedDelegationAmount)

	totalDelegationAmount, err = suite.App.DelegationKeeper.TotalDelegatedAmountForStakerAsset(suite.Ctx, stakerID, assetID)
	suite.NoError(err)
	suite.Equal(sdkmath.ZeroInt(), totalDelegationAmount)

	records, err = suite.App.DelegationKeeper.GetStakerUndelegationRecords(suite.Ctx, stakerID, assetID)
	suite.NoError(err)
	suite.Equal(1, len(records))
	UndelegationRecord = &delegationtype.UndelegationRecord{
		StakerId:                 stakerID,
		AssetId:                  assetID,
		OperatorAddr:             delegationEvent.OperatorAddress.String(),
		TxHash:                   delegationEvent.TxHash.String(),
		BlockNumber:              uint64(suite.Ctx.BlockHeight()),
		Amount:                   delegationEvent.OpAmount,
		ActualCompletedAmount:    delegationEvent.OpAmount,
		CompletedEpochIdentifier: epochtypes.NullEpochIdentifier,
		CompletedEpochNumber:     epochtypes.NullEpochNumber,
		UndelegationId:           initialUndelegationID + 1,
	}
	suite.Equal(UndelegationRecord, records[0].Undelegation)

	waitUndelegationRecords, err = suite.App.DelegationKeeper.GetUnCompletableUndelegations(suite.Ctx, epochtypes.NullEpochIdentifier, epochtypes.NullEpochNumber)
	suite.NoError(err)
	suite.Equal(2, len(waitUndelegationRecords))
	suite.Equal(UndelegationRecord, waitUndelegationRecords[1].Undelegation)
}

func (suite *DelegationTestSuite) TestCompleteUndelegation() {
	suite.basicPrepare()
	epochID := suite.App.StakingKeeper.GetEpochIdentifier(suite.Ctx)
	epochInfo, found := suite.App.EpochsKeeper.GetEpochInfo(suite.Ctx, epochID)
	suite.Equal(true, found)
	epochsUntilUnbonded := suite.App.StakingKeeper.GetEpochsUntilUnbonded(suite.Ctx)
	// Adding 1 ensures that the completion time falls at the start of
	// `epochInfo.CurrentEpoch + int64(epochsUntilUnbonded) + 1`.
	// This guarantees the unbonding duration is at least `epochsUntilUnbonded`,
	// regardless of when the undelegation is submitted during the current epoch.
	matureEpochs := epochInfo.CurrentEpoch + int64(epochsUntilUnbonded) + 1

	epochInfo, _ = suite.App.EpochsKeeper.GetEpochInfo(suite.Ctx, epochID)
	stakerID, assetID := types.GetStakerIDAndAssetID(suite.clientChainLzID, suite.Address[:], suite.assetAddr.Bytes())
	depositAmount, delegationEvent := suite.prepareOptingInDogfood(assetID)
	err := suite.App.DelegationKeeper.UndelegateFrom(suite.Ctx, delegationEvent)
	suite.NoError(err)

	// test complete Undelegation
	// run to next block
	suite.Ctx = suite.Ctx.WithBlockHeight(suite.Ctx.BlockHeight() + 1)
	// update epochs to mature pending delegations from dogfood
	for i := 0; i <= int(epochsUntilUnbonded); i++ {
		epochEndTime := epochInfo.CurrentEpochStartTime.Add(epochInfo.Duration)
		suite.Ctx = suite.Ctx.WithBlockTime(epochEndTime.Add(1 * time.Second))
		suite.App.EpochsKeeper.BeginBlocker(suite.Ctx)
		epochInfo, _ = suite.App.EpochsKeeper.GetEpochInfo(suite.Ctx, epochID)
	}

	suite.Equal(epochInfo.CurrentEpoch, matureEpochs)
	// update epochs to mature pending delegations from exocore-native-token by decrementing holdcount
	suite.App.StakingKeeper.EndBlock(suite.Ctx)
	suite.App.DelegationKeeper.EndBlock(suite.Ctx, abci.RequestEndBlock{})

	// check state
	restakerState, err := suite.App.AssetsKeeper.GetStakerSpecifiedAssetInfo(suite.Ctx, stakerID, assetID)
	suite.NoError(err)
	suite.Equal(types.StakerAssetInfo{
		TotalDepositAmount:        depositAmount,
		WithdrawableAmount:        depositAmount,
		PendingUndelegationAmount: sdkmath.ZeroInt(),
	}, *restakerState)

	operatorState, err := suite.App.AssetsKeeper.GetOperatorSpecifiedAssetInfo(suite.Ctx, delegationEvent.OperatorAddress, assetID)
	suite.NoError(err)
	suite.Equal(types.OperatorAssetInfo{
		TotalAmount:               sdkmath.ZeroInt(),
		PendingUndelegationAmount: sdkmath.ZeroInt(),
		TotalShare:                sdkmath.LegacyZeroDec(),
		OperatorShare:             sdkmath.LegacyZeroDec(),
	}, *operatorState)

	specifiedDelegationAmount, err := suite.App.DelegationKeeper.GetSingleDelegationInfo(suite.Ctx, stakerID, assetID, delegationEvent.OperatorAddress.String())
	suite.NoError(err)
	suite.Equal(delegationtype.DelegationAmounts{
		UndelegatableShare:     sdkmath.LegacyZeroDec(),
		WaitUndelegationAmount: sdkmath.ZeroInt(),
	}, *specifiedDelegationAmount)

	totalDelegationAmount, err := suite.App.DelegationKeeper.TotalDelegatedAmountForStakerAsset(suite.Ctx, stakerID, assetID)
	suite.NoError(err)
	suite.Equal(sdkmath.ZeroInt(), totalDelegationAmount)

	records, err := suite.App.DelegationKeeper.GetStakerUndelegationRecords(suite.Ctx, stakerID, assetID)
	suite.NoError(err)
	suite.Equal(0, len(records))

	waitUndelegationRecords, err := suite.App.DelegationKeeper.GetCompletableUndelegations(suite.Ctx)
	suite.NoError(err)
	suite.Equal(0, len(waitUndelegationRecords))

	// test exocore-native-token
	delegationEvent = suite.prepareDelegationNativeToken()
	err = suite.App.DelegationKeeper.UndelegateFrom(suite.Ctx, delegationEvent)
	suite.NoError(err)

	// test complete Undelegation
	// run to next block
	suite.Ctx = suite.Ctx.WithBlockHeight(suite.Ctx.BlockHeight() + 1)

	epochID = suite.App.StakingKeeper.GetEpochIdentifier(suite.Ctx)
	epochInfo, _ = suite.App.EpochsKeeper.GetEpochInfo(suite.Ctx, epochID)
	epochsUntilUnbonded = suite.App.StakingKeeper.GetEpochsUntilUnbonded(suite.Ctx)
	matureEpochs = epochInfo.CurrentEpoch + int64(epochsUntilUnbonded) + 1

	for i := 0; i <= int(epochsUntilUnbonded); i++ {
		epochEndTime := epochInfo.CurrentEpochStartTime.Add(epochInfo.Duration)
		suite.Ctx = suite.Ctx.WithBlockTime(epochEndTime.Add(1 * time.Second))
		suite.App.EpochsKeeper.BeginBlocker(suite.Ctx)
		epochInfo, _ = suite.App.EpochsKeeper.GetEpochInfo(suite.Ctx, epochID)
	}
	suite.Equal(epochInfo.CurrentEpoch, matureEpochs)
	// update epochs to mature pending delegations from exocore-native-token by decrementing holdcount
	suite.App.StakingKeeper.EndBlock(suite.Ctx)

	suite.App.DelegationKeeper.EndBlock(suite.Ctx, abci.RequestEndBlock{})

	// check state
	stakerID, assetID = types.GetStakerIDAndAssetID(delegationEvent.ClientChainID, delegationEvent.StakerAddress, delegationEvent.AssetsAddress)
	restakerState, err = suite.App.AssetsKeeper.GetStakerSpecifiedAssetInfo(suite.Ctx, stakerID, assetID)
	suite.NoError(err)

	balance := suite.App.BankKeeper.GetBalance(suite.Ctx, suite.accAddr, assetstypes.ExocoreAssetDenom)
	suite.Equal(types.StakerAssetInfo{
		TotalDepositAmount:        balance.Amount,
		WithdrawableAmount:        balance.Amount,
		PendingUndelegationAmount: sdkmath.ZeroInt(),
	}, *restakerState)

	operatorState, err = suite.App.AssetsKeeper.GetOperatorSpecifiedAssetInfo(suite.Ctx, delegationEvent.OperatorAddress, assetID)
	suite.NoError(err)
	suite.Equal(types.OperatorAssetInfo{
		TotalAmount:               sdkmath.ZeroInt(),
		PendingUndelegationAmount: sdkmath.ZeroInt(),
		TotalShare:                sdkmath.LegacyZeroDec(),
		OperatorShare:             sdkmath.LegacyZeroDec(),
	}, *operatorState)

	specifiedDelegationAmount, err = suite.App.DelegationKeeper.GetSingleDelegationInfo(suite.Ctx, stakerID, assetID, delegationEvent.OperatorAddress.String())
	suite.NoError(err)
	suite.Equal(delegationtype.DelegationAmounts{
		UndelegatableShare:     sdkmath.LegacyZeroDec(),
		WaitUndelegationAmount: sdkmath.ZeroInt(),
	}, *specifiedDelegationAmount)

	totalDelegationAmount, err = suite.App.DelegationKeeper.TotalDelegatedAmountForStakerAsset(suite.Ctx, stakerID, assetID)
	suite.NoError(err)
	suite.Equal(sdkmath.ZeroInt(), totalDelegationAmount)

	records, err = suite.App.DelegationKeeper.GetStakerUndelegationRecords(suite.Ctx, stakerID, assetID)
	suite.NoError(err)
	suite.Equal(0, len(records))

	waitUndelegationRecords, err = suite.App.DelegationKeeper.GetCompletableUndelegations(suite.Ctx)
	suite.NoError(err)
	suite.Equal(0, len(waitUndelegationRecords))
}

func (suite *DelegationTestSuite) TestMultipleUndelegations() {
	suite.basicPrepare()
	stakerID, assetID := types.GetStakerIDAndAssetID(suite.clientChainLzID, suite.Address[:], suite.assetAddr.Bytes())
	_, delegationEvent := suite.prepareOptingInDogfood(assetID)

	undelegationNumber := int64(20)
	opAmount := delegationEvent.OpAmount.Quo(sdkmath.NewInt(undelegationNumber))
	suite.True(opAmount.GT(sdkmath.ZeroInt()))
	delegationEvent.OpAmount = opAmount
	for i := int64(0); i < undelegationNumber; i++ {
		err := suite.App.DelegationKeeper.UndelegateFrom(suite.Ctx, delegationEvent)
		suite.NoError(err)
	}

	epochID := suite.App.StakingKeeper.GetEpochIdentifier(suite.Ctx)
	epochInfo, found := suite.App.EpochsKeeper.GetEpochInfo(suite.Ctx, epochID)
	suite.Equal(true, found)
	epochsUntilUnbonded := suite.App.StakingKeeper.GetEpochsUntilUnbonded(suite.Ctx)
	matureEpochs := epochInfo.CurrentEpoch + int64(epochsUntilUnbonded)

	// check the global undelegationID
	undelegationID := suite.App.DelegationKeeper.GetLastUndelegationID(suite.Ctx)
	suite.Equal(uint64(undelegationNumber), undelegationID)
	// test the function GetStakerUndelegationRecords
	// check state
	undelegationsAndHoldCount, err := suite.App.DelegationKeeper.GetStakerUndelegationRecords(suite.Ctx, stakerID, assetID)
	suite.NoError(err)
	suite.Equal(undelegationNumber, int64(len(undelegationsAndHoldCount)))
	for i, undelegation := range undelegationsAndHoldCount {
		suite.Equal(uint64(i), undelegation.Undelegation.UndelegationId)
		suite.Equal(opAmount, undelegation.Undelegation.Amount)
		suite.Equal(epochInfo.Identifier, undelegation.Undelegation.CompletedEpochIdentifier)
		suite.Equal(matureEpochs, undelegation.Undelegation.CompletedEpochNumber)
	}

	// test the function GetUnCompletableUndelegations
	epochNumber := epochInfo.CurrentEpoch
	undelegationsAndHoldCount, err = suite.App.DelegationKeeper.GetUnCompletableUndelegations(suite.Ctx, epochID, epochNumber)
	suite.NoError(err)
	suite.Equal(undelegationNumber, int64(len(undelegationsAndHoldCount)))
	for i, undelegation := range undelegationsAndHoldCount {
		suite.Equal(uint64(i), undelegation.Undelegation.UndelegationId)
		suite.Equal(opAmount, undelegation.Undelegation.Amount)
	}
	// test the final epoch for unbonding
	epochNumber = epochInfo.CurrentEpoch + int64(epochsUntilUnbonded)
	undelegationsAndHoldCount, err = suite.App.DelegationKeeper.GetUnCompletableUndelegations(suite.Ctx, epochID, epochNumber)
	suite.NoError(err)
	suite.Equal(undelegationNumber, int64(len(undelegationsAndHoldCount)))
	// test the completed epoch for unbonding
	epochNumber = epochInfo.CurrentEpoch + int64(epochsUntilUnbonded) + 1
	undelegationsAndHoldCount, err = suite.App.DelegationKeeper.GetUnCompletableUndelegations(suite.Ctx, epochID, epochNumber)
	suite.NoError(err)
	suite.Equal(int64(0), int64(len(undelegationsAndHoldCount)))

	// test the function GetCompletableUndelegations
	undelegations, err := suite.App.DelegationKeeper.GetCompletableUndelegations(suite.Ctx)
	suite.NoError(err)
	suite.Equal(int64(0), int64(len(undelegations)))
	// run to the matured epoch
	for i := uint32(0); i <= epochsUntilUnbonded; i++ {
		suite.CommitAfter(epochInfo.Duration)
	}
	epochInfo, found = suite.App.EpochsKeeper.GetEpochInfo(suite.Ctx, epochID)
	suite.Equal(true, found)
	suite.Equal(epochNumber, epochInfo.CurrentEpoch)
	undelegations, err = suite.App.DelegationKeeper.GetCompletableUndelegations(suite.Ctx)
	suite.NoError(err)
	suite.Equal(undelegationNumber, int64(len(undelegations)))
}
