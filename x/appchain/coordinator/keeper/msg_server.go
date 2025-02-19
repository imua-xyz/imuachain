package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/imua-xyz/imuachain/x/appchain/coordinator/types"
)

// msgServer is a wrapper around the Keeper (which is the actual implementation) and
// satisfies the MsgServer interface.
type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

// interface guard
var _ types.MsgServer = msgServer{}

func (m msgServer) RegisterSubscriberChain(
	goCtx context.Context, req *types.RegisterSubscriberChainRequest,
) (res *types.RegisterSubscriberChainResponse, err error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	if res, err = m.Keeper.AddSubscriberChain(ctx, req); err != nil {
		return nil, errorsmod.Wrapf(err, "RegisterSubscriberChain: key is %s", req.ChainID)
	}
	return res, nil
}
