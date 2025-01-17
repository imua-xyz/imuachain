package types_test

import (
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	"github.com/ExocoreNetwork/exocore/x/oracle/types"
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
				PricesList: []types.Prices{
					{
						TokenID: 0,
					},
					{
						TokenID: 1,
					},
				},
				ValidatorUpdateBlock: &types.ValidatorUpdateBlock{},
				IndexRecentParams:    &types.IndexRecentParams{},
				IndexRecentMsg:       &types.IndexRecentMsg{},
				RecentMsgList: []types.RecentMsg{
					{
						Block: 0,
					},
					{
						Block: 1,
					},
				},
				RecentParamsList: []types.RecentParams{
					{
						Block: 0,
					},
					{
						Block: 1,
					},
				},
				Params: types.Params{
					MaxNonce:      3,
					ThresholdA:    2,
					ThresholdB:    3,
					Mode:          types.ConsensusModeASAP,
					MaxDetId:      5,
					MaxSizePrices: 100,
					Slashing: &types.SlashingParams{
						ReportedRoundsWindow:        100,
						MinReportedPerWindow:        sdkmath.LegacyNewDec(1).Quo(sdkmath.LegacyNewDec(2)),
						OracleMissJailDuration:      600 * time.Second,
						OracleMaliciousJailDuration: 30 * 24 * time.Hour,
						SlashFractionMalicious:      sdkmath.LegacyNewDec(1).Quo(sdkmath.LegacyNewDec(10)),
					},
				},
				// this line is used by starport scaffolding # types/genesis/validField
			},
			valid: true,
		},
		{
			desc: "duplicated prices",
			genState: &types.GenesisState{
				PricesList: []types.Prices{
					{
						TokenID: 0,
					},
					{
						TokenID: 0,
					},
				},
			},
			valid: false,
		},
		{
			desc: "duplicated recentMsg",
			genState: &types.GenesisState{
				RecentMsgList: []types.RecentMsg{
					{
						Block: 0,
					},
					{
						Block: 0,
					},
				},
			},
			valid: false,
		},
		{
			desc: "duplicated recentParams",
			genState: &types.GenesisState{
				RecentParamsList: []types.RecentParams{
					{
						Block: 0,
					},
					{
						Block: 0,
					},
				},
			},
			valid: false,
		},
		{
			desc: "length not match for stakerInfosAssets and stakerListAssets",
			genState: &types.GenesisState{
				StakerListAssets: []types.StakerListAssets{
					{
						AssetId:    "0xasset1",
						StakerList: &types.StakerList{},
					},
					{
						AssetId:    "0xasset2",
						StakerList: &types.StakerList{},
					},
				},
				StakerInfosAssets: []types.StakerInfosAssets{
					{
						AssetId:     "0xasset1",
						StakerInfos: []*types.StakerInfo{},
					},
				},
			},
			valid: false,
		},
		{
			desc: "assetIds not match for stakerInfosAssets and stakerListAssets",
			genState: &types.GenesisState{
				StakerListAssets: []types.StakerListAssets{
					{
						AssetId:    "0xasset1",
						StakerList: &types.StakerList{},
					},
					{
						AssetId:    "0xasset2",
						StakerList: &types.StakerList{},
					},
				},
				StakerInfosAssets: []types.StakerInfosAssets{
					{
						AssetId:     "0xasset1",
						StakerInfos: []*types.StakerInfo{},
					},
					{
						AssetId:     "0xasset3",
						StakerInfos: []*types.StakerInfo{},
					},
				},
			},
			valid: false,
		},
		{
			desc: "invalid",
			genState: &types.GenesisState{
				StakerListAssets: []types.StakerListAssets{
					{
						AssetId:    "0xasset1",
						StakerList: &types.StakerList{},
					},
					{
						AssetId:    "0xasset2",
						StakerList: &types.StakerList{},
					},
				},
				StakerInfosAssets: []types.StakerInfosAssets{
					{
						AssetId:     "0xasset1",
						StakerInfos: []*types.StakerInfo{},
					},
					{
						AssetId:     "0xasset2",
						StakerInfos: []*types.StakerInfo{},
					},
				},
			},
			valid: false,
		},
		{
			desc: "stakerAddr not matched for stakerInfosAsset and stakerListAsset",
			genState: &types.GenesisState{
				StakerListAssets: []types.StakerListAssets{
					{
						AssetId: "0xasset1",
						StakerList: &types.StakerList{
							StakerAddrs: []string{
								"staker_01",
								"staker_02",
							},
						},
					},
					{
						AssetId:    "0xasset2",
						StakerList: &types.StakerList{},
					},
				},
				StakerInfosAssets: []types.StakerInfosAssets{
					{
						AssetId: "0xasset1",
						StakerInfos: []*types.StakerInfo{
							{
								StakerIndex: 0,
								StakerAddr:  "staker_01",
							},
							{
								StakerIndex: 2,
								StakerAddr:  "staker_03",
							},
						},
					},
					{
						AssetId:     "0xasset2",
						StakerInfos: []*types.StakerInfo{},
					},
				},
			},
			valid: false,
		},
		{
			desc: "stakerIndex not matched for stakerInfosAsset and stakerListAsset",
			genState: &types.GenesisState{
				StakerListAssets: []types.StakerListAssets{
					{
						AssetId: "0xasset1",
						StakerList: &types.StakerList{
							StakerAddrs: []string{
								"staker_01",
								"staker_02",
							},
						},
					},
					{
						AssetId:    "0xasset2",
						StakerList: &types.StakerList{},
					},
				},
				StakerInfosAssets: []types.StakerInfosAssets{
					{
						AssetId: "0xasset1",
						StakerInfos: []*types.StakerInfo{
							{
								StakerIndex: 0,
								StakerAddr:  "staker_01",
							},
							{
								StakerIndex: 2,
								StakerAddr:  "staker_02",
							},
						},
					},
					{
						AssetId:     "0xasset2",
						StakerInfos: []*types.StakerInfo{},
					},
				},
			},
			valid: false,
		},
		{
			desc: "valid",
			genState: &types.GenesisState{
				StakerListAssets: []types.StakerListAssets{
					{
						AssetId: "0xasset1",
						StakerList: &types.StakerList{
							StakerAddrs: []string{
								"staker_01",
								"staker_02",
							},
						},
					},
					{
						AssetId:    "0xasset2",
						StakerList: &types.StakerList{},
					},
				},
				StakerInfosAssets: []types.StakerInfosAssets{
					{
						AssetId: "0xasset1",
						StakerInfos: []*types.StakerInfo{
							{
								StakerIndex: 0,
								StakerAddr:  "staker_01",
							},
							{
								StakerIndex: 1,
								StakerAddr:  "staker_02",
							},
						},
					},
					{
						AssetId:     "0xasset2",
						StakerInfos: []*types.StakerInfo{},
					},
				},
			},
			valid: false,
		},

		// this line is used by starport scaffolding # types/genesis/testcase
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
