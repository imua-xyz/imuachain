package oracle

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"sort"
	"strings"

	// this line is used by starport scaffolding # 1

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"

	"github.com/ExocoreNetwork/exocore/x/oracle/client/cli"
	"github.com/ExocoreNetwork/exocore/x/oracle/keeper"
	"github.com/ExocoreNetwork/exocore/x/oracle/keeper/cache"
	"github.com/ExocoreNetwork/exocore/x/oracle/types"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

var (
	_ module.AppModule      = AppModule{}
	_ module.AppModuleBasic = AppModuleBasic{}
)

// ----------------------------------------------------------------------------
// AppModuleBasic
// ----------------------------------------------------------------------------

// AppModuleBasic implements the AppModuleBasic interface that defines the independent methods a Cosmos SDK module needs to implement.
type AppModuleBasic struct {
	cdc codec.BinaryCodec
}

func NewAppModuleBasic(cdc codec.BinaryCodec) AppModuleBasic {
	return AppModuleBasic{cdc: cdc}
}

// Name returns the name of the module as a string
func (AppModuleBasic) Name() string {
	return types.ModuleName
}

// RegisterLegacyAminoCodec registers the amino codec for the module, which is used to marshal and unmarshal structs to/from []byte in order to persist them in the module's KVStore
func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	types.RegisterCodec(cdc)
}

// RegisterInterfaces registers a module's interface types and their concrete implementations as proto.Message
func (a AppModuleBasic) RegisterInterfaces(reg cdctypes.InterfaceRegistry) {
	types.RegisterInterfaces(reg)
}

// DefaultGenesis returns a default GenesisState for the module, marshaled to json.RawMessage. The default GenesisState need to be defined by the module developer and is primarily used for testing
func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(types.DefaultGenesis())
}

// ValidateGenesis used to validate the GenesisState, given in its json.RawMessage form
func (AppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, _ client.TxEncodingConfig, bz json.RawMessage) error {
	var genState types.GenesisState
	if err := cdc.UnmarshalJSON(bz, &genState); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", types.ModuleName, err)
	}
	return genState.Validate()
}

// RegisterGRPCGatewayRoutes registers the gRPC Gateway routes for the module
func (AppModuleBasic) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *runtime.ServeMux) {
	if err := types.RegisterQueryHandlerClient(context.Background(), mux, types.NewQueryClient(clientCtx)); err != nil {
		panic(err)
	}
}

// GetTxCmd returns the root Tx command for the module. The subcommands of this root command are used by end-users to generate new transactions containing messages defined in the module
func (a AppModuleBasic) GetTxCmd() *cobra.Command {
	return cli.GetTxCmd()
}

// GetQueryCmd returns the root query command for the module. The subcommands of this root command are used by end-users to generate new queries to the subset of the state defined by the module
func (AppModuleBasic) GetQueryCmd() *cobra.Command {
	return cli.GetQueryCmd(types.StoreKey)
}

// ----------------------------------------------------------------------------
// AppModule
// ----------------------------------------------------------------------------

// AppModule implements the AppModule interface that defines the inter-dependent methods that modules need to implement
type AppModule struct {
	AppModuleBasic

	keeper keeper.Keeper

	// used for simulation
	accountKeeper types.AccountKeeper

	// used for simulation
	bankKeeper types.BankKeeper
}

func NewAppModule(
	cdc codec.Codec,
	keeper keeper.Keeper,
	accountKeeper types.AccountKeeper,
	bankKeeper types.BankKeeper,
) AppModule {
	return AppModule{
		AppModuleBasic: NewAppModuleBasic(cdc),
		keeper:         keeper,
		accountKeeper:  accountKeeper,
		bankKeeper:     bankKeeper,
	}
}

// RegisterServices registers a gRPC query service to respond to the module-specific gRPC queries
func (am AppModule) RegisterServices(cfg module.Configurator) {
	types.RegisterMsgServer(cfg.MsgServer(), keeper.NewMsgServerImpl(am.keeper))
	types.RegisterQueryServer(cfg.QueryServer(), am.keeper)
}

// RegisterInvariants registers the invariants of the module. If an invariant deviates from its predicted value, the InvariantRegistry triggers appropriate logic (most often the chain will be halted)
func (am AppModule) RegisterInvariants(_ sdk.InvariantRegistry) {}

// InitGenesis performs the module's genesis initialization. It returns no validator updates.
func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, gs json.RawMessage) []abci.ValidatorUpdate {
	var genState types.GenesisState
	// Initialize global index to index in genesis state
	cdc.MustUnmarshalJSON(gs, &genState)

	InitGenesis(ctx, am.keeper, genState)

	return []abci.ValidatorUpdate{}
}

// ExportGenesis returns the module's exported genesis state as raw JSON bytes.
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	genState := ExportGenesis(ctx, am.keeper)
	return cdc.MustMarshalJSON(genState)
}

// ConsensusVersion is a sequence number for state-breaking change of the module. It should be incremented on each consensus-breaking change introduced by the module. To avoid wrong/empty versions, the initial version should be set to 1
func (AppModule) ConsensusVersion() uint64 { return 1 }

// BeginBlock contains the logic that is automatically triggered at the beginning of each block
func (am AppModule) BeginBlock(ctx sdk.Context, _ abci.RequestBeginBlock) {
	// init caches and aggregatorContext for node restart
	// TODO: try better way to init caches and aggregatorContext than beginBlock
	_ = am.keeper.GetCaches()
	agc := am.keeper.GetAggregatorContext(ctx)
	validatorPowers := agc.GetValidatorPowers()
	// set validatorReportInfo to track performance
	for validator := range validatorPowers {
		am.keeper.InitValidatorReportInfo(ctx, validator, ctx.BlockHeight())
	}
}

// EndBlock contains the logic that is automatically triggered at the end of each block
func (am AppModule) EndBlock(ctx sdk.Context, _ abci.RequestEndBlock) []abci.ValidatorUpdate {
	cs := am.keeper.GetCaches()
	validatorUpdates := am.keeper.GetValidatorUpdates(ctx)
	forceSeal := false
	agc := am.keeper.GetAggregatorContext(ctx)

	logger := am.keeper.Logger(ctx)
	height := ctx.BlockHeight()
	if len(validatorUpdates) > 0 {
		validatorList := make(map[string]*big.Int)
		for _, vu := range validatorUpdates {
			pubKey, _ := cryptocodec.FromTmProtoPublicKey(vu.PubKey)
			validatorStr := sdk.ConsAddress(pubKey.Address()).String()
			validatorList[validatorStr] = big.NewInt(vu.Power)
			// add possible new added validator info for slashing tracking
			if vu.Power > 0 {
				am.keeper.InitValidatorReportInfo(ctx, validatorStr, height)
			}
		}
		// update validator set information in cache
		cs.AddCache(cache.ItemV(validatorList))
		validatorPowers := make(map[string]*big.Int)
		cs.GetCache(cache.ItemV(validatorPowers))
		// update validatorPowerList in aggregatorContext
		agc.SetValidatorPowers(validatorPowers)
		// TODO: seal all alive round since validatorSet changed here
		forceSeal = true
		logger.Info("validator set changed, force seal all active rounds", "height", height)
	}

	// TODO: for v1 use mode==1, just check the failed feeders
	_, failed, _, windowClosed := agc.SealRound(ctx, forceSeal)
	defer func() {
		for _, feederID := range windowClosed {
			agc.RemoveWorker(feederID)
			am.keeper.RemoveNonceWithFeederIDForValidators(ctx, feederID, agc.GetValidators())
		}
	}()
	// update&check slashing info
	validatorPowers := agc.GetValidatorPowers()
	validators := make([]string, 0, len(validatorPowers))
	for validator := range validatorPowers {
		validators = append(validators, validator)
	}
	sort.Strings(validators)
	for _, validator := range validators {
		power := validatorPowers[validator]
		reportedInfo, found := am.keeper.GetValidatorReportInfo(ctx, validator)
		if !found {
			logger.Error(fmt.Sprintf("Expected report info for validator %s but not found", validator))
			continue
		}
		// TODO: for the round calculation, now only sourceID=1 is used so {feederID, sourceID} have only one value for each feederID which corresponding to one round.
		// But when we came to multiple sources, we should consider the round corresponding to feedeerID instead of {feederID, sourceID}
		for _, finalPrice := range agc.GetFinalPriceListForFeederIDs(windowClosed) {
			exist, matched := agc.PerformanceReview(ctx, finalPrice, validator)
			if exist && !matched {
				// TODO: malicious price, just slash&jail immediately
				logger.Info(
					"confirmed malicious price",
					"validator", validator,
					"infraction_height", height,
					"infraction_time", ctx.BlockTime(),
					"feederID", finalPrice.FeederID,
					"detID", finalPrice.DetID,
					"sourceID", finalPrice.SourceID,
					"finalPrice", finalPrice.Price,
				)
				consAddr, err := sdk.ConsAddressFromBech32(validator)
				if err != nil {
					panic("invalid consAddr string")
				}

				operator := am.keeper.ValidatorByConsAddr(ctx, consAddr)
				if operator != nil && !operator.IsJailed() {
					coinsBurned := am.keeper.SlashWithInfractionReason(ctx, consAddr, height, power.Int64(), am.keeper.GetSlashFractionMalicious(ctx), stakingtypes.Infraction_INFRACTION_UNSPECIFIED)
					ctx.EventManager().EmitEvent(
						sdk.NewEvent(
							types.EventTypeOracleSlash,
							sdk.NewAttribute(types.AttributeKeyValidatorKey, validator),
							sdk.NewAttribute(types.AttributeKeyPower, fmt.Sprintf("%d", power)),
							sdk.NewAttribute(types.AttributeKeyReason, types.AttributeValueMaliciousReportPrice),
							sdk.NewAttribute(types.AttributeKeyJailed, validator),
							sdk.NewAttribute(types.AttributeKeyBurnedCoins, coinsBurned.String()),
						),
					)
					am.keeper.Jail(ctx, consAddr)
					jailUntil := ctx.BlockHeader().Time.Add(am.keeper.GetMaliciousJailDuration(ctx))
					am.keeper.JailUntil(ctx, consAddr, jailUntil)
					reportedInfo.MissedRoundsCounter = 0
					reportedInfo.IndexOffset = 0
					am.keeper.ClearValidatorMissedRoundBitArray(ctx, validator)
				}
				continue
			}

			index := uint64(reportedInfo.IndexOffset % am.keeper.GetReportedRoundsWindow(ctx))
			reportedInfo.IndexOffset++
			// Update reported round bit array & counter
			// This counter just tracks the sum of the bit array
			// That way we avoid needing to read/write the whole array each time
			previous := am.keeper.GetValidatorMissedRoundBitArray(ctx, validator, index)
			missed := !exist
			switch {
			case !previous && missed:
				// Array value has changed from not missed to missed, increment counter
				am.keeper.SetValidatorMissedRoundBitArray(ctx, validator, index, true)
				reportedInfo.MissedRoundsCounter++
			case previous && !missed:
				// Array value has changed from missed to not missed, decrement counter
				am.keeper.SetValidatorMissedRoundBitArray(ctx, validator, index, false)
				reportedInfo.MissedRoundsCounter--
			default:
				// Array value at this index has not changed, no need to update counter
			}

			minReportedPerWindow := am.keeper.GetMinReportedPerWindow(ctx)

			if missed {
				ctx.EventManager().EmitEvent(
					sdk.NewEvent(
						types.EventTypeOracleLiveness,
						sdk.NewAttribute(types.AttributeKeyValidatorKey, validator),
						sdk.NewAttribute(types.AttributeKeyMissedRounds, fmt.Sprintf("%d", reportedInfo.MissedRoundsCounter)),
						sdk.NewAttribute(types.AttributeKeyHeight, fmt.Sprintf("%d", height)),
					),
				)

				logger.Debug(
					"absent validator",
					"height", ctx.BlockHeight(),
					"validator", validator,
					"missed", reportedInfo.MissedRoundsCounter,
					"threshold", minReportedPerWindow,
				)
			}

			minHeight := reportedInfo.StartHeight + am.keeper.GetReportedRoundsWindow(ctx)
			maxMissed := am.keeper.GetReportedRoundsWindow(ctx) - minReportedPerWindow
			// if we are past the minimum height and the validator has missed too many rounds reporting prices, punish them
			if height > minHeight && reportedInfo.MissedRoundsCounter > maxMissed {
				consAddr, err := sdk.ConsAddressFromBech32(validator)
				if err != nil {
					panic("invalid consAddr string")
				}
				operator := am.keeper.ValidatorByConsAddr(ctx, consAddr)
				if operator != nil && !operator.IsJailed() {
					// missing rounds confirmed: slash and jail the validator
					coinsBurned := am.keeper.SlashWithInfractionReason(ctx, consAddr, height, power.Int64(), am.keeper.GetSlashFractionMiss(ctx), stakingtypes.Infraction_INFRACTION_UNSPECIFIED)
					ctx.EventManager().EmitEvent(
						sdk.NewEvent(
							types.EventTypeOracleSlash,
							sdk.NewAttribute(types.AttributeKeyValidatorKey, validator),
							sdk.NewAttribute(types.AttributeKeyPower, fmt.Sprintf("%d", power)),
							sdk.NewAttribute(types.AttributeKeyReason, types.AttributeValueMissingReportPrice),
							sdk.NewAttribute(types.AttributeKeyJailed, validator),
							sdk.NewAttribute(types.AttributeKeyBurnedCoins, coinsBurned.String()),
						),
					)
					am.keeper.Jail(ctx, consAddr)
					jailUntil := ctx.BlockHeader().Time.Add(am.keeper.GetMissJailDuration(ctx))
					am.keeper.JailUntil(ctx, consAddr, jailUntil)

					// We need to reset the counter & array so that the validator won't be immediately slashed for miss report info upon rebonding.
					reportedInfo.MissedRoundsCounter = 0
					reportedInfo.IndexOffset = 0
					am.keeper.ClearValidatorMissedRoundBitArray(ctx, validator)

					logger.Info(
						"slashing and jailing validator due to liveness fault",
						"height", height,
						"validator", consAddr.String(),
						"min_height", minHeight,
						"threshold", minReportedPerWindow,
						"slashed", am.keeper.GetSlashFractionMiss(ctx).String(),
						"jailed_until", jailUntil,
					)
				} else {
					// validator was (a) not found or (b) already jailed so we do not slash
					logger.Info(
						"validator would have been slashed for too many missed repoerting price, but was either not found in store or already jailed",
						"validator", validator,
					)
				}
			}
			// Set the updated reportInfo
			am.keeper.SetValidatorReportInfo(ctx, validator, reportedInfo)
		}
	}

	// append new round with previous price for fail-sealed token
	for _, tokenID := range failed {
		prevPrice, nextRoundID := am.keeper.GrowRoundID(ctx, tokenID)
		logger.Info("add new round with previous price under fail aggregation", "tokenID", tokenID, "roundID", nextRoundID, "price", prevPrice)
	}

	am.keeper.ResetAggregatorContextCheckTx()

	if _, _, paramsUpdated := cs.CommitCache(ctx, false, am.keeper); paramsUpdated {
		var p cache.ItemP
		cs.GetCache(&p)
		params := types.Params(p)
		agc.SetParams(&params)
		ctx.EventManager().EmitEvent(sdk.NewEvent(
			types.EventTypeCreatePrice,
			sdk.NewAttribute(types.AttributeKeyParamsUpdated, types.AttributeValueParamsUpdatedSuccess),
		))
	}

	if feederIDs := am.keeper.GetUpdatedFeederIDs(); len(feederIDs) > 0 {
		feederIDsStr := strings.Join(feederIDs, "_")
		ctx.EventManager().EmitEvent(sdk.NewEvent(
			types.EventTypeCreatePrice,
			sdk.NewAttribute(types.AttributeKeyPriceUpdated, types.AttributeValuePriceUpdatedSuccess),
			sdk.NewAttribute(types.AttributeKeyFeederIDs, feederIDsStr),
		))
		am.keeper.ResetUpdatedFeederIDs()
	}

	newRoundFeederIDs := agc.PrepareRoundEndBlock(uint64(ctx.BlockHeight()))
	for _, feederID := range newRoundFeederIDs {
		am.keeper.AddZeroNonceItemWithFeederIDForValidators(ctx, feederID, agc.GetValidators())
	}
	return []abci.ValidatorUpdate{}
}
