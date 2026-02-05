package keeper

import (
	"context"
	"strings"

	"github.com/imua-xyz/imuachain/utils"
	assetstype "github.com/imua-xyz/imuachain/x/assets/types"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
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
	_, _, err := assetstype.ValidateID(req.StakerId, false, false)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid stakerID,err:%v", err)
	}
	_, _, err = assetstype.ValidateID(req.AssetId, false, false)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid assetID,err:%v", err)
	}
	_, err = sdk.AccAddressFromBech32(req.OperatorAddr)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid operator address,err:%v", err)
	}
	delegationInfo, stakingAssetAmount, _, err := k.GetDelegationInfoWithAmounts(c, strings.ToLower(req.StakerId), strings.ToLower(req.AssetId), req.OperatorAddr)
	if err != nil {
		return nil, err
	}
	return &delegationtype.SingleDelegationInfoResponse{
		SingleDelegationInfo: &delegationtype.SingleDelegationInfo{
			DelegationAmounts:      delegationInfo,
			MaxUndelegatableAmount: stakingAssetAmount,
		},
	}, nil
}

func (k *Keeper) QueryDelegationInfo(ctx context.Context, req *delegationtype.DelegationInfoReq) (*delegationtype.QueryDelegationInfoResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}
	c := sdk.UnwrapSDKContext(ctx)
	_, _, err := assetstype.ValidateID(req.StakerId, false, false)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid stakerID,err:%v", err)
	}
	_, _, err = assetstype.ValidateID(req.AssetId, false, false)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid assetID,err:%v", err)
	}
	return k.GetDelegationInfo(c, strings.ToLower(req.StakerId), strings.ToLower(req.AssetId))
}

func (k *Keeper) QueryUndelegations(ctx context.Context, req *delegationtype.UndelegationsReq) (*delegationtype.UndelegationRecordList, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}
	c := sdk.UnwrapSDKContext(ctx)
	_, _, err := assetstype.ValidateID(req.StakerId, false, false)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid stakerID,err:%v", err)
	}
	_, _, err = assetstype.ValidateID(req.AssetId, false, false)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid assetID,err:%v", err)
	}
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
	if req.EpochNumber < 0 {
		return nil, status.Errorf(codes.InvalidArgument, "negative epoch number:%d", req.EpochNumber)
	}
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
	_, _, err := assetstype.ValidateID(req.StakerId, false, false)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid stakerID,err:%v", err)
	}
	_, _, err = assetstype.ValidateID(req.AssetId, false, false)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid assetID,err:%v", err)
	}
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
	_, _, err := assetstype.ValidateID(req.StakerId, false, false)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid stakerID,err:%v", err)
	}
	return &delegationtype.QueryAssociatedOperatorByStakerResponse{
		Operator: k.GetAssociatedOperator(c, strings.ToLower(req.StakerId)),
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
	_, _, err := assetstype.ValidateID(req.AssetId, false, false)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid assetID,err:%v", err)
	}
	_, err = sdk.AccAddressFromBech32(req.Operator)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid operator address,err:%v", err)
	}
	keyPrefix := utils.AppendMany(
		delegationtype.KeyPrefixStakersByOperator,
		utils.GetJoinedStoreKeyForPrefix(req.Operator, strings.ToLower(req.AssetId)),
	)
	// prefix.NewStore returns keys stripped of the prefix.
	// sdk.KVStorePrefixIterator returns keys with the prefix.
	store := prefix.NewStore(c.KVStore(k.storeKey), keyPrefix)
	var stakers []string
	pageRes, err := query.Paginate(store, req.Pagination, func(key []byte, _ []byte) error {
		// the key is relative to the prefix, so we simply append it to the list.
		// the value is []byte{1} which we can ignore.
		stakers = append(stakers, string(key))
		return nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &delegationtype.QueryDelegatedStakersByOperatorResponse{
		Stakers:    stakers,
		Pagination: pageRes,
	}, nil
}

func (k Keeper) QueryParams(goCtx context.Context, req *delegationtype.QueryParamsRequest) (*delegationtype.QueryParamsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	return &delegationtype.QueryParamsResponse{Params: k.GetParams(ctx)}, nil
}
