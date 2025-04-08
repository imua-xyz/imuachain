package types

// DONTCOVER

import (
	errorsmod "cosmossdk.io/errors"
)

// x/feedistribution module sentinel errors
var (
	ErrEpochNotFound = errorsmod.Register(
		ModuleName, 2,
		"Error: epoch info not found",
	)

	ErrNoKeyInTheStore = errorsmod.Register(
		ModuleName, 3,
		"there is no such key in the store",
	)

	ErrNotAVSRewardDistribution = errorsmod.Register(
		ModuleName, 4,
		"Error: avs reward distribution information not found",
	)

	ErrInvalidRewardAssetParameter = errorsmod.Register(
		ModuleName, 5,
		"invalid parameter of reward asset",
	)

	ErrAVSRewardAssetNotFound = errorsmod.Register(
		ModuleName, 6,
		"Error: the avs reward asset not found",
	)

	ErrInvalidRewardDistribution = errorsmod.Register(
		ModuleName, 7,
		"invalid parameter of reward distribution information",
	)

	ErrInvalidJailOrUnJailHeight = errorsmod.Register(
		ModuleName, 8,
		"invalid height of jail or unJail",
	)

	ErrNegativeCoinAmount = errorsmod.Register(
		ModuleName, 9,
		"negative coin amount",
	)

	ErrInvalidAssetUSDValue = errorsmod.Register(
		ModuleName, 10,
		"invalid USD value of asset",
	)

	ErrInvalidInputParameter = errorsmod.Register(
		ModuleName, 11,
		"invalid input parameter",
	)

	ErrNegativeAVSRewards = errorsmod.Register(
		ModuleName, 12,
		"negative avs rewards",
	)

	ErrInvalidStartingInfo = errorsmod.Register(
		ModuleName, 13,
		"invalid starting information for a delegation",
	)

	ErrInvalidGenesisData = errorsmod.Register(
		ModuleName, 14,
		"the genesis data supplied is invalid",
	)

	ErrFailedToAllocateRewardsForOperators = errorsmod.Register(
		ModuleName, 15,
		"failed to allocate the rewards to operators of an AVS",
	)

	ErrInvalidImuaReceiptAddr = errorsmod.Register(
		ModuleName, 16,
		"invalid imua receipt address",
	)
)
