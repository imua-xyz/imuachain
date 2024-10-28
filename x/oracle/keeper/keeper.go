package keeper

import (
	"fmt"

	sdkmath "cosmossdk.io/math"

	"github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	"github.com/ExocoreNetwork/exocore/x/oracle/keeper/common"
	"github.com/ExocoreNetwork/exocore/x/oracle/types"
)

type (
	Keeper struct {
		cdc        codec.BinaryCodec
		storeKey   storetypes.StoreKey
		memKey     storetypes.StoreKey
		paramstore paramtypes.Subspace
		authority  string
		common.KeeperDogfood
		delegationKeeper types.DelegationKeeper
		assetsKeeper     types.AssetsKeeper
	}
)

var _ common.KeeperOracle = Keeper{}

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	memKey storetypes.StoreKey,
	ps paramtypes.Subspace,
	sKeeper common.KeeperDogfood,
	delegationKeeper types.DelegationKeeper,
	assetsKeeper types.AssetsKeeper,
	authority string,
) Keeper {
	// ensure authority is a valid bech32 address
	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		panic(fmt.Sprintf("authority address %s is invalid: %s", authority, err))
	}
	// set KeyTable if it has not already been set
	if !ps.HasKeyTable() {
		ps = ps.WithKeyTable(types.ParamKeyTable())
	}

	return Keeper{
		cdc:              cdc,
		storeKey:         storeKey,
		memKey:           memKey,
		paramstore:       ps,
		KeeperDogfood:    sKeeper,
		delegationKeeper: delegationKeeper,
		assetsKeeper:     assetsKeeper,
		authority:        authority,
	}
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// UpdateNativeTokenValidatorInfo it's used to fix the issue of missing interface.
// it will be removed when merging with the oracle PR.
func (k Keeper) UpdateNativeTokenValidatorInfo(_ sdk.Context, _, _, _ string, _ sdkmath.Int) error {
	return nil
}
