package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	epochstypes "github.com/imua-xyz/imuachain/x/epochs/types"
)

// AccountKeeper defines the expected interface for the account keeper.
type AccountKeeper interface {
	GetModuleAddress(name string) sdk.AccAddress
}

// BankKeeper defines the expected interface needed to retrieve account balances.
type BankKeeper interface {
	SendCoinsFromModuleToAccount(
		ctx sdk.Context,
		senderModule string,
		recipientAddr sdk.AccAddress,
		amt sdk.Coins,
	) error
	SendCoinsFromModuleToModule(
		ctx sdk.Context,
		senderModule, recipientModule string,
		amt sdk.Coins,
	) error
	MintCoins(ctx sdk.Context, name string, amt sdk.Coins) error
}

// EpochsKeeper represents the expected keeper interface for the epochs module.
type EpochsKeeper interface {
	GetEpochInfo(sdk.Context, string) (epochstypes.EpochInfo, bool)
}
