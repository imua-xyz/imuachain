package keeper_test

import (
	"fmt"
	"math/rand"
	"slices"
	"testing"
	"time"

	"cosmossdk.io/math"
	sdkmath "cosmossdk.io/math"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	evidencetypes "github.com/cosmos/cosmos-sdk/x/evidence/types"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/imua-xyz/imuachain/testutil"
	testutiltx "github.com/imua-xyz/imuachain/testutil/tx"
	keytypes "github.com/imua-xyz/imuachain/types/keys"
	"github.com/imua-xyz/imuachain/utils"
	assetskeeper "github.com/imua-xyz/imuachain/x/assets/keeper"
	assetstypes "github.com/imua-xyz/imuachain/x/assets/types"
	delegationtypes "github.com/imua-xyz/imuachain/x/delegation/types"
	dogfoodtypes "github.com/imua-xyz/imuachain/x/dogfood/types"
	epochstypes "github.com/imua-xyz/imuachain/x/epochs/types"
	operatorkeeper "github.com/imua-xyz/imuachain/x/operator/keeper"
	operatortypes "github.com/imua-xyz/imuachain/x/operator/types"
	"github.com/stretchr/testify/suite"
)

var a *KeyChangeEscapeTestSuite

type KeyChangeEscapeTestSuite struct {
	testutil.BaseTestSuite
	EpochIdentifier        string
	ChainIDWithoutRevision string
	AvsAddress             string
	MinSelfDelegation      sdkmath.Int
	EpochDuration          time.Duration
}

func TestKeyChangeEscapeTestSuite(t *testing.T) {
	a = new(KeyChangeEscapeTestSuite)
	suite.Run(t, a)
}

func (suite *KeyChangeEscapeTestSuite) SetupTest() {
	suite.DoSetupTest()
	suite.EpochIdentifier = suite.App.StakingKeeper.GetEpochIdentifier(
		suite.Ctx,
	)
	epochInfo, _ := suite.App.EpochsKeeper.GetEpochInfo(
		suite.Ctx, suite.EpochIdentifier,
	)
	suite.EpochDuration = epochInfo.Duration + time.Nanosecond
	suite.ChainIDWithoutRevision = utils.ChainIDWithoutRevision(
		suite.Ctx.ChainID(),
	)
	suite.AvsAddress = utils.GenerateAVSAddress(
		suite.ChainIDWithoutRevision,
	)
	suite.MinSelfDelegation = suite.App.StakingKeeper.GetDogfoodParams(
		suite.Ctx,
	).MinSelfDelegation
	// move a few blocks ahead
	suite.RunBlocks(10)
}

// Registers an operator with the given address
func (suite *KeyChangeEscapeTestSuite) RegisterOperator(
	operatorAddr sdk.AccAddress,
) {
	{
		res, err := suite.OperatorMsgServer.RegisterOperator(
			sdk.WrapSDKContext(suite.Ctx),
			&operatortypes.RegisterOperatorReq{
				FromAddress: operatorAddr.String(),
				Info: &operatortypes.OperatorInfo{
					OperatorAddr: operatorAddr.String(),
					Description: stakingtypes.NewDescription(
						// unique name
						fmt.Sprintf("operator%d", time.Now().UnixNano()),
						"", "", "", "",
					),
					Commission: stakingtypes.NewCommission(
						sdk.ZeroDec(), sdk.ZeroDec(), sdk.ZeroDec(),
					),
				},
			},
		)
		suite.Require().NoError(err)
		suite.Require().NotNil(res)
	}
}

func (suite *KeyChangeEscapeTestSuite) DepositFromStaker(
	stakerAddr common.Address,
	humanReadableUsd int64,
) {
	assetID := suite.AssetIDs[0]
	stakerID, _ := assetstypes.GetStakerIDAndAssetIDFromStr(
		suite.Assets[0].LayerZeroChainID,
		stakerAddr.String(),
		"",
	)
	decimalsUsd := math.NewIntWithDecimal(
		humanReadableUsd, int(suite.Assets[0].Decimals),
	)
	openState, err := suite.App.AssetsKeeper.GetStakerSpecifiedAssetInfo(
		suite.Ctx, stakerID, assetID,
	)
	if err != nil {
		suite.Require().ErrorIs(err, assetstypes.ErrNoStakerAssetKey)
		openState = &assetstypes.StakerAssetInfo{
			TotalDepositAmount:        sdkmath.ZeroInt(),
			WithdrawableAmount:        sdkmath.ZeroInt(),
			PendingUndelegationAmount: sdkmath.ZeroInt(),
		}
	}
	depositParams := &assetskeeper.DepositWithdrawParams{
		ClientChainLzID: suite.Assets[0].LayerZeroChainID,
		Action:          assetstypes.DepositLST,
		AssetsAddress:   common.HexToAddress(suite.Assets[0].Address).Bytes(),
		StakerAddress:   stakerAddr.Bytes(),
		OpAmount:        decimalsUsd,
	}
	closeState, err := suite.App.AssetsKeeper.PerformDepositOrWithdraw(
		suite.Ctx, depositParams,
	)
	suite.Require().NoError(err)
	suite.Require().Equal(openState.TotalDepositAmount.Add(decimalsUsd), closeState)
}

func (suite *KeyChangeEscapeTestSuite) DelegateToOperator(
	delegatorAddr common.Address,
	operatorAddr sdk.AccAddress,
	humanReadableUsd int64,
) {
	decimalsUsd := math.NewIntWithDecimal(
		humanReadableUsd, int(suite.Assets[0].Decimals),
	)
	delegateParams := delegationtypes.NewDelegationOrUndelegationParams(
		suite.Assets[0].LayerZeroChainID,
		common.HexToAddress(suite.Assets[0].Address).Bytes(),
		operatorAddr,
		delegatorAddr.Bytes(),
		decimalsUsd,
		// tx hash
		common.BytesToHash([]byte("test")),
		// is instant unbonding, no effect when delegating
		false,
	)
	_, _, err := suite.App.DelegationKeeper.DelegateTo(suite.Ctx, delegateParams)
	suite.Require().NoError(err)
}

func (suite *KeyChangeEscapeTestSuite) AssociateOperatorWithStaker(
	stakerAddr common.Address,
	operatorAddr sdk.AccAddress,
) {
	err := suite.App.DelegationKeeper.AssociateOperatorWithStaker(
		suite.Ctx, suite.Assets[0].LayerZeroChainID,
		operatorAddr, stakerAddr.Bytes(),
	)
	suite.Require().NoError(err)
}

func (suite *KeyChangeEscapeTestSuite) OptIn(
	operatorAddr sdk.AccAddress,
	consKey keytypes.WrappedConsKey,
) {
	res, err := suite.OperatorMsgServer.OptIntoAVS(
		sdk.WrapSDKContext(suite.Ctx),
		&operatortypes.OptIntoAVSReq{
			FromAddress:   operatorAddr.String(),
			AvsAddress:    suite.AvsAddress,
			PublicKeyJSON: consKey.ToJSON(),
		},
	)
	suite.Require().NotNil(res)
	suite.Require().NoError(err)
}

func (suite *KeyChangeEscapeTestSuite) OptOut(
	operatorAddr sdk.AccAddress,
) {
	res, err := suite.OperatorMsgServer.OptOutOfAVS(
		sdk.WrapSDKContext(suite.Ctx),
		&operatortypes.OptOutOfAVSReq{
			FromAddress: operatorAddr.String(),
			AvsAddress:  suite.AvsAddress,
		},
	)
	suite.Require().NotNil(res)
	suite.Require().NoError(err)
}

func (suite *KeyChangeEscapeTestSuite) UndelegateFromOperator(
	delegatorAddr common.Address,
	operatorAddr sdk.AccAddress,
	humanReadableUsd int64,
) {
	decimalsUsd := math.NewIntWithDecimal(
		humanReadableUsd, int(suite.Assets[0].Decimals),
	)
	undelegateParams := delegationtypes.NewDelegationOrUndelegationParams(
		suite.Assets[0].LayerZeroChainID,
		common.HexToAddress(suite.Assets[0].Address).Bytes(),
		operatorAddr,
		delegatorAddr.Bytes(),
		decimalsUsd,
		// tx hash
		common.BytesToHash([]byte("test_undelegate")),
		// is instant unbonding, no effect when delegating
		false,
	)
	err := suite.App.DelegationKeeper.UndelegateFrom(suite.Ctx, undelegateParams)
	suite.Require().NoError(err)
}

func (suite *KeyChangeEscapeTestSuite) CheckOperatorUSDValueExact(
	operatorAddr sdk.AccAddress,
	expectedUsdValue sdkmath.LegacyDec,
) {
	usd, err := suite.App.OperatorKeeper.GetOrCalculateOperatorUSDValues(
		suite.Ctx, operatorAddr, suite.AvsAddress,
	)
	suite.Require().NoError(err)
	suite.Require().Equal(expectedUsdValue, usd.SelfUSDValue)
}

func (suite *KeyChangeEscapeTestSuite) CheckTombstoned(
	consAddr sdk.ConsAddress,
	expectedTombstoned bool,
) {
	signInfo, found := suite.App.SlashingKeeper.GetValidatorSigningInfo(suite.Ctx, consAddr)
	if expectedTombstoned {
		suite.Require().True(found)
		suite.Require().True(signInfo.Tombstoned)
		suite.Require().Equal(signInfo.JailedUntil.UTC(), evidencetypes.DoubleSignJailEndTime.UTC())
	} else {
		if found {
			suite.Require().False(signInfo.Tombstoned)
		}
	}
}

func (suite *KeyChangeEscapeTestSuite) RunBlocks(
	numBlocks int,
) {
	for i := 0; i < numBlocks; i++ {
		suite.Commit()
	}
}

// Adds a validator
// contract: must not already exist!
func (suite *KeyChangeEscapeTestSuite) AddValidator(
	factor int64,
) (
	sdk.AccAddress, keytypes.WrappedConsKey, int64,
) {
	operatorAddr, _ := testutiltx.NewAccAddressAndKey()
	consKey := testutiltx.GenerateConsensusKey()
	stakerAddr := common.Address(operatorAddr.Bytes())
	suite.RegisterOperator(operatorAddr)
	humanReadableUsd := suite.MinSelfDelegation.Int64() * factor
	suite.DepositFromStaker(
		stakerAddr, humanReadableUsd,
	)
	suite.DelegateToOperator(
		stakerAddr, operatorAddr,
		suite.MinSelfDelegation.Int64()*factor,
	)
	suite.AssociateOperatorWithStaker(
		stakerAddr, operatorAddr,
	)
	suite.OptIn(operatorAddr, consKey)
	// check power
	return operatorAddr, consKey, humanReadableUsd
}

func (suite *KeyChangeEscapeTestSuite) CheckSlashEffect(
	operatorAddr sdk.AccAddress,
	slashProportion sdkmath.LegacyDec,
	startingPower int64,
) (endingPower int64) {
	valAddr := sdk.ValAddress(operatorAddr)
	validator, found := suite.App.StakingKeeper.GetValidator(
		suite.Ctx, valAddr,
	)
	suite.Require().True(found)
	slashValue := sdkmath.LegacyNewDec(startingPower).Mul(slashProportion)
	effectiveSlashProportion := sdkmath.LegacyMinDec(
		sdkmath.LegacyNewDec(1), slashValue.QuoInt64(startingPower),
	)
	subtract := effectiveSlashProportion.MulInt64(startingPower)
	endingPower = sdkmath.LegacyNewDec(startingPower).Sub(subtract).TruncateInt64()
	delegation := suite.App.StakingKeeper.Delegation(
		suite.Ctx, operatorAddr, valAddr,
	)
	delegationTokens := validator.TokensFromShares(
		delegation.GetShares(),
	).TruncateInt()
	delegationPower := sdk.TokensToConsensusPower(
		delegationTokens, sdk.DefaultPowerReduction,
	)
	suite.Require().Equal(endingPower, delegationPower)
	return
}

func (suite *KeyChangeEscapeTestSuite) CheckValidator(
	expectedAccAddr sdk.AccAddress,
	expectedConsAddr sdk.ConsAddress,
	expectedPower int64,
	expectedSelfDelegationPower int64,
) {
	// forward lookup
	found, key, err := suite.App.OperatorKeeper.GetOperatorConsKeyForChainID(
		suite.Ctx, expectedAccAddr, suite.ChainIDWithoutRevision,
	)
	suite.Require().True(found)
	suite.Require().NoError(err)
	suite.Require().Equal(expectedConsAddr, key.ToConsAddr())
	// reverse lookup
	found, operatorAddr := suite.App.OperatorKeeper.GetOperatorAddressForChainIDAndConsAddr(
		suite.Ctx, suite.ChainIDWithoutRevision, expectedConsAddr,
	)
	suite.Require().True(found)
	suite.Require().Equal(expectedAccAddr, operatorAddr)
	// power lookup - use SDK method instead of our own
	validator, found := suite.App.StakingKeeper.GetValidator(
		suite.Ctx, sdk.ValAddress(expectedAccAddr),
	)
	suite.Require().True(found)
	tokens := validator.GetTokens()
	validatorPower := sdk.TokensToConsensusPower(tokens, sdk.DefaultPowerReduction)
	suite.Require().Equal(expectedPower, validatorPower)
	// Delegation always returns self delegation (associated)
	delegation := suite.App.StakingKeeper.Delegation(
		suite.Ctx, expectedAccAddr, sdk.ValAddress(expectedAccAddr),
	)
	delegationTokens := validator.TokensFromShares(
		delegation.GetShares(),
	).TruncateInt()
	delegationPower := sdk.TokensToConsensusPower(
		delegationTokens, sdk.DefaultPowerReduction,
	)
	suite.Require().Equal(expectedSelfDelegationPower, delegationPower)
}

func (suite *KeyChangeEscapeTestSuite) SubmitEvidence(
	consAddr sdk.ConsAddress,
	infractionHeight int64,
	blockTime time.Time,
	power int64,
) {
	misbehavior := abci.Misbehavior{
		Type: abci.MisbehaviorType_DUPLICATE_VOTE,
		Validator: abci.Validator{
			Address: consAddr,
			Power:   power,
		},
		Height: infractionHeight,
		Time:   blockTime,
		// not used AFAICT
		TotalVotingPower: suite.TotalPower,
	}
	evidence := evidencetypes.FromABCIEvidence(misbehavior)
	equivocation := evidence.(*evidencetypes.Equivocation)
	suite.App.EvidenceKeeper.HandleEquivocationEvidence(suite.Ctx, equivocation)
}

func (suite *KeyChangeEscapeTestSuite) CommitWithInfo(
	validators []abci.Validator,
	nonSigners []int,
	t time.Duration,
) {
	header := suite.Ctx.BlockHeader()
	suite.App.EndBlocker(suite.Ctx, abci.RequestEndBlock{Height: header.Height})
	suite.App.Commit()
	header.Height++
	header.Time = header.Time.Add(t)
	header.AppHash = suite.App.LastCommitID().Hash
	suite.Ctx = suite.Ctx.WithBlockHeader(header)
	// in the begin blocker, we must set a validator's signing status
	votes := make([]abci.VoteInfo, 0, len(validators))
	for i, validator := range validators {
		votes = append(
			votes, abci.VoteInfo{
				Validator:       validator,
				SignedLastBlock: !slices.Contains(nonSigners, i),
			},
		)
	}
	req := abci.RequestBeginBlock{
		Header: header,
		LastCommitInfo: abci.CommitInfo{
			Round: 0,
			Votes: votes,
		},
	}
	suite.App.BeginBlock(req)
	newCtx := suite.App.BaseApp.NewContext(false, header)
	newCtx = newCtx.WithMinGasPrices(suite.Ctx.MinGasPrices())
	newCtx = newCtx.WithEventManager(suite.Ctx.EventManager())
	newCtx = newCtx.WithKVGasConfig(suite.Ctx.KVGasConfig())
	newCtx = newCtx.WithTransientKVGasConfig(suite.Ctx.TransientKVGasConfig())
	suite.Ctx = newCtx
}

func (suite *KeyChangeEscapeTestSuite) ChangeKey(
	addr sdk.AccAddress,
	expectSuccess bool,
) keytypes.WrappedConsKey {
	newKey := testutiltx.GenerateConsensusKey()
	response, err := suite.OperatorMsgServer.SetConsKey(
		sdk.WrapSDKContext(suite.Ctx),
		&operatortypes.SetConsKeyReq{
			Address:       addr.String(),
			AvsAddress:    suite.AvsAddress,
			PublicKeyJSON: newKey.ToJSON(),
		},
	)
	if expectSuccess {
		suite.Require().NoError(err)
		suite.Require().NotNil(response)
	} else {
		suite.Require().Error(err)
	}
	return newKey
}

func (suite *KeyChangeEscapeTestSuite) Unjail(
	addr sdk.AccAddress, expectSuccess bool,
) {
	msgServer := slashingkeeper.NewMsgServerImpl(suite.App.SlashingKeeper)
	resp, err := msgServer.Unjail(
		sdk.WrapSDKContext(suite.Ctx),
		&slashingtypes.MsgUnjail{
			ValidatorAddr: sdk.ValAddress(addr).String(),
		},
	)
	if expectSuccess {
		suite.Require().NoError(err)
		suite.Require().NotNil(resp)
	} else {
		suite.Require().Error(err)
		suite.Require().ErrorIs(err, slashingtypes.ErrValidatorJailed)
	}
}

// what we want to test is a validator who:
// 1. performs a double signing action
// 2. goes offline to be kicked out of the set
// 3. changes consensus key (to cause pruning of the old key)
// 4. is reported against, and correctly slashed.
// 5. cannot join the validator set (with the new or the old key).
func (suite *KeyChangeEscapeTestSuite) TestKeyChangeEscape() {
	// at genesis + 10 blocks, add a new validator
	operatorAddr, consKey, power := suite.AddValidator(3)
	// valAddr := sdk.ValAddress(operatorAddr)
	consAddr := consKey.ToConsAddr()
	// wait a few epochs for said validator to activate
	// during this wait there is no downtime slashing since
	// LastCommitInfo is empty.
	suite.RunToEpochEndNoEndBlockerN(
		suite.EpochIdentifier, 3,
	)
	// check that the validator is in the validator set
	suite.CheckValidator(operatorAddr, consAddr, power, power)
	// save this information for use later
	infractionHeight := suite.Ctx.BlockHeight()
	infractionTime := suite.Ctx.BlockTime()
	infractionPower := power
	infractionAddr := consAddr

	// original validator set
	validators := []abci.Validator{}
	for _, val := range suite.ValSet.Validators {
		validator, found := suite.App.StakingKeeper.GetImuachainValidator(
			suite.Ctx, val.Address.Bytes(),
		)
		suite.Require().True(found)
		validators = append(
			validators, abci.Validator{
				Address: validator.Address,
				Power:   validator.Power,
			},
		)
	}
	// recent validator
	validators = append(
		validators, abci.Validator{
			Address: consAddr.Bytes(),
			Power:   power,
		},
	)

	// 2. take the validator offline
	// determine the amount of blocks to commit
	signInfo, found := suite.App.SlashingKeeper.GetValidatorSigningInfo(suite.Ctx, consAddr)
	suite.Require().True(found)
	signedBlocksWindow := suite.App.SlashingKeeper.SignedBlocksWindow(suite.Ctx)
	minHeight := signInfo.StartHeight + signedBlocksWindow
	minSignedPerWindow := suite.App.SlashingKeeper.MinSignedPerWindow(suite.Ctx)
	maxMissed := signedBlocksWindow - minSignedPerWindow
	currentlyMissed := int64(0)
	blocksToMiss := maxMissed - currentlyMissed + 1
	blocksToReachMin := minHeight
	blocksToCommit := blocksToMiss
	if blocksToCommit < blocksToReachMin {
		blocksToCommit = blocksToReachMin
	}
	for i := 0; i < int(blocksToCommit); i++ {
		suite.CommitWithInfo(validators, []int{2}, time.Nanosecond)
	}
	signInfo, found = suite.App.SlashingKeeper.GetValidatorSigningInfo(suite.Ctx, consAddr)
	suite.Require().True(found)
	suite.Require().True(signInfo.JailedUntil.After(suite.Ctx.BlockTime()))
	// check if the validator is available?
	_, found = suite.App.StakingKeeper.GetImuachainValidator(suite.Ctx, consAddr)
	suite.Require().False(found)
	// check slashing
	downtimeSlashFactor := suite.App.SlashingKeeper.SlashFractionDowntime(suite.Ctx)
	power = suite.CheckSlashEffect(operatorAddr, downtimeSlashFactor, power)
	// now that the validator is jailed, change its key
	newConsKey := suite.ChangeKey(operatorAddr, true)
	suite.Commit()
	// now that the key is removed, we advance some epochs
	suite.RunToEpochEndNoEndBlocker(suite.EpochIdentifier)
	// submit the evidence for slashing
	suite.SubmitEvidence(
		infractionAddr, infractionHeight, infractionTime, infractionPower,
	)
	// include it in the latest block
	suite.Commit()
	doubleSignSlashFactor := suite.App.SlashingKeeper.SlashFractionDoubleSign(suite.Ctx)
	power = suite.CheckSlashEffect(operatorAddr, doubleSignSlashFactor, power)
	// check
	suite.CheckTombstoned(consAddr, true)
	suite.CheckTombstoned(newConsKey.ToConsAddr(), true)
	// go forward in time
	suite.RunToEpochEndNoEndBlockerN(suite.EpochIdentifier, 1)
	// now, we need to check if validator can rejoin the set
	// to do so, the first thing required is unjailing
	// unjailing is routed by the slashing keeper to the staking keeper
	// the slashing keeper detects the validator as tombstoned and exits
	// the tombstone status is stored in the validator signing info
	// the functions that edit it are
	// Tombstone -> which will panic if you try to tombstone again
	// JailUntil -> called by x/evidence HandleEquivocationEvidence, which short circuits if
	//   you are already tombstoned
	// HandleValidatorSignature -> which handles data from abci.RequestBeginBlock.LastCommitInfo
	//   it is set by TM only if the validator is in the set, in this case it is not.
	//   even if it were in the set for one block or two, the tombstone status is left untouched
	//   by that function
	// however, the slashing module indexes the signing info by consensus address and not
	// account address. if a validator is tombstoned and then changes its key, it can be
	// unjailed successfully. we have to avoid that somehow!
	suite.Unjail(operatorAddr, false)
	// check if we can submit the evidence again - there should be no panic
	suite.Commit()
	// first with same key
	suite.SubmitEvidence(
		consAddr, suite.Ctx.BlockHeight(),
		suite.Ctx.BlockTime(), power,
	)
	suite.Commit()
	// then with new key
	suite.SubmitEvidence(
		newConsKey.ToConsAddr(), suite.Ctx.BlockHeight(),
		suite.Ctx.BlockTime(), power,
	)
	// check power, no slashing should be applied.
	power = suite.CheckSlashEffect(operatorAddr, sdkmath.LegacyZeroDec(), power)
}

// Active Set Key Rotation ("Early Escape")
func (suite *KeyChangeEscapeTestSuite) TestEarlyEscape() {
	// add a new validator
	operatorAddr, consKeyA, power := suite.AddValidator(3)
	consAddrA := consKeyA.ToConsAddr()

	// wait a few epochs for activation
	suite.RunToEpochEndNoEndBlockerN(suite.EpochIdentifier, 3)
	suite.Commit()
	suite.CheckValidator(operatorAddr, consAddrA, power, power)

	// save double-sign info for Key A
	infractionHeight := suite.Ctx.BlockHeight()
	infractionTime := suite.Ctx.BlockTime()
	infractionPower := power

	// Step 1: Operator rotates to Key B while still strictly in the active set
	// (no downtime jailing has occurred)
	consKeyB := suite.ChangeKey(operatorAddr, true)
	suite.Commit()

	// Step 2: Epoch advances to process the key change
	suite.RunToEpochEndNoEndBlocker(suite.EpochIdentifier)
	suite.Commit()

	consAddrB := consKeyB.ToConsAddr()
	// Validator should now be active under Key B
	suite.CheckValidator(operatorAddr, consAddrB, power, power)

	// Step 3: Evidence is submitted for Key A
	suite.SubmitEvidence(consAddrA, infractionHeight, infractionTime, infractionPower)
	suite.Commit()

	// Step 4: Verify Slash & Tombstone
	// The operator should be tombstoned on *both* the old and current key
	suite.CheckTombstoned(consAddrA, true)
	suite.CheckTombstoned(consAddrB, true)

	// operator should have been slashed for the double sign
	doubleSignSlashFactor := suite.App.SlashingKeeper.SlashFractionDoubleSign(suite.Ctx)
	suite.CheckSlashEffect(operatorAddr, doubleSignSlashFactor, power)

	// The validator should be jailed and removed from the active set
	_, found := suite.App.StakingKeeper.GetImuachainValidator(suite.Ctx, consAddrB)
	suite.Require().False(found)
}

// Voluntary Opt-Out / Deregistration
func (suite *KeyChangeEscapeTestSuite) TestKeyChangeVoluntaryOptOut() {
	operatorAddr, consKeyA, power := suite.AddValidator(3)
	consAddrA := consKeyA.ToConsAddr()

	suite.RunToEpochEndNoEndBlockerN(suite.EpochIdentifier, 3)
	suite.Commit()
	suite.CheckValidator(operatorAddr, consAddrA, power, power)

	infractionHeight := suite.Ctx.BlockHeight()
	infractionTime := suite.Ctx.BlockTime()
	infractionPower := power

	// Step 1: Operator Double signs.
	// Step 2: Operator voluntarily Opts Out before evidence
	suite.OptOut(operatorAddr)
	suite.Commit()

	// Step 3: Run to the exact end of the epoch to process the OptOut
	suite.RunToEpochEndNoEndBlocker(suite.EpochIdentifier)
	suite.Commit()

	// Step 3: Evidence is submitted while they are unbonding / opted out
	suite.SubmitEvidence(consAddrA, infractionHeight, infractionTime, infractionPower)
	suite.Commit()

	// Verify the operator is tombstoned
	suite.CheckTombstoned(consAddrA, true)

	// Operator is slashed despite opting out
	doubleSignSlashFactor := suite.App.SlashingKeeper.SlashFractionDoubleSign(suite.Ctx)
	suite.CheckSlashEffect(operatorAddr, doubleSignSlashFactor, power)
}

// Forced Ejection via Delegator Unbonding
func (suite *KeyChangeEscapeTestSuite) TestKeyChangeForcedEjection() {
	operatorAddr, consKeyA, power := suite.AddValidator(3)
	consAddrA := consKeyA.ToConsAddr()

	suite.RunToEpochEndNoEndBlockerN(suite.EpochIdentifier, 3)
	suite.Commit()
	suite.CheckValidator(operatorAddr, consAddrA, power, power)

	infractionHeight := suite.Ctx.BlockHeight()
	infractionTime := suite.Ctx.BlockTime()
	infractionPower := power

	// Step 1: Delegator undelegates massively, dropping operator below MinSelfDelegation
	// Withdraw 2x the minimum, leaving 1x (which is exactly the minimum, so we'll withdraw a bit more)
	amountToWithdraw := suite.MinSelfDelegation.Int64()*2 + 1
	suite.UndelegateFromOperator(common.Address(operatorAddr.Bytes()), operatorAddr, amountToWithdraw)
	suite.Commit()

	// Process the unbonding and wait an epoch for the validator to be ejected from active set
	suite.RunToEpochEndNoEndBlocker(suite.EpochIdentifier)
	suite.Commit()

	// The validator should be inactive
	_, found := suite.App.StakingKeeper.GetImuachainValidator(suite.Ctx, consAddrA)
	suite.Require().False(found)

	// Step 2: Operator frantically rotates key to bypass the impending slash
	consKeyB := suite.ChangeKey(operatorAddr, true)
	suite.Commit()
	suite.RunToEpochEndNoEndBlocker(suite.EpochIdentifier)
	suite.Commit()

	// Step 3: Evidence submitted for the old key
	suite.SubmitEvidence(consAddrA, infractionHeight, infractionTime, infractionPower)
	suite.Commit()

	// Verify all affected keys are tombstoned
	suite.CheckTombstoned(consAddrA, true)
	suite.CheckTombstoned(consKeyB.ToConsAddr(), true)

	// Verify slash fraction was caught directly into the unbonding queues
	// We can't check consensus power cleanly if they are completely evicted, so we check USD value
	expectedRemainingUsdHuman := (suite.MinSelfDelegation.Int64() * 3) - amountToWithdraw

	doubleSignSlashFactor := suite.App.SlashingKeeper.SlashFractionDoubleSign(suite.Ctx)

	slashValue := sdkmath.LegacyNewDec(expectedRemainingUsdHuman).Mul(doubleSignSlashFactor)
	effectiveSlashProportion := sdkmath.LegacyMinDec(
		sdkmath.LegacyNewDec(1), slashValue.QuoInt64(expectedRemainingUsdHuman),
	)
	subtract := effectiveSlashProportion.MulInt64(expectedRemainingUsdHuman)
	endingUsd := sdkmath.LegacyNewDec(expectedRemainingUsdHuman).Sub(subtract)

	suite.CheckOperatorUSDValueExact(operatorAddr, endingUsd)
}

// Opt-In with a Pending Slash
func (suite *KeyChangeEscapeTestSuite) TestOptInWithPendingSlash() {
	operatorAddr, consKeyA, power := suite.AddValidator(3)
	consAddrA := consKeyA.ToConsAddr()

	suite.RunToEpochEndNoEndBlockerN(suite.EpochIdentifier, 3)
	suite.CheckValidator(operatorAddr, consAddrA, power, power)

	infractionHeight := suite.Ctx.BlockHeight()
	infractionTime := suite.Ctx.BlockTime()
	infractionPower := power

	// Step 1: Double sign, then OptOut voluntarily. This begins
	// the unbonding period (e.g., 8 epochs).
	suite.OptOut(operatorAddr)
	suite.Commit()

	// Step 2: Advance a few epochs so we are inside the unbonding period
	forwarded := 2
	suite.RunToEpochEndNoEndBlockerN(suite.EpochIdentifier, forwarded)
	suite.Commit()

	// Step 3: Evidence is submitted. The operator is still in the unbonding period,
	// so the slash and tombstone should apply to them.
	suite.SubmitEvidence(consAddrA, infractionHeight, infractionTime, infractionPower)
	suite.Commit()

	// Verify the operator is tombstoned
	suite.CheckTombstoned(consAddrA, true)

	// Step 4: Advance the remaining epochs to process the Opt-Out fully
	unbondingPeriod := suite.App.StakingKeeper.GetDogfoodParams(suite.Ctx).EpochsUntilUnbonded
	suite.RunToEpochEndNoEndBlockerN(suite.EpochIdentifier, int(unbondingPeriod)-forwarded+1)
	suite.Commit()

	// Step 5: After unbonding is complete, the operator (now completely unbonded)
	// attempts to opt-in with a brand new key B to sneak back into the active set.
	consKeyB := testutiltx.GenerateConsensusKey()
	res, err := suite.OperatorMsgServer.OptIntoAVS(
		sdk.WrapSDKContext(suite.Ctx),
		&operatortypes.OptIntoAVSReq{
			FromAddress:   operatorAddr.String(),
			AvsAddress:    suite.AvsAddress,
			PublicKeyJSON: consKeyB.ToJSON(),
		},
	)

	// It should fail because they are permanently jailed (tombstoned)
	suite.Require().Nil(res)
	suite.Require().Error(err)
	suite.Require().ErrorIs(err, operatortypes.ErrOperatorIsFrozen)

	_, found := suite.App.StakingKeeper.GetImuachainValidator(suite.Ctx, consKeyB.ToConsAddr())
	suite.Require().False(found, "Operator should not be allowed back in active set")
}

// Assimilating a Tombstoned Key
func (suite *KeyChangeEscapeTestSuite) TestAssimilatingTombstonedKey() {
	// Operator 1 gets tombstoned on Key A
	operatorAddr1, consKeyA, power := suite.AddValidator(3)
	consAddrA := consKeyA.ToConsAddr()

	suite.RunToEpochEndNoEndBlockerN(suite.EpochIdentifier, 3)
	suite.CheckValidator(operatorAddr1, consAddrA, power, power)

	infractionHeight := suite.Ctx.BlockHeight()
	infractionTime := suite.Ctx.BlockTime()
	infractionPower := power

	suite.SubmitEvidence(consAddrA, infractionHeight, infractionTime, infractionPower)
	suite.Commit()

	suite.CheckTombstoned(consAddrA, true)

	// Operator 2 arrives and tries to use Key A
	operatorAddr2, _ := testutiltx.NewAccAddressAndKey()
	delegator2 := common.Address(operatorAddr2.Bytes())
	suite.RegisterOperator(operatorAddr2)
	amount := suite.MinSelfDelegation.Int64() * 3
	suite.DepositFromStaker(
		delegator2, amount,
	)
	suite.DelegateToOperator(
		delegator2, operatorAddr2, amount,
	)
	suite.AssociateOperatorWithStaker(
		delegator2, operatorAddr2,
	)

	// Attempt to set cons key should fail
	response, err := suite.OperatorMsgServer.OptIntoAVS(
		sdk.WrapSDKContext(suite.Ctx),
		&operatortypes.OptIntoAVSReq{
			FromAddress:   operatorAddr2.String(),
			AvsAddress:    suite.AvsAddress,
			PublicKeyJSON: consKeyA.ToJSON(),
		},
	)
	// it should be fully rejected to take over a jailed key
	// we already test unjailed key elsewhere.
	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().ErrorIs(err, operatortypes.ErrConsKeyAlreadyInUse)
}

// Rotating back to an old, innocent key
func (suite *KeyChangeEscapeTestSuite) TestRotateToOldInnocentKey() {
	operatorAddr, consKeyA, power := suite.AddValidator(3)

	suite.RunToEpochEndNoEndBlockerN(suite.EpochIdentifier, 3)
	suite.CheckValidator(operatorAddr, consKeyA.ToConsAddr(), power, power)

	// Rotate to Key B
	consKeyB := suite.ChangeKey(operatorAddr, true)
	suite.Commit()
	suite.RunToEpochEndNoEndBlocker(suite.EpochIdentifier)
	suite.Commit()
	suite.CheckValidator(operatorAddr, consKeyB.ToConsAddr(), power, power)

	// Double sign on Key B
	infractionHeight := suite.Ctx.BlockHeight()
	infractionTime := suite.Ctx.BlockTime()
	infractionPower := power

	suite.SubmitEvidence(consKeyB.ToConsAddr(), infractionHeight, infractionTime, infractionPower)
	suite.Commit()

	suite.CheckTombstoned(consKeyB.ToConsAddr(), true)

	// key changing is rejected because we are frozen!
	response, err := suite.OperatorMsgServer.SetConsKey(
		sdk.WrapSDKContext(suite.Ctx),
		&operatortypes.SetConsKeyReq{
			Address:       operatorAddr.String(),
			AvsAddress:    suite.AvsAddress,
			PublicKeyJSON: consKeyA.ToJSON(),
		},
	)
	suite.Require().Nil(response)
	suite.Require().Error(err)
	suite.Require().ErrorIs(err, operatortypes.ErrOperatorIsFrozen)
	// Key A was naturally innocent. It shouldn't be tombstoned on its own;
	// rather, the operator is securely jailed, preventing reuse.
	suite.CheckTombstoned(consKeyA.ToConsAddr(), false)
}

// Simultaneous Downtime and Double Sign on Different Keys
func (suite *KeyChangeEscapeTestSuite) TestSimultaneousInfractions() {
	operatorAddr, consKeyA, power := suite.AddValidator(3)
	consAddrA := consKeyA.ToConsAddr()

	suite.RunToEpochEndNoEndBlockerN(suite.EpochIdentifier, 3)
	suite.CheckValidator(operatorAddr, consAddrA, power, power)

	// Setup double sign on Key A, but don't submit yet
	infractionHeight := suite.Ctx.BlockHeight()
	infractionTime := suite.Ctx.BlockTime()
	infractionPower := power

	// Rotate to Key B
	consKeyB := suite.ChangeKey(operatorAddr, true)
	suite.Commit()
	suite.RunToEpochEndNoEndBlocker(suite.EpochIdentifier)
	suite.CheckValidator(operatorAddr, consKeyB.ToConsAddr(), power, power)

	// Setup validators for downtime checking
	validators := []abci.Validator{}
	for _, val := range suite.ValSet.Validators {
		validator, found := suite.App.StakingKeeper.GetImuachainValidator(
			suite.Ctx, val.Address.Bytes(),
		)
		suite.Require().True(found)
		validators = append(
			validators, abci.Validator{
				Address: validator.Address,
				Power:   validator.Power,
			},
		)
	}
	nonSigners := []int{len(validators)}
	validators = append(
		validators, abci.Validator{
			Address: consKeyB.ToConsAddr().Bytes(),
			Power:   power,
		},
	)

	// Go offline to accrue downtime on Key B
	signedBlocksWindow := suite.App.SlashingKeeper.SignedBlocksWindow(suite.Ctx)
	minSignedPerWindow := suite.App.SlashingKeeper.MinSignedPerWindow(suite.Ctx)
	maxMissed := signedBlocksWindow - minSignedPerWindow

	// Submit double sign ONE block before downtime jail
	for i := int64(0); i < maxMissed-1; i++ {
		suite.CommitWithInfo(validators, nonSigners, time.Nanosecond)
	}

	// Submit Double Sign on Key A
	suite.SubmitEvidence(consAddrA, infractionHeight, infractionTime, infractionPower)

	// Final block that triggers downtime on Key B
	suite.CommitWithInfo(validators, []int{2}, time.Nanosecond)
	suite.Commit()

	// Should not panic, should tombstone both
	suite.CheckTombstoned(consAddrA, true)
	suite.CheckTombstoned(consKeyB.ToConsAddr(), true)
}

// Stale Evidence Escape
func (suite *KeyChangeEscapeTestSuite) TestStaleEvidence() {
	operatorAddr, consKeyA, power := suite.AddValidator(3)
	consAddrA := consKeyA.ToConsAddr()

	suite.RunToEpochEndNoEndBlockerN(suite.EpochIdentifier, 3)
	suite.Commit()
	suite.CheckValidator(operatorAddr, consAddrA, power, power)

	infractionHeight := suite.Ctx.BlockHeight()
	infractionTime := suite.Ctx.BlockTime()
	infractionPower := power

	// Set consensus params for evidence maximums to be small, to avoid looping
	consensusParams := suite.App.GetConsensusParams(suite.Ctx)
	consensusParams.Evidence.MaxAgeNumBlocks = 10
	consensusParams.Evidence.MaxAgeDuration = 10 * time.Second
	suite.Ctx = suite.Ctx.WithConsensusParams(consensusParams)
	suite.App.ConsensusParamsKeeper.Set(suite.Ctx, consensusParams)

	maxAgeNumBlocks := consensusParams.Evidence.MaxAgeNumBlocks
	maxAgeDuration := consensusParams.Evidence.MaxAgeDuration

	// Age the evidence past the allowed boundary
	for i := int64(0); i < maxAgeNumBlocks+1; i++ {
		suite.Commit()
	}

	// Ensure time also passes the max age duration
	header := suite.Ctx.BlockHeader()
	header.Time = header.Time.Add(maxAgeDuration + time.Second)
	suite.Ctx = suite.Ctx.WithBlockHeader(header)

	// Rotate key at the boundary
	consKeyB := suite.ChangeKey(operatorAddr, true)
	suite.Commit()
	suite.RunToEpochEndNoEndBlocker(suite.EpochIdentifier)
	suite.Commit()

	// Submit evidence. Since EvidenceKeeper ignores it without returning an error
	// (as seen in the provided handleEquivocationEvidence behavior), we simply
	// call it and verify tombstoning didn't happen.
	suite.App.EvidenceKeeper.HandleEquivocationEvidence(suite.Ctx, &evidencetypes.Equivocation{
		Height:           infractionHeight,
		Time:             infractionTime,
		Power:            infractionPower,
		ConsensusAddress: consAddrA.String(),
	})

	// Since evidence was rejected as stale, they should not be tombstoned
	suite.CheckTombstoned(consAddrA, false)
	suite.CheckTombstoned(consKeyB.ToConsAddr(), false)
}

// Exact Zero Stake Remaining
func (suite *KeyChangeEscapeTestSuite) TestExactZeroStake() {
	operatorAddr, consKeyA, power := suite.AddValidator(3)

	suite.RunToEpochEndNoEndBlockerN(suite.EpochIdentifier, 3)
	suite.CheckValidator(operatorAddr, consKeyA.ToConsAddr(), power, power)

	infractionHeight := suite.Ctx.BlockHeight()
	infractionTime := suite.Ctx.BlockTime()
	infractionPower := power

	// Withdraw exactly all self-stake to hit 0
	amountToWithdraw := suite.MinSelfDelegation.Int64() * 3
	suite.UndelegateFromOperator(common.Address(operatorAddr.Bytes()), operatorAddr, amountToWithdraw)
	suite.Commit()
	suite.RunToEpochEndNoEndBlocker(suite.EpochIdentifier)

	consKeyB := suite.ChangeKey(operatorAddr, true)
	suite.Commit()
	suite.RunToEpochEndNoEndBlocker(suite.EpochIdentifier)

	suite.SubmitEvidence(consKeyA.ToConsAddr(), infractionHeight, infractionTime, infractionPower)
	suite.Commit()

	suite.CheckTombstoned(consKeyA.ToConsAddr(), true)
	suite.CheckTombstoned(consKeyB.ToConsAddr(), true)

	// Verify exact zero math works without panic
	zeroUsd := sdkmath.LegacyZeroDec()
	suite.CheckOperatorUSDValueExact(operatorAddr, zeroUsd)
}

// Unit Test C: Landmine Key Debt Reset
func (suite *KeyChangeEscapeTestSuite) TestLandmineKeyDebtReset() {
	operatorAddr1, consKeyA, power := suite.AddValidator(3)
	consAddrA := consKeyA.ToConsAddr()

	suite.RunToEpochEndNoEndBlockerN(suite.EpochIdentifier, 3)
	suite.CheckValidator(operatorAddr1, consAddrA, power, power)

	validators := make([]abci.Validator, 0)
	for _, val := range suite.ValSet.Validators {
		validator, found := suite.App.StakingKeeper.GetImuachainValidator(
			suite.Ctx, val.Address.Bytes(),
		)
		suite.Require().True(found)
		validators = append(
			validators, abci.Validator{
				Address: validator.Address,
				Power:   validator.Power,
			},
		)
	}
	nonSigners := []int{len(validators)}
	validators = append(
		validators, abci.Validator{
			Address: consAddrA.Bytes(),
			Power:   power,
		},
	)

	minSignedPerWindow := suite.App.SlashingKeeper.MinSignedPerWindow(suite.Ctx)
	window := suite.App.SlashingKeeper.SignedBlocksWindow(suite.Ctx)
	maxMissed := window - minSignedPerWindow
	loop := int(maxMissed) - 1
	for i := 0; i < loop; i++ {
		suite.CommitWithInfo(validators, nonSigners, time.Nanosecond)
	}
	suite.Commit()

	signInfo1, found := suite.App.SlashingKeeper.GetValidatorSigningInfo(suite.Ctx, consAddrA)
	suite.Require().True(found)
	suite.Require().Equal(int64(loop), signInfo1.MissedBlocksCounter)

	// Step 2: Operator 1 rotates to Key B. Key A begins unbonding.
	suite.ChangeKey(operatorAddr1, true)
	suite.Commit()
	suite.RunToEpochEndNoEndBlocker(suite.EpochIdentifier)

	// Step 3: Wait for 8 epochs for Key A's unbonding to finish, pruning the reverse lookup
	// This invokes abci.go which calls ResetValidatorSigningInfo!
	unbondingPeriod := suite.App.StakingKeeper.GetDogfoodParams(suite.Ctx).EpochsUntilUnbonded
	suite.RunToEpochEndNoEndBlockerN(suite.EpochIdentifier, int(unbondingPeriod)+1)
	suite.Commit()

	// Step 4: Create a new validator 2
	operatorAddr2, _ := testutiltx.NewAccAddressAndKey()
	delegator2 := common.Address(operatorAddr2.Bytes())
	suite.RegisterOperator(operatorAddr2)
	amount := suite.MinSelfDelegation.Int64() * 3
	suite.DepositFromStaker(delegator2, amount)
	suite.DelegateToOperator(delegator2, operatorAddr2, amount)
	suite.AssociateOperatorWithStaker(delegator2, operatorAddr2)

	// Step 5: Validator 2 registers Key A
	response, err := suite.OperatorMsgServer.OptIntoAVS(
		sdk.WrapSDKContext(suite.Ctx),
		&operatortypes.OptIntoAVSReq{
			FromAddress:   operatorAddr2.String(),
			AvsAddress:    suite.AvsAddress,
			PublicKeyJSON: consKeyA.ToJSON(),
		},
	)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)

	suite.RunToEpochEndNoEndBlockerN(suite.EpochIdentifier, 3)

	// Step 6: Verify Key A's signing info was reset!
	signInfo2, found := suite.App.SlashingKeeper.GetValidatorSigningInfo(suite.Ctx, consAddrA)
	suite.Require().True(found)

	// The debt should be completely wiped clean!
	suite.Require().Equal(int64(0), signInfo2.MissedBlocksCounter)
}

// Unit Test A: Tombstoned key rejection for new validator
func (suite *KeyChangeEscapeTestSuite) TestTombstonedKeyRejectionOnNewValidator() {
	operatorAddr1, consKeyA, power := suite.AddValidator(3)
	consAddrA := consKeyA.ToConsAddr()

	suite.RunToEpochEndNoEndBlockerN(suite.EpochIdentifier, 3)
	suite.CheckValidator(operatorAddr1, consAddrA, power, power)

	infractionHeight := suite.Ctx.BlockHeight()
	infractionTime := suite.Ctx.BlockTime()
	infractionPower := power

	// Step 1: Operator 1 rotates to Key B (so Key A begins an unbonding process and will be pruned eventually)
	suite.ChangeKey(operatorAddr1, true)
	suite.Commit()
	suite.RunToEpochEndNoEndBlocker(suite.EpochIdentifier)

	// Step 2: Evidence is submitted for Key A. Operator 1 gets slashed and Key A gets tombstoned.
	suite.SubmitEvidence(consAddrA, infractionHeight, infractionTime, infractionPower)
	suite.Commit()

	suite.CheckTombstoned(consAddrA, true)

	// Step 3: Wait for 8 epochs for Key A's unbonding to finish, pruning the reverse lookup
	unbondingPeriod := suite.App.StakingKeeper.GetDogfoodParams(suite.Ctx).EpochsUntilUnbonded
	suite.RunToEpochEndNoEndBlockerN(suite.EpochIdentifier, int(unbondingPeriod)+1)
	suite.Commit()

	// Step 4: Create a new validator with enough self delegation
	operatorAddr2, _ := testutiltx.NewAccAddressAndKey()
	delegator2 := common.Address(operatorAddr2.Bytes())
	suite.RegisterOperator(operatorAddr2)
	amount := suite.MinSelfDelegation.Int64() * 3
	suite.DepositFromStaker(delegator2, amount)
	suite.DelegateToOperator(delegator2, operatorAddr2, amount)
	suite.AssociateOperatorWithStaker(delegator2, operatorAddr2)

	// Step 5: Try to add same consensus key (Key A) to the new validator
	response, err := suite.OperatorMsgServer.OptIntoAVS(
		sdk.WrapSDKContext(suite.Ctx),
		&operatortypes.OptIntoAVSReq{
			FromAddress:   operatorAddr2.String(),
			AvsAddress:    suite.AvsAddress,
			PublicKeyJSON: consKeyA.ToJSON(),
		},
	)

	// Should fail with our error, instead of ErrConsKeyAlreadyInUse
	suite.Require().Nil(response)
	suite.Require().Error(err)
	suite.Require().ErrorIs(err, dogfoodtypes.ErrConsKeyAlreadyTombstoned)
}

// Unit Test B: Tombstoned key rejection on key replacement
func (suite *KeyChangeEscapeTestSuite) TestTombstonedKeyRejectionOnKeyReplacement() {
	// Operator 1
	operatorAddr1, consKeyA, power := suite.AddValidator(3)
	consAddrA := consKeyA.ToConsAddr()

	// Operator 2
	operatorAddr2, _, power2 := suite.AddValidator(3)

	suite.RunToEpochEndNoEndBlockerN(suite.EpochIdentifier, 3)
	suite.CheckValidator(operatorAddr1, consAddrA, power, power)
	// We don't strictly need to check validator 2, but we can verify it's active
	suite.CheckOperatorUSDValueExact(operatorAddr2, sdkmath.LegacyNewDec(power2))

	infractionHeight := suite.Ctx.BlockHeight()
	infractionTime := suite.Ctx.BlockTime()
	infractionPower := power

	// Step 1: Operator 1 rotates to Key B (so Key A begins an unbonding process and will be pruned)
	suite.ChangeKey(operatorAddr1, true)
	suite.Commit()
	suite.RunToEpochEndNoEndBlocker(suite.EpochIdentifier)

	// Step 2: Evidence is submitted for Key A. Operator 1 gets slashed and Key A gets tombstoned.
	suite.SubmitEvidence(consAddrA, infractionHeight, infractionTime, infractionPower)
	suite.Commit()

	suite.CheckTombstoned(consAddrA, true)

	// Step 3: Wait for 8 epochs for Key A's unbonding to finish, pruning the reverse lookup
	unbondingPeriod := suite.App.StakingKeeper.GetDogfoodParams(suite.Ctx).EpochsUntilUnbonded
	suite.RunToEpochEndNoEndBlockerN(suite.EpochIdentifier, int(unbondingPeriod)+1)
	suite.Commit()

	// Step 4: Existing Operator 2 tries to replace their key with Key A
	response, err := suite.OperatorMsgServer.SetConsKey(
		sdk.WrapSDKContext(suite.Ctx),
		&operatortypes.SetConsKeyReq{
			Address:       operatorAddr2.String(),
			AvsAddress:    suite.AvsAddress,
			PublicKeyJSON: consKeyA.ToJSON(),
		},
	)

	// Should fail with our error
	suite.Require().Nil(response)
	suite.Require().Error(err)
	suite.Require().ErrorIs(err, dogfoodtypes.ErrConsKeyAlreadyTombstoned)
}

// Unit Test: Simulate UNSPECIFIED copy + DOWNTIME copy in the same epoch causing MissedBlocksCounter drift
func (suite *KeyChangeEscapeTestSuite) TestDowntimeDriftKeyReplacement() {
	operatorAddr, consKeyA, power := suite.AddValidator(3)
	consAddrA := consKeyA.ToConsAddr()

	suite.RunToEpochEndNoEndBlockerN(suite.EpochIdentifier, 3)
	suite.CheckValidator(operatorAddr, consAddrA, power, power)

	validators := make([]abci.Validator, 0)
	for _, val := range suite.ValSet.Validators {
		validator, found := suite.App.StakingKeeper.GetImuachainValidator(
			suite.Ctx, val.Address.Bytes(),
		)
		suite.Require().True(found)
		validators = append(
			validators, abci.Validator{
				Address: validator.Address,
				Power:   validator.Power,
			},
		)
	}
	nonSigners := []int{len(validators)} // the last validator is the one we added
	validators = append(
		validators, abci.Validator{
			Address: consAddrA.Bytes(),
			Power:   power,
		},
	)

	// Step 1: go offline for some blocks, but not enough to trigger downtime
	signInfo, found := suite.App.SlashingKeeper.GetValidatorSigningInfo(suite.Ctx, consAddrA)
	suite.Require().True(found)
	signedBlocksWindow := suite.App.SlashingKeeper.SignedBlocksWindow(suite.Ctx)
	minHeight := signInfo.StartHeight + signedBlocksWindow
	minSignedPerWindow := suite.App.SlashingKeeper.MinSignedPerWindow(suite.Ctx)
	maxMissed := signedBlocksWindow - minSignedPerWindow

	blocksToReachMin := minHeight
	blocksToMiss := maxMissed + 1
	blocksToCommit := blocksToMiss
	if blocksToCommit < blocksToReachMin {
		blocksToCommit = blocksToReachMin
	}

	// Miss half the blocks needed
	missesBeforeRotation := blocksToCommit / 2
	for i := int64(0); i < missesBeforeRotation; i++ {
		suite.CommitWithInfo(validators, nonSigners, time.Nanosecond)
	}

	// Step 2: Rotate key to Key B
	consKeyB := suite.ChangeKey(operatorAddr, true)

	// Step 3: Continue going offline on Key A until downtime slash is triggered
	blocksToTriggerSlash := blocksToCommit - missesBeforeRotation
	for i := int64(0); i < blocksToTriggerSlash; i++ {
		suite.CommitWithInfo(validators, nonSigners, time.Nanosecond)
	}

	// Now Key A should have been slashed for downtime. Operator Addr gets jailed.
	// Inside SlashWithInfractionReason, since slashingOldKey = true,
	// CopyValidatorSigningInfo (DOWNTIME) is called from Key A to Key B.
	// This resets Key B's MissedBlocksCounter and IndexOffset to 0.

	// Step 4: Run to epoch end so Key B becomes the active key
	suite.RunToEpochEndNoEndBlocker(suite.EpochIdentifier)

	// Operator is currently jailed. Unjail them.
	downtimeJailDur := suite.App.SlashingKeeper.DowntimeJailDuration(suite.Ctx)
	header := suite.Ctx.BlockHeader()
	header.Time = header.Time.Add(downtimeJailDur).Add(time.Second)
	suite.Ctx = suite.Ctx.WithBlockHeader(header)

	suite.Unjail(operatorAddr, true)

	// Run to epoch end so Key B re-enters the active set
	suite.RunToEpochEndNoEndBlocker(suite.EpochIdentifier)

	// Now Key B is an active signer.
	validatorsForB := make([]abci.Validator, 0)
	for _, val := range suite.ValSet.Validators {
		validator, found := suite.App.StakingKeeper.GetImuachainValidator(
			suite.Ctx, val.Address.Bytes(),
		)
		suite.Require().True(found)
		validatorsForB = append(
			validatorsForB, abci.Validator{
				Address: validator.Address,
				Power:   validator.Power,
			},
		)
	}

	// We need to find Key B's new power, as they might have been slashed softly during downtime
	valB, found := suite.App.StakingKeeper.GetImuachainValidator(suite.Ctx, consKeyB.ToConsAddr())
	suite.Require().True(found)

	validatorsForB = append(
		validatorsForB, abci.Validator{
			Address: consKeyB.ToConsAddr().Bytes(),
			Power:   valB.Power,
		},
	)

	// Key B signs ONE block successfully (not in nonSigners).
	// This invokes handleValidatorSignature at IndexOffset = 0.
	// With the fix, the bit array is completely empty, so MissedBlocksCounter remains 0.
	// Without the fix, bit 0 is true, so MissedBlocksCounter decrements from 0 to -1.
	suite.CommitWithInfo(validatorsForB, []int{}, time.Nanosecond)

	signInfo, found = suite.App.SlashingKeeper.GetValidatorSigningInfo(suite.Ctx, consKeyB.ToConsAddr())
	suite.Require().True(found)

	// Assert that it did NOT drift negative.
	suite.Require().GreaterOrEqual(signInfo.MissedBlocksCounter, int64(0), "Missed blocks counter should not drift negatively")
	suite.Require().Equal(int64(0), signInfo.MissedBlocksCounter)
}

// sumMissedBitsInWindow counts true entries in the x/slashing missed-block bit array for consAddr.
func (suite *KeyChangeEscapeTestSuite) sumMissedBitsInWindow(consAddr sdk.ConsAddress) int64 {
	window := suite.App.SlashingKeeper.SignedBlocksWindow(suite.Ctx)
	var sum int64
	for i := int64(0); i < window; i++ {
		if suite.App.SlashingKeeper.GetValidatorMissedBlockBitArray(suite.Ctx, consAddr, i) {
			sum++
		}
	}
	return sum
}

// TestChecklist_A3_B4_KeyRotationSigningInfoInvariant covers review checklist A3, D2, and rotation-time B4/A6:
// - After Msg SetConsKey, the old consensus address still has ValidatorSigningInfo (not yet pruned).
// - On the new address, MissedBlocksCounter matches the sum of the bit array after UNSPECIFIED CopyValidatorSigningInfo.
func (suite *KeyChangeEscapeTestSuite) TestChecklist_A3_B4_KeyRotationSigningInfoInvariant() {
	operatorAddr, consKeyA, power := suite.AddValidator(3)
	consAddrA := consKeyA.ToConsAddr()

	suite.RunToEpochEndNoEndBlockerN(suite.EpochIdentifier, 3)
	suite.CheckValidator(operatorAddr, consAddrA, power, power)

	validators := make([]abci.Validator, 0)
	for _, val := range suite.ValSet.Validators {
		validator, found := suite.App.StakingKeeper.GetImuachainValidator(
			suite.Ctx, val.Address.Bytes(),
		)
		suite.Require().True(found)
		validators = append(
			validators, abci.Validator{
				Address: validator.Address,
				Power:   validator.Power,
			},
		)
	}
	nonSigners := []int{len(validators)}
	validators = append(
		validators, abci.Validator{
			Address: consAddrA.Bytes(),
			Power:   power,
		},
	)

	const missBlocks int64 = 7
	for i := int64(0); i < missBlocks; i++ {
		suite.CommitWithInfo(validators, nonSigners, time.Nanosecond)
	}

	oldInfo, foundOld := suite.App.SlashingKeeper.GetValidatorSigningInfo(suite.Ctx, consAddrA)
	suite.Require().True(foundOld, "old key must have signing info before rotation")
	suite.Require().Positive(
		oldInfo.MissedBlocksCounter,
		"expected missed blocks on old key before rotation",
	)

	consKeyB := suite.ChangeKey(operatorAddr, true)
	consAddrB := consKeyB.ToConsAddr()

	_, foundOldAfter := suite.App.SlashingKeeper.GetValidatorSigningInfo(suite.Ctx, consAddrA)
	suite.Require().True(
		foundOldAfter,
		"old cons addr should still have signing info immediately after rotation (pre-prune, checklist B4)",
	)

	newInfo, foundNew := suite.App.SlashingKeeper.GetValidatorSigningInfo(suite.Ctx, consAddrB)
	suite.Require().True(foundNew)
	suite.Require().Equal(
		oldInfo.MissedBlocksCounter,
		newInfo.MissedBlocksCounter,
		"UNSPECIFIED copy should preserve MissedBlocksCounter",
	)
	sum := suite.sumMissedBitsInWindow(consAddrB)
	suite.Require().Equal(
		newInfo.MissedBlocksCounter,
		sum,
		"after UNSPECIFIED Copy, counter must equal sum of bit array (SDK invariant)",
	)
}

func (suite *KeyChangeEscapeTestSuite) dogfoodEpochWallDuration() time.Duration {
	switch suite.EpochIdentifier {
	case epochstypes.MinuteEpochID:
		return time.Minute
	case epochstypes.HourEpochID:
		return time.Hour
	case epochstypes.DayEpochID:
		return 24 * time.Hour
	case epochstypes.WeekEpochID:
		return 7 * 24 * time.Hour
	default:
		suite.Require().Failf("unknown dogfood epoch identifier %q", suite.EpochIdentifier)
		return 0
	}
}

// TestChecklist_B2_EvidenceWindowCoversDogfoodUnbonding (checklist B2) asserts default consensus
// evidence limits are looser than the minimum wall-clock and block span implied by dogfood
// EpochsUntilUnbonded under this test harness (minute epochs, TestBlockNumberPerEpoch blocks each).
// Production uses the same DefaultConsensusParams in app/test_helpers.go vs dogfood defaults.
func (suite *KeyChangeEscapeTestSuite) TestChecklist_B2_EvidenceWindowCoversDogfoodUnbonding() {
	cp := suite.App.GetConsensusParams(suite.Ctx)
	suite.Require().NotNil(cp.Evidence)

	epochs := suite.App.StakingKeeper.GetEpochsUntilUnbonded(suite.Ctx)
	epochDur := suite.dogfoodEpochWallDuration()
	minWall := time.Duration(epochs) * epochDur
	suite.Require().GreaterOrEqual(
		cp.Evidence.MaxAgeDuration,
		minWall,
		"Evidence.MaxAgeDuration must cover worst-case wall time until old-key prune (EpochsUntilUnbonded epochs)",
	)

	// Lower-bound block span: full epochs of advancement (conservative vs real prune boundary).
	minBlocks := int64(epochs) * testutil.TestBlockNumberPerEpoch
	suite.Require().GreaterOrEqual(
		cp.Evidence.MaxAgeNumBlocks,
		minBlocks,
		"Evidence.MaxAgeNumBlocks should cover at least epochs*blocks-per-epoch for test config",
	)
	// Production defaults live in app/test_helpers.go (DefaultConsensusParams); they vastly exceed test unbonding.
}

// TestChecklist_D5_PostPruneReverseLookupAndSlashNoOp (checklist D5 / B4(3)): after waiting past
// unbonding, the old consensus address loses reverse lookup and signing info; dogfood slash is a no-op.
func (suite *KeyChangeEscapeTestSuite) TestChecklist_D5_PostPruneReverseLookupAndSlashNoOp() {
	operatorAddr, consKeyA, power := suite.AddValidator(3)
	consAddrA := consKeyA.ToConsAddr()

	suite.RunToEpochEndNoEndBlockerN(suite.EpochIdentifier, 3)
	suite.CheckValidator(operatorAddr, consAddrA, power, power)

	suite.ChangeKey(operatorAddr, true)
	suite.Commit()
	suite.RunToEpochEndNoEndBlocker(suite.EpochIdentifier)
	suite.Commit()

	unbondingPeriod := suite.App.StakingKeeper.GetDogfoodParams(suite.Ctx).EpochsUntilUnbonded
	suite.RunToEpochEndNoEndBlockerN(suite.EpochIdentifier, int(unbondingPeriod)+1)
	suite.Commit()

	foundRev, _ := suite.App.OperatorKeeper.GetOperatorAddressForChainIDAndConsAddr(
		suite.Ctx, suite.ChainIDWithoutRevision, consAddrA,
	)
	suite.Require().False(foundRev, "reverse lookup for pruned old cons addr should be gone")

	_, foundSign := suite.App.SlashingKeeper.GetValidatorSigningInfo(suite.Ctx, consAddrA)
	suite.Require().False(foundSign, "signing info for pruned non-tombstone key should be deleted")

	slashFrac := suite.App.SlashingKeeper.SlashFractionDowntime(suite.Ctx)
	burned := suite.App.StakingKeeper.SlashWithInfractionReason(
		suite.Ctx,
		consAddrA,
		suite.Ctx.BlockHeight()-1,
		power,
		slashFrac,
		stakingtypes.Infraction_INFRACTION_DOWNTIME,
	)
	suite.Require().True(
		burned.IsZero(),
		"SlashWithInfractionReason on pruned cons addr should not burn (no operator mapping)",
	)
}

// TestSameEpochDoubleKeyRotation_SkipsSecondHookRisk documents when the second key replacement
// skips AfterOperatorKeyReplaced (alreadyRecorded) and therefore skips CopyValidatorSigningInfo /
// AfterValidatorCreated for the final key.
//
// Two Msg SetConsKey txs each followed by Commit() usually do NOT hit this path: dogfood EndBlock
// (when ShouldUpdateValidatorSet) runs ClearPreviousConsensusKeys, so the second tx sees
// alreadyRecorded == false and hooks run again. To reproduce the skip without an intervening
// EndBlock, this test calls OperatorKeeper.SetOperatorConsKeyForChainID twice on the same context.
func (suite *KeyChangeEscapeTestSuite) TestSameEpochDoubleKeyRotation_SkipsSecondHookRisk() {
	operatorAddr, consKeyA, power := suite.AddValidator(3)
	consAddrA := consKeyA.ToConsAddr()

	suite.RunToEpochEndNoEndBlockerN(suite.EpochIdentifier, 3)
	suite.CheckValidator(operatorAddr, consAddrA, power, power)

	consKeyB := testutiltx.GenerateConsensusKey()
	consKeyC := testutiltx.GenerateConsensusKey()
	chainID := suite.ChainIDWithoutRevision

	err := suite.App.OperatorKeeper.SetOperatorConsKeyForChainID(suite.Ctx, operatorAddr, chainID, consKeyB)
	suite.Require().NoError(err)
	_, foundB := suite.App.SlashingKeeper.GetValidatorSigningInfo(suite.Ctx, consKeyB.ToConsAddr())
	suite.Require().True(foundB, "first replacement runs hooks; B must exist in x/slashing")

	err = suite.App.OperatorKeeper.SetOperatorConsKeyForChainID(suite.Ctx, operatorAddr, chainID, consKeyC)
	suite.Require().NoError(err)

	foundFwd, wrapped, err2 := suite.App.OperatorKeeper.GetOperatorConsKeyForChainID(
		suite.Ctx, operatorAddr, chainID,
	)
	suite.Require().NoError(err2)
	suite.Require().True(foundFwd)
	suite.Require().Equal(consKeyC.ToConsAddr(), wrapped.ToConsAddr())

	_, foundC := suite.App.SlashingKeeper.GetValidatorSigningInfo(suite.Ctx, consKeyC.ToConsAddr())
	suite.Require().False(foundC,
		"second replacement without ClearPreviousConsensusKeys: alreadyRecorded skips hooks; "+
			"C missing in x/slashing (would panic on HandleValidatorSignature when C is active)")
}

// TestChecklist_D3_DoubleKeyRotationInvariants (checklist D3): two consecutive SetConsKey rotations;
// forward key is the latest; newest key keeps slashing bit-array vs counter invariant.
func (suite *KeyChangeEscapeTestSuite) TestChecklist_D3_DoubleKeyRotationInvariants() {
	operatorAddr, consKeyA, power := suite.AddValidator(3)
	consAddrA := consKeyA.ToConsAddr()

	suite.RunToEpochEndNoEndBlockerN(suite.EpochIdentifier, 3)
	suite.CheckValidator(operatorAddr, consAddrA, power, power)

	validators := make([]abci.Validator, 0)
	for _, val := range suite.ValSet.Validators {
		validator, found := suite.App.StakingKeeper.GetImuachainValidator(
			suite.Ctx, val.Address.Bytes(),
		)
		suite.Require().True(found)
		validators = append(
			validators, abci.Validator{
				Address: validator.Address,
				Power:   validator.Power,
			},
		)
	}
	nonSigners := []int{len(validators)}
	validators = append(
		validators, abci.Validator{
			Address: consAddrA.Bytes(),
			Power:   power,
		},
	)

	for i := int64(0); i < 4; i++ {
		suite.CommitWithInfo(validators, nonSigners, time.Nanosecond)
	}

	suite.ChangeKey(operatorAddr, true)
	suite.Commit()
	// Dogfood EndBlock clears previous-consensus-key records at epoch boundaries; without this,
	// a second replacement in the same epoch skips AfterOperatorKeyReplaced (alreadyRecorded).
	suite.RunToEpochEndNoEndBlocker(suite.EpochIdentifier)
	suite.Commit()

	consKeyB := suite.ChangeKey(operatorAddr, true)
	suite.Commit()
	consAddrB := consKeyB.ToConsAddr()

	suite.CheckValidator(operatorAddr, consAddrB, power, power)

	info, found := suite.App.SlashingKeeper.GetValidatorSigningInfo(suite.Ctx, consAddrB)
	suite.Require().True(found)
	suite.Require().Equal(
		info.MissedBlocksCounter,
		suite.sumMissedBitsInWindow(consAddrB),
		"after two rotations, latest key: counter equals bit-array sum",
	)
}

// TestChecklist_E1_DoubleSignSlashFreezesOperator (checklist E1): dogfood SlashWithInfractionReason
// with double-sign must set the operator frozen flag (global liveness / participation lock).
func (suite *KeyChangeEscapeTestSuite) TestChecklist_E1_DoubleSignSlashFreezesOperator() {
	operatorAddr, consKey, power := suite.AddValidator(3)
	consAddr := consKey.ToConsAddr()

	suite.RunToEpochEndNoEndBlockerN(suite.EpochIdentifier, 3)
	suite.CheckValidator(operatorAddr, consAddr, power, power)

	suite.Require().False(suite.App.OperatorKeeper.IsOperatorFrozen(suite.Ctx, operatorAddr))

	frac := suite.App.SlashingKeeper.SlashFractionDoubleSign(suite.Ctx)
	infractionHeight := suite.Ctx.BlockHeight()
	_ = suite.App.StakingKeeper.SlashWithInfractionReason(
		suite.Ctx,
		consAddr,
		infractionHeight,
		power,
		frac,
		stakingtypes.Infraction_INFRACTION_DOUBLE_SIGN,
	)

	suite.Require().True(
		suite.App.OperatorKeeper.IsOperatorFrozen(suite.Ctx, operatorAddr),
		"INFRACTION_DOUBLE_SIGN through dogfood must call FreezeOperator",
	)
}

// TestChecklist_E2_UnjailNoOpWhenOperatorFrozen (checklist E2): dogfood Unjail is a no-op when the
// operator is frozen, so MsgUnjail can succeed at the x/slashing layer while opted-in jail persists.
func (suite *KeyChangeEscapeTestSuite) TestChecklist_E2_UnjailNoOpWhenOperatorFrozen() {
	operatorAddr, consKey, power := suite.AddValidator(3)
	consAddr := consKey.ToConsAddr()

	suite.RunToEpochEndNoEndBlockerN(suite.EpochIdentifier, 3)
	suite.CheckValidator(operatorAddr, consAddr, power, power)

	suite.App.OperatorKeeper.Jail(suite.Ctx, consAddr, suite.ChainIDWithoutRevision)
	suite.Require().True(
		suite.App.OperatorKeeper.IsOperatorJailedForChainID(
			suite.Ctx, consAddr, suite.ChainIDWithoutRevision,
		),
	)

	err := suite.App.OperatorKeeper.FreezeOperator(suite.Ctx, operatorAddr)
	suite.Require().NoError(err)

	msgServer := slashingkeeper.NewMsgServerImpl(suite.App.SlashingKeeper)
	resp, err := msgServer.Unjail(
		sdk.WrapSDKContext(suite.Ctx),
		&slashingtypes.MsgUnjail{
			ValidatorAddr: sdk.ValAddress(operatorAddr).String(),
		},
	)
	suite.Require().NoError(err)
	suite.Require().NotNil(resp)

	suite.Require().True(
		suite.App.OperatorKeeper.IsOperatorJailedForChainID(
			suite.Ctx, consAddr, suite.ChainIDWithoutRevision,
		),
		"frozen operator: dogfood Unjail must not clear jail in x/operator",
	)
}

// TestChecklist_E4_DowntimeSlashRemovesImuaValidatorButRetainsSigningInfo (checklist E4):
// ApplyValidatorChanges deletes the ImuachainValidator when Comet power goes to 0, but x/slashing
// signing info for that consensus address remains until the scheduled prune path (rotation /
// unbonding) — do not assume "no ImuachainValidator" implies "no ValidatorSigningInfo".
func (suite *KeyChangeEscapeTestSuite) TestChecklist_E4_DowntimeSlashRemovesImuaValidatorButRetainsSigningInfo() {
	operatorAddr, consKey, power := suite.AddValidator(3)
	consAddr := consKey.ToConsAddr()

	suite.RunToEpochEndNoEndBlockerN(suite.EpochIdentifier, 3)
	suite.CheckValidator(operatorAddr, consAddr, power, power)

	validators := []abci.Validator{}
	for _, val := range suite.ValSet.Validators {
		validator, found := suite.App.StakingKeeper.GetImuachainValidator(
			suite.Ctx, val.Address.Bytes(),
		)
		suite.Require().True(found)
		validators = append(
			validators, abci.Validator{
				Address: validator.Address,
				Power:   validator.Power,
			},
		)
	}
	validators = append(
		validators, abci.Validator{
			Address: consAddr.Bytes(),
			Power:   power,
		},
	)
	nonSigners := []int{len(validators) - 1}

	signInfo, found := suite.App.SlashingKeeper.GetValidatorSigningInfo(suite.Ctx, consAddr)
	suite.Require().True(found)
	signedBlocksWindow := suite.App.SlashingKeeper.SignedBlocksWindow(suite.Ctx)
	minHeight := signInfo.StartHeight + signedBlocksWindow
	minSignedPerWindow := suite.App.SlashingKeeper.MinSignedPerWindow(suite.Ctx)
	maxMissed := signedBlocksWindow - minSignedPerWindow
	currentlyMissed := int64(0)
	blocksToMiss := maxMissed - currentlyMissed + 1
	blocksToReachMin := minHeight
	blocksToCommit := blocksToMiss
	if blocksToCommit < blocksToReachMin {
		blocksToCommit = blocksToReachMin
	}
	for i := int64(0); i < blocksToCommit; i++ {
		suite.CommitWithInfo(validators, nonSigners, time.Nanosecond)
	}

	_, inDogfood := suite.App.StakingKeeper.GetImuachainValidator(suite.Ctx, consAddr)
	suite.Require().False(inDogfood, "downtime slash removes validator from dogfood Imuachain set")

	signInfoAfter, foundSI := suite.App.SlashingKeeper.GetValidatorSigningInfo(suite.Ctx, consAddr)
	suite.Require().True(foundSI, "x/slashing signing info must still exist (not pruned with power-0 alone)")
	suite.Require().True(signInfoAfter.JailedUntil.After(suite.Ctx.BlockTime()))
}

// TestChecklist_E5_DoubleSignDogfoodSlashWritesOperatorSlashRecord (checklist E5): dogfood
// SlashWithInfractionReason delegates to operator Slash; slash ID encodes infraction + height.
func (suite *KeyChangeEscapeTestSuite) TestChecklist_E5_DoubleSignDogfoodSlashWritesOperatorSlashRecord() {
	operatorAddr, consKey, power := suite.AddValidator(3)
	consAddr := consKey.ToConsAddr()

	suite.RunToEpochEndNoEndBlockerN(suite.EpochIdentifier, 3)
	suite.CheckValidator(operatorAddr, consAddr, power, power)

	infractionHeight := suite.Ctx.BlockHeight()
	frac := suite.App.SlashingKeeper.SlashFractionDoubleSign(suite.Ctx)
	_ = suite.App.StakingKeeper.SlashWithInfractionReason(
		suite.Ctx,
		consAddr,
		infractionHeight,
		power,
		frac,
		stakingtypes.Infraction_INFRACTION_DOUBLE_SIGN,
	)

	slashID := operatorkeeper.GetSlashIDForDogfood(
		stakingtypes.Infraction_INFRACTION_DOUBLE_SIGN,
		infractionHeight,
	)
	slashInfo, err := suite.App.OperatorKeeper.GetOperatorSlashInfo(
		suite.Ctx, suite.AvsAddress, operatorAddr.String(), slashID,
	)
	suite.Require().NoError(err)
	suite.Require().NotNil(slashInfo)
	suite.Require().Equal(uint32(stakingtypes.Infraction_INFRACTION_DOUBLE_SIGN), slashInfo.SlashType)
	suite.Require().Equal(infractionHeight, slashInfo.EventHeight)
	suite.Require().Equal(frac, slashInfo.SlashProportion)
}

// TestChecklist_E6_SecondEquivocationSameConsAddrIgnored (checklist E6): x/evidence short-circuits
// when the consensus addr is already tombstoned, so dogfood SlashWithInfractionReason (and
// FreezeOperator) are not invoked again — no chain panic from FreezeOperator idempotency.
func (suite *KeyChangeEscapeTestSuite) TestChecklist_E6_SecondEquivocationSameConsAddrIgnored() {
	operatorAddr, consKey, power := suite.AddValidator(3)
	consAddr := consKey.ToConsAddr()

	suite.RunToEpochEndNoEndBlockerN(suite.EpochIdentifier, 3)
	suite.CheckValidator(operatorAddr, consAddr, power, power)

	infractionHeight := suite.Ctx.BlockHeight()
	infractionTime := suite.Ctx.BlockTime()

	suite.SubmitEvidence(consAddr, infractionHeight, infractionTime, power)
	suite.Require().True(suite.App.SlashingKeeper.IsTombstoned(suite.Ctx, consAddr))
	suite.Require().True(suite.App.OperatorKeeper.IsOperatorFrozen(suite.Ctx, operatorAddr))

	// Same evidence shape again: must not panic; evidence handler ignores already-tombstoned.
	suite.SubmitEvidence(consAddr, infractionHeight, infractionTime, power)

	suite.Require().True(suite.App.OperatorKeeper.IsOperatorFrozen(suite.Ctx, operatorAddr))
}

// TestChecklist_F1_RepeatedDoubleSignSlashDoesNotPanicWhenAlreadyFrozen verifies that direct
// repeated double-sign slash entry is idempotent with respect to freeze semantics.
func (suite *KeyChangeEscapeTestSuite) TestChecklist_F1_RepeatedDoubleSignSlashDoesNotPanicWhenAlreadyFrozen() {
	operatorAddr, consKey, power := suite.AddValidator(3)
	consAddr := consKey.ToConsAddr()

	suite.RunToEpochEndNoEndBlockerN(suite.EpochIdentifier, 3)
	suite.CheckValidator(operatorAddr, consAddr, power, power)

	frac := suite.App.SlashingKeeper.SlashFractionDoubleSign(suite.Ctx)
	height := suite.Ctx.BlockHeight()
	_ = suite.App.StakingKeeper.SlashWithInfractionReason(
		suite.Ctx, consAddr, height, power, frac, stakingtypes.Infraction_INFRACTION_DOUBLE_SIGN,
	)
	suite.Require().True(suite.App.OperatorKeeper.IsOperatorFrozen(suite.Ctx, operatorAddr))

	// Second call must not panic even though operator is already frozen.
	suite.Require().NotPanics(func() {
		_ = suite.App.StakingKeeper.SlashWithInfractionReason(
			suite.Ctx, consAddr, height+1, power, frac, stakingtypes.Infraction_INFRACTION_DOUBLE_SIGN,
		)
	})
	suite.Require().True(suite.App.OperatorKeeper.IsOperatorFrozen(suite.Ctx, operatorAddr))
}

// TestChecklist_J1_RandomizedLifecycleInvariant runs a deterministic randomized sequence of
// key rotations + epoch transitions and checks that the latest active key keeps the slashing
// counter/bit-array invariant whenever its signing info exists.
func (suite *KeyChangeEscapeTestSuite) TestChecklist_J1_RandomizedLifecycleInvariant() {
	operatorAddr, consKey, power := suite.AddValidator(3)
	consAddr := consKey.ToConsAddr()

	suite.RunToEpochEndNoEndBlockerN(suite.EpochIdentifier, 3)
	suite.CheckValidator(operatorAddr, consAddr, power, power)

	r := rand.New(rand.NewSource(42))
	currentConsAddr := consAddr

	for i := 0; i < 12; i++ {
		// occasionally accrue misses on the currently active key
		if r.Intn(3) == 0 {
			validators := []abci.Validator{}
			for _, val := range suite.ValSet.Validators {
				validator, found := suite.App.StakingKeeper.GetImuachainValidator(
					suite.Ctx, val.Address.Bytes(),
				)
				suite.Require().True(found)
				validators = append(validators, abci.Validator{
					Address: validator.Address,
					Power:   validator.Power,
				})
			}
			validators = append(validators, abci.Validator{
				Address: currentConsAddr.Bytes(),
				Power:   power,
			})
			suite.CommitWithInfo(validators, []int{len(validators) - 1}, time.Nanosecond)
		}

		// rotate key and force epoch boundary so hooks execute on each turn
		newKey := suite.ChangeKey(operatorAddr, true)
		suite.Commit()
		suite.RunToEpochEndNoEndBlocker(suite.EpochIdentifier)
		suite.Commit()
		currentConsAddr = newKey.ToConsAddr()

		foundFwd, wrapped, err := suite.App.OperatorKeeper.GetOperatorConsKeyForChainID(
			suite.Ctx, operatorAddr, suite.ChainIDWithoutRevision,
		)
		suite.Require().NoError(err)
		suite.Require().True(foundFwd)
		suite.Require().Equal(currentConsAddr, wrapped.ToConsAddr())

		info, found := suite.App.SlashingKeeper.GetValidatorSigningInfo(suite.Ctx, currentConsAddr)
		if found {
			suite.Require().Equal(
				info.MissedBlocksCounter,
				suite.sumMissedBitsInWindow(currentConsAddr),
				"randomized lifecycle: counter equals bit-array sum on current key",
			)
		}
	}
}

// TestChecklist_I1_RotationBoundaryUnspecifiedSlashMigratesToCurrentKey hardens oracle-style
// UNSPECIFIED slash semantics: when slash is reported on a stale key after rotation, dogfood
// uses slashingOldKey and migrates signing state to the current key.
func (suite *KeyChangeEscapeTestSuite) TestChecklist_I1_RotationBoundaryUnspecifiedSlashMigratesToCurrentKey() {
	operatorAddr, consKeyA, power := suite.AddValidator(3)
	consAddrA := consKeyA.ToConsAddr()

	suite.RunToEpochEndNoEndBlockerN(suite.EpochIdentifier, 3)
	suite.CheckValidator(operatorAddr, consAddrA, power, power)

	// Build some missed-block debt on key A before rotation.
	validators := []abci.Validator{}
	for _, val := range suite.ValSet.Validators {
		validator, found := suite.App.StakingKeeper.GetImuachainValidator(
			suite.Ctx, val.Address.Bytes(),
		)
		suite.Require().True(found)
		validators = append(validators, abci.Validator{
			Address: validator.Address,
			Power:   validator.Power,
		})
	}
	validators = append(validators, abci.Validator{
		Address: consAddrA.Bytes(),
		Power:   power,
	})
	for i := 0; i < 5; i++ {
		suite.CommitWithInfo(validators, []int{len(validators) - 1}, time.Nanosecond)
	}
	infoA, foundA := suite.App.SlashingKeeper.GetValidatorSigningInfo(suite.Ctx, consAddrA)
	suite.Require().True(foundA)
	suite.Require().Positive(infoA.MissedBlocksCounter)

	// Rotate A -> B, process epoch boundary so B is active while A reverse-mapping still exists.
	consKeyB := suite.ChangeKey(operatorAddr, true)
	consAddrB := consKeyB.ToConsAddr()
	suite.Commit()
	suite.RunToEpochEndNoEndBlocker(suite.EpochIdentifier)
	suite.Commit()

	frac := suite.App.SlashingKeeper.SlashFractionDowntime(suite.Ctx)
	_ = suite.App.StakingKeeper.SlashWithInfractionReason(
		suite.Ctx,
		consAddrA,
		suite.Ctx.BlockHeight()-1,
		power,
		frac,
		stakingtypes.Infraction_INFRACTION_UNSPECIFIED,
	)

	infoB, foundB := suite.App.SlashingKeeper.GetValidatorSigningInfo(suite.Ctx, consAddrB)
	suite.Require().True(foundB, "UNSPECIFIED slash on stale A should copy/migrate to current B")
	suite.Require().Equal(
		infoB.MissedBlocksCounter,
		suite.sumMissedBitsInWindow(consAddrB),
		"migrated current key must keep counter/bit-array invariant",
	)
}

// TestChecklist_I2_JailOnOldKeyAppliesToCurrentKeyCoherently verifies jail coherence after key
// rotation: jailing by old consensus address propagates through operator state, so current key is
// considered jailed too (single opted-info source of truth).
func (suite *KeyChangeEscapeTestSuite) TestChecklist_I2_JailOnOldKeyAppliesToCurrentKeyCoherently() {
	operatorAddr, consKeyA, power := suite.AddValidator(3)
	consAddrA := consKeyA.ToConsAddr()

	suite.RunToEpochEndNoEndBlockerN(suite.EpochIdentifier, 3)
	suite.CheckValidator(operatorAddr, consAddrA, power, power)

	consKeyB := suite.ChangeKey(operatorAddr, true)
	consAddrB := consKeyB.ToConsAddr()
	suite.Commit()
	suite.RunToEpochEndNoEndBlocker(suite.EpochIdentifier)
	suite.Commit()

	suite.App.StakingKeeper.Jail(suite.Ctx, consAddrA)

	suite.Require().True(
		suite.App.OperatorKeeper.IsOperatorJailedForChainID(
			suite.Ctx, consAddrA, suite.ChainIDWithoutRevision,
		),
		"old key lookup still maps to operator and must report jailed",
	)
	suite.Require().True(
		suite.App.OperatorKeeper.IsOperatorJailedForChainID(
			suite.Ctx, consAddrB, suite.ChainIDWithoutRevision,
		),
		"current key should observe same jailed operator state",
	)

	// SDK unjail message should clear jail when not frozen.
	suite.Unjail(operatorAddr, true)
	suite.Require().False(
		suite.App.OperatorKeeper.IsOperatorJailedForChainID(
			suite.Ctx, consAddrB, suite.ChainIDWithoutRevision,
		),
	)
}
