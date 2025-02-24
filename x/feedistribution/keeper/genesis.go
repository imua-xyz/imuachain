package keeper

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/imua-xyz/imuachain/x/feedistribution/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func (k Keeper) InitGenesis(ctx sdk.Context, genState types.GenesisState) {
	k.SetParams(ctx, genState.Params)
	epochID := genState.Params.EpochIdentifier
	_, found := k.epochsKeeper.GetEpochInfo(ctx, epochID)
	if !found {
		// the panic is suitable here because it is being done at genesis, when the node
		// is not running. it means that the genesis file is malformed.
		panic(fmt.Sprintf("epoch info not found %s", epochID))
	}
	// Set fee pool
	k.SetFeePool(ctx, &genState.FeePool)

	// Set all the validatorAccumulatedCommission
	for _, elem := range genState.ValidatorAccumulatedCommissions {
		validatorAddr, _ := sdk.AccAddressFromBech32(elem.ValAddr)
		k.SetValidatorAccumulatedCommission(ctx, sdk.ValAddress(validatorAddr), *elem.Commission)
	}

	// Set all the validatorCurrentRewards
	for _, elem := range genState.ValidatorCurrentRewardsList {
		validatorAddr, _ := sdk.AccAddressFromBech32(elem.ValAddr)
		k.SetValidatorCurrentRewards(ctx, sdk.ValAddress(validatorAddr), *elem.CurrentRewards)
	}

	// Set all the validatorOutstandingRewards
	for _, elem := range genState.ValidatorOutstandingRewardsList {
		validatorAddr, _ := sdk.AccAddressFromBech32(elem.ValAddr)
		k.SetValidatorOutstandingRewards(ctx, sdk.ValAddress(validatorAddr), *elem.OutstandingRewards)
	}

	// Set all the stakerRewards
	for _, elem := range genState.StakerOutstandingRewardsList {
		k.SetStakerRewards(ctx, elem.StakerAddr, *elem.StakerOutstandingRewards)
	}
}

// ExportGenesis returns the module's exported genesis
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	genesis := types.GenesisState{}
	genesis.Params = k.GetParams(ctx)
	feePool := k.GetFeePool(ctx)
	if feePool == nil {
		panic("fee pool cannot be nil in genesis export")
	}
	genesis.FeePool = *feePool
	validatorData, err := k.GetAllValidatorData(ctx)
	if err != nil {
		panic(errorsmod.Wrap(err, "Error getting validator data").Error())
	}

	if accumulatedCommissions, ok := validatorData["ValidatorAccumulatedCommissions"].([]types.ValidatorAccumulatedCommissions); ok {
		genesis.ValidatorAccumulatedCommissions = accumulatedCommissions
	} else {
		panic("Failed to assert ValidatorAccumulatedCommissions type")
	}

	if currentRewardsList, ok := validatorData["ValidatorCurrentRewardsList"].([]types.ValidatorCurrentRewardsList); ok {
		genesis.ValidatorCurrentRewardsList = currentRewardsList
	} else {
		panic("Failed to assert ValidatorCurrentRewardsList type")
	}

	if outstandingRewardsList, ok := validatorData["ValidatorOutstandingRewardsList"].([]types.ValidatorOutstandingRewardsList); ok {
		genesis.ValidatorOutstandingRewardsList = outstandingRewardsList
	} else {
		panic("Failed to assert ValidatorOutstandingRewardsList type")
	}

	if stakerRewardsList, ok := validatorData["StakerOutstandingRewardsList"].([]types.StakerOutstandingRewardsList); ok {
		genesis.StakerOutstandingRewardsList = stakerRewardsList
	} else {
		panic("Failed to assert StakerOutstandingRewardsList type")
	}
	return &genesis
}
