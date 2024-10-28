package subscriber

import (
	"encoding/json"
	"fmt"

	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	tmtypes "github.com/cometbft/cometbft/types"
)

// ExportAppStateAndValidators implements the simapp app interface
// by exporting the state of the application
func (app *App) ExportAppStateAndValidators(
	forZeroHeight bool, jailAllowedAddrs, modulesToExport []string,
) (servertypes.ExportedApp, error) {
	// as if they could withdraw from the start of the next block
	ctx := app.NewContext(true, tmproto.Header{Height: app.LastBlockHeight()})

	// We export at last height + 1, because that's the height at which
	// Tendermint will start InitChain.
	height := app.LastBlockHeight() + 1
	if forZeroHeight {
		height = 0
		app.prepForZeroHeightGenesis(ctx, jailAllowedAddrs)
	}

	genState := app.ModuleManager().ExportGenesisForModules(ctx, app.appCodec, modulesToExport)
	appState, err := json.MarshalIndent(genState, "", "  ")
	if err != nil {
		return servertypes.ExportedApp{}, err
	}

	// any validators supplied to TM at in the validators object will be overwritten
	// by InitGenesis, so this is not super relevant. however, we use it to align
	// with cosmos-sdk export implementation.
	validators, err := app.GetValidatorSet(ctx)
	if err != nil {
		return servertypes.ExportedApp{}, err
	}

	return servertypes.ExportedApp{
		AppState:        appState,
		Height:          height,
		Validators:      validators,
		ConsensusParams: app.BaseApp.GetConsensusParams(ctx),
	}, nil
}

// prepare for fresh start at zero height
// NOTE zero height genesis is a temporary feature which will be deprecated
//
//	in favor of export at a block height
func (app *App) prepForZeroHeightGenesis(ctx sdk.Context, _ []string) {
	/* Just to be safe, assert the invariants on current state. */
	app.CrisisKeeper.AssertInvariants(ctx)

	// set context height to zero
	height := ctx.BlockHeight()
	ctx = ctx.WithBlockHeight(0)

	// reset context height
	ctx = ctx.WithBlockHeight(height)

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
}

// GetValidatorSet returns a slice of bonded validators.
func (app *App) GetValidatorSet(ctx sdk.Context) ([]tmtypes.GenesisValidator, error) {
	cVals := app.SubscriberKeeper.GetAllSubscriberValidators(ctx)
	if len(cVals) == 0 {
		return nil, fmt.Errorf("empty validator set")
	}

	vals := []tmtypes.GenesisValidator{}
	for _, v := range cVals {
		vals = append(vals, tmtypes.GenesisValidator{Address: v.ConsAddress, Power: v.Power})
	}
	return vals, nil
}
