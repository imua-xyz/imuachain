package types_test

import (
	"testing"
	"time"

	"github.com/ExocoreNetwork/exocore/x/epochs/types"
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
	testCases := []struct {
		name     string
		genState *types.GenesisState
		expPass  bool
		expError string
		malleate func(*types.GenesisState)
	}{
		{
			name:     "constructor",
			genState: &types.GenesisState{},
			expPass:  true,
		},
		{
			name:     "default",
			genState: types.DefaultGenesis(),
			expPass:  true,
		},
		{
			name: "NewGenesis call",
			genState: types.NewGenesisState(
				[]types.EpochInfo{},
			),
			expPass: true,
		},
		{
			name: "duplicate epoch identifiers",
			genState: types.NewGenesisState(
				[]types.EpochInfo{
					types.NewGenesisEpochInfo("epoch1", 1),
					types.NewGenesisEpochInfo("epoch1", 2),
				},
			),
			expPass:  false,
			expError: "epoch identifier should be unique",
		},
		{
			name: "blank epoch identifiers",
			genState: types.NewGenesisState(
				[]types.EpochInfo{
					types.NewGenesisEpochInfo("", 1),
				},
			),
			expPass:  false,
			expError: "empty epoch identifier",
		},
		{
			name: "zero epoch duration",
			genState: types.NewGenesisState(
				[]types.EpochInfo{
					types.NewGenesisEpochInfo("i am an epoch identifier", 0),
				},
			),
			expPass:  false,
			expError: "epoch duration should NOT be non-positive",
		},
		{
			name: "negative current epoch number",
			genState: types.NewGenesisState(
				[]types.EpochInfo{
					types.NewGenesisEpochInfo("hourly", time.Hour),
				},
			),
			malleate: func(genState *types.GenesisState) {
				genState.Epochs[0].CurrentEpoch = -1
			},
			expPass:  false,
			expError: "epoch CurrentEpoch must be non-negative",
		},
		{
			name: "negative epoch start height",
			genState: types.NewGenesisState(
				[]types.EpochInfo{
					types.NewGenesisEpochInfo("hourly", time.Hour),
				},
			),
			malleate: func(genState *types.GenesisState) {
				genState.Epochs[0].CurrentEpochStartHeight = -1
			},
			expPass:  false,
			expError: "epoch CurrentEpochStartHeight must be non-negative",
		},
	}

	for _, tc := range testCases {
		tc := tc
		if tc.malleate != nil {
			tc.malleate(tc.genState)
		}
		err := tc.genState.Validate()
		if tc.expPass {
			suite.Require().Equal("", tc.expError, tc.name)
			suite.Require().NoError(err, tc.name)
		} else {
			suite.Require().NotEqual("", tc.expError, tc.name)
			suite.Require().Error(err, tc.name)
			suite.Require().Contains(err.Error(), tc.expError, tc.name)
		}
	}
}
