package keeper

import (
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/imua-xyz/imuachain/x/appchain/coordinator/types"
)

func (k Keeper) InitGenesis(ctx sdk.Context, gs types.GenesisState) []abci.ValidatorUpdate {
	k.SetParams(ctx, gs.Params)
	return []abci.ValidatorUpdate{}
}

func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	return &types.GenesisState{
		Params: k.GetParams(ctx),
	}
}
