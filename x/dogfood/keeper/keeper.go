package keeper

import (
	"fmt"

	operatortypes "github.com/ExocoreNetwork/exocore/x/operator/types"

	sdkmath "cosmossdk.io/math"

	"github.com/ExocoreNetwork/exocore/x/delegation/keeper"
	delegationtype "github.com/ExocoreNetwork/exocore/x/delegation/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/ExocoreNetwork/exocore/x/dogfood/types"
)

type (
	Keeper struct {
		cdc      codec.BinaryCodec
		storeKey storetypes.StoreKey

		// internal hooks to allow other modules to subscriber to our events
		dogfoodHooks types.DogfoodHooks

		// external keepers as interfaces
		epochsKeeper     types.EpochsKeeper
		operatorKeeper   types.OperatorKeeper
		delegationKeeper types.DelegationKeeper
		restakingKeeper  types.AssetsKeeper
		avsKeeper        types.AVSKeeper

		// edit params
		authority string
	}
)

// NewKeeper creates a new dogfood keeper.
func NewKeeper(
	cdc codec.BinaryCodec, storeKey storetypes.StoreKey,
	epochsKeeper types.EpochsKeeper, operatorKeeper types.OperatorKeeper,
	delegationKeeper keeper.Keeper, restakingKeeper types.AssetsKeeper,
	avsKeeper types.AVSKeeper, authority string,
) Keeper {
	k := Keeper{
		cdc:              cdc,
		storeKey:         storeKey,
		epochsKeeper:     epochsKeeper,
		operatorKeeper:   operatorKeeper,
		delegationKeeper: delegationKeeper,
		restakingKeeper:  restakingKeeper,
		avsKeeper:        avsKeeper,
		authority:        authority,
	}
	k.mustValidateFields()

	return k
}

// Logger returns a logger object for use within the module.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// SetHooks sets the hooks on the keeper. It intentionally has a pointer receiver so that
// changes can be saved to the object.
func (k *Keeper) SetHooks(sh types.DogfoodHooks) {
	if k.dogfoodHooks != nil {
		panic("cannot set dogfood hooks twice")
	}
	if sh == nil {
		panic("cannot set nil dogfood hooks")
	}
	k.dogfoodHooks = sh
}

// Hooks returns the hooks registered to the module.
func (k Keeper) Hooks() types.DogfoodHooks {
	if k.dogfoodHooks != nil {
		return k.dogfoodHooks
	}
	return types.MultiDogfoodHooks{}
}

// MarkUpdateValidatorSetFlag marks that the validator set needs to be updated at the end of this block.
// Mostly, these updates occur in response to the epoch ending. In other cases, they are the result of slashing.
func (k Keeper) MarkUpdateValidatorSetFlag(ctx sdk.Context) {
	store := ctx.KVStore(k.storeKey)
	key := types.ShouldUpdateValidatorSetByteKey()
	store.Set(key, []byte{1})
}

// ShouldUpdateValidatorSet returns true if the epoch ended in the beginning of this block, or the end of the
// previous block.
func (k Keeper) ShouldUpdateValidatorSet(ctx sdk.Context) bool {
	store := ctx.KVStore(k.storeKey)
	key := types.ShouldUpdateValidatorSetByteKey()
	return store.Has(key)
}

// ClearValidatorSetUpdateFlag clears the epoch end marker. It is called after the epoch end operations are
// applied.
func (k Keeper) ClearValidatorSetUpdateFlag(ctx sdk.Context) {
	store := ctx.KVStore(k.storeKey)
	key := types.ShouldUpdateValidatorSetByteKey()
	store.Delete(key)
}

// MarkEmitAvsEventFlag marks that an AVS event should be emitted in the BeginBlocker.
func (k Keeper) MarkEmitAvsEventFlag(ctx sdk.Context) {
	store := ctx.KVStore(k.storeKey)
	key := types.EmitAvsEventKey()
	store.Set(key, []byte{1})
}

// ShouldEmitAvsEvent returns true if an AVS event should be emitted in the BeginBlocker.
func (k Keeper) ShouldEmitAvsEvent(ctx sdk.Context) bool {
	store := ctx.KVStore(k.storeKey)
	key := types.EmitAvsEventKey()
	return store.Has(key)
}

// ClearEmitAvsEventFlag clears the AVS event marker. It is called after the AVS event is emitted.
func (k Keeper) ClearEmitAvsEventFlag(ctx sdk.Context) {
	store := ctx.KVStore(k.storeKey)
	key := types.EmitAvsEventKey()
	store.Delete(key)
}

func (k Keeper) mustValidateFields() {
	types.PanicIfNil(k.storeKey, "storeKey")
	types.PanicIfNil(k.cdc, "cdc")
	types.PanicIfNil(k.epochsKeeper, "epochsKeeper")
	types.PanicIfNil(k.operatorKeeper, "operatorKeeper")
	types.PanicIfNil(k.delegationKeeper, "delegationKeeper")
	types.PanicIfNil(k.restakingKeeper, "restakingKeeper")
	types.PanicIfNil(k.avsKeeper, "avsKeeper")
	// ensure authority is a valid bech32 address
	if _, err := sdk.AccAddressFromBech32(k.authority); err != nil {
		panic(fmt.Sprintf("authority address %s is invalid: %s", k.authority, err))
	}
}

// Add the function to get detail information through the operatorKeeper within the dogfood
func (k Keeper) ValidatorByConsAddrForChainID(ctx sdk.Context, consAddr sdk.ConsAddress, chainID string) (stakingtypes.Validator, bool) {
	return k.operatorKeeper.ValidatorByConsAddrForChainID(ctx, consAddr, chainID)
}

func (k *Keeper) GetStakersByOperator(ctx sdk.Context, operator, assetID string) (delegationtype.StakerList, error) {
	return k.delegationKeeper.GetStakersByOperator(ctx, operator, assetID)
}

func (k Keeper) GetAVSSupportedAssets(ctx sdk.Context, avsAddr string) (map[string]interface{}, error) {
	return k.avsKeeper.GetAVSSupportedAssets(ctx, avsAddr)
}

func (k Keeper) GetOptedInAVSForOperator(ctx sdk.Context, operatorAddr string) ([]string, error) {
	return k.operatorKeeper.GetOptedInAVSForOperator(ctx, operatorAddr)
}

func (k Keeper) CalculateUSDValueForStaker(ctx sdk.Context, stakerID, avsAddr string, operator sdk.AccAddress) (sdkmath.LegacyDec, error) {
	return k.operatorKeeper.CalculateUSDValueForStaker(ctx, stakerID, avsAddr, operator)
}

func (k *Keeper) OperatorInfo(ctx sdk.Context, addr string) (info *operatortypes.OperatorInfo, err error) {
	return k.operatorKeeper.OperatorInfo(ctx, addr)
}
