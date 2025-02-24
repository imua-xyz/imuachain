package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/imua-xyz/imuachain/testutil/keeper"
	"github.com/imua-xyz/imuachain/testutil/nullify"
	"github.com/imua-xyz/imuachain/x/oracle/keeper"
	"github.com/imua-xyz/imuachain/x/oracle/types"
)

func createTestIndexRecentParams(keeper *keeper.Keeper, ctx sdk.Context) types.IndexRecentParams {
	item := types.IndexRecentParams{}
	keeper.SetIndexRecentParams(ctx, item)
	return item
}

func TestIndexRecentParamsGet(t *testing.T) {
	keeper, ctx := keepertest.OracleKeeper(t)
	item := createTestIndexRecentParams(keeper, ctx)
	rst, found := keeper.GetIndexRecentParams(ctx)
	require.True(t, found)
	require.Equal(t,
		nullify.Fill(&item),
		nullify.Fill(&rst),
	)
}

func TestIndexRecentParamsRemove(t *testing.T) {
	keeper, ctx := keepertest.OracleKeeper(t)
	createTestIndexRecentParams(keeper, ctx)
	keeper.RemoveIndexRecentParams(ctx)
	_, found := keeper.GetIndexRecentParams(ctx)
	require.False(t, found)
}
