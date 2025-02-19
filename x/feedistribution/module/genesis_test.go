package feedistribute_test

import (
	"testing"

	keepertest "github.com/imua-xyz/imuachain/testutil/keeper"
	"github.com/imua-xyz/imuachain/testutil/nullify"
	feedistribute "github.com/imua-xyz/imuachain/x/feedistribution/module"
	"github.com/imua-xyz/imuachain/x/feedistribution/types"
	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),

		// this line is used by starport scaffolding # genesis/test/state
	}

	k, ctx := keepertest.FeedistributeKeeper(t)
	feedistribute.InitGenesis(ctx, k, genesisState)
	got := feedistribute.ExportGenesis(ctx, k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	// this line is used by starport scaffolding # genesis/test/assert
}
