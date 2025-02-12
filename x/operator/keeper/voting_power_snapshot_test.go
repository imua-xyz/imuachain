package keeper_test

import (
	"encoding/json"
	"fmt"
	"math/big"
	"sort"
	"time"

	avstypes "github.com/ExocoreNetwork/exocore/x/avs/types"

	sdkmath "cosmossdk.io/math"

	testutiltx "github.com/ExocoreNetwork/exocore/testutil/tx"
	epochstypes "github.com/ExocoreNetwork/exocore/x/epochs/types"
	"github.com/ExocoreNetwork/exocore/x/operator/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
)

var (
	operatorNumber      = 3
	blockNumberPerEpoch = int64(3)
)

type testHelperInfo struct {
	depositAmount  sdkmath.Int
	delegateAmount sdkmath.Int
	operators      []sdk.AccAddress
	stakers        []common.Address
}

func (suite *OperatorTestSuite) prepareForSnapshotTesting(operatorNumber int) testHelperInfo {
	// set default client chainID and asset
	suite.clientChainLzID = defaultClientChainID
	// prepare AVS
	suite.prepareAvs(defaultAVSName, []string{usdtAssetID}, epochstypes.DayEpochID, defaultUnbondingPeriod)
	// prepare stakers and operators
	operators := make([]sdk.AccAddress, operatorNumber)
	stakers := make([]common.Address, operatorNumber)

	decimalAmount := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(assetDecimal)), nil)
	depositAmount := sdkmath.NewInt(10000).Mul(sdkmath.NewIntFromBigInt(decimalAmount))
	delegateAmount := sdkmath.NewInt(1000).Mul(sdkmath.NewIntFromBigInt(decimalAmount))
	for i := 0; i < operatorNumber; i++ {
		ethAddr := testutiltx.GenerateAddress()
		stakers[i] = ethAddr
		operators[i] = ethAddr.Bytes()
		// register operator
		suite.registerOperator(operators[i].String())
		// associate the stakers with operators
		err := suite.App.DelegationKeeper.AssociateOperatorWithStaker(suite.Ctx, suite.clientChainLzID, operators[i], stakers[i].Bytes())
		suite.NoError(err)
		// deposit assets
		suite.prepareDeposit(stakers[i], usdtAddr, depositAmount)
		// delegate assets
		suite.prepareDelegation(true, stakers[i], usdtAddr, operators[i], delegateAmount)
		// opt in the test AVS
		err = suite.App.OperatorKeeper.OptIn(suite.Ctx, operators[i], suite.avsAddr)
		suite.NoError(err)
		// opt in the dogfood AVS
		chainIDWithoutRevision := avstypes.ChainIDWithoutRevision(suite.Ctx.ChainID())
		dogfoodAVSAddr := avstypes.GenerateAVSAddress(chainIDWithoutRevision)
		pubKey := testutiltx.GenerateConsensusKey()
		suite.Require().NotNil(pubKey)
		err = suite.App.OperatorKeeper.OptInWithConsKey(suite.Ctx, operators[i], dogfoodAVSAddr, pubKey)
		suite.NoError(err)
	}
	sort.Slice(stakers, func(i, j int) bool {
		return operators[i].String() < operators[j].String()
	})
	sort.Slice(operators, func(i, j int) bool {
		return operators[i].String() < operators[j].String()
	})

	return testHelperInfo{
		depositAmount:  depositAmount,
		delegateAmount: delegateAmount,
		operators:      operators,
		stakers:        stakers,
	}
}

func (suite *OperatorTestSuite) runToEpochEnd() {
	// the default AVS epoch identifier is day
	// Configure 3 blocks per epoch for testing, so the block duration is 8 hours
	// so starting from the initial block of the epoch, it takes three blocks to
	// reach the epoch’s end block.
	for i := int64(0); i < blockNumberPerEpoch; i++ {
		suite.CommitAfter(8 * time.Hour)
	}
}

func (suite *OperatorTestSuite) printAllSnapshot(avs string) {
	epochInfo, found := suite.App.EpochsKeeper.GetEpochInfo(suite.Ctx, epochstypes.DayEpochID)
	suite.True(found)
	fmt.Println("epoch", epochInfo.CurrentEpoch, "startHeight", epochInfo.CurrentEpochStartHeight)
	opFunc := func(height int64, snapshot *types.VotingPowerSnapshot) error {
		fmt.Println("snapshot height is：", height)
		bytes, err := json.MarshalIndent(snapshot, " ", " ")
		suite.NoError(err)
		fmt.Println(string(bytes))
		return nil
	}
	err := suite.App.OperatorKeeper.IterateVotingPowerSnapshot(suite.Ctx, avs, false, opFunc)
	suite.NoError(err)
}

func (suite *OperatorTestSuite) TestInitializeSnapshot() {
	helperInfo := suite.prepareForSnapshotTesting(operatorNumber)
	suite.runToEpochEnd()
	epochInfo, found := suite.App.EpochsKeeper.GetEpochInfo(suite.Ctx, epochstypes.DayEpochID)
	suite.True(found)
	// the height in the snapshot key should be the start height of next epoch.
	snapshotHeight, snapshot, err := suite.App.OperatorKeeper.LoadVotingPowerSnapshot(suite.Ctx, suite.avsAddr, epochInfo.CurrentEpochStartHeight)
	suite.NoError(err)
	suite.Equal(epochInfo.CurrentEpochStartHeight, snapshotHeight)
	avsUSDValue, err := suite.App.OperatorKeeper.GetAVSUSDValue(suite.Ctx, suite.avsAddr)
	suite.NoError(err)
	suite.Equal(avsUSDValue, snapshot.TotalVotingPower)

	expectedVotingPowerSet := make([]*types.OperatorVotingPower, operatorNumber)
	for i := 0; i < operatorNumber; i++ {
		expectedVotingPowerSet[i] = &types.OperatorVotingPower{
			OperatorAddr: helperInfo.operators[i].String(),
			VotingPower:  avsUSDValue.Quo(sdkmath.LegacyNewDec(int64(operatorNumber))),
		}
	}
	expectedSnapshot := types.VotingPowerSnapshot{
		TotalVotingPower:     avsUSDValue,
		OperatorVotingPowers: expectedVotingPowerSet,
		LastChangedHeight:    epochInfo.CurrentEpochStartHeight,
		EpochIdentifier:      epochInfo.Identifier,
		EpochNumber:          epochInfo.CurrentEpoch,
	}
	suite.Equal(expectedSnapshot, *snapshot)

	snapshotHelper, err := suite.App.OperatorKeeper.GetSnapshotHelper(suite.Ctx, suite.avsAddr)
	suite.NoError(err)
	suite.Equal(types.SnapshotHelper{
		LastChangedHeight: snapshot.LastChangedHeight,
	}, snapshotHelper)

	_, _, err = suite.App.OperatorKeeper.LoadVotingPowerSnapshot(suite.Ctx, suite.avsAddr, epochInfo.CurrentEpochStartHeight-1)
	suite.Error(err)
}

func (suite *OperatorTestSuite) TestSnapshotVPUnchanged() {
	suite.prepareForSnapshotTesting(operatorNumber)
	suite.runToEpochEnd()
	lastChangeHeight := suite.Ctx.BlockHeight()
	_, initialSnapshot, err := suite.App.OperatorKeeper.LoadVotingPowerSnapshot(suite.Ctx, suite.avsAddr, lastChangeHeight)
	suite.NoError(err)
	runToEpochNumber := 2

	for i := 0; i < runToEpochNumber; i++ {
		suite.runToEpochEnd()
		epochInfo, found := suite.App.EpochsKeeper.GetEpochInfo(suite.Ctx, epochstypes.DayEpochID)
		suite.True(found)
		startHeight := epochInfo.CurrentEpochStartHeight
		endHeight := startHeight + blockNumberPerEpoch - 1
		for j := startHeight; j <= endHeight; j++ {
			snapshotKeyLastHeight, snapshot, err := suite.App.OperatorKeeper.LoadVotingPowerSnapshot(suite.Ctx, suite.avsAddr, j)
			suite.NoError(err)
			suite.Equal(startHeight, snapshotKeyLastHeight)
			suite.Equal(initialSnapshot, snapshot)

			snapshotHelper, err := suite.App.OperatorKeeper.GetSnapshotHelper(suite.Ctx, suite.avsAddr)
			suite.NoError(err)
			suite.Equal(types.SnapshotHelper{
				LastChangedHeight: snapshot.LastChangedHeight,
			}, snapshotHelper)

			if j != startHeight {
				key := types.KeyForVotingPowerSnapshot(suite.assetAddr, j)
				_, err = suite.App.OperatorKeeper.GetVotingPowerSnapshot(suite.Ctx, key)
				suite.Error(err)
			}
		}
	}
}

func (suite *OperatorTestSuite) TestSnapshotVPChanged() {
	testHelper := suite.prepareForSnapshotTesting(operatorNumber)
	suite.runToEpochEnd()
	_, initialSnapshot, err := suite.App.OperatorKeeper.LoadVotingPowerSnapshot(suite.Ctx, suite.avsAddr, suite.Ctx.BlockHeight())
	suite.NoError(err)

	// change the voting power of the operator at index 0.
	index := 0
	suite.prepareDelegation(true, testHelper.stakers[index], usdtAddr, testHelper.operators[index], testHelper.delegateAmount)
	suite.runToEpochEnd()
	_, snapshotAfterUpdate, err := suite.App.OperatorKeeper.LoadVotingPowerSnapshot(suite.Ctx, suite.avsAddr, suite.Ctx.BlockHeight())
	suite.NoError(err)

	addVotingPower := initialSnapshot.OperatorVotingPowers[index].VotingPower
	expectedTotalVotingPower := initialSnapshot.TotalVotingPower.Add(addVotingPower)
	expectedVotingPowerSet := initialSnapshot.OperatorVotingPowers
	expectedVotingPowerSet[index].VotingPower = expectedVotingPowerSet[index].VotingPower.Add(addVotingPower)
	suite.Equal(expectedTotalVotingPower, snapshotAfterUpdate.TotalVotingPower)
	suite.Equal(expectedVotingPowerSet, snapshotAfterUpdate.OperatorVotingPowers)
	suite.Equal(initialSnapshot.EpochNumber+1, snapshotAfterUpdate.EpochNumber)
}

func (suite *OperatorTestSuite) TestSnapshotWithOptOut() {
	testHelper := suite.prepareForSnapshotTesting(operatorNumber)
	suite.runToEpochEnd()

	// opt out if the index of operator is 0.
	index := 0
	err := suite.App.OperatorKeeper.OptOut(suite.Ctx, testHelper.operators[index], suite.avsAddr)
	suite.NoError(err)
	snapshotHelper, err := suite.App.OperatorKeeper.GetSnapshotHelper(suite.Ctx, suite.avsAddr)
	suite.NoError(err)
	suite.True(snapshotHelper.HasOptOut)

	suite.runToEpochEnd()
	_, snapshot, err := suite.App.OperatorKeeper.LoadVotingPowerSnapshot(suite.Ctx, suite.avsAddr, suite.Ctx.BlockHeight())
	suite.NoError(err)
	suite.Equal(operatorNumber-1, len(snapshot.OperatorVotingPowers))
	votingPower := types.GetSpecifiedVotingPower(testHelper.operators[index].String(), snapshot.OperatorVotingPowers)
	suite.Nil(votingPower)

	snapshotHelper, err = suite.App.OperatorKeeper.GetSnapshotHelper(suite.Ctx, suite.avsAddr)
	suite.NoError(err)
	suite.False(snapshotHelper.HasOptOut)

	// opt all operators out of the AVS.
	for i := index + 1; i < operatorNumber; i++ {
		err = suite.App.OperatorKeeper.OptOut(suite.Ctx, testHelper.operators[i], suite.avsAddr)
		suite.NoError(err)
	}
	suite.runToEpochEnd()
	_, snapshot, err = suite.App.OperatorKeeper.LoadVotingPowerSnapshot(suite.Ctx, suite.avsAddr, suite.Ctx.BlockHeight())
	suite.NoError(err)
	suite.Equal(suite.Ctx.BlockHeight(), snapshot.LastChangedHeight)
	suite.Equal(0, len(snapshot.OperatorVotingPowers))
	for i := 0; i < operatorNumber; i++ {
		votingPower = types.GetSpecifiedVotingPower(testHelper.operators[i].String(), snapshot.OperatorVotingPowers)
		suite.Nil(votingPower)
	}
	suite.runToEpochEnd()
	key := types.KeyForVotingPowerSnapshot(common.HexToAddress(suite.avsAddr), suite.Ctx.BlockHeight())
	_, err = suite.App.OperatorKeeper.GetVotingPowerSnapshot(suite.Ctx, key)
	suite.Error(err)
}

func (suite *OperatorTestSuite) TestSnapshotWithSlash() {
	testHelper := suite.prepareForSnapshotTesting(operatorNumber)
	suite.runToEpochEnd()
	_, snapshotBeforeSlash, err := suite.App.OperatorKeeper.LoadVotingPowerSnapshot(suite.Ctx, suite.avsAddr, suite.Ctx.BlockHeight())
	suite.NoError(err)
	// run to next block to execute slashing
	index := 0
	suite.CommitAfter(8 * time.Hour)
	slashProportion := sdkmath.LegacyMustNewDecFromStr("0.1")
	remainingProportion := sdkmath.LegacyNewDec(1).Sub(slashProportion)
	slashParam := &types.SlashInputInfo{
		IsDogFood:        false,
		Power:            0,
		SlashType:        0,
		Operator:         testHelper.operators[index],
		AVSAddr:          suite.avsAddr,
		SlashID:          "testSlashID",
		SlashEventHeight: suite.Ctx.BlockHeight(),
		SlashProportion:  slashProportion,
	}
	err = suite.App.OperatorKeeper.Slash(suite.Ctx, slashParam)
	suite.NoError(err)
	_, snapshotAfterSlash, err := suite.App.OperatorKeeper.LoadVotingPowerSnapshot(suite.Ctx, suite.avsAddr, suite.Ctx.BlockHeight())
	suite.NoError(err)
	suite.Equal(suite.Ctx.BlockHeight(), snapshotAfterSlash.LastChangedHeight)
	operatorVPBeforeSlash := types.GetSpecifiedVotingPower(testHelper.operators[index].String(), snapshotBeforeSlash.OperatorVotingPowers)
	operatorVPAfterSlash := types.GetSpecifiedVotingPower(testHelper.operators[index].String(), snapshotAfterSlash.OperatorVotingPowers)
	suite.Equal(operatorVPBeforeSlash.VotingPower.Mul(remainingProportion), operatorVPAfterSlash.VotingPower)

	snapshotHelper, err := suite.App.OperatorKeeper.GetSnapshotHelper(suite.Ctx, suite.avsAddr)
	suite.NoError(err)
	suite.Equal(suite.Ctx.BlockHeight(), snapshotHelper.LastChangedHeight)

	shouldUpdateValidatorSet := suite.App.StakingKeeper.ShouldUpdateValidatorSet(suite.Ctx)
	suite.True(shouldUpdateValidatorSet)
}

func (suite *OperatorTestSuite) TestGenesisSnapshot() {
	suite.prepareForSnapshotTesting(operatorNumber)
	firstBlockHeight := suite.Ctx.BlockHeight()
	chainIDWithoutRevision := avstypes.ChainIDWithoutRevision(suite.Ctx.ChainID())
	dogfoodAVSAddr := avstypes.GenerateAVSAddress(chainIDWithoutRevision)
	for i := int64(0); i < blockNumberPerEpoch; i++ {
		height, snapshot, err := suite.App.OperatorKeeper.LoadVotingPowerSnapshot(suite.Ctx, dogfoodAVSAddr, firstBlockHeight)
		suite.NoError(err)
		suite.Equal(firstBlockHeight, height)
		suite.Equal(firstBlockHeight, snapshot.LastChangedHeight)
		snapshotHelper, err := suite.App.OperatorKeeper.GetSnapshotHelper(suite.Ctx, dogfoodAVSAddr)
		suite.NoError(err)
		suite.Equal(firstBlockHeight, snapshotHelper.LastChangedHeight)
	}
}

func (suite *OperatorTestSuite) TestSnapshotPruning() {
	testHelper := suite.prepareForSnapshotTesting(operatorNumber)
	suite.runToEpochEnd()
	firstSnapshotHeight := suite.Ctx.BlockHeight()

	avsUnbondingDuration, err := suite.App.AVSManagerKeeper.GetAVSUnbondingDuration(suite.Ctx, suite.avsAddr)
	suite.NoError(err)

	runEpochNumber := avsUnbondingDuration + 2
	for i := uint64(0); i < runEpochNumber; i++ {
		suite.runToEpochEnd()
	}
	_, _, err = suite.App.OperatorKeeper.LoadVotingPowerSnapshot(suite.Ctx, suite.avsAddr, firstSnapshotHeight)
	suite.NoError(err)

	key := types.KeyForVotingPowerSnapshot(common.HexToAddress(suite.avsAddr), firstSnapshotHeight+blockNumberPerEpoch)
	_, err = suite.App.OperatorKeeper.GetVotingPowerSnapshot(suite.Ctx, key)
	suite.Error(err)

	// change the voting power of the operator at index 0.
	index := 0
	suite.prepareDelegation(true, testHelper.stakers[index], usdtAddr, testHelper.operators[index], testHelper.delegateAmount)
	for i := uint64(0); i < runEpochNumber; i++ {
		suite.runToEpochEnd()
	}
	key = types.KeyForVotingPowerSnapshot(common.HexToAddress(suite.avsAddr), firstSnapshotHeight)
	_, err = suite.App.OperatorKeeper.GetVotingPowerSnapshot(suite.Ctx, key)
	suite.Error(err)

	// Test pruning of slash-created snapshots
	slashParam := &types.SlashInputInfo{
		IsDogFood:        false,
		Power:            0,
		SlashType:        0,
		Operator:         testHelper.operators[index],
		AVSAddr:          suite.avsAddr,
		SlashID:          "testSlashID",
		SlashEventHeight: suite.Ctx.BlockHeight(),
		SlashProportion:  sdkmath.LegacyMustNewDecFromStr("0.1"),
	}
	err = suite.App.OperatorKeeper.Slash(suite.Ctx, slashParam)
	suite.NoError(err)
	_, snapshotAfterSlash, err := suite.App.OperatorKeeper.LoadVotingPowerSnapshot(suite.Ctx, suite.avsAddr, suite.Ctx.BlockHeight())
	suite.NoError(err)
	suite.Equal(suite.Ctx.BlockHeight(), snapshotAfterSlash.LastChangedHeight)
	suite.runToEpochEnd()
	suite.prepareDelegation(true, testHelper.stakers[index], usdtAddr, testHelper.operators[index], testHelper.delegateAmount)
	for i := uint64(0); i < runEpochNumber; i++ {
		suite.runToEpochEnd()
	}
	key = types.KeyForVotingPowerSnapshot(common.HexToAddress(suite.avsAddr), snapshotAfterSlash.LastChangedHeight)
	_, err = suite.App.OperatorKeeper.GetVotingPowerSnapshot(suite.Ctx, key)
	suite.Error(err)
}
