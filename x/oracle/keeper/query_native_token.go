package keeper

import (
	"context"
	"errors"

	assetstypes "github.com/ExocoreNetwork/exocore/x/assets/types"
	"github.com/ExocoreNetwork/exocore/x/oracle/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrInvalidRequest   = status.Error(codes.InvalidArgument, "invalid request")
	ErrUnsupportedAsset = errors.New("assetID doesn't represent any supported native restaking token")
)

func (k Keeper) StakerInfos(goCtx context.Context, req *types.QueryStakerInfosRequest) (*types.QueryStakerInfosResponse, error) {
	if req == nil {
		return nil, ErrInvalidRequest
	}
	if !assetstypes.IsNST(req.AssetId) {
		return nil, ErrUnsupportedAsset
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	return k.GetStakerInfos(ctx, req)
}

func (k Keeper) StakerInfo(goCtx context.Context, req *types.QueryStakerInfoRequest) (*types.QueryStakerInfoResponse, error) {
	if req == nil {
		return nil, ErrInvalidRequest
	}
	if !assetstypes.IsNST(req.AssetId) {
		return nil, ErrUnsupportedAsset
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	stakerInfo := k.GetStakerInfo(ctx, req.AssetId, req.StakerAddr)
	return &types.QueryStakerInfoResponse{StakerInfo: &stakerInfo}, nil
}

func (k Keeper) StakerList(goCtx context.Context, req *types.QueryStakerListRequest) (*types.QueryStakerListResponse, error) {
	if req == nil {
		return nil, ErrInvalidRequest
	}
	if !assetstypes.IsNST(req.AssetId) {
		return nil, ErrUnsupportedAsset
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	stakerList := k.GetStakerList(ctx, req.AssetId)
	return &types.QueryStakerListResponse{StakerList: &stakerList}, nil
}
