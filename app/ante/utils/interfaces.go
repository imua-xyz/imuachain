package utils

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	oracletypes "github.com/imua-xyz/imuachain/x/oracle/types"
)

// BankKeeper defines the exposed interface for using functionality of the bank keeper
// in the context of the AnteHandler utils package.
type BankKeeper interface {
	// GetBalance returns the balance of a specific denomination for a given account address.
	GetBalance(ctx sdk.Context, addr sdk.AccAddress, denom string) sdk.Coin
}

// DistributionKeeper defines the exposed interface for using functionality of the distribution
// keeper in the context of the AnteHandler utils package.
type DistributionKeeper interface {
	WithdrawDelegationRewards(ctx sdk.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) (sdk.Coins, error)
}

// StakingKeeper defines the exposed interface for using functionality of the staking keeper
// in the context of the AnteHandler utils package.
type StakingKeeper interface {
	// BondDenom returns the denomination used for staking bonds.
	BondDenom(ctx sdk.Context) string

	// IterateDelegations iterates over all delegations for a given delegator, calling the provided function for each.
	IterateDelegations(ctx sdk.Context, delegator sdk.AccAddress, fn func(index int64, delegation stakingtypes.DelegationI) (stop bool))
}

// OracleKeeper defines the exposed interface for using functionality of the oracle keeper
type OracleKeeper interface {
	// CheckAndIncreaseNonce checks the current nonce for a validator and feeder, and increases it if valid.
	//
	// Parameters:
	//   - ctx: the current context for the operation
	//   - validator: the validator's consensus address (string)
	//   - feederID: the feeder's unique identifier
	//   - nonce: the nonce to check and increase
	//
	// Returns:
	//   - prevNonce: the previous nonce value
	//   - err: error if the nonce is invalid or cannot be increased
	CheckAndIncreaseNonce(ctx sdk.Context, validator string, feederID uint64, nonce uint32) (prevNonce uint32, err error)
	// NextPieceIndexByFeederID returns the next piece index for the given feeder ID
	NextPieceIndexByFeederID(ctx sdk.Context, feederID uint64) (uint32, bool)
	// CheckAndIncreaseToNextPieceIndex validates a feeder's piece index and increments it for the next transaction
	// It returns the updated piece index or an error if the validation fails.
	CheckAndIncreaseToNextPieceIndex(ctx sdk.Context, validator string, feederID uint64, nextPieceIndex uint32) (udpatedNextPieceIndex uint32, err error)
	// GetMaxNonceFromCache returns the maximum nonce value from the cache.
	GetMaxNonceFromCache() int32
	// GetPieceWithProof retrieves a piece with its proof for a given price creation message.
	GetPieceWithProof(msg *oracletypes.MsgCreatePrice) (*oracletypes.PieceWithProof, bool)
	// MinimalProofPathByIndex returns the minimal proof path for a given feeder ID and index.
	MinimalProofPathByIndex(feederID uint64, index uint32) []uint32
	// LatestRoundBaseBlock returns the base block of the latest round for a given feeder ID.
	LatestRoundBaseBlock(feederID uint64) (uint64, bool)
}
