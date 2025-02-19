package keeper_test

import (
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	keepertest "github.com/imua-xyz/imuachain/testutil/keeper"
	"github.com/imua-xyz/imuachain/testutil/nullify"
	"github.com/imua-xyz/imuachain/x/oracle/types"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func TestPricesQuerySingle(t *testing.T) {
	keeper, ctx := keepertest.OracleKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	msgs := createNPrices(keeper, ctx, 2)
	tests := []struct {
		desc     string
		request  *types.QueryGetPricesRequest
		response *types.QueryGetPricesResponse
		err      error
	}{
		{
			desc: "First",
			request: &types.QueryGetPricesRequest{
				TokenId: msgs[0].TokenID,
			},
			response: &types.QueryGetPricesResponse{Prices: msgs[0]},
		},
		{
			desc: "Second",
			request: &types.QueryGetPricesRequest{
				TokenId: msgs[1].TokenID,
			},
			response: &types.QueryGetPricesResponse{Prices: msgs[1]},
		},
		{
			desc: "KeyNotFound",
			request: &types.QueryGetPricesRequest{
				TokenId: 100000,
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
			response, err := keeper.Prices(wctx, tc.request)
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

//func TestPricesQueryPaginated(t *testing.T) {
//	keeper, ctx := keepertest.OracleKeeper(t)
//	wctx := sdk.WrapSDKContext(ctx)
//	msgs := createNPrices(keeper, ctx, 5)
//
//	request := func(next []byte, offset, limit uint64, total bool) *types.QueryAllPricesRequest {
//		return &types.QueryAllPricesRequest{
//			Pagination: &query.PageRequest{
//				Key:        next,
//				Offset:     offset,
//				Limit:      limit,
//				CountTotal: total,
//			},
//		}
//	}
//	t.Run("ByOffset", func(t *testing.T) {
//		step := 2
//		for i := 0; i < len(msgs); i += step {
//			resp, err := keeper.PricesAll(wctx, request(nil, uint64(i), uint64(step), false))
//			require.NoError(t, err)
//			require.LessOrEqual(t, len(resp.Prices), step)
//			require.Subset(t,
//				nullify.Fill(msgs),
//				nullify.Fill(resp.Prices),
//			)
//		}
//	})
//	t.Run("ByKey", func(t *testing.T) {
//		step := 2
//		var next []byte
//		for i := 0; i < len(msgs); i += step {
//			resp, err := keeper.PricesAll(wctx, request(next, 0, uint64(step), false))
//			require.NoError(t, err)
//			require.LessOrEqual(t, len(resp.Prices), step)
//			require.Subset(t,
//				nullify.Fill(msgs),
//				nullify.Fill(resp.Prices),
//			)
//			next = resp.Pagination.NextKey
//		}
//	})
//	t.Run("Total", func(t *testing.T) {
//		resp, err := keeper.PricesAll(wctx, request(nil, 0, 0, true))
//		require.NoError(t, err)
//		require.Equal(t, len(msgs), int(resp.Pagination.Total))
//		require.ElementsMatch(t,
//			nullify.Fill(msgs),
//			nullify.Fill(resp.Prices),
//		)
//	})
//	t.Run("InvalidRequest", func(t *testing.T) {
//		_, err := keeper.PricesAll(wctx, nil)
//		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid request"))
//	})
//}
