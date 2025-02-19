//go:build devmode

package oracle

import (
	"fmt"

	"github.com/ExocoreNetwork/exocore/x/oracle/keeper"
	"github.com/ExocoreNetwork/exocore/x/oracle/keeper/feedermanagement"
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// BeginBlock contains the logic that is automatically triggered at the beginning of each block
func (am AppModule) BeginBlock(ctx sdk.Context, _ abci.RequestBeginBlock) {
	logger := am.keeper.Logger(ctx)
	am.keeper.BeginBlock(ctx)

	logger.Info("start simulating recovery in BeginBlock", "height", ctx.BlockHeight())
	// check the result of recovery
	f := recoveryFeederManagerOnNextBlock(ctx, am.keeper)
	if ok := am.keeper.FeederManager.Equals(f); !ok {
		panic(fmt.Sprintf("there's something wrong in the recovery logic of feedermanager, block:%d", ctx.BlockHeight()))
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
