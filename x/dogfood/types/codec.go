package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

func RegisterCodec(*codec.LegacyAmino) {
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	cryptocodec.RegisterInterfaces(registry)
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

var (
	Amino          = codec.NewLegacyAmino()
	ModuleRegistry = cdctypes.NewInterfaceRegistry()
	ModuleCdc      = codec.NewProtoCodec(ModuleRegistry)
)
