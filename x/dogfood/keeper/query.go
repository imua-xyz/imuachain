package keeper

import (
	"context"

	"github.com/imua-xyz/imuachain/x/dogfood/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ types.QueryServer = Keeper{}

func (k Keeper) Params(
	goCtx context.Context,
	req *types.QueryParamsRequest,
) (*types.QueryParamsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	return &types.QueryParamsResponse{Params: k.GetDogfoodParams(ctx)}, nil
}

func (k Keeper) OptOutsToFinish(
	goCtx context.Context,
	req *types.QueryOptOutsToFinishRequest,
) (*types.AccountAddresses, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	epoch := req.Epoch
	addresses := k.GetOptOutsToFinish(ctx, epoch)
	// TODO: consider converting this to a slice of strings?
	return &types.AccountAddresses{List: addresses}, nil
}

func (k Keeper) OperatorOptOutFinishEpoch(
	goCtx context.Context,
	req *types.QueryOperatorOptOutFinishEpochRequest,
) (*types.QueryOperatorOptOutFinishEpochResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	accAddr, err := sdk.AccAddressFromBech32(req.OperatorAccAddr)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid operator address")
	}
	epoch := k.GetOperatorOptOutFinishEpoch(ctx, accAddr)
	return &types.QueryOperatorOptOutFinishEpochResponse{Epoch: epoch}, nil
}

func (k Keeper) UndelegationsToMature(
	goCtx context.Context,
	req *types.QueryUndelegationsToMatureRequest,
) (*types.UndelegationRecordKeys, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	epoch := req.Epoch
	keys := k.GetUndelegationsToMature(ctx, epoch)
	return &types.UndelegationRecordKeys{List: keys}, nil
}

func (k Keeper) UndelegationMaturityEpoch(
	goCtx context.Context,
	req *types.QueryUndelegationMaturityEpochRequest,
) (*types.QueryUndelegationMaturityEpochResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	epoch, found := k.GetUndelegationMaturityEpoch(ctx, []byte(req.RecordKey))
	if !found {
		return nil, status.Error(codes.NotFound, "undelegation record not found")
	}
	return &types.QueryUndelegationMaturityEpochResponse{Epoch: epoch}, nil
}

func (k Keeper) QueryValidator(
	goCtx context.Context,
	req *types.QueryValidatorRequest,
) (*types.ExocoreValidator, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	consAddress := req.ConsAddr
	consAddressBytes, err := sdk.ConsAddressFromBech32(consAddress)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid consensus address")
	}
	validator, found := k.GetExocoreValidator(ctx, consAddressBytes)
	if !found {
		return nil, status.Error(codes.NotFound, "validator not found")
	}
	return &validator, nil
}
