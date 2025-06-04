package keeper_test

import (
	"fmt"
	"strings"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/imua-xyz/imuachain/testutil"
	"github.com/imua-xyz/imuachain/utils"
	dogfoodtypes "github.com/imua-xyz/imuachain/x/dogfood/types"
	feedistributiontypes "github.com/imua-xyz/imuachain/x/feedistribution/types"
)

var (
	TestStakerNumber         = 2
	TestOperatorNumber       = 2
	TestAVSNumber            = 2
	DefaultEpochRewardAmount = int64(10000)
	RewardAssetNumberPerAVS  = 1
	DefaultRewardParameter   = feedistributiontypes.AVSRewardParam{
		CustomRewardInflation: true,
		CustomOperatorRatio:   false,
	}
)

type expectedAllocationStates struct {
	rewardAllocationTotal sdk.Dec
	communityFeePool      sdk.DecCoins
	// key is the operator address
	accumulatedCommission map[string]sdk.DecCoins
	// key is the operator address
	outstandingRewards map[string]sdk.DecCoins
	// the first key is the operator address, and the second key is the assetID
	operatorCurrentReward map[string]map[string]feedistributiontypes.OperatorCurrentRewards
}

func addDecCoin(m map[string]sdk.DecCoins, assetSymbol, operator string, amount sdk.Dec) {
	coin := sdk.NewDecCoinFromDec(assetSymbol, amount)
	if pre, ok := m[operator]; ok {
		m[operator] = pre.Add(coin)
	} else {
		m[operator] = sdk.DecCoins{coin}
	}
}

func (expectedStates *expectedAllocationStates) addAccumulatedCommission(assetSymbol, operator string, amount sdk.Dec) {
	addDecCoin(expectedStates.accumulatedCommission, assetSymbol, operator, amount)
}

func (expectedStates *expectedAllocationStates) addOutstandingRewards(assetSymbol, operator string, amount sdk.Dec) {
	addDecCoin(expectedStates.outstandingRewards, assetSymbol, operator, amount)
}

func (expectedStates *expectedAllocationStates) addOperatorCurrentReward(avsAddr, assetSymbol, operator, assetID string, amount sdk.Dec) {
	coin := sdk.NewDecCoinFromDec(assetSymbol, amount)
	if _, ok := expectedStates.operatorCurrentReward[operator]; !ok {
		expectedStates.operatorCurrentReward[operator] = make(map[string]feedistributiontypes.OperatorCurrentRewards)
	}
	rewardMap := expectedStates.operatorCurrentReward[operator]
	if pre, ok := rewardMap[assetID]; ok {
		pre.Rewards[0].Rewards = pre.Rewards[0].Rewards.Add(coin)
		rewardMap[assetID] = pre
	} else {
		rewardMap[assetID] = feedistributiontypes.OperatorCurrentRewards{
			Rewards: []feedistributiontypes.CommonAVSRewardData{
				{
					AVSAddress: avsAddr,
					Rewards:    sdk.DecCoins{coin},
				},
			},
			Period: 1,
		}
	}
	expectedStates.operatorCurrentReward[operator] = rewardMap
}

func (suite *KeeperTestSuite) prepareTestBase(stakerNumber, operatorNumber, avsNumber int) {
	suite.Require().GreaterOrEqual(stakerNumber, operatorNumber, "There should be an equal number of stakers and operators, as each pair is associated by index.")
	testClientChainID := suite.ClientChains[0].LayerZeroChainID
	// create test stakers
	stakerAddrs, stakerIDs := suite.CreateStakers(stakerNumber, testClientChainID)
	suite.testStakers = stakerAddrs
	suite.testStakerIDs = stakerIDs
	// create and register test operators
	operators := suite.RegisterOperators(operatorNumber)
	suite.testOperators = operators
	// create and register test AVSs
	// using the same epoch identifier as the dogfood AVS
	testAVSs := suite.RegisterAVSs(avsNumber, dogfoodtypes.DefaultEpochIdentifier)
	suite.testAVSs = testAVSs

	// deposit and delegate the test asset
	suite.DepositAndDelegateToOperators(true, testClientChainID,
		common.HexToAddress(suite.Assets[0].Address), suite.Assets[0].Decimals,
		stakerAddrs, operators, testutil.DefaultDepositAmount, testutil.DefaultDelegateAmount)
	// deposit and delegate the IMUA token
	suite.DepositAndDelegateIMUAToOperators(stakerAddrs, operators, testutil.DefaultDepositAmount, testutil.DefaultDelegateAmount)

	// opts the operators into the test AVSs
	suite.OptIntoAVSs(operators, testAVSs)
	// opts the operators into the dogfood AVS
	suite.OptIntoDogfood(operators)
}

func (suite *KeeperTestSuite) setAVSEpochRewards(avsList []common.Address, rewardAmountPerEpoch int64) {
	for _, avs := range avsList {
		// set the allocated reward of each epoch
		avsStr := strings.ToLower(avs.String())
		avsRewardAsset, err := suite.App.DistrKeeper.GetAllRewardAssetsByAVS(suite.Ctx, avsStr)
		suite.Require().NoError(err)

		epochRewards := make(sdk.DecCoins, 0)
		if rewardAmountPerEpoch > 0 {
			for _, rewardAsset := range avsRewardAsset.AvsRewardAssets {
				epochRewards = append(epochRewards, sdk.DecCoin{
					Denom:  rewardAsset.AssetBasicInfo.Symbol,
					Amount: sdk.NewDec(rewardAmountPerEpoch),
				})
			}
		}
		err = suite.App.DistrKeeper.SetAVSEpochRewardExclusive(suite.Ctx, avsStr, epochRewards)
		suite.Require().NoError(err)
	}
}

func (suite *KeeperTestSuite) updateDogfoodAssetsList(assetIDS []string) {
	dogfoodParam := suite.App.StakingKeeper.GetDogfoodParams(suite.Ctx)
	dogfoodParam.AssetIDs = assetIDS
	authAddr := authtypes.NewModuleAddress(govtypes.ModuleName)
	authAddrString := authAddr.String()
	_, err := suite.App.StakingKeeper.UpdateParams(suite.Ctx, &dogfoodtypes.MsgUpdateParams{
		Params:    dogfoodParam,
		Authority: authAddrString,
	})
	suite.Require().NoError(err)
}

func (suite *KeeperTestSuite) checkAllocationStates(testAVSAddr string, states expectedAllocationStates) {
	allRewardAssets, err := suite.App.DistrKeeper.GetAllRewardAssetsByAVS(suite.Ctx, testAVSAddr)
	suite.Require().NoError(err)
	suite.Require().Equal(RewardAssetNumberPerAVS, len(allRewardAssets.AvsRewardAssets))
	for _, rewardAsset := range allRewardAssets.AvsRewardAssets {
		suite.Require().Equal(states.rewardAllocationTotal, rewardAsset.RewardAssetState.RewardAllocationTotal)
	}

	// checkDelegationStates the community fee
	communityFeePool, err := suite.App.DistrKeeper.GetAVSFeePool(suite.Ctx, testAVSAddr)
	suite.Require().NoError(err)
	suite.Require().Equal(states.communityFeePool, communityFeePool.CommunityPool)

	// checkDelegationStates the accumulated commission
	for operator, expectedCommission := range states.accumulatedCommission {
		accumulatedCommission, err := suite.App.DistrKeeper.GetOperatorAccumulatedCommission(suite.Ctx, operator, testAVSAddr)
		suite.Require().NoError(err)
		suite.Require().Equal(expectedCommission, accumulatedCommission.Commission, "operator:%s,avs:%s", operator, testAVSAddr)
	}

	// checkDelegationStates the operator outstanding rewards
	for operator, expectedOutstandingRewards := range states.outstandingRewards {
		outstandingRewards, err := suite.App.DistrKeeper.GetOperatorOutstandingRewards(suite.Ctx, operator, testAVSAddr)
		suite.Require().NoError(err)
		suite.Require().Equal(expectedOutstandingRewards, outstandingRewards.Rewards)
	}

	// checkDelegationStates the current rewards for the operator after splitting into different asset pools.
	for operator, assetIDAndCurrentReward := range states.operatorCurrentReward {
		for assetID, expectedCurrentReward := range assetIDAndCurrentReward {
			operatorCurrentReward, err := suite.App.DistrKeeper.GetOperatorCurrentRewards(suite.Ctx, operator, assetID, dogfoodtypes.DefaultEpochIdentifier)
			suite.Require().NoError(err)
			suite.Require().Equal(expectedCurrentReward, operatorCurrentReward, "operator:%s, assetID:%s", operator, assetID)
		}
	}
}

func (suite *KeeperTestSuite) TestAllocateRewardsByAVS() {
	testAVSIndex := 0
	testcases := []struct {
		name              string
		malleate          func() (string, int64)
		readOnly          bool
		expPass           bool
		isDogfood         bool
		errContains       string
		getExpectedStates func(runToEpochNumber int64) *expectedAllocationStates
	}{
		{
			name: "fail - test the case where the AVS is not active in the first epoch.",
			malleate: func() (string, int64) {
				suite.registerRewardAssets(suite.testAVSs)
				suite.setAVSEpochRewards(suite.testAVSs, DefaultEpochRewardAmount)
				suite.setRewardParams(suite.testAVSs)
				return strings.ToLower(suite.testAVSs[testAVSIndex].String()), 0
			},
			readOnly:    false,
			expPass:     false,
			isDogfood:   false,
			errContains: feedistributiontypes.ErrNoKeyInTheStore.Error(),
		},
		{
			name: "pass - test the case where the AVS is active after the first epoch.",
			malleate: func() (string, int64) {
				suite.registerRewardAssets(suite.testAVSs)
				suite.setAVSEpochRewards(suite.testAVSs, DefaultEpochRewardAmount)
				suite.setRewardParams(suite.testAVSs)
				suite.RunToEpochEnd(dogfoodtypes.DefaultEpochIdentifier)
				return strings.ToLower(suite.testAVSs[testAVSIndex].String()), 1
			},
			readOnly:  false,
			expPass:   true,
			isDogfood: false,
			getExpectedStates: func(_ int64) *expectedAllocationStates {
				testAVSAddr := strings.ToLower(suite.testAVSs[testAVSIndex].String())
				assetSymbol := fmt.Sprintf("avs%dsymbol%d", testAVSIndex, 0)

				// calculate the expected state
				totalRewardDec := sdk.NewDec(DefaultEpochRewardAmount)
				proportion := math.LegacyOneDec().Sub(feedistributiontypes.DefaultParams().CommunityTax)
				rewardsExcludeCommunityTax := totalRewardDec.MulTruncate(proportion)

				operatorCommissionRate := testutil.DefaultOperatorCommission.Rate
				expectedOperatorOutstandingRewards := rewardsExcludeCommunityTax.QuoInt64(int64(TestOperatorNumber))
				expectedOperatorCommission := expectedOperatorOutstandingRewards.MulTruncate(operatorCommissionRate)
				expectedCommunityFee := totalRewardDec.Sub(rewardsExcludeCommunityTax)

				// Rewards for each staking asset are the same after splitting,
				// since we delegated equal amounts and all asset prices use default values.
				expectedRewardPerAsset := expectedOperatorOutstandingRewards.Sub(expectedOperatorCommission).QuoInt64(int64(len(suite.AssetIDs)))

				expectedStates := expectedAllocationStates{
					rewardAllocationTotal: sdk.NewDec(DefaultEpochRewardAmount),
					communityFeePool:      sdk.DecCoins{sdk.NewDecCoinFromDec(assetSymbol, expectedCommunityFee)},
					accumulatedCommission: make(map[string]sdk.DecCoins),
					outstandingRewards:    make(map[string]sdk.DecCoins),
					operatorCurrentReward: make(map[string]map[string]feedistributiontypes.OperatorCurrentRewards),
				}

				for _, operator := range suite.testOperators {
					expectedStates.accumulatedCommission[operator.String()] = sdk.DecCoins{sdk.NewDecCoinFromDec(assetSymbol, expectedOperatorCommission)}
					expectedStates.outstandingRewards[operator.String()] = sdk.DecCoins{sdk.NewDecCoinFromDec(assetSymbol, expectedOperatorOutstandingRewards)}
					// checkDelegationStates the current rewards for the operator after splitting into different asset pools.
					for _, stakingAssetID := range suite.AssetIDs {
						expectedStates.addOperatorCurrentReward(testAVSAddr, assetSymbol, operator.String(), stakingAssetID, expectedRewardPerAsset)
					}
				}
				return &expectedStates
			},
		},
		{
			name: "pass - test the reward distribution at the first epoch for the dogfood AVS",
			malleate: func() (string, int64) {
				suite.mintDogfoodTestReward()
				return suite.DogfoodAVSAddr, 0
			},
			readOnly:  false,
			expPass:   true,
			isDogfood: true,
			getExpectedStates: func(_ int64) *expectedAllocationStates {
				assetSymbol := utils.BaseDenom
				mintParam := suite.App.ImmintKeeper.GetParams(suite.Ctx)
				// calculate the expected state
				totalRewardDec := sdk.NewDecFromInt(mintParam.EpochReward)
				proportion := math.LegacyOneDec().Sub(feedistributiontypes.DefaultParams().CommunityTax)
				rewardsExcludeCommunityTax := totalRewardDec.MulTruncate(proportion)

				expectedOperator0OutstandingRewards := rewardsExcludeCommunityTax.MulTruncate(sdk.NewDec(suite.Powers[0]).QuoTruncate(sdk.NewDec(suite.TotalPower)))
				expectedOperator1OutstandingRewards := rewardsExcludeCommunityTax.MulTruncate(sdk.NewDec(suite.Powers[1]).QuoTruncate(sdk.NewDec(suite.TotalPower)))
				expectedCommunityFee := totalRewardDec.Sub(expectedOperator0OutstandingRewards).Sub(expectedOperator1OutstandingRewards)

				// Since the operator doesn't take any commission and the dogfood AVS supports only one asset,
				// the rewards for each staking asset are equal to the expected outstanding rewards.
				expectedOperator0RewardPerAsset := expectedOperator0OutstandingRewards
				expectedOperator1RewardPerAsset := expectedOperator1OutstandingRewards

				expectedStates := expectedAllocationStates{
					rewardAllocationTotal: totalRewardDec,
					communityFeePool:      sdk.DecCoins{sdk.NewDecCoinFromDec(assetSymbol, expectedCommunityFee)},
					// the commission rate is set to zero, so the operator doesn't accumulate any commission
					accumulatedCommission: nil,
					outstandingRewards: map[string]sdk.DecCoins{
						suite.Operators[0].String(): {sdk.NewDecCoinFromDec(assetSymbol, expectedOperator0OutstandingRewards)},
						suite.Operators[1].String(): {sdk.NewDecCoinFromDec(assetSymbol, expectedOperator1OutstandingRewards)},
					},
					operatorCurrentReward: map[string]map[string]feedistributiontypes.OperatorCurrentRewards{
						suite.Operators[0].String(): {
							suite.AssetIDs[0]: {
								Rewards: []feedistributiontypes.CommonAVSRewardData{
									{
										AVSAddress: suite.DogfoodAVSAddr,
										Rewards:    sdk.DecCoins{sdk.NewDecCoinFromDec(assetSymbol, expectedOperator0RewardPerAsset)},
									},
								},
								Period: 1,
							},
						},
						suite.Operators[1].String(): {
							suite.AssetIDs[0]: {
								Rewards: []feedistributiontypes.CommonAVSRewardData{
									{
										AVSAddress: suite.DogfoodAVSAddr,
										Rewards:    sdk.DecCoins{sdk.NewDecCoinFromDec(assetSymbol, expectedOperator1RewardPerAsset)},
									},
								},
								Period: 1,
							},
						},
					},
				}
				return &expectedStates
			},
		},
		{
			name: "pass - test the reward distribution for the dogfood AVS after N epochs.",
			malleate: func() (string, int64) {
				// add the IMUA token to the asset list of the dogfood AVS to test the case with two restaking assets
				// the newly added test operators who opted in at the first epoch have already deposited and delegated
				// these assets to the dogfood AVS.
				// the assetIDs in the suite have been updated when another test AVS was registered.
				suite.updateDogfoodAssetsList(suite.AssetIDs)
				n := 2
				suite.RunToEpochEndN(dogfoodtypes.DefaultEpochIdentifier, n)
				suite.mintDogfoodTestReward()
				return suite.DogfoodAVSAddr, int64(n)
			},
			readOnly:  false,
			expPass:   true,
			isDogfood: true,
			getExpectedStates: func(runToEpochNumber int64) *expectedAllocationStates {
				assetSymbol := utils.BaseDenom
				mintParam := suite.App.ImmintKeeper.GetParams(suite.Ctx)
				// calculate the expected state
				// we added 1 because we minted the reward for another epoch to test AllocateRewardsByAVS.
				epochRewardDec := sdk.NewDecFromInt(mintParam.EpochReward)
				totalRewardDec := epochRewardDec.MulInt64(runToEpochNumber + 1)
				proportion := math.LegacyOneDec().Sub(feedistributiontypes.DefaultParams().CommunityTax)
				epochRewardsExcludeCommunityTax := epochRewardDec.MulTruncate(proportion)
				expectedCommunityFee := totalRewardDec.Clone()

				expectedStates := expectedAllocationStates{
					rewardAllocationTotal: totalRewardDec,
					// the commission rates of newly added operators are not zero , so they will accumulate some commissions
					accumulatedCommission: make(map[string]sdk.DecCoins),
					outstandingRewards:    make(map[string]sdk.DecCoins),
					operatorCurrentReward: make(map[string]map[string]feedistributiontypes.OperatorCurrentRewards),
				}

				epochTotalPower := sdk.NewDec(suite.TotalPower)
				// The default price amount and decimal are 1 and 0, respectively. So we can directly use the
				// default delegation amount multiplied by 4 (for the two stakers and two assets) as the voting power.
				newOperatorPower := sdk.NewDec(testutil.DefaultDelegateAmount * 4)
				otherEpochsTotalPower := epochTotalPower.Add(newOperatorPower.MulInt64(int64(TestOperatorNumber)))

				for i := int64(0); i < runToEpochNumber+1; i++ {
					// update epochTotalPower for epochs > 0
					if i > 0 {
						epochTotalPower = otherEpochsTotalPower
					}

					// handle all operators (default + test)
					// handle the default operators
					allOperators := append([]sdk.AccAddress{}, suite.Operators...)
					// The dogfood AVS has only two default operators active in the first epoch.
					// The other two test operators will be active from the second epoch onward.
					// handle the new test operators
					if i > 0 {
						allOperators = append(allOperators, suite.testOperators...)
					}

					for idx, operator := range allOperators {
						operatorStr := operator.String()
						var power sdk.Dec
						var assetIDs []string
						var commissionRate sdk.Dec
						if idx < len(suite.Operators) {
							// default operator
							power = sdk.NewDec(suite.Powers[idx])
							assetIDs = []string{suite.AssetIDs[0]}
							commissionRate = sdk.ZeroDec()
						} else {
							// new operator
							power = newOperatorPower
							assetIDs = suite.AssetIDs
							commissionRate = testutil.DefaultOperatorCommission.Rate
						}

						// calculate outstanding rewards
						expectedEpochOutstandingRewards := epochRewardsExcludeCommunityTax.MulTruncate(power.QuoTruncate(epochTotalPower))
						expectedStates.addOutstandingRewards(assetSymbol, operatorStr, expectedEpochOutstandingRewards)

						// calculate commission
						operatorCommission := expectedEpochOutstandingRewards.MulTruncate(commissionRate)
						if commissionRate.IsPositive() {
							expectedStates.addAccumulatedCommission(assetSymbol, operatorStr, operatorCommission)
						}
						expectedCommunityFee.SubMut(operatorCommission)

						// calculate staking rewards
						stakingRewards := expectedEpochOutstandingRewards.Sub(operatorCommission)
						// For new operators, staking rewards are split equally between assets,
						// as each asset was delegated with the same USD value.
						// For default operators, only one asset was delegated, so no splitting is needed.
						rewardPerAsset := stakingRewards.QuoTruncate(sdk.NewDec(int64(len(assetIDs))))
						for _, assetID := range assetIDs {
							expectedStates.addOperatorCurrentReward(suite.DogfoodAVSAddr, assetSymbol, operatorStr, assetID, rewardPerAsset)
							expectedCommunityFee.SubMut(rewardPerAsset)
						}
					}
				}

				expectedStates.communityFeePool = sdk.DecCoins{sdk.NewDecCoinFromDec(assetSymbol, expectedCommunityFee)}
				return &expectedStates
			},
		},
	}

	for _, tc := range testcases {
		tc := tc
		s.Run(tc.name, func() {
			s.SetupTest() // Reset state for each test case
			s.prepareTestBase(TestStakerNumber, TestOperatorNumber, 1)

			testAVSAddr, runToEpochNumber := tc.malleate()
			err := suite.App.DistrKeeper.AllocateRewardsByAVS(suite.Ctx, testAVSAddr, dogfoodtypes.DefaultEpochIdentifier)
			if tc.expPass {
				s.Require().NoError(err)
			} else if tc.errContains != "" {
				s.Require().ErrorContains(err, tc.errContains)
			}

			// checkDelegationStates the state after unit test
			if tc.getExpectedStates != nil {
				expectedStates := tc.getExpectedStates(runToEpochNumber)
				s.checkAllocationStates(testAVSAddr, *expectedStates)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestAllocateRewardsByEpoch() {
	testcases := []struct {
		name        string
		malleate    func()
		readOnly    bool
		expPass     bool
		errContains string
		checkState  func()
	}{
		{
			name:     "pass - do nothing if the AVS hasn't configured the reward parameters or distribution info.",
			readOnly: false,
			expPass:  true,
			checkState: func() {
				for _, avs := range suite.testAVSs {
					allRewardAssets, err := suite.App.DistrKeeper.GetAllRewardAssetsByAVS(suite.Ctx, strings.ToLower(avs.String()))
					suite.Require().NoError(err)
					suite.Require().Equal(0, len(allRewardAssets.AvsRewardAssets))
				}
			},
		},
		{
			name: "pass - The AVS has configured the reward parameters or distribution info.",
			malleate: func() {
				suite.registerRewardAssets(suite.testAVSs)
				suite.setAVSEpochRewards(suite.testAVSs, DefaultEpochRewardAmount)
				suite.setRewardParams(suite.testAVSs)
				suite.RunToEpochEnd(dogfoodtypes.DefaultEpochIdentifier)
			},
			readOnly: false,
			expPass:  true,
			checkState: func() {
				for _, avs := range suite.testAVSs {
					allRewardAssets, err := suite.App.DistrKeeper.GetAllRewardAssetsByAVS(suite.Ctx, strings.ToLower(avs.String()))
					suite.Require().NoError(err)
					suite.Require().Equal(RewardAssetNumberPerAVS, len(allRewardAssets.AvsRewardAssets))
					for _, rewardAsset := range allRewardAssets.AvsRewardAssets {
						suite.Require().Equal(sdk.NewDec(DefaultEpochRewardAmount), rewardAsset.RewardAssetState.RewardAllocationTotal)
					}
				}
			},
		},
	}

	for _, tc := range testcases {
		tc := tc
		s.Run(tc.name, func() {
			s.SetupTest() // Reset state for each test case
			s.prepareTestBase(TestStakerNumber, TestOperatorNumber, TestAVSNumber)

			if tc.malleate != nil {
				tc.malleate()
			}
			// get current epoch info
			epochInfo, exist := suite.App.EpochsKeeper.GetEpochInfo(suite.Ctx, dogfoodtypes.DefaultEpochIdentifier)
			suite.Require().True(exist)

			err := suite.App.DistrKeeper.AllocateRewardsByEpoch(suite.Ctx, dogfoodtypes.DefaultEpochIdentifier, epochInfo.CurrentEpoch)
			if tc.expPass {
				s.Require().NoError(err)
			} else if tc.errContains != "" {
				s.Require().ErrorContains(err, tc.errContains)
			}
			// checkDelegationStates the state after unit test
			if tc.checkState != nil {
				tc.checkState()
			}
		})
	}
}
