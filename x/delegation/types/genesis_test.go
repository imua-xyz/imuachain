package types_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	utiltx "github.com/imua-xyz/imuachain/testutil/tx"
	assetstypes "github.com/imua-xyz/imuachain/x/assets/types"
	"github.com/imua-xyz/imuachain/x/delegation/types"
	"github.com/stretchr/testify/suite"
)

type GenesisTestSuite struct {
	suite.Suite
}

func (suite *GenesisTestSuite) SetupTest() {
}

func TestGenesisTestSuite(t *testing.T) {
	suite.Run(t, new(GenesisTestSuite))
}

func (suite *GenesisTestSuite) TestValidateGenesis() {
	assetAddress := utiltx.GenerateAddress()
	stakerAddress := utiltx.GenerateAddress()
	lzID := uint64(101)
	stakerID, assetID := assetstypes.GetStakerIDAndAssetID(
		lzID, stakerAddress[:], assetAddress[:],
	)
	operatorAddress := sdk.AccAddress(utiltx.GenerateAddress().Bytes())
	singleStateKey := assetstypes.GetJoinedStoreKey(stakerID, assetID, operatorAddress.String())
	delegationStates := []types.DelegationStates{
		{
			Key: string(singleStateKey),
			States: types.DelegationAmounts{
				WaitUndelegationAmount: math.NewInt(0),
				UndelegatableShare:     math.LegacyNewDec(1000),
			},
		},
	}
	stakersByOperator := []types.StakersByOperator{
		{
			Key: string(assetstypes.GetJoinedStoreKey(operatorAddress.String(), assetID)),
			Stakers: []string{
				stakerID,
			},
		},
	}
	testCases := []struct {
		name       string
		genState   *types.GenesisState
		expPass    bool
		malleate   func(*types.GenesisState)
		unmalleate func(*types.GenesisState)
	}{
		{
			name:     "valid empty genesis",
			genState: &types.GenesisState{},
			expPass:  true,
		},
		{
			name:     "default",
			genState: types.DefaultGenesis(),
			expPass:  true,
		},
		{
			name:     "base, should pass",
			genState: types.NewGenesis(nil, delegationStates, stakersByOperator, nil),
			expPass:  true,
		},
		{
			name:     "invalid staker id",
			genState: types.NewGenesis(nil, delegationStates, stakersByOperator, nil),
			expPass:  false,
			malleate: func(gs *types.GenesisState) {
				invalidStateKey := assetstypes.GetJoinedStoreKey("invalid", assetID, operatorAddress.String())
				gs.DelegationStates[0].Key = string(invalidStateKey)
			},
			unmalleate: func(gs *types.GenesisState) {
				gs.DelegationStates[0].Key = string(singleStateKey)
			},
		},
		{
			name:     "duplicate state key",
			genState: types.NewGenesis(nil, delegationStates, stakersByOperator, nil),
			expPass:  false,
			malleate: func(gs *types.GenesisState) {
				gs.DelegationStates = append(gs.DelegationStates, gs.DelegationStates[0])
			},
			unmalleate: func(gs *types.GenesisState) {
				gs.DelegationStates = gs.DelegationStates[:1]
			},
		},
		{
			name:     "invalid asset id",
			genState: types.NewGenesis(nil, delegationStates, stakersByOperator, nil),
			expPass:  false,
			malleate: func(gs *types.GenesisState) {
				invalidStateKey := assetstypes.GetJoinedStoreKey(stakerID, "invalid", operatorAddress.String())
				gs.DelegationStates[0].Key = string(invalidStateKey)
			},
			unmalleate: func(gs *types.GenesisState) {
				gs.DelegationStates[0].Key = string(singleStateKey)
			},
		},
		{
			name:     "asset id mismatch",
			genState: types.NewGenesis(nil, delegationStates, stakersByOperator, nil),
			expPass:  false,
			malleate: func(gs *types.GenesisState) {
				stakerID, _ := assetstypes.GetStakerIDAndAssetID(
					lzID+1, stakerAddress[:], assetAddress[:],
				)
				invalidStateKey := assetstypes.GetJoinedStoreKey(stakerID, assetID, operatorAddress.String())
				gs.DelegationStates[0].Key = string(invalidStateKey)
			},
			unmalleate: func(gs *types.GenesisState) {
				gs.DelegationStates[0].Key = string(singleStateKey)
			},
		},
		{
			name:     "nil wrapped undelegatable share",
			genState: types.NewGenesis(nil, delegationStates, stakersByOperator, nil),
			expPass:  false,
			malleate: func(gs *types.GenesisState) {
				gs.DelegationStates[0].States.UndelegatableShare = math.LegacyDec{}
			},
			unmalleate: func(gs *types.GenesisState) {
				gs.DelegationStates[0].States.UndelegatableShare = math.LegacyNewDec(1000)
			},
		},
		{
			name:     "nil wrapped unbonding amount",
			genState: types.NewGenesis(nil, delegationStates, stakersByOperator, nil),
			expPass:  false,
			malleate: func(gs *types.GenesisState) {
				gs.DelegationStates[0].States.WaitUndelegationAmount = math.Int{}
			},
			unmalleate: func(gs *types.GenesisState) {
				gs.DelegationStates[0].States.WaitUndelegationAmount = math.NewInt(0)
			},
		},
		{
			name:     "negative wrapped undelegatable share",
			genState: types.NewGenesis(nil, delegationStates, stakersByOperator, nil),
			expPass:  false,
			malleate: func(gs *types.GenesisState) {
				gs.DelegationStates[0].States.UndelegatableShare = math.LegacyNewDec(-1)
			},
			unmalleate: func(gs *types.GenesisState) {
				gs.DelegationStates[0].States.UndelegatableShare = math.LegacyNewDec(1000)
			},
		},
		{
			name:     "invalid operator address",
			genState: types.NewGenesis(nil, delegationStates, stakersByOperator, nil),
			expPass:  false,
			malleate: func(gs *types.GenesisState) {
				invalidStateKey := assetstypes.GetJoinedStoreKey(stakerID, assetID, "invalid")
				gs.DelegationStates[0].Key = string(invalidStateKey)
			},
			unmalleate: func(gs *types.GenesisState) {
				gs.DelegationStates[0].Key = string(singleStateKey)
			},
		},
		{
			name:     "duplicate stakerID in associations",
			genState: types.NewGenesis(nil, delegationStates, stakersByOperator, nil),
			expPass:  false,
			malleate: func(gs *types.GenesisState) {
				gs.Associations = make([]types.StakerToOperator, 2)
				gs.Associations[0].StakerId = stakerID
				gs.Associations[0].Operator = operatorAddress.String()
				gs.Associations[1].StakerId = stakerID
				gs.Associations[1].Operator = operatorAddress.String()
			},
			unmalleate: func(gs *types.GenesisState) {
				gs.Associations = nil
			},
		},
		{
			name:     "one stakerID in associations",
			genState: types.NewGenesis(nil, delegationStates, stakersByOperator, nil),
			expPass:  true,
			malleate: func(gs *types.GenesisState) {
				gs.Associations = make([]types.StakerToOperator, 1)
				gs.Associations[0].StakerId = stakerID
				gs.Associations[0].Operator = operatorAddress.String()
			},
			unmalleate: func(gs *types.GenesisState) {
				gs.Associations = nil
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		if tc.malleate != nil {
			tc.malleate(tc.genState)
			// require that unmalleate is defined
			suite.Require().NotNil(tc.unmalleate, tc.name)
		}
		err := tc.genState.Validate()
		if tc.expPass {
			suite.Require().NoError(err, tc.name)
		} else {
			suite.Require().Error(err, tc.name)
		}
		if tc.unmalleate != nil {
			tc.unmalleate(tc.genState)
		}
	}
}
