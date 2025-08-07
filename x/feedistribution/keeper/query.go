package keeper

import (
	"context"
	"strings"

	"github.com/ethereum/go-ethereum/common"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	sdk "github.com/cosmos/cosmos-sdk/types"
	assetstype "github.com/imua-xyz/imuachain/x/assets/types"
	"github.com/imua-xyz/imuachain/x/feedistribution/types"
)

var _ types.QueryServer = Keeper{}

// AVSRewardAsset queries the specific reward asset for an AVS.
func (k Keeper) AVSRewardAsset(ctx context.Context, req *types.QueryAVSRewardAssetRequest) (*types.QueryAVSRewardAssetResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}
	c := sdk.UnwrapSDKContext(ctx)
	_, _, err := assetstype.ValidateID(req.AssetId, false, false)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid assetID,err:%v", err)
	}
	if !common.IsHexAddress(req.Avs) {
		return nil, status.Errorf(codes.InvalidArgument, "avs should be an EVM address,AVS:%s", req.Avs)
	}
	assetInfo, err := k.GetAVSRewardAssetInfo(c, strings.ToLower(req.Avs), strings.ToLower(req.AssetId))
	if err != nil {
		return nil, err
	}
	return &types.QueryAVSRewardAssetResponse{AvsRewardAsset: assetInfo}, nil
}

// RewardAssetsByAVS queries all reward assets for an AVS.
func (k Keeper) RewardAssetsByAVS(ctx context.Context, req *types.AVSRequest) (*types.QueryRewardAssetsByAVSResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}
	c := sdk.UnwrapSDKContext(ctx)
	allRewardAssets, err := k.GetAllRewardAssetsByAVS(c, strings.ToLower(req.Avs))
	if err != nil {
		return nil, err
	}
	return &types.QueryRewardAssetsByAVSResponse{RewardAssets: &allRewardAssets}, nil
}

// AVSRewardParam queries the reward param for an AVS.
func (k Keeper) AVSRewardParam(ctx context.Context, req *types.AVSRequest) (*types.QueryAVSRewardParamResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}
	c := sdk.UnwrapSDKContext(ctx)
	if !common.IsHexAddress(req.Avs) {
		return nil, status.Errorf(codes.InvalidArgument, "avs should be an EVM address,AVS:%s", req.Avs)
	}
	rewardParam, err := k.GetAVSRewardParam(c, req.Avs)
	if err != nil {
		return nil, err
	}
	return &types.QueryAVSRewardParamResponse{AvsRewardParam: rewardParam}, nil
}

// AVSCommunityPool queries the community reward pool for an AVS.
func (k Keeper) AVSCommunityPool(ctx context.Context, req *types.AVSRequest) (*types.QueryAVSCommunityPoolResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}
	c := sdk.UnwrapSDKContext(ctx)
	if !common.IsHexAddress(req.Avs) {
		return nil, status.Errorf(codes.InvalidArgument, "avs should be an EVM address,AVS:%s", req.Avs)
	}
	feePool, err := k.GetAVSFeePool(c, req.Avs)
	if err != nil {
		return nil, err
	}
	return &types.QueryAVSCommunityPoolResponse{FeePool: &feePool}, nil
}

// AVSRewardDistribution queries the distribution information for an AVS.
func (k Keeper) AVSRewardDistribution(ctx context.Context, req *types.AVSRequest) (*types.QueryAVSRewardDistributionResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}
	c := sdk.UnwrapSDKContext(ctx)
	if !common.IsHexAddress(req.Avs) {
		return nil, status.Errorf(codes.InvalidArgument, "avs should be an EVM address,AVS:%s", req.Avs)
	}
	avsRewardDistribution, err := k.GetAVSRewardDistribution(c, req.Avs)
	if err != nil {
		return nil, err
	}
	return &types.QueryAVSRewardDistributionResponse{AvsRewardDistribution: avsRewardDistribution}, nil
}

// OperatorOutstandingRewards queries the outstanding rewards for an operator.
func (k Keeper) OperatorOutstandingRewards(ctx context.Context, req *types.OperatorAVSRequest) (*types.QueryOperatorOutstandingRewardsResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}
	c := sdk.UnwrapSDKContext(ctx)
	if !common.IsHexAddress(req.Avs) {
		return nil, status.Errorf(codes.InvalidArgument, "avs should be an EVM address,AVS:%s", req.Avs)
	}
	_, err := sdk.AccAddressFromBech32(req.Operator)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid operator address,err:%v", err)
	}
	outstandingRewards, err := k.GetOperatorOutstandingRewards(c, req.Operator, strings.ToLower(req.Avs))
	if err != nil {
		return nil, err
	}
	return &types.QueryOperatorOutstandingRewardsResponse{OperatorOutstandingRewards: &outstandingRewards}, nil
}

// StakerOutstandingRewards queries the outstanding rewards for a staker.
func (k Keeper) StakerOutstandingRewards(ctx context.Context, req *types.QueryStakerOutstandingRewardsRequest) (*types.QueryStakerOutstandingRewardsResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}
	c := sdk.UnwrapSDKContext(ctx)
	_, _, err := assetstype.ValidateID(req.StakerId, false, false)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid stakerID,err:%v", err)
	}
	if !common.IsHexAddress(req.Avs) {
		return nil, status.Errorf(codes.InvalidArgument, "avs should be an EVM address,AVS:%s", req.Avs)
	}
	outstandingRewards, err := k.GetStakerOutstandingRewards(c, strings.ToLower(req.StakerId), strings.ToLower(req.Avs))
	if err != nil {
		return nil, err
	}
	return &types.QueryStakerOutstandingRewardsResponse{StakerOutstandingRewards: &outstandingRewards}, nil
}

// StakeChangeDelegations queries the delegations whose stake has changed.
func (k Keeper) StakeChangeDelegations(ctx context.Context, req *types.QueryStakeChangeDelegationsRequest) (*types.QueryStakeChangeDelegationsResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}
	c := sdk.UnwrapSDKContext(ctx)
	_, err := sdk.AccAddressFromBech32(req.Operator)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid operator address,err:%v", err)
	}
	_, _, err = assetstype.ValidateID(req.AssetId, false, false)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid assetID,err:%v", err)
	}
	stakeChangeDelegations, err := k.GetStakeChangedDelegations(c, req.EpochIdentifier, req.Operator, strings.ToLower(req.AssetId))
	if err != nil {
		return nil, err
	}
	return &types.QueryStakeChangeDelegationsResponse{DelegationChangeInfo: &stakeChangeDelegations}, nil
}

// DelegationStartingInfo queries the delegation starting information.
func (k Keeper) DelegationStartingInfo(ctx context.Context, req *types.QueryDelegationStartingInfoRequest) (*types.QueryDelegationStartingInfoResponse, error) {
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
	_, err = sdk.AccAddressFromBech32(req.Operator)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid operator address,err:%v", err)
	}
	delegationKey := assetstype.GetJoinedStoreKey(strings.ToLower(req.StakerId), strings.ToLower(req.AssetId), req.Operator)
	delegationStartingInfo, err := k.GetDelegationStartingInfo(c, string(delegationKey), req.EpochIdentifier)
	if err != nil {
		return nil, err
	}
	return &types.QueryDelegationStartingInfoResponse{DelegationStartingInfo: &delegationStartingInfo}, nil
}

// OperatorHistoricalRewards queries the operator historical rewards.
func (k Keeper) OperatorHistoricalRewards(ctx context.Context, req *types.QueryOperatorHistoricalRewardsRequest) (*types.QueryOperatorHistoricalRewardsResponse, error) {
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
	historicalRewards, err := k.GetOperatorHistoricalReward(c, req.Operator, strings.ToLower(req.AssetId),
		req.EpochIdentifier, req.Period)
	if err != nil {
		return nil, err
	}
	return &types.QueryOperatorHistoricalRewardsResponse{OperatorHistoricalRewards: &historicalRewards}, nil
}

// AllOperatorHistoricalRewards queries the operator historical rewards for all periods.
func (k Keeper) AllOperatorHistoricalRewards(ctx context.Context, req *types.QueryAllOperatorHistoricalRewardsRequest) (*types.QueryAllOperatorHistoricalRewardsResponse, error) {
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
	historicalRewards, err := k.OperatorRewardsForAllPeriods(c, req.Operator, strings.ToLower(req.AssetId), req.EpochIdentifier)
	if err != nil {
		return nil, err
	}
	return &types.QueryAllOperatorHistoricalRewardsResponse{AllOperatorHistoricalRewards: historicalRewards}, nil
}

// OperatorCurrentRewards queries the operator current rewards.
func (k Keeper) OperatorCurrentRewards(ctx context.Context, req *types.QueryOperatorCurrentRewardsRequest) (*types.QueryOperatorCurrentRewardsResponse, error) {
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
	currentRewards, err := k.GetOperatorCurrentRewards(c, req.Operator, strings.ToLower(req.AssetId),
		req.EpochIdentifier)
	if err != nil {
		return nil, err
	}
	return &types.QueryOperatorCurrentRewardsResponse{OperatorCurrentRewards: &currentRewards}, nil
}

// OperatorAccumulatedCommission queries the operator accumulated commission.
func (k Keeper) OperatorAccumulatedCommission(ctx context.Context, req *types.OperatorAVSRequest) (*types.QueryOperatorAccumulatedCommissionResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}
	c := sdk.UnwrapSDKContext(ctx)
	_, err := sdk.AccAddressFromBech32(req.Operator)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid operator address,err:%v", err)
	}
	if !common.IsHexAddress(req.Avs) {
		return nil, status.Errorf(codes.InvalidArgument, "avs should be an EVM address,AVS:%s", req.Avs)
	}
	commission, err := k.GetOperatorAccumulatedCommission(c, req.Operator, strings.ToLower(req.Avs))
	if err != nil {
		return nil, err
	}
	return &types.QueryOperatorAccumulatedCommissionResponse{OperatorAccumulatedCommission: &commission}, nil
}

// OperatorSlashEvent queries the operator slash event.
func (k Keeper) OperatorSlashEvent(ctx context.Context, req *types.QueryOperatorSlashEventRequest) (*types.QueryOperatorSlashEventResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}
	c := sdk.UnwrapSDKContext(ctx)
	_, _, err := assetstype.ValidateID(req.AssetId, false, false)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid assetID,err:%v", err)
	}
	slashEvent, err := k.GetOperatorSlashEvent(c, req.Operator, strings.ToLower(req.AssetId),
		req.EpochIdentifier, req.EpochNumber, req.BlockHeight)
	if err != nil {
		return nil, err
	}
	return &types.QueryOperatorSlashEventResponse{OperatorSlashEvent: &slashEvent}, nil
}

// OperatorSlashEvents queries the operator slash events.
func (k Keeper) OperatorSlashEvents(ctx context.Context, req *types.QueryOperatorSlashEventsRequest) (*types.QueryOperatorSlashEventsResponse, error) {
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
	slashEvents, err := k.GetOperatorSlashEvents(c, req.Operator, strings.ToLower(req.AssetId), req.EpochIdentifier)
	if err != nil {
		return nil, err
	}
	return &types.QueryOperatorSlashEventsResponse{OperatorSlashEvents: slashEvents}, nil
}

// StakerUnclaimedRewards queries the unclaimed rewards for a staker.
func (k Keeper) StakerUnclaimedRewards(ctx context.Context, req *types.QueryStakerUnclaimedRewardsRequest) (*types.QueryStakerUnclaimedRewardsResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}
	c := sdk.UnwrapSDKContext(ctx)
	_, _, err := assetstype.ValidateID(req.StakerId, false, false)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid stakerID,err:%v", err)
	}
	unclaimedRewards, err := k.GetStakerUnclaimedRewards(c, strings.ToLower(req.StakerId))
	if err != nil {
		return nil, err
	}
	return &types.QueryStakerUnclaimedRewardsResponse{Rewards: unclaimedRewards}, nil
}
