package types_test

import (
	"testing"

	"github.com/imua-xyz/imuachain/x/feedistribution/types"
	"github.com/stretchr/testify/require"
)

func TestGenesisState_Validate(t *testing.T) {
	tests := []struct {
		desc     string
		genState *types.GenesisState
		valid    bool
	}{
		{
			desc:     "default is valid",
			genState: types.DefaultGenesis(),
			valid:    true,
		},
		{
			desc: "valid genesis state",
			genState: &types.GenesisState{
				Params: types.Params{
					EpochIdentifier: "day",
				},
			},
			valid: true,
		},
		{
			desc: "invalid genesis state",
			genState: &types.GenesisState{
				Params: types.Params{
					EpochIdentifier: "",
				},
			},
			valid: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.genState.Validate()
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
