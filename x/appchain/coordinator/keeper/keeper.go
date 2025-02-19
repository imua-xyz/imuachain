package keeper

import (
	"fmt"

	"github.com/cometbft/cometbft/libs/log"

	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	commontypes "github.com/imua-xyz/imuachain/x/appchain/common/types"
	"github.com/imua-xyz/imuachain/x/appchain/coordinator/types"
)

type Keeper struct {
	cdc            codec.BinaryCodec
	storeKey       storetypes.StoreKey
	avsKeeper      types.AVSKeeper
	epochsKeeper   types.EpochsKeeper
	operatorKeeper types.OperatorKeeper
	stakingKeeper  types.StakingKeeper
	clientKeeper   commontypes.ClientKeeper
}

// NewKeeper creates a new coordinator keeper.
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	avsKeeper types.AVSKeeper,
	epochsKeeper types.EpochsKeeper,
	operatorKeeper types.OperatorKeeper,
	stakingKeeper types.StakingKeeper,
	clientKeeper commontypes.ClientKeeper,
) Keeper {
	return Keeper{
		cdc:            cdc,
		storeKey:       storeKey,
		avsKeeper:      avsKeeper,
		epochsKeeper:   epochsKeeper,
		operatorKeeper: operatorKeeper,
		stakingKeeper:  stakingKeeper,
		clientKeeper:   clientKeeper,
	}
}

// Logger returns a logger object for use within the module.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}
