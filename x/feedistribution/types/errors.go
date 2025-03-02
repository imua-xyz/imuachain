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
)
