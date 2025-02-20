package keeper

import (
	"context"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/imua-xyz/imuachain/x/avs/types"
)

var _ types.QueryServer = &Keeper{}

func (k Keeper) QueryAVSInfo(ctx context.Context, req *types.QueryAVSInfoReq) (*types.QueryAVSInfoResponse, error) {
	c := sdk.UnwrapSDKContext(ctx)
	return k.GetAVSInfo(c, req.AVSAddress)
}

func (k Keeper) QueryAVSTaskInfo(ctx context.Context, req *types.QueryAVSTaskInfoReq) (*types.TaskInfo, error) {
	c := sdk.UnwrapSDKContext(ctx)
	return k.GetTaskInfo(c, req.TaskId, req.TaskAddress)
}

// QueryAVSAddressByChainID is an implementation of the QueryAVSAddrByChainID gRPC method
func (k Keeper) QueryAVSAddressByChainID(ctx context.Context, req *types.QueryAVSAddressByChainIDReq) (*types.QueryAVSAddressByChainIDResponse, error) {
	c := sdk.UnwrapSDKContext(ctx)
	isChainAvs, avsAddr := k.IsAVSByChainID(c, types.ChainIDWithoutRevision(req.Chain))
	if !isChainAvs {
		return nil, types.ErrNotYetRegistered
	}
	return &types.QueryAVSAddressByChainIDResponse{AVSAddress: avsAddr}, nil
}

func (k Keeper) QuerySubmitTaskResult(ctx context.Context, req *types.QuerySubmitTaskResultReq) (*types.QuerySubmitTaskResultResponse, error) {
	c := sdk.UnwrapSDKContext(ctx)
	id, err := strconv.ParseUint(req.TaskId, 10, 64)
	if err != nil {
		return &types.QuerySubmitTaskResultResponse{}, err
	}

	info, err := k.GetTaskResultInfo(c, req.OperatorAddress, req.TaskAddress, id)
	return &types.QuerySubmitTaskResultResponse{
		Info: info,
	}, err
}

func (k Keeper) QueryChallengeInfo(ctx context.Context, req *types.QueryChallengeInfoReq) (*types.QueryChallengeInfoResponse, error) {
	c := sdk.UnwrapSDKContext(ctx)
	id, err := strconv.ParseUint(req.TaskId, 10, 64)
	if err != nil {
		return &types.QueryChallengeInfoResponse{}, err
	}

	addr, err := k.GetTaskChallengedInfo(c, req.TaskAddress, id)
	return &types.QueryChallengeInfoResponse{
		ChallengeAddress: addr,
	}, err
}
