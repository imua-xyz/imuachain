package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/imua-xyz/imuachain/utils"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/imua-xyz/imuachain/x/avs/types"
)

type MsgServerImpl struct {
	keeper Keeper
}

// UpdateParams is used to trigger a params update.
// TODO: It must be signed by the authority.
func (m MsgServerImpl) UpdateParams(ctx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	c := sdk.UnwrapSDKContext(ctx)
	if utils.IsMainnet(c.ChainID()) && m.keeper.authority != msg.Authority {
		return nil, govtypes.ErrInvalidSigner.Wrapf(
			"invalid authority; expected %s, got %s",
			m.keeper.authority, msg.Authority,
		)
	}
	m.keeper.Logger(c).Info(
		"UpdateParams request",
		"authority", m.keeper.authority,
		"params.Authority", msg.Authority,
	)
	prevParams := m.keeper.GetParams(c)
	nextParams := msg.Params
	// stateless validations
	overParams := nextParams.OverrideIfRequired(*prevParams, m.keeper.Logger(c))
	if err := overParams.Validate(); err != nil {
		return nil, errorsmod.Wrapf(
			types.ErrInconsistentParams,
			"invalid params: %s", err,
		)
	}
	// stateful validations
	// no need to check if MintDenom is registered in BankKeeper, since it does not itself
	// perform such checks.
	// the reward is already guaranteed to be positive and fits in the bit length.
	// so, we just have to check epoch here.
	if _, found := m.keeper.epochsKeeper.GetEpochInfo(c, overParams.EpochIdentifier); !found {
		m.keeper.Logger(c).Info("UpdateParams", "overriding EpochIdentifier with value", prevParams.EpochIdentifier)
		overParams.EpochIdentifier = prevParams.EpochIdentifier
	}
	m.keeper.SetParams(c, &overParams)
	return &types.MsgUpdateParamsResponse{}, nil
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
