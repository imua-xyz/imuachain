package keeper

import (
	"context"

	"github.com/imua-xyz/imuachain/x/appchain/coordinator/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ types.QueryServer = Keeper{}

// QueryParams is the implementation of the QueryServer method
func (k Keeper) QueryParams(
	goCtx context.Context,
	req *types.QueryParamsRequest,
) (*types.QueryParamsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	return &types.QueryParamsResponse{Params: k.GetParams(ctx)}, nil
}

// QuerySubscriberGenesis is the implementation of the QueryServer method
func (k Keeper) QuerySubscriberGenesis(
	goCtx context.Context,
	req *types.QuerySubscriberGenesisRequest,
) (*types.QuerySubscriberGenesisResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	genesis, found := k.GetSubscriberGenesis(ctx, req.Chain)
	if !found {
		return nil, status.Error(codes.NotFound, "subscriber genesis not found")
	}
	return &types.QuerySubscriberGenesisResponse{
		SubscriberGenesis: genesis,
	}, nil
}
