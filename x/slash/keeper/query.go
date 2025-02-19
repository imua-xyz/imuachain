package keeper

import (
	"github.com/imua-xyz/imuachain/x/slash/types"
)

var _ types.QueryServer = Keeper{}
