package types

// DONTCOVER

import errorsmod "cosmossdk.io/errors"

const (
	errCodeInvalidParams = iota + 2
)

var ErrInvalidParams = errorsmod.Register(
	ModuleName, errCodeInvalidParams,
	"the sanitized params are invalid",
)
