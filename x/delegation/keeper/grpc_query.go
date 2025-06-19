package keeper

import (
	"context"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	delegationtype "github.com/imua-xyz/imuachain/x/delegation/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ delegationtype.QueryServer = &Keeper{}

func (k *Keeper) QuerySingleDelegationInfo(ctx context.Context, req *delegationtype.SingleDelegationInfoReq) (*delegationtype.SingleDelegationInfoResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}
	c := sdk.UnwrapSDKContext(ctx)
	delegationInfo, delegatedAmount, err := k.GetDelegationInfoWithAmount(c, strings.ToLower(req.StakerId), strings.ToLower(req.AssetId), req.OperatorAddr)
	if err != nil {
		return nil, err
	}
	return &delegationtype.SingleDelegationInfoResponse{
		SingleDelegationInfo: &delegationtype.SingleDelegationInfo{
			DelegationAmounts:      delegationInfo,
			MaxUndelegatableAmount: delegatedAmount,
		},
	}, nil
}

func (k *Keeper) QueryDelegationInfo(ctx context.Context, req *delegationtype.DelegationInfoReq) (*delegationtype.QueryDelegationInfoResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}
	c := sdk.UnwrapSDKContext(ctx)
	return k.GetDelegationInfo(c, strings.ToLower(req.StakerId), strings.ToLower(req.AssetId))
}

func (k *Keeper) QueryUndelegations(ctx context.Context, req *delegationtype.UndelegationsReq) (*delegationtype.UndelegationRecordList, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}
	c := sdk.UnwrapSDKContext(ctx)
	undelegations, err := k.GetStakerUndelegationRecords(c, strings.ToLower(req.StakerId), strings.ToLower(req.AssetId))
	if err != nil {
		return nil, err
	}
	return &delegationtype.UndelegationRecordList{
		Undelegations: undelegations,
	}, nil
}

func (k *Keeper) QueryUndelegationsByEpochInfo(ctx context.Context, req *delegationtype.UndelegationsByEpochInfoReq) (*delegationtype.UndelegationRecordList, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}
	c := sdk.UnwrapSDKContext(ctx)
	undelegations, err := k.GetUnCompletableUndelegations(c, req.EpochIdentifier, req.EpochNumber)
	if err != nil {
		return nil, err
	}
	return &delegationtype.UndelegationRecordList{
		Undelegations: undelegations,
	}, nil
}

func (k Keeper) QueryUndelegationHoldCount(ctx context.Context, req *delegationtype.UndelegationHoldCountReq) (*delegationtype.UndelegationHoldCountResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}
	c := sdk.UnwrapSDKContext(ctx)
	recordKey, err := k.GetUndelegationRecKey(c, strings.ToLower(req.StakerId), strings.ToLower(req.AssetId), req.UndelegationId)
	if err != nil {
		return nil, err
	}
	res := k.GetUndelegationHoldCount(c, recordKey)
	return &delegationtype.UndelegationHoldCountResponse{HoldCount: res}, nil
}

func (k Keeper) QueryAssociatedOperatorByStaker(ctx context.Context, req *delegationtype.QueryAssociatedOperatorByStakerReq) (*delegationtype.QueryAssociatedOperatorByStakerResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}
	c := sdk.UnwrapSDKContext(ctx)
	operator, err := k.GetAssociatedOperator(c, strings.ToLower(req.StakerId))
	if err != nil {
		return nil, err
	}
	return &delegationtype.QueryAssociatedOperatorByStakerResponse{
		Operator: operator,
	}, nil
}

func (k Keeper) QueryAssociatedStakersByOperator(ctx context.Context, req *delegationtype.QueryAssociatedStakersByOperatorReq) (*delegationtype.QueryAssociatedStakersByOperatorResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}
	c := sdk.UnwrapSDKContext(ctx)
	stakers, err := k.GetAssociatedStakers(c, req.Operator)
	if err != nil {
		return nil, err
	}
	return &delegationtype.QueryAssociatedStakersByOperatorResponse{
		Stakers: stakers,
	}, nil
}

func (k Keeper) QueryDelegatedStakersByOperator(ctx context.Context, req *delegationtype.QueryDelegatedStakersByOperatorReq) (*delegationtype.QueryDelegatedStakersByOperatorResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}
	c := sdk.UnwrapSDKContext(ctx)
	stakers, err := k.GetStakersByOperator(c, req.Operator, strings.ToLower(req.AssetId))
	if err != nil {
		return nil, err
	}
	return &delegationtype.QueryDelegatedStakersByOperatorResponse{
		Count:   uint64(len(stakers.Stakers)),
		Stakers: stakers.Stakers,
	}, nil
}

func (k Keeper) QueryParams(goCtx context.Context, req *delegationtype.QueryParamsRequest) (*delegationtype.QueryParamsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	return &delegationtype.QueryParamsResponse{Params: k.GetParams(ctx)}, nil
}
