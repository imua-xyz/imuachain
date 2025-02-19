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

func (k Keeper) RecentMsgAll(goCtx context.Context, req *types.QueryAllRecentMsgRequest) (*types.QueryAllRecentMsgResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var recentMsgs []types.RecentMsg
	ctx := sdk.UnwrapSDKContext(goCtx)

	store := ctx.KVStore(k.storeKey)
	recentMsgStore := prefix.NewStore(store, types.KeyPrefix(types.RecentMsgKeyPrefix))

	pageRes, err := query.Paginate(recentMsgStore, req.Pagination, func(_ []byte, value []byte) error {
		var recentMsg types.RecentMsg
		if err := k.cdc.Unmarshal(value, &recentMsg); err != nil {
			return err
		}

		recentMsgs = append(recentMsgs, recentMsg)
		return nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllRecentMsgResponse{RecentMsg: recentMsgs, Pagination: pageRes}, nil
}

func (k Keeper) RecentMsg(goCtx context.Context, req *types.QueryGetRecentMsgRequest) (*types.QueryGetRecentMsgResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	val, found := k.GetRecentMsg(
		ctx,
		req.Block,
	)
	if !found {
		return nil, status.Error(codes.NotFound, "not found")
	}

	return &types.QueryGetRecentMsgResponse{RecentMsg: val}, nil
}
