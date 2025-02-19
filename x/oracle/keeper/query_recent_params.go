//nolint:dupl
package keeper

import (
	"context"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/imua-xyz/imuachain/x/oracle/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) RecentParamsAll(goCtx context.Context, req *types.QueryAllRecentParamsRequest) (*types.QueryAllRecentParamsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var recentParamss []types.RecentParams
	ctx := sdk.UnwrapSDKContext(goCtx)

	store := ctx.KVStore(k.storeKey)
	recentParamsStore := prefix.NewStore(store, types.KeyPrefix(types.RecentParamsKeyPrefix))

	pageRes, err := query.Paginate(recentParamsStore, req.Pagination, func(_ []byte, value []byte) error {
		var recentParams types.RecentParams
		if err := k.cdc.Unmarshal(value, &recentParams); err != nil {
			return err
		}

		recentParamss = append(recentParamss, recentParams)
		return nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllRecentParamsResponse{RecentParams: recentParamss, Pagination: pageRes}, nil
}

func (k Keeper) RecentParams(goCtx context.Context, req *types.QueryGetRecentParamsRequest) (*types.QueryGetRecentParamsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	val, found := k.GetRecentParams(
		ctx,
		req.Block,
	)
	if !found {
		return nil, status.Error(codes.NotFound, "not found")
	}

	return &types.QueryGetRecentParamsResponse{RecentParams: val}, nil
}
