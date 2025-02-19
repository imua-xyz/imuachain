package keeper

import (
	"github.com/imua-xyz/imuachain/x/oracle/types"
)

var _ types.QueryServer = Keeper{}
