package types

import (
	time "time"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
)

// AccountKeeper defines the expected account keeper used for simulations (noalias)
type AccountKeeper interface {
	GetAccount(ctx sdk.Context, addr sdk.AccAddress) types.AccountI
	// Methods imported from account should be defined here
}

// BankKeeper defines the expected interface needed to retrieve account balances.
type BankKeeper interface {
	SpendableCoins(ctx sdk.Context, addr sdk.AccAddress) sdk.Coins
	// Methods imported from bank should be defined here
}

// DelegationKeeper defines the expected interfaces needed to update nst token balance change
type DelegationKeeper interface {
	UpdateNSTBalance(ctx sdk.Context, stakerID, assetID string, amount sdkmath.Int) error
}

type AssetsKeeper interface {
	GetAssetsDecimal(ctx sdk.Context, assets map[string]interface{}) (decimals map[string]uint32, err error)
}

type SlashingKeeper interface {
	JailUntil(sdk.Context, sdk.ConsAddress, time.Time)
}
