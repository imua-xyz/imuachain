package keeper_test

import (
	"fmt"
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	keepertest "github.com/imua-xyz/imuachain/testutil/keeper"
	"github.com/imua-xyz/imuachain/x/oracle/keeper"
	"github.com/imua-xyz/imuachain/x/oracle/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestQueryStakerInfosPaginated(t *testing.T) {
	assetID := string(keeper.NSTETHAssetIDMainnet)
	keeper, ctx := keepertest.OracleKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	msgs := createNStakerInfos(keeper, ctx, assetID, 5)

	request := func(assetID string, next []byte, offset, limit uint64, total bool) *types.QueryStakerInfosRequest {
		return &types.QueryStakerInfosRequest{
			AssetId: assetID,
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
			resp, err := keeper.StakerInfos(wctx, request(assetID, nil, uint64(i), uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.StakerInfos), step)
			require.Subset(t,
				msgs,
				resp.StakerInfos,
			)
		}
		resp, err := keeper.StakerInfos(wctx, request(assetID, nil, uint64(len(msgs)), 0, false))
		require.Empty(t, resp.StakerInfos)
		require.Equal(t, uint64(len(msgs)), resp.Pagination.Total)
		require.NoError(t, err)
	})
	t.Run("ByKey", func(t *testing.T) {
		step := 2
		var next []byte
		for i := 0; i < len(msgs); i += step {
			resp, err := keeper.StakerInfos(wctx, request(assetID, next, 0, uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.StakerInfos), step)
			require.Subset(t,
				msgs,
				resp.StakerInfos,
			)
			next = resp.Pagination.NextKey
		}
	})
	t.Run("Total", func(t *testing.T) {
		resp, err := keeper.StakerInfos(wctx, request(assetID, nil, 0, 0, true))
		require.NoError(t, err)
		require.Equal(t, len(msgs), int(resp.Pagination.Total))
		require.ElementsMatch(t,
			msgs,
			resp.StakerInfos,
		)
	})
	t.Run("InvalidRequest", func(t *testing.T) {
		_, err := keeper.StakerInfos(wctx, nil)
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid request"))
	})
}

func createNStakerInfos(keeper *keeper.Keeper, ctx sdk.Context, assetID string, n int) []*types.StakerInfo {
	ret := make([]*types.StakerInfo, 0, n)
	for i := 0; i < n; i++ {
		ret = append(ret, &types.StakerInfo{
			StakerAddr:          fmt.Sprintf("Staker_%d", i),
			StakerIndex:         int64(i),
			ValidatorPubkeyList: []string{strconv.Itoa(i + 1)},
			BalanceList: []*types.BalanceInfo{
				{
					RoundID: 0,
					Block:   0,
					Index:   0,
					Balance: 32,
					Change:  types.Action_ACTION_DEPOSIT,
				},
			},
		})
	}
	keeper.SetStakerInfos(ctx, assetID, ret)
	return ret
}
