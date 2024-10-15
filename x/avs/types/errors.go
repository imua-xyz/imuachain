package types

// DONTCOVER

import (
	errorsmod "cosmossdk.io/errors"
)

// x/avs module sentinel errors
var (
	ErrNoKeyInTheStore = errorsmod.Register(ModuleName, 2, "there is no such key in the store")

	ErrAlreadyRegistered = errorsmod.Register(
		ModuleName, 3,
		"Error: Already registered",
	)
	ErrUnregisterNonExistent = errorsmod.Register(
		ModuleName, 4,
		"Error: No available avs ",
	)

	ErrInvalidAction = errorsmod.Register(
		ModuleName, 5,
		"Error: Undefined action",
	)
	ErrUnbondingPeriod = errorsmod.Register(
		ModuleName, 6,
		"Error: UnbondingPeriod check failed",
	)
	ErrEpochNotFound = errorsmod.Register(
		ModuleName, 7,
		"Error: epoch info not found",
	)
	ErrCallerAddressUnauthorized = errorsmod.Register(
		ModuleName, 8,
		"Error: caller address is not authorized",
	)
	ErrAvsNameMismatch = errorsmod.Register(
		ModuleName, 9,
		"Error: avs Name mismatch",
	)
	ErrNotYetRegistered = errorsmod.Register(
		ModuleName, 10,
		"Error: this AVS has not been registered yet",
	)
	ErrNotNull = errorsmod.Register(
		ModuleName, 11,
		"Error: param shouldn't be null",
	)

	ErrInvalidAddr = errorsmod.Register(
		ModuleName, 12,
		"The address isn't a valid account address",
	)
	ErrAlreadyExists = errorsmod.Register(
		ModuleName, 13,
		"The record already exists",
	)
	ErrTaskIsNotExists = errorsmod.Register(
		ModuleName, 14,
		"The task does not exist",
	)
	ErrHashValue = errorsmod.Register(
		ModuleName, 15,
		"The task response hash does not match the expected value.",
	)

	ErrPubKeyIsNotExists = errorsmod.Register(
		ModuleName, 16,
		"The pubKey does not exist",
	)
	ErrSigVerifyError = errorsmod.Register(
		ModuleName, 17,
		"Signature verification error",
	)

	ErrParamError = errorsmod.Register(
		ModuleName, 18,
		"The parameter must be 1 or 2",
	)

	ErrParamNotEmptyError = errorsmod.Register(
		ModuleName, 19,
		"In the first stage, the parameter must be empty.",
	)
	ErrSubmitTooLateError = errorsmod.Register(
		ModuleName, 20,
		" The response was submitted too late.",
	)
	ErrResAlreadyExists = errorsmod.Register(
		ModuleName, 21,
		"The submit result already exists",
	)
	ErrInconsistentParams = errorsmod.Register(
		ModuleName, 22,
		"Inconsistent parameters",
	)
	ErrVotingPowerIncorrect = errorsmod.Register(
		ModuleName, 23,
		"Voting power must be greater than 0",
	)
	ErrSigNotMatchPubKey = errorsmod.Register(
		ModuleName, 24,
		"Signature and pubkey do not match",
	)
	ErrParsePubKey = errorsmod.Register(
		ModuleName, 25,
		"The pubKey parsing failed",
	)
	ErrInvalidAssetID = errorsmod.Register(
		ModuleName, 26,
		"Invalid asset ID",
	)
	ErrSubmitTooSoonError = errorsmod.Register(
		ModuleName, 27,
		" The response was submitted too soon.",
	)
)
