package oracle

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/imua-xyz/imuachain/x/oracle/keeper"
	"github.com/imua-xyz/imuachain/x/oracle/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	// Set all the prices
	for _, elem := range genState.PricesList {
		k.SetPrices(ctx, elem)
	}
	// Set if defined
	if genState.ValidatorUpdateBlock != nil {
		k.SetValidatorUpdateBlock(ctx, *genState.ValidatorUpdateBlock)
	}
	// Set if defined
	if genState.IndexRecentParams != nil {
		k.SetIndexRecentParams(ctx, *genState.IndexRecentParams)
	}
	// Set if defined
	if genState.IndexRecentMsg != nil {
		k.SetIndexRecentMsg(ctx, *genState.IndexRecentMsg)
	}
	// Set all the recentMsg
	for _, elem := range genState.RecentMsgList {
		k.SetRecentMsg(ctx, elem)
	}
	// Set all the recentParams
	for _, elem := range genState.RecentParamsList {
		k.SetRecentParams(ctx, elem)
	}
	// Set all stakerList for assetIDs
	for _, elem := range genState.StakerListAssets {
		k.SetStakerList(ctx, elem.AssetId, elem.StakerList)
	}
	// Set all stakerInfos for assetIDs
	for _, elem := range genState.StakerInfosAssets {
		k.SetStakerInfos(ctx, elem.AssetId, elem.StakerInfos)
	}
	// this line is used by starport scaffolding # genesis/module/init
	k.SetParams(ctx, genState.Params)
}

// ExportGenesis returns the module's exported genesis
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)

	genesis.PricesList = k.GetAllPrices(ctx)
	// Get all validatorUpdateBlock
	validatorUpdateBlock, found := k.GetValidatorUpdateBlock(ctx)
	if found {
		genesis.ValidatorUpdateBlock = &validatorUpdateBlock
	}
	// Get all indexRecentParams
	indexRecentParams, found := k.GetIndexRecentParams(ctx)
	if found {
		genesis.IndexRecentParams = &indexRecentParams
	}
	// Get all indexRecentMsg
	indexRecentMsg, found := k.GetIndexRecentMsg(ctx)
	if found {
		genesis.IndexRecentMsg = &indexRecentMsg
	}
	genesis.RecentMsgList = k.GetAllRecentMsg(ctx)
	genesis.RecentParamsList = k.GetAllRecentParams(ctx)
	// TODO: export stakerListAssets, and stakerInfosAssets
	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
