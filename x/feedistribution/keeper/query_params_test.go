package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/imua-xyz/imuachain/testutil/keeper"
	"github.com/imua-xyz/imuachain/x/feedistribution/types"
)

func TestParamsQuery(t *testing.T) {
	keeper, ctx := keepertest.FeedistributeKeeper(t)
	params := types.DefaultParams()
	keeper.SetParams(ctx, params)

	response, err := keeper.Params(ctx, &types.QueryParamsRequest{})
	require.NoError(t, err)
	require.Equal(t, &types.QueryParamsResponse{Params: params}, response)
}
