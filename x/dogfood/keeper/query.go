package keeper

import (
	"context"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/imua-xyz/imuachain/x/dogfood/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type Querier struct {
	Keeper
}

var _ types.QueryServer = &Querier{}

func NewQueryServer(keeper Keeper) types.QueryServer {
	return &Querier{Keeper: keeper}
}

func (q Querier) Params(
	goCtx context.Context,
	req *types.QueryParamsRequest,
) (*types.QueryParamsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	return &types.QueryParamsResponse{Params: q.Keeper.GetDogfoodParams(ctx)}, nil
}

func (q Querier) OptOutsToFinish(
	goCtx context.Context,
	req *types.QueryOptOutsToFinishRequest,
) (*types.AccountAddresses, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	epoch := req.Epoch
	addresses := q.Keeper.GetOptOutsToFinish(ctx, epoch)
	// TODO: consider converting this to a slice of strings?
	return &types.AccountAddresses{List: addresses}, nil
}

func (q Querier) OperatorOptOutFinishEpoch(
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
	epoch := q.Keeper.GetOperatorOptOutFinishEpoch(ctx, accAddr)
	return &types.QueryOperatorOptOutFinishEpochResponse{Epoch: epoch}, nil
}

func (q Querier) UndelegationsToMature(
	goCtx context.Context,
	req *types.QueryUndelegationsToMatureRequest,
) (*types.UndelegationRecordKeys, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	epoch := req.Epoch
	keys := q.Keeper.GetUndelegationsToMature(ctx, epoch)
	return &types.UndelegationRecordKeys{List: keys}, nil
}

func (q Querier) UndelegationMaturityEpoch(
	goCtx context.Context,
	req *types.QueryUndelegationMaturityEpochRequest,
) (*types.QueryUndelegationMaturityEpochResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	epoch, found := q.Keeper.GetUndelegationMaturityEpoch(ctx, []byte(req.RecordKey))
	if !found {
		return nil, status.Error(codes.NotFound, "undelegation record not found")
	}
	return &types.QueryUndelegationMaturityEpochResponse{Epoch: epoch}, nil
}

func (q Querier) Validator(
	goCtx context.Context,
	req *types.QueryValidatorRequest,
) (*types.QueryValidatorResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	consAddress := req.ConsAddr
	consAddressBytes, err := sdk.ConsAddressFromBech32(consAddress)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid consensus address")
	}
	validator, found := q.Keeper.GetImuachainValidator(ctx, consAddressBytes)
	if !found {
		return nil, status.Error(codes.NotFound, "validator not found")
	}
	return &types.QueryValidatorResponse{Validator: &validator}, nil
}

func (q Querier) Validators(
	goCtx context.Context,
	req *types.QueryAllValidatorsRequest,
) (*types.QueryAllValidatorsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	store := ctx.KVStore(q.Keeper.storeKey)
	valStore := prefix.NewStore(store, []byte{types.ImuachainValidatorBytePrefix})

	validators, pageRes, err := query.GenericFilteredPaginate(
		q.Keeper.cdc, valStore, req.Pagination, func(_ []byte, val *types.ImuachainValidator) (*types.ImuachainValidator, error) {
			return val, nil
		}, func() *types.ImuachainValidator {
			return &types.ImuachainValidator{}
		})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// convert pointer to value
	vals := make([]types.ImuachainValidator, len(validators))
	for i, val := range validators {
		vals[i] = *val
	}

	return &types.QueryAllValidatorsResponse{Validators: vals, Pagination: pageRes}, nil
}
