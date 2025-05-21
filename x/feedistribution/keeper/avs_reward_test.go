package keeper_test

import (
	"cosmossdk.io/math"
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	testutiltx "github.com/imua-xyz/imuachain/testutil/tx"
	assetstype "github.com/imua-xyz/imuachain/x/assets/types"
	dogfoodtypes "github.com/imua-xyz/imuachain/x/dogfood/types"
	feedistributiontypes "github.com/imua-xyz/imuachain/x/feedistribution/types"
	"strings"
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
		suite.NoError(err)
	}
}

func (suite *KeeperTestSuite) setRewardParams(avsList []common.Address) {
	//set reward parameter for the test AVSs
	for _, avs := range avsList {
		// set reward parameter
		err := suite.App.DistrKeeper.SetAVSRewardParam(suite.Ctx, strings.ToLower(avs.String()), DefaultRewardParameter)
		suite.NoError(err)
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
	suite.NoError(err)
	err = suite.App.ImmintKeeper.AddCollectedFees(suite.Ctx, mintedCoins)
	suite.NoError(err)
}

func (suite *KeeperTestSuite) TestSetAVSRewardParam() {
	suite.prepareTestBase(1, 1, 1)
	suite.setRewardParams(suite.testAVSs)
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
				suite.NoError(err)

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
			// check the state after setting rewards
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
					suite.Equal(sdk.NewDec(1).QuoInt64(int64(testOperatorNumber)), operatorRewardProportion.RewardProportion)
				}
			},
		},
		{
			name: "pass - the dogfood AVS reward distribution should be fetched correctly.",
			malleate: func() string {
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
				suite.NoError(err)
				for _, operatorRewardProportion := range rewardsAndProportions.OperatorRewardProportions {
					operatorUSDValue, err := suite.App.OperatorKeeper.GetOperatorOptedUSDValue(suite.Ctx, suite.DogfoodAVSAddr, operatorRewardProportion.OperatorAddr)
					suite.NoError(err)
					expectedProportion := operatorUSDValue.ActiveUSDValue.QuoTruncate(avsUSDValue)
					suite.Equal(expectedProportion, operatorRewardProportion.RewardProportion)
				}
			},
		},
	}
	for _, tc := range testcases {
		tc := tc
		s.Run(tc.name, func() {
			s.SetupTest() // Reset state for each test case
			s.prepareTestBase(testStakerNumber, testOperatorNumber, testAVSNumber)

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
