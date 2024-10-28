package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	auth "github.com/cosmos/cosmos-sdk/x/auth/types"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	transfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	conntypes "github.com/cosmos/ibc-go/v7/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	ibcexported "github.com/cosmos/ibc-go/v7/modules/core/exported"
)

// ClientKeeper defines the expected IBC client keeper
type ClientKeeper interface {
	CreateClient(
		sdk.Context, ibcexported.ClientState, ibcexported.ConsensusState,
	) (string, error)
	GetClientState(sdk.Context, string) (ibcexported.ClientState, bool)
	GetLatestClientConsensusState(
		sdk.Context, string,
	) (ibcexported.ConsensusState, bool)
	GetSelfConsensusState(
		sdk.Context, ibcexported.Height,
	) (ibcexported.ConsensusState, error)
}

// ScopedKeeper defines the expected IBC capability keeper
type ScopedKeeper interface {
	GetCapability(sdk.Context, string) (*capabilitytypes.Capability, bool)
	AuthenticateCapability(sdk.Context, *capabilitytypes.Capability, string) bool
	ClaimCapability(sdk.Context, *capabilitytypes.Capability, string) error
}

// PortKeeper defines the expected IBC port keeper
type PortKeeper interface {
	BindPort(ctx sdk.Context, portID string) *capabilitytypes.Capability
}

// ConnectionKeeper defines the expected IBC connection keeper
type ConnectionKeeper interface {
	GetConnection(ctx sdk.Context, connectionID string) (conntypes.ConnectionEnd, bool)
}

// ChannelKeeper defines the expected IBC channel keeper
type ChannelKeeper interface {
	GetChannel(sdk.Context, string, string) (channeltypes.Channel, bool)
	GetNextSequenceSend(sdk.Context, string, string) (uint64, bool)
	SendPacket(
		sdk.Context, *capabilitytypes.Capability,
		string, string, clienttypes.Height,
		uint64, []byte,
	) (uint64, error)
	WriteAcknowledgement(
		sdk.Context, *capabilitytypes.Capability,
		ibcexported.PacketI, ibcexported.Acknowledgement,
	) error
	ChanCloseInit(
		sdk.Context, string, string, *capabilitytypes.Capability,
	) error
	GetChannelConnection(sdk.Context, string, string) (string, ibcexported.ConnectionI, error)
}

// IBCKeeper defines the expected interface needed for openning a
// channel
type IBCCoreKeeper interface {
	ChannelOpenInit(
		context.Context, *channeltypes.MsgChannelOpenInit,
	) (*channeltypes.MsgChannelOpenInitResponse, error)
}

// AccountKeeper defines the expected account keeper
type AccountKeeper interface {
	GetModuleAccount(ctx sdk.Context, name string) auth.ModuleAccountI
}

// BankKeeper defines the expected bank keeper
type BankKeeper interface {
	GetBalance(ctx sdk.Context, addr sdk.AccAddress, denom string) sdk.Coin
	GetAllBalances(ctx sdk.Context, addr sdk.AccAddress) sdk.Coins
	SendCoinsFromModuleToModule(
		ctx sdk.Context,
		senderModule, recipientModule string,
		amt sdk.Coins,
	) error
}

// IBCTransferKeeper defines the expected IBC transfer keeper
type IBCTransferKeeper interface {
	Transfer(
		context.Context,
		*transfertypes.MsgTransfer,
	) (*transfertypes.MsgTransferResponse, error)
}

// SubscriberHooks defines the hooks used by other modules to subscribe to x/appchain/subscriber
// related events.
type SubscriberHooks interface {
	AfterValidatorBonded(
		sdk.Context, sdk.ConsAddress, sdk.ValAddress,
	) error
}
