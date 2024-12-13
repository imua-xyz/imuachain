package types

import (
	errorsmod "cosmossdk.io/errors"
)

// errors
var (
	ErrNoClientChainKey = errorsmod.Register(
		ModuleName, 2,
		"there is no stored key for the input chain index",
	)

	ErrNoClientChainAssetKey = errorsmod.Register(
		ModuleName, 3,
		"there is no stored key for the input assetID",
	)

	ErrNoStakerAssetKey = errorsmod.Register(
		ModuleName, 4,
		"there is no stored key for the input staker and assetID",
	)

	ErrSubAmountIsMoreThanOrigin = errorsmod.Register(
		ModuleName, 5,
		"the amount that want to decrease is more than the original state amount",
	)

	ErrNoOperatorAssetKey = errorsmod.Register(
		ModuleName, 6,
		"there is no stored key for the input operator address and assetID",
	)

	ErrParseAssetsStateKey = errorsmod.Register(
		ModuleName, 7,
		"assets state key can't be parsed",
	)

	ErrInvalidCliCmdArg = errorsmod.Register(
		ModuleName, 8,
		"the input client command arguments are invalid",
	)

	ErrInputPointerIsNil = errorsmod.Register(
		ModuleName, 9,
		"the input pointer is nil",
	)

	ErrInvalidOperatorAddr = errorsmod.Register(
		ModuleName, 10,
		"the operator address isn't a valid account address",
	)

	ErrInvalidEvmAddressFormat = errorsmod.Register(
		ModuleName, 11,
		"the evm address format is error",
	)

	ErrInvalidLzUaTopicIDLength = errorsmod.Register(
		ModuleName, 12,
		"the LZUaTopicID length isn't equal to HashLength",
	)

	ErrNoParamsKey = errorsmod.Register(
		ModuleName, 13,
		"there is no stored key for deposit module params",
	)

	ErrNotAuthorizedGateway = errorsmod.Register(
		ModuleName, 14,
		"the address isn't authorized to be a gateway",
	)

	ErrInvalidGenesisData = errorsmod.Register(
		ModuleName, 15,
		"the genesis data supplied is invalid",
	)

	ErrInvalidInputParameter = errorsmod.Register(
		ModuleName, 16,
		"the input parameter is invalid",
	)

	ErrInvalidAmount = errorsmod.Register(
		ModuleName, 17,
		"the amount is invalid")

	ErrInvalidOperationType = errorsmod.Register(
		ModuleName, 18,
		"the operation type is invalid")

	ErrRegisterDuplicateAssetID = errorsmod.Register(
		ModuleName, 19,
		"register new asset with an existing assetID")

	ErrParseJoinedKey = errorsmod.Register(
		ModuleName, 20,
		"the joined key can't be parsed",
	)
)
