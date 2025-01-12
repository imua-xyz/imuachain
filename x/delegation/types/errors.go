package types

import errorsmod "cosmossdk.io/errors"

var (
	ErrNoKeyInTheStore = errorsmod.Register(
		ModuleName, 2,
		"there is not the key for in the store",
	)
	ErrOperatorIsFrozen = errorsmod.Register(
		ModuleName, 3,
		"the operator has been frozen",
	)

	ErrOperatorNotExist = errorsmod.Register(
		ModuleName, 4,
		"the operator has not been registered",
	)

	ErrAmountIsNotPositive = errorsmod.Register(
		ModuleName, 5,
		"the amount isn't positive",
	)

	ErrOperatorAddrIsNotAccAddr = errorsmod.Register(
		ModuleName, 6,
		"the operator address isn't a valid acc addr",
	)

	ErrNotSupportYet = errorsmod.Register(
		ModuleName, 7,
		"don't have supported it yet",
	)

	ErrDelegationAmountTooBig = errorsmod.Register(
		ModuleName, 8,
		"the delegation amount is bigger than the canWithdraw amount",
	)

	ErrCannotIncHoldCount = errorsmod.Register(
		ModuleName, 9,
		"cannot increment undelegation hold count above max uint64",
	)

	ErrCannotDecHoldCount = errorsmod.Register(
		ModuleName, 10,
		"cannot decrement undelegation hold count below zero",
	)

	ErrInvalidGenesisData = errorsmod.Register(
		ModuleName, 11,
		"the genesis data supplied is invalid",
	)

	ErrDivisorIsZero = errorsmod.Register(
		ModuleName, 12,
		"the divisor is zero")

	ErrInsufficientShares = errorsmod.Register(
		ModuleName, 13,
		"insufficient delegation shares")

	ErrInvalidHash = errorsmod.Register(
		ModuleName, 14,
		"invalid hash",
	)

	ErrOperatorAlreadyAssociated = errorsmod.Register(
		ModuleName, 15,
		"the operator is already associated by this staker",
	)

	ErrNoAssociatedOperatorByStaker = errorsmod.Register(
		ModuleName, 16,
		"there isn't any operator marked by the staker",
	)

	ErrClientChainNotExist = errorsmod.Register(
		ModuleName, 17,
		"the client chain has not been registered",
	)
	ErrInvalidAssetID = errorsmod.Register(
		ModuleName, 18,
		"assetID is invalid",
	)
	ErrInvalidCompletionEpoch = errorsmod.Register(
		ModuleName, 23,
		"the epoch number to complete the undelegation is invalid",
	)
	ErrEpochIdentifierNotExist = errorsmod.Register(
		ModuleName, 24,
		"the epoch identifier doesn't exist",
	)

	ErrCannotIncUndelegationID = errorsmod.Register(
		ModuleName, 25,
		"cannot increment undelegation ID above max uint64",
	)
)
