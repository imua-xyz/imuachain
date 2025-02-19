package keeper

import (
	"fmt"

	sdkmath "cosmossdk.io/math"

	"github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	"github.com/imua-xyz/imuachain/x/oracle/keeper/common"
	"github.com/imua-xyz/imuachain/x/oracle/types"
)

type (
	Keeper struct {
		cdc        codec.BinaryCodec
		storeKey   storetypes.StoreKey
		memKey     storetypes.StoreKey
		paramstore paramtypes.Subspace
		// authority  string
		common.KeeperDogfood
		delegationKeeper types.DelegationKeeper
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
) Keeper {
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
