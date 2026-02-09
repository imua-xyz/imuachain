package keeper_test

import (
	"fmt"
	"strings"

	delegationkeeper "github.com/imua-xyz/imuachain/x/delegation/keeper"
	delegationtype "github.com/imua-xyz/imuachain/x/delegation/types"
	imminttypes "github.com/imua-xyz/imuachain/x/immint/types"
	operatorkeeper "github.com/imua-xyz/imuachain/x/operator/keeper"

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
	StakerClaimedRewards map[string]*feedistributiontypes.StakerClaimedRewards
}

func (suite *KeeperTestSuite) checkDelegationStates(expectedStates *expectedDelegationRewardStates) {
	// check the states related to the operator asset
	for operator, assetsState := range expectedStates.OperatorAssetStates {
		for assetID, operatorAssetState := range assetsState {
			// check the delegation starting info
			for stakerID, delegationStartingInfo := range operatorAssetState.DelegationStartingInfos {
				delegationKey := string(utils.GetJoinedStoreKey(stakerID, assetID, operator))
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
			prefix := utils.GetJoinedStoreKey(operator, assetID, expectedStates.EpochIdentifier)
			err = suite.App.DistrKeeper.IterateOperatorHistoricalRewards(suite.Ctx, false, prefix, opFunc)
			suite.Require().NoError(err, "prefix for operator historical rewards:%s", string(prefix))
			// check the length to ensure that no expected periods are missing in the store.
			suite.Require().Equal(len(operatorAssetState.OperatorHistoricalRewards), totalPeriodNumber, "prefix for operator historical rewards:%s", string(prefix))
		}
	}
	// check the outstanding rewards for the staker
	for stakerID, outStandingRewards := range expectedStates.StakerClaimedRewards {
		actualStartingInfo, err := suite.App.DistrKeeper.GetStakerClaimedRewards(suite.Ctx, stakerID, expectedStates.AvsAddr)
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
		StakerClaimedRewards: map[string]*feedistributiontypes.StakerClaimedRewards{
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
			s.prepareTestBase(TestStakerNumber, TestOperatorNumber, 1, true)

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
			// check the state after unit test
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

// calcExpectedOperatorAssetReward is a test helper that mirrors the actual on-chain reward distribution logic.
//
// Note: Both `power` and `USD value` are required inputs because `power` is derived from the truncated
// `USD value`, which may lead to slight inconsistencies. The calculation strictly follows this logic:
// 1. The Operator's total reward share is calculated based on `Voting Power`.
// 2. The subsequent allocation to the specific asset pool is calculated based on `USD Value`.
// it returns the total rewards for stakers and the reward ratio
func (suite *KeeperTestSuite) calcExpectedOperatorAssetReward(
	operatorTotalPower, totalPower int64,
	operatorAssetUSDValue, operatorTotalUSDValue, stakerTotalStake,
	rewardPerEpoch, communityTax, commissionRate sdk.Dec,
	epochNumber int, avsAddr, rewardAssetSymbol string,
) (feedistributiontypes.CommonAVSRewardData, feedistributiontypes.CommonAVSRewardData) {
	totalReward := rewardPerEpoch.MulInt64(int64(epochNumber))
	proportion := math.LegacyOneDec().Sub(communityTax)
	totalRewardsExcludeCommunityTax := totalReward.MulTruncate(proportion)

	operatorRewardProportion := sdk.NewDec(operatorTotalPower).QuoTruncate(sdk.NewDec(totalPower))
	operatorTotalReward := totalRewardsExcludeCommunityTax.MulTruncate(operatorRewardProportion)
	operatorCommission := operatorTotalReward.MulTruncate(commissionRate)
	totalRewardForStakers := operatorTotalReward.Sub(operatorCommission)

	totalAssetReward := totalRewardForStakers.MulTruncate(operatorAssetUSDValue.QuoTruncate(operatorTotalUSDValue))
	rewardRito := totalAssetReward.QuoTruncate(stakerTotalStake)

	return feedistributiontypes.CommonAVSRewardData{
			AVSAddress: avsAddr,
			Rewards:    sdk.NewDecCoins(sdk.NewDecCoinFromDec(rewardAssetSymbol, totalAssetReward)),
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
					StakerClaimedRewards: map[string]*feedistributiontypes.StakerClaimedRewards{
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
					_, rewardRatio := suite.calcExpectedOperatorAssetReward(
						suite.Powers[i], suite.TotalPower,
						sdk.NewDec(suite.Powers[i]), sdk.NewDec(suite.Powers[i]), sdk.NewDec(suite.Powers[i]),
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
					defaultDelegationRewardState.StakerClaimedRewards[newStaker] = nil
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
					_, tmpRewardRatio := suite.calcExpectedOperatorAssetReward(
						operatorPower, totalPower,
						sdk.NewDec(operatorPower), sdk.NewDec(operatorPower), sdk.NewDec(delegationStake),
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
				defaultDelegationRewardState.StakerClaimedRewards[suite.StakerIDs[operatorIndex]] = &feedistributiontypes.StakerClaimedRewards{
					OutstandingRewards:     stakerReward,
					HistoricalTotalRewards: stakerReward,
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
				operators := suite.RegisterOperators(1, true)
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
				operatorTotalPower := operatorAssetPower1 * int64(assetNumber)
				totalPowerAfterDelegations := initialTotalPower + operatorTotalPower
				mintParam := suite.App.ImmintKeeper.GetParams(suite.Ctx)
				epochRewardDec := sdk.NewDecFromInt(mintParam.EpochReward)
				_, assetRewardRatio1 := suite.calcExpectedOperatorAssetReward(
					operatorTotalPower, totalPowerAfterDelegations,
					sdk.NewDec(operatorAssetPower1), sdk.NewDec(operatorTotalPower),
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
				operatorTotalPower += testutil.DefaultDelegateAmount * int64(testStakersNumber2) * int64(assetNumber)
				totalPowerAfterDelegations = initialTotalPower + operatorTotalPower
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
				_, assetRewardRatio2 := suite.calcExpectedOperatorAssetReward(
					operatorTotalPower, totalPowerAfterDelegations,
					sdk.NewDec(operatorAssetPower2), sdk.NewDec(operatorTotalPower),
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

				stakerClaimedRewards := make(map[string]*feedistributiontypes.StakerClaimedRewards)
				for i, stakerID := range allTestStakerIDs {
					// all stake has been undelegated, so the starting info will be deleted
					operatorStatePerAsset.DelegationStartingInfos[stakerID] = nil
					if i >= firstBatchStakerCount {
						stakerClaimedRewards[stakerID] = &feedistributiontypes.StakerClaimedRewards{
							OutstandingRewards:     stakerReward2,
							HistoricalTotalRewards: stakerReward2,
						}
					} else {
						stakerClaimedRewards[stakerID] = &feedistributiontypes.StakerClaimedRewards{
							OutstandingRewards:     stakerReward1,
							HistoricalTotalRewards: stakerReward1,
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
					StakerClaimedRewards: stakerClaimedRewards,
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
			// check the state after unit test
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
				_, operatorAsset1RewardRatio1 := suite.calcExpectedOperatorAssetReward(
					suite.Powers[0], suite.TotalPower,
					sdk.NewDec(suite.Powers[0]), sdk.NewDec(suite.Powers[0]),
					sdk.NewDec(suite.Powers[0]), epochRewardDec,
					feedistributiontypes.DefaultParams().CommunityTax, sdk.ZeroDec(),
					1, suite.DogfoodAVSAddr, utils.BaseDenom)
				// calculate the reward and ratio for the operator asset1 during the next four epochs.
				totalPower := suite.TotalPower + testutil.DefaultDelegateAmount
				_, operatorAsset1RewardRatio2 := suite.calcExpectedOperatorAssetReward(
					suite.Powers[0], totalPower,
					sdk.NewDec(suite.Powers[0]), sdk.NewDec(suite.Powers[0]),
					sdk.NewDec(suite.Powers[0]), epochRewardDec,
					feedistributiontypes.DefaultParams().CommunityTax, sdk.ZeroDec(),
					runEpochNumber-1, suite.DogfoodAVSAddr, utils.BaseDenom)
				// calculate the reward and ratio for the operator asset2 during the next four epochs.
				// this asset is delegated after the first epoch
				_, operatorAsset2RewardRatio := suite.calcExpectedOperatorAssetReward(
					testutil.DefaultDelegateAmount, totalPower,
					sdk.NewDec(testutil.DefaultDelegateAmount),
					sdk.NewDec(testutil.DefaultDelegateAmount),
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
					StakerClaimedRewards: map[string]*feedistributiontypes.StakerClaimedRewards{
						testStakerID: {
							OutstandingRewards:     stakerDelegation1Rewards.Add(stakerDelegation2Rewards...),
							HistoricalTotalRewards: stakerDelegation1Rewards.Add(stakerDelegation2Rewards...),
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
			// check the state after unit test
			suite.checkDelegationStates(&expectedStates)

			suite.Require().Equal(feedistributiontypes.CommonAVSRewards{
				{
					AVSAddress: suite.DogfoodAVSAddr,
					Rewards:    expectedStates.StakerClaimedRewards[testStakerID].OutstandingRewards,
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
				slashBlockHeight := suite.Ctx.BlockHeight()
				suite.App.OperatorKeeper.SlashWithInfractionReason(suite.Ctx, suite.Operators[0], slashBlockHeight, operatorPower, slashFactor, slashType)
				suite.RunToEpochEnd(dogfoodtypes.DefaultEpochIdentifier)

				// undelegate to claim the reward
				slashPower := sdk.NewDec(operatorPower).Mul(slashFactor)
				operatorPowerAfterSlash := sdk.NewDec(operatorPower).Sub(slashPower)
				multiplier := math.NewIntWithDecimal(1, int(suite.Assets[0].Decimals)) // 10^decimals
				delegationAmountBigInt := operatorPowerAfterSlash.MulInt(multiplier).TruncateInt()
				_, undelegable, _, err := suite.App.DelegationKeeper.GetDelegationInfoWithAmounts(suite.Ctx, suite.StakerIDs[0], suite.AssetIDs[0], suite.Operators[0].String())
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
				_, rewardRatioEpoch1 := suite.calcExpectedOperatorAssetReward(
					suite.Powers[0], suite.TotalPower,
					sdk.NewDec(suite.Powers[0]), sdk.NewDec(suite.Powers[0]),
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
				_, rewardRatioForEpoch2And3 := suite.calcExpectedOperatorAssetReward(
					operatorPowerAfterSlash.TruncateInt64(), totalPowerAfterSlash.TruncateInt64(),
					operatorPowerAfterSlash, operatorPowerAfterSlash,
					stakerTotalStake, epochRewardDec,
					feedistributiontypes.DefaultParams().CommunityTax, sdk.ZeroDec(),
					2, suite.DogfoodAVSAddr, utils.BaseDenom)
				rewardForEpoch2And3 := rewardRatioForEpoch2And3.Rewards.MulDecTruncate(stakerTotalStake)

				epochNumberHexStr := hexutil.Encode(sdk.Uint64ToBigEndian(2))
				blockHeightHexStr := hexutil.Encode(sdk.Uint64ToBigEndian(uint64(slashBlockHeight)))
				slashEventkey := utils.GetJoinedStoreKey(suite.Operators[0].String(), suite.AssetIDs[0], dogfoodtypes.DefaultEpochIdentifier, epochNumberHexStr, blockHeightHexStr)
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
					StakerClaimedRewards: map[string]*feedistributiontypes.StakerClaimedRewards{
						testStakerID: {
							OutstandingRewards:     rewardEpoch1.Add(rewardForEpoch2And3...),
							HistoricalTotalRewards: rewardEpoch1.Add(rewardForEpoch2And3...),
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

			// check the state after unit test
			suite.checkDelegationStates(&expectedStates.commonStates)
			// check the slash states
			allSlashEvents, err := suite.App.DistrKeeper.GetAllOperatorSlashEvent(suite.Ctx)
			suite.Require().NoError(err)
			suite.Require().Equal(expectedStates.slashEvents, allSlashEvents)
		})
	}
}

func (suite *KeeperTestSuite) TestRewardsCompounding() {
	testcases := []struct {
		name        string
		runFn       func()
		readOnly    bool
		expPass     bool
		errContains string
	}{
		{
			name:     "pass - test the voting power update with rewards compounding enabled",
			readOnly: false,
			expPass:  true,
			runFn: func() {
				operatorInitialPower := sdk.NewDec(testutil.DefaultDelegateAmount * int64(len(s.testStakers)))
				totalPower := sdk.NewDec(suite.TotalPower).Add(operatorInitialPower)
				// run one epoch to activate the operator and delegations
				// and run another epoch to earn the rewards and update the voting power at the end of epoch
				suite.RunToEpochEndN(dogfoodtypes.DefaultEpochIdentifier, 2)
				// check the unclaimed rewards for the test operator
				mintParam := suite.App.ImmintKeeper.GetParams(suite.Ctx)
				epochRewardDec := sdk.NewDecFromInt(mintParam.EpochReward)
				expectedAVSRewards, _ := suite.calcExpectedOperatorAssetReward(
					operatorInitialPower.TruncateInt64(), totalPower.TruncateInt64(),
					operatorInitialPower, operatorInitialPower,
					operatorInitialPower, epochRewardDec,
					feedistributiontypes.DefaultParams().CommunityTax, testutil.DefaultOperatorCommission.Rate,
					1, suite.DogfoodAVSAddr, utils.BaseDenom)

				// get the unclaimed rewards of test operator
				operatorUnclaimedRewards, err := suite.App.DistrKeeper.GetOperatorUnclaimedRewards(suite.Ctx, suite.testOperators[0].String(), suite.DogfoodAVSAddr)
				suite.Require().NoError(err)
				suite.Require().Equal(expectedAVSRewards.Rewards, operatorUnclaimedRewards.OutstandingRewards)

				// check the voting power for the test operator
				expectedRewardsUSDValueDec := utils.CalculateRewardUSDValue(expectedAVSRewards.Rewards[0].Amount, feedistributiontypes.IMUARewardToken.RewardAssetInfo.DenominationExponent, suite.Assets[1].Decimals, math.NewInt(1), 0)
				expectedRewardsUSDValues := []operatortypes.AVSRewardsUSDValues{
					{
						AvsAddress: suite.DogfoodAVSAddr,
						RewardsUsdValues: []operatortypes.RewardUSDValue{
							{
								Denomination: utils.BaseDenom,
								UsdValue:     expectedRewardsUSDValueDec,
							},
						},
					},
				}
				rewardsUSDValues, err := suite.App.OperatorKeeper.GetRewardsUSDValues(suite.Ctx, suite.DogfoodAVSAddr, suite.testOperators[0].String())
				suite.Require().NoError(err)
				suite.Require().Equal(expectedRewardsUSDValues, rewardsUSDValues)

				expectedVotingPower := operatorInitialPower.Add(expectedRewardsUSDValueDec)
				operatorOptedUSDValues, err := suite.App.OperatorKeeper.GetOperatorOptedUSDValue(suite.Ctx, suite.DogfoodAVSAddr, suite.testOperators[0].String())
				suite.Require().NoError(err)
				suite.Require().Equal(expectedVotingPower, operatorOptedUSDValues.ActiveUSDValue)
			},
		},
		{
			name:     "pass - test the operator unclaimed rewards generated by rewards compounding",
			readOnly: false,
			expPass:  true,
			runFn: func() {
				operatorInitialPower := sdk.NewDec(testutil.DefaultDelegateAmount * int64(len(s.testStakers)))
				totalPower := sdk.NewDec(suite.TotalPower).Add(operatorInitialPower)
				// run one epoch to activate the operator and delegations
				// and run another two epochs to generate the rewards from rewards compounding
				suite.RunToEpochEndN(dogfoodtypes.DefaultEpochIdentifier, 3)
				// check the unclaimed rewards for the test operator
				mintParam := suite.App.ImmintKeeper.GetParams(suite.Ctx)
				epochRewardDec := sdk.NewDecFromInt(mintParam.EpochReward)
				stakingRewardsEpoch1, _ := suite.calcExpectedOperatorAssetReward(
					operatorInitialPower.TruncateInt64(), totalPower.TruncateInt64(),
					operatorInitialPower, operatorInitialPower,
					operatorInitialPower, epochRewardDec,
					feedistributiontypes.DefaultParams().CommunityTax, testutil.DefaultOperatorCommission.Rate,
					1, suite.DogfoodAVSAddr, utils.BaseDenom)

				compoundingReward := stakingRewardsEpoch1.Rewards[0].Amount
				compoundingPower := utils.CalculateRewardUSDValue(compoundingReward, feedistributiontypes.IMUARewardToken.RewardAssetInfo.DenominationExponent, suite.Assets[1].Decimals, math.NewInt(1), 0)
				totalPower.AddMut(compoundingPower.TruncateDec())
				operatorCurPower := operatorInitialPower.Add(compoundingPower.TruncateDec())
				// calculate the expected rewards for staking assets and compounding rewards
				stakingRewardsEpoch2, _ := suite.calcExpectedOperatorAssetReward(
					operatorCurPower.TruncateInt64(), totalPower.TruncateInt64(),
					operatorInitialPower, operatorInitialPower.Add(compoundingPower),
					operatorCurPower, epochRewardDec,
					feedistributiontypes.DefaultParams().CommunityTax, testutil.DefaultOperatorCommission.Rate,
					1, suite.DogfoodAVSAddr, utils.BaseDenom)
				compoundingRewardsEpoch2, _ := suite.calcExpectedOperatorAssetReward(
					operatorCurPower.TruncateInt64(), totalPower.TruncateInt64(),
					compoundingPower, operatorInitialPower.Add(compoundingPower),
					operatorCurPower, epochRewardDec,
					feedistributiontypes.DefaultParams().CommunityTax, testutil.DefaultOperatorCommission.Rate,
					1, suite.DogfoodAVSAddr, utils.BaseDenom)

				// get the unclaimed rewards of test operator
				operatorUnclaimedRewards, err := suite.App.DistrKeeper.GetOperatorUnclaimedRewards(suite.Ctx, suite.testOperators[0].String(), suite.DogfoodAVSAddr)
				suite.Require().NoError(err)
				expectedOutstandingRewards := stakingRewardsEpoch1.Rewards.Add(stakingRewardsEpoch2.Rewards...)
				suite.Require().Equal(expectedOutstandingRewards, operatorUnclaimedRewards.OutstandingRewards)
				expectedCompoundingRewards := []feedistributiontypes.CompoundingRewardsPerAsset{
					{
						RewardDenomination: utils.BaseDenom,
						Rewards: feedistributiontypes.NewCommonAVSRewards(feedistributiontypes.CommonAVSRewardData{
							AVSAddress: suite.DogfoodAVSAddr,
							Rewards:    compoundingRewardsEpoch2.Rewards,
						}),
					},
				}
				suite.Require().Equal(expectedCompoundingRewards, operatorUnclaimedRewards.RewardsFromCompounding)

				// check the voting power for the test operator
				outstandingRewardsUSD := utils.CalculateRewardUSDValue(expectedOutstandingRewards[0].Amount, feedistributiontypes.IMUARewardToken.RewardAssetInfo.DenominationExponent, suite.Assets[1].Decimals, math.NewInt(1), 0)
				compoundingRewardsUSD := utils.CalculateRewardUSDValue(compoundingRewardsEpoch2.Rewards[0].Amount, feedistributiontypes.IMUARewardToken.RewardAssetInfo.DenominationExponent, suite.Assets[1].Decimals, math.NewInt(1), 0)
				expectedRewardsUSDValueDec := outstandingRewardsUSD.Add(compoundingRewardsUSD)
				expectedRewardsUSDValues := []operatortypes.AVSRewardsUSDValues{
					{
						AvsAddress: suite.DogfoodAVSAddr,
						RewardsUsdValues: []operatortypes.RewardUSDValue{
							{
								Denomination: utils.BaseDenom,
								UsdValue:     expectedRewardsUSDValueDec,
							},
						},
					},
				}
				rewardsUSDValues, err := suite.App.OperatorKeeper.GetRewardsUSDValues(suite.Ctx, suite.DogfoodAVSAddr, suite.testOperators[0].String())
				suite.Require().NoError(err)
				suite.Require().Equal(expectedRewardsUSDValues, rewardsUSDValues)

				expectedVotingPower := operatorInitialPower.Add(expectedRewardsUSDValueDec)
				operatorOptedUSDValues, err := suite.App.OperatorKeeper.GetOperatorOptedUSDValue(suite.Ctx, suite.DogfoodAVSAddr, suite.testOperators[0].String())
				suite.Require().NoError(err)
				suite.Require().Equal(expectedVotingPower, operatorOptedUSDValues.ActiveUSDValue)
			},
		},
		{
			name:     "pass - test distributing compounding rewards to changed delegation",
			readOnly: false,
			expPass:  true,
			runFn: func() {
				operatorInitialPower := sdk.NewDec(testutil.DefaultDelegateAmount * int64(len(s.testStakers)))
				totalPower := sdk.NewDec(suite.TotalPower).Add(operatorInitialPower)
				// run one epoch to activate the operator and delegations
				// and run another epoch to earn the rewards and update the voting power at the end of epoch
				suite.RunToEpochEndN(dogfoodtypes.DefaultEpochIdentifier, 2)
				// undelegate all staking asset to trigger the rewards distribution
				suite.Delegation(false, assetstype.ImuachainLzID, common.Address(suite.testOperators[0]), common.HexToAddress(assetstype.ImuachainAssetAddr), suite.testOperators[0], math.NewInt(testutil.DefaultDelegateAmount))
				// run an epoch to trigger reward distribution for this undelegation.
				suite.RunToEpochEnd(dogfoodtypes.DefaultEpochIdentifier)

				// calculate the rewards for operator unclaimed rewards
				mintParam := suite.App.ImmintKeeper.GetParams(suite.Ctx)
				epochRewardDec := sdk.NewDecFromInt(mintParam.EpochReward)
				stakingRewardsEpoch1, _ := suite.calcExpectedOperatorAssetReward(
					operatorInitialPower.TruncateInt64(), totalPower.TruncateInt64(),
					operatorInitialPower, operatorInitialPower,
					operatorInitialPower, epochRewardDec,
					feedistributiontypes.DefaultParams().CommunityTax, testutil.DefaultOperatorCommission.Rate,
					1, suite.DogfoodAVSAddr, utils.BaseDenom)

				compoundingReward := stakingRewardsEpoch1.Rewards[0].Amount
				compoundingPower := utils.CalculateRewardUSDValue(compoundingReward, feedistributiontypes.IMUARewardToken.RewardAssetInfo.DenominationExponent, suite.Assets[1].Decimals, math.NewInt(1), 0)
				totalPower.AddMut(compoundingPower.TruncateDec())
				operatorCurPower := operatorInitialPower.Add(compoundingPower.TruncateDec())

				stakingRewardsEpoch2, _ := suite.calcExpectedOperatorAssetReward(
					operatorCurPower.TruncateInt64(), totalPower.TruncateInt64(),
					operatorInitialPower, operatorInitialPower.Add(compoundingPower),
					operatorCurPower, epochRewardDec,
					feedistributiontypes.DefaultParams().CommunityTax, testutil.DefaultOperatorCommission.Rate,
					1, suite.DogfoodAVSAddr, utils.BaseDenom)
				compoundingRewardsEpoch2, _ := suite.calcExpectedOperatorAssetReward(
					operatorCurPower.TruncateInt64(), totalPower.TruncateInt64(),
					compoundingPower, operatorInitialPower.Add(compoundingPower),
					operatorCurPower, epochRewardDec,
					feedistributiontypes.DefaultParams().CommunityTax, testutil.DefaultOperatorCommission.Rate,
					1, suite.DogfoodAVSAddr, utils.BaseDenom)
				totalStakingRewards := stakingRewardsEpoch1.Rewards.Add(stakingRewardsEpoch2.Rewards...)

				// calculate the rewards of the test staker using the reward ratio to avoid
				// inaccurate values caused by truncation.
				stakingRewardRatio := totalStakingRewards.QuoDecTruncate(math.LegacyNewDec(testutil.DefaultDelegateAmount))
				stakerStakingRewards := stakingRewardRatio.MulDec(math.LegacyNewDec(testutil.DefaultDelegateAmount))
				stakerCompoundingRewards := compoundingRewardsEpoch2.Rewards.MulDecTruncate(stakerStakingRewards[0].Amount.QuoTruncate(totalStakingRewards[0].Amount))
				expectedStakerTotalRewards := stakerStakingRewards.Add(stakerCompoundingRewards...)

				// check the claimed rewards for the staker
				stakerID, _ := assetstype.GetStakerIDAndAssetID(assetstype.ImuachainLzID, suite.testOperators[0], nil)
				stakerRewards, err := suite.App.DistrKeeper.GetStakerClaimedRewards(suite.Ctx, stakerID, suite.DogfoodAVSAddr)
				suite.Require().NoError(err)
				suite.Require().Equal(expectedStakerTotalRewards, stakerRewards.OutstandingRewards)

				// get the unclaimed rewards of test operator
				operatorUnclaimedRewards, err := suite.App.DistrKeeper.GetOperatorUnclaimedRewards(suite.Ctx, suite.testOperators[0].String(), suite.DogfoodAVSAddr)
				suite.Require().NoError(err)
				suite.Require().Equal(totalStakingRewards.Sub(stakerStakingRewards), operatorUnclaimedRewards.OutstandingRewards)
				expectedCompoundingRewards := []feedistributiontypes.CompoundingRewardsPerAsset{
					{
						RewardDenomination: utils.BaseDenom,
						Rewards: feedistributiontypes.NewCommonAVSRewards(feedistributiontypes.CommonAVSRewardData{
							AVSAddress: suite.DogfoodAVSAddr,
							Rewards:    compoundingRewardsEpoch2.Rewards.Sub(stakerCompoundingRewards),
						}),
					},
				}
				suite.Require().Equal(expectedCompoundingRewards, operatorUnclaimedRewards.RewardsFromCompounding)

				// check the voting power for the test operator
				operatorOptedUSDValues, err := suite.App.OperatorKeeper.GetOperatorOptedUSDValue(suite.Ctx, suite.DogfoodAVSAddr, suite.testOperators[0].String())
				suite.Require().NoError(err)
				suite.Require().Equal(math.LegacyZeroDec(), operatorOptedUSDValues.ActiveUSDValue)
			},
		},
		{
			name:     "pass - test automatic redelegation of claimed rewards when delegation changes.",
			readOnly: false,
			expPass:  true,
			runFn: func() {
				operatorInitialPower := sdk.NewDec(testutil.DefaultDelegateAmount * int64(len(s.testStakers)))
				totalPower := sdk.NewDec(suite.TotalPower).Add(operatorInitialPower)
				// run one epoch to activate the operator and delegations
				// and run another epoch to earn the rewards and update the voting power at the end of epoch
				suite.RunToEpochEndN(dogfoodtypes.DefaultEpochIdentifier, 2)
				// enable the automatic redelegation for the test staker
				stakerID, _ := assetstype.GetStakerIDAndAssetID(assetstype.ImuachainLzID, suite.testOperators[0], nil)
				err := suite.App.DistrKeeper.SetStakerRewardParams(suite.Ctx, stakerID, feedistributiontypes.StakerRewardParams{
					RedelegateReward:       true,
					RedelegateOperatorAddr: suite.testOperators[0].String(),
				})
				suite.Require().NoError(err)

				// undelegate all staking asset to trigger the rewards distribution
				multiplier := math.NewIntWithDecimal(1, int(suite.Assets[1].Decimals)) // 10^decimals
				delegationAmountInt := sdk.NewInt(testutil.DefaultDelegateAmount).Mul(multiplier)
				suite.Delegation(false, assetstype.ImuachainLzID, common.Address(suite.testOperators[0]), common.HexToAddress(assetstype.ImuachainAssetAddr), suite.testOperators[0], delegationAmountInt)
				// run an epoch to trigger reward distribution for this undelegation.
				suite.RunToEpochEnd(dogfoodtypes.DefaultEpochIdentifier)

				// calculate the rewards for operator unclaimed rewards
				mintParam := suite.App.ImmintKeeper.GetParams(suite.Ctx)
				epochRewardDec := sdk.NewDecFromInt(mintParam.EpochReward)
				stakingRewardsEpoch1, _ := suite.calcExpectedOperatorAssetReward(
					operatorInitialPower.TruncateInt64(), totalPower.TruncateInt64(),
					operatorInitialPower, operatorInitialPower,
					operatorInitialPower, epochRewardDec,
					feedistributiontypes.DefaultParams().CommunityTax, testutil.DefaultOperatorCommission.Rate,
					1, suite.DogfoodAVSAddr, utils.BaseDenom)

				compoundingReward := stakingRewardsEpoch1.Rewards[0].Amount
				compoundingPower := utils.CalculateRewardUSDValue(compoundingReward, feedistributiontypes.IMUARewardToken.RewardAssetInfo.DenominationExponent, suite.Assets[1].Decimals, math.NewInt(1), 0)
				totalPower.AddMut(compoundingPower.TruncateDec())
				operatorCurPower := operatorInitialPower.Add(compoundingPower.TruncateDec())

				stakingRewardsEpoch2, _ := suite.calcExpectedOperatorAssetReward(
					operatorCurPower.TruncateInt64(), totalPower.TruncateInt64(),
					operatorInitialPower, operatorInitialPower.Add(compoundingPower),
					operatorCurPower, epochRewardDec,
					feedistributiontypes.DefaultParams().CommunityTax, testutil.DefaultOperatorCommission.Rate,
					1, suite.DogfoodAVSAddr, utils.BaseDenom)
				compoundingRewardsEpoch2, _ := suite.calcExpectedOperatorAssetReward(
					operatorCurPower.TruncateInt64(), totalPower.TruncateInt64(),
					compoundingPower, operatorInitialPower.Add(compoundingPower),
					operatorCurPower, epochRewardDec,
					feedistributiontypes.DefaultParams().CommunityTax, testutil.DefaultOperatorCommission.Rate,
					1, suite.DogfoodAVSAddr, utils.BaseDenom)
				totalStakingRewards := stakingRewardsEpoch1.Rewards.Add(stakingRewardsEpoch2.Rewards...)

				// calculate the rewards of the test staker using the reward ratio to avoid
				// inaccurate values caused by truncation.
				stakingRewardRatio := totalStakingRewards.QuoDecTruncate(math.LegacyNewDec(testutil.DefaultDelegateAmount))
				stakerStakingRewards := stakingRewardRatio.MulDec(math.LegacyNewDec(testutil.DefaultDelegateAmount))
				stakerCompoundingRewards := compoundingRewardsEpoch2.Rewards.MulDecTruncate(stakerStakingRewards[0].Amount.QuoTruncate(totalStakingRewards[0].Amount))
				expectedStakerTotalRewards := stakerStakingRewards.Add(stakerCompoundingRewards...)

				// check the claimed rewards for the staker
				stakerRewards, err := suite.App.DistrKeeper.GetStakerClaimedRewards(suite.Ctx, stakerID, suite.DogfoodAVSAddr)
				suite.Require().NoError(err)
				suite.Require().Equal(expectedStakerTotalRewards, stakerRewards.HistoricalTotalRewards)
				suite.Require().True(stakerRewards.OutstandingRewards.IsZero())
				redelegateAmount := feedistributiontypes.UnscaleDecToInt(expectedStakerTotalRewards[0].Amount, feedistributiontypes.IMUARewardToken.RewardAssetInfo.DenominationExponent)
				suite.Require().Equal([]feedistributiontypes.RewardsDelegationShare{
					{
						OperatorAddr: suite.testOperators[0].String(),
						Shares:       sdk.NewDecCoins(sdk.NewDecCoin(utils.BaseDenom, redelegateAmount)),
					},
				}, stakerRewards.DelegationRewardsShares)

				// check the reward delegation state
				delegation, err := suite.App.DelegationKeeper.GetSingleDelegationInfo(suite.Ctx, stakerID, assetstype.ImuachainAssetID, suite.testOperators[0].String())
				suite.Require().NoError(err)
				suite.Require().Equal(delegationtype.DelegationAmounts{
					UndelegatableShare:              math.LegacyZeroDec(),
					PendingUndelegationAmount:       delegationAmountInt,
					RewardUndelegatableShare:        math.LegacyNewDecFromBigInt(redelegateAmount.BigInt()),
					RewardPendingUndelegationAmount: sdk.ZeroInt(),
				}, *delegation)

				// check the operator asset state
				operatorAssetStates, err := suite.App.AssetsKeeper.GetOperatorSpecifiedAssetInfo(suite.Ctx, suite.testOperators[0], assetstype.ImuachainAssetID)
				suite.Require().NoError(err)
				suite.Require().NoError(err)
				suite.Require().Equal(assetstype.OperatorAssetInfo{
					TotalAmount:               redelegateAmount,
					PendingUndelegationAmount: delegationAmountInt,
					TotalShare:                math.LegacyNewDecFromInt(redelegateAmount),
					OperatorShare:             math.LegacyNewDecFromInt(redelegateAmount),
				}, *operatorAssetStates)

				// check the voting power for the test operator
				operatorOptedUSDValues, err := suite.App.OperatorKeeper.GetOperatorOptedUSDValue(suite.Ctx, suite.DogfoodAVSAddr, suite.testOperators[0].String())
				suite.Require().NoError(err)
				expectedUSDValue := utils.CalculateUSDValue(redelegateAmount, math.NewInt(1), suite.Assets[1].Decimals, 0)
				suite.Require().Equal(expectedUSDValue, operatorOptedUSDValues.SelfUSDValue)
			},
		},
		{
			name:     "pass - test undelegation of automatically redelegated rewards.",
			readOnly: false,
			expPass:  true,
			runFn: func() {
				// run one epoch to activate the operator and delegations
				// and run another epoch to earn the rewards and update the voting power at the end of epoch
				suite.RunToEpochEndN(dogfoodtypes.DefaultEpochIdentifier, 2)
				// enable the automatic redelegation for the test staker
				stakerID, _ := assetstype.GetStakerIDAndAssetID(assetstype.ImuachainLzID, suite.testOperators[0], nil)
				err := suite.App.DistrKeeper.SetStakerRewardParams(suite.Ctx, stakerID, feedistributiontypes.StakerRewardParams{
					RedelegateReward:       true,
					RedelegateOperatorAddr: suite.testOperators[0].String(),
				})
				suite.Require().NoError(err)

				// undelegate all staking asset to trigger the rewards distribution
				multiplier := math.NewIntWithDecimal(1, int(suite.Assets[1].Decimals)) // 10^decimals
				delegationAmountBigInt := sdk.NewInt(testutil.DefaultDelegateAmount).Mul(multiplier)
				suite.Delegation(false, assetstype.ImuachainLzID, common.Address(suite.testOperators[0]), common.HexToAddress(assetstype.ImuachainAssetAddr), suite.testOperators[0], delegationAmountBigInt)
				// run an epoch to trigger reward distribution for this undelegation.
				suite.RunToEpochEnd(dogfoodtypes.DefaultEpochIdentifier)

				// undelegate rewards
				delegation, err := suite.App.DelegationKeeper.GetSingleDelegationInfo(suite.Ctx, stakerID, assetstype.ImuachainAssetID, suite.testOperators[0].String())
				suite.Require().NoError(err)
				operatorAssetStates, err := suite.App.AssetsKeeper.GetOperatorSpecifiedAssetInfo(suite.Ctx, suite.testOperators[0], assetstype.ImuachainAssetID)
				suite.Require().NoError(err)
				suite.Require().Equal(delegationAmountBigInt, operatorAssetStates.PendingUndelegationAmount)

				undelegateAmount, err := delegationkeeper.TokensFromShares(delegation.RewardUndelegatableShare, operatorAssetStates.TotalShare, operatorAssetStates.TotalAmount)
				suite.Require().NoError(err)
				err = suite.App.DistrKeeper.UndelegateClaimedRewards(suite.Ctx, stakerID, assetstype.ImuachainAssetID, suite.testOperators[0], false, undelegateAmount)
				suite.Require().NoError(err)

				// check the undelegation states
				delegation, err = suite.App.DelegationKeeper.GetSingleDelegationInfo(suite.Ctx, stakerID, assetstype.ImuachainAssetID, suite.testOperators[0].String())
				suite.Require().NoError(err)
				suite.Require().Equal(delegationtype.DelegationAmounts{
					UndelegatableShare:              math.LegacyZeroDec(),
					PendingUndelegationAmount:       delegationAmountBigInt,
					RewardUndelegatableShare:        math.LegacyZeroDec(),
					RewardPendingUndelegationAmount: undelegateAmount,
				}, *delegation)
				stakerRewards, err := suite.App.DistrKeeper.GetStakerClaimedRewards(suite.Ctx, stakerID, suite.DogfoodAVSAddr)
				suite.Require().NoError(err)
				suite.Require().True(feedistributiontypes.RewardsDelegationShares(stakerRewards.DelegationRewardsShares).IsZeroShares())
				suite.Require().Equal(sdk.NewDecCoins(sdk.NewDecCoin(utils.BaseDenom, undelegateAmount)), stakerRewards.PendingUndelegationRewards)

				operatorAssetStates, err = suite.App.AssetsKeeper.GetOperatorSpecifiedAssetInfo(suite.Ctx, suite.testOperators[0], assetstype.ImuachainAssetID)
				suite.Require().NoError(err)
				suite.Require().Equal(delegationAmountBigInt.Add(undelegateAmount), operatorAssetStates.PendingUndelegationAmount)

				undelegationRecordKey, err := suite.App.DelegationKeeper.GetUndelegationRecKey(suite.Ctx, stakerID, assetstype.ImuachainAssetID, 1)
				undelegationRecords, err := suite.App.DelegationKeeper.GetUndelegationRecords(suite.Ctx, [][]byte{undelegationRecordKey})
				suite.Require().True(undelegationRecords[0].Undelegation.RewardAsset)
				suite.Require().Equal(undelegateAmount, undelegationRecords[0].Undelegation.Amount)
				suite.Require().Equal(undelegateAmount, undelegationRecords[0].Undelegation.ActualCompletedAmount)
				suite.Require().Equal([]delegationtype.UndelegationAmountPerAVS{
					{
						AvsAddress:            suite.DogfoodAVSAddr,
						Amount:                undelegateAmount,
						ActualCompletedAmount: undelegateAmount,
					},
				}, undelegationRecords[0].Undelegation.RewardUndelegations)

				// run up to the completed epoch
				currentEpochInfo, found := suite.App.EpochsKeeper.GetEpochInfo(suite.Ctx, undelegationRecords[0].Undelegation.CompletedEpochIdentifier)
				suite.Require().True(found)
				runEpochNumber := undelegationRecords[0].Undelegation.CompletedEpochNumber - currentEpochInfo.CurrentEpoch + 1
				suite.RunToEpochEndN(undelegationRecords[0].Undelegation.CompletedEpochIdentifier, int(runEpochNumber))

				// check whether the undelegation has been completed
				undelegationRecords, err = suite.App.DelegationKeeper.GetUndelegationRecords(suite.Ctx, [][]byte{undelegationRecordKey})
				suite.Require().ErrorContains(err, delegationtype.ErrNoKeyInTheStore.Error())

				delegation, err = suite.App.DelegationKeeper.GetSingleDelegationInfo(suite.Ctx, stakerID, assetstype.ImuachainAssetID, suite.testOperators[0].String())
				suite.Require().NoError(err)
				suite.Require().Equal(delegationtype.DelegationAmounts{
					UndelegatableShare:              math.LegacyZeroDec(),
					PendingUndelegationAmount:       math.ZeroInt(),
					RewardUndelegatableShare:        math.LegacyZeroDec(),
					RewardPendingUndelegationAmount: math.ZeroInt(),
				}, *delegation)
				stakerRewards, err = suite.App.DistrKeeper.GetStakerClaimedRewards(suite.Ctx, stakerID, suite.DogfoodAVSAddr)
				suite.Require().NoError(err)
				suite.Require().True(stakerRewards.PendingUndelegationRewards.IsZero())
				suite.Require().Equal(sdk.NewDecCoins(sdk.NewDecCoin(utils.BaseDenom, undelegateAmount)), stakerRewards.WithdrawableRewards)

				operatorAssetStates, err = suite.App.AssetsKeeper.GetOperatorSpecifiedAssetInfo(suite.Ctx, suite.testOperators[0], assetstype.ImuachainAssetID)
				suite.Require().NoError(err)
				suite.Require().Equal(assetstype.OperatorAssetInfo{
					TotalAmount:               math.ZeroInt(),
					PendingUndelegationAmount: math.ZeroInt(),
					TotalShare:                math.LegacyZeroDec(),
					OperatorShare:             math.LegacyZeroDec(),
				}, *operatorAssetStates)
			},
		},
		{
			name:     "pass - test slashing from compounding rewards",
			readOnly: false,
			expPass:  true,
			runFn: func() {
				operatorInitialPower := sdk.NewDec(testutil.DefaultDelegateAmount * int64(len(s.testStakers)))
				totalPower := sdk.NewDec(suite.TotalPower).Add(operatorInitialPower)
				// run one epoch to activate the operator and delegations
				// and run another two epochs to generate the rewards from rewards compounding
				suite.RunToEpochEndN(dogfoodtypes.DefaultEpochIdentifier, 3)

				// calculate the rewards
				mintParam := suite.App.ImmintKeeper.GetParams(suite.Ctx)
				epochRewardDec := sdk.NewDecFromInt(mintParam.EpochReward)
				stakingRewardsEpoch1, _ := suite.calcExpectedOperatorAssetReward(
					operatorInitialPower.TruncateInt64(), totalPower.TruncateInt64(),
					operatorInitialPower, operatorInitialPower,
					operatorInitialPower, epochRewardDec,
					feedistributiontypes.DefaultParams().CommunityTax, testutil.DefaultOperatorCommission.Rate,
					1, suite.DogfoodAVSAddr, utils.BaseDenom)

				compoundingReward := stakingRewardsEpoch1.Rewards[0].Amount
				compoundingPower := utils.CalculateRewardUSDValue(compoundingReward, feedistributiontypes.IMUARewardToken.RewardAssetInfo.DenominationExponent, suite.Assets[1].Decimals, math.NewInt(1), 0)
				totalPower.AddMut(compoundingPower.TruncateDec())
				operatorPowerDurEpoch3 := operatorInitialPower.Add(compoundingPower.TruncateDec())

				operatorUnclaimedRewards, err := suite.App.DistrKeeper.GetOperatorUnclaimedRewards(suite.Ctx, suite.testOperators[0].String(), suite.DogfoodAVSAddr)
				suite.Require().NoError(err)
				outstandingRewardAmount := operatorUnclaimedRewards.OutstandingRewards.AmountOf(utils.BaseDenom)
				outstandingRewardsUSDValue := utils.CalculateRewardUSDValue(outstandingRewardAmount, feedistributiontypes.IMUARewardToken.RewardAssetInfo.DenominationExponent, suite.Assets[1].Decimals, math.NewInt(1), 0)
				compoundingRewardAmount := operatorUnclaimedRewards.RewardsFromCompounding[0].Rewards[0].Rewards.AmountOf(utils.BaseDenom)
				compoundingRewardsUSDValue := utils.CalculateRewardUSDValue(compoundingRewardAmount, feedistributiontypes.IMUARewardToken.RewardAssetInfo.DenominationExponent, suite.Assets[1].Decimals, math.NewInt(1), 0)
				operatorTotalUSDValue := outstandingRewardsUSDValue.Add(compoundingRewardsUSDValue).Add(operatorInitialPower)
				// slash the operator at the end of epoch 3
				slashFactor := suite.App.SlashingKeeper.SlashFractionDowntime(suite.Ctx)
				slashType := stakingtypes.Infraction_INFRACTION_DOWNTIME
				infractionHeight := suite.Ctx.BlockHeight()
				infractionPower := operatorPowerDurEpoch3.TruncateInt64()
				suite.App.OperatorKeeper.SlashWithInfractionReason(suite.Ctx, suite.testOperators[0], infractionHeight, infractionPower, slashFactor, slashType)

				// check the states after slashing from compounding rewards.
				totalSlashedAmount := math.LegacyNewDec(infractionPower).Mul(slashFactor)
				actualSlashProportion := totalSlashedAmount.Quo(operatorTotalUSDValue)
				delegationAmountBigInt := feedistributiontypes.UnscaleDecToInt(math.LegacyNewDec(testutil.DefaultDelegateAmount), suite.Assets[1].Decimals)
				slashedAmountFromAssetPool := actualSlashProportion.MulInt(delegationAmountBigInt).TruncateInt()
				slashedAmountDecFromOutStandingReward := actualSlashProportion.Mul(outstandingRewardAmount)
				slashedAmountDecFromCompoundingReward := actualSlashProportion.Mul(compoundingRewardAmount)
				slashID := operatorkeeper.GetSlashIDForDogfood(slashType, infractionHeight)
				slashRecord, err := suite.App.OperatorKeeper.GetOperatorSlashInfo(suite.Ctx, suite.DogfoodAVSAddr, suite.testOperators[0].String(), slashID)
				suite.Require().NoError(err)
				expectedSlashRecord := operatortypes.OperatorSlashInfo{
					SubmittedHeight: infractionHeight,
					EventHeight:     infractionHeight,
					SlashProportion: slashFactor,
					SlashType:       uint32(slashType),
					ExecutionInfo: &operatortypes.SlashExecutionInfo{
						SlashProportion: actualSlashProportion,
						SlashValue:      totalSlashedAmount,
						SlashAssetsPool: []operatortypes.SlashAssetAmount{
							{
								AssetID: assetstype.ImuachainAssetID,
								Amount:  slashedAmountFromAssetPool,
							},
						},
						UndelegationFilterHeight: infractionHeight,
						HistoricalVotingPower:    infractionPower,
						SlashUnclaimedRewards: []operatortypes.SlashFromUnclaimedRewards{
							{
								Avs: suite.DogfoodAVSAddr,
								SlashAssets: []operatortypes.SlashAssetAmount{
									{
										AssetID: assetstype.ImuachainAssetID,
										Amount:  slashedAmountDecFromOutStandingReward.TruncateInt().Add(slashedAmountDecFromCompoundingReward.TruncateInt()),
									},
								},
							},
						},
					},
				}
				suite.Require().Equal(expectedSlashRecord, *slashRecord)

				operatorSlashEvents, err := suite.App.DistrKeeper.GetOperatorSlashEvents(suite.Ctx, suite.testOperators[0].String(), assetstype.ImuachainAssetID, dogfoodtypes.DefaultEpochIdentifier)
				suite.Require().NoError(err)
				suite.Require().Equal([]string{assetstype.ImuachainAssetID}, operatorSlashEvents[0].OperatorSlashEvent.SlashedRewardAssets)

				unclaimedRewardsAfterSlash, err := suite.App.DistrKeeper.GetOperatorUnclaimedRewards(suite.Ctx, suite.testOperators[0].String(), suite.DogfoodAVSAddr)
				suite.Require().NoError(err)
				outstandingRewardsSlashed := sdk.NewDecCoins(sdk.NewDecCoinFromDec(utils.BaseDenom, slashedAmountDecFromOutStandingReward))
				compoundingRewardsSlashed := feedistributiontypes.NewCompoundingRewards(feedistributiontypes.CompoundingRewardsPerAsset{
					RewardDenomination: utils.BaseDenom,
					Rewards: feedistributiontypes.NewCommonAVSRewards(feedistributiontypes.CommonAVSRewardData{
						AVSAddress: suite.DogfoodAVSAddr,
						Rewards:    sdk.NewDecCoins(sdk.NewDecCoinFromDec(utils.BaseDenom, slashedAmountDecFromCompoundingReward)),
					}),
				})
				suite.Require().Equal(feedistributiontypes.OperatorUnclaimedRewards{
					OutstandingRewards:        operatorUnclaimedRewards.OutstandingRewards.Sub(outstandingRewardsSlashed),
					OutstandingRewardsSlashed: outstandingRewardsSlashed,
					RewardsFromCompounding:    feedistributiontypes.CompoundingRewards(operatorUnclaimedRewards.RewardsFromCompounding).Sub(compoundingRewardsSlashed),
					CompoundingRewardsSlashed: compoundingRewardsSlashed,
				}, unclaimedRewardsAfterSlash)
			},
		},
		{
			name:     "pass - test slashing from redelegated rewards",
			readOnly: false,
			expPass:  true,
			runFn: func() {
				// run one epoch to activate the operator and delegations
				// and run another epoch to earn the rewards and update the voting power at the end of epoch
				suite.RunToEpochEndN(dogfoodtypes.DefaultEpochIdentifier, 2)
				// enable the automatic redelegation for the test staker
				stakerID, _ := assetstype.GetStakerIDAndAssetID(assetstype.ImuachainLzID, suite.testOperators[0], nil)
				err := suite.App.DistrKeeper.SetStakerRewardParams(suite.Ctx, stakerID, feedistributiontypes.StakerRewardParams{
					RedelegateReward:       true,
					RedelegateOperatorAddr: suite.testOperators[0].String(),
				})
				suite.Require().NoError(err)

				// set the infraction height before undelegations to test slashing on subsequent undelegations.
				infractionHeight := suite.Ctx.BlockHeight()

				// undelegate all staking asset to trigger the rewards distribution
				delegationAmountBigInt := feedistributiontypes.UnscaleDecToInt(math.LegacyNewDec(testutil.DefaultDelegateAmount), suite.Assets[1].Decimals)
				suite.Delegation(false, assetstype.ImuachainLzID, common.Address(suite.testOperators[0]), common.HexToAddress(assetstype.ImuachainAssetAddr), suite.testOperators[0], delegationAmountBigInt)
				// run an epoch to trigger reward distribution for this undelegation.
				suite.RunToEpochEnd(dogfoodtypes.DefaultEpochIdentifier)

				// undelegate rewards
				delegation, err := suite.App.DelegationKeeper.GetSingleDelegationInfo(suite.Ctx, stakerID, assetstype.ImuachainAssetID, suite.testOperators[0].String())
				suite.Require().NoError(err)
				operatorAssetStates, err := suite.App.AssetsKeeper.GetOperatorSpecifiedAssetInfo(suite.Ctx, suite.testOperators[0], assetstype.ImuachainAssetID)
				suite.Require().NoError(err)
				suite.Require().Equal(delegationAmountBigInt, operatorAssetStates.PendingUndelegationAmount)

				undelegateRewardAmount, err := delegationkeeper.TokensFromShares(delegation.RewardUndelegatableShare, operatorAssetStates.TotalShare, operatorAssetStates.TotalAmount)
				suite.Require().NoError(err)
				err = suite.App.DistrKeeper.UndelegateClaimedRewards(suite.Ctx, stakerID, assetstype.ImuachainAssetID, suite.testOperators[0], false, undelegateRewardAmount)
				suite.Require().NoError(err)

				// calculate the rewards
				operatorInitialPower := sdk.NewDec(testutil.DefaultDelegateAmount * int64(len(s.testStakers)))
				totalPower := sdk.NewDec(suite.TotalPower).Add(operatorInitialPower)
				mintParam := suite.App.ImmintKeeper.GetParams(suite.Ctx)
				epochRewardDec := sdk.NewDecFromInt(mintParam.EpochReward)
				stakingRewardsEpoch1, _ := suite.calcExpectedOperatorAssetReward(
					operatorInitialPower.TruncateInt64(), totalPower.TruncateInt64(),
					operatorInitialPower, operatorInitialPower,
					operatorInitialPower, epochRewardDec,
					feedistributiontypes.DefaultParams().CommunityTax, testutil.DefaultOperatorCommission.Rate,
					1, suite.DogfoodAVSAddr, utils.BaseDenom)

				compoundingReward := stakingRewardsEpoch1.Rewards[0].Amount
				compoundingPower := utils.CalculateRewardUSDValue(compoundingReward, feedistributiontypes.IMUARewardToken.RewardAssetInfo.DenominationExponent, suite.Assets[1].Decimals, math.NewInt(1), 0)
				totalPower.AddMut(compoundingPower.TruncateDec())
				operatorPowerDurEpoch3 := operatorInitialPower.Add(compoundingPower.TruncateDec())

				// slash the operator at the end of epoch 3
				slashFactor := suite.App.SlashingKeeper.SlashFractionDowntime(suite.Ctx)
				slashType := stakingtypes.Infraction_INFRACTION_DOWNTIME
				infractionPower := operatorPowerDurEpoch3.TruncateInt64()
				suite.App.OperatorKeeper.SlashWithInfractionReason(suite.Ctx, suite.testOperators[0], infractionHeight, infractionPower, slashFactor, slashType)

				// check the states after slashing from compounding rewards.
				operatorAssetInfo, err := suite.App.AssetsKeeper.GetOperatorSpecifiedAssetInfo(suite.Ctx, suite.testOperators[0], assetstype.ImuachainAssetID)
				suite.NoError(err)
				operatorTotalUSDValue := utils.CalculateUSDValue(operatorAssetInfo.PendingUndelegationAmount, math.NewInt(1), suite.Assets[1].Decimals, 0)

				totalSlashedAmount := math.LegacyNewDec(infractionPower).Mul(slashFactor)
				actualSlashProportion := totalSlashedAmount.Quo(operatorTotalUSDValue)
				slashAmountFromStakingUndelegation := actualSlashProportion.MulInt(delegationAmountBigInt).TruncateInt()
				slashAmountFromRewardUndelegation := actualSlashProportion.MulInt(undelegateRewardAmount).TruncateInt()

				slashID := operatorkeeper.GetSlashIDForDogfood(slashType, infractionHeight)
				slashRecord, err := suite.App.OperatorKeeper.GetOperatorSlashInfo(suite.Ctx, suite.DogfoodAVSAddr, suite.testOperators[0].String(), slashID)
				suite.Require().NoError(err)
				expectedSlashRecord := operatortypes.OperatorSlashInfo{
					SubmittedHeight: suite.Ctx.BlockHeight(),
					EventHeight:     infractionHeight,
					SlashProportion: slashFactor,
					SlashType:       uint32(slashType),
					ExecutionInfo: &operatortypes.SlashExecutionInfo{
						SlashProportion: actualSlashProportion,
						SlashValue:      totalSlashedAmount,
						SlashUndelegations: []operatortypes.SlashFromUndelegation{
							{
								StakerID:       stakerID,
								AssetID:        assetstype.ImuachainAssetID,
								Amount:         slashAmountFromStakingUndelegation,
								UndelegationId: 0,
								RewardAsset:    false,
							},
							{
								StakerID:       stakerID,
								AssetID:        assetstype.ImuachainAssetID,
								Amount:         slashAmountFromRewardUndelegation,
								UndelegationId: 1,
								RewardAsset:    true,
							},
						},
						UndelegationFilterHeight: infractionHeight,
						HistoricalVotingPower:    infractionPower,
					},
				}
				suite.Require().Equal(expectedSlashRecord, *slashRecord)

				stakerClaimedRewards, err := suite.App.DistrKeeper.GetStakerClaimedRewards(suite.Ctx, stakerID, suite.DogfoodAVSAddr)
				suite.Require().NoError(err)
				suite.Require().Equal(sdk.NewDecCoins(sdk.NewDecCoin(utils.BaseDenom, undelegateRewardAmount)), stakerClaimedRewards.PendingUndelegationRewards)
				suite.Require().Equal(sdk.NewDecCoins(sdk.NewDecCoin(utils.BaseDenom, slashAmountFromRewardUndelegation)), stakerClaimedRewards.PendingSlashedRewards)
			},
		},
	}
	for _, tc := range testcases {
		tc := tc
		s.Run(tc.name, func() {
			s.SetupTest() // Reset state for each test case

			// increase the epoch rewards for voting power test.
			mintParams := imminttypes.DefaultParams()
			multiplier := math.NewIntWithDecimal(1, int(suite.Assets[1].Decimals)) // 10^decimals
			mintParams.EpochReward = mintParams.EpochReward.Mul(multiplier)
			suite.App.ImmintKeeper.SetParams(suite.Ctx, mintParams)

			// add the IMUA token into the support list of dogfood
			_, imuaAssetID := assetstype.GetStakerIDAndAssetIDFromStr(
				suite.ClientChains[1].LayerZeroChainID,
				"", suite.Assets[1].Address,
			)
			suite.AssetIDs = []string{suite.AssetIDs[0], imuaAssetID}
			suite.updateDogfoodAssetsList(suite.AssetIDs)

			// register an operator with compounding of unclaimed rewards enabled.
			// prepare the test operators and delegations
			operators := suite.RegisterOperators(1, false)
			suite.testOperators = operators
			suite.testStakers = []common.Address{common.Address(operators[0])}
			// deposit and delegate the IMUA token before the operator has opted-in
			suite.DepositAndDelegateIMUAToOperators(suite.testStakers, operators, testutil.DefaultDepositAmount, testutil.DefaultDelegateAmount)
			// opts the operators into the dogfood AVS
			suite.OptIntoDogfood(operators)
			tc.runFn()
		})
	}
}

func (suite *KeeperTestSuite) TestGetStakerUnclaimedRewards() {
	testcases := []struct {
		name     string
		malleate func() (string, feedistributiontypes.CommonAVSRewards, feedistributiontypes.CommonAVSRewards)
		expPass  bool
	}{
		{
			name:    "pass - query unclaimed rewards with no triggered actions (pure accumulation)",
			expPass: true,
			malleate: func() (string, feedistributiontypes.CommonAVSRewards, feedistributiontypes.CommonAVSRewards) {
				// 1. Setup: Register asset and update dogfood list
				testStakerID := suite.StakerIDs[0]

				// 2. Action: Run some epochs to accumulate rewards. No need to deposit or delegate
				// as we use the genesis delegation for this test.
				runEpochNumber := 5
				suite.RunToEpochEndN(dogfoodtypes.DefaultEpochIdentifier, runEpochNumber)

				// 3. Calculate Expected Rewards
				mintParam := suite.App.ImmintKeeper.GetParams(suite.Ctx)
				epochRewardDec := sdk.NewDecFromInt(mintParam.EpochReward)

				// Calculate expected reward ratio for the duration
				// Note: The power doesn't change during these epochs
				_, operatorRewardRatio := suite.calcExpectedOperatorAssetReward(
					suite.Powers[0], suite.TotalPower,
					sdk.NewDec(suite.Powers[0]), sdk.NewDec(suite.Powers[0]),
					sdk.NewDec(suite.Powers[0]), epochRewardDec,
					feedistributiontypes.DefaultParams().CommunityTax, sdk.ZeroDec(),
					runEpochNumber, suite.DogfoodAVSAddr, utils.BaseDenom)

				// Calculate total expected staking rewards
				expectedStakingRewards := operatorRewardRatio.Rewards.MulDecTruncate(sdk.NewDec(suite.Powers[0]))

				// Return stakerID, expected Staking Rewards, expected Compounding Rewards (0 in this case)
				return testStakerID,
					feedistributiontypes.CommonAVSRewards{
						{
							AVSAddress: suite.DogfoodAVSAddr,
							Rewards:    expectedStakingRewards,
						},
					},
					([]feedistributiontypes.CommonAVSRewardData)(nil)
			},
		},
		{
			name:    "pass - verify unclaimed rewards are zero after claim(triggered by new delegation) and before next epoch accumulation",
			expPass: true,
			malleate: func() (string, feedistributiontypes.CommonAVSRewards, feedistributiontypes.CommonAVSRewards) {
				// 1. Setup & Accumulate
				testStakerID := suite.StakerIDs[0]
				testOperator := suite.Operators[0]
				assetAddr := common.HexToAddress(suite.Assets[0].Address)

				phase1Epochs := 5
				suite.RunToEpochEndN(dogfoodtypes.DefaultEpochIdentifier, phase1Epochs)

				// 2. Trigger Claim via delegation (Half Amount)
				delegationAmount := math.NewIntWithDecimal(suite.Powers[0]/2, int(suite.Assets[0].Decimals))
				suite.Delegation(false, suite.testClientChainID, common.Address(testOperator), assetAddr, testOperator, delegationAmount)

				// 3. run to the end of current epoch to trigger the claiming.
				suite.RunToEpochEnd(dogfoodtypes.DefaultEpochIdentifier)

				// 4. advance blocks without ending the epoch to verify rewards remain zero.
				advanceBlockNumber := 10
				for i := 0; i < advanceBlockNumber; i++ {
					suite.NextBlock()
				}

				// Since undelegation triggers an auto-claim, the "Unclaimed Rewards" in the store
				// should be cleared. The expected return value for this test case is EMPTY.
				return testStakerID,
					([]feedistributiontypes.CommonAVSRewardData)(nil), // Expecting 0 rewards
					([]feedistributiontypes.CommonAVSRewardData)(nil)
			},
		},
		{
			name:    "pass - verify rewards re-accumulate correctly for the remaining stake after a cleared epoch",
			expPass: true,
			malleate: func() (string, feedistributiontypes.CommonAVSRewards, feedistributiontypes.CommonAVSRewards) {
				// 1. Setup & Accumulate (Same as above)
				testStakerID := suite.StakerIDs[0]
				testOperator := suite.Operators[0]
				assetAddr := common.HexToAddress(suite.Assets[0].Address)

				phase1Epochs := 5
				suite.RunToEpochEndN(dogfoodtypes.DefaultEpochIdentifier, phase1Epochs)

				// 2. Trigger Claim via delegation (Half Amount)
				delegationAmount := math.NewIntWithDecimal(suite.Powers[0]/2, int(suite.Assets[0].Decimals))

				suite.Delegation(true, suite.testClientChainID, common.Address(testOperator), assetAddr, testOperator, delegationAmount)
				suite.RunToEpochEnd(dogfoodtypes.DefaultEpochIdentifier)

				// Optional: Internal assertion to ensure starting state is clean (defensive testing)
				rewardsCheck, _, _ := suite.App.DistrKeeper.GetStakerUnclaimedRewards(suite.Ctx, testStakerID)
				suite.Require().True(rewardsCheck.IsZeroRewards(), "Sanity check: Rewards must be zero before re-accumulation phase")

				// 3. Phase 2 Accumulation (Run 1 new epoch)
				phase2Epochs := 1
				suite.RunToEpochEndN(dogfoodtypes.DefaultEpochIdentifier, phase2Epochs)

				// 4. Calculate Expected Rewards for Phase 2 ONLY
				mintParam := suite.App.ImmintKeeper.GetParams(suite.Ctx)
				epochRewardDec := sdk.NewDecFromInt(mintParam.EpochReward)

				// Calculate power changes
				addedPower := suite.Powers[0] / 2
				newTotalPower := suite.TotalPower + addedPower
				newOperatorAssetPower := suite.Powers[0] + addedPower

				// Calculate expected reward for the new epoch
				_, operatorRewardRatio := suite.calcExpectedOperatorAssetReward(
					newOperatorAssetPower, newTotalPower,
					sdk.NewDec(newOperatorAssetPower), sdk.NewDec(newOperatorAssetPower),
					sdk.NewDec(newOperatorAssetPower), epochRewardDec,
					feedistributiontypes.DefaultParams().CommunityTax, sdk.ZeroDec(),
					phase2Epochs, suite.DogfoodAVSAddr, utils.BaseDenom)

				expectedNewStakingRewards := operatorRewardRatio.Rewards.MulDecTruncate(sdk.NewDec(newOperatorAssetPower))

				// Return only the NEW rewards generated in Phase 2
				return testStakerID,
					feedistributiontypes.CommonAVSRewards{{AVSAddress: suite.DogfoodAVSAddr, Rewards: expectedNewStakingRewards}},
					([]feedistributiontypes.CommonAVSRewardData)(nil)
			},
		},
		{
			name:    "pass - verify compounding rewards when operator reward compounding is enabled",
			expPass: true,
			malleate: func() (string, feedistributiontypes.CommonAVSRewards, feedistributiontypes.CommonAVSRewards) {
				// 1. enable the reward compounding and add the reward asset to the support list of dogfood.
				err := suite.App.OperatorKeeper.UpdateRewardCompoundingFlag(suite.Ctx, suite.Operators[0], false)
				suite.Require().NoError(err)
				operatorInfo, err := suite.App.OperatorKeeper.OperatorInfo(suite.Ctx, suite.Operators[0].String())
				suite.Require().NoError(err)
				suite.Require().False(operatorInfo.DisableCompoundRewards)
				// add the IMUA token into the support list of dogfood
				suite.AssetIDs = []string{suite.AssetIDs[0], assetstype.ImuachainAssetID}
				suite.updateDogfoodAssetsList(suite.AssetIDs)

				// 2. update the mint parameter to distribute enough rewards to contribute voting power.
				mintParam := suite.App.ImmintKeeper.GetParams(suite.Ctx)
				mintParam.EpochReward = math.NewIntWithDecimal(mintParam.EpochReward.Int64(), 18)
				suite.App.ImmintKeeper.SetParams(suite.Ctx, mintParam)
				epochRewardDec := sdk.NewDecFromInt(mintParam.EpochReward)

				// 3. advance epochs to accumulate the staking rewards.
				phase1Epochs := 1
				suite.RunToEpochEndN(dogfoodtypes.DefaultEpochIdentifier, phase1Epochs)
				testStakerID := suite.StakerIDs[0]
				// get the rewards for phase1
				stakingRewards, compoundingRewards, err := suite.App.DistrKeeper.GetStakerUnclaimedRewards(suite.Ctx, testStakerID)
				suite.Require().Nil(compoundingRewards)
				operatorStakingRewardEpoch1, operatorStakingRewardRatio := suite.calcExpectedOperatorAssetReward(
					suite.Powers[0], suite.TotalPower,
					sdk.NewDec(suite.Powers[0]), sdk.NewDec(suite.Powers[0]),
					sdk.NewDec(suite.Powers[0]), epochRewardDec,
					feedistributiontypes.DefaultParams().CommunityTax, sdk.ZeroDec(),
					1, suite.DogfoodAVSAddr, utils.BaseDenom)
				expectedOperatorStakingRewardEpoch1 := operatorStakingRewardRatio.Rewards.MulDecTruncate(sdk.NewDec(suite.Powers[0]))
				suite.Require().Equal(feedistributiontypes.CommonAVSRewards{{AVSAddress: suite.DogfoodAVSAddr, Rewards: expectedOperatorStakingRewardEpoch1}}, stakingRewards)

				// 4. calculate the voting power of stakingRewards
				avsRewardAssets, err := suite.App.DistrKeeper.GetAllRewardAssetsByAVS(suite.Ctx, suite.DogfoodAVSAddr)
				imRewardAssetInfo := avsRewardAssets.AvsRewardAssets[0].RewardAssetInfo
				suite.Require().NoError(err)
				rewardAmountDec := stakingRewards.RewardsOf(suite.DogfoodAVSAddr).AmountOf(imRewardAssetInfo.RewardDenomination)
				price, err := suite.App.OracleKeeper.GetSpecifiedAssetsPrice(suite.Ctx, assetstype.ImuachainAssetID)
				suite.Require().NoError(err)
				rewardUSDValue := utils.CalculateRewardUSDValue(rewardAmountDec, imRewardAssetInfo.DenominationExponent, imRewardAssetInfo.Decimals, price.Value, price.Decimal)
				rewardPower := rewardUSDValue.TruncateInt64()
				// calculate power changes
				newTotalPower := suite.TotalPower + rewardPower
				operatorTotalPower := suite.Powers[0] + rewardPower
				dogfoodVotingPower := suite.App.StakingKeeper.GetLastTotalPower(suite.Ctx)
				suite.Require().Equal(newTotalPower, dogfoodVotingPower.Int64())
				// 5. advance an epoch to accumulate the compounding rewards.
				suite.RunToEpochEnd(dogfoodtypes.DefaultEpochIdentifier)

				// 6. calculate the expected rewards
				operatorStakingRewardEpoch2, operatorStakingRewardRatio := suite.calcExpectedOperatorAssetReward(
					operatorTotalPower, newTotalPower,
					sdk.NewDec(suite.Powers[0]), sdk.NewDec(suite.Powers[0]).Add(rewardUSDValue),
					sdk.NewDec(suite.Powers[0]), epochRewardDec,
					feedistributiontypes.DefaultParams().CommunityTax, sdk.ZeroDec(),
					1, suite.DogfoodAVSAddr, utils.BaseDenom)
				newStakingRewards := operatorStakingRewardRatio.Rewards.MulDecTruncate(sdk.NewDec(suite.Powers[0]))
				expectedTotalStakingRewards := stakingRewards.Add(feedistributiontypes.CommonAVSRewardData{AVSAddress: suite.DogfoodAVSAddr, Rewards: newStakingRewards})

				rewardAmount := feedistributiontypes.UnscaleDecToInt(rewardAmountDec, imRewardAssetInfo.DenominationExponent)
				operatorCompoundingRewards, _ := suite.calcExpectedOperatorAssetReward(
					operatorTotalPower, newTotalPower,
					rewardUSDValue, sdk.NewDec(suite.Powers[0]).Add(rewardUSDValue),
					feedistributiontypes.ScaleIntByDecimals(rewardAmount, imRewardAssetInfo.Decimals), epochRewardDec,
					feedistributiontypes.DefaultParams().CommunityTax, sdk.ZeroDec(),
					1, suite.DogfoodAVSAddr, utils.BaseDenom)

				totalStakingRewardAmount := expectedTotalStakingRewards.RewardsOf(suite.DogfoodAVSAddr).AmountOf(imRewardAssetInfo.RewardDenomination)
				operatorUnclaimedRewardAmount := operatorStakingRewardEpoch1.Add(operatorStakingRewardEpoch2).Rewards.AmountOf(imRewardAssetInfo.RewardDenomination)
				compoundingRewardProportion := totalStakingRewardAmount.QuoTruncate(operatorUnclaimedRewardAmount)
				expectedCompoundingReward, err := feedistributiontypes.CommonAVSRewards{operatorCompoundingRewards}.MulDecTruncate(compoundingRewardProportion)
				suite.NoError(err)

				return testStakerID,
					expectedTotalStakingRewards,
					expectedCompoundingReward
			},
		},
	}

	for _, tc := range testcases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest() // Reset state
			suite.testClientChainID = suite.ClientChains[0].LayerZeroChainID

			// Execute Malleate
			testStakerID, expStakingRewards, expCompoundingRewards := tc.malleate()

			// Check State: GetStakerUnclaimedRewards should return the accumulated rewards
			// without modifying the state (it uses CacheContext internally).
			stakingRewards, compoundingRewards, err := suite.App.DistrKeeper.GetStakerUnclaimedRewards(suite.Ctx, testStakerID)
			if tc.expPass {
				suite.Require().NoError(err)
				// Use specific equality checks for rewards
				suite.Require().Equal(expStakingRewards, stakingRewards, "Staking rewards mismatch")
				suite.Require().Equal(expCompoundingRewards, compoundingRewards, "Compounding rewards mismatch")
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
