package keeper_test

import (
	"time"

	abci "github.com/cometbft/cometbft/abci/types"

	sdkmath "cosmossdk.io/math"
	avstypes "github.com/ExocoreNetwork/exocore/x/avs/types"
	"github.com/ExocoreNetwork/exocore/x/operator/keeper"
	"github.com/ExocoreNetwork/exocore/x/operator/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
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
	avsAddr := avstypes.GenerateAVSAddress(avstypes.ChainIDWithoutRevision(suite.Ctx.ChainID()))
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
	completedEpochId, completedEpochNumber, err := suite.App.OperatorKeeper.GetUnbondingExpiration(suite.Ctx, suite.operatorAddr)
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
	exoSlashValue := suite.App.OperatorKeeper.SlashWithInfractionReason(suite.Ctx, suite.operatorAddr, infractionHeight, power, slashFactor, slashType)
	suite.Equal(sdkmath.ZeroInt(), exoSlashValue)

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
	suite.Equal(types.SlashFromAssetsPool{
		AssetID: suite.assetID,
		Amount:  newSlashProportion.MulInt(delegationRemaining).TruncateInt(),
	}, slashInfo.ExecutionInfo.SlashAssetsPool[0])
	suite.Equal(undelegationFilterHeight, slashInfo.ExecutionInfo.UndelegationFilterHeight)

	// check the assets state of undelegation and assets pool
	assetsInfo, err := suite.App.AssetsKeeper.GetOperatorSpecifiedAssetInfo(suite.Ctx, suite.operatorAddr, suite.assetID)
	suite.NoError(err)
	suite.Equal(delegationRemaining.Sub(slashInfo.ExecutionInfo.SlashAssetsPool[0].Amount), assetsInfo.TotalAmount)

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
