package testutil

import (
	"cosmossdk.io/math"
	"github.com/ExocoreNetwork/exocore/utils"
	exominttypes "github.com/ExocoreNetwork/exocore/x/exomint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
)

// FundAccount is a utility function that funds an account by minting and
// sending the coins to the address.
func FundAccount(ctx sdk.Context, bankKeeper bankkeeper.Keeper, addr sdk.AccAddress, amounts sdk.Coins) error {
	if err := bankKeeper.MintCoins(ctx, exominttypes.ModuleName, amounts); err != nil {
		return err
	}

	return bankKeeper.SendCoinsFromModuleToAccount(ctx, exominttypes.ModuleName, addr, amounts)
}

// FundAccountWithBaseDenom is a utility function that uses the FundAccount function
// to fund an account with the default Evmos denomination.
func FundAccountWithBaseDenom(ctx sdk.Context, bankKeeper bankkeeper.Keeper, addr sdk.AccAddress, amount int64) error {
	coins := sdk.NewCoins(
		sdk.NewCoin(utils.BaseDenom, math.NewInt(amount)),
	)
	return FundAccount(ctx, bankKeeper, addr, coins)
}

// FundModuleAccount is a utility function that funds a module account by
// minting and sending the coins to the address.
func FundModuleAccount(ctx sdk.Context, bankKeeper bankkeeper.Keeper, recipientMod string, amounts sdk.Coins) error {
	if err := bankKeeper.MintCoins(ctx, exominttypes.ModuleName, amounts); err != nil {
		return err
	}

	return bankKeeper.SendCoinsFromModuleToModule(ctx, exominttypes.ModuleName, recipientMod, amounts)
}
