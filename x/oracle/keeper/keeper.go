package keeper

import (
	"fmt"

	"github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	"github.com/imua-xyz/imuachain/x/oracle/keeper/common"
	"github.com/imua-xyz/imuachain/x/oracle/keeper/feedermanagement"
	"github.com/imua-xyz/imuachain/x/oracle/types"
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
		types.SlashingKeeper
		*feedermanagement.FeederManager
		postHandlers               map[int64]common.PostAggregationHandler
		cachedNSTStakersEventValue *string
		c                          *common.Caches
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
	slashingKeeper types.SlashingKeeper,
) Keeper {
	// ensure authority is a valid bech32 address
	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		panic(fmt.Sprintf("authority address %s is invalid: %s", authority, err))
	}
	// set KeyTable if it has not already been set
	if !ps.HasKeyTable() {
		ps = ps.WithKeyTable(types.ParamKeyTable())
	}

	ret := Keeper{
		cdc:                        cdc,
		storeKey:                   storeKey,
		memKey:                     memKey,
		paramstore:                 ps,
		KeeperDogfood:              sKeeper,
		delegationKeeper:           delegationKeeper,
		assetsKeeper:               assetsKeeper,
		authority:                  authority,
		SlashingKeeper:             slashingKeeper,
		FeederManager:              feedermanagement.NewFeederManager(nil),
		postHandlers:               make(map[int64]common.PostAggregationHandler),
		cachedNSTStakersEventValue: new(string),
		c:                          common.NewCaches(),
	}
	ret.FeederManager.SetKeeper(&ret)
	return ret
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

func (k Keeper) FlushCachedNSTStakersEvent(ctx sdk.Context) {
	if len(*k.cachedNSTStakersEventValue) > 0 {
		ctx.EventManager().EmitEvent(sdk.NewEvent(
			types.EventTypeCreatePrice,
			sdk.NewAttribute(types.AttributeKeyNSTStakersChange, *k.cachedNSTStakersEventValue),
		))
		*k.cachedNSTStakersEventValue = ""
	}
}
