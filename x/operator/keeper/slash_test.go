package keeper_test

import (
	"time"

	"github.com/imua-xyz/imuachain/utils"

	abci "github.com/cometbft/cometbft/abci/types"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/imua-xyz/imuachain/x/operator/keeper"
	"github.com/imua-xyz/imuachain/x/operator/types"
)

func (suite *OperatorTestSuite) TestSlashWithInfractionReason() {
	// current height: 1 epoch: 1
	// prepare the deposit and delegation
	suite.prepareOperator()
	depositAmount := sdkmath.NewIntWithDecimal(200, assetDecimal)
	suite.prepareDeposit(suite.Address, usdtAddr, depositAmount)
	delegationAmount := sdkmath.NewIntWithDecimal(100, assetDecimal)
	suite.prepareDelegation(true, suite.Address, suite.assetAddr, suite.operatorAddr, delegationAmount)
	err := suite.App.DelegationKeeper.AssociateOperatorWithStaker(suite.Ctx, suite.clientChainLzID, suite.operatorAddr, suite.Address[:])
	suite.NoError(err)

	// opt into the AVS
	avsAddr := utils.GenerateAVSAddress(utils.ChainIDWithoutRevision(suite.Ctx.ChainID()))
	err = suite.App.OperatorKeeper.OptIn(suite.Ctx, suite.operatorAddr, avsAddr)
	suite.NoError(err)

	// the epoch identifier of dogfood AVS is day
	// call the EndBlock to update the voting power
	suite.CommitAfter(time.Hour*24 + time.Nanosecond)

	// current height: 2 epoch: 2
	optedUSDValues, err := suite.App.OperatorKeeper.GetOperatorOptedUSDValue(suite.Ctx, avsAddr, suite.operatorAddr.String())
	suite.NoError(err)
	// get the historical voting power
	power := optedUSDValues.TotalUSDValue.TruncateInt64()
	// run to next block
	suite.NextBlock()
	// current height: 3 epoch: 2
	infractionHeight := suite.Ctx.BlockHeight()
	// undelegationFilterHeight should be the first height of this epoch, it should be 2
	undelegationFilterHeight := infractionHeight - 1
	suite.Equal(int64(3), infractionHeight)

	// delegates new amount to the operator
	newDelegateAmount := sdkmath.NewIntWithDecimal(20, assetDecimal)
	suite.prepareDelegation(true, suite.Address, suite.assetAddr, suite.operatorAddr, newDelegateAmount)
	// updating the voting power
	suite.CommitAfter(time.Hour*24 + time.Nanosecond)
	// current height: 4 epoch: 3
	newOptedUSDValues, err := suite.App.OperatorKeeper.GetOperatorOptedUSDValue(suite.Ctx, avsAddr, suite.operatorAddr.String())
	suite.NoError(err)
	// submits an undelegation to test the slashFromUndelegation
	undelegationAmount := sdkmath.NewIntWithDecimal(10, assetDecimal)
	suite.prepareDelegation(false, suite.Address, suite.assetAddr, suite.operatorAddr, undelegationAmount)
	delegationRemaining := delegationAmount.Add(newDelegateAmount).Sub(undelegationAmount)
	completedEpochId, completedEpochNumber, _, err := suite.App.OperatorKeeper.GetUnbondingExpiration(suite.Ctx, suite.operatorAddr)
	suite.NoError(err)
	epochInfo, found := suite.App.EpochsKeeper.GetEpochInfo(suite.Ctx, completedEpochId)
	suite.True(found)
	// the reason of plussing 1 is that the undelegation will be completed at the start height of completedEpochNumber+1
	// completedTime := epochInfo.CurrentEpochStartTime.Add(time.Duration(completedEpochNumber+1-epochInfo.CurrentEpoch) * epochInfo.Duration)

	// trigger the slash with a downtime event
	// run to next block
	suite.CommitAfter(time.Hour + time.Nanosecond)
	// current height: 5 epoch: 3
	slashFactor := suite.App.SlashingKeeper.SlashFractionDowntime(suite.Ctx)
	slashType := stakingtypes.Infraction_INFRACTION_DOWNTIME
	imuaSlashValue := suite.App.OperatorKeeper.SlashWithInfractionReason(suite.Ctx, suite.operatorAddr, infractionHeight, power, slashFactor, slashType)
	suite.Equal(sdkmath.ZeroInt(), imuaSlashValue)

	// verify the state after the slash
	slashID := keeper.GetSlashIDForDogfood(slashType, infractionHeight)
	slashInfo, err := suite.App.OperatorKeeper.GetOperatorSlashInfo(suite.Ctx, avsAddr, suite.operatorAddr.String(), slashID)
	suite.NoError(err)

	// check the stored slash records
	slashValue := optedUSDValues.TotalUSDValue.Mul(slashFactor)
	newSlashProportion := slashValue.Quo(newOptedUSDValues.TotalUSDValue)
	suite.Equal(suite.Ctx.BlockHeight(), slashInfo.SubmittedHeight)
	suite.Equal(infractionHeight, slashInfo.EventHeight)
	suite.Equal(slashFactor, slashInfo.SlashProportion)
	suite.Equal(uint32(slashType), slashInfo.SlashType)
	suite.NotEmpty(slashInfo.ExecutionInfo.SlashUndelegations)
	suite.Equal(types.SlashFromUndelegation{
		StakerID: suite.stakerID,
		AssetID:  suite.assetID,
		Amount:   newSlashProportion.MulInt(undelegationAmount).TruncateInt(),
	}, slashInfo.ExecutionInfo.SlashUndelegations[0])
	suite.NotEmpty(slashInfo.ExecutionInfo.SlashAssetsPool)
	suite.Equal(types.SlashFromAssetPool{
		AssetID:            suite.assetID,
		TotalAmount:        newSlashProportion.MulInt(delegationRemaining).TruncateInt(),
		SnapshotTotalShare: delegationRemaining.ToLegacyDec(),
	}, slashInfo.ExecutionInfo.SlashAssetsPool[0])
	suite.Equal(undelegationFilterHeight, slashInfo.ExecutionInfo.UndelegationFilterHeight)

	// check the assets state of undelegation and assets pool
	assetsInfo, err := suite.App.AssetsKeeper.GetOperatorSpecifiedAssetInfo(suite.Ctx, suite.operatorAddr, suite.assetID)
	suite.NoError(err)
	suite.Equal(delegationRemaining.Sub(slashInfo.ExecutionInfo.SlashAssetsPool[0].TotalAmount), assetsInfo.TotalAmount)

	undelegations, err := suite.App.DelegationKeeper.GetStakerUndelegationRecords(suite.Ctx, suite.stakerID, suite.assetID)
	suite.NoError(err)
	suite.Equal(undelegationAmount.Sub(slashInfo.ExecutionInfo.SlashUndelegations[0].Amount), undelegations[0].Undelegation.ActualCompletedAmount)

	// run to the epoch at which the undelegation is completed
	epochInfo, found = suite.App.EpochsKeeper.GetEpochInfo(suite.Ctx, completedEpochId)
	suite.True(found)
	for i := epochInfo.CurrentEpoch; i <= completedEpochNumber; i++ {
		suite.CommitAfter(time.Hour*24 + time.Nanosecond)
	}
	suite.App.DelegationKeeper.EndBlock(suite.Ctx, abci.RequestEndBlock{})
	epochInfo, found = suite.App.EpochsKeeper.GetEpochInfo(suite.Ctx, completedEpochId)
	suite.True(found)
	suite.Greaterf(epochInfo.CurrentEpoch, completedEpochNumber, "invalid epoch number to complete the undelegation")
	undelegations, err = suite.App.DelegationKeeper.GetStakerUndelegationRecords(suite.Ctx, suite.stakerID, suite.assetID)
	suite.NoError(err)
	suite.Equal(0, len(undelegations))
}

func (suite *OperatorTestSuite) TestMaxSlashProportion() {
	testMaxSlashProportion, err := sdk.NewDecFromStr("0.005")
	suite.Require().NoError(err)
	params := suite.App.OperatorKeeper.GetParams(suite.Ctx)
	params.MaxSlashProportion = testMaxSlashProportion
	suite.App.OperatorKeeper.SetParams(suite.Ctx, params)

	slashFactor := suite.App.SlashingKeeper.SlashFractionDowntime(suite.Ctx)
	slashType := stakingtypes.Infraction_INFRACTION_DOWNTIME
	infractionHeight := suite.Ctx.BlockHeight()
	power := suite.Powers[0]
	suite.NextBlock()
	suite.App.OperatorKeeper.SlashWithInfractionReason(suite.Ctx, suite.Operators[0], infractionHeight, power, slashFactor, slashType)

	slashID := keeper.GetSlashIDForDogfood(slashType, infractionHeight)
	slashInfo, err := suite.App.OperatorKeeper.GetOperatorSlashInfo(suite.Ctx, suite.DogfoodAVSAddr, suite.Operators[0].String(), slashID)
	suite.Require().NoError(err)
	suite.Require().Equal(slashFactor, slashInfo.SlashProportion)
	suite.Require().Equal(testMaxSlashProportion, slashInfo.ExecutionInfo.SlashProportion)
}

func (suite *OperatorTestSuite) TestVetoSlash() {
	testGenesisStakerIndex := 0
	// prepare a pending undelegation for slashing veto
	undelegationAmount := suite.Powers[testGenesisStakerIndex] / 2
	undelegationAmountBigInt := sdkmath.NewIntWithDecimal(undelegationAmount, int(suite.Assets[testGenesisStakerIndex].Decimals))
	suite.Delegation(false, suite.ClientChains[0].LayerZeroChainID, common.Address(suite.Operators[testGenesisStakerIndex]), common.HexToAddress(suite.Assets[0].Address), suite.Operators[testGenesisStakerIndex], undelegationAmountBigInt)

	infractionHeight := suite.Ctx.BlockHeight()
	suite.NextBlock()
	power := suite.Powers[testGenesisStakerIndex]
	slashFactor := suite.App.SlashingKeeper.SlashFractionDowntime(suite.Ctx)
	slashType := stakingtypes.Infraction_INFRACTION_DOWNTIME
	suite.App.OperatorKeeper.SlashWithInfractionReason(suite.Ctx, suite.Operators[0], infractionHeight, power, slashFactor, slashType)

	// check the states after the slash
	undelegations, err := suite.App.DelegationKeeper.GetStakerUndelegationRecords(suite.Ctx, suite.StakerIDs[testGenesisStakerIndex], suite.AssetIDs[0])
	suite.Require().NoError(err)
	suite.Require().Equal(1, len(undelegations))
	slashedAmountFromUndelegation := slashFactor.MulInt(undelegationAmountBigInt).TruncateInt()
	actualCompletedAmount := undelegationAmountBigInt.Sub(slashedAmountFromUndelegation)
	suite.Require().Equal(actualCompletedAmount, undelegations[0].Undelegation.ActualCompletedAmount)

	assetsInfo, err := suite.App.AssetsKeeper.GetOperatorSpecifiedAssetInfo(suite.Ctx, suite.Operators[testGenesisStakerIndex], suite.AssetIDs[0])
	suite.Require().NoError(err)
	assetPoolAmountAfterUndelegation := sdkmath.NewIntWithDecimal(suite.Powers[testGenesisStakerIndex]-undelegationAmount, int(suite.Assets[testGenesisStakerIndex].Decimals))
	slashedAmountFromAssetsPool := slashFactor.MulInt(assetPoolAmountAfterUndelegation).TruncateInt()
	expectedAssetsPoolAmount := assetPoolAmountAfterUndelegation.Sub(slashedAmountFromAssetsPool)
	suite.Require().Equal(expectedAssetsPoolAmount, assetsInfo.TotalAmount)
	suite.Require().Equal(sdk.NewDecFromInt(assetPoolAmountAfterUndelegation), assetsInfo.TotalShare)

	stakerAssetInfo, err := suite.App.AssetsKeeper.GetStakerSpecifiedAssetInfo(suite.Ctx, suite.StakerIDs[testGenesisStakerIndex], suite.AssetIDs[0])
	suite.Require().NoError(err)
	suite.Require().Equal(sdkmath.ZeroInt(), stakerAssetInfo.WithdrawableAmount)

	// check the slash staker share snapshot
	slashStakerShareSnapshots, err := suite.App.OperatorKeeper.GetAllSlashStakerShareSnapshot(suite.Ctx)
	suite.Require().NoError(err)
	suite.Require().Equal(1, len(slashStakerShareSnapshots))
	slashID := keeper.GetSlashIDForDogfood(slashType, infractionHeight)
	suite.Require().Equal(string(utils.GetJoinedStoreKey(slashID, suite.AssetIDs[0], suite.StakerIDs[testGenesisStakerIndex])), slashStakerShareSnapshots[0].Key)
	expectedStakingUndelegatableShare := sdk.NewDecFromInt(sdkmath.NewIntWithDecimal(suite.Powers[testGenesisStakerIndex]-undelegationAmount, int(suite.Assets[0].Decimals)))
	suite.Require().Equal(expectedStakingUndelegatableShare, slashStakerShareSnapshots[0].Value.StakingUndelegatableShare)
	suite.Require().Equal(0, len(slashStakerShareSnapshots[0].Value.RewardUndelegatableShareBreakdown))

	slashInfo, err := suite.App.OperatorKeeper.GetOperatorSlashInfo(suite.Ctx, suite.DogfoodAVSAddr, suite.Operators[testGenesisStakerIndex].String(), slashID)
	suite.Require().NoError(err)
	suite.Require().Equal(1, len(slashInfo.ExecutionInfo.SlashUndelegations))
	suite.Require().Equal(types.SlashFromUndelegation{
		StakerID:       suite.StakerIDs[testGenesisStakerIndex],
		AssetID:        suite.AssetIDs[0],
		Amount:         slashedAmountFromUndelegation,
		UndelegationId: 0,
	}, slashInfo.ExecutionInfo.SlashUndelegations[0])
	suite.Require().Equal(1, len(slashInfo.ExecutionInfo.SlashAssetsPool))
	suite.Require().Equal(types.SlashFromAssetPool{
		AssetID:            suite.AssetIDs[0],
		TotalAmount:        slashedAmountFromAssetsPool,
		SnapshotTotalShare: sdk.NewDecFromInt(assetPoolAmountAfterUndelegation),
	}, slashInfo.ExecutionInfo.SlashAssetsPool[0])

	// check the slash execution info
	// advance some blocks to test the slash veto
	for i := 0; i < 10; i++ {
		suite.NextBlock()
	}

	// veto the slash
	vetoReason := "test veto reason"
	err = suite.App.OperatorKeeper.VetoSlash(suite.Ctx, suite.DogfoodAVSAddr, suite.Operators[testGenesisStakerIndex].String(), slashID, vetoReason)
	suite.Require().NoError(err)

	slashInfo, err = suite.App.OperatorKeeper.GetOperatorSlashInfo(suite.Ctx, suite.DogfoodAVSAddr, suite.Operators[testGenesisStakerIndex].String(), slashID)
	suite.Require().NoError(err)
	suite.Require().True(slashInfo.IsVetoed, "the slash should be vetoed")
	// check the slash veto
	// the slash veto won't return the fund to the pending undelegation, but will return it to the staker's withdrawable amount directly. So the actual completed amount should still remain slashed.
	undelegations, err = suite.App.DelegationKeeper.GetStakerUndelegationRecords(suite.Ctx, suite.StakerIDs[testGenesisStakerIndex], suite.AssetIDs[0])
	suite.Require().NoError(err)
	suite.Require().Equal(1, len(undelegations))
	suite.Require().Equal(actualCompletedAmount, undelegations[0].Undelegation.ActualCompletedAmount)
	// check if the vetoed fund is returned to the staker's withdrawable amount directly.
	staker, err := suite.App.AssetsKeeper.GetStakerSpecifiedAssetInfo(suite.Ctx, suite.StakerIDs[testGenesisStakerIndex], suite.AssetIDs[0])
	suite.Require().NoError(err)
	// both the vetoed funds from the undelegation and from the asset pool are returned to the staker's withdrawable amount directly.
	suite.Require().Equal(slashedAmountFromAssetsPool.Add(slashedAmountFromUndelegation), staker.WithdrawableAmount)

	// the operator's asset pool should not be affected by the slash veto because the vetoed funds are
	// returned to the related staker's withdrawable amount directly.
	assetsInfo, err = suite.App.AssetsKeeper.GetOperatorSpecifiedAssetInfo(suite.Ctx, suite.Operators[testGenesisStakerIndex], suite.AssetIDs[0])
	suite.Require().NoError(err)
	suite.Require().Equal(expectedAssetsPoolAmount, assetsInfo.TotalAmount)
	suite.Require().Equal(sdk.NewDecFromInt(assetPoolAmountAfterUndelegation), assetsInfo.TotalShare)
}
