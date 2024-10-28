package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"

	distr "github.com/ExocoreNetwork/exocore/x/feedistribution"
	distrkeeper "github.com/ExocoreNetwork/exocore/x/feedistribution/keeper"
	distrtypes "github.com/ExocoreNetwork/exocore/x/feedistribution/types"
	"github.com/ExocoreNetwork/exocore/x/oracle"

	oracleKeeper "github.com/ExocoreNetwork/exocore/x/oracle/keeper"
	oracleTypes "github.com/ExocoreNetwork/exocore/x/oracle/types"

	"github.com/ExocoreNetwork/exocore/x/avs"
	"github.com/ExocoreNetwork/exocore/x/operator"
	operatorKeeper "github.com/ExocoreNetwork/exocore/x/operator/keeper"

	exoslash "github.com/ExocoreNetwork/exocore/x/slash"

	avsManagerKeeper "github.com/ExocoreNetwork/exocore/x/avs/keeper"
	avsManagerTypes "github.com/ExocoreNetwork/exocore/x/avs/types"
	slashKeeper "github.com/ExocoreNetwork/exocore/x/slash/keeper"
	exoslashTypes "github.com/ExocoreNetwork/exocore/x/slash/types"

	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"
	reflectionv1 "cosmossdk.io/api/cosmos/reflection/v1"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"

	// for EIP-1559 fee handling
	ethante "github.com/ExocoreNetwork/exocore/app/ante/evm"
	// for encoding and decoding of EIP-712 messages
	"github.com/evmos/evmos/v16/ethereum/eip712"

	"github.com/ExocoreNetwork/exocore/x/assets"
	assetsKeeper "github.com/ExocoreNetwork/exocore/x/assets/keeper"
	assetsTypes "github.com/ExocoreNetwork/exocore/x/assets/types"

	"github.com/ExocoreNetwork/exocore/x/delegation"
	delegationKeeper "github.com/ExocoreNetwork/exocore/x/delegation/keeper"
	delegationTypes "github.com/ExocoreNetwork/exocore/x/delegation/types"

	operatorTypes "github.com/ExocoreNetwork/exocore/x/operator/types"

	"github.com/ExocoreNetwork/exocore/x/reward"
	rewardKeeper "github.com/ExocoreNetwork/exocore/x/reward/keeper"
	rewardTypes "github.com/ExocoreNetwork/exocore/x/reward/types"

	// increases or decreases block gas limit based on usage
	"github.com/evmos/evmos/v16/x/feemarket"
	feemarketkeeper "github.com/evmos/evmos/v16/x/feemarket/keeper"
	feemarkettypes "github.com/evmos/evmos/v16/x/feemarket/types"

	runtimeservices "github.com/cosmos/cosmos-sdk/runtime/services"

	"github.com/gorilla/mux"
	"github.com/rakyll/statik/fs"
	"github.com/spf13/cast"

	"github.com/ExocoreNetwork/exocore/app/ante"
	dbm "github.com/cometbft/cometbft-db"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/log"

	"github.com/evmos/evmos/v16/precompiles/common"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/grpc/node"
	"github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/server/api"
	"github.com/cosmos/cosmos-sdk/server/config"

	ica "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts"
	icahost "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/host"
	icahostkeeper "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/host/keeper"
	icahosttypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/host/types"
	icatypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/types"
	ibctestingtypes "github.com/cosmos/ibc-go/v7/testing/types"

	ibctransfer "github.com/cosmos/ibc-go/v7/modules/apps/transfer"
	ibctransfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	ibc "github.com/cosmos/ibc-go/v7/modules/core"
	ibcclient "github.com/cosmos/ibc-go/v7/modules/core/02-client"
	ibcclientclient "github.com/cosmos/ibc-go/v7/modules/core/02-client/client"
	ibcclienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	porttypes "github.com/cosmos/ibc-go/v7/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v7/modules/core/exported"
	ibckeeper "github.com/cosmos/ibc-go/v7/modules/core/keeper"
	ibctm "github.com/cosmos/ibc-go/v7/modules/light-clients/07-tendermint"

	// this module allows the transfer of ERC20 tokens over IBC. for such transfers to occur,
	// they must be enabled in the ERC20 keeper.
	transfer "github.com/evmos/evmos/v16/x/ibc/transfer"
	transferkeeper "github.com/evmos/evmos/v16/x/ibc/transfer/keeper"

	"cosmossdk.io/simapp"
	simappparams "cosmossdk.io/simapp/params"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/cosmos/cosmos-sdk/store/streaming"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/mempool"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/version"
	srvflags "github.com/evmos/evmos/v16/server/flags"

	"github.com/cosmos/cosmos-sdk/x/auth"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	"github.com/cosmos/cosmos-sdk/x/auth/posthandler"
	authsims "github.com/cosmos/cosmos-sdk/x/auth/simulation"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/auth/vesting"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"

	"github.com/cosmos/cosmos-sdk/x/authz"
	authzmodule "github.com/cosmos/cosmos-sdk/x/authz/module"

	"github.com/cosmos/cosmos-sdk/x/bank"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/cosmos/cosmos-sdk/x/capability"
	capabilitykeeper "github.com/cosmos/cosmos-sdk/x/capability/keeper"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"

	"github.com/cosmos/cosmos-sdk/x/consensus"
	consensusparamkeeper "github.com/cosmos/cosmos-sdk/x/consensus/keeper"
	consensusparamtypes "github.com/cosmos/cosmos-sdk/x/consensus/types"

	"github.com/cosmos/cosmos-sdk/x/crisis"
	crisiskeeper "github.com/cosmos/cosmos-sdk/x/crisis/keeper"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"

	"github.com/cosmos/cosmos-sdk/x/evidence"
	evidencekeeper "github.com/cosmos/cosmos-sdk/x/evidence/keeper"
	evidencetypes "github.com/cosmos/cosmos-sdk/x/evidence/types"

	"github.com/cosmos/cosmos-sdk/x/feegrant"
	feegrantkeeper "github.com/cosmos/cosmos-sdk/x/feegrant/keeper"
	feegrantmodule "github.com/cosmos/cosmos-sdk/x/feegrant/module"

	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"

	"github.com/cosmos/cosmos-sdk/x/gov"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"

	"github.com/cosmos/cosmos-sdk/x/params"
	paramsclient "github.com/cosmos/cosmos-sdk/x/params/client"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	paramproposal "github.com/cosmos/cosmos-sdk/x/params/types/proposal"

	"github.com/cosmos/cosmos-sdk/x/slashing"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"

	staking "github.com/ExocoreNetwork/exocore/x/dogfood"
	stakingkeeper "github.com/ExocoreNetwork/exocore/x/dogfood/keeper"
	stakingtypes "github.com/ExocoreNetwork/exocore/x/dogfood/types"

	exomint "github.com/ExocoreNetwork/exocore/x/exomint"
	exomintkeeper "github.com/ExocoreNetwork/exocore/x/exomint/keeper"
	exominttypes "github.com/ExocoreNetwork/exocore/x/exomint/types"

	"github.com/cosmos/cosmos-sdk/x/upgrade"
	upgradeclient "github.com/cosmos/cosmos-sdk/x/upgrade/client"
	upgradekeeper "github.com/cosmos/cosmos-sdk/x/upgrade/keeper"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	"github.com/ExocoreNetwork/exocore/x/evm"
	evmkeeper "github.com/ExocoreNetwork/exocore/x/evm/keeper"
	evmtypes "github.com/evmos/evmos/v16/x/evm/types"

	"github.com/evmos/evmos/v16/encoding"
	evmostypes "github.com/evmos/evmos/v16/types"

	"github.com/ExocoreNetwork/exocore/x/epochs"
	epochskeeper "github.com/ExocoreNetwork/exocore/x/epochs/keeper"
	epochstypes "github.com/ExocoreNetwork/exocore/x/epochs/types"

	"github.com/evmos/evmos/v16/x/erc20"
	erc20keeper "github.com/evmos/evmos/v16/x/erc20/keeper"
	erc20types "github.com/evmos/evmos/v16/x/erc20/types"

	// unnamed import of statik for swagger UI support
	_ "github.com/evmos/evmos/v16/client/docs/statik"

	// Force-load the tracer engines to trigger registration due to Go-Ethereum v1.10.15 changes
	_ "github.com/ethereum/go-ethereum/eth/tracers/js"
	_ "github.com/ethereum/go-ethereum/eth/tracers/native"
)

// Name defines the application binary name
const Name = "exocored"

func init() {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	DefaultNodeHome = filepath.Join(userHomeDir, "."+Name)

	// manually update the power reduction by replacing micro (u) -> atto (a) evmos
	sdk.DefaultPowerReduction = evmostypes.PowerReduction
	// modify fee market parameter defaults through global
	feemarkettypes.DefaultMinGasPrice = MainnetMinGasPrices
	feemarkettypes.DefaultMinGasMultiplier = MainnetMinGasMultiplier
}

var (
	// DefaultNodeHome default home directories for the application daemon
	DefaultNodeHome string

	// ModuleBasics defines the module BasicManager is in charge of setting up basic,
	// non-dependant module elements, such as codec registration
	// and genesis verification.
	ModuleBasics = module.NewBasicManager(
		auth.AppModuleBasic{},
		genutil.NewAppModuleBasic(genutiltypes.DefaultMessageValidator),
		bank.AppModuleBasic{},
		capability.AppModuleBasic{},
		staking.AppModuleBasic{},
		exomint.AppModuleBasic{},
		gov.NewAppModuleBasic(
			[]govclient.ProposalHandler{
				paramsclient.ProposalHandler,
				upgradeclient.LegacyProposalHandler, upgradeclient.LegacyCancelProposalHandler,
				ibcclientclient.UpdateClientProposalHandler,
				ibcclientclient.UpgradeProposalHandler,
				// Evmos proposal types
			},
		),
		params.AppModuleBasic{},
		crisis.AppModuleBasic{},
		slashing.AppModuleBasic{},
		ibc.AppModuleBasic{},
		ibctm.AppModuleBasic{},
		ica.AppModuleBasic{},
		authzmodule.AppModuleBasic{},
		vesting.AppModuleBasic{},
		feegrantmodule.AppModuleBasic{},
		upgrade.AppModuleBasic{},
		evidence.AppModuleBasic{},
		transfer.AppModuleBasic{AppModuleBasic: &ibctransfer.AppModuleBasic{}},
		evm.AppModuleBasic{},
		feemarket.AppModuleBasic{},
		// evmos modules
		erc20.AppModuleBasic{},
		epochs.AppModuleBasic{},
		consensus.AppModuleBasic{},
		// Exocore modules
		assets.AppModuleBasic{},
		operator.AppModuleBasic{},
		delegation.AppModuleBasic{},
		reward.AppModuleBasic{},
		exoslash.AppModuleBasic{},
		avs.AppModuleBasic{},
		oracle.AppModuleBasic{},
		distr.AppModuleBasic{},
	)

	// module account permissions
	maccPerms = map[string][]string{
		authtypes.FeeCollectorName:  nil,
		govtypes.ModuleName:         {authtypes.Burner},
		ibctransfertypes.ModuleName: {authtypes.Minter, authtypes.Burner},
		icatypes.ModuleName:         nil,
		evmtypes.ModuleName: {
			authtypes.Minter,
			authtypes.Burner,
		}, // used for secure addition and subtraction of balance using module account
		exominttypes.ModuleName:           {authtypes.Minter},
		erc20types.ModuleName:             {authtypes.Minter, authtypes.Burner},
		delegationTypes.DelegatedPoolName: {authtypes.Burner, authtypes.Staking},
		distrtypes.ModuleName:             nil,
	}

	// module accounts that are allowed to receive tokens
	allowedReceivingModAcc = map[string]bool{}
)

var (
	_ servertypes.Application = (*ExocoreApp)(nil)
	_ ibctesting.TestingApp   = (*ExocoreApp)(nil)
)

// ExocoreApp implements an extended ABCI application. It is an application
// that may process transactions through Ethereum's EVM running atop of
// Tendermint consensus.
type ExocoreApp struct {
	*baseapp.BaseApp

	// encoding
	cdc               *codec.LegacyAmino
	appCodec          codec.Codec
	interfaceRegistry types.InterfaceRegistry

	invCheckPeriod uint

	// keys to access the substores
	keys    map[string]*storetypes.KVStoreKey
	tkeys   map[string]*storetypes.TransientStoreKey
	memKeys map[string]*storetypes.MemoryStoreKey

	// keepers
	AccountKeeper    authkeeper.AccountKeeper
	BankKeeper       bankkeeper.Keeper
	CapabilityKeeper *capabilitykeeper.Keeper
	StakingKeeper    stakingkeeper.Keeper
	SlashingKeeper   slashingkeeper.Keeper
	GovKeeper        govkeeper.Keeper
	CrisisKeeper     crisiskeeper.Keeper
	UpgradeKeeper    upgradekeeper.Keeper
	ParamsKeeper     paramskeeper.Keeper
	FeeGrantKeeper   feegrantkeeper.Keeper
	AuthzKeeper      authzkeeper.Keeper
	// IBC Keeper must be a pointer in the app, so we can SetRouter on it correctly
	IBCKeeper             *ibckeeper.Keeper
	ICAHostKeeper         icahostkeeper.Keeper
	EvidenceKeeper        evidencekeeper.Keeper
	TransferKeeper        transferkeeper.Keeper
	ConsensusParamsKeeper consensusparamkeeper.Keeper

	// make scoped keepers public for test purposes
	ScopedIBCKeeper      capabilitykeeper.ScopedKeeper
	ScopedTransferKeeper capabilitykeeper.ScopedKeeper

	// Ethermint keepers
	EvmKeeper       *evmkeeper.Keeper
	FeeMarketKeeper feemarketkeeper.Keeper

	// Evmos keepers
	Erc20Keeper  erc20keeper.Keeper
	EpochsKeeper epochskeeper.Keeper

	// exocore assets module keepers
	AssetsKeeper     assetsKeeper.Keeper
	DelegationKeeper delegationKeeper.Keeper
	RewardKeeper     rewardKeeper.Keeper
	OperatorKeeper   operatorKeeper.Keeper
	ExoSlashKeeper   slashKeeper.Keeper
	AVSManagerKeeper avsManagerKeeper.Keeper
	OracleKeeper     oracleKeeper.Keeper
	ExomintKeeper    exomintkeeper.Keeper
	DistrKeeper      distrkeeper.Keeper

	// the module manager
	mm *module.Manager

	// the configurator
	configurator module.Configurator

	// simulation manager
	sm *module.SimulationManager

	tpsCounter *tpsCounter
}

// SimulationManager implements runtime.AppI
func (*ExocoreApp) SimulationManager() *module.SimulationManager {
	panic("unimplemented")
}

// NewExocoreApp is the constructor for new Exocore
func NewExocoreApp(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	loadLatest bool,
	skipUpgradeHeights map[int64]bool,
	homePath string,
	invCheckPeriod uint,
	encodingConfig simappparams.EncodingConfig,
	appOpts servertypes.AppOptions,
	baseAppOptions ...func(*baseapp.BaseApp),
) *ExocoreApp {
	appCodec := encodingConfig.Codec
	cdc := encodingConfig.Amino
	interfaceRegistry := encodingConfig.InterfaceRegistry

	eip712.SetEncodingConfig(encodingConfig)

	// Setup Mempool and Proposal Handlers
	baseAppOptions = append(baseAppOptions, func(app *baseapp.BaseApp) {
		// NOTE: we use a NoOpMempool here, for oracle create-price, it works fine since we have set a infinitgasmeterwithlimit in the ante handler to avoid the out-of-gas error no matter what the amount/gas is set by tx builder, and we set the highest priority for oracle create-price txs to work properly with tendermint mempool to make sure oracle creat-prie tx will be included in the mempool if received. And if we want to use some other application mempool, we need to take care of the gas limit and gas price in the oracle create-price txs.(we don't need to bother this since tendermint mempool use gasMeter.limit() instead of tx.Gas())
		mempool := mempool.NoOpMempool{}
		app.SetMempool(mempool)
		handler := baseapp.NewDefaultProposalHandler(mempool, app)
		app.SetPrepareProposal(handler.PrepareProposalHandler())
		app.SetProcessProposal(handler.ProcessProposalHandler())
	})
	// NOTE we use custom transaction decoder that supports the sdk.Tx interface instead of
	// sdk.StdTx
	bApp := baseapp.NewBaseApp(
		Name,
		logger,
		db,
		encodingConfig.TxConfig.TxDecoder(),
		baseAppOptions...,
	)
	bApp.SetCommitMultiStoreTracer(traceStore)
	bApp.SetVersion(version.Version)
	bApp.SetInterfaceRegistry(interfaceRegistry)

	keys := sdk.NewKVStoreKeys(
		// SDK keys
		authtypes.StoreKey, banktypes.StoreKey, stakingtypes.StoreKey,
		slashingtypes.StoreKey,
		govtypes.StoreKey, paramstypes.StoreKey, upgradetypes.StoreKey,
		evidencetypes.StoreKey, capabilitytypes.StoreKey, consensusparamtypes.StoreKey,
		feegrant.StoreKey, authzkeeper.StoreKey, crisistypes.StoreKey,
		// ibc keys
		ibcexported.StoreKey, ibctransfertypes.StoreKey,
		// ica keys
		icahosttypes.StoreKey,
		// ethermint keys
		evmtypes.StoreKey, feemarkettypes.StoreKey,
		// evmos keys
		erc20types.StoreKey,
		epochstypes.StoreKey,
		// exoCore module keys
		assetsTypes.StoreKey,
		delegationTypes.StoreKey,
		rewardTypes.StoreKey,
		exoslashTypes.StoreKey,
		operatorTypes.StoreKey,
		avsManagerTypes.StoreKey,
		oracleTypes.StoreKey,
		exominttypes.StoreKey,
		distrtypes.StoreKey,
	)

	tkeys := sdk.NewTransientStoreKeys(paramstypes.TStoreKey, evmtypes.TransientKey, feemarkettypes.TransientKey)
	memKeys := sdk.NewMemoryStoreKeys(capabilitytypes.MemStoreKey, oracleTypes.MemStoreKey)

	// load state streaming if enabled
	if _, _, err := streaming.LoadStreamingServices(bApp, appOpts, appCodec, logger, keys); err != nil {
		fmt.Printf("failed to load state streaming: %s", err)
		os.Exit(1)
	}

	app := &ExocoreApp{
		BaseApp:           bApp,
		cdc:               cdc,
		appCodec:          appCodec,
		interfaceRegistry: interfaceRegistry,
		invCheckPeriod:    invCheckPeriod,
		keys:              keys,
		tkeys:             tkeys,
		memKeys:           memKeys,
	}

	// init params keeper and subspaces
	app.ParamsKeeper = initParamsKeeper(
		appCodec, cdc, keys[paramstypes.StoreKey], tkeys[paramstypes.TStoreKey],
	)

	// get the address of the authority, which is the governance module.
	// as the authority, the governance module can modify parameters in the modules that support
	// such modifications.
	authAddr := authtypes.NewModuleAddress(govtypes.ModuleName)
	authAddrString := authAddr.String()

	// set the BaseApp's parameter store which is used for setting Tendermint parameters
	app.ConsensusParamsKeeper = consensusparamkeeper.NewKeeper(
		appCodec, keys[consensusparamtypes.StoreKey], authAddrString,
	)
	bApp.SetParamStore(&app.ConsensusParamsKeeper)

	// add the account keeper first, since it is something required by almost every other module
	app.AccountKeeper = authkeeper.NewAccountKeeper(
		appCodec, keys[authtypes.StoreKey], evmostypes.ProtoAccount, maccPerms,
		sdk.GetConfig().GetBech32AccountAddrPrefix(), authAddrString,
	)

	// add the bank keeper, which is used to handle balances of accounts.
	app.BankKeeper = bankkeeper.NewBaseKeeper(
		appCodec, keys[banktypes.StoreKey], app.AccountKeeper,
		app.BlockedAddrs(), authAddrString,
	)

	// the crisis keeper is used to halt the chain in case of an invariant failure. typically,
	// invariants are clearly defined in respective modules and used to ensure that something
	// that must not change (invariant) has not actually changed. an example of an invariant
	// is that the total delegations given by delegators equal the total delegations received
	// by validators. this module is key to the system safety (although it hasn't been set up
	// for our modules) and requires the bank keeper so it must be initialized after that.
	app.CrisisKeeper = *crisiskeeper.NewKeeper(
		appCodec, keys[crisistypes.StoreKey],
		invCheckPeriod, // how many blocks to auto-check all invariants, 0 to disable.
		// to charge fees for user triggered checks, amount editable by governance.
		app.BankKeeper, authtypes.FeeCollectorName, authAddrString,
	)

	// the x/upgrade module is designed to allow seamless upgrades to the chain, without too
	// much manual coordination. if a proposal passes, the module stops the chain at a
	// predetermined height, executes the proposal (and associate module migrations), and then
	// resumes the chain. as long as the binary doesn't change for such proposals, this module
	// can handle any proposals by itself. however, if the binary needs to be updated, sidecar
	// software like cosmovisor must be used for unattended upgrades.
	app.UpgradeKeeper = *upgradekeeper.NewKeeper(
		skipUpgradeHeights, keys[upgradetypes.StoreKey], appCodec,
		homePath, app.BaseApp, authAddrString,
	)

	// UX module to allow an address to use some address's balance for gas fee (permiessioned).
	app.FeeGrantKeeper = feegrantkeeper.NewKeeper(
		appCodec, keys[feegrant.StoreKey], app.AccountKeeper,
	)

	// the x/authz module allows a grantor to give a grantee the right to execute txs on their
	// behalf. kind of UX module and kind of utility for the grantor.
	app.AuthzKeeper = authzkeeper.NewKeeper(
		keys[authzkeeper.StoreKey], appCodec, app.MsgServiceRouter(), app.AccountKeeper,
	)

	// the epochs keeper is used to count the number of epochs that have passed since genesis.
	// its params can be used to define the type of epoch to track (hour, week, day).
	app.EpochsKeeper = *epochskeeper.NewKeeper(appCodec, keys[epochstypes.StoreKey])

	// Exocore keepers begin. TODO: replace virtual keepers with actual implementation.

	// the exomint keeper is used to mint the reward for validators and delegators. it needs
	// the epochs keeper and the bank / account keepers.
	app.ExomintKeeper = exomintkeeper.NewKeeper(
		appCodec, keys[exominttypes.StoreKey],
		app.AccountKeeper, app.BankKeeper, app.EpochsKeeper, authtypes.FeeCollectorName,
		authAddrString,
	)

	// asset and client chain registry.
	app.AssetsKeeper = assetsKeeper.NewKeeper(
		keys[assetsTypes.StoreKey], appCodec, &app.OracleKeeper,
		app.BankKeeper, &app.DelegationKeeper, authAddrString,
	)

	// handles delegations by stakers, and must know if the delegatee operator is registered.
	app.DelegationKeeper = delegationKeeper.NewKeeper(
		keys[delegationTypes.StoreKey], appCodec,
		app.AssetsKeeper,
		delegationTypes.VirtualSlashKeeper{},
		&app.OperatorKeeper,
		app.AccountKeeper,
		app.BankKeeper,
	)

	// the dogfood module is the first AVS. it receives slashing calls from either x/slashing
	// or x/evidence and forwards them to the operator module which handles it.
	app.StakingKeeper = stakingkeeper.NewKeeper(
		appCodec, keys[stakingtypes.StoreKey],
		app.EpochsKeeper,     // epoch hook to be registered separately
		&app.OperatorKeeper,  // operator registration / opt in
		app.DelegationKeeper, // undelegation response
		app.AssetsKeeper,     // assets for vote power
		// intentionally a pointer since it is not yet initialized
		&app.AVSManagerKeeper, // used to create the AVS from the chainID
		authAddrString,        // authority to edit params
	)

	// these two modules aren't finalized yet.
	app.RewardKeeper = rewardKeeper.NewKeeper(
		appCodec, keys[rewardTypes.StoreKey], app.AssetsKeeper,
		app.AVSManagerKeeper, authAddrString,
	)
	app.ExoSlashKeeper = slashKeeper.NewKeeper(
		appCodec, keys[exoslashTypes.StoreKey], app.AssetsKeeper, authAddrString,
	)

	// x/oracle is not fully integrated (or enabled) but allows for exchange rates to be added.
	app.OracleKeeper = oracleKeeper.NewKeeper(
		appCodec, keys[oracleTypes.StoreKey], memKeys[oracleTypes.MemStoreKey],
		app.GetSubspace(oracleTypes.ModuleName), app.StakingKeeper,
		&app.DelegationKeeper, &app.AssetsKeeper, authAddrString,
	)

	// the SDK slashing module is used to slash validators in the case of downtime. it tracks
	// the validator signature rate and informs the staking keeper to perform the requisite
	// slashing. all its other operations are offloaded to Exocore keepers via the dogfood or
	// the operator module.
	// NOTE: the slashing keeper stores the parameters (slash rate) for the dogfood
	// keeper, since all slashing (for this chain) begins within this keeper.
	app.SlashingKeeper = slashingkeeper.NewKeeper(
		appCodec, app.LegacyAmino(), keys[slashingtypes.StoreKey],
		app.StakingKeeper, authAddrString,
	)

	// the evidence module handles any external evidence of misbehavior submitted to it, if such
	// an evidence is registered in its router. we have not set up any such router, and hence

	// this module cannot handle external evidence. however, by itself, the module is built
	// to handle evidence received from Tendermint, which is the equivocation evidence.
	// it is created after the Staking and Slashing keepers have been set up.
	app.EvidenceKeeper = *evidencekeeper.NewKeeper(
		appCodec, keys[evidencetypes.StoreKey], app.StakingKeeper, app.SlashingKeeper,
	)

	// initialize the IBC keeper but the rest of the stack is done after EVM.
	// add capability keeper and ScopeToModule for ibc module
	app.CapabilityKeeper = capabilitykeeper.NewKeeper(
		appCodec,
		keys[capabilitytypes.StoreKey],
		memKeys[capabilitytypes.MemStoreKey],
	)
	scopedIBCKeeper := app.CapabilityKeeper.ScopeToModule(ibcexported.ModuleName)
	scopedTransferKeeper := app.CapabilityKeeper.ScopeToModule(ibctransfertypes.ModuleName)
	scopedICAHostKeeper := app.CapabilityKeeper.ScopeToModule(icahosttypes.SubModuleName)
	// Applications that wish to enforce statically created ScopedKeepers should call `Seal`
	// after creating their scoped modules in `NewApp` with `ScopeToModule`
	app.CapabilityKeeper.Seal()
	// Create IBC Keeper
	app.IBCKeeper = ibckeeper.NewKeeper(
		appCodec,
		keys[ibcexported.StoreKey],
		app.GetSubspace(ibcexported.ModuleName),
		app.StakingKeeper,
		app.UpgradeKeeper,
		scopedIBCKeeper,
	)

	// add the governance module, with first step being setting up the proposal types.
	// any new proposals that are created within any module must be added here.
	govRouter := govv1beta1.NewRouter()
	govRouter.AddRoute(govtypes.RouterKey, govv1beta1.ProposalHandler).
		AddRoute(paramproposal.RouterKey, params.NewParamChangeProposalHandler(app.ParamsKeeper)).
		AddRoute(upgradetypes.RouterKey, upgrade.NewSoftwareUpgradeProposalHandler(&app.UpgradeKeeper)).
		AddRoute(ibcclienttypes.RouterKey, ibcclient.NewClientProposalHandler(app.IBCKeeper.ClientKeeper)).
		AddRoute(erc20types.RouterKey, erc20.NewErc20ProposalHandler(&app.Erc20Keeper))
	app.GovKeeper = *govkeeper.NewKeeper(
		appCodec, keys[govtypes.StoreKey], app.AccountKeeper, app.BankKeeper,
		// must be a pointer, since it is not yet initialized
		// (but could alternatively make governance keeper later)
		&app.StakingKeeper,
		app.MsgServiceRouter(), govtypes.DefaultConfig(), authAddrString,
	)
	// Set legacy router for backwards compatibility with gov v1beta1
	(&app.GovKeeper).SetLegacyRouter(govRouter)

	// the EVM stack is below.

	// the fee market keeper is used to increase or decrease block gas limit in response to
	// demands (from the previous block gas limit).
	app.FeeMarketKeeper = feemarketkeeper.NewKeeper(
		appCodec, authAddr,
		keys[feemarkettypes.StoreKey],
		tkeys[feemarkettypes.TransientKey],
		app.GetSubspace(feemarkettypes.ModuleName),
	)

	// the evm keeper adds the EVM to the Cosmos state machine.
	app.EvmKeeper = evmkeeper.NewKeeper(
		appCodec, keys[evmtypes.StoreKey], tkeys[evmtypes.TransientKey], authAddr,
		// to set account storage and/or code
		app.AccountKeeper,
		// to charge transaction fees
		app.BankKeeper,
		// to fetch prior block hash from within the precompile
		app.StakingKeeper,
		// EIP-1559 implementation
		app.FeeMarketKeeper,
		cast.ToString(appOpts.Get(srvflags.EVMTracer)),
		app.GetSubspace(evmtypes.ModuleName),
	)

	// the AVS manager keeper is the AVS registry. this keeper is initialized after the EVM
	// keeper because it depends on the EVM keeper to set a lookup from codeHash to code,
	// at genesis.
	app.AVSManagerKeeper = avsManagerKeeper.NewKeeper(
		appCodec, keys[avsManagerTypes.StoreKey],
		&app.OperatorKeeper,
		app.AssetsKeeper,
		app.EpochsKeeper,
		app.EvmKeeper,
	)
	// operator registry, which handles vote power (and this requires delegation keeper).
	// this keeper is initialized after the avs keeper because it depends on the avs keeper
	// to determine whether an AVS is registered or not.
	app.OperatorKeeper = operatorKeeper.NewKeeper(
		keys[operatorTypes.StoreKey], appCodec,
		app.AssetsKeeper,
		&app.DelegationKeeper, // intentionally a pointer, since not yet initialized.
		&app.OracleKeeper,
		&app.AVSManagerKeeper,
		delegationTypes.VirtualSlashKeeper{},
	)
	// the fee distribution keeper is used to allocate reward to exocore validators on epoch-basis,
	// and it'll interact with other modules, like delegation for voting power, mint and inflation and etc.
	// this keeper is initialized after the StakingKeeper  because it depends on the StakingKeeper
	app.DistrKeeper = distrkeeper.NewKeeper(
		appCodec, logger,
		authtypes.FeeCollectorName,
		authAddrString,
		keys[distrtypes.StoreKey],
		app.BankKeeper,
		app.AccountKeeper,
		app.StakingKeeper,
		app.EpochsKeeper,
	)

	app.EvmKeeper.WithPrecompiles(
		evmkeeper.AvailablePrecompiles(
			app.AuthzKeeper,
			app.TransferKeeper,
			app.IBCKeeper.ChannelKeeper,
			app.DelegationKeeper,
			app.AssetsKeeper,
			app.ExoSlashKeeper,
			app.RewardKeeper,
			app.AVSManagerKeeper,
		),
	)

	app.Erc20Keeper = erc20keeper.NewKeeper(
		keys[erc20types.StoreKey], appCodec, authtypes.NewModuleAddress(govtypes.ModuleName),
		app.AccountKeeper, app.BankKeeper, app.EvmKeeper, app.StakingKeeper,
		app.AuthzKeeper, &app.TransferKeeper,
	)

	app.TransferKeeper = transferkeeper.NewKeeper(
		appCodec, keys[ibctransfertypes.StoreKey], app.GetSubspace(ibctransfertypes.ModuleName),
		app.IBCKeeper.ChannelKeeper, // ICS4 Wrapper: claims IBC middleware
		app.IBCKeeper.ChannelKeeper, &app.IBCKeeper.PortKeeper,
		app.AccountKeeper, app.BankKeeper, scopedTransferKeeper,
		app.Erc20Keeper, // Add ERC20 Keeper for ERC20 transfers
	)

	// Override the ICS20 app module
	transferModule := transfer.NewAppModule(app.TransferKeeper)

	// Create the app.ICAHostKeeper
	app.ICAHostKeeper = icahostkeeper.NewKeeper(
		appCodec, app.keys[icahosttypes.StoreKey],
		app.GetSubspace(icahosttypes.SubModuleName),
		app.IBCKeeper.ChannelKeeper,
		app.IBCKeeper.ChannelKeeper,
		&app.IBCKeeper.PortKeeper,
		app.AccountKeeper,
		scopedICAHostKeeper,
		bApp.MsgServiceRouter(),
	)

	// create host IBC module
	icaHostIBCModule := icahost.NewIBCModule(app.ICAHostKeeper)

	/*
		Create Transfer Stack

		transfer stack contains (from bottom to top):
		- ERC-20 Middleware
		- Recovery Middleware
		- IBC Transfer

		SendPacket, since it is originating from the application to core IBC:
		transferKeeper.SendPacket -> recovery.SendPacket -> erc20.SendPacket -> channel.SendPacket

		RecvPacket, message that originates from core IBC and goes down to app, the flow is the other way
		channel.RecvPacket -> erc20.OnRecvPacket -> recovery.OnRecvPacket -> transfer.OnRecvPacket
	*/

	// create IBC module from top to bottom of stack
	var transferStack porttypes.IBCModule

	transferStack = transfer.NewIBCModule(app.TransferKeeper)
	transferStack = erc20.NewIBCMiddleware(app.Erc20Keeper, transferStack)

	// Create static IBC router, add transfer route, then set and seal it
	ibcRouter := porttypes.NewRouter()
	ibcRouter.
		AddRoute(icahosttypes.SubModuleName, icaHostIBCModule).
		AddRoute(ibctransfertypes.ModuleName, transferStack)

	app.IBCKeeper.SetRouter(ibcRouter)

	// set the hooks at the end, after all modules are instantiated.
	(&app.OperatorKeeper).SetHooks(
		app.StakingKeeper.OperatorHooks(),
	)

	(&app.DelegationKeeper).SetHooks(
		app.StakingKeeper.DelegationHooks(),
	)

	(&app.EpochsKeeper).SetHooks(
		epochstypes.NewMultiEpochHooks(
			app.DistrKeeper.EpochsHooks(),      // come first for using the voting power of last epoch
			app.OperatorKeeper.EpochsHooks(),   // must come before staking keeper so it can set the USD value
			app.StakingKeeper.EpochsHooks(),    // at this point, the order is irrelevant.
			app.ExomintKeeper.EpochsHooks(),    // however, this may change once we have distribution
			app.AVSManagerKeeper.EpochsHooks(), // no-op for now
		),
	)

	(&app.StakingKeeper).SetHooks(
		stakingtypes.NewMultiDogfoodHooks(
			app.SlashingKeeper.Hooks(),
		),
	)

	app.EvmKeeper.SetHooks(
		evmkeeper.NewMultiEvmHooks(
			app.Erc20Keeper.Hooks(),
		),
	)

	/****  Module Options ****/

	// NOTE: we may consider parsing `appOpts` inside module constructors. For the moment
	// we prefer to be more strict in what arguments the modules expect.
	skipGenesisInvariants := cast.ToBool(appOpts.Get(crisis.FlagSkipGenesisInvariants))

	// NOTE: Any module instantiated in the module manager that is later modified
	// must be passed by reference here.
	app.mm = module.NewManager(
		// SDK app modules
		genutil.NewAppModule(
			app.AccountKeeper, app.StakingKeeper, app.BaseApp.DeliverTx,
			encodingConfig.TxConfig,
		),
		auth.NewAppModule(
			appCodec, app.AccountKeeper,
			authsims.RandomGenesisAccounts,
			app.GetSubspace(authtypes.ModuleName),
		),
		vesting.NewAppModule(app.AccountKeeper, app.BankKeeper),
		bank.NewAppModule(
			appCodec, app.BankKeeper, app.AccountKeeper,
			app.GetSubspace(banktypes.ModuleName),
		),
		capability.NewAppModule(appCodec, *app.CapabilityKeeper, false),
		crisis.NewAppModule(
			&app.CrisisKeeper, skipGenesisInvariants,
			app.GetSubspace(crisistypes.ModuleName),
		),
		gov.NewAppModule(
			appCodec, &app.GovKeeper, app.AccountKeeper,
			app.BankKeeper, app.GetSubspace(govtypes.ModuleName),
		),
		slashing.NewAppModule(
			appCodec, app.SlashingKeeper, app.AccountKeeper,
			app.BankKeeper, app.StakingKeeper,
			app.GetSubspace(slashingtypes.ModuleName),
		),
		staking.NewAppModule(
			appCodec, app.StakingKeeper,
		),
		upgrade.NewAppModule(&app.UpgradeKeeper),
		evidence.NewAppModule(app.EvidenceKeeper),
		params.NewAppModule(app.ParamsKeeper),
		feegrantmodule.NewAppModule(
			appCodec, app.AccountKeeper, app.BankKeeper,
			app.FeeGrantKeeper,
			app.interfaceRegistry,
		),
		authzmodule.NewAppModule(
			appCodec,
			app.AuthzKeeper,
			app.AccountKeeper,
			app.BankKeeper,
			app.interfaceRegistry,
		),
		consensus.NewAppModule(appCodec, app.ConsensusParamsKeeper),

		// ibc modules
		ibc.NewAppModule(app.IBCKeeper),
		ica.NewAppModule(nil, &app.ICAHostKeeper),
		transferModule,
		// Ethermint app modules
		evm.NewAppModule(
			app.EvmKeeper,
			app.AccountKeeper,
			app.GetSubspace(evmtypes.ModuleName),
		),
		feemarket.NewAppModule(app.FeeMarketKeeper, app.GetSubspace(feemarkettypes.ModuleName)),
		// Evmos app modules
		erc20.NewAppModule(app.Erc20Keeper, app.AccountKeeper,
			app.GetSubspace(erc20types.ModuleName)),
		epochs.NewAppModule(appCodec, app.EpochsKeeper),
		// exoCore app modules
		exomint.NewAppModule(appCodec, app.ExomintKeeper),
		assets.NewAppModule(appCodec, app.AssetsKeeper),
		operator.NewAppModule(appCodec, app.OperatorKeeper),
		delegation.NewAppModule(appCodec, app.DelegationKeeper),
		reward.NewAppModule(appCodec, app.RewardKeeper),
		exoslash.NewAppModule(appCodec, app.ExoSlashKeeper),
		avs.NewAppModule(appCodec, app.AVSManagerKeeper),
		oracle.NewAppModule(appCodec, app.OracleKeeper, app.AccountKeeper, app.BankKeeper),
		distr.NewAppModule(appCodec, app.DistrKeeper),
	)

	// During begin block slashing happens after reward.BeginBlocker so that
	// there is nothing left over in the validator fee pool, to keep the
	// CanWithdrawInvariant invariant.
	app.mm.SetOrderBeginBlockers(
		upgradetypes.ModuleName,    // to upgrade the chain
		capabilitytypes.ModuleName, // before any module with capabilities like IBC
		epochstypes.ModuleName,     // to update the epoch
		feemarkettypes.ModuleName,  // set EIP-1559 gas prices
		evmtypes.ModuleName,        // stores chain id in memory
		slashingtypes.ModuleName,   // TODO after reward
		evidencetypes.ModuleName,   // TODO after reward
		stakingtypes.ModuleName,    // track historical info
		ibcexported.ModuleName,     // handles upgrades of chain and hence client
		authz.ModuleName,           // clear expired approvals
		// no-op modules
		ibctransfertypes.ModuleName,
		icatypes.ModuleName,
		authtypes.ModuleName,
		banktypes.ModuleName,
		govtypes.ModuleName,
		crisistypes.ModuleName,
		genutiltypes.ModuleName,
		feegrant.ModuleName,
		paramstypes.ModuleName,
		vestingtypes.ModuleName,
		consensusparamtypes.ModuleName,
		erc20types.ModuleName,
		exominttypes.ModuleName, // called via hooks not directly
		assetsTypes.ModuleName,
		operatorTypes.ModuleName,
		delegationTypes.ModuleName,
		rewardTypes.ModuleName,
		exoslashTypes.ModuleName,
		avsManagerTypes.ModuleName,
		oracleTypes.ModuleName,
		distrtypes.ModuleName,
	)

	app.mm.SetOrderEndBlockers(
		capabilitytypes.ModuleName,
		crisistypes.ModuleName,     // easy quit
		operatorTypes.ModuleName,   // first, so that USD value is recorded
		stakingtypes.ModuleName,    // uses the USD value recorded in operator to calculate vote power
		delegationTypes.ModuleName, // process the undelegations matured by dogfood
		govtypes.ModuleName,        // after staking keeper to ensure new vote powers
		oracleTypes.ModuleName,     // prepares for next round with new vote powers from staking keeper
		evmtypes.ModuleName,        // can be anywhere
		feegrant.ModuleName,        // can be anywhere
		// no-op modules
		ibcexported.ModuleName,
		ibctransfertypes.ModuleName,
		icatypes.ModuleName,
		authtypes.ModuleName,
		banktypes.ModuleName,
		slashingtypes.ModuleName, // begin blocker only
		genutiltypes.ModuleName,
		evidencetypes.ModuleName,
		authz.ModuleName,
		paramstypes.ModuleName,
		upgradetypes.ModuleName,
		vestingtypes.ModuleName,
		epochstypes.ModuleName, // begin blocker only
		erc20types.ModuleName,
		exominttypes.ModuleName,
		consensusparamtypes.ModuleName,
		assetsTypes.ModuleName,
		rewardTypes.ModuleName,
		exoslashTypes.ModuleName,
		avsManagerTypes.ModuleName,
		distrtypes.ModuleName,
		// op module
		feemarkettypes.ModuleName, // last in order to retrieve the block gas used
	)

	app.mm.SetOrderInitGenesis(
		// first, so other modules can claim capabilities safely.
		capabilitytypes.ModuleName,
		// initial accounts
		authtypes.ModuleName,
		// their balances
		banktypes.ModuleName,
		// gas paid by other accounts
		feegrant.ModuleName,
		// permissions to execute txs on behalf of other accounts
		authz.ModuleName,
		feemarkettypes.ModuleName,
		genutiltypes.ModuleName, // after feemarket
		epochstypes.ModuleName,  // must be before dogfood and exomint
		evmtypes.ModuleName,     // must be before avs, since dogfood calls avs which calls this
		exominttypes.ModuleName,
		assetsTypes.ModuleName,
		avsManagerTypes.ModuleName, // before dogfood, since dogfood registers itself as an AVS
		operatorTypes.ModuleName,   // must be before delegation
		delegationTypes.ModuleName,
		stakingtypes.ModuleName, // must be after delegation
		// must be after staking to `IterateValidators` but it is not implemented anyway
		slashingtypes.ModuleName,
		evidencetypes.ModuleName,
		govtypes.ModuleName, // can be anywhere after bank
		erc20types.ModuleName,
		ibcexported.ModuleName,
		ibctransfertypes.ModuleName,
		icatypes.ModuleName,
		oracleTypes.ModuleName, // after staking module to ensure total vote power available
		// no-op modules
		paramstypes.ModuleName,
		vestingtypes.ModuleName,
		consensusparamtypes.ModuleName,
		upgradetypes.ModuleName,  // no-op since we don't call SetInitVersionMap
		rewardTypes.ModuleName,   // not fully implemented yet
		exoslashTypes.ModuleName, // not fully implemented yet
		distrtypes.ModuleName,
		// must be the last module after others have been set up, so that it can check
		// the invariants (if configured to do so).
		crisistypes.ModuleName,
	)

	app.mm.RegisterInvariants(&app.CrisisKeeper)
	app.configurator = module.NewConfigurator(
		app.appCodec,
		app.MsgServiceRouter(),
		app.GRPCQueryRouter(),
	)
	app.mm.RegisterServices(app.configurator)

	// add test gRPC service for testing gRPC queries in isolation
	// testdata.RegisterTestServiceServer(app.GRPCQueryRouter(), testdata.TestServiceImpl{})

	// create the simulation manager and define the order of the modules for deterministic
	// simulations
	//
	// NOTE: this is not required apps that don't use the simulator for fuzz testing
	// transactions
	overrideModules := map[string]module.AppModuleSimulation{
		authtypes.ModuleName: auth.NewAppModule(
			app.appCodec,
			app.AccountKeeper,
			authsims.RandomGenesisAccounts,
			app.GetSubspace(authtypes.ModuleName),
		),
	}
	app.sm = module.NewSimulationManagerFromAppModules(app.mm.Modules, overrideModules)

	autocliv1.RegisterQueryServer(
		app.GRPCQueryRouter(),
		runtimeservices.NewAutoCLIQueryService(app.mm.Modules),
	)

	reflectionSvc, err := runtimeservices.NewReflectionService()
	if err != nil {
		panic(err)
	}
	reflectionv1.RegisterReflectionServiceServer(app.GRPCQueryRouter(), reflectionSvc)

	app.sm.RegisterStoreDecoders()

	// initialize stores
	app.MountKVStores(keys)
	app.MountTransientStores(tkeys)
	app.MountMemoryStores(memKeys)

	// initialize BaseApp
	app.SetInitChainer(app.InitChainer)
	app.SetBeginBlocker(app.BeginBlocker)

	maxGasWanted := cast.ToUint64(appOpts.Get(srvflags.EVMMaxTxGasWanted))

	app.setAnteHandler(encodingConfig.TxConfig, maxGasWanted)
	app.setPostHandler()
	app.SetEndBlocker(app.EndBlocker)

	if loadLatest {
		if err := app.LoadLatestVersion(); err != nil {
			logger.Error("error on loading last version", "err", err)
			os.Exit(1)
		}
	}

	app.ScopedIBCKeeper = scopedIBCKeeper
	app.ScopedTransferKeeper = scopedTransferKeeper

	// Finally start the tpsCounter.
	app.tpsCounter = newTPSCounter(logger)
	go func() {
		// Unfortunately golangci-lint is so pedantic
		// so we have to ignore this error explicitly.
		_ = app.tpsCounter.start(context.Background())
	}()

	return app
}

// Name returns the name of the App
func (app *ExocoreApp) Name() string { return app.BaseApp.Name() }

func (app *ExocoreApp) setAnteHandler(txConfig client.TxConfig, maxGasWanted uint64) {
	options := ante.HandlerOptions{
		Cdc:                    app.appCodec,
		AccountKeeper:          app.AccountKeeper,
		BankKeeper:             app.BankKeeper,
		ExtensionOptionChecker: evmostypes.HasDynamicFeeExtensionOption,
		StakingKeeper:          app.StakingKeeper,
		FeegrantKeeper:         app.FeeGrantKeeper,
		DistributionKeeper:     app.RewardKeeper,
		IBCKeeper:              app.IBCKeeper,
		SignModeHandler:        txConfig.SignModeHandler(),
		SigGasConsumer:         ante.SigVerificationGasConsumer,
		MaxTxGasWanted:         maxGasWanted,
		FeeMarketKeeper:        app.FeeMarketKeeper,
		EvmKeeper:              app.EvmKeeper,
		TxFeeChecker:           ethante.NewDynamicFeeChecker(app.EvmKeeper),
		OracleKeeper:           app.OracleKeeper,
	}

	if err := options.Validate(); err != nil {
		panic(err)
	}

	app.SetAnteHandler(ante.NewAnteHandler(options))
}

func (app *ExocoreApp) setPostHandler() {
	postHandler, err := posthandler.NewPostHandler(
		posthandler.HandlerOptions{},
	)
	if err != nil {
		panic(err)
	}

	app.SetPostHandler(postHandler)
}

// BeginBlocker runs the Tendermint ABCI BeginBlock logic. It executes state changes at the
// beginning of the new block for every registered module. If there is a registered fork at the
// current height,
// BeginBlocker will schedule the upgrade plan and perform the state migration (if any).
func (app *ExocoreApp) BeginBlocker(
	ctx sdk.Context,
	req abci.RequestBeginBlock,
) abci.ResponseBeginBlock {
	// Perform any scheduled forks before executing the modules logic
	app.ScheduleForkUpgrade(ctx)
	return app.mm.BeginBlock(ctx, req)
}

// EndBlocker updates every end block
func (app *ExocoreApp) EndBlocker(
	ctx sdk.Context,
	req abci.RequestEndBlock,
) abci.ResponseEndBlock {
	return app.mm.EndBlock(ctx, req)
}

// The DeliverTx method is intentionally decomposed to calculate the transactions per second.
func (app *ExocoreApp) DeliverTx(req abci.RequestDeliverTx) (res abci.ResponseDeliverTx) {
	defer func() {
		// TODO: Record the count along with the code and or reason so as to display
		// in the transactions per second live dashboards.
		if res.IsErr() {
			app.tpsCounter.incrementFailure()
		} else {
			app.tpsCounter.incrementSuccess()
		}
	}()
	return app.BaseApp.DeliverTx(req)
}

// InitChainer updates at chain initialization
func (app *ExocoreApp) InitChainer(
	ctx sdk.Context,
	req abci.RequestInitChain,
) abci.ResponseInitChain {
	var genesisState simapp.GenesisState
	if err := json.Unmarshal(req.AppStateBytes, &genesisState); err != nil {
		panic(err)
	}

	app.UpgradeKeeper.SetModuleVersionMap(ctx, app.mm.GetVersionMap())

	return app.mm.InitGenesis(ctx, app.appCodec, genesisState)
}

// LoadHeight loads state at a particular height
func (app *ExocoreApp) LoadHeight(height int64) error {
	return app.LoadVersion(height)
}

// ModuleAccountAddrs returns all the app's module account addresses.
func (app *ExocoreApp) ModuleAccountAddrs() map[string]bool {
	modAccAddrs := make(map[string]bool)

	accs := make([]string, 0, len(maccPerms))
	for k := range maccPerms {
		accs = append(accs, k)
	}
	sort.Strings(accs)

	for _, acc := range accs {
		modAccAddrs[authtypes.NewModuleAddress(acc).String()] = true
	}

	return modAccAddrs
}

// LegacyAmino returns Evmos's amino codec.
//
// NOTE: This is solely to be used for testing purposes as it may be desirable
// for modules to register their own custom testing types.
func (app *ExocoreApp) LegacyAmino() *codec.LegacyAmino {
	return app.cdc
}

// AppCodec returns Evmos's app codec.
//
// NOTE: This is solely to be used for testing purposes as it may be desirable
// for modules to register their own custom testing types.
func (app *ExocoreApp) AppCodec() codec.Codec {
	return app.appCodec
}

// InterfaceRegistry returns Evmos's InterfaceRegistry
func (app *ExocoreApp) InterfaceRegistry() types.InterfaceRegistry {
	return app.interfaceRegistry
}

// GetKey returns the KVStoreKey for the provided store key.
//
// NOTE: This is solely to be used for testing purposes.
func (app *ExocoreApp) GetKey(storeKey string) *storetypes.KVStoreKey {
	return app.keys[storeKey]
}

// GetTKey returns the TransientStoreKey for the provided store key.
//
// NOTE: This is solely to be used for testing purposes.
func (app *ExocoreApp) GetTKey(storeKey string) *storetypes.TransientStoreKey {
	return app.tkeys[storeKey]
}

// GetMemKey returns the MemStoreKey for the provided mem key.
//
// NOTE: This is solely used for testing purposes.
func (app *ExocoreApp) GetMemKey(storeKey string) *storetypes.MemoryStoreKey {
	return app.memKeys[storeKey]
}

// GetSubspace returns a param subspace for a given module name.
//
// NOTE: This is solely to be used for testing purposes.
func (app *ExocoreApp) GetSubspace(moduleName string) paramstypes.Subspace {
	subspace, _ := app.ParamsKeeper.GetSubspace(moduleName)
	return subspace
}

// RegisterAPIRoutes registers all application module routes with the provided
// API server.
func (app *ExocoreApp) RegisterAPIRoutes(apiSvr *api.Server, apiConfig config.APIConfig) {
	clientCtx := apiSvr.ClientCtx

	// Register new tx routes from grpc-gateway.
	authtx.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)
	// Register new tendermint queries routes from grpc-gateway.
	tmservice.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)
	// Register node gRPC service for grpc-gateway.
	node.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// Register legacy and grpc-gateway routes for all modules.
	ModuleBasics.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// register swagger API from root so that other applications can override easily
	if apiConfig.Swagger {
		RegisterSwaggerAPI(clientCtx, apiSvr.Router)
	}
}

func (app *ExocoreApp) RegisterTxService(clientCtx client.Context) {
	authtx.RegisterTxService(
		app.BaseApp.GRPCQueryRouter(),
		clientCtx,
		app.BaseApp.Simulate,
		app.interfaceRegistry,
	)
}

// RegisterTendermintService implements the Application.RegisterTendermintService method.
func (app *ExocoreApp) RegisterTendermintService(clientCtx client.Context) {
	tmservice.RegisterTendermintService(
		clientCtx,
		app.BaseApp.GRPCQueryRouter(),
		app.interfaceRegistry,
		app.Query,
	)
}

// RegisterNodeService registers the node gRPC service on the provided
// application gRPC query router.
func (app *ExocoreApp) RegisterNodeService(clientCtx client.Context) {
	node.RegisterNodeService(clientCtx, app.GRPCQueryRouter())
}

// IBC Go TestingApp functions

// GetBaseApp implements the TestingApp interface.
func (app *ExocoreApp) GetBaseApp() *baseapp.BaseApp {
	return app.BaseApp
}

// GetStakingKeeper implements the TestingApp interface.
func (app *ExocoreApp) GetStakingKeeper() ibctestingtypes.StakingKeeper {
	return app.StakingKeeper
}

// GetIBCKeeper implements the TestingApp interface.
func (app *ExocoreApp) GetIBCKeeper() *ibckeeper.Keeper {
	return app.IBCKeeper
}

// GetTxConfig implements the TestingApp interface.
func (app *ExocoreApp) GetTxConfig() client.TxConfig {
	cfg := encoding.MakeConfig(ModuleBasics)
	return cfg.TxConfig
}

// RegisterSwaggerAPI registers swagger route with API Server
func RegisterSwaggerAPI(_ client.Context, rtr *mux.Router) {
	statikFS, err := fs.New()
	if err != nil {
		panic(err)
	}

	staticServer := http.FileServer(statikFS)
	rtr.PathPrefix("/swagger/").Handler(http.StripPrefix("/swagger/", staticServer))
}

// GetMaccPerms returns a copy of the module account permissions
func GetMaccPerms() map[string][]string {
	dupMaccPerms := make(map[string][]string)
	for k, v := range maccPerms {
		dupMaccPerms[k] = v
	}

	return dupMaccPerms
}

// initParamsKeeper init params keeper and its subspaces
func initParamsKeeper(
	appCodec codec.BinaryCodec, legacyAmino *codec.LegacyAmino, key, tkey storetypes.StoreKey,
) paramskeeper.Keeper {
	paramsKeeper := paramskeeper.NewKeeper(appCodec, legacyAmino, key, tkey)

	// SDK subspaces
	paramsKeeper.Subspace(authtypes.ModuleName)
	paramsKeeper.Subspace(banktypes.ModuleName)
	paramsKeeper.Subspace(slashingtypes.ModuleName)
	paramsKeeper.Subspace(govtypes.ModuleName).
		WithKeyTable(govv1.ParamKeyTable()) //nolint:staticcheck
	paramsKeeper.Subspace(crisistypes.ModuleName)
	paramsKeeper.Subspace(ibctransfertypes.ModuleName)
	paramsKeeper.Subspace(ibcexported.ModuleName)
	paramsKeeper.Subspace(icahosttypes.SubModuleName)
	// ethermint subspaces
	// nolint:staticcheck
	paramsKeeper.Subspace(evmtypes.ModuleName).WithKeyTable(evmtypes.ParamKeyTable())
	paramsKeeper.Subspace(oracleTypes.ModuleName).WithKeyTable(oracleTypes.ParamKeyTable())
	return paramsKeeper
}

// BlockedAddrs returns all the app's module account addresses that are not
// allowed to receive external tokens.
func (app *ExocoreApp) BlockedAddrs() map[string]bool {
	blockedAddrs := make(map[string]bool)

	accs := make([]string, 0, len(maccPerms))
	for k := range maccPerms {
		accs = append(accs, k)
	}
	sort.Strings(accs)

	for _, acc := range accs {
		blockedAddrs[authtypes.NewModuleAddress(acc).String()] = !allowedReceivingModAcc[acc]
	}

	for _, precompile := range common.DefaultPrecompilesBech32 {
		blockedAddrs[precompile] = true
	}

	return blockedAddrs
}

// GetScopedIBCKeeper implements the TestingApp interface.
func (app *ExocoreApp) GetScopedIBCKeeper() capabilitykeeper.ScopedKeeper {
	return app.ScopedIBCKeeper
}
