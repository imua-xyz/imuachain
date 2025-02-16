package operator

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ExocoreNetwork/exocore/x/operator/client/cli"
	"github.com/ExocoreNetwork/exocore/x/operator/keeper"
	operatortypes "github.com/ExocoreNetwork/exocore/x/operator/types"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"
)

// consensusVersion defines the current x/operator module consensus version.
const consensusVersion = 1

// type check to ensure the interface is properly implemented
var (
	_ module.AppModule           = AppModule{}
	_ module.AppModuleBasic      = AppModuleBasic{}
	_ module.AppModuleSimulation = AppModule{}
)

type AppModuleBasic struct{}

func (b AppModuleBasic) Name() string {
	return operatortypes.ModuleName
}

func (b AppModuleBasic) RegisterLegacyAminoCodec(_ *codec.LegacyAmino) {}

func (b AppModuleBasic) RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	operatortypes.RegisterInterfaces(registry)
}

func (b AppModuleBasic) RegisterGRPCGatewayRoutes(
	c client.Context,
	serveMux *runtime.ServeMux,
) {
	if err := operatortypes.RegisterQueryHandlerClient(
		context.Background(), serveMux, operatortypes.NewQueryClient(c),
	); err != nil {
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
	operatortypes.RegisterMsgServer(cfg.MsgServer(), keeper.NewMsgServerImpl(am.keeper))
	operatortypes.RegisterQueryServer(cfg.QueryServer(), &am.keeper)
}

func (am AppModule) GenerateGenesisState(_ *module.SimulationState) {
}

func (am AppModule) RegisterStoreDecoder(_ sdk.StoreDecoderRegistry) {
}

func (am AppModule) WeightedOperations(_ module.SimulationState) []simtypes.WeightedOperation {
	return []simtypes.WeightedOperation{}
}

// EndBlock executes all ABCI EndBlock logic respective to the claim module. It
// returns no validator updates.
func (am AppModule) EndBlock(ctx sdk.Context, req abci.RequestEndBlock) []abci.ValidatorUpdate {
	am.keeper.EndBlock(ctx, req)
	return []abci.ValidatorUpdate{}
}

// DefaultGenesis returns a default GenesisState for the module, marshaled to json.RawMessage.
// The default GenesisState need to be defined by the module developer and is primarily used for
// testing
func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(operatortypes.DefaultGenesis())
}

// ValidateGenesis used to validate the GenesisState, given in its json.RawMessage form
func (AppModuleBasic) ValidateGenesis(
	cdc codec.JSONCodec,
	_ client.TxEncodingConfig,
	bz json.RawMessage,
) error {
	var genState operatortypes.GenesisState
	if err := cdc.UnmarshalJSON(bz, &genState); err != nil {
		return fmt.Errorf(
			"failed to unmarshal %s genesis state: %w",
			operatortypes.ModuleName,
			err,
		)
	}
	return genState.Validate()
}

// InitGenesis performs the module's genesis initialization.
func (am AppModule) InitGenesis(
	ctx sdk.Context,
	cdc codec.JSONCodec,
	gs json.RawMessage,
) []abci.ValidatorUpdate {
	var genState operatortypes.GenesisState
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
