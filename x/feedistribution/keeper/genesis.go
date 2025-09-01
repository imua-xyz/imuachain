package keeper

import (
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	feedistributiontypes "github.com/imua-xyz/imuachain/x/feedistribution/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func (k Keeper) InitGenesis(ctx sdk.Context, genState feedistributiontypes.GenesisState) {
	// init the state from the general exporting genesis file
	k.SetParams(ctx, genState.Params)
	err := k.SetAllAVSRewardAssets(ctx, genState.AllAvsRewardAssets)
	if err != nil {
		panic(errorsmod.Wrap(err, "failed to set all avs reward assets"))
	}
	err = k.SetAllAVSRewardParams(ctx, genState.AllAvsRewardParams)
	if err != nil {
		panic(errorsmod.Wrap(err, "failed to set all avs reward params"))
	}
	err = k.SetAllAVSFeePools(ctx, genState.AllAvsFeePools)
	if err != nil {
		panic(errorsmod.Wrap(err, "failed to set all avs fee pools"))
	}
	err = k.SetAllAVSRewardDistributions(ctx, genState.AllAvsRewardDistributions)
	if err != nil {
		panic(errorsmod.Wrap(err, "failed to set all avs reward distributions"))
	}
	err = k.SetAllOperatorOutstandingRewards(ctx, genState.AllOperatorOutstandingRewards)
	if err != nil {
		panic(errorsmod.Wrap(err, "failed to set all operator outstanding rewards"))
	}
	err = k.SetAllDelegationChangeInfo(ctx, genState.AllDelegationChangeInfos)
	if err != nil {
		panic(errorsmod.Wrap(err, "failed to set all delegation change info"))
	}
	err = k.SetAllDelegationStartingInfo(ctx, genState.AllDelegationStartingInfos)
	if err != nil {
		panic(errorsmod.Wrap(err, "failed to set all delegation starting info"))
	}
	err = k.SetAllOperatorHistoricalRewards(ctx, genState.AllOperatorHistoricalRewards)
	if err != nil {
		panic(errorsmod.Wrap(err, "failed to set all operator historical rewards"))
	}
	err = k.SetAllOperatorCurrentRewards(ctx, genState.AllOperatorCurrentRewards)
	if err != nil {
		panic(errorsmod.Wrap(err, "failed to set all operator current rewards"))
	}
	err = k.SetAllOperatorCommission(ctx, genState.AllOperatorCommission)
	if err != nil {
		panic(errorsmod.Wrap(err, "failed to set all operator commissions"))
	}
	err = k.SetAllOperatorSlashEvent(ctx, genState.AllOperatorSlashEvents)
	if err != nil {
		panic(errorsmod.Wrap(err, "failed to set all operator slash events"))
	}
	err = k.SetAllStakerClaimedRewards(ctx, genState.AllStakerClaimedRewards)
	if err != nil {
		panic(errorsmod.Wrap(err, "failed to set all staker claimed rewards"))
	}
}

// ExportGenesis returns the module's exported genesis
func (k Keeper) ExportGenesis(ctx sdk.Context) *feedistributiontypes.GenesisState {
	genesis := feedistributiontypes.GenesisState{}
	genesis.Params = k.GetParams(ctx)

	var err error
	genesis.AllAvsRewardAssets, err = k.GetAllAVSRewardAssets(ctx)
	if err != nil {
		panic(errorsmod.Wrap(err, "failed to get all avs reward assets"))
	}
	genesis.AllAvsRewardParams, err = k.GetAllAVSRewardParams(ctx)
	if err != nil {
		panic(errorsmod.Wrap(err, "failed to get all avs reward params"))
	}
	genesis.AllAvsFeePools, err = k.GetAllAVSFeePools(ctx)
	if err != nil {
		panic(errorsmod.Wrap(err, "failed to get all avs fee pools"))
	}
	genesis.AllAvsRewardDistributions, err = k.GetAllAVSRewardDistributions(ctx)
	if err != nil {
		panic(errorsmod.Wrap(err, "failed to get all avs reward distributions"))
	}
	genesis.AllOperatorOutstandingRewards, err = k.GetAllOperatorOutstandingRewards(ctx)
	if err != nil {
		panic(errorsmod.Wrap(err, "failed to get all operator outstanding rewards"))
	}
	genesis.AllDelegationChangeInfos, err = k.GetAllDelegationChangeInfo(ctx)
	if err != nil {
		panic(errorsmod.Wrap(err, "failed to get all delegation change info"))
	}
	genesis.AllDelegationStartingInfos, err = k.GetAllDelegationStartingInfo(ctx)
	if err != nil {
		panic(errorsmod.Wrap(err, "failed to get all delegation starting info"))
	}
	genesis.AllOperatorHistoricalRewards, err = k.GetAllOperatorHistoricalRewards(ctx)
	if err != nil {
		panic(errorsmod.Wrap(err, "failed to get all operator historical rewards"))
	}
	genesis.AllOperatorCurrentRewards, err = k.GetAllOperatorCurrentRewards(ctx)
	if err != nil {
		panic(errorsmod.Wrap(err, "failed to get all operator current rewards"))
	}
	genesis.AllOperatorCommission, err = k.GetAllOperatorCommission(ctx)
	if err != nil {
		panic(errorsmod.Wrap(err, "failed to get all operator commissions"))
	}
	genesis.AllOperatorSlashEvents, err = k.GetAllOperatorSlashEvent(ctx)
	if err != nil {
		panic(errorsmod.Wrap(err, "failed to get all operator slash events"))
	}
	genesis.AllStakerClaimedRewards, err = k.GetAllStakerClaimedRewards(ctx)
	if err != nil {
		panic(errorsmod.Wrap(err, "failed to get all staker claimed rewards"))
	}
	return &genesis
}
