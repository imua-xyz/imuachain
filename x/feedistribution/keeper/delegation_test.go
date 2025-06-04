package keeper_test

import (
	"fmt"
	"strings"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/imua-xyz/imuachain/testutil"
	"github.com/imua-xyz/imuachain/utils"
	assetstype "github.com/imua-xyz/imuachain/x/assets/types"
	dogfoodtypes "github.com/imua-xyz/imuachain/x/dogfood/types"
	feedistributiontypes "github.com/imua-xyz/imuachain/x/feedistribution/types"
	operatortypes "github.com/imua-xyz/imuachain/x/operator/types"
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
	// the period in current rewards starts from 1.
	defaultOperatorCurrentPeriod := uint64(1)
	defaultOperatorHistoricalReward := feedistributiontypes.OperatorHistoricalRewards{
		CumulativeRewardRatios: []feedistributiontypes.CommonAVSRewardData(nil),
		// set the reference count to 2 because it will be referenced by the current reward and a default delegation.
		ReferenceCount: 2,
	}

	delegation1StartingInfo := feedistributiontypes.DelegationStartingInfo{
		// genesis delegation references the first historical reward as the starting point.
		PreviousPeriod: 0,
		Stake:          sdk.NewDec(suite.Powers[0]),
		// using 0 as the epochNumber of genesis block
		EpochNumber: uint64(operatortypes.InitialEpochNumber - 1),
	}
	delegation2StartingInfo := feedistributiontypes.DelegationStartingInfo{
		// genesis delegation references the first historical reward as the starting point.
		PreviousPeriod: 0,
		Stake:          sdk.NewDec(suite.Powers[1]),
		// using 0 as the epochNumber of genesis block
		EpochNumber: uint64(operatortypes.InitialEpochNumber - 1),
	}
	return expectedDelegationRewardStates{
		AvsAddr:         suite.DogfoodAVSAddr,
		EpochIdentifier: dogfoodtypes.DefaultEpochIdentifier,
		OperatorAssetStates: map[string]map[string]operatorAssetState{
			suite.Operators[0].String(): {
				suite.AssetIDs[0]: {
					DelegationStartingInfos: map[string]*feedistributiontypes.DelegationStartingInfo{
						suite.StakerIDs[0]: &delegation1StartingInfo,
					},
					HasCurrentOperatorRewards: true,
					OperatorCurrentPeriod:     defaultOperatorCurrentPeriod,
					OperatorHistoricalRewards: map[uint64]feedistributiontypes.OperatorHistoricalRewards{
						0: defaultOperatorHistoricalReward,
					},
				},
			},
			suite.Operators[1].String(): {
				suite.AssetIDs[0]: {
					DelegationStartingInfos: map[string]*feedistributiontypes.DelegationStartingInfo{
						suite.StakerIDs[1]: &delegation2StartingInfo,
					},
					HasCurrentOperatorRewards: true,
					OperatorCurrentPeriod:     1,
					OperatorHistoricalRewards: map[uint64]feedistributiontypes.OperatorHistoricalRewards{
						0: defaultOperatorHistoricalReward,
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
					suite.DepositAndDelegateToOperators(false, defaultLzChainID,
						common.HexToAddress(suite.Assets[0].Address), suite.Assets[0].Decimals,
						[]common.Address{defaultStakerAddr}, []sdk.AccAddress{defaultOperator},
						testutil.DefaultDelegateAmount, testutil.DefaultDelegateAmount)
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
				suite.DepositAndDelegateToOperators(false, defaultLzChainID,
					common.HexToAddress(suite.Assets[0].Address), suite.Assets[0].Decimals,
					[]common.Address{defaultStakerAddr}, []sdk.AccAddress{defaultOperator}, testutil.DefaultDelegateAmount, testutil.DefaultDelegateAmount)
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
				suite.DepositAndDelegateToOperators(false, defaultLzChainID,
					common.HexToAddress(suite.Assets[0].Address), suite.Assets[0].Decimals,
					suite.testStakers, suite.testOperators, testutil.DefaultDepositAmount, testutil.DefaultDelegateAmount)

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
	operatorAssetPower, totalPower, stakerTotalStake,
	rewardPerEpoch, communityTax, commissionRate sdk.Dec,
	epochNumber int, avsAddr, rewardAssetSymbol string,
) (feedistributiontypes.CommonAVSRewardData, feedistributiontypes.CommonAVSRewardData) {
	totalReward := rewardPerEpoch.MulInt64(int64(epochNumber))
	proportion := math.LegacyOneDec().Sub(communityTax)
	totalRewardsExcludeCommunityTax := totalReward.MulTruncate(proportion)

	operatorTotalReward := totalRewardsExcludeCommunityTax.MulTruncate(operatorAssetPower.QuoTruncate(totalPower))
	operatorCommission := operatorTotalReward.MulTruncate(commissionRate)
	totalRewardForStakers := operatorTotalReward.Sub(operatorCommission)
	rewardRito := totalRewardForStakers.QuoTruncate(stakerTotalStake)

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
				// The state of genesis operators and delegations doesn't need to be recorded in the
				// genesis file, because it will be initialized automatically before rewards are distributed.
				nullOperatorAssetState := operatorAssetState{
					DelegationStartingInfos: map[string]*feedistributiontypes.DelegationStartingInfo{
						// the starting info will be initialized when initializing operator, which
						// happens at the end of first epoch.
						suite.StakerIDs[0]: nil,
					},
					HasCurrentOperatorRewards: false,
					OperatorHistoricalRewards: map[uint64]feedistributiontypes.OperatorHistoricalRewards{},
				}
				return expectedDelegationRewardStates{
					AvsAddr:         suite.DogfoodAVSAddr,
					EpochIdentifier: dogfoodtypes.DefaultEpochIdentifier,
					OperatorAssetStates: map[string]map[string]operatorAssetState{
						suite.Operators[0].String(): {
							suite.AssetIDs[0]: nullOperatorAssetState,
						},
						suite.Operators[1].String(): {
							suite.AssetIDs[0]: nullOperatorAssetState,
						},
					},
					StakerOutstandingRewards: map[string]*feedistributiontypes.StakerOutstandingRewards{
						suite.StakerIDs[0]: nil,
						suite.StakerIDs[1]: nil,
					},
				}
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
					suite.DepositAndDelegateToOperators(false, suite.testClientChainID,
						common.HexToAddress(suite.Assets[0].Address), suite.Assets[0].Decimals,
						suite.testStakers, suite.Operators, testutil.DefaultDepositAmount, testutil.DefaultDelegateAmount)
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
						sdk.NewDec(suite.Powers[i]), sdk.NewDec(suite.TotalPower), sdk.NewDec(suite.Powers[i]),
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
						false, suite.testClientChainID,
						common.HexToAddress(suite.Assets[0].Address), suite.Assets[0].Decimals,
						[]common.Address{common.Address(testOperator)},
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
						sdk.NewDec(operatorPower), sdk.NewDec(totalPower), sdk.NewDec(delegationStake),
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
		{
			name:               "pass - add new test operators and delegations, then distribute rewards for them.",
			shouldCallTestFunc: false,
			readOnly:           false,
			expPass:            true,
			malleate: func() expectedDelegationRewardStates {
				epochDuration := 5
				// prepare the test operators and delegations
				operators := suite.RegisterOperators(1)
				suite.testOperators = operators

				// key is the assetID and stakerID
				allTestDelegations := make(map[string]map[string]*feedistributiontypes.DelegationStartingInfo)
				for _, assetID := range suite.AssetIDs {
					allTestDelegations[assetID] = make(map[string]*feedistributiontypes.DelegationStartingInfo)
				}
				allTestStakerIDs := make([]string, 0)

				// deposit and delegate the test asset before the operator has opted-in
				suite.DepositAndDelegateToOperators(true, s.testClientChainID,
					common.HexToAddress(suite.Assets[0].Address), suite.Assets[0].Decimals,
					s.testStakers, operators, testutil.DefaultDepositAmount, testutil.DefaultDelegateAmount)
				// deposit and delegate the IMUA token before the operator has opted-in
				suite.DepositAndDelegateIMUAToOperators(s.testStakers, operators, testutil.DefaultDepositAmount, testutil.DefaultDelegateAmount)
				operatorAssetPower1 := testutil.DefaultDelegateAmount * int64(len(s.testStakers))

				initialTotalPower := suite.TotalPower
				assetNumber := len(suite.AssetIDs)
				totalPowerAfterDelegations := initialTotalPower + operatorAssetPower1*int64(assetNumber)
				mintParam := suite.App.ImmintKeeper.GetParams(suite.Ctx)
				epochRewardDec := sdk.NewDecFromInt(mintParam.EpochReward)
				_, assetRewardRatio1 := suite.calculateExpectedOperatorReward(
					sdk.NewDec(operatorAssetPower1), sdk.NewDec(totalPowerAfterDelegations),
					sdk.NewDec(operatorAssetPower1), epochRewardDec,
					feedistributiontypes.DefaultParams().CommunityTax, testutil.DefaultOperatorCommission.Rate,
					epochDuration, suite.DogfoodAVSAddr, utils.BaseDenom)

				assetsDecimals := make(map[string]uint32)
				// record the test delegations
				for i, assetID := range suite.AssetIDs {
					assetsDecimals[assetID] = suite.Assets[i].Decimals
					_, clientChainID, err := assetstype.ParseID(assetID)
					suite.Require().NoError(err)
					for _, stakerAddr := range s.testStakers {
						// The IDs of stakers who delegated the IMUA token are different, because they use IMUA's chain ID.
						// So there will be four staker IDs after the test delegations.
						stakerID, _ := assetstype.GetStakerIDAndAssetID(clientChainID, stakerAddr[:], nil)
						allTestDelegations[assetID][stakerID] = &feedistributiontypes.DelegationStartingInfo{
							PreviousPeriod: 0,
							Stake:          sdk.NewDec(testutil.DefaultDelegateAmount),
							// start from the first epoch, but active from the second epoch
							EpochNumber: 1,
						}
						allTestStakerIDs = append(allTestStakerIDs, stakerID)
					}
				}
				firstBatchStakerCount := len(allTestStakerIDs)

				// opts the operators into the dogfood AVS
				suite.OptIntoDogfood(operators)

				// run some epochs to activate the operator and delegations
				suite.RunToEpochEndN(dogfoodtypes.DefaultEpochIdentifier, epochDuration)

				// Create another test staker and delegate two assets to the test operator,
				// to test the case where the delegation occurs after the operator is activated.
				testStakersNumber2 := 1
				stakerAddrs, _ := s.CreateStakers(testStakersNumber2, s.testClientChainID)
				// deposit and delegate the test asset after the operator is activated.
				suite.DepositAndDelegateToOperators(true, s.testClientChainID,
					common.HexToAddress(suite.Assets[0].Address), suite.Assets[0].Decimals,
					stakerAddrs, operators, testutil.DefaultDepositAmount, testutil.DefaultDelegateAmount)
				// deposit and delegate the IMUA token after the operator is activated.
				suite.DepositAndDelegateIMUAToOperators(stakerAddrs, operators, testutil.DefaultDepositAmount, testutil.DefaultDelegateAmount)
				operatorAssetPower2 := operatorAssetPower1 + testutil.DefaultDelegateAmount*int64(testStakersNumber2)
				totalPowerAfterDelegations += testutil.DefaultDelegateAmount * int64(testStakersNumber2) * int64(assetNumber)
				// record the new test delegations
				for _, assetID := range suite.AssetIDs {
					_, clientChainID, err := assetstype.ParseID(assetID)
					suite.Require().NoError(err)
					for _, stakerAddr := range stakerAddrs {
						// The IDs of stakers who delegated the IMUA token are different, because they use IMUA's chain ID.
						// So there will be two staker IDs after the test delegations.
						stakerID, _ := assetstype.GetStakerIDAndAssetID(clientChainID, stakerAddr[:], nil)
						allTestDelegations[assetID][stakerID] = &feedistributiontypes.DelegationStartingInfo{
							// the new delegation will increase the operator period
							PreviousPeriod: 1,
							Stake:          sdk.NewDec(testutil.DefaultDelegateAmount),
							// start from 1+epochDuration but active from 2+epochDuration
							EpochNumber: uint64(1 + epochDuration),
						}
						allTestStakerIDs = append(allTestStakerIDs, stakerID)
					}
				}

				// run some epochs to activate the new delegations
				suite.RunToEpochEndN(dogfoodtypes.DefaultEpochIdentifier, epochDuration)

				// undelegate all stakes to claim all rewards automatically
				for assetID, delegationsPerStaker := range allTestDelegations {
					for stakerID, delegationStartingInfo := range delegationsPerStaker {
						assetAddressStr, clientChainID, err := assetstype.ParseID(assetID)
						suite.Require().NoError(err)
						stakerAddressStr, _, err := assetstype.ParseID(stakerID)
						suite.Require().NoError(err)
						multiplier := math.NewIntWithDecimal(1, int(assetsDecimals[assetID])) // 10^decimals
						delegationAmountBigInt := delegationStartingInfo.Stake.MulInt(multiplier).TruncateInt()
						suite.Delegation(false, clientChainID, common.HexToAddress(stakerAddressStr), common.HexToAddress(assetAddressStr), operators[0], delegationAmountBigInt)
					}
				}
				// run to epoch end to activate the undelegations
				suite.RunToEpochEnd(dogfoodtypes.DefaultEpochIdentifier)

				// construct the expected states
				_, assetRewardRatio2 := suite.calculateExpectedOperatorReward(
					sdk.NewDec(operatorAssetPower2), sdk.NewDec(totalPowerAfterDelegations),
					sdk.NewDec(operatorAssetPower2), epochRewardDec,
					feedistributiontypes.DefaultParams().CommunityTax, testutil.DefaultOperatorCommission.Rate,
					epochDuration, suite.DogfoodAVSAddr, utils.BaseDenom)
				// The expected state for the two test assets should be the same,
				// because they have the same price and the same test delegations.
				operatorStatePerAsset := operatorAssetState{
					HasCurrentOperatorRewards: true,
					// there were three delegation changes.
					OperatorCurrentPeriod:   3,
					DelegationStartingInfos: make(map[string]*feedistributiontypes.DelegationStartingInfo),
					OperatorHistoricalRewards: map[uint64]feedistributiontypes.OperatorHistoricalRewards{
						2: {
							// it will only be referenced by the current reward
							ReferenceCount:         1,
							CumulativeRewardRatios: []feedistributiontypes.CommonAVSRewardData{assetRewardRatio2.Add(assetRewardRatio1)},
						},
					},
				}

				// The two test stakers who delegated at epoch 1 will accumulate rewards
				// over 2 * epochDuration epochs.
				stakerReward1 := assetRewardRatio2.Add(assetRewardRatio1).Rewards.MulDecTruncate(sdk.NewDec(testutil.DefaultDelegateAmount))

				// The test stakers who delegated at epoch 6 will accumulate rewards
				// over 1*epochDuration epochs.
				stakerReward2 := assetRewardRatio2.Rewards.MulDecTruncate(sdk.NewDec(testutil.DefaultDelegateAmount))

				stakerOutstandingRewards := make(map[string]*feedistributiontypes.StakerOutstandingRewards)
				for i, stakerID := range allTestStakerIDs {
					// all stake has been undelegated, so the starting info will be deleted
					operatorStatePerAsset.DelegationStartingInfos[stakerID] = nil
					if i >= firstBatchStakerCount {
						stakerOutstandingRewards[stakerID] = &feedistributiontypes.StakerOutstandingRewards{
							Rewards: stakerReward2,
						}
					} else {
						stakerOutstandingRewards[stakerID] = &feedistributiontypes.StakerOutstandingRewards{
							Rewards: stakerReward1,
						}
					}
				}

				return expectedDelegationRewardStates{
					EpochIdentifier: dogfoodtypes.DefaultEpochIdentifier,
					AvsAddr:         suite.DogfoodAVSAddr,
					OperatorAssetStates: map[string]map[string]operatorAssetState{
						operators[0].String(): {
							suite.AssetIDs[0]: operatorStatePerAsset,
							suite.AssetIDs[1]: operatorStatePerAsset,
						},
					},
					StakerOutstandingRewards: stakerOutstandingRewards,
				}
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

			// add the IMUA token into the support list of dogfood
			_, imuaAssetID := assetstype.GetStakerIDAndAssetIDFromStr(
				suite.ClientChains[1].LayerZeroChainID,
				"", suite.Assets[1].Address,
			)
			suite.AssetIDs = []string{suite.AssetIDs[0], imuaAssetID}
			suite.updateDogfoodAssetsList(suite.AssetIDs)

			expectedStates := tc.malleate()
			// checkDelegationStates the state after unit test
			suite.checkDelegationStates(&expectedStates)
		})
	}
}

func (suite *KeeperTestSuite) TestClaimDelegationRewards() {
	testcases := []struct {
		name     string
		malleate func() (string, expectedDelegationRewardStates)
		// In some test cases, the function is already called automatically in `malleate`,
		// so it may not need to be called again in the main flow.
		shouldCallTestFunc bool
		readOnly           bool
		expPass            bool
		errContains        string
	}{
		{
			name:               "pass - claim the delegation rewards actively",
			shouldCallTestFunc: true,
			readOnly:           false,
			expPass:            true,
			malleate: func() (string, expectedDelegationRewardStates) {
				testStakerID := suite.StakerIDs[0]
				// Create a new asset to test claiming rewards from multiple delegations for one staker.
				newTokenDecimal := uint32(18)
				newAssetAddrs, newAssetIDs := suite.RegisterAssets(1, newTokenDecimal)
				// add the new asset into the support list of dogfood
				assetIDs := []string{suite.AssetIDs[0], newAssetIDs[0]}
				suite.updateDogfoodAssetsList(assetIDs)

				// delegate the new asset to the test operator
				testOperator := suite.Operators[0]
				stakerAddrs := []common.Address{
					common.Address(suite.Operators[0]),
				}
				suite.DepositAndDelegateToOperators(
					false, suite.testClientChainID,
					newAssetAddrs[0], newTokenDecimal,
					stakerAddrs, []sdk.AccAddress{testOperator},
					testutil.DefaultDepositAmount, testutil.DefaultDelegateAmount)
				// run some epochs to accumulate rewards
				runEpochNumber := 5
				suite.RunToEpochEndN(dogfoodtypes.DefaultEpochIdentifier, runEpochNumber)

				// construct the expected reward state after claiming
				mintParam := suite.App.ImmintKeeper.GetParams(suite.Ctx)
				epochRewardDec := sdk.NewDecFromInt(mintParam.EpochReward)

				// Calculate the reward and ratio for operator asset1 during the first epoch.
				_, operatorAsset1RewardRatio1 := suite.calculateExpectedOperatorReward(
					sdk.NewDec(suite.Powers[0]), sdk.NewDec(suite.TotalPower),
					sdk.NewDec(suite.Powers[0]), epochRewardDec,
					feedistributiontypes.DefaultParams().CommunityTax, sdk.ZeroDec(),
					1, suite.DogfoodAVSAddr, utils.BaseDenom)
				// calculate the reward and ratio for the operator asset1 during the next four epochs.
				_, operatorAsset1RewardRatio2 := suite.calculateExpectedOperatorReward(
					sdk.NewDec(suite.Powers[0]), sdk.NewDec(suite.TotalPower+testutil.DefaultDelegateAmount),
					sdk.NewDec(suite.Powers[0]), epochRewardDec,
					feedistributiontypes.DefaultParams().CommunityTax, sdk.ZeroDec(),
					runEpochNumber-1, suite.DogfoodAVSAddr, utils.BaseDenom)
				// calculate the reward and ratio for the operator asset2 during the next four epochs.
				// this asset is delegated after the first epoch
				_, operatorAsset2RewardRatio := suite.calculateExpectedOperatorReward(
					sdk.NewDec(testutil.DefaultDelegateAmount),
					sdk.NewDec(suite.TotalPower+testutil.DefaultDelegateAmount),
					sdk.NewDec(testutil.DefaultDelegateAmount), epochRewardDec,
					feedistributiontypes.DefaultParams().CommunityTax, sdk.ZeroDec(),
					runEpochNumber-1, suite.DogfoodAVSAddr, utils.BaseDenom)

				// calculate the staker rewards from two delegations
				stakerDelegation1Rewards := operatorAsset1RewardRatio1.Add(operatorAsset1RewardRatio2).Rewards.MulDecTruncate(sdk.NewDec(suite.Powers[0]))
				stakerDelegation2Rewards := operatorAsset1RewardRatio2.Rewards.MulDecTruncate(sdk.NewDec(testutil.DefaultDelegateAmount))

				return testStakerID, expectedDelegationRewardStates{
					AvsAddr:         suite.DogfoodAVSAddr,
					EpochIdentifier: dogfoodtypes.DefaultEpochIdentifier,
					OperatorAssetStates: map[string]map[string]operatorAssetState{
						testOperator.String(): {
							assetIDs[0]: {
								DelegationStartingInfos: map[string]*feedistributiontypes.DelegationStartingInfo{
									testStakerID: {
										PreviousPeriod: 1,
										Stake:          sdk.NewDec(suite.Powers[0]),
										EpochNumber:    uint64(runEpochNumber),
									},
								},
								HasCurrentOperatorRewards: true,
								OperatorCurrentPeriod:     2,
								OperatorHistoricalRewards: map[uint64]feedistributiontypes.OperatorHistoricalRewards{
									1: {
										ReferenceCount: 2,
										CumulativeRewardRatios: []feedistributiontypes.CommonAVSRewardData{
											operatorAsset1RewardRatio1.Add(operatorAsset1RewardRatio2),
										},
									},
								},
							},
							assetIDs[1]: {
								DelegationStartingInfos: map[string]*feedistributiontypes.DelegationStartingInfo{
									testStakerID: {
										PreviousPeriod: 1,
										Stake:          sdk.NewDec(testutil.DefaultDelegateAmount),
										EpochNumber:    uint64(runEpochNumber),
									},
								},
								HasCurrentOperatorRewards: true,
								OperatorCurrentPeriod:     2,
								OperatorHistoricalRewards: map[uint64]feedistributiontypes.OperatorHistoricalRewards{
									1: {
										ReferenceCount: 2,
										CumulativeRewardRatios: []feedistributiontypes.CommonAVSRewardData{
											operatorAsset2RewardRatio,
										},
									},
								},
							},
						},
					},
					StakerOutstandingRewards: map[string]*feedistributiontypes.StakerOutstandingRewards{
						testStakerID: {
							Rewards: stakerDelegation1Rewards.Add(stakerDelegation2Rewards...),
						},
					},
				}
			},
		},
	}
	for _, tc := range testcases {
		tc := tc
		s.Run(tc.name, func() {
			s.SetupTest() // Reset state for each test case
			s.testClientChainID = s.ClientChains[0].LayerZeroChainID
			testStakerID, expectedStates := tc.malleate()
			totalClaimedRewards, err := suite.App.DistrKeeper.ClaimDelegationRewards(suite.Ctx, testStakerID)
			if tc.expPass {
				s.Require().NoError(err)
			} else if tc.errContains != "" {
				s.Require().ErrorContains(err, tc.errContains)
			}
			// checkDelegationStates the state after unit test
			suite.checkDelegationStates(&expectedStates)

			suite.Require().Equal(feedistributiontypes.CommonAVSRewards{
				{
					AVSAddress: suite.DogfoodAVSAddr,
					Rewards:    expectedStates.StakerOutstandingRewards[testStakerID].Rewards,
				},
			}, totalClaimedRewards)
		})
	}
}

func (suite *KeeperTestSuite) TestSlashedDelegationRewards() {
	type expectedRewardState struct {
		commonStates expectedDelegationRewardStates
		slashEvents  []feedistributiontypes.KeyAndOperatorSlashEvent
	}
	testcases := []struct {
		name     string
		malleate func() expectedRewardState
		// In some test cases, the function is already called automatically in `malleate`,
		// so it may not need to be called again in the main flow.
		shouldCallTestFunc bool
		readOnly           bool
		expPass            bool
		errContains        string
	}{
		{
			name:               "pass - distribute rewards for a slashed delegation.",
			shouldCallTestFunc: true,
			readOnly:           false,
			expPass:            true,
			malleate: func() expectedRewardState {
				testStakerID := suite.StakerIDs[0]
				testOperator := suite.Operators[0]
				// deposit and delegate to the first operator from the default staker
				// This is used to prevent the delegation from dropping below the minimum self-delegation
				// requirement after slashing.
				suite.DepositAndDelegateToOperators(false, suite.testClientChainID,
					common.HexToAddress(suite.Assets[0].Address), suite.Assets[0].Decimals,
					[]common.Address{common.Address(testOperator)}, []sdk.AccAddress{testOperator},
					testutil.DefaultDelegateAmount, testutil.DefaultDelegateAmount)
				operatorPower := suite.Powers[0] + testutil.DefaultDelegateAmount
				totalPower := suite.TotalPower + testutil.DefaultDelegateAmount

				suite.RunToEpochEnd(dogfoodtypes.DefaultEpochIdentifier)

				slashFactor := suite.App.SlashingKeeper.SlashFractionDowntime(suite.Ctx)

				slashType := stakingtypes.Infraction_INFRACTION_DOWNTIME
				suite.App.OperatorKeeper.SlashWithInfractionReason(suite.Ctx, suite.Operators[0], suite.Ctx.BlockHeight(), operatorPower, slashFactor, slashType)
				suite.RunToEpochEnd(dogfoodtypes.DefaultEpochIdentifier)

				// undelegate to claim the reward
				slashPower := sdk.NewDec(operatorPower).Mul(slashFactor)
				operatorPowerAfterSlash := sdk.NewDec(operatorPower).Sub(slashPower)
				multiplier := math.NewIntWithDecimal(1, int(suite.Assets[0].Decimals)) // 10^decimals
				delegationAmountBigInt := operatorPowerAfterSlash.MulInt(multiplier).TruncateInt()
				_, undelegable, err := suite.App.DelegationKeeper.GetDelegationInfoWithAmount(suite.Ctx, suite.StakerIDs[0], suite.AssetIDs[0], suite.Operators[0].String())
				suite.Require().NoError(err)
				suite.Require().Equal(delegationAmountBigInt, undelegable)

				suite.Delegation(false, suite.testClientChainID, common.Address(suite.Operators[0]), common.HexToAddress(suite.Assets[0].Address), suite.Operators[0], delegationAmountBigInt)
				suite.RunToEpochEnd(dogfoodtypes.DefaultEpochIdentifier)

				// Test case: Use the genesis delegation related to operator1 as the test subject.
				// add delegation at epoch2, slash operator1 at epoch 2, then undelegate the stake at epoch 3.
				// The reward will be claimed at the end of epoch 3.
				// The slash event will increase operator1's period, so the current period should be 4.
				// The delegation starting info will be deleted because the stake becomes zero after undelegation.
				mintParam := suite.App.ImmintKeeper.GetParams(suite.Ctx)
				epochRewardDec := sdk.NewDecFromInt(mintParam.EpochReward)
				_, rewardRatioEpoch1 := suite.calculateExpectedOperatorReward(
					sdk.NewDec(suite.Powers[0]), sdk.NewDec(suite.TotalPower),
					sdk.NewDec(suite.Powers[0]), epochRewardDec,
					feedistributiontypes.DefaultParams().CommunityTax, sdk.ZeroDec(),
					1, suite.DogfoodAVSAddr, utils.BaseDenom)
				// The reward ratio caused by the slash event should be zero, as the current rewards are empty —
				// the rewards for epoch 1 have already been distributed. The reward ratio will be non-zero if
				// there was no new delegation in epoch 1.
				rewardRatioForSlash := feedistributiontypes.CommonAVSRewardData{}
				rewardEpoch1 := rewardRatioEpoch1.Rewards.MulDecTruncate(sdk.NewDec(suite.Powers[0]))

				stakerTotalStake := operatorPowerAfterSlash
				totalPowerAfterSlash := sdk.NewDec(totalPower).Sub(slashPower).TruncateDec()
				operatorPowerAfterSlash = operatorPowerAfterSlash.TruncateDec()
				_, rewardRatioForEpoch2And3 := suite.calculateExpectedOperatorReward(
					operatorPowerAfterSlash, totalPowerAfterSlash,
					stakerTotalStake, epochRewardDec,
					feedistributiontypes.DefaultParams().CommunityTax, sdk.ZeroDec(),
					2, suite.DogfoodAVSAddr, utils.BaseDenom)
				rewardForEpoch2And3 := rewardRatioForEpoch2And3.Rewards.MulDecTruncate(stakerTotalStake)

				epochNumberHexStr := hexutil.Encode(sdk.Uint64ToBigEndian(2))
				slashEventkey := assetstype.GetJoinedStoreKey(suite.Operators[0].String(), suite.AssetIDs[0], dogfoodtypes.DefaultEpochIdentifier, epochNumberHexStr)
				commonStates := expectedDelegationRewardStates{
					AvsAddr:         suite.DogfoodAVSAddr,
					EpochIdentifier: dogfoodtypes.DefaultEpochIdentifier,
					OperatorAssetStates: map[string]map[string]operatorAssetState{
						testOperator.String(): {
							suite.AssetIDs[0]: {
								DelegationStartingInfos: map[string]*feedistributiontypes.DelegationStartingInfo{
									testStakerID: nil,
								},
								HasCurrentOperatorRewards: true,
								OperatorCurrentPeriod:     4,
								OperatorHistoricalRewards: map[uint64]feedistributiontypes.OperatorHistoricalRewards{
									2: {
										// it's only referenced by the slash event.
										ReferenceCount: 1,
										CumulativeRewardRatios: []feedistributiontypes.CommonAVSRewardData{
											rewardRatioEpoch1.Add(rewardRatioForSlash),
										},
									},
									3: {
										// it's only referenced by the current reward
										ReferenceCount: 1,
										CumulativeRewardRatios: []feedistributiontypes.CommonAVSRewardData{
											rewardRatioEpoch1.Add(rewardRatioForSlash).Add(rewardRatioForEpoch2And3),
										},
									},
								},
							},
						},
					},
					StakerOutstandingRewards: map[string]*feedistributiontypes.StakerOutstandingRewards{
						testStakerID: {
							Rewards: rewardEpoch1.Add(rewardForEpoch2And3...),
						},
					},
				}
				return expectedRewardState{
					commonStates: commonStates,
					slashEvents: []feedistributiontypes.KeyAndOperatorSlashEvent{
						{
							Key: string(slashEventkey),
							OperatorSlashEvent: feedistributiontypes.OperatorSlashEvent{
								OperatorPeriod: 2,
								Fraction:       slashFactor,
							},
						},
					},
				}
			},
		},
	}
	for _, tc := range testcases {
		tc := tc
		s.Run(tc.name, func() {
			s.SetupTest() // Reset state for each test case
			s.testClientChainID = s.ClientChains[0].LayerZeroChainID
			expectedStates := tc.malleate()

			// checkDelegationStates the state after unit test
			suite.checkDelegationStates(&expectedStates.commonStates)
			// check the slash states
			allSlashEvents, err := suite.App.DistrKeeper.GetAllOperatorSlashEvent(suite.Ctx)
			suite.Require().NoError(err)
			suite.Require().Equal(expectedStates.slashEvents, allSlashEvents)
		})
	}
}
