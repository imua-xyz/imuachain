package keeper

import (
	"fmt"

	commontypes "github.com/ExocoreNetwork/exocore/x/appchain/common/types"
	"github.com/ExocoreNetwork/exocore/x/appchain/coordinator/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
)

// OnRecvSlashPacket processes a slashing packet upon its receipt from
// the subscriber chain. At this point, it only handles DOWNTIME infractions.
// TODO: Design and implement EQUIVOCATION slashing.
// The returned value is a byte slice containing the acknowledgment to send to
// the sender. Otherwise, it should be an error.
func (k Keeper) OnRecvSlashPacket(
	ctx sdk.Context, packet channeltypes.Packet, data commontypes.SlashPacketData,
) ([]byte, error) {
	chainID, found := k.GetChainForChannel(ctx, packet.DestinationChannel)
	if !found {
		k.Logger(ctx).Error(
			"received slash packet for unknown channel",
			"channel", packet.DestinationChannel,
		)
		return nil, types.ErrUnknownSubscriberChannelID.Wrapf(
			"slash packet on %s", packet.DestinationChannel,
		)
	}
	// stateless validation
	if err := data.Validate(); err != nil {
		return nil, commontypes.ErrInvalidPacketData.Wrapf(
			"invalid slash packet: %s", err,
		)
	}
	// stateful validation
	if err := k.ValidateSlashPacket(ctx, chainID, data); err != nil {
		k.Logger(ctx).Error(
			"invalid slash packet",
			"error", err,
			"chainID", chainID,
			"vscID", data.ValsetUpdateID,
			"consensus address", fmt.Sprintf("%x", data.Validator.Address),
			"infraction type", data.Infraction,
		)
		return nil, commontypes.ErrInvalidPacketData.Wrapf(
			"invalid slash packet %s", err,
		)
	}
	// TODO: handle throttling of slash packets to ensure that malicious / misconfigured
	// appchains don't spam the coordinator with slash packets to produce repeated
	// slashing events. When throttling is implemented, indicate to the subscriber
	// that a packet wasn't handled and should be retried later.
	k.HandleSlashPacket(ctx, chainID, data)
	k.Logger(ctx).Info(
		"slash packet received and handled",
		"chainID", chainID,
		"consensus address", fmt.Sprintf("%x", data.Validator.Address),
		"vscID", data.ValsetUpdateID,
		"infractionType", data.Infraction,
	)

	// Return result ack that the packet was handled successfully
	return commontypes.SlashPacketHandledResult.Bytes(), nil
}

// OnRecvVscMaturedPacket handles a VscMatured packet and returns a no-op result ack.
func (k Keeper) OnRecvVscMaturedPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	data commontypes.VscMaturedPacketData,
) error {
	// check that the channel is established, panic if not
	chainID, found := k.GetChainForChannel(ctx, packet.DestinationChannel)
	if !found {
		// VSCMatured packet was sent on a channel different than any of the established
		// channels; this should never happen
		k.Logger(ctx).Error(
			"VscMaturedPacket received on unknown channel",
			"channelID", packet.DestinationChannel,
		)
		return types.ErrUnknownSubscriberChannelID.Wrapf(
			"vsc matured packet on %s", packet.DestinationChannel,
		)
	}

	k.HandleVscMaturedPacket(ctx, chainID, data)

	k.Logger(ctx).Info(
		"VscMaturedPacket handled",
		"chainID", chainID,
		"vscID", data.ValsetUpdateID,
	)

	return nil
}

// HandleVscMaturedPacket handles a VscMatured packet.
func (k Keeper) HandleVscMaturedPacket(
	sdk.Context, string, commontypes.VscMaturedPacketData,
) {
	// records := k.GetUndelegationsToMature(ctx, chainID, data.ValsetUpdateID)
	// // this is matured at EndBlock, because the delegation keeper only releases the funds
	// // at EndBlock. it is pointless to mature any of these now.
	// // do note that this is the reason that the EndBlocker of this module is triggered
	// // before that of the undelegation module.
	// k.AppendMaturedUndelegations(ctx, records)
	// k.ClearUndelegationsToMature(ctx, chainID, data.ValsetUpdateID)

	// operators := k.GetOptOutsToFinish(ctx, chainID, data.ValsetUpdateID)
	// k.AppendFinishedOptOutsForChainID(ctx, chainID, operators)
	// k.ClearOptOutsToFinish(ctx, chainID, data.ValsetUpdateID)

	// // if there are any opt outs, the key can be removed. similarly,
	// // if there are any key replacements, the old key should be pruned
	// addrs := k.GetConsensusKeysToPrune(ctx, chainID, data.ValsetUpdateID)
	// for _, addr := range addrs {
	// 	// this is pruned immediately so that an operator may reuse the same key immediately
	// 	k.Logger(ctx).Debug("pruning key", "addr", addr, "chainId", chainID)
	// 	k.operatorKeeper.DeleteOperatorAddressForChainIDAndConsAddr(ctx, chainID, addr)
	// }
	// k.ClearConsensusKeysToPrune(ctx, chainID, data.ValsetUpdateID)
}

// OnAcknowledgementPacket handles acknowledgments for sent VSC packets
func (k Keeper) OnAcknowledgementPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	ack channeltypes.Acknowledgement,
) error {
	if err := ack.GetError(); err != "" {
		k.Logger(ctx).Error(
			"recv ErrorAcknowledgement",
			"channelID", packet.SourceChannel,
			"error", err,
		)
		if chainID, ok := k.GetChainForChannel(ctx, packet.DestinationChannel); ok {
			return k.StopSubscriberChain(ctx, chainID, false)
		}
		return types.ErrUnknownSubscriberChannelID.Wrapf(
			"ack packet on %s", packet.DestinationChannel,
		)
	}
	return nil
}

// OnTimeoutPacket aborts the transaction if no chain exists for the destination channel,
// otherwise it stops the chain
func (k Keeper) OnTimeoutPacket(ctx sdk.Context, packet channeltypes.Packet) error {
	chainID, found := k.GetChainForChannel(ctx, packet.SourceChannel)
	if !found {
		k.Logger(ctx).Error(
			"packet timeout, unknown channel",
			"channelID", packet.SourceChannel,
		)
		return types.ErrUnknownSubscriberChannelID.Wrapf(
			"ack packet on %s", packet.DestinationChannel,
		)
	}
	// stop chain and release unbondings
	k.Logger(ctx).Info(
		"packet timeout, removing the subscriber",
		"chainID", chainID,
	)
	return k.StopSubscriberChain(ctx, chainID, false)
}

// StopSubscriberChain stops the subscriber chain and releases any unbondings.
// During the stoppage, it will prune any information that is no longer needed
// to save space.
// The closeChannel flag indicates whether the channel should be closed.
func (k Keeper) StopSubscriberChain(
	ctx sdk.Context, chainID string, closeChannel bool,
) error {
	k.Logger(ctx).Info(
		"stopping subscriber chain",
		"chainID", chainID,
		"closeChannel", closeChannel,
	)
	// not yet implemented
	return nil
}
