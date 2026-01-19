package keeper

import (
	"context"
	"strings"

	"github.com/imua-xyz/imuachain/utils"

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
	assetInfo, err := k.GetAVSRewardAsset(c, strings.ToLower(req.Avs), strings.ToLower(req.AssetId))
	if err != nil {
		return nil, err
	}
	return &types.QueryAVSRewardAssetResponse{AvsRewardAsset: assetInfo}, nil
}

// AVSRewardAssetByDenom queries the specific AVS reward asset by the denomination.
func (k Keeper) AVSRewardAssetByDenom(ctx context.Context, req *types.QueryAVSRewardAssetByDenomRequest) (*types.QueryAVSRewardAssetByDenomResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}
	c := sdk.UnwrapSDKContext(ctx)
	if !common.IsHexAddress(req.Avs) {
		return nil, status.Errorf(codes.InvalidArgument, "avs should be an EVM address,AVS:%s", req.Avs)
	}
	err := types.ValidateRewardAssetDenomination(req.Denomination)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "denomination should be a valid denomination,denomination:%s,err:%s", req.Denomination, err)
	}
	_, assetInfo, err := k.GetAVSRewardAssetByDenomination(c, strings.ToLower(req.Avs), req.Denomination)
	if err != nil {
		return nil, err
	}
	return &types.QueryAVSRewardAssetByDenomResponse{AvsRewardAsset: assetInfo}, nil
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

// OperatorUnclaimedRewards queries the unclaimed rewards for an operator.
func (k Keeper) OperatorUnclaimedRewards(ctx context.Context, req *types.OperatorAVSRequest) (*types.QueryOperatorUnclaimedRewardsResponse, error) {
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
	avsAddr := strings.ToLower(req.Avs)
	unclaimedRewards, err := k.GetOperatorUnclaimedRewards(c, req.Operator, avsAddr)
	if err != nil {
		return nil, err
	}
	normalizedRewards, err := k.NormalizeRewardDecCoins(c, avsAddr, unclaimedRewards.OutstandingRewards)
	if err != nil {
		return nil, err
	}
	unclaimedRewards.OutstandingRewards = normalizedRewards

	// normalize the rewards from compounding
	for i, compoundingRewardsPerSymbol := range unclaimedRewards.RewardsFromCompounding {
		normalizedRewardsPerSymbol, err := k.BatchNormalizeRewardDecimals(c, compoundingRewardsPerSymbol.Rewards)
		if err != nil {
			return nil, err
		}
		unclaimedRewards.RewardsFromCompounding[i].Rewards = normalizedRewardsPerSymbol
	}
	return &types.QueryOperatorUnclaimedRewardsResponse{OperatorUnclaimedRewards: &unclaimedRewards}, nil
}

// StakerClaimedRewards queries the claimed rewards for a staker.
func (k Keeper) StakerClaimedRewards(ctx context.Context, req *types.QueryStakerClaimedRewardsRequest) (*types.QueryStakerClaimedRewardsResponse, error) {
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
	avsAddr := strings.ToLower(req.Avs)
	claimedRewards, err := k.GetStakerClaimedRewards(c, strings.ToLower(req.StakerId), avsAddr)
	if err != nil {
		return nil, err
	}
	normalizedOutstandingRewards, err := k.NormalizeRewardDecCoins(c, avsAddr, claimedRewards.OutstandingRewards)
	if err != nil {
		return nil, err
	}
	normalizedWithdrawnRewards, err := k.NormalizeRewardDecCoins(c, avsAddr, claimedRewards.WithdrawnRewards)
	if err != nil {
		return nil, err
	}
	claimedRewards.OutstandingRewards = normalizedOutstandingRewards
	claimedRewards.WithdrawnRewards = normalizedWithdrawnRewards
	return &types.QueryStakerClaimedRewardsResponse{StakerClaimedRewards: &claimedRewards}, nil
}

// StakerAllClaimedRewards queries all claimed rewards for a staker.
func (k Keeper) StakerAllClaimedRewards(ctx context.Context, req *types.QueryStakerAllClaimedRewardsRequest) (*types.QueryStakerAllClaimedRewardsResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}
	c := sdk.UnwrapSDKContext(ctx)
	_, _, err := assetstype.ValidateID(req.StakerId, false, false)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid stakerID,err:%v", err)
	}

	allClaimedRewards, err := k.GetStakerAllClaimedRewards(c, strings.ToLower(req.StakerId))
	if err != nil {
		return nil, err
	}
	normalizedRewards, err := k.BatchNormalizeClaimedRewardDecimals(c, allClaimedRewards)
	if err != nil {
		return nil, err
	}
	return &types.QueryStakerAllClaimedRewardsResponse{Rewards: normalizedRewards}, nil
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
	delegationKey := utils.GetJoinedStoreKey(strings.ToLower(req.StakerId), strings.ToLower(req.AssetId), req.Operator)
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

// OperatorCommission queries the operator commission.
func (k Keeper) OperatorCommission(ctx context.Context, req *types.OperatorAVSRequest) (*types.QueryOperatorCommissionResponse, error) {
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
	avsAddr := strings.ToLower(req.Avs)
	commission, err := k.GetOperatorCommission(c, req.Operator, avsAddr)
	if err != nil {
		return nil, err
	}
	normalizedUnwithdrawnCommission, err := k.NormalizeRewardDecCoins(c, avsAddr, commission.UnwithdrawnCommission)
	if err != nil {
		return nil, err
	}
	normalizedWithdrawnCommission, err := k.NormalizeRewardDecCoins(c, avsAddr, commission.WithdrawnCommission)
	if err != nil {
		return nil, err
	}
	commission.UnwithdrawnCommission = normalizedUnwithdrawnCommission
	commission.WithdrawnCommission = normalizedWithdrawnCommission
	return &types.QueryOperatorCommissionResponse{OperatorCommission: &commission}, nil
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

// DelegationUnclaimedRewards queries the unclaimed rewards for a delegation.
func (k Keeper) DelegationUnclaimedRewards(ctx context.Context, req *types.QueryDelegationUnclaimedRewardsRequest) (*types.QueryDelegationUnclaimedRewardsResponse, error) {
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
	stakingRewards, compoundingRewards, err := k.GetDelegationUnclaimedRewards(c, true, strings.ToLower(req.StakerId), strings.ToLower(req.AssetId), req.Operator)
	if err != nil {
		return nil, err
	}
	normalizedStakingRewards, err := k.BatchNormalizeRewardDecimals(c, stakingRewards)
	if err != nil {
		return nil, err
	}
	normalizedCompoundingRewards, err := k.BatchNormalizeRewardDecimals(c, compoundingRewards)
	if err != nil {
		return nil, err
	}
	return &types.QueryDelegationUnclaimedRewardsResponse{
		Rewards:            normalizedStakingRewards,
		CompoundingRewards: normalizedCompoundingRewards,
	}, nil
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
	stakingRewards, compoundingRewards, err := k.GetStakerUnclaimedRewards(c, strings.ToLower(req.StakerId))
	if err != nil {
		return nil, err
	}
	normalizedStakingRewards, err := k.BatchNormalizeRewardDecimals(c, stakingRewards)
	if err != nil {
		return nil, err
	}
	normalizedCompoundingRewards, err := k.BatchNormalizeRewardDecimals(c, compoundingRewards)
	if err != nil {
		return nil, err
	}
	return &types.QueryStakerUnclaimedRewardsResponse{
		Rewards:            normalizedStakingRewards,
		CompoundingRewards: normalizedCompoundingRewards,
	}, nil
}

func (k Keeper) StakerAllRewards(
	ctx context.Context,
	req *types.QueryStakerAllRewardsRequest,
) (*types.QueryStakerAllRewardsResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}
	c := sdk.UnwrapSDKContext(ctx)
	_, _, err := assetstype.ValidateID(req.StakerId, false, false)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid stakerID,err:%v", err)
	}
	claimedRewards, err := k.GetStakerAllClaimedRewards(c, strings.ToLower(req.StakerId))
	if err != nil {
		return nil, err
	}
	stakingRewards, compoundingRewards, err := k.GetStakerUnclaimedRewards(c, strings.ToLower(req.StakerId))
	if err != nil {
		return nil, err
	}
	// The rewards don't need to be normalized here because MergeStakerRewards converts DecCoins to RewardInfo
	// (which uses Int amounts) via DecCoinsToRewardInfos, which handles the conversion appropriately.
	stakerAllRewards, err := k.MergeStakerRewards(c, claimedRewards, stakingRewards, compoundingRewards)
	if err != nil {
		return nil, err
	}

	return &types.QueryStakerAllRewardsResponse{
		Rewards: stakerAllRewards,
	}, nil
}

func (k Keeper) StakerRewardParams(
	ctx context.Context,
	req *types.QueryStakerRewardParamsRequest,
) (*types.QueryStakerRewardParamsResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}
	c := sdk.UnwrapSDKContext(ctx)
	_, _, err := assetstype.ValidateID(req.StakerId, false, false)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid stakerID,err:%v", err)
	}
	rewardParams, err := k.GetStakerRewardParams(c, strings.ToLower(req.StakerId))
	if err != nil {
		return nil, err
	}

	return &types.QueryStakerRewardParamsResponse{
		RewardParams: rewardParams,
	}, nil
}
