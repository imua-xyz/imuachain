package keeper

import (
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/imua-xyz/imuachain/x/avs/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
// Since this action typically occurs on chain starts, this function is allowed to panic.
func (k Keeper) InitGenesis(
	ctx sdk.Context,
	_ types.GenesisState,
) []abci.ValidatorUpdate {
	// Store a lookup from codeHash to code. Since these are static parameters,
	// such a lookup is stored at genesis and never updated.
	k.evmKeeper.SetCode(ctx, types.ChainIDCodeHash.Bytes(), types.ChainIDCode)
	return []abci.ValidatorUpdate{}
}

// ExportGenesis returns the module's exported genesis
func (Keeper) ExportGenesis(sdk.Context) *types.GenesisState {
	// TODO
	return types.DefaultGenesis()
}
