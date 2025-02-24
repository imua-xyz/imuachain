package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/imua-xyz/imuachain/x/avs/types"
)

type MsgServerImpl struct {
	keeper Keeper
}

func NewMsgServerImpl(keeper Keeper) *MsgServerImpl {
	return &MsgServerImpl{keeper: keeper}
}

var _ types.MsgServer = &MsgServerImpl{}

func (m MsgServerImpl) SubmitTaskResult(goCtx context.Context, req *types.SubmitTaskResultReq) (*types.SubmitTaskResultResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := m.keeper.SubmitTaskResult(ctx, req.FromAddress, req.Info); err != nil {
		return nil, err
	}
	return &types.SubmitTaskResultResponse{}, nil
}

func (m MsgServerImpl) RegisterAVS(_ context.Context, _ *types.RegisterAVSReq) (*types.RegisterAVSResponse, error) {
	// TODO implement me
	panic("implement me")
}

func (m MsgServerImpl) DeRegisterAVS(_ context.Context, _ *types.DeRegisterAVSReq) (*types.DeRegisterAVSResponse, error) {
	// TODO implement me
	panic("implement me")
}

func (m MsgServerImpl) RegisterAVSTask(_ context.Context, _ *types.RegisterAVSTaskReq) (*types.RegisterAVSTaskResponse, error) {
	// TODO implement me
	panic("implement me")
}
