package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/feedistribution module sentinel errors
var (
	ErrEpochNotFound = sdkerrors.Register(
		ModuleName, 2,
		"Error: epoch info not found",
	)

	ErrNotAVSRewardDistribution = sdkerrors.Register(
		ModuleName, 3,
		"Error: avs reward distribution information not found",
	)

	ErrOperatorNotFound = sdkerrors.Register(
		ModuleName, 4,
		"Error: the operator not found by the validator consensus key",
	)

	ErrInvalidRewardAssetParameter = sdkerrors.Register(
		ModuleName, 5,
		"invalid parameter of reward asset",
	)

	ErrAVSRewardAssetNotFound = sdkerrors.Register(
		ModuleName, 6,
		"Error: the avs reward asset not found",
	)

	ErrInvalidRewardDistribution = sdkerrors.Register(
		ModuleName, 7,
		"invalid parameter of reward distribution information",
	)

	ErrInvalidJailOrUnJailHeight = sdkerrors.Register(
		ModuleName, 8,
		"invalid height of jail or unJail",
	)
)
