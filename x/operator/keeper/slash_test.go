package keeper_test

import (
	"time"

	sdkmath "cosmossdk.io/math"
	avstypes "github.com/ExocoreNetwork/exocore/x/avs/types"
	"github.com/ExocoreNetwork/exocore/x/operator/keeper"
	"github.com/ExocoreNetwork/exocore/x/operator/types"
	abci "github.com/cometbft/cometbft/abci/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/common"
)

func (suite *OperatorTestSuite) TestSlashWithInfractionReason() {
	// prepare the deposit and delegation
	suite.prepareOperator()
	usdtAddress := common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7")
	assetDecimal := 6
	depositAmount := sdkmath.NewIntWithDecimal(200, assetDecimal)
	suite.prepareDeposit(usdtAddress, depositAmount)
	delegationAmount := sdkmath.NewIntWithDecimal(100, assetDecimal)
	suite.prepareDelegation(true, suite.assetAddr, delegationAmount)
	err := suite.App.DelegationKeeper.AssociateOperatorWithStaker(suite.Ctx, suite.clientChainLzID, suite.operatorAddr, suite.Address[:])
	suite.NoError(err)

	// opt into the AVS
	avsAddr := avstypes.GenerateAVSAddr(avstypes.ChainIDWithoutRevision(suite.Ctx.ChainID()))
	err = suite.App.OperatorKeeper.OptIn(suite.Ctx, suite.operatorAddr, avsAddr)
	suite.NoError(err)
	// call the EndBlock to update the voting power
	suite.CommitAfter(time.Hour*24 + time.Nanosecond)
	infractionHeight := suite.Ctx.BlockHeight() + 1
	optedUSDValues, err := suite.App.OperatorKeeper.GetOperatorOptedUSDValue(suite.Ctx, avsAddr, suite.operatorAddr.String())
	suite.NoError(err)
	// get the historical voting power
	power := optedUSDValues.TotalUSDValue.TruncateInt64()
	// run to next block
	suite.NextBlock()

	// delegates new amount to the operator
	newDelegateAmount := sdkmath.NewIntWithDecimal(20, assetDecimal)
	suite.prepareDelegation(true, suite.assetAddr, newDelegateAmount)
	// updating the voting power
	suite.CommitAfter(time.Hour*24 + time.Nanosecond)
	newOptedUSDValues, err := suite.App.OperatorKeeper.GetOperatorOptedUSDValue(suite.Ctx, avsAddr, suite.operatorAddr.String())
	suite.NoError(err)
	// submits an undelegation to test the slashFromUndelegation
	undelegationAmount := sdkmath.NewIntWithDecimal(10, assetDecimal)
	suite.prepareDelegation(false, suite.assetAddr, undelegationAmount)
	delegationRemaining := delegationAmount.Add(newDelegateAmount).Sub(undelegationAmount)
	startHeight := uint64(suite.Ctx.BlockHeight())
	completedHeight := suite.App.OperatorKeeper.GetUnbondingExpirationBlockNumber(suite.Ctx, suite.operatorAddr, startHeight)

	// trigger the slash with a downtime event
	slashFactor := suite.App.SlashingKeeper.SlashFractionDowntime(suite.Ctx)
	slashType := stakingtypes.Infraction_INFRACTION_DOWNTIME
	exoSlashValue := suite.App.OperatorKeeper.SlashWithInfractionReason(suite.Ctx, suite.operatorAddr, infractionHeight, power, slashFactor, slashType)
	suite.Equal(sdkmath.NewInt(0), exoSlashValue)

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
	suite.Equal(types.SlashFromUndelegation{
		StakerID: suite.stakerID,
		AssetID:  suite.assetID,
		Amount:   newSlashProportion.MulInt(undelegationAmount).TruncateInt(),
	}, slashInfo.ExecutionInfo.SlashUndelegations[0])
	suite.Equal(types.SlashFromAssetsPool{
		AssetID: suite.assetID,
		Amount:  newSlashProportion.MulInt(delegationRemaining).TruncateInt(),
	}, slashInfo.ExecutionInfo.SlashAssetsPool[0])

	// check the assets state of undelegation and assets pool
	assetsInfo, err := suite.App.AssetsKeeper.GetOperatorSpecifiedAssetInfo(suite.Ctx, suite.operatorAddr, suite.assetID)
	suite.NoError(err)
	suite.Equal(delegationRemaining.Sub(slashInfo.ExecutionInfo.SlashAssetsPool[0].Amount), assetsInfo.TotalAmount)

	undelegations, err := suite.App.DelegationKeeper.GetStakerUndelegationRecords(suite.Ctx, suite.stakerID, suite.assetID)
	suite.NoError(err)
	suite.Equal(undelegationAmount.Sub(slashInfo.ExecutionInfo.SlashUndelegations[0].Amount), undelegations[0].ActualCompletedAmount)

	// run to the block at which the undelegation is completed
	for i := startHeight; i < completedHeight; i++ {
		suite.NextBlock()
	}
	suite.App.DelegationKeeper.EndBlock(suite.Ctx, abci.RequestEndBlock{})
	undelegations, err = suite.App.DelegationKeeper.GetStakerUndelegationRecords(suite.Ctx, suite.stakerID, suite.assetID)
	suite.NoError(err)
	suite.Equal(0, len(undelegations))
}
