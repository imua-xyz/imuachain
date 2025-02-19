package keeper

import (
	"github.com/imua-xyz/imuachain/x/reward/types"
)

var _ types.QueryServer = Keeper{}
