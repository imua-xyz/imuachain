package keeper

import (
	"time"

	commontypes "github.com/ExocoreNetwork/exocore/x/appchain/common/types"
	types "github.com/ExocoreNetwork/exocore/x/appchain/coordinator/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ValidateSlashPacket validates a slashing packet. It checks that
// (1) the valset update id maps back to an Exocore height, and
// (2) the validator (cons) address maps back to an operator account address.
// The caller must perform stateless validation by themselves.
func (k Keeper) ValidateSlashPacket(
	ctx sdk.Context, chainID string, data commontypes.SlashPacketData,
) error {
	// the validator set is generated at each epoch and following each slash. even if the set
	// does not change at each epoch, we send an empty update anyway since, otherwise, the
	// subscriber could time out the coordinator.
	if k.GetHeightForChainVscID(ctx, chainID, data.ValsetUpdateID) == 0 {
		return commontypes.ErrInvalidPacketData.Wrapf(
			"invalid chainID %s valsetUpdateID %d", chainID, data.ValsetUpdateID,
		)
	}
	// the second step is to find the operator account address against the consensus address
	// of the validator. if the operator is not found, the slashing packet is invalid.
	// note that this works even if the operator changes their consensus key, as long as the
	// key hasn't yet been pruned from the operator module.
	if found, _ := k.operatorKeeper.GetOperatorAddressForChainIDAndConsAddr(
		ctx, chainID, sdk.ConsAddress(data.Validator.Address),
	); !found {
		// don't bech32 encode it in the error since the appchain may have a different prefix
		return commontypes.ErrInvalidPacketData.Wrapf(
			"operator not found %x", data.Validator.Address,
		)
	}
	return nil
}

// HandleSlashPacket handles a slashing packet. The caller must ensure that the slashing packet
// is valid before calling this function. The function forwards the slashing request to the
// operator module, which will trigger a slashing hook and thus a validator set update containing
// the slashing acknowledgment.
func (k Keeper) HandleSlashPacket(ctx sdk.Context, chainID string, data commontypes.SlashPacketData) {
	consAddress := sdk.ConsAddress(data.Validator.Address)
	// never 0, since already validated
	height := k.GetHeightForChainVscID(ctx, chainID, data.ValsetUpdateID)
	// guaranteed to exist, since already validated
	_, operatorAccAddress := k.operatorKeeper.GetOperatorAddressForChainIDAndConsAddr(
		ctx, chainID, consAddress,
	)
	slashProportion := k.GetSubSlashFractionDowntime(ctx, chainID)
	// #nosec G703 // already validated
	slashProportionDecimal, _ := sdk.NewDecFromStr(slashProportion)
	jailDuration := k.GetSubDowntimeJailDuration(ctx, chainID)
	_, avsAddress := k.avsKeeper.IsAVSByChainID(ctx, chainID)
	// the slashing hook should trigger a validator set update for all affected AVSs. since the `chainID` is one of them
	// we should make sure we are well set up for that update. we will include an ack of the slash packet in the next
	// validator set update; record that here.
	k.AppendSlashAck(ctx, chainID, consAddress)
	if err := k.operatorKeeper.ApplySlashForHeight(
		ctx, operatorAccAddress, avsAddress, height,
		slashProportionDecimal, data.Infraction, jailDuration,
	); err != nil {
		k.Logger(ctx).Error(
			"failed to apply slash for height",
			"chainID", chainID, "height", height, "consAddress", consAddress, "error", err,
		)
	}
}

// AppendSlashAck appends a slashing acknowledgment for a chain, to be sent in the next validator set update.
func (k Keeper) AppendSlashAck(ctx sdk.Context, chainID string, consAddress sdk.ConsAddress) {
	prev := k.GetSlashAcks(ctx, chainID)
	prev.List = append(prev.List, consAddress)
	k.SetSlashAcks(ctx, chainID, prev)
}

// GetSlashAcks gets the slashing acknowledgments for a chain, to be sent in the next validator set update.
func (k Keeper) GetSlashAcks(ctx sdk.Context, chainID string) types.ConsensusAddresses {
	store := ctx.KVStore(k.storeKey)
	consAddresses := types.ConsensusAddresses{}
	key := types.SlashAcksKey(chainID)
	value := store.Get(key)
	k.cdc.MustUnmarshal(value, &consAddresses)
	return consAddresses
}

// SetSlashAcks sets the slashing acknowledgments for a chain, to be sent in the next validator set update.
func (k Keeper) SetSlashAcks(ctx sdk.Context, chainID string, consAddresses types.ConsensusAddresses) {
	store := ctx.KVStore(k.storeKey)
	key := types.SlashAcksKey(chainID)
	if len(consAddresses.List) == 0 {
		store.Delete(key)
	} else {
		store.Set(key, k.cdc.MustMarshal(&consAddresses))
	}
}

// ConsumeSlashAcks consumes the slashing acknowledgments for a chain, to be sent in the next validator set update.
func (k Keeper) ConsumeSlashAcks(ctx sdk.Context, chainID string) [][]byte {
	ret := k.GetSlashAcks(ctx, chainID)
	k.SetSlashAcks(ctx, chainID, types.ConsensusAddresses{})
	return ret.List
}

// TODO: these fields should be in the AVS keeper instead.
// SetSubSlashFractionDowntime sets the sub slash fraction downtime for a chain
func (k Keeper) SetSubSlashFractionDowntime(ctx sdk.Context, chainID string, fraction string) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.SubSlashFractionDowntimeKey(chainID), []byte(fraction))
}

// GetSubSlashFractionDowntime gets the sub slash fraction downtime for a chain
func (k Keeper) GetSubSlashFractionDowntime(ctx sdk.Context, chainID string) string {
	store := ctx.KVStore(k.storeKey)
	key := types.SubSlashFractionDowntimeKey(chainID)
	return string(store.Get(key))
}

// SetSubSlashFractionDoubleSign sets the sub slash fraction double sign for a chain
func (k Keeper) SetSubSlashFractionDoubleSign(ctx sdk.Context, chainID string, fraction string) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.SubSlashFractionDoubleSignKey(chainID), []byte(fraction))
}

// GetSubSlashFractionDoubleSign gets the sub slash fraction double sign for a chain
func (k Keeper) GetSubSlashFractionDoubleSign(ctx sdk.Context, chainID string) string {
	store := ctx.KVStore(k.storeKey)
	key := types.SubSlashFractionDoubleSignKey(chainID)
	return string(store.Get(key))
}

// SetSubDowntimeJailDuration sets the sub downtime jail duration for a chain
func (k Keeper) SetSubDowntimeJailDuration(ctx sdk.Context, chainID string, duration time.Duration) {
	store := ctx.KVStore(k.storeKey)
	// duration is always positive
	store.Set(types.SubDowntimeJailDurationKey(chainID), sdk.Uint64ToBigEndian(uint64(duration)))
}

// GetSubDowntimeJailDuration gets the sub downtime jail duration for a chain
func (k Keeper) GetSubDowntimeJailDuration(ctx sdk.Context, chainID string) time.Duration {
	store := ctx.KVStore(k.storeKey)
	key := types.SubDowntimeJailDurationKey(chainID)
	return time.Duration(sdk.BigEndianToUint64(store.Get(key)))
}
