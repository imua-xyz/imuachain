package app

import (
	"encoding/json"

	"cosmossdk.io/simapp"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/codec"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govtypesv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	evmtypes "github.com/evmos/evmos/v16/x/evm/types"
	"github.com/imua-xyz/imuachain/utils"

	"github.com/evmos/evmos/v16/encoding"
)

// NewDefaultGenesisState generates the default state for the application.
func NewDefaultGenesisState(cdc codec.Codec) simapp.GenesisState {
	encCfg := encoding.MakeConfig(ModuleBasics)
	defaultGenesis := ModuleBasics.DefaultGenesis(encCfg.Codec)

	// crisis module
	crisisGenesis := crisistypes.GenesisState{}
	rawGenesis := defaultGenesis[crisistypes.ModuleName]
	cdc.MustUnmarshalJSON(rawGenesis, &crisisGenesis)
	crisisGenesis.ConstantFee.Denom = utils.BaseDenom
	defaultGenesis[crisistypes.ModuleName] = cdc.MustMarshalJSON(&crisisGenesis)

	// gov module
	govGenesis := govtypesv1.GenesisState{}
	rawGenesis = defaultGenesis[govtypes.ModuleName]
	cdc.MustUnmarshalJSON(rawGenesis, &govGenesis)
	govGenesis.Params.MinDeposit[0].Denom = utils.BaseDenom
	defaultGenesis[govtypes.ModuleName] = cdc.MustMarshalJSON(&govGenesis)

	// evm module
	evmGenesis := evmtypes.GenesisState{}
	rawGenesis = defaultGenesis[evmtypes.ModuleName]
	cdc.MustUnmarshalJSON(rawGenesis, &evmGenesis)
	evmGenesis.Params.EvmDenom = utils.BaseDenom
	defaultGenesis[evmtypes.ModuleName] = cdc.MustMarshalJSON(&evmGenesis)

	return defaultGenesis
}

// ExportAppStateAndValidators exports the state of the application for a genesis
// file.
func (app *ImuachainApp) ExportAppStateAndValidators(
	forZeroHeight bool, jailAllowedAddrs []string, modulesToExport []string,
) (servertypes.ExportedApp, error) {
	// Creates context with current height and checks txs for ctx to be usable by start of next
	// block
	ctx := app.NewContext(true, tmproto.Header{Height: app.LastBlockHeight()}).
		WithChainID(app.ChainID())

	// We export at last height + 1, because that's the height at which
	// Tendermint will start InitChain.
	height := app.LastBlockHeight() + 1
	if forZeroHeight {
		height = 0

		if err := app.prepForZeroHeightGenesis(ctx, jailAllowedAddrs); err != nil {
			return servertypes.ExportedApp{}, err
		}
	}

	genState := app.mm.ExportGenesisForModules(ctx, app.appCodec, modulesToExport)
	appState, err := json.MarshalIndent(genState, "", "  ")
	if err != nil {
		return servertypes.ExportedApp{}, err
	}

	// the x/dogfood validator set is exported in its `val_set` key, and hence,
	// does not need to be part of the app export. in other words, we do not
	// duplicate the exported validator set. besides, as far as i can tell, the
	// SDK does not use the Validators member of the ExportedApp struct.

	return servertypes.ExportedApp{
		AppState:        appState,
		Height:          height,
		ConsensusParams: app.BaseApp.GetConsensusParams(ctx),
	}, nil
}

// prepare for fresh start at zero height
// NOTE zero height genesis is a temporary feature which will be deprecated
//
//	in favor of export at a block height
func (app *ImuachainApp) prepForZeroHeightGenesis(
	ctx sdk.Context,
	_ []string,
) error {
	// TODO: use the []string to mark validators as jailed.

	/* Just to be safe, assert the invariants on current state. */
	app.CrisisKeeper.AssertInvariants(ctx)

	/* Handle fee distribution state. */

	// TODO(mm): replace with new reward distribution keeper.

	// withdraw all delegator rewards

	// clear validator slash events

	// clear validator historical rewards

	// set context height to zero
	height := ctx.BlockHeight()
	ctx = ctx.WithBlockHeight(0)

	// reinitialize all validators

	// reinitialize all delegations

	// reset context height
	ctx = ctx.WithBlockHeight(height)

	/* Handle staking state. */

	// not supported: iterate through redelegations, reset creation height

	// iterate through unbonding delegations, reset creation height

	// Iterate through validators by power descending, reset bond heights, and
	// update bond intra-tx counters.

	if _, err := app.StakingKeeper.ApplyAndReturnValidatorSetUpdates(ctx); err != nil {
		return err
	}

	/* Handle slashing state. */

	// reset start height on signing infos
	app.SlashingKeeper.IterateValidatorSigningInfos(
		ctx,
		func(addr sdk.ConsAddress, info slashingtypes.ValidatorSigningInfo) (stop bool) {
			info.StartHeight = 0
			app.SlashingKeeper.SetValidatorSigningInfo(ctx, addr, info)
			return false
		},
	)
	return nil
}
