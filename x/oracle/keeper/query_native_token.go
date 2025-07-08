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
	_, chainID, err := assetstypes.ParseID(req.AssetId)
	if err != nil {
		return nil, ErrInvalidRequest
	}
	// #nosec G115
	versions, found := k.GetNSTVersionsFromAssetID(ctx, req.AssetId)
	if !found {
		return nil, status.Error(codes.NotFound, "versions not found for the given asset ID")
	}
	stakerInfo := k.GetStakerInfo(ctx, chainID, req.StakerAddr)
	return &types.QueryStakerInfoResponse{Version: &versions, StakerInfo: &stakerInfo}, nil
}

func (k Keeper) StakerList(goCtx context.Context, req *types.QueryStakerListRequest) (*types.QueryStakerListResponse, error) {
	if req == nil {
		return nil, ErrInvalidRequest
	}
	if !assetstypes.IsNST(req.AssetId) {
		return nil, ErrUnsupportedAsset
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	//#nosec G115
	versions, found := k.GetNSTVersionsFromAssetID(ctx, req.AssetId)
	if !found {
		return nil, status.Error(codes.NotFound, "versions not found for the given asset ID")
	}

	stakerList := k.GetStakerList(ctx, req.AssetId, 0)
	return &types.QueryStakerListResponse{Version: versions.Version.Version, StakerList: &stakerList}, nil
}
