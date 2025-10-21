package keeper

import (
	errorsmod "cosmossdk.io/errors"
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	keytypes "github.com/imua-xyz/imuachain/types/keys"
	"github.com/imua-xyz/imuachain/x/operator/types"
)

func (k Keeper) InitGenesis(ctx sdk.Context, state types.GenesisState) []abci.ValidatorUpdate {
	// set the params
	if err := state.Params.Validate(); err != nil {
		panic(errorsmod.Wrap(err, "invalid params"))
	}
	k.SetParams(ctx, state.Params)
	// set the operators
	for i := range state.Operators {
		op := state.Operators[i] // avoid implicit memory aliasing
		if op.OperatorInfo.EarningsAddr == "" {
			op.OperatorInfo.EarningsAddr = op.OperatorAddress
		}
		// remember that Cosmos will 100% set the block time regardless of
		// genesis or chain restart or whatever
		if op.OperatorInfo.Commission.UpdateTime.After(ctx.BlockTime()) {
			panic(errorsmod.Wrap(stakingtypes.ErrCommissionUpdateTime, "commission update time is in the future"))
		}
		if err := k.RegisterOperator(ctx, op.OperatorAddress, &op.OperatorInfo); err != nil {
			panic(errorsmod.Wrap(err, "failed to set operator info"))
		}
	}
	// set consensus keys (operator records)
	for _, record := range state.OperatorRecords {
		addr := record.OperatorAddress
		// #nosec G703 // already validated
		operatorAddr, _ := sdk.AccAddressFromBech32(addr)
		for _, detail := range record.Chains {
			wrappedKey := keytypes.NewWrappedConsKeyFromHex(detail.ConsensusKey)
			k.setOperatorConsKeyForChainIDUnchecked(
				ctx, operatorAddr, detail.ChainID, wrappedKey,
			)
		}
	}
	// init the state from the general exporting genesis file
	err := k.SetAllOptedInfo(ctx, state.OptStates)
	if err != nil {
		panic(errorsmod.Wrap(err, "failed to set all opted info"))
	}
	err = k.SetAllOperatorUSDValues(ctx, state.OperatorUSDValues)
	if err != nil {
		panic(errorsmod.Wrap(err, "failed to set all operator USD values"))
	}
	err = k.SetAllAVSUSDValues(ctx, state.AVSUSDValues)
	if err != nil {
		panic(errorsmod.Wrap(err, "failed to set all AVS USD values"))
	}
	err = k.SetAllSlashStates(ctx, state.SlashStates)
	if err != nil {
		panic(errorsmod.Wrap(err, "failed to set all slash info"))
	}
	err = k.SetAllPrevConsKeys(ctx, state.PreConsKeys)
	if err != nil {
		panic(errorsmod.Wrap(err, "failed to set all previous consensus keys"))
	}
	err = k.SetAllOperatorKeyRemovals(ctx, state.OperatorKeyRemovals)
	if err != nil {
		panic(errorsmod.Wrap(err, "failed to set all key removals for operators"))
	}
	err = k.SetAllOperatorAssetUSDValues(ctx, state.OperatorAssetUsdValues)
	if err != nil {
		panic(errorsmod.Wrap(err, "failed to set all operator asset USD values"))
	}
	return []abci.ValidatorUpdate{}
}

func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	res := types.GenesisState{}
	// get the params
	res.Params = k.GetParams(ctx)
	// get the operators
	res.Operators = k.AllOperators(ctx)
	// get the operator records
	var err error

	res.OperatorRecords, err = k.GetAllOperatorConsKeyRecords(ctx)
	if err != nil {
		panic(errorsmod.Wrap(err, "failed to get all consensus keys for operators").Error())
	}

	res.OptStates, err = k.GetAllOptedInfo(ctx)
	if err != nil {
		panic(errorsmod.Wrap(err, "failed to get all opted info").Error())
	}

	res.AVSUSDValues, err = k.GetAllAVSUSDValues(ctx)
	if err != nil {
		panic(errorsmod.Wrap(err, "failed to get all AVS USD values").Error())
	}

	res.OperatorUSDValues, err = k.GetAllOperatorUSDValues(ctx)
	if err != nil {
		panic(errorsmod.Wrap(err, "failed to get all operator USD values").Error())
	}

	res.SlashStates, err = k.GetAllSlashStates(ctx)
	if err != nil {
		panic(errorsmod.Wrap(err, "failed to set all slashing info").Error())
	}

	res.PreConsKeys, err = k.GetAllPrevConsKeys(ctx)
	if err != nil {
		panic(errorsmod.Wrap(err, "failed to get all previous consensus keys").Error())
	}

	res.OperatorKeyRemovals, err = k.GetAllOperatorKeyRemovals(ctx)
	if err != nil {
		panic(errorsmod.Wrap(err, "failed to get all key removals for operators").Error())
	}

	res.OperatorAssetUsdValues, err = k.GetAllOperatorAssetUSDValues(ctx)
	if err != nil {
		panic(errorsmod.Wrap(err, "failed to get all operator asset USD values").Error())
	}

	return &res
}
