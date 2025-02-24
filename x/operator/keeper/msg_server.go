package keeper

import (
	context "context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	keytypes "github.com/imua-xyz/imuachain/types/keys"
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
	if err := msgServer.keeper.SetOperatorInfo(ctx, req.FromAddress, req.Info); err != nil {
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
			return nil, errorsmod.Wrap(types.ErrInvalidPubKey, "public key is not required for non-chain AVS")
		}
		err = msgServer.keeper.OptIn(ctx, accAddr, req.AvsAddress)
		if err != nil {
			return nil, err
		}
	} else {
		key := keytypes.NewWrappedConsKeyFromJSON(req.PublicKeyJSON)
		if key == nil {
			return nil, errorsmod.Wrap(types.ErrInvalidPubKey, "invalid public key")
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
	chainID, isAvs := msgServer.keeper.avsKeeper.GetChainIDByAVSAddr(ctx, req.AvsAddress)
	if !isAvs {
		return nil, errorsmod.Wrap(types.ErrNoSuchAvs, "AVS not found")
	}
	// #nosec G703 // already validated
	accAddr, _ := sdk.AccAddressFromBech32(req.Address)
	if !msgServer.keeper.IsActive(ctx, accAddr, req.AvsAddress) {
		return nil, errorsmod.Wrap(types.ErrNotOptedIn, "operator is not active")
	}
	wrappedKey := keytypes.NewWrappedConsKeyFromJSON(req.PublicKeyJSON)
	if wrappedKey == nil {
		return nil, errorsmod.Wrap(types.ErrInvalidPubKey, "invalid public key")
	}
	if err := msgServer.keeper.SetOperatorConsKeyForChainID(ctx, accAddr, chainID, wrappedKey); err != nil {
		return nil, err
	}
	return &types.SetConsKeyResponse{}, nil
}
