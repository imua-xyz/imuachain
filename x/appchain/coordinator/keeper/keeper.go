package keeper

import (
	"fmt"

	"github.com/ExocoreNetwork/exocore/utils"
	commontypes "github.com/ExocoreNetwork/exocore/x/appchain/common/types"
	"github.com/ExocoreNetwork/exocore/x/appchain/coordinator/types"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	host "github.com/cosmos/ibc-go/v7/modules/core/24-host"
)

type Keeper struct {
	cdc              codec.BinaryCodec
	storeKey         storetypes.StoreKey
	avsKeeper        types.AVSKeeper
	epochsKeeper     types.EpochsKeeper
	operatorKeeper   types.OperatorKeeper
	stakingKeeper    types.StakingKeeper
	delegationKeeper types.DelegationKeeper
	clientKeeper     commontypes.ClientKeeper
	portKeeper       commontypes.PortKeeper
	scopedKeeper     commontypes.ScopedKeeper
	channelKeeper    commontypes.ChannelKeeper
	connectionKeeper commontypes.ConnectionKeeper
	accountKeeper    commontypes.AccountKeeper
}

// NewKeeper creates a new coordinator keeper.
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	avsKeeper types.AVSKeeper,
	epochsKeeper types.EpochsKeeper,
	operatorKeeper types.OperatorKeeper,
	stakingKeeper types.StakingKeeper,
	delegationKeeper types.DelegationKeeper,
	clientKeeper commontypes.ClientKeeper,
	portKeeper commontypes.PortKeeper,
	scopedKeeper commontypes.ScopedKeeper,
	channelKeeper commontypes.ChannelKeeper,
	connectionKeeper commontypes.ConnectionKeeper,
	accountKeeper commontypes.AccountKeeper,
) Keeper {
	k := Keeper{
		cdc:              cdc,
		storeKey:         storeKey,
		avsKeeper:        avsKeeper,
		epochsKeeper:     epochsKeeper,
		operatorKeeper:   operatorKeeper,
		stakingKeeper:    stakingKeeper,
		delegationKeeper: delegationKeeper,
		clientKeeper:     clientKeeper,
		portKeeper:       portKeeper,
		scopedKeeper:     scopedKeeper,
		channelKeeper:    channelKeeper,
		connectionKeeper: connectionKeeper,
		accountKeeper:    accountKeeper,
	}
	k.mustValidateFields()
	return k
}

// Logger returns a logger object for use within the module.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// BindPort defines a wrapper function for the port Keeper's function in
// order to expose it to module's InitGenesis function
func (k Keeper) BindPort(ctx sdk.Context, portID string) error {
	capability := k.portKeeper.BindPort(ctx, portID)
	return k.ClaimCapability(ctx, capability, host.PortPath(portID))
}

// GetPort returns the portID for the IBC app module. Used in ExportGenesis
func (k Keeper) GetPort(ctx sdk.Context) string {
	store := ctx.KVStore(k.storeKey)
	return string(store.Get(types.PortKey()))
}

// SetPort sets the portID for the IBC app module. Used in InitGenesis
func (k Keeper) SetPort(ctx sdk.Context, portID string) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.PortKey(), []byte(portID))
}

// ClaimCapability allows the IBC app module to claim a capability that core IBC
// passes to it
func (k Keeper) ClaimCapability(
	ctx sdk.Context, cap *capabilitytypes.Capability, name string,
) error {
	return k.scopedKeeper.ClaimCapability(ctx, cap, name)
}

// mustValidateFields ensures that all the required fields are set. It does not count the number
func (k Keeper) mustValidateFields() {
	// TODO: there is no way to count the number of fields here, besides using reflect, which
	// fails the Linter. The developer should ensure to add the fields here when adding new fields.
	utils.PanicIfNil(k.storeKey, "storeKey")
	utils.PanicIfNil(k.cdc, "cdc")
	utils.PanicIfNil(k.avsKeeper, "avsKeeper")
	utils.PanicIfNil(k.epochsKeeper, "epochsKeeper")
	utils.PanicIfNil(k.operatorKeeper, "operatorKeeper")
	utils.PanicIfNil(k.stakingKeeper, "stakingKeeper")
	utils.PanicIfNil(k.delegationKeeper, "delegationKeeper")
	utils.PanicIfNil(k.clientKeeper, "clientKeeper")
	utils.PanicIfNil(k.portKeeper, "portKeeper")
	utils.PanicIfNil(k.scopedKeeper, "scopedKeeper")
	utils.PanicIfNil(k.channelKeeper, "channelKeeper")
	utils.PanicIfNil(k.connectionKeeper, "connectionKeeper")
	utils.PanicIfNil(k.accountKeeper, "accountKeeper")
}
