package keeper

import "github.com/imua-xyz/imuachain/x/appchain/subscriber/types"

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
