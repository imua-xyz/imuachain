package keeper

import (
	context "context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	keytypes "github.com/imua-xyz/imuachain/types/keys"
	"github.com/imua-xyz/imuachain/utils"
	"github.com/imua-xyz/imuachain/x/operator/types"
)

type MsgServerImpl struct {
	keeper Keeper
}

func NewMsgServerImpl(keeper Keeper) *MsgServerImpl {
	return &MsgServerImpl{keeper: keeper}
}

var _ types.MsgServer = &MsgServerImpl{}

// RegisterOperator is an implementation of the msg server for the operator module.
func (msgServer *MsgServerImpl) RegisterOperator(goCtx context.Context, req *types.RegisterOperatorReq) (*types.RegisterOperatorResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := msgServer.keeper.RegisterOperator(ctx, req.FromAddress, req.Info); err != nil {
		return nil, err
	}
	return &types.RegisterOperatorResponse{}, nil
}

// OptIntoAVS is an implementation of the msg server for the operator module.
func (msgServer *MsgServerImpl) OptIntoAVS(goCtx context.Context, req *types.OptIntoAVSReq) (res *types.OptIntoAVSResponse, err error) {
	uncachedCtx := sdk.UnwrapSDKContext(goCtx)
	// only write if both calls succeed
	ctx, writeFunc := uncachedCtx.CacheContext()
	defer func() {
		if err == nil {
			writeFunc()
		}
	}()
	// check if the AVS is a chain-type of AVS
	_, isChainAvs := msgServer.keeper.avsKeeper.GetChainIDByAVSAddr(ctx, req.AvsAddress)
	// #nosec G703 // already validated
	accAddr, _ := sdk.AccAddressFromBech32(req.FromAddress)
	if !isChainAvs {
		if len(req.PublicKeyJSON) > 0 {
			return nil, types.ErrInvalidPubKey.Wrap("public key is not required for non-chain AVS")
		}
		err = msgServer.keeper.OptIn(ctx, accAddr, req.AvsAddress)
		if err != nil {
			return nil, err
		}
	} else {
		key := keytypes.NewWrappedConsKeyFromJSON(req.PublicKeyJSON)
		if key == nil {
			return nil, types.ErrInvalidPubKey.Wrap("invalid public key")
		}
		err = msgServer.keeper.OptInWithConsKey(ctx, accAddr, req.AvsAddress, key)
		if err != nil {
			return nil, err
		}
	}
	return &types.OptIntoAVSResponse{}, nil
}

// OptOutOfAVS is an implementation of the msg server for the operator module.
func (msgServer *MsgServerImpl) OptOutOfAVS(goCtx context.Context, req *types.OptOutOfAVSReq) (res *types.OptOutOfAVSResponse, err error) {
	uncachedCtx := sdk.UnwrapSDKContext(goCtx)
	ctx, writeFunc := uncachedCtx.CacheContext()
	defer func() {
		if err == nil {
			writeFunc()
		}
	}()
	// #nosec G703 // already validated
	accAddr, _ := sdk.AccAddressFromBech32(req.FromAddress)
	err = msgServer.keeper.OptOut(ctx, accAddr, req.AvsAddress)
	if err != nil {
		return nil, err
	}
	return &types.OptOutOfAVSResponse{}, nil
}

// SetConsKey is an implementation of the msg server for the operator module.
func (msgServer *MsgServerImpl) SetConsKey(goCtx context.Context, req *types.SetConsKeyReq) (*types.SetConsKeyResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	chainID, isAVS := msgServer.keeper.avsKeeper.GetChainIDByAVSAddr(ctx, req.AvsAddress)
	if !isAVS {
		return nil, types.ErrNoSuchAvs.Wrap("AVS not found")
	}
	// #nosec G703 // already validated
	accAddr, _ := sdk.AccAddressFromBech32(req.Address)
	if !msgServer.keeper.IsOptedInAndNotJailed(ctx, accAddr.String(), req.AvsAddress) {
		return nil, types.ErrIsOptedOutOrJailed
	}
	wrappedKey := keytypes.NewWrappedConsKeyFromJSON(req.PublicKeyJSON)
	if wrappedKey == nil {
		return nil, types.ErrInvalidPubKey.Wrap("invalid public key")
	}
	if err := msgServer.keeper.SetOperatorConsKeyForChainID(ctx, accAddr, chainID, wrappedKey); err != nil {
		return nil, err
	}
	return &types.SetConsKeyResponse{}, nil
}

// UpdateCommissionRate is an implementation of the msg server for the operator module.
func (msgServer *MsgServerImpl) UpdateCommissionRate(
	goCtx context.Context, req *types.UpdateCommissionRateReq,
) (*types.UpdateCommissionRateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	accAddr, err := sdk.AccAddressFromBech32(req.Address)
	if err != nil {
		return nil, err
	}
	if err := msgServer.keeper.ValidateAndUpdateCommissionRate(
		ctx, accAddr, req.CommissionRate,
	); err != nil {
		return nil, err
	}
	return &types.UpdateCommissionRateResponse{}, nil
}

// EditOperator is an implementation of the msg server for the operator module.
func (msgServer *MsgServerImpl) EditOperator(
	goCtx context.Context, req *types.EditOperatorReq,
) (*types.EditOperatorResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	accAddr, err := sdk.AccAddressFromBech32(req.Address)
	if err != nil {
		return nil, err
	}
	if err := msgServer.keeper.EditOperator(ctx, accAddr, req.OperatorMetaInfo); err != nil {
		return nil, err
	}
	return &types.EditOperatorResponse{}, nil
}

// UpdateParams is an implementation of the msg server for the operator module.
func (msgServer *MsgServerImpl) UpdateParams(
	goCtx context.Context, req *types.MsgUpdateParams,
) (*types.MsgUpdateParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	if utils.IsMainnet(ctx.ChainID()) && msgServer.keeper.authority != req.Authority {
		return nil, govtypes.ErrInvalidSigner.Wrapf(
			"invalid authority; expected %s, got %s",
			msgServer.keeper.authority, req.Authority,
		)
	}
	logger := msgServer.keeper.Logger(ctx)
	logger.Info(
		"UpdateParams request",
		"authority", msgServer.keeper.authority,
		"params.Authority", req.Authority,
	)
	// check new params are valid
	if err := req.Params.Validate(); err != nil {
		return nil, err
	}
	// update params
	msgServer.keeper.SetParams(ctx, req.Params)
	// emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeUpdateParams,
			sdk.NewAttribute(types.AttributeKeyAuthority, msgServer.keeper.authority),
			sdk.NewAttribute(types.AttributeKeyMinCommissionRate, req.Params.MinCommissionRate.String()),
			sdk.NewAttribute(types.AttributeKeyMinCommissionUpdateInterval, req.Params.MinCommissionUpdateInterval.String()),
		),
	)
	return &types.MsgUpdateParamsResponse{}, nil
}
