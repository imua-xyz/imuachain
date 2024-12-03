package types

import (
	math "math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	// ModuleName defines the module name
	ModuleName = "dogfood"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// RouterKey defines the module's message routing key
	RouterKey = ModuleName
)

const (
	// ExocoreValidatorBytePrefix is the prefix for the validator store.
	ExocoreValidatorBytePrefix byte = iota + 1

	// QueuedOperationsByte was the byte used to store the queue of operations.
	// It is no longer used, but it is retained for backward compatibility.
	_

	// OptOutsToFinishBytePrefix is the byte used to store the list of operator addresses whose
	// opt outs are maturing at the provided epoch.
	OptOutsToFinishBytePrefix

	// OperatorOptOutFinishEpochBytePrefix is the byte prefix to store the epoch at which an
	// operator's opt out will mature.
	OperatorOptOutFinishEpochBytePrefix

	// ConsensusAddrsToPruneBytePrefix is the byte prefix to store the list of consensus
	// addresses that can be pruned from the operator module at the provided epoch.
	ConsensusAddrsToPruneBytePrefix

	// UnbondingReleaseMaturityBytePrefix is the byte prefix to store the list of undelegations
	// that will mature at the provided epoch.
	UnbondingReleaseMaturityBytePrefix

	// PendingOperationsByte was the byte used to store the list of operations to be applied at
	// the end of the current block. It is no longer used, and is retained for backward
	// compatibility.
	_

	// PendingOptOutsByte is the byte used to store the list of operator addresses whose opt
	// outs will be made effective at the end of the current block.
	PendingOptOutsByte

	// PendingConsensusAddrsByte is the byte used to store the list of consensus addresses to be
	// pruned at the end of the block.
	PendingConsensusAddrsByte

	// PendingUndelegationsByte is the byte used to store the list of undelegations that will
	// mature at the end of the current block.
	PendingUndelegationsByte

	// ShouldUpdateValidatorSetByte is the byte key for the epoch end store.
	ShouldUpdateValidatorSetByte

	// HistoricalInfoBytePrefix is the byte prefix for the historical info store.
	HistoricalInfoBytePrefix

	// UndelegationMaturityEpochByte is the byte key for the undelegation maturity epoch store.
	UndelegationMaturityEpochByte

	// LastTotalPowerByte is the byte key for the last total power store.
	LastTotalPowerByte

	// ValidatorUpdatesByte is the byte key for the validator updates store.
	ValidatorUpdatesByte

	// ParamsByte is the single-byte key for the params store.
	ParamsByte
)

// ExocoreValidatorKey returns the key for the validator store.
func ExocoreValidatorKey(address sdk.AccAddress) []byte {
	return append([]byte{ExocoreValidatorBytePrefix}, address.Bytes()...)
}

// OptOutsToFinishKey returns the key for the operator opt out maturity store (epoch -> list of
// addresses).
func OptOutsToFinishKey(epoch int64) ([]byte, bool) {
	uepoch, ok := SafeInt64ToUint64(epoch)
	if !ok {
		return nil, false
	}
	return append(
		[]byte{OptOutsToFinishBytePrefix},
		sdk.Uint64ToBigEndian(uepoch)...,
	), true
}

// OperatorOptOutFinishEpochKey is the key for the operator opt out maturity store
// (sdk.AccAddress -> epoch)
func OperatorOptOutFinishEpochKey(address sdk.AccAddress) []byte {
	return append([]byte{OperatorOptOutFinishEpochBytePrefix}, address.Bytes()...)
}

// ConsensusAddrsToPruneKey is the key to lookup the list of operator consensus addresses that
// can be pruned from the operator module at the provided epoch.
func ConsensusAddrsToPruneKey(epoch int64) ([]byte, bool) {
	uepoch, ok := SafeInt64ToUint64(epoch)
	if !ok {
		return nil, false
	}
	return append(
		[]byte{ConsensusAddrsToPruneBytePrefix},
		sdk.Uint64ToBigEndian(uepoch)...,
	), true
}

// UnbondingReleaseMaturityKey is the key to lookup the list of undelegations that will mature
// at the provided epoch.
func UnbondingReleaseMaturityKey(epoch int64) ([]byte, bool) {
	uepoch, ok := SafeInt64ToUint64(epoch)
	if !ok {
		return nil, false
	}
	return append(
		[]byte{UnbondingReleaseMaturityBytePrefix},
		sdk.Uint64ToBigEndian(uepoch)...,
	), true
}

// PendingOptOutsKey returns the key for the pending opt-outs store.
func PendingOptOutsKey() []byte {
	return []byte{PendingOptOutsByte}
}

// PendingConsensusAddrsByte is the byte used to store the list of consensus addresses to be
// pruned at the end of the block.
func PendingConsensusAddrsKey() []byte {
	return []byte{PendingConsensusAddrsByte}
}

// PendingUndelegationsKey returns the key for the pending undelegations store.
func PendingUndelegationsKey() []byte {
	return []byte{PendingUndelegationsByte}
}

// ShouldUpdateValidatorSetByteKey returns the key for the epoch end store.
func ShouldUpdateValidatorSetByteKey() []byte {
	return []byte{ShouldUpdateValidatorSetByte}
}

// UndelegationMaturityEpochKey returns the key for the undelegation maturity epoch store.
func UndelegationMaturityEpochKey(recordKey []byte) []byte {
	return append([]byte{UndelegationMaturityEpochByte}, recordKey...)
}

// SafeInt64ToUint64 is a wrapper function to convert an int64
// to a uint64. It returns (0, false) if the conversion is not possible.
// This is safe as long as the int64 is non-negative.
func SafeInt64ToUint64(id int64) (uint64, bool) {
	if id < 0 {
		return 0, false
	}
	return uint64(id), true // #nosec G701 // already checked.
}

// SafeUint64ToInt64 is a wrapper function to convert a uint64
// to an int64. It returns (0, false) if the conversion is not possible,
// which happens when the uint64 is greater than the maximum int64 value.
func SafeUint64ToInt64(id uint64) (int64, bool) {
	if id > math.MaxInt64 {
		return 0, false
	}
	return int64(id), true // #nosec G115 // already checked.
}

// HistoricalInfoKey returns the key to historical info to a given block height
func HistoricalInfoKey(height int64) ([]byte, bool) {
	uheight, ok := SafeInt64ToUint64(height)
	if !ok {
		return nil, false
	}
	return append(
		[]byte{HistoricalInfoBytePrefix}, sdk.Uint64ToBigEndian(uheight)...,
	), true
}

// LastTotalPowerKey returns the key for the last total power store.
func LastTotalPowerKey() []byte {
	return []byte{LastTotalPowerByte}
}

// ValidatorUpdatesKey returns the key for the validator updates store.
func ValidatorUpdatesKey() []byte {
	return []byte{ValidatorUpdatesByte}
}

// ParamsKey returns the key for the params store.
func ParamsKey() []byte {
	return []byte{ParamsByte}
}
