package keeper

import (
	"context"
	"strconv"

	"github.com/ExocoreNetwork/exocore/utils"
	"github.com/ExocoreNetwork/exocore/x/avs/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ types.QueryServer = &Keeper{}

func (k Keeper) QueryAVSInfo(ctx context.Context, req *types.QueryAVSInfoReq) (*types.QueryAVSInfoResponse, error) {
	c := sdk.UnwrapSDKContext(ctx)
	return k.GetAVSInfo(c, req.AVSAddress)
}

func (k Keeper) QueryAVSTaskInfo(ctx context.Context, req *types.QueryAVSTaskInfoReq) (*types.TaskInfo, error) {
	c := sdk.UnwrapSDKContext(ctx)
	return k.GetTaskInfo(c, req.TaskId, req.TaskAddr)
}

// QueryAVSAddrByChainID is an implementation of the QueryAVSAddrByChainID gRPC method
func (k Keeper) QueryAVSAddrByChainID(ctx context.Context, req *types.QueryAVSAddrByChainIDReq) (*types.QueryAVSAddrByChainIDResponse, error) {
	c := sdk.UnwrapSDKContext(ctx)
	if req == nil {
		return nil, sdkerrors.ErrInvalidRequest
	}
	if req.ChainID == "" {
		return nil, sdkerrors.ErrInvalidRequest
	}
	if req.ChainID == c.ChainID() {
		// only for the same chain, we remove the revision.
		req.ChainID = utils.ChainIDWithoutRevision(req.ChainID)
	}
	isChainAvs, avsAddr := k.IsAVSByChainID(c, req.ChainID)
	if !isChainAvs {
		return nil, types.ErrNotYetRegistered
	}
	return &types.QueryAVSAddrByChainIDResponse{AVSAddress: avsAddr}, nil
}

func (k Keeper) QuerySubmitTaskResult(ctx context.Context, req *types.QuerySubmitTaskResultReq) (*types.QuerySubmitTaskResultResponse, error) {
	c := sdk.UnwrapSDKContext(ctx)
	id, err := strconv.ParseUint(req.TaskId, 10, 64)
	if err != nil {
		return &types.QuerySubmitTaskResultResponse{}, err
	}

	info, err := k.GetTaskResultInfo(c, req.OperatorAddr, req.TaskAddress, id)
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

	addr, err := k.GetTaskChallengedInfo(c, req.OperatorAddr, req.TaskAddress, id)
	return &types.QueryChallengeInfoResponse{
		ChallengeAddr: addr,
	}, err
}
