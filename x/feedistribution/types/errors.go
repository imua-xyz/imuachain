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

	ErrInvalidCommunityTax = errorsmod.Register(
		ModuleName, 17,
		"invalid community tax",
	)

	ErrInvalidCliCmdArg = errorsmod.Register(
		ModuleName, 18,
		"the input client command arguments are invalid",
	)
	ErrFailedToRedelegateRewards = errorsmod.Register(
		ModuleName, 19,
		"failed to redelegate rewards",
	)

	ErrFailedToUndelegateRewards = errorsmod.Register(
		ModuleName, 20,
		"failed to undelegate rewards",
	)

	ErrFailedToCompleteRewardsUndelegation = errorsmod.Register(
		ModuleName, 21,
		"failed to complete rewards undelegation",
	)
	ErrFailedToSlashUnclaimedRewards = errorsmod.Register(
		ModuleName, 22,
		"failed to slash unclaimed rewards",
	)
)
