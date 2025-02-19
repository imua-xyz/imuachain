package oracle_test

import (
	"testing"

	keepertest "github.com/imua-xyz/imuachain/testutil/keeper"
	"github.com/imua-xyz/imuachain/testutil/nullify"
	"github.com/imua-xyz/imuachain/x/oracle"
	"github.com/imua-xyz/imuachain/x/oracle/types"
	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),

		PricesList: []types.Prices{
			{
				TokenID: 1,
				PriceList: []*types.PriceTimeRound{
					{
						Price:     "100",
						Decimal:   1,
						Timestamp: "-",
						RoundID:   1,
					},
				},
				NextRoundID: 2,
			},
			{
				TokenID: 2,
				PriceList: []*types.PriceTimeRound{
					{
						Price:     "109",
						Decimal:   1,
						Timestamp: "-",
						RoundID:   1,
					},
					{
						Price:     "119",
						Decimal:   1,
						Timestamp: "-",
						RoundID:   2,
					},
				},
				NextRoundID: 3,
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
		// this line is used by starport scaffolding # genesis/test/state
	}

	k, ctx := keepertest.OracleKeeper(t)
	oracle.InitGenesis(ctx, *k, genesisState)
	got := oracle.ExportGenesis(ctx, *k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	require.ElementsMatch(t, genesisState.PricesList, got.PricesList)
	require.Equal(t, genesisState.ValidatorUpdateBlock, got.ValidatorUpdateBlock)
	require.Equal(t, genesisState.IndexRecentParams, got.IndexRecentParams)
	require.Equal(t, genesisState.IndexRecentMsg, got.IndexRecentMsg)
	require.ElementsMatch(t, genesisState.RecentMsgList, got.RecentMsgList)
	require.ElementsMatch(t, genesisState.RecentParamsList, got.RecentParamsList)
	// this line is used by starport scaffolding # genesis/test/assert
}
