package common

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// PostAggregationHandler defines a function that processes data after the initial aggregation phase.
// It is called during the second phase of the two-phases aggregation process.
// Parameters:
//   - ctx: the SDK context for the transaction
//   - data: raw data bytes (for non-price scenarios)
//   - feederID: the unique identifier of the feeder
//   - roundID: the round identifier for the current aggregation cycle
//   - k: the oracle keeper interface providing access to the module state
//
// Returns an error if the post-aggregation processing fails
type PostAggregationHandler func(ctx sdk.Context, data []byte, feederID, roundID uint64, k KeeperOracle) error
