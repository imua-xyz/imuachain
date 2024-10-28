package keeper

import (
	"fmt"
	"time"

	commontypes "github.com/ExocoreNetwork/exocore/x/appchain/common/types"
	"github.com/ExocoreNetwork/exocore/x/appchain/subscriber/types"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	host "github.com/cosmos/ibc-go/v7/modules/core/24-host"
)

type Keeper struct {
	cdc               codec.BinaryCodec
	storeKey          storetypes.StoreKey
	accountKeeper     commontypes.AccountKeeper
	bankKeeper        commontypes.BankKeeper
	scopedKeeper      commontypes.ScopedKeeper
	portKeeper        commontypes.PortKeeper
	clientKeeper      commontypes.ClientKeeper
	connectionKeeper  commontypes.ConnectionKeeper
	channelKeeper     commontypes.ChannelKeeper
	ibcCoreKeeper     commontypes.IBCCoreKeeper
	ibcTransferKeeper commontypes.IBCTransferKeeper
	feeCollectorName  string
	subscriberHooks   commontypes.SubscriberHooks
}

// NewKeeper creates a new subscriber keeper.
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	accountKeeper commontypes.AccountKeeper,
	bankKeeper commontypes.BankKeeper,
	scopedKeeper commontypes.ScopedKeeper,
	portKeeper commontypes.PortKeeper,
	clientKeeper commontypes.ClientKeeper,
	connectionKeeper commontypes.ConnectionKeeper,
	channelKeeper commontypes.ChannelKeeper,
	ibcCoreKeeper commontypes.IBCCoreKeeper,
	ibcTransferKeeper commontypes.IBCTransferKeeper,
	feeCollectorName string,
) Keeper {
	return Keeper{
		cdc:               cdc,
		storeKey:          storeKey,
		accountKeeper:     accountKeeper,
		bankKeeper:        bankKeeper,
		scopedKeeper:      scopedKeeper,
		portKeeper:        portKeeper,
		clientKeeper:      clientKeeper,
		connectionKeeper:  connectionKeeper,
		channelKeeper:     channelKeeper,
		ibcCoreKeeper:     ibcCoreKeeper,
		ibcTransferKeeper: ibcTransferKeeper,
		feeCollectorName:  feeCollectorName,
	}
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
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

// IsBound checks if the IBC app module is already bound to the desired port
func (k Keeper) IsBound(ctx sdk.Context, portID string) bool {
	_, ok := k.scopedKeeper.GetCapability(ctx, host.PortPath(portID))
	return ok
}

// BindPort defines a wrapper function for the port Keeper's function in
// order to expose it to module's InitGenesis function
func (k Keeper) BindPort(ctx sdk.Context, portID string) error {
	capability := k.portKeeper.BindPort(ctx, portID)
	return k.ClaimCapability(ctx, capability, host.PortPath(portID))
}

// ClaimCapability allows the IBC app module to claim a capability that core IBC
// passes to it
func (k Keeper) ClaimCapability(
	ctx sdk.Context,
	cap *capabilitytypes.Capability,
	name string,
) error {
	return k.scopedKeeper.ClaimCapability(ctx, cap, name)
}

// GetPendingChanges gets the pending validator set changes that will be applied
// at the end of this block.
func (k Keeper) GetPendingChanges(
	ctx sdk.Context,
) *commontypes.ValidatorSetChangePacketData {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.PendingChangesKey())
	if bz == nil {
		return nil
	}
	res := &commontypes.ValidatorSetChangePacketData{}
	k.cdc.MustUnmarshal(bz, res)
	return res
}

// SetPendingChanges sets the pending validator set changes that will be applied
// at the end of this block.
func (k Keeper) SetPendingChanges(
	ctx sdk.Context,
	data *commontypes.ValidatorSetChangePacketData,
) {
	store := ctx.KVStore(k.storeKey)
	bz := k.cdc.MustMarshal(data)
	store.Set(types.PendingChangesKey(), bz)
}

// DeletePendingChanges deletes the pending validator set changes that will be applied
// at the end of this block.
func (k Keeper) DeletePendingChanges(ctx sdk.Context) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(types.PendingChangesKey())
}

// SetPacketMaturityTime sets the maturity time for a given received VSC packet id
func (k Keeper) SetPacketMaturityTime(
	ctx sdk.Context, vscID uint64, maturityTime time.Time,
) {
	store := ctx.KVStore(k.storeKey)
	maturingVSCPacket := &types.MaturingVSCPacket{
		ValidatorSetChangeID: vscID,
		MaturityTime:         maturityTime,
	}
	store.Set(
		types.PacketMaturityTimeKey(maturityTime, vscID),
		k.cdc.MustMarshal(maturingVSCPacket),
	)
}

// GetElapsedVscPackets returns all VSC packets that have matured as of the current block time
func (k Keeper) GetElapsedVscPackets(ctx sdk.Context) []types.MaturingVSCPacket {
	store := ctx.KVStore(k.storeKey)
	prefix := []byte{types.PacketMaturityTimeBytePrefix}
	iterator := sdk.KVStorePrefixIterator(store, prefix)
	defer iterator.Close()

	var ret []types.MaturingVSCPacket
	for ; iterator.Valid(); iterator.Next() {
		var packet types.MaturingVSCPacket
		k.cdc.MustUnmarshal(iterator.Value(), &packet)
		// since these are stored in order of maturity time, we can break early
		if ctx.BlockTime().Before(packet.MaturityTime) {
			break
		}
		ret = append(ret, packet)
	}
	return ret
}

// DeletePacketMaturityTime deletes the maturity time for a given received VSC packet id
func (k Keeper) DeletePacketMaturityTime(
	ctx sdk.Context, vscID uint64, maturityTime time.Time,
) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(types.PacketMaturityTimeKey(maturityTime, vscID))
}

// DeleteOutstandingDowntime deletes the outstanding downtime flag for the given validator
// consensus address
func (k Keeper) DeleteOutstandingDowntime(
	ctx sdk.Context, consAddress sdk.ConsAddress,
) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(types.OutstandingDowntimeKey(consAddress))
}

// SetOutstandingDowntime sets the outstanding downtime flag for the given validator
func (k Keeper) SetOutstandingDowntime(
	ctx sdk.Context, consAddress sdk.ConsAddress,
) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.OutstandingDowntimeKey(consAddress), []byte{1})
}

// HasOutstandingDowntime returns true if the given validator has an outstanding downtime
func (k Keeper) HasOutstandingDowntime(
	ctx sdk.Context, consAddress sdk.ConsAddress,
) bool {
	store := ctx.KVStore(k.storeKey)
	return store.Has(types.OutstandingDowntimeKey(consAddress))
}

// NewEmptyKeeper creates a new empty subscriber keeper. It is used for temporary initialization
// of the keeper in app.go file.
func NewEmptyKeeper(
	cdc codec.BinaryCodec,
	key storetypes.StoreKey,
) Keeper {
	return Keeper{
		cdc:      cdc,
		storeKey: key,
	}
}

// SetHooks sets the hooks for the subscriber keeper. It panics if hooks are already set, or
// if an empty hooks object is supplied.
func (k *Keeper) SetHooks(sh commontypes.SubscriberHooks) *Keeper {
	if k.subscriberHooks != nil {
		panic("cannot set subscriber hooks twice")
	}
	if sh == nil {
		panic("cannot set nil subscriber hooks")
	}
	k.subscriberHooks = sh
	return k
}

// Hooks returns the hooks registered to the module.
func (k Keeper) Hooks() commontypes.SubscriberHooks {
	if k.subscriberHooks != nil {
		return k.subscriberHooks
	}
	return commontypes.MultiSubscriberHooks{}
}
