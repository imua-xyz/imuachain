package common

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// the input data could be either rawData bytes of data with big size for non-price senarios or 'price' info
type PostAggregationHandler func(data []byte, ctx sdk.Context, k KeeperOracle) error
