package keeper

import (
	"context"
	"errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	assetstypes "github.com/imua-xyz/imuachain/x/assets/types"
	"github.com/imua-xyz/imuachain/x/oracle/types"
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

	stakerInfosResp, err := k.GetStakerInfos(ctx, req)
	if err != nil {
		return stakerInfosResp, err
	}
	version := k.GetNSTVersion(ctx, req.AssetId)
	stakerInfosResp.Version = version
	return stakerInfosResp, nil
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
	version := k.GetNSTVersion(ctx, req.AssetId)
	return &types.QueryStakerInfoResponse{Version: version, StakerInfo: &stakerInfo}, nil
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
	version := k.GetNSTVersion(ctx, req.AssetId)
	return &types.QueryStakerListResponse{Version: version, StakerList: &stakerList}, nil
}
