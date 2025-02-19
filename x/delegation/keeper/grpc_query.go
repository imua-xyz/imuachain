package keeper

import (
	"context"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	delegationtype "github.com/imua-xyz/imuachain/x/delegation/types"
)

var _ delegationtype.QueryServer = &Keeper{}

func (k *Keeper) QuerySingleDelegationInfo(ctx context.Context, req *delegationtype.SingleDelegationInfoReq) (*delegationtype.SingleDelegationInfoResponse, error) {
	c := sdk.UnwrapSDKContext(ctx)
	delegationAmounts, err := k.GetSingleDelegationInfo(c, strings.ToLower(req.StakerId), strings.ToLower(req.AssetId), req.OperatorAddr)
	if err != nil {
		return nil, err
	}
	// calculate the maximum undelegatable amount
	singleAmount, err := k.UndelegatableAmount(c, strings.ToLower(req.AssetId), req.OperatorAddr, delegationAmounts)
	if err != nil {
		return nil, err
	}
	return &delegationtype.SingleDelegationInfoResponse{
		DelegationAmounts:      delegationAmounts,
		MaxUndelegatableAmount: singleAmount,
	}, nil
}

func (k *Keeper) QueryDelegationInfo(ctx context.Context, info *delegationtype.DelegationInfoReq) (*delegationtype.QueryDelegationInfoResponse, error) {
	c := sdk.UnwrapSDKContext(ctx)
	return k.GetDelegationInfo(c, strings.ToLower(info.StakerId), strings.ToLower(info.AssetId))
}

func (k *Keeper) QueryUndelegations(ctx context.Context, req *delegationtype.UndelegationsReq) (*delegationtype.UndelegationRecordList, error) {
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
	c := sdk.UnwrapSDKContext(ctx)
	recordKey, err := k.GetUndelegationRecKey(c, strings.ToLower(req.StakerId), strings.ToLower(req.AssetId), req.UndelegationId)
	if err != nil {
		return nil, err
	}
	res := k.GetUndelegationHoldCount(c, recordKey)
	return &delegationtype.UndelegationHoldCountResponse{HoldCount: res}, nil
}

func (k Keeper) QueryAssociatedOperatorByStaker(ctx context.Context, req *delegationtype.QueryAssociatedOperatorByStakerReq) (*delegationtype.QueryAssociatedOperatorByStakerResponse, error) {
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
