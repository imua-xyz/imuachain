//go:build devmode

package oracle

import (
	"github.com/ExocoreNetwork/exocore/x/oracle/keeper"
	"github.com/ExocoreNetwork/exocore/x/oracle/keeper/feedermanagement"
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// BeginBlock contains the logic that is automatically triggered at the beginning of each block
func (am AppModule) BeginBlock(ctx sdk.Context, _ abci.RequestBeginBlock) {
	am.keeper.BeginBlock(ctx)

	// check the result of recovery
	f := recoveryFeederManagerOnNextBlock(ctx, am.keeper)
	if ok := am.keeper.FeederManager.Equals(f); !ok {
		panic("there's something wrong in the recovery logic of feedermanager")
	}
}

func recoveryFeederManagerOnNextBlock(ctx sdk.Context, k keeper.Keeper) *feedermanagement.FeederManager {
	f := feedermanagement.NewFeederManager(k)
	recovered := f.BeginBlock(ctx)
	if ctx.BlockHeight() > 1 && !recovered {
		panic("failed to do recovery for feedermanager")
	}
	return f
}
