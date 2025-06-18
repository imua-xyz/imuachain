package keeper_test

import (
	"fmt"
	"strings"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	testutiltx "github.com/imua-xyz/imuachain/testutil/tx"
	assetstype "github.com/imua-xyz/imuachain/x/assets/types"
	avstypes "github.com/imua-xyz/imuachain/x/avs/types"
	dogfoodtypes "github.com/imua-xyz/imuachain/x/dogfood/types"
	"github.com/imua-xyz/imuachain/x/feedistribution/keeper"
	feedistributiontypes "github.com/imua-xyz/imuachain/x/feedistribution/types"
)

func (suite *KeeperTestSuite) registerRewardAssets(avsList []common.Address) {
	// register reward assets for the test AVSs
	for i, avs := range avsList {
		rewardAssets := make([]assetstype.AssetInfo, 0)
		for j := 0; j < RewardAssetNumberPerAVS; j++ {
			addr, _ := testutiltx.NewAddrKey()
			assetName := fmt.Sprintf("avs%dRewardAsset%d", i, j)
			assetSymbol := fmt.Sprintf("avs%dsymbol%d", i, j)
			rewardAssets = append(rewardAssets, assetstype.AssetInfo{
				Name:             assetName,
				Symbol:           assetSymbol,
				Address:          strings.ToLower(addr.String()),
				Decimals:         6,
				LayerZeroChainID: suite.ClientChains[0].LayerZeroChainID,
			})
		}

		err := suite.App.DistrKeeper.SetAVSRewardAssets(suite.Ctx, strings.ToLower(avs.String()), rewardAssets)
		suite.Require().NoError(err)
	}
}

func (suite *KeeperTestSuite) setRewardParams(avsList []common.Address) {
	// set reward parameter for the test AVSs
	for _, avs := range avsList {
		// set reward parameter
		err := suite.App.DistrKeeper.SetAVSRewardParam(suite.Ctx, strings.ToLower(avs.String()), DefaultRewardParameter)
		suite.Require().NoError(err)
	}
}

func (suite *KeeperTestSuite) mintDogfoodTestReward() {
	// mint test rewards
	mintParam := suite.App.ImmintKeeper.GetParams(suite.Ctx)
	mintedCoin := sdk.NewCoin(
		mintParam.MintDenom, mintParam.EpochReward,
	)
	mintedCoins := sdk.NewCoins(mintedCoin)
	err := suite.App.ImmintKeeper.MintCoins(suite.Ctx, mintedCoins)
	suite.Require().NoError(err)
	err = suite.App.ImmintKeeper.AddCollectedFees(suite.Ctx, mintedCoins)
	suite.Require().NoError(err)
}

func (suite *KeeperTestSuite) TestSetAVSRewardParam() {
	suite.prepareTestBase(1, 1, 1)
	suite.setRewardParams(suite.testAVSs)

	// Verify the parameters were set correctly
	for _, avs := range suite.testAVSs {
		param, err := suite.App.DistrKeeper.GetAVSRewardParam(suite.Ctx, strings.ToLower(avs.String()))
		suite.Require().NoError(err)
		suite.Require().Equal(DefaultRewardParameter, *param)
	}
}

func (suite *KeeperTestSuite) TestSetAVSEpochRewardExclusive() {
	testStakerNumber := 1
	testOperatorNumber := 1
	testAVSNumber := 1
	testcases := []struct {
		name         string
		malleate     func() sdk.DecCoins
		readOnly     bool
		expPass      bool
		errContains  string
		rewardExists bool
	}{
		{
			name: "fail - the reward assets haven't been registered",
			malleate: func() sdk.DecCoins {
				return sdk.DecCoins{
					{
						Denom:  "InvalidSymbol",
						Amount: sdk.NewDec(1),
					},
				}
			},
			readOnly:     false,
			expPass:      false,
			errContains:  feedistributiontypes.ErrAVSRewardAssetNotFound.Error(),
			rewardExists: false,
		},
		{
			name: "pass - set epoch rewards for the avs exclusively",
			malleate: func() sdk.DecCoins {
				suite.registerRewardAssets(suite.testAVSs)
				avsStr := strings.ToLower(suite.testAVSs[0].String())
				avsRewardAsset, err := suite.App.DistrKeeper.GetAllRewardAssetsByAVS(suite.Ctx, avsStr)
				suite.Require().NoError(err)

				epochRewards := make(sdk.DecCoins, 0)
				for _, rewardAsset := range avsRewardAsset.AvsRewardAssets {
					multiplier := math.NewIntWithDecimal(1, int(rewardAsset.AssetBasicInfo.Decimals)) // 10^decimals
					rewardAmount := sdk.NewDec(1).MulInt(multiplier)
					epochRewards = append(epochRewards, sdk.DecCoin{
						Denom:  rewardAsset.AssetBasicInfo.Symbol,
						Amount: rewardAmount,
					})
				}
				return epochRewards
			},
			readOnly:     false,
			expPass:      true,
			rewardExists: true,
		},
		{
			name: "pass - set null rewards to disable the rewards distribution",
			malleate: func() sdk.DecCoins {
				return nil
			},
			readOnly:     false,
			expPass:      true,
			rewardExists: true,
		},
	}

	for _, tc := range testcases {
		tc := tc
		s.Run(tc.name, func() {
			s.SetupTest() // Reset state for each test case
			s.prepareTestBase(testStakerNumber, testOperatorNumber, testAVSNumber)

			rewards := tc.malleate()
			testAvs := strings.ToLower(suite.testAVSs[0].String())
			err := suite.App.DistrKeeper.SetAVSEpochRewardExclusive(suite.Ctx, testAvs, rewards)
			if tc.expPass {
				s.Require().NoError(err)
			} else if tc.errContains != "" {
				s.Require().ErrorContains(err, tc.errContains)
			}
			// checkDelegationStates the state after setting rewards
			distributionInfo, err := suite.App.DistrKeeper.GetAVSRewardDistribution(suite.Ctx, testAvs)
			if !tc.rewardExists {
				s.Require().ErrorIs(err, feedistributiontypes.ErrNotAVSRewardDistribution)
			} else {
				s.Require().Equal(rewards, distributionInfo.Rewards)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestAVSRewardDistributionByParam() {
	// using two operators to test whether the operator reward proportion is correct.
	testStakerNumber := TestStakerNumber
	testOperatorNumber := TestOperatorNumber
	testAVSNumber := 1
	testcases := []struct {
		name        string
		malleate    func() string
		isDogfood   bool
		readOnly    bool
		expPass     bool
		errContains string
		checkState  func(rewardsAndProportions feedistributiontypes.EpochRewardsAndProportions)
	}{
		{
			name: "fail - the reward parameter hasn't been configured",
			malleate: func() string {
				suite.prepareTestBase(testStakerNumber, testOperatorNumber, testAVSNumber)
				return strings.ToLower(suite.testAVSs[0].String())
			},
			isDogfood:   false,
			readOnly:    false,
			expPass:     false,
			errContains: feedistributiontypes.ErrNoKeyInTheStore.Error(),
		},
		{
			name: "fail - the reward distribution info hasn't been configured",
			malleate: func() string {
				suite.prepareTestBase(testStakerNumber, testOperatorNumber, testAVSNumber)
				// set reward parameter
				suite.setRewardParams(suite.testAVSs)
				suite.registerRewardAssets(suite.testAVSs)
				return strings.ToLower(suite.testAVSs[0].String())
			},
			isDogfood:   false,
			readOnly:    false,
			expPass:     false,
			errContains: feedistributiontypes.ErrNotAVSRewardDistribution.Error(),
		},
		{
			name: "fail - the AVS USD value hasn't been updated because it hasn't run to the end of the epoch.",
			malleate: func() string {
				suite.prepareTestBase(testStakerNumber, testOperatorNumber, testAVSNumber)
				// set reward parameter
				suite.setRewardParams(suite.testAVSs)
				suite.registerRewardAssets(suite.testAVSs)
				// set reward distribution
				suite.setAVSEpochRewards(suite.testAVSs, DefaultEpochRewardAmount)
				return strings.ToLower(suite.testAVSs[0].String())
			},
			isDogfood:   false,
			readOnly:    false,
			expPass:     false,
			errContains: feedistributiontypes.ErrNoKeyInTheStore.Error(),
		},
		{
			name: "pass - the avs reward distribution should be fetched correctly, but the operator reward proportions should be null because null epoch rewards are configured",
			malleate: func() string {
				suite.prepareTestBase(testStakerNumber, testOperatorNumber, testAVSNumber)
				// set reward parameter
				suite.setRewardParams(suite.testAVSs)
				suite.registerRewardAssets(suite.testAVSs)
				// set reward distribution
				suite.setAVSEpochRewards(suite.testAVSs, 0)
				// run to the end of epoch
				suite.RunToEpochEnd(dogfoodtypes.DefaultEpochIdentifier)
				return strings.ToLower(suite.testAVSs[0].String())
			},
			isDogfood: false,
			readOnly:  false,
			expPass:   true,
			checkState: func(rewardsAndProportions feedistributiontypes.EpochRewardsAndProportions) {
				suite.Require().Equal(sdk.DecCoins(nil), rewardsAndProportions.Rewards)
				suite.Require().Equal([]feedistributiontypes.OperatorRewardProportion(nil), rewardsAndProportions.OperatorRewardProportions)
			},
		},
		{
			name: "pass - the AVS reward distribution should be fetched correctly. The reward proportion of each test operator should be 0.5 because there are 2 test operators with the same deposits and delegations.",
			malleate: func() string {
				suite.prepareTestBase(testStakerNumber, testOperatorNumber, testAVSNumber)
				// set reward parameter
				suite.setRewardParams(suite.testAVSs)
				suite.registerRewardAssets(suite.testAVSs)
				// set reward distribution
				suite.setAVSEpochRewards(suite.testAVSs, DefaultEpochRewardAmount)
				// run to the end of epoch
				suite.RunToEpochEnd(dogfoodtypes.DefaultEpochIdentifier)
				return strings.ToLower(suite.testAVSs[0].String())
			},
			isDogfood: false,
			readOnly:  false,
			expPass:   true,
			checkState: func(rewardsAndProportions feedistributiontypes.EpochRewardsAndProportions) {
				suite.Require().GreaterOrEqual(len(rewardsAndProportions.Rewards), 1)
				suite.Require().Equal(testOperatorNumber, len(rewardsAndProportions.OperatorRewardProportions))
				for _, operatorRewardProportion := range rewardsAndProportions.OperatorRewardProportions {
					suite.Require().Contains(suite.testOperators, sdk.MustAccAddressFromBech32(operatorRewardProportion.OperatorAddr))
					suite.Require().Equal(sdk.NewDec(1).QuoInt64(int64(testOperatorNumber)), operatorRewardProportion.RewardProportion)
				}
			},
		},
		{
			name: "pass - the dogfood AVS reward distribution should be fetched correctly.",
			malleate: func() string {
				suite.prepareTestBase(testStakerNumber, testOperatorNumber, testAVSNumber)
				// run to the end of epoch
				suite.RunToEpochEnd(dogfoodtypes.DefaultEpochIdentifier)
				suite.mintDogfoodTestReward()
				return suite.DogfoodAVSAddr
			},
			isDogfood: true,
			readOnly:  false,
			expPass:   true,
			checkState: func(rewardsAndProportions feedistributiontypes.EpochRewardsAndProportions) {
				suite.Require().GreaterOrEqual(len(rewardsAndProportions.Rewards), 1)
				suite.Require().Equal(testOperatorNumber+len(suite.Operators), len(rewardsAndProportions.OperatorRewardProportions))

				avsUSDValue, err := suite.App.OperatorKeeper.GetAVSUSDValue(suite.Ctx, suite.DogfoodAVSAddr)
				suite.Require().NoError(err)
				for _, operatorRewardProportion := range rewardsAndProportions.OperatorRewardProportions {
					operatorUSDValue, err := suite.App.OperatorKeeper.GetOperatorOptedUSDValue(suite.Ctx, suite.DogfoodAVSAddr, operatorRewardProportion.OperatorAddr)
					suite.Require().NoError(err)
					expectedProportion := operatorUSDValue.ActiveUSDValue.QuoTruncate(avsUSDValue)
					suite.Require().Equal(expectedProportion, operatorRewardProportion.RewardProportion)
				}
			},
		},
		{
			name: "pass - test the reward distribution parameter of the Dogfood AVS in the jail case",
			malleate: func() string {
				// run to the end of epoch
				suite.RunToEpochEnd(dogfoodtypes.DefaultEpochIdentifier)
				suite.mintDogfoodTestReward()

				// test case: [4,5,6,7,8,9,10] are the block numbers of the current epoch.
				// The operator is jailed at block 5 and unjailed at block 9,
				// so the jailed blocks are (5,9] = [6,7,8,9], which makes the total jailed block count 4.
				chainIDWithoutRevision := avstypes.ChainIDWithoutRevision(suite.Ctx.ChainID())
				found, consensusKey, err := suite.App.OperatorKeeper.GetOperatorConsKeyForChainID(suite.Ctx, suite.Operators[0], chainIDWithoutRevision)
				suite.NoError(err)
				suite.True(found)
				// jail the first default operator from the second block of current epoch
				suite.NextBlock()
				suite.App.OperatorKeeper.Jail(suite.Ctx, consensusKey.ToConsAddr(), chainIDWithoutRevision)
				jailedBlockNumber := 4
				for i := 0; i < jailedBlockNumber; i++ {
					suite.NextBlock()
				}
				// unjail the operator at the second-to-last block of the current epoch.
				suite.App.OperatorKeeper.Unjail(suite.Ctx, consensusKey.ToConsAddr(), chainIDWithoutRevision)
				suite.NextBlock()
				return suite.DogfoodAVSAddr
			},
			isDogfood: true,
			readOnly:  false,
			expPass:   true,
			checkState: func(rewardsAndProportions feedistributiontypes.EpochRewardsAndProportions) {
				suite.Require().GreaterOrEqual(len(rewardsAndProportions.Rewards), 1)
				suite.Require().Equal(len(suite.Operators), len(rewardsAndProportions.OperatorRewardProportions))

				avsUSDValue := sdk.NewDec(suite.TotalPower)
				operator1InitialPower := sdk.NewDec(suite.Powers[0])
				// the effective block number is 3, the total number of current epoch is 7
				operator1EffectiveRatio := sdk.NewDec(3).QuoInt64(7)
				operator2InitialPower := sdk.NewDec(suite.Powers[1])

				operator1EffectivePower := operator1InitialPower.Mul(operator1EffectiveRatio)
				decreasedPower := operator1InitialPower.Sub(operator1EffectivePower)
				avsUSDValue.SubMut(decreasedPower)

				var expectedProportion sdk.Dec
				for _, operatorRewardProportion := range rewardsAndProportions.OperatorRewardProportions {
					if operatorRewardProportion.OperatorAddr == suite.Operators[0].String() {
						expectedProportion = operator1EffectivePower.QuoTruncate(avsUSDValue)
					} else {
						expectedProportion = operator2InitialPower.QuoTruncate(avsUSDValue)
					}
					suite.Require().Equal(expectedProportion, operatorRewardProportion.RewardProportion)
				}
			},
		},
	}
	for _, tc := range testcases {
		tc := tc
		s.Run(tc.name, func() {
			s.SetupTest() // Reset state for each test case

			avsAddrStr := tc.malleate()
			isDogfood, rewardAndProportions, err := suite.App.DistrKeeper.AVSRewardAndProportionsByParam(suite.Ctx, avsAddrStr)
			s.Require().Equal(tc.isDogfood, isDogfood)

			if tc.expPass {
				s.Require().NoError(err)
			} else if tc.errContains != "" {
				s.Require().ErrorContains(err, tc.errContains)
			}

			if tc.checkState != nil {
				tc.checkState(rewardAndProportions)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestCalcJailedBlocksInEpoch() {
	type args struct {
		toggles []uint64
		start   uint64
		end     uint64
		jailed  bool
	}
	testcases := []struct {
		name        string
		args        args
		expect      uint64
		expectError bool
	}{
		{
			name: "no jail toggles",
			args: args{
				toggles: []uint64{},
				start:   100,
				end:     110,
				jailed:  false,
			},
			expect:      0,
			expectError: false,
		},
		{
			name: "invalid: odd toggles but not jailed",
			args: args{
				toggles: []uint64{105},
				start:   100,
				end:     110,
				jailed:  false,
			},
			expectError: true,
		},
		{
			name: "invalid: even toggles but still jailed",
			args: args{
				toggles: []uint64{105, 108},
				start:   100,
				end:     110,
				jailed:  true,
			},
			expectError: true,
		},
		{
			name: "toggle fully outside epoch, should be skipped",
			args: args{
				toggles: []uint64{10, 20},
				start:   100,
				end:     110,
				jailed:  false,
			},
			expect:      0,
			expectError: false,
		},
		{
			name: "one jail period fully inside epoch",
			args: args{
				toggles: []uint64{102, 105},
				start:   100,
				end:     110,
				jailed:  false,
			},
			expect:      3, // (102, 105] = [103,104,105]
			expectError: false,
		},
		{
			name: "one jail period overlaps partially",
			args: args{
				toggles: []uint64{95, 103},
				start:   100,
				end:     110,
				jailed:  false,
			},
			expect:      4, // (95, 103] => [96-103], overlap = [100,101,102,103]
			expectError: false,
		},
		{
			name: "odd number of toggles, currently jailed",
			args: args{
				toggles: []uint64{105, 108, 109},
				start:   100,
				end:     112,
				jailed:  true,
			},
			expect:      6, // [106,108] + [110,112] => 3+3
			expectError: false,
		},
		{
			name: "complex: multiple jail-unjail pairs with current jail ongoing",
			args: args{
				toggles: []uint64{90, 93, 95, 97, 100},
				start:   100,
				end:     110,
				jailed:  true,
			},
			expect:      10, // (100,110] = [101-110]
			expectError: false,
		},
		{
			name: "edge case: jail/unjailed exactly at epoch start/end",
			args: args{
				toggles: []uint64{100, 110},
				start:   100,
				end:     110,
				jailed:  false,
			},
			expect:      10, // (100,110] = [101-110]
			expectError: false,
		},
	}

	for _, tc := range testcases {
		tc := tc
		suite.Run(tc.name, func() {
			got, err := keeper.CalcJailedBlocksInEpoch(tc.args.toggles, tc.args.start, tc.args.end, tc.args.jailed)
			if tc.expectError {
				suite.Require().Error(err)
			} else {
				suite.Require().NoError(err)
				suite.Require().Equal(tc.expect, got)
			}
		})
	}
}
