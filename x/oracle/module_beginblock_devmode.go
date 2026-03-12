//go:build devmode

package oracle

import (
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/imua-xyz/imuachain/x/oracle/keeper"
	"github.com/imua-xyz/imuachain/x/oracle/keeper/feedermanagement"
)

// BeginBlock contains the logic that is automatically triggered at the beginning of each block
func (am AppModule) BeginBlock(ctx sdk.Context, req abci.RequestBeginBlock) {
	logger := am.keeper.Logger(ctx)
	am.keeper.BeginBlock(ctx, req)

	logger.Info("start simulating recovery in BeginBlock", "height", ctx.BlockHeight())
	// check the result of recovery so that node restart can replay to the same state
	f := recoveryFeederManagerOnNextBlock(ctx, am.keeper)
	if reason := am.keeper.FeederManager.EqualsWithReason(f); reason != "" {
		logger.Error("recovery vs live FeederManager mismatch", "block", ctx.BlockHeight(), "reason", reason)
		panic(fmt.Sprintf("there's something wrong in the recovery logic of feedermanager, block:%d, reason:%s", ctx.BlockHeight(), reason))
	}
}

func recoveryFeederManagerOnNextBlock(ctx sdk.Context, k keeper.Keeper) *feedermanagement.FeederManager {
	f := feedermanagement.NewFeederManager(k)
	recovered := f.BeginBlock(ctx)
	if ctx.BlockHeight() > 1 && !recovered {
		panic(fmt.Sprintf("failed to do recovery for feedermanager, block:%d", ctx.BlockHeight()))
	}
	return f
}
