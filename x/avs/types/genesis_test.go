package types_test

import (
	"testing"

	"github.com/ExocoreNetwork/exocore/x/avs/types"
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
				AvsInfos: []types.AVSInfo{
					{AvsAddress: "0x1234567890abcdef1234567890abcdef12345678"},
					{AvsAddress: "0x9876543210fedcba9876543210fedcba98765432"},
				}},
			valid: true,
		},
		{
			desc: "duplicated avs",
			genState: &types.GenesisState{
				AvsInfos: []types.AVSInfo{
					{AvsAddress: "0x9876543210fedcba9876543210fedcba98765432"},
					{AvsAddress: "0x9876543210fedcba9876543210fedcba98765432"},
				}},
			valid: false,
		},
		{
			desc: "invalid genesis state due hex address",
			genState: &types.GenesisState{
				AvsInfos: []types.AVSInfo{
					{AvsAddress: ""},
				}},
			valid: false,
		},
		{
			desc: "duplicated task",
			genState: &types.GenesisState{
				TaskInfos: []types.TaskInfo{
					{TaskContractAddress: "0x9876543210fedcba9876543210fedcba98765432"},
					{TaskContractAddress: "0x9876543210fedcba9876543210fedcba98765432"},
				}},
			valid: false,
		},
		{
			desc: "duplicated task num",
			genState: &types.GenesisState{
				TaskNums: []types.TaskID{
					{TaskAddr: "0x9876543210fedcba9876543210fedcba98765432", TaskId: 1},
					{TaskAddr: "0x9876543210fedcba9876543210fedcba98765432", TaskId: 1},
				}},
			valid: false,
		},
		{
			desc: "invalid operator address",
			genState: &types.GenesisState{
				TaskResultInfos: []types.TaskResultInfo{
					{OperatorAddress: "0x9876543210fedcba9876543210fedcba98765432", TaskId: 1, TaskContractAddress: "0x9876543210fedcba9876543210fedcba98765432"},
				}},
			valid: false,
		},
		{
			desc: "duplicated task result",
			genState: &types.GenesisState{
				TaskResultInfos: []types.TaskResultInfo{
					{OperatorAddress: "0x9876543210fedcba9876543210fedcba98765432", TaskId: 1, TaskContractAddress: "0x9876543210fedcba9876543210fedcba98765432"},
					{OperatorAddress: "0x9876543210fedcba9876543210fedcba98765432", TaskId: 1, TaskContractAddress: "0x9876543210fedcba9876543210fedcba98765432"},
				}},
			valid: false,
		},
		{
			desc: "invalid task result with different task IDs",
			genState: &types.GenesisState{
				TaskResultInfos: []types.TaskResultInfo{
					{OperatorAddress: "0x9876543210fedcba9876543210fedcba98765432", TaskId: 1, TaskContractAddress: "0x9876543210fedcba9876543210fedcba98765432"},
					{OperatorAddress: "0x9876543210fedcba9876543210fedcba98765432", TaskId: 2, TaskContractAddress: "0x9876543210fedcba9876543210fedcba98765432"},
				}},
			valid: false,
		},
		{
			desc: "invalid BLS public key info with invalid operator address",
			genState: &types.GenesisState{
				BlsPubKeys: []types.BlsPubKeyInfo{
					{Operator: "0x9876543210fedcba9876543210fedcba98765432", PubKey: nil},
				}},
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
