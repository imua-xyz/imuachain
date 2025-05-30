package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgCreatePrice{}, "oracle/CreatePrice", nil)
	cdc.RegisterConcrete(Params{}, "imua/x/oracle/Params", nil)
	// this line is used by starport scaffolding # 2
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	// TODO: remove this
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgCreatePrice{},
	)
	// this line is used by starport scaffolding # 3

	// this method registered sdk.Msg, so we don't need the above one
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

var (
	Amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)
