package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdkmath "cosmossdk.io/math"

	"github.com/cockroachdb/errors"
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	evidencetypes "github.com/cosmos/cosmos-sdk/x/evidence/types"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/imua-xyz/imuachain/testutil"
	testutiltx "github.com/imua-xyz/imuachain/testutil/tx"
	"github.com/imua-xyz/imuachain/utils"
	assetskeeper "github.com/imua-xyz/imuachain/x/assets/keeper"
	assetstypes "github.com/imua-xyz/imuachain/x/assets/types"
	delegationtypes "github.com/imua-xyz/imuachain/x/delegation/types"
	operatortypes "github.com/imua-xyz/imuachain/x/operator/types"
	"github.com/stretchr/testify/suite"
)

var a *KeyChangeEscapeTestSuite

type KeyChangeEscapeTestSuite struct {
	testutil.BaseTestSuite
	EpochDuration time.Duration
}

func TestKeyChangeEscapeTestSuite(t *testing.T) {
	a = new(KeyChangeEscapeTestSuite)
	suite.Run(t, a)
}

func (suite *KeyChangeEscapeTestSuite) SetupTest() {
	suite.DoSetupTest()
	epochID := suite.App.StakingKeeper.GetEpochIdentifier(suite.Ctx)
	epochInfo, _ := suite.App.EpochsKeeper.GetEpochInfo(suite.Ctx, epochID)
	suite.EpochDuration = epochInfo.Duration + time.Nanosecond
}

// what we want to test is a validator who:
// 1. performs slashable action
// 2. goes offline to be kicked out of the set
// 3. changes consensus key (to cause pruning of the old key)
// 4. is reported against, and correctly slashed.
// 5. cannot join the validator set (with the new or the old key).
func (suite *KeyChangeEscapeTestSuite) TestKeyChangeEscape() {
	// move a few blocks ahead
	for i := 0; i < 10; i++ {
		suite.Commit()
	}
	// add a new validator - we use a new one to avoid impacting other tests
	epochIdentifier := suite.App.StakingKeeper.GetEpochIdentifier(suite.Ctx)
	chainID := suite.Ctx.ChainID()
	chainIDWithoutRevision := utils.ChainIDWithoutRevision(chainID)
	avsAddress := utils.GenerateAVSAddress(chainIDWithoutRevision)
	operatorAddr, _ := testutiltx.NewAccAddressAndKey()
	consKey := testutiltx.GenerateConsensusKey()
	consAddr := consKey.ToConsAddr()
	// register the operator
	{
		res, err := suite.OperatorMsgServer.RegisterOperator(
			sdk.WrapSDKContext(suite.Ctx),
			&operatortypes.RegisterOperatorReq{
				FromAddress: operatorAddr.String(),
				Info: &operatortypes.OperatorInfo{
					OperatorAddr: operatorAddr.String(),
					Description:  stakingtypes.NewDescription("operator3", "", "", "", ""),
					Commission: stakingtypes.NewCommission(
						sdk.ZeroDec(), sdk.ZeroDec(), sdk.ZeroDec(),
					),
				},
			},
		)
		suite.Require().NoError(err)
		suite.Require().NotNil(res)
	}
	// delegate to the operator 3 times the minimum required
	power := int64(0)
	{
		factor := int64(3)
		requiredUsd := suite.App.StakingKeeper.GetDogfoodParams(
			suite.Ctx,
		).MinSelfDelegation
		requiredUsd = requiredUsd.Mul(sdk.NewInt(factor))
		usd, err := suite.App.OperatorKeeper.GetOrCalculateOperatorUSDValues(
			suite.Ctx, operatorAddr, avsAddress,
		)
		requiredUsdDec := sdkmath.LegacyNewDecFromInt(requiredUsd)
		suite.Require().NoError(err)
		haveUsd := usd.SelfUSDValue
		if haveUsd.LT(requiredUsdDec) {
			diff := requiredUsdDec.Sub(haveUsd)
			usdAmountHuman := diff.TruncateInt64() + 1
			// rough!
			power = usdAmountHuman
			usdAmountDecimals := math.NewIntWithDecimal(usdAmountHuman, int(suite.Assets[0].Decimals))
			stakerAddr := testutiltx.GenerateAddress()
			depositParams := &assetskeeper.DepositWithdrawParams{
				ClientChainLzID: suite.Assets[0].LayerZeroChainID,
				Action:          assetstypes.DepositLST,
				AssetsAddress:   common.HexToAddress(suite.Assets[0].Address).Bytes(),
				StakerAddress:   stakerAddr.Bytes(),
				OpAmount:        usdAmountDecimals,
			}
			postDepositAmount, err := suite.App.AssetsKeeper.PerformDepositOrWithdraw(
				suite.Ctx, depositParams,
			)
			suite.Require().NoError(err)
			suite.Require().Equal(usdAmountDecimals, postDepositAmount)
			delegateParams := delegationtypes.NewDelegationOrUndelegationParams(
				depositParams.ClientChainLzID,
				depositParams.AssetsAddress,
				operatorAddr,
				depositParams.StakerAddress,
				usdAmountDecimals,
				common.BytesToHash([]byte("test")),
				false,
			)
			_, _, err = suite.App.DelegationKeeper.DelegateTo(suite.Ctx, delegateParams)
			suite.Require().NoError(err)
			// associate this to be self stake
			err = suite.App.DelegationKeeper.AssociateOperatorWithStaker(
				suite.Ctx, depositParams.ClientChainLzID, operatorAddr, stakerAddr.Bytes(),
			)
			suite.Require().NoError(err)
			suite.Commit()
			suite.RunToEpochEndNoEndBlocker(epochIdentifier)
		}
		usd, err = suite.App.OperatorKeeper.GetOrCalculateOperatorUSDValues(
			suite.Ctx, operatorAddr, avsAddress,
		)
		suite.Require().NoError(err)
	}
	// opt in
	{
		res, err := suite.OperatorMsgServer.OptIntoAVS(
			sdk.WrapSDKContext(suite.Ctx),
			&operatortypes.OptIntoAVSReq{
				FromAddress:   operatorAddr.String(),
				AvsAddress:    avsAddress,
				PublicKeyJSON: consKey.ToJSON(),
			},
		)
		suite.Require().NoError(err)
		suite.Require().NotNil(res)
	}
	// go forward 3 epochs for the validator to activate
	suite.RunToEpochEndNoEndBlockerN(epochIdentifier, 3)
	found, _ := suite.App.OperatorKeeper.GetOperatorAddressForChainIDAndConsAddr(
		suite.Ctx, chainIDWithoutRevision, consAddr,
	)
	suite.Assert().True(found)
	valAddr := sdk.ValAddress(operatorAddr)
	found, wrappedKey, err := suite.App.OperatorKeeper.GetOperatorConsKeyForChainID(
		suite.Ctx, operatorAddr, chainIDWithoutRevision,
	)
	suite.Assert().True(found)
	suite.Assert().NoError(err)
	infractionHeight := suite.Ctx.BlockHeight()
	// we also need the power of the validator
	// Tendermint waits for 1 block before applying any changes to the validator set
	// this delay is accounted for in x/evidence
	// for example, if you send a validator set update during EndBlock 69
	// it should have ideally been effective at block 70; however,
	// it starts at the beginning of block 71.
	// so if the evidence is for double signing at block 71
	// we go back to block 70 to get the distribution or infraction height.
	// example:
	// total power 1000
	// at block 69, A unstakes 100 tokens, remaining 900
	// tendermint still sees 1000
	// at block 70, B unstakes 50 tokens, remaining 850
	// tendermint still sees 1000
	// at block 71, voting power is 900 and double signing happens
	// so we calculate 70 as the infraction height and punish only B
	// since A had already unstaked by then
	validator, found := suite.App.StakingKeeper.GetValidator(
		suite.Ctx, valAddr,
	)
	suite.Assert().True(found)
	tokens := validator.GetTokens()
	validatorPower := sdk.TokensToConsensusPower(tokens, sdk.DefaultPowerReduction)
	suite.Require().Equal(
		power, validatorPower, "power %d, validatorPower %d", power, validatorPower,
	)

	// we abstract away the proving and inclusion of the evidence into the canonical chain
	// by directly calling the relevant x/evidence function. however, for understanding, note
	// that the detection of the double signing and its inclusion is done by CometBFT, which
	// validates it and puts it into the abci.RequestBeginBlock.ByzantineValidators for the
	// state machine to handle. the x/evidence's BeginBlocker handles it.
	misbehavior := abci.Misbehavior{
		Type: abci.MisbehaviorType_DUPLICATE_VOTE,
		Validator: abci.Validator{
			Address: wrappedKey.ToConsAddr(),
			Power:   power,
		},
		Height:           infractionHeight,
		Time:             suite.Ctx.BlockTime(),
		TotalVotingPower: suite.TotalPower,
	}
	evidence := evidencetypes.FromABCIEvidence(misbehavior)
	equivocation := evidence.(*evidencetypes.Equivocation)
	// we submit the evidence after jailing the validator, but we prepare it at this height
	submitEvidence := func(e *evidencetypes.Equivocation) {
		suite.App.EvidenceKeeper.HandleEquivocationEvidence(suite.Ctx, e)
	}

	// original validator set
	validators := []abci.Validator{}
	for _, val := range suite.ValSet.Validators {
		validator, found := suite.App.StakingKeeper.GetImuachainValidator(
			suite.Ctx, val.Address.Bytes(),
		)
		suite.Assert().True(found)
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
			Address: wrappedKey.ToConsAddr().Bytes(),
			Power:   power,
		},
	)

	// 2. take the validator offline
	commit := func(t time.Duration) {
		header := suite.Ctx.BlockHeader()
		suite.App.EndBlocker(suite.Ctx, abci.RequestEndBlock{Height: header.Height})
		suite.App.Commit()
		header.Height++
		header.Time = header.Time.Add(t)
		header.AppHash = suite.App.LastCommitID().Hash
		suite.Ctx = suite.Ctx.WithBlockHeader(header)
		// in the begin blocker, we must set a validator's signing status
		req := abci.RequestBeginBlock{
			Header: header,
			LastCommitInfo: abci.CommitInfo{
				Round: 0,
				Votes: []abci.VoteInfo{
					{
						Validator:       validators[0],
						SignedLastBlock: true,
					},
					{
						Validator:       validators[1],
						SignedLastBlock: true,
					},
					{
						Validator:       validators[2],
						SignedLastBlock: false,
					},
				},
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
	// check starting power via Delegation function
	// the previous value is via Validator function
	delegation := suite.App.StakingKeeper.Delegation(
		suite.Ctx, operatorAddr, valAddr,
	)
	delegationTokens := validator.TokensFromShares(
		delegation.GetShares(),
	).TruncateInt()
	delegationPower := sdk.TokensToConsensusPower(
		delegationTokens, sdk.DefaultPowerReduction,
	)
	suite.Require().Equal(power, delegationPower)
	// determine the amount of blocks to commit
	consAddr = wrappedKey.ToConsAddr()
	signInfo, found := suite.App.SlashingKeeper.GetValidatorSigningInfo(suite.Ctx, consAddr)
	suite.Assert().True(found)
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
		commit(time.Nanosecond)
	}
	signInfo, found = suite.App.SlashingKeeper.GetValidatorSigningInfo(suite.Ctx, consAddr)
	suite.Assert().True(found)
	// check if the validator is available?
	_, found = suite.App.StakingKeeper.GetImuachainValidator(suite.Ctx, consAddr)
	suite.Assert().False(found)
	// now that the validator is jailed, change its key
	newConsKey := testutiltx.GenerateConsensusKey()
	response, err := suite.OperatorMsgServer.SetConsKey(
		sdk.WrapSDKContext(suite.Ctx),
		&operatortypes.SetConsKeyReq{
			Address:       operatorAddr.String(),
			AvsAddress:    avsAddress,
			PublicKeyJSON: newConsKey.ToJSON(),
		},
	)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	suite.Commit()
	// now that the key is removed, we advance some epochs
	suite.RunToEpochEndNoEndBlocker(epochIdentifier)
	// check our stake
	// this is a rough appromixation!
	checkPower := func(
		slashProportion sdkmath.LegacyDec, startingPower int64,
	) (endingPower int64) {
		slashValue := sdkmath.LegacyNewDec(startingPower).Mul(slashProportion)
		effectiveSlashProportion := sdkmath.LegacyMinDec(
			sdkmath.LegacyNewDec(1), slashValue.QuoInt64(startingPower),
		)
		subtract := effectiveSlashProportion.MulInt64(startingPower)
		endingPower = sdkmath.LegacyNewDec(startingPower).Sub(subtract).TruncateInt64()
		delegation = suite.App.StakingKeeper.Delegation(
			suite.Ctx, operatorAddr, valAddr,
		)
		delegationTokens = validator.TokensFromShares(
			delegation.GetShares(),
		).TruncateInt()
		delegationPower = sdk.TokensToConsensusPower(
			delegationTokens, sdk.DefaultPowerReduction,
		)
		suite.Require().Equal(endingPower, delegationPower)
		return
	}
	power = checkPower(
		sdkmath.LegacyMinDec(
			sdkmath.LegacyNewDec(1), suite.App.SlashingKeeper.SlashFractionDowntime(suite.Ctx),
		), power,
	)
	// now the power must have been reduced, but by how much?
	// submit the evidence for slashing
	submitEvidence(equivocation)
	// include it in the latest block
	suite.Commit()
	power = checkPower(
		sdkmath.LegacyMinDec(
			sdkmath.LegacyNewDec(1), suite.App.SlashingKeeper.SlashFractionDoubleSign(suite.Ctx),
		), power,
	)
	// check
	signInfo, found = suite.App.SlashingKeeper.GetValidatorSigningInfo(suite.Ctx, consAddr)
	suite.Assert().True(found)
	suite.Assert().True(signInfo.Tombstoned)
	suite.Assert().Equal(signInfo.JailedUntil.UTC(), evidencetypes.DoubleSignJailEndTime.UTC())
	// go forward in time
	suite.RunToEpochEndNoEndBlockerN(epochIdentifier, 1)
	// now, we need to check if validator can rejoin the set
	// to do so, the first thing required is unjailing
	// to unjail, we first add some stake
	requiredUsd := suite.App.StakingKeeper.GetDogfoodParams(suite.Ctx).MinSelfDelegation
	usd, err := suite.App.OperatorKeeper.GetOrCalculateOperatorUSDValues(
		suite.Ctx, operatorAddr, avsAddress,
	)
	requiredUsdDec := sdkmath.LegacyNewDecFromInt(requiredUsd)
	suite.Require().NoError(err)
	haveUsd := usd.SelfUSDValue
	if haveUsd.LT(requiredUsdDec) {
		diff := requiredUsdDec.Sub(haveUsd)
		usdAmountHuman := diff.TruncateInt64() + 1
		usdAmountDecimals := math.NewIntWithDecimal(usdAmountHuman, int(suite.Assets[0].Decimals))
		stakerAddr := testutiltx.GenerateAddress()
		depositParams := &assetskeeper.DepositWithdrawParams{
			ClientChainLzID: suite.Assets[0].LayerZeroChainID,
			Action:          assetstypes.DepositLST,
			AssetsAddress:   common.HexToAddress(suite.Assets[0].Address).Bytes(),
			StakerAddress:   stakerAddr.Bytes(),
			OpAmount:        usdAmountDecimals,
		}
		postDepositAmount, err := suite.App.AssetsKeeper.PerformDepositOrWithdraw(
			suite.Ctx, depositParams,
		)
		suite.Require().NoError(err)
		suite.Require().Equal(usdAmountDecimals, postDepositAmount)
		delegateParams := delegationtypes.NewDelegationOrUndelegationParams(
			depositParams.ClientChainLzID,
			depositParams.AssetsAddress,
			operatorAddr,
			depositParams.StakerAddress,
			usdAmountDecimals,
			common.BytesToHash([]byte("test")),
			false,
		)
		_, _, err = suite.App.DelegationKeeper.DelegateTo(suite.Ctx, delegateParams)
		suite.Require().NoError(err)
		// associate this to be self stake
		err = suite.App.DelegationKeeper.AssociateOperatorWithStaker(
			suite.Ctx, depositParams.ClientChainLzID, operatorAddr, stakerAddr.Bytes(),
		)
		suite.Require().NoError(err)
		suite.Commit()
		suite.RunToEpochEndNoEndBlocker(epochIdentifier)
	}
	usd, err = suite.App.OperatorKeeper.GetOrCalculateOperatorUSDValues(
		suite.Ctx, operatorAddr, avsAddress,
	)
	suite.Require().NoError(err)
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
	{
		msgServer := slashingkeeper.NewMsgServerImpl(suite.App.SlashingKeeper)
		resp, err := msgServer.Unjail(
			sdk.WrapSDKContext(suite.Ctx),
			&slashingtypes.MsgUnjail{
				ValidatorAddr: sdk.ValAddress(operatorAddr).String(),
			},
		)
		suite.Require().Error(err)
		suite.Require().True(errors.Is(err, slashingtypes.ErrValidatorJailed))
		suite.Require().Nil(resp)
	}
	// check if we can submit the evidence again
	suite.Commit()
	misbehavior.Height = suite.Ctx.BlockHeight()
	evidence = evidencetypes.FromABCIEvidence(misbehavior)
	equivocation = evidence.(*evidencetypes.Equivocation)
	submitEvidence(equivocation)
	// check power, no slashing should be applied.
	power = checkPower(sdkmath.LegacyZeroDec(), power)
}
