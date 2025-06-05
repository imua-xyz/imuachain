package keeper

import (
	"github.com/imua-xyz/imuachain/x/immint/types"
)

var _ types.QueryServer = Keeper{}
