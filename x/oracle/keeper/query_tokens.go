package keeper

import (
	"context"

	"github.com/ExocoreNetwork/exocore/x/oracle/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) TokenIndexes(goCtx context.Context, req *types.QueryTokenIndexesRequest) (*types.QueryTokenIndexesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	ret := k.GetTokens(ctx)
	return &types.QueryTokenIndexesResponse{
		TokenIndexes: ret,
	}, nil
}
