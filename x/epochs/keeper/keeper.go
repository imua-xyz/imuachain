package keeper

import (
	"fmt"

	"github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/imua-xyz/imuachain/x/epochs/types"
)

type (
	Keeper struct {
		cdc      codec.BinaryCodec
		storeKey storetypes.StoreKey
		hooks    types.EpochHooks
	}
)

// NewKeeper creates a new epochs keeper. it is returned as a pointer since
// the primary purpose of an epochs keeper to a caller is to use its hooks.
// the hooks should be set on the pointer for them to take effect.
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
) *Keeper {
	return &Keeper{
		cdc:      cdc,
		storeKey: storeKey,
	}
}

// Logger returns a logger object for use within the module.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// SetHooks sets the hooks on the keeper. It intentionally has a pointer receiver so that
// changes can be saved to the object.
func (k *Keeper) SetHooks(sh types.EpochHooks) {
	if k.hooks != nil {
		panic("cannot set hooks twice")
	}
	if sh == nil {
		panic("cannot set nil hooks")
	}
	k.hooks = sh
}

// Hooks returns the hooks registered to the module.
func (k Keeper) Hooks() types.EpochHooks {
	if k.hooks != nil {
		return k.hooks
	}
	return types.MultiEpochHooks{}
}
