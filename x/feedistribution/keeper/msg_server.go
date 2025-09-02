package keeper

import (
	"context"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	sdk "github.com/cosmos/cosmos-sdk/types"
	assetstype "github.com/imua-xyz/imuachain/x/assets/types"
	"github.com/imua-xyz/imuachain/x/feedistribution/types"
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

func (k Keeper) WithdrawDogfoodCommission(ctx context.Context, req *types.MsgWithdrawDogfoodCommission) (*types.MsgWithdrawDogfoodCommissionResponse, error) {
	c := sdk.UnwrapSDKContext(ctx)
	operatorAccAddr, err := sdk.AccAddressFromBech32(req.FromAddress)
	if err != nil {
		return nil, err
	}
	withdrawAmount, err := k.WithdrawCommissionFromDogfood(c, operatorAccAddr)
	if err != nil {
		return nil, err
	}
	return &types.MsgWithdrawDogfoodCommissionResponse{
		Amount: withdrawAmount,
	}, nil
}

func (k Keeper) ClaimAndWithdrawDogfoodReward(ctx context.Context, req *types.MsgClaimAndWithdrawDogfoodReward) (*types.MsgClaimAndWithdrawDogfoodRewardResponse, error) {
	c := sdk.UnwrapSDKContext(ctx)
	stakerAccAddr, err := sdk.AccAddressFromBech32(req.FromAddress)
	if err != nil {
		return nil, err
	}
	stakerID, _ := assetstype.GetStakerIDAndAssetID(assetstype.ImuachainLzID, stakerAccAddr, nil)
	claimedRewards, err := k.ClaimDelegationRewards(c, stakerID)
	if err != nil {
		return nil, err
	}

	actualWithdrawReward, err := k.WithdrawRewardFromDogfood(c, stakerID, req.Amount, stakerAccAddr)
	if err != nil {
		return nil, err
	}
	return &types.MsgClaimAndWithdrawDogfoodRewardResponse{
		ClaimedRewards:  claimedRewards,
		WithdrawnAmount: actualWithdrawReward,
	}, nil
}

func (k Keeper) UpdateStakerRewardParams(
	ctx context.Context,
	req *types.MsgUpdateStakerRewardParams,
) (*types.MsgUpdateStakerRewardParamsResponse, error) {
	c := sdk.UnwrapSDKContext(ctx)
	stakerAccAddr, err := sdk.AccAddressFromBech32(req.FromAddress)
	if err != nil {
		return nil, err
	}
	err = req.RewardParams.Validate()
	if err != nil {
		return nil, err
	}
	stakerID, _ := assetstype.GetStakerIDAndAssetID(assetstype.ImuachainLzID, stakerAccAddr, nil)
	err = k.SetStakerRewardParams(c, stakerID, req.RewardParams)
	if err != nil {
		return nil, err
	}
	return &types.MsgUpdateStakerRewardParamsResponse{}, nil
}

func (k Keeper) UndelegateReward(
	ctx context.Context,
	req *types.MsgUndelegateReward,
) (*types.MsgUndelegateRewardResponse, error) {
	c := sdk.UnwrapSDKContext(ctx)
	stakerAccAddr, err := sdk.AccAddressFromBech32(req.FromAddress)
	if err != nil {
		return nil, err
	}
	stakerID, _ := assetstype.GetStakerIDAndAssetID(assetstype.ImuachainLzID, stakerAccAddr, nil)
	_, _, err = assetstype.ValidateID(req.AssetId, false, false)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid assetID,err:%v", err)
	}
	operatorAccAddr, err := sdk.AccAddressFromBech32(req.OperatorAddr)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid operator address,err:%v", err)
	}
	err = k.UndelegateClaimedRewards(c, stakerID, strings.ToLower(req.AssetId), operatorAccAddr, req.InstantUnbonding, req.Amount)
	if err != nil {
		return nil, err
	}
	return &types.MsgUndelegateRewardResponse{}, nil
}
