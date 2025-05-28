package keeper_test

import (
	"cosmossdk.io/math"
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/imua-xyz/imuachain/testutil"
	"github.com/imua-xyz/imuachain/utils"
	assetstype "github.com/imua-xyz/imuachain/x/assets/types"
	dogfoodtypes "github.com/imua-xyz/imuachain/x/dogfood/types"
	feedistributiontypes "github.com/imua-xyz/imuachain/x/feedistribution/types"
	"strings"
)

type markChangedDelegationsArgs struct {
	stakerID       string
	assetID        string
	operator       sdk.AccAddress
	prevAssetState assetstype.OperatorAssetInfo
}

type operatorAssetState struct {
	// map key is the stakerID in the delegation key
	DelegationStartingInfos   map[string]*feedistributiontypes.DelegationStartingInfo
	HasCurrentOperatorRewards bool
	OperatorCurrentPeriod     uint64
	// map key is the period.
	OperatorHistoricalRewards map[uint64]feedistributiontypes.OperatorHistoricalRewards
}

type expectedDelegationRewardStates struct {
	EpochIdentifier string
	AvsAddr         string
	// map key is the operator and assetID
	OperatorAssetStates map[string]map[string]operatorAssetState
	// map key is the stakerID
	StakerOutstandingRewards map[string]*feedistributiontypes.StakerOutstandingRewards
}

func (suite *KeeperTestSuite) checkDelegationStates(expectedStates *expectedDelegationRewardStates) {
	// checkDelegationStates the states related to the operator asset
	for operator, assetsState := range expectedStates.OperatorAssetStates {
		for assetID, operatorAssetState := range assetsState {
			// checkDelegationStates the delegation starting info
			for stakerID, delegationStartingInfo := range operatorAssetState.DelegationStartingInfos {
				delegationKey := string(assetstype.GetJoinedStoreKey(stakerID, assetID, operator))
				actualStartingInfo, err := suite.App.DistrKeeper.GetDelegationStartingInfo(suite.Ctx, delegationKey, expectedStates.EpochIdentifier)
				if delegationStartingInfo == nil {
					suite.Require().ErrorContains(err, feedistributiontypes.ErrNoKeyInTheStore.Error(), "delegationKey:%s EpochIdentifier:%s", delegationKey, expectedStates.EpochIdentifier)
				} else {
					suite.Require().NoError(err)
					suite.Require().Equal(*delegationStartingInfo, actualStartingInfo, "delegationKey:%s EpochIdentifier:%s", delegationKey, expectedStates.EpochIdentifier)
				}
			}
			// check the operator current rewards
			operatorCurrentReward, err := suite.App.DistrKeeper.GetOperatorCurrentRewards(suite.Ctx, operator, assetID, expectedStates.EpochIdentifier)
			if !operatorAssetState.HasCurrentOperatorRewards {
				suite.Require().ErrorContains(err, feedistributiontypes.ErrNoKeyInTheStore.Error(), "operator:%s,assetID:%s, EpochIdentifier:%s", operator, assetID, expectedStates.EpochIdentifier)
			} else {
				suite.Require().NoError(err)
				suite.Require().Equal(operatorAssetState.OperatorCurrentPeriod, operatorCurrentReward.Period, "operator:%s,assetID:%s, EpochIdentifier:%s", operator, assetID, expectedStates.EpochIdentifier)
			}
			// check the operator historical rewards
			totalPeriodNumber := 0
			opFunc := func(operator, assetID, epochIdentifier string, period uint64, operatorHistoricalReward *feedistributiontypes.OperatorHistoricalRewards) (bool, error) {
				expectedHistoricalReward, ok := operatorAssetState.OperatorHistoricalRewards[period]
				// all period should exist in the expected map
				suite.Require().True(ok, "unexpected period:%d, operator:%s, assetID:%s, EpochIdentifier:%s",
					period, operator, assetID, expectedStates.EpochIdentifier)
				suite.Require().Equal(expectedHistoricalReward, *operatorHistoricalReward, "invalid operator historical reward, period:%d, operator:%s, assetID:%s, EpochIdentifier:%s",
					period, operator, assetID, expectedStates.EpochIdentifier)
				totalPeriodNumber++
				return false, nil
			}
			prefix := assetstype.GetJoinedStoreKey(operator, assetID, expectedStates.EpochIdentifier)
			err = suite.App.DistrKeeper.IterateOperatorHistoricalRewards(suite.Ctx, false, prefix, opFunc)
			suite.Require().NoError(err, "prefix for operator historical rewards:%s", string(prefix))
			// check the length to ensure that no expected periods are missing in the store.
			suite.Require().Equal(len(operatorAssetState.OperatorHistoricalRewards), totalPeriodNumber, "prefix for operator historical rewards:%s", string(prefix))
		}
	}
	// check the outstanding rewards for the staker
	for stakerID, outStandingRewards := range expectedStates.StakerOutstandingRewards {
		actualStartingInfo, err := suite.App.DistrKeeper.GetStakerOutstandingRewards(suite.Ctx, stakerID, expectedStates.AvsAddr)
		if outStandingRewards == nil {
			suite.Require().ErrorContains(err, feedistributiontypes.ErrNoKeyInTheStore.Error(), "stakerID:%s avs:%s", stakerID, expectedStates.AvsAddr)
		} else {
			suite.Require().NoError(err)
			suite.Require().Equal(*outStandingRewards, actualStartingInfo, "stakerID:%s avs:%s", stakerID, expectedStates.AvsAddr)
		}
	}
}

func (suite *KeeperTestSuite) defaultDelegationRewardStates() expectedDelegationRewardStates {
	return expectedDelegationRewardStates{
		AvsAddr:         suite.DogfoodAVSAddr,
		EpochIdentifier: dogfoodtypes.DefaultEpochIdentifier,
		OperatorAssetStates: map[string]map[string]operatorAssetState{
			suite.Operators[0].String(): {
				suite.AssetIDs[0]: {
					DelegationStartingInfos: map[string]*feedistributiontypes.DelegationStartingInfo{
						suite.StakerIDs[0]: &suite.DistributionGenesis.AllDelegationStartingInfos[0].DelegationStartingInfo,
					},
					HasCurrentOperatorRewards: true,
					OperatorCurrentPeriod:     suite.DistributionGenesis.AllOperatorCurrentRewards[0].OperatorCurrentRewards.Period,
					OperatorHistoricalRewards: map[uint64]feedistributiontypes.OperatorHistoricalRewards{
						0: suite.DistributionGenesis.AllOperatorHistoricalRewards[0].OperatorHistoricalRewards,
					},
				},
			},
			suite.Operators[1].String(): {
				suite.AssetIDs[0]: {
					DelegationStartingInfos: map[string]*feedistributiontypes.DelegationStartingInfo{
						suite.StakerIDs[1]: &suite.DistributionGenesis.AllDelegationStartingInfos[1].DelegationStartingInfo,
					},
					HasCurrentOperatorRewards: true,
					OperatorCurrentPeriod:     suite.DistributionGenesis.AllOperatorCurrentRewards[1].OperatorCurrentRewards.Period,
					OperatorHistoricalRewards: map[uint64]feedistributiontypes.OperatorHistoricalRewards{
						0: suite.DistributionGenesis.AllOperatorHistoricalRewards[1].OperatorHistoricalRewards,
					},
				},
			},
		},
		StakerOutstandingRewards: map[string]*feedistributiontypes.StakerOutstandingRewards{
			suite.StakerIDs[0]: nil,
			suite.StakerIDs[1]: nil,
		},
	}
}

func (suite *KeeperTestSuite) TestMarkChangedDelegations() {
	var defaultLzChainID uint64
	var defaultOperator sdk.AccAddress
	var defaultStakerAddr common.Address
	var defaultStakerID, defaultAssetID string
	var defaultArgs markChangedDelegationsArgs
	var defaultExpectedState map[string]*feedistributiontypes.DelegationChangeInfo
	fmt.Println("call TestMarkChangedDelegations")

	testcases := []struct {
		name     string
		malleate func() (markChangedDelegationsArgs, map[string]*feedistributiontypes.DelegationChangeInfo)
		// In some test cases, the function is already called automatically in `malleate`,
		// so it may not need to be called again in the main flow.
		shouldCallTestFunc bool
		readOnly           bool
		expPass            bool
	}{
		{
			name:               "pass - no state since no delegation or undelegation was performed.",
			shouldCallTestFunc: false,
			readOnly:           false,
			expPass:            true,
			malleate: func() (markChangedDelegationsArgs, map[string]*feedistributiontypes.DelegationChangeInfo) {
				return defaultArgs, defaultExpectedState
			},
		},
		{
			name:               "pass - mark changed delegations by delegating multiple times using the default staker during the first epoch.",
			shouldCallTestFunc: false,
			readOnly:           false,
			expPass:            true,
			malleate: func() (markChangedDelegationsArgs, map[string]*feedistributiontypes.DelegationChangeInfo) {
				// change the delegation state by another delegation
				preTotalDelegationAmount := suite.Powers[0]
				delegationTimes := 5
				for i := 0; i < delegationTimes; i++ {
					// deposit and delegate to the first operator from the default staker
					suite.DepositAndDelegateToOperators(false, defaultLzChainID, []common.Address{defaultStakerAddr},
						[]sdk.AccAddress{defaultOperator}, testutil.DefaultDelegateAmount, testutil.DefaultDelegateAmount)
				}

				return defaultArgs, map[string]*feedistributiontypes.DelegationChangeInfo{
					dogfoodtypes.DefaultEpochIdentifier: {
						StakerDelegationChanges: []feedistributiontypes.StakerDelegationChange{
							{StakerId: defaultStakerID, PreviousDelegatedAmount: sdk.NewDec(preTotalDelegationAmount)},
						},
						TotalAmount: sdk.NewDec(preTotalDelegationAmount),
					},
				}
			},
		},
		{
			name:               "pass - the changed delegations mark will be deleted at the end of the epoch.",
			shouldCallTestFunc: false,
			readOnly:           false,
			expPass:            true,
			malleate: func() (markChangedDelegationsArgs, map[string]*feedistributiontypes.DelegationChangeInfo) {
				// change the delegation state by another delegation
				// deposit and delegate to the first operator from the default staker
				suite.DepositAndDelegateToOperators(false, defaultLzChainID, []common.Address{defaultStakerAddr},
					[]sdk.AccAddress{defaultOperator}, testutil.DefaultDelegateAmount, testutil.DefaultDelegateAmount)
				// run to the end of current epoch
				suite.RunToEpochEnd(dogfoodtypes.DefaultEpochIdentifier)

				return defaultArgs, defaultExpectedState
			},
		},
		{
			name:               "pass - delegations from multiple stakers changed, all involving the same asset and operator.",
			shouldCallTestFunc: false,
			readOnly:           false,
			expPass:            true,
			malleate: func() (markChangedDelegationsArgs, map[string]*feedistributiontypes.DelegationChangeInfo) {
				// run to the end of current epoch to delete the initial states
				suite.RunToEpochEnd(dogfoodtypes.DefaultEpochIdentifier)
				preTotalDelegationAmount := testutil.DefaultDelegateAmount * int64(len(suite.testStakers))
				// deposit and delegate the test asset again, which will call the function `MarkChangedDelegations`
				suite.DepositAndDelegateToOperators(false, defaultLzChainID, suite.testStakers, suite.testOperators, testutil.DefaultDepositAmount, testutil.DefaultDelegateAmount)

				expectedStakerIDs := make([]feedistributiontypes.StakerDelegationChange, 0)
				for _, stakerAddr := range suite.testStakers {
					stakerID, _ := assetstype.GetStakerIDAndAssetIDFromStr(
						defaultLzChainID, strings.ToLower(stakerAddr.String()), "",
					)
					expectedStakerIDs = append(expectedStakerIDs, feedistributiontypes.StakerDelegationChange{
						StakerId:                stakerID,
						PreviousDelegatedAmount: sdk.NewDec(testutil.DefaultDelegateAmount),
					})
				}
				return markChangedDelegationsArgs{
						operator: suite.testOperators[0],
						assetID:  defaultAssetID,
					}, map[string]*feedistributiontypes.DelegationChangeInfo{
						dogfoodtypes.DefaultEpochIdentifier: {
							StakerDelegationChanges: expectedStakerIDs,
							TotalAmount:             sdk.NewDec(preTotalDelegationAmount),
						},
					}
			},
		},
	}

	for _, tc := range testcases {
		tc := tc
		s.Run(tc.name, func() {
			s.SetupTest() // Reset state for each test case
			s.prepareTestBase(TestStakerNumber, TestOperatorNumber, 1)

			// set the default test object
			defaultLzChainID = suite.ClientChains[0].LayerZeroChainID
			defaultOperator = suite.Operators[0]
			defaultStakerAddr = common.Address(suite.Operators[0])
			defaultStakerID, defaultAssetID = suite.StakerIDs[0], suite.AssetIDs[0]
			defaultArgs = markChangedDelegationsArgs{
				operator: defaultOperator,
				assetID:  defaultAssetID,
			}
			defaultExpectedState = map[string]*feedistributiontypes.DelegationChangeInfo{
				dogfoodtypes.DefaultEpochIdentifier: nil,
			}

			args, expectedStates := tc.malleate()
			// checkDelegationStates the state after unit test
			for epochIdentifier, expectedState := range expectedStates {
				actualState, err := suite.App.DistrKeeper.GetStakeChangedDelegations(suite.Ctx, epochIdentifier, args.operator.String(), args.assetID)
				if expectedState != nil {
					suite.Require().NoError(err)
					suite.Require().Equal(*expectedState, actualState, fmt.Sprintf("EpochIdentifier:%s,operator:%s,assetID:%s", epochIdentifier, args.operator.String(), args.assetID))
				} else {
					suite.Require().ErrorContains(err, feedistributiontypes.ErrNoKeyInTheStore.Error())
				}
			}
		})
	}
}

// return the total rewards for stakers and the reward ratio
func (suite *KeeperTestSuite) calculateExpectedOperatorReward(
	operatorPower, totalPower, totalStake int64,
	rewardPerEpoch, communityTax, commissionRate sdk.Dec,
	epochNumber int, avsAddr, rewardAssetSymbol string) (feedistributiontypes.CommonAVSRewardData, feedistributiontypes.CommonAVSRewardData) {
	totalReward := rewardPerEpoch.MulInt64(int64(epochNumber))
	proportion := math.LegacyOneDec().Sub(communityTax)
	totalRewardsExcludeCommunityTax := totalReward.MulTruncate(proportion)

	operatorTotalReward := totalRewardsExcludeCommunityTax.MulTruncate(sdk.NewDec(operatorPower).QuoTruncate(sdk.NewDec(totalPower)))
	operatorCommission := operatorTotalReward.MulTruncate(commissionRate)
	totalRewardForStakers := operatorTotalReward.Sub(operatorCommission)
	rewardRito := totalRewardForStakers.QuoTruncate(sdk.NewDec(totalStake))

	return feedistributiontypes.CommonAVSRewardData{
			AVSAddress: avsAddr,
			Rewards:    sdk.NewDecCoins(sdk.NewDecCoinFromDec(rewardAssetSymbol, totalRewardForStakers)),
		}, feedistributiontypes.CommonAVSRewardData{
			AVSAddress: avsAddr,
			Rewards:    sdk.NewDecCoins(sdk.NewDecCoinFromDec(rewardAssetSymbol, rewardRito)),
		}
}

func (suite *KeeperTestSuite) TestDistributeRewardsToDelegations() {
	testcases := []struct {
		name     string
		malleate func() expectedDelegationRewardStates
		// In some test cases, the function is already called automatically in `malleate`,
		// so it may not need to be called again in the main flow.
		shouldCallTestFunc bool
		readOnly           bool
		expPass            bool
	}{
		{
			name:               "pass - check the distribution state for genesis operator and delegation",
			shouldCallTestFunc: false,
			readOnly:           false,
			expPass:            true,
			malleate: func() expectedDelegationRewardStates {
				return suite.defaultDelegationRewardStates()
			},
		},
		{
			name:               "pass - check default distribution state for dogfood AVS at the end of the first epoch (no changes made)",
			shouldCallTestFunc: false,
			readOnly:           false,
			expPass:            true,
			malleate: func() expectedDelegationRewardStates {
				// run to the end of current epoch
				suite.RunToEpochEnd(dogfoodtypes.DefaultEpochIdentifier)
				return suite.defaultDelegationRewardStates()
			},
		},
		{
			name:               "pass - check distribution state for dogfood AVS at the end of the first epoch after adding test stakers and delegations",
			shouldCallTestFunc: false,
			readOnly:           false,
			expPass:            true,
			malleate: func() expectedDelegationRewardStates {
				// deposit and delegate the test asset to the default operators
				opNumber := 5
				for i := 0; i < opNumber; i++ {
					suite.DepositAndDelegateToOperators(false, suite.testClientChainID, suite.testStakers,
						suite.Operators, testutil.DefaultDepositAmount, testutil.DefaultDelegateAmount)
				}
				// run to the end of current epoch
				suite.RunToEpochEnd(dogfoodtypes.DefaultEpochIdentifier)

				// construct the expected distribution state
				defaultDelegationRewardState := suite.defaultDelegationRewardStates()
				for i, defaultOperator := range suite.Operators {
					operatorAssetState := defaultDelegationRewardState.OperatorAssetStates[defaultOperator.String()][suite.AssetIDs[0]]
					for _, newStaker := range suite.testStakerIDs {
						operatorAssetState.DelegationStartingInfos[newStaker] = &feedistributiontypes.DelegationStartingInfo{
							PreviousPeriod: operatorAssetState.OperatorCurrentPeriod,
							Stake:          sdk.NewDec(int64(opNumber) * testutil.DefaultDelegateAmount),
							EpochNumber:    1,
						}
					}
					// new delegations change the total delegated amount, so the operator needs to increase its period.
					operatorAssetState.OperatorCurrentPeriod++
					// current rewards doesn't reference the first historical period(0)
					firstHistoricalReward, ok := operatorAssetState.OperatorHistoricalRewards[0]
					suite.Require().True(ok)
					firstHistoricalReward.ReferenceCount--
					operatorAssetState.OperatorHistoricalRewards[0] = firstHistoricalReward

					// the period 1 will become a historical period, and it will be referenced by the current
					// reward and two new delegations.
					mintParam := suite.App.ImmintKeeper.GetParams(suite.Ctx)
					epochRewardDec := sdk.NewDecFromInt(mintParam.EpochReward)
					_, rewardRatio := suite.calculateExpectedOperatorReward(
						suite.Powers[i], suite.TotalPower, suite.Powers[i],
						epochRewardDec, feedistributiontypes.DefaultParams().CommunityTax,
						sdk.ZeroDec(), 1, suite.DogfoodAVSAddr, utils.BaseDenom,
					)
					operatorAssetState.OperatorHistoricalRewards[1] = feedistributiontypes.OperatorHistoricalRewards{
						CumulativeRewardRatios: []feedistributiontypes.CommonAVSRewardData{rewardRatio},
						ReferenceCount:         uint32(1 + len(suite.testStakers)),
					}
					defaultDelegationRewardState.OperatorAssetStates[defaultOperator.String()][suite.AssetIDs[0]] = operatorAssetState
				}
				// The new test stakers haven’t accumulated or claimed any rewards.
				for _, newStaker := range suite.testStakerIDs {
					defaultDelegationRewardState.StakerOutstandingRewards[newStaker] = nil
				}
				fmt.Println("the expect distribution state is3")
				suite.DebugPrintObject(defaultDelegationRewardState)
				return defaultDelegationRewardState
			},
		},
		{
			name:               "pass - distribute reward to the genesis delegation multiple times.",
			shouldCallTestFunc: false,
			readOnly:           false,
			expPass:            true,
			malleate: func() expectedDelegationRewardStates {
				distributionCount := 3
				distributionDuration := 5 // epoch number
				operatorIndex := 0
				testOperator := suite.Operators[operatorIndex]
				for i := 0; i < distributionCount; i++ {
					// run to the end of some epochs
					suite.RunToEpochEndN(dogfoodtypes.DefaultEpochIdentifier, distributionDuration)
					// change the default delegation amount for the operator through a new delegation.
					// it will trigger the reward distribution for this delegation at the end of epoch.
					suite.DepositAndDelegateToOperators(
						false, suite.testClientChainID, []common.Address{common.Address(testOperator)},
						[]sdk.AccAddress{testOperator}, testutil.DefaultDepositAmount, testutil.DefaultDelegateAmount)
				}
				// run to epoch again to trigger the reward distribution for the delegation
				suite.RunToEpochEnd(dogfoodtypes.DefaultEpochIdentifier)

				distributionEpochNumber := distributionDuration*distributionCount + 1

				// construct the expected distribution state
				defaultDelegationRewardState := suite.defaultDelegationRewardStates()

				operatorAssetState := defaultDelegationRewardState.OperatorAssetStates[testOperator.String()][suite.AssetIDs[0]]
				// new delegations change the total delegated amount, so the operator needs to increase its period.
				operatorAssetState.OperatorCurrentPeriod += uint64(distributionCount)
				// current rewards and the default delegation doesn't reference the first historical period(0)
				delete(operatorAssetState.OperatorHistoricalRewards, 0)

				// the starting info of the default delegation should be updated
				delegationStartingInfo, ok := operatorAssetState.DelegationStartingInfos[suite.StakerIDs[operatorIndex]]
				suite.Require().True(ok)
				delegationStartingInfo.PreviousPeriod += uint64(distributionCount)
				delegationStartingInfo.Stake.AddMut(sdk.NewDec(testutil.DefaultDelegateAmount * int64(distributionCount)))
				delegationStartingInfo.EpochNumber += uint64(distributionEpochNumber)

				// the period(0+distributionCount) will become a historical period, and it will be referenced by the current
				// reward and the updated default delegation.
				mintParam := suite.App.ImmintKeeper.GetParams(suite.Ctx)
				epochRewardDec := sdk.NewDecFromInt(mintParam.EpochReward)
				operatorPower := suite.Powers[operatorIndex]
				totalPower := suite.TotalPower
				delegationStake := operatorPower
				rewardRatio := feedistributiontypes.CommonAVSRewardData{}
				stakerReward := sdk.DecCoins{}
				var numEpochsNoPowerChange int
				for i := 0; i < distributionCount; i++ {
					if i == 0 {
						// The delegation occurs after RunToEpochEndN,
						// so the number of epochs without power change in the first duration
						// should be distributionDuration + 1.
						// For example, if the distributionDuration is 5, and the distributionCount is 3
						// the epochs without voting power change should be:
						// 1~6: initialOperatorPower(101)
						// 7~11: +DefaultDelegateAmount(201)
						// 12~16: +DefaultDelegateAmount(301)
						numEpochsNoPowerChange = distributionDuration + 1
					} else {
						numEpochsNoPowerChange = distributionDuration
					}
					// calculate the reward ratio for each duration without power change
					_, tmpRewardRatio := suite.calculateExpectedOperatorReward(
						operatorPower, totalPower, delegationStake,
						epochRewardDec, feedistributiontypes.DefaultParams().CommunityTax,
						sdk.ZeroDec(), numEpochsNoPowerChange, suite.DogfoodAVSAddr, utils.BaseDenom,
					)
					if len(rewardRatio.Rewards) == 0 {
						// handle the first duration
						rewardRatio = tmpRewardRatio
						stakerReward = tmpRewardRatio.Rewards.MulDecTruncate(sdk.NewDec(delegationStake))
					} else {
						// accumulate the reward ratio and staker reward for the other durations
						rewardRatio = rewardRatio.Add(tmpRewardRatio)
						increasedStakerReward := tmpRewardRatio.Rewards.MulDecTruncate(sdk.NewDec(delegationStake))
						stakerReward = stakerReward.Add(increasedStakerReward...)
					}
					// update the power and delegation stake
					totalPower += testutil.DefaultDelegateAmount
					operatorPower += testutil.DefaultDelegateAmount
					delegationStake += testutil.DefaultDelegateAmount
				}
				// Only one delegation refers to the operator period, and each change in the delegation
				// will increment the period. So only the period at (0 + distributionCount) needs to be stored.
				operatorAssetState.OperatorHistoricalRewards[uint64(distributionCount)] = feedistributiontypes.OperatorHistoricalRewards{
					CumulativeRewardRatios: []feedistributiontypes.CommonAVSRewardData{rewardRatio},
					ReferenceCount:         2,
				}
				defaultDelegationRewardState.OperatorAssetStates[testOperator.String()][suite.AssetIDs[0]] = operatorAssetState
				// the delegation rewards should be recorded in the staker's outstanding rewards
				defaultDelegationRewardState.StakerOutstandingRewards[suite.StakerIDs[operatorIndex]] = &feedistributiontypes.StakerOutstandingRewards{
					Rewards: stakerReward,
				}
				return defaultDelegationRewardState
			},
		},
	}
	for _, tc := range testcases {
		tc := tc
		s.Run(tc.name, func() {
			s.SetupTest() // Reset state for each test case
			s.testClientChainID = s.ClientChains[0].LayerZeroChainID
			// create test stakers
			stakerAddrs, stakerIDs := s.CreateStakers(TestStakerNumber, s.testClientChainID)
			s.testStakers = stakerAddrs
			s.testStakerIDs = stakerIDs

			expectedStates := tc.malleate()
			// checkDelegationStates the state after unit test
			suite.checkDelegationStates(&expectedStates)
		})
	}
}
