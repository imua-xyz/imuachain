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
		k.SetValidatorUpdateForCache(ctx, *genState.ValidatorUpdateBlock)
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
		k.SetNSTVersion(ctx, elem.AssetId, elem.NstVersion)
	}
	// set validatorReportInfos
	for _, elem := range genState.ValidatorReportInfos {
		k.SetValidatorReportInfo(ctx, elem.Address, elem)
	}
	// set validatorMissedRounds
	for _, elem := range genState.ValidatorMissedRounds {
		for _, missedRound := range elem.MissedRounds {
			k.SetValidatorMissedRoundBitArray(ctx, elem.Address, missedRound.Index, missedRound.Missed)
		}
	}
	// this line is used by starport scaffolding # genesis/module/init
	k.SetParams(ctx, genState.Params)
}

// ExportGenesis returns the module's exported genesis
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	// params
	genesis.Params = k.GetParams(ctx)

	// priceList
	genesis.PricesList = k.GetAllPrices(ctx)

	// cache recovery related, used by agc
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

	// NST related
	genesis.StakerInfosAssets = k.GetAllStakerInfosAssets(ctx)
	genesis.StakerListAssets = k.GetAllStakerListAssets(ctx)

	// slashing related
	reportInfos := make([]types.ValidatorReportInfo, 0)
	validatorMissedRounds := make([]types.ValidatorMissedRounds, 0)
	k.IterateValidatorReportInfos(ctx, func(validator string, reportInfo types.ValidatorReportInfo) bool {
		reportInfos = append(reportInfos, reportInfo)
		missedRounds := k.GetValidatorMissedRounds(ctx, validator)
		validatorMissedRounds = append(validatorMissedRounds, types.ValidatorMissedRounds{
			Address:      validator,
			MissedRounds: missedRounds,
		})
		return false
	})
	genesis.ValidatorReportInfos = reportInfos
	genesis.ValidatorMissedRounds = validatorMissedRounds
	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
