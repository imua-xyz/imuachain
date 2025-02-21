package types

// DONTCOVER

import (
	errorsmod "cosmossdk.io/errors"
)

// x/slash module sentinel errors
var (
	ErrInvalidEvmAddressFormat  = errorsmod.Register(ModuleName, 2, "the evm address format is error")
	ErrInvalidLzUaTopicIDLength = errorsmod.Register(ModuleName, 3, "the LZUaTopicID length isn't equal to HashLength")
	ErrNoParamsKey              = errorsmod.Register(ModuleName, 4, "there is no stored key for slash module params")
	ErrSlashAmountIsNegative    = errorsmod.Register(ModuleName, 5, "the slash amount is negative")
	ErrSlashAssetNotExist       = errorsmod.Register(ModuleName, 6, "the slash asset doesn't exist")
	ErrNoOperatorStatusKey      = errorsmod.Register(ModuleName, 7, "there is no stored key for slash OpratorStatus")
)
