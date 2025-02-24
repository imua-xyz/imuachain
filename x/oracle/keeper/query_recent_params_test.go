package keeper_test

import (
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	keepertest "github.com/imua-xyz/imuachain/testutil/keeper"
	"github.com/imua-xyz/imuachain/testutil/nullify"
	"github.com/imua-xyz/imuachain/x/oracle/types"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func TestRecentParamsQuerySingle(t *testing.T) {
	keeper, ctx := keepertest.OracleKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	msgs := createNRecentParams(keeper, ctx, 2)
	tests := []struct {
		desc     string
		request  *types.QueryGetRecentParamsRequest
		response *types.QueryGetRecentParamsResponse
		err      error
	}{
		{
			desc: "First",
			request: &types.QueryGetRecentParamsRequest{
				Block: msgs[0].Block,
			},
			response: &types.QueryGetRecentParamsResponse{RecentParams: msgs[0]},
		},
		{
			desc: "Second",
			request: &types.QueryGetRecentParamsRequest{
				Block: msgs[1].Block,
			},
			response: &types.QueryGetRecentParamsResponse{RecentParams: msgs[1]},
		},
		{
			desc: "KeyNotFound",
			request: &types.QueryGetRecentParamsRequest{
				Block: 100000,
			},
			err: status.Error(codes.NotFound, "not found"),
		},
		{
			desc: "InvalidRequest",
			err:  status.Error(codes.InvalidArgument, "invalid request"),
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			response, err := keeper.RecentParams(wctx, tc.request)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
			} else {
				require.NoError(t, err)
				require.Equal(t,
					nullify.Fill(tc.response),
					nullify.Fill(response),
				)
			}
		})
	}
}

func TestRecentParamsQueryPaginated(t *testing.T) {
	keeper, ctx := keepertest.OracleKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	msgs := createNRecentParams(keeper, ctx, 5)

	request := func(next []byte, offset, limit uint64, total bool) *types.QueryAllRecentParamsRequest {
		return &types.QueryAllRecentParamsRequest{
			Pagination: &query.PageRequest{
				Key:        next,
				Offset:     offset,
				Limit:      limit,
				CountTotal: total,
			},
		}
	}
	t.Run("ByOffset", func(t *testing.T) {
		step := 2
		for i := 0; i < len(msgs); i += step {
			resp, err := keeper.RecentParamsAll(wctx, request(nil, uint64(i), uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.RecentParams), step)
			require.Subset(t,
				nullify.Fill(msgs),
				nullify.Fill(resp.RecentParams),
			)
		}
	})
	t.Run("ByKey", func(t *testing.T) {
		step := 2
		var next []byte
		for i := 0; i < len(msgs); i += step {
			resp, err := keeper.RecentParamsAll(wctx, request(next, 0, uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.RecentParams), step)
			require.Subset(t,
				nullify.Fill(msgs),
				nullify.Fill(resp.RecentParams),
			)
			next = resp.Pagination.NextKey
		}
	})
	t.Run("Total", func(t *testing.T) {
		resp, err := keeper.RecentParamsAll(wctx, request(nil, 0, 0, true))
		require.NoError(t, err)
		require.Equal(t, len(msgs), int(resp.Pagination.Total))
		require.ElementsMatch(t,
			nullify.Fill(msgs),
			nullify.Fill(resp.RecentParams),
		)
	})
	t.Run("InvalidRequest", func(t *testing.T) {
		_, err := keeper.RecentParamsAll(wctx, nil)
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid request"))
	})
}
