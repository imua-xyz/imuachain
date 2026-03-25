package types

// DONTCOVER

import errorsmod "cosmossdk.io/errors"

var ErrInvalidGenesisData = errorsmod.Register(
	ModuleName, 2,
	"the genesis data supplied is invalid",
)

var ErrNotAVSByChainID = errorsmod.Register(
	ModuleName, 3,
	"AVS doesn't exist by chain ID",
)

var ErrUpdateAVSInfo = errorsmod.Register(
	ModuleName, 4,
	"failed to update AVS information",
)

var ErrConsKeyAlreadyTombstoned = errorsmod.Register(
	ModuleName, 5,
	"consensus key is already tombstoned",
)
