package keeper_test

import (
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/imua-xyz/imuachain/testutil"
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
						StakerIds:   []string{defaultStakerID},
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

				expectedStakerIDs := make([]string, 0)
				for _, stakerAddr := range suite.testStakers {
					stakerID, _ := assetstype.GetStakerIDAndAssetIDFromStr(
						defaultLzChainID, strings.ToLower(stakerAddr.String()), "",
					)
					expectedStakerIDs = append(expectedStakerIDs, stakerID)
				}
				return markChangedDelegationsArgs{
						operator: suite.testOperators[0],
						assetID:  defaultAssetID,
					}, map[string]*feedistributiontypes.DelegationChangeInfo{
						dogfoodtypes.DefaultEpochIdentifier: {
							StakerIds:   expectedStakerIDs,
							TotalAmount: sdk.NewDec(preTotalDelegationAmount),
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
			defaultStakerID, defaultAssetID = assetstype.GetStakerIDAndAssetIDFromStr(
				defaultLzChainID, strings.ToLower(defaultStakerAddr.String()), suite.Assets[0].Address,
			)
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
					suite.NoError(err)
					suite.Require().Equal(*expectedState, actualState, fmt.Sprintf("epochIdentifier:%s,operator:%s,assetID:%s", epochIdentifier, args.operator.String(), args.assetID))
				} else {
					suite.ErrorContains(err, feedistributiontypes.ErrNoKeyInTheStore.Error())
				}
			}
		})
	}
}

/*func (suite *KeeperTestSuite) TestHandleChangedDelegations() {
	testcases := []struct {
		name     string
		malleate func()
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
			malleate: func() {
			},
		},
	}
	for _, tc := range testcases {
		tc := tc
		s.Run(tc.name, func() {
			s.SetupTest() // Reset state for each test case
			s.prepareTestBase(TestStakerNumber, TestOperatorNumber, 1)

			args, expectedStates := tc.malleate()
			// check the state after unit test
			for epochIdentifier, expectedState := range expectedStates {
				actualState, err := suite.App.DistrKeeper.GetStakeChangedDelegations(suite.Ctx, epochIdentifier, args.operator.String(), args.assetID)
				if expectedState != nil {
					suite.NoError(err)
					suite.Require().Equal(*expectedState, actualState, fmt.Sprintf("epochIdentifier:%s,operator:%s,assetID:%s", epochIdentifier, args.operator.String(), args.assetID))
				} else {
					suite.ErrorContains(err, feedistributiontypes.ErrNoKeyInTheStore.Error())
				}
			}
		})
	}
}*/
