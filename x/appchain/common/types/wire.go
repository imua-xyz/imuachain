package types

import (
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// WrappedSubscriberPacketData is a wrapper interface for SubscriberPacketData. It allows
// exposting the private interface defined in `wire.pb.go` to the outside world.
type WrappedSubscriberPacketData interface {
	isSubscriberPacketData_Data
}

// NewSubscriberPacketData creates a new SubscriberPacketData instance.
func NewSubscriberPacketData(
	packetType SubscriberPacketDataType, packet isSubscriberPacketData_Data,
) SubscriberPacketData {
	return SubscriberPacketData{
		Type: packetType,
		Data: packet,
	}
}

// NewSlashPacketData creates a new SlashPacketData instance.
func NewSlashPacketData(
	validator abci.Validator,
	valUpdateID uint64,
	infractionType stakingtypes.Infraction,
) *SlashPacketData {
	return &SlashPacketData{
		Validator:      validator,
		ValsetUpdateID: valUpdateID,
		Infraction:     infractionType,
	}
}

// NewVscPacketData creates a new ValidatorSetChangePacketData instance.
func NewVscPacketData(
	updates []abci.ValidatorUpdate,
	valsetUpdateID uint64,
	slashAcks [][]byte,
) ValidatorSetChangePacketData {
	return ValidatorSetChangePacketData{
		ValidatorUpdates: updates,
		ValsetUpdateID:   valsetUpdateID,
		SlashAcks:        slashAcks,
	}
}

// NewVscPacketData creates a new VscMaturedPacketData instance.
func NewVscMaturedPacketData(
	valsetUpdateID uint64,
) *VscMaturedPacketData {
	return &VscMaturedPacketData{
		ValsetUpdateID: valsetUpdateID,
	}
}

// PacketAckResult is the acknowledgment result of a packet.
type PacketAckResult []byte

var (
	// VscPacketHandledResult is the success acknowledgment result of a validator set change packet.
	VscPacketHandledResult = PacketAckResult([]byte{byte(1)})
	// SlashPacketHandledResult is the success acknowledgment result of a slash packet.
	SlashPacketHandledResult = PacketAckResult([]byte{byte(2)})
)

// Validate validates the SlashPacketData. It only performs stateless validation.
// (1) The address must be a valid consensus address.
// (2) The power must be positive.
// (3) The infraction type must be downtime.
func (vdt SlashPacketData) Validate() error {
	// vdt.Validator.Address must be a consensus address
	if err := sdk.VerifyAddressFormat(vdt.Validator.Address); err != nil {
		return ErrInvalidPacketData.Wrapf("invalid validator: %s", err.Error())
	}
	// vdt.Validator.Power must be positive
	if vdt.Validator.Power <= 0 {
		return ErrInvalidPacketData.Wrap("validator power must be positive")
	}
	// ValsetUpdateId can be zero for the first validator set, so we don't validate it here.
	if vdt.Infraction != stakingtypes.Infraction_INFRACTION_DOWNTIME {
		// only downtime infractions are supported at this time
		return ErrInvalidPacketData.Wrapf("invalid infraction type: %s", vdt.Infraction.String())
	}

	return nil
}

// Bytes returns the byte representation of the PacketAckResult.
func (res PacketAckResult) Bytes() []byte {
	return res
}
