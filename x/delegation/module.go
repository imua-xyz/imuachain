package delegation

import (
	"context"
	"encoding/json"
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/imua-xyz/imuachain/x/delegation/client/cli"
	"github.com/imua-xyz/imuachain/x/delegation/keeper"
	delegationtype "github.com/imua-xyz/imuachain/x/delegation/types"
	"github.com/spf13/cobra"
)

// consensusVersion defines the current x/delegation module consensus version.
const consensusVersion = 1

// type check to ensure the interface is properly implemented
var (
	_ module.AppModule           = AppModule{}
	_ module.AppModuleBasic      = AppModuleBasic{}
	_ module.AppModuleSimulation = AppModule{}
)

type AppModuleBasic struct{}

func (b AppModuleBasic) Name() string {
	return delegationtype.ModuleName
}

func (b AppModuleBasic) RegisterLegacyAminoCodec(amino *codec.LegacyAmino) {
	delegationtype.RegisterLegacyAminoCodec(amino)
}

func (b AppModuleBasic) RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	delegationtype.RegisterInterfaces(registry)
}

func (b AppModuleBasic) RegisterGRPCGatewayRoutes(
	c client.Context,
	serveMux *runtime.ServeMux,
) {
	if err := delegationtype.RegisterQueryHandlerClient(context.Background(), serveMux, delegationtype.NewQueryClient(c)); err != nil {
		panic(err)
	}
}

func (b AppModuleBasic) GetTxCmd() *cobra.Command {
	return cli.NewTxCmd()
}

func (b AppModuleBasic) GetQueryCmd() *cobra.Command {
	return cli.GetQueryCmd()
}

type AppModule struct {
	AppModuleBasic
	keeper keeper.Keeper
}

func NewAppModule(_ codec.Codec, keeper keeper.Keeper) AppModule {
	return AppModule{
		AppModuleBasic: AppModuleBasic{},
		keeper:         keeper,
	}
}

// IsOnePerModuleType implements the depinject.OnePerModuleType interface.
func (am AppModule) IsOnePerModuleType() {}

// IsAppModule implements the appmodule.AppModule interface.
func (am AppModule) IsAppModule() {}

// RegisterServices registers module services.
func (am AppModule) RegisterServices(cfg module.Configurator) {
	delegationtype.RegisterMsgServer(cfg.MsgServer(), &am.keeper)
	delegationtype.RegisterQueryServer(cfg.QueryServer(), &am.keeper)
}

func (am AppModule) GenerateGenesisState(*module.SimulationState) {
}

func (am AppModule) RegisterStoreDecoder(sdk.StoreDecoderRegistry) {
}

func (am AppModule) WeightedOperations(module.SimulationState) []simtypes.WeightedOperation {
	return []simtypes.WeightedOperation{}
}

// EndBlock executes all ABCI EndBlock logic respective to this module.
// It returns no validator updates.
func (am AppModule) EndBlock(ctx sdk.Context, req abci.RequestEndBlock) []abci.ValidatorUpdate {
	am.keeper.EndBlock(ctx, req)
	return []abci.ValidatorUpdate{}
}

// BeginBlock executes all ABCI BeginBlock logic respective to this module.
func (AppModule) BeginBlock(sdk.Context, abci.RequestBeginBlock) {}

// DefaultGenesis returns a default GenesisState for the module, marshaled to json.RawMessage.
// The default GenesisState need to be defined by the module developer and is primarily used for
// testing
func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(delegationtype.DefaultGenesis())
}

// ValidateGenesis used to validate the GenesisState, given in its json.RawMessage form
func (AppModuleBasic) ValidateGenesis(
	cdc codec.JSONCodec,
	_ client.TxEncodingConfig,
	bz json.RawMessage,
) error {
	var genState delegationtype.GenesisState
	if err := cdc.UnmarshalJSON(bz, &genState); err != nil {
		return fmt.Errorf(
			"failed to unmarshal %s genesis state: %w",
			delegationtype.ModuleName,
			err,
		)
	}
	return genState.Validate()
}

// InitGenesis performs the module's genesis initialization. It returns no validator updates.
func (am AppModule) InitGenesis(
	ctx sdk.Context,
	cdc codec.JSONCodec,
	gs json.RawMessage,
) []abci.ValidatorUpdate {
	var genState delegationtype.GenesisState
	// Initialize global index to index in genesis state
	cdc.MustUnmarshalJSON(gs, &genState)

	return am.keeper.InitGenesis(ctx, genState)
}

// ExportGenesis returns the module's exported genesis state as raw JSON bytes.
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	genState := am.keeper.ExportGenesis(ctx)
	return cdc.MustMarshalJSON(genState)
}

// ConsensusVersion implements AppModule/ConsensusVersion.
func (AppModule) ConsensusVersion() uint64 { return consensusVersion }
