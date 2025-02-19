package keeper_test

import (
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	keepertest "github.com/imua-xyz/imuachain/testutil/keeper"
	"github.com/imua-xyz/imuachain/testutil/nullify"
	"github.com/imua-xyz/imuachain/x/oracle/keeper"
	"github.com/imua-xyz/imuachain/x/oracle/types"
	"github.com/stretchr/testify/require"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func createNRecentParams(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.RecentParams {
	items := make([]types.RecentParams, n)
	for i := range items {
		items[i].Block = uint64(i)

		keeper.SetRecentParams(ctx, items[i])
	}
	return items
}

func TestRecentParamsGet(t *testing.T) {
	keeper, ctx := keepertest.OracleKeeper(t)
	items := createNRecentParams(keeper, ctx, 10)
	for _, item := range items {
		rst, found := keeper.GetRecentParams(ctx,
			item.Block,
		)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&rst),
		)
	}
}

func TestRecentParamsRemove(t *testing.T) {
	keeper, ctx := keepertest.OracleKeeper(t)
	items := createNRecentParams(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemoveRecentParams(ctx,
			item.Block,
		)
		_, found := keeper.GetRecentParams(ctx,
			item.Block,
		)
		require.False(t, found)
	}
}

func TestRecentParamsGetAll(t *testing.T) {
	keeper, ctx := keepertest.OracleKeeper(t)
	items := createNRecentParams(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllRecentParams(ctx)),
	)
}
