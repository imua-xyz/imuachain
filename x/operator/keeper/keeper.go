package keeper

import (
	"context"

	sdkmath "cosmossdk.io/math"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	operatortypes "github.com/ExocoreNetwork/exocore/x/operator/types"
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type Keeper struct {
	storeKey storetypes.StoreKey
	cdc      codec.BinaryCodec
	// other keepers
	assetsKeeper     operatortypes.AssetsKeeper
	delegationKeeper operatortypes.DelegationKeeper
	oracleKeeper     operatortypes.OracleKeeper
	avsKeeper        operatortypes.AVSKeeper
	stakingKeeper    operatortypes.StakingKeeper
	hooks            operatortypes.OperatorHooks // set separately via call to SetHooks
	slashKeeper      operatortypes.SlashKeeper   // for jailing and unjailing check TODO(mm)
}

func NewKeeper(
	storeKey storetypes.StoreKey,
	cdc codec.BinaryCodec,
	assetsKeeper operatortypes.AssetsKeeper,
	delegationKeeper operatortypes.DelegationKeeper,
	oracleKeeper operatortypes.OracleKeeper,
	avsKeeper operatortypes.AVSKeeper,
	stakingKeeper operatortypes.StakingKeeper,
	slashKeeper operatortypes.SlashKeeper,
) Keeper {
	return Keeper{
		storeKey:         storeKey,
		cdc:              cdc,
		assetsKeeper:     assetsKeeper,
		delegationKeeper: delegationKeeper,
		oracleKeeper:     oracleKeeper,
		avsKeeper:        avsKeeper,
		stakingKeeper:    stakingKeeper,
		slashKeeper:      slashKeeper,
	}
}

func (k *Keeper) OracleInterface() operatortypes.OracleKeeper {
	return k.oracleKeeper
}

func (k Keeper) GetUnbondingExpirationBlockNumber(_ sdk.Context, _ sdk.AccAddress, startHeight uint64) uint64 {
	return startHeight + operatortypes.UnbondingExpiration
}

// OperatorKeeper interface will be implemented by deposit keeper
type OperatorKeeper interface {
	// RegisterOperator handle the registerOperator txs from msg service
	RegisterOperator(ctx context.Context, req *operatortypes.RegisterOperatorReq) (*operatortypes.RegisterOperatorResponse, error)

	IsOperator(ctx sdk.Context, addr sdk.AccAddress) bool

	GetUnbondingExpirationBlockNumber(ctx sdk.Context, OperatorAddress sdk.AccAddress, startHeight uint64) uint64

	OptIn(ctx sdk.Context, operatorAddress sdk.AccAddress, AVSAddr string) error

	OptOut(ctx sdk.Context, OperatorAddress sdk.AccAddress, AVSAddr string) error

	Slash(ctx sdk.Context, parameter *operatortypes.SlashInputInfo) error

	SlashWithInfractionReason(
		ctx sdk.Context, addr sdk.AccAddress, infractionHeight, power int64,
		slashFactor sdk.Dec, infraction stakingtypes.Infraction,
	) sdkmath.Int
}

// SetHooks stores the given hooks implementations.
// Note that the Keeper is changed into a pointer to prevent an ineffective assignment.
func (k *Keeper) SetHooks(hooks operatortypes.OperatorHooks) {
	if hooks == nil {
		panic("cannot set nil hooks")
	}
	if k.hooks != nil {
		panic("cannot set hooks twice")
	}
	k.hooks = hooks
}

// Hooks returns the keeper's hooks.
func (k *Keeper) Hooks() operatortypes.OperatorHooks {
	if k.hooks == nil {
		// return a no-op implementation if no hooks are set to prevent calling nil functions
		return operatortypes.MultiOperatorHooks{}
	}
	return k.hooks
}
