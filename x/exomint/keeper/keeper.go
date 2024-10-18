package keeper

import (
	"fmt"

	"github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/ExocoreNetwork/exocore/x/exomint/types"
)

type (
	Keeper struct {
		cdc              codec.BinaryCodec
		storeKey         storetypes.StoreKey
		bankKeeper       types.BankKeeper
		epochsKeeper     types.EpochsKeeper
		feeCollectorName string
		// the address capable of executing a MsgUpdateParams message, typically x/gov.
		authority string
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	ak types.AccountKeeper,
	bk types.BankKeeper,
	ek types.EpochsKeeper,
	feeCollectorName string,
	authority string,
) Keeper {
	// ensure mint module account is set
	if addr := ak.GetModuleAddress(types.ModuleName); addr == nil {
		panic(fmt.Sprintf("the x/%s module account has not been set", types.ModuleName))
	}
	// ensure authority is a valid bech32 address
	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		panic(fmt.Sprintf("authority address %s is invalid: %s", authority, err))
	}

	return Keeper{
		cdc:              cdc,
		storeKey:         storeKey,
		bankKeeper:       bk,
		epochsKeeper:     ek,
		feeCollectorName: feeCollectorName,
		authority:        authority,
	}
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// MintCoins implements an alias call to the underlying supply keeper's
// MintCoins to be used in the x/epochs hook.
func (k Keeper) MintCoins(ctx sdk.Context, newCoins sdk.Coins) error {
	// the bank keeper validates `newCoins`, so we don't have to do it here.
	// it does not complain about `newCoins` being empty, but it does
	// complain about the following:
	// 1. valid denomination, which we have checked in the params validation.
	// 2. sorted denominations amongst multiple coins, which doesn't apply to us
	//    since we are only minting one coin.
	// 3. duplicate denomination, which doesn't apply to us since we are only
	//    minting one coin.
	// 4. coin amount being positive, which is also true since we check for
	//    negative amounts in params validation, and zero amount short circuits
	//    the epochs hook to skip this function.
	return k.bankKeeper.MintCoins(ctx, types.ModuleName, newCoins)
}

// AddCollectedFees implements an alias call to the underlying supply keeper's
// AddCollectedFees to be used in the x/epochs hook.
func (k Keeper) AddCollectedFees(ctx sdk.Context, fees sdk.Coins) error {
	return k.bankKeeper.SendCoinsFromModuleToModule(
		ctx, types.ModuleName, k.feeCollectorName, fees,
	)
}

// GetAuthority returns the authority address that can execute MsgUpdateParams.
func (k Keeper) GetAuthority() string {
	return k.authority
}
