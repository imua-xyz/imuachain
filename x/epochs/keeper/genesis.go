package keeper

import (
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/imua-xyz/imuachain/x/epochs/types"
)

// InitGenesis loads the initial state from a genesis file.
func (k Keeper) InitGenesis(ctx sdk.Context, genState types.GenesisState) []abci.ValidatorUpdate {
	for _, epoch := range genState.Epochs {
		// #nosec G703 // already validated that epoch is unique and valid
		_ = k.AddEpochInfo(ctx, epoch)
	}
	return nil
}

// ExportGenesis returns the module's exported genesis.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	return types.NewGenesisState(
		k.AllEpochInfos(ctx),
	)
}
