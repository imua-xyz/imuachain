package keeper

import (
	"github.com/imua-xyz/imuachain/x/exomint/types"
)

var _ types.QueryServer = Keeper{}
