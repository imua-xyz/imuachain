package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	delegationtype "github.com/imua-xyz/imuachain/x/delegation/types"
)

var _ delegationtype.QueryServer = &Keeper{}

func (k *Keeper) QuerySingleDelegationInfo(ctx context.Context, req *delegationtype.SingleDelegationInfoReq) (*delegationtype.DelegationAmounts, error) {
	c := sdk.UnwrapSDKContext(ctx)
	return k.GetSingleDelegationInfo(c, req.StakerID, req.AssetID, req.OperatorAddr)
}

func (k *Keeper) QueryDelegationInfo(ctx context.Context, info *delegationtype.DelegationInfoReq) (*delegationtype.QueryDelegationInfoResponse, error) {
	c := sdk.UnwrapSDKContext(ctx)
	return k.GetDelegationInfo(c, info.StakerID, info.AssetID)
}

func (k *Keeper) QueryUndelegations(ctx context.Context, req *delegationtype.UndelegationsReq) (*delegationtype.UndelegationRecordList, error) {
	c := sdk.UnwrapSDKContext(ctx)
	undelegations, err := k.GetStakerUndelegationRecords(c, req.StakerID, req.AssetID)
	if err != nil {
		return nil, err
	}
	return &delegationtype.UndelegationRecordList{
		Undelegations: undelegations,
	}, nil
}

func (k *Keeper) QueryUndelegationsByHeight(ctx context.Context, req *delegationtype.UndelegationsByHeightReq) (*delegationtype.UndelegationRecordList, error) {
	c := sdk.UnwrapSDKContext(ctx)
	undelegations, err := k.GetPendingUndelegationRecords(c, req.BlockHeight)
	if err != nil {
		return nil, err
	}
	return &delegationtype.UndelegationRecordList{
		Undelegations: undelegations,
	}, nil
}

func (k Keeper) QueryUndelegationHoldCount(ctx context.Context, req *delegationtype.UndelegationHoldCountReq) (*delegationtype.UndelegationHoldCountResponse, error) {
	c := sdk.UnwrapSDKContext(ctx)
	res := k.GetUndelegationHoldCount(c, []byte(req.RecordKey))
	return &delegationtype.UndelegationHoldCountResponse{HoldCount: res}, nil
}

func (k Keeper) QueryAssociatedOperatorByStaker(ctx context.Context, req *delegationtype.QueryAssociatedOperatorByStakerReq) (*delegationtype.QueryAssociatedOperatorByStakerResponse, error) {
	c := sdk.UnwrapSDKContext(ctx)
	operator, err := k.GetAssociatedOperator(c, req.StakerID)
	if err != nil {
		return nil, err
	}
	return &delegationtype.QueryAssociatedOperatorByStakerResponse{
		Operator: operator,
	}, nil
}
