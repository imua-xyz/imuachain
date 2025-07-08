package keeper

import (
	"context"

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
