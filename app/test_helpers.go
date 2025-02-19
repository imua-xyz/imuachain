package app

import (
	"encoding/json"
	"os"
	"time"

	pruningtypes "github.com/cosmos/cosmos-sdk/store/pruning/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"

	"cosmossdk.io/math"
	"cosmossdk.io/simapp"
	dbm "github.com/cometbft/cometbft-db"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/log"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	tmtypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
	"github.com/cosmos/ibc-go/v7/testing/mock"

	avstypes "github.com/ExocoreNetwork/exocore/x/avs/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/evmos/evmos/v16/crypto/ethsecp256k1"
	"github.com/evmos/evmos/v16/encoding"
	evmostypes "github.com/evmos/evmos/v16/types"
	feemarkettypes "github.com/evmos/evmos/v16/x/feemarket/types"

	"github.com/ExocoreNetwork/exocore/cmd/config"
	"github.com/ExocoreNetwork/exocore/utils"
	assetstypes "github.com/ExocoreNetwork/exocore/x/assets/types"
	delegationtypes "github.com/ExocoreNetwork/exocore/x/delegation/types"
	dogfoodtypes "github.com/ExocoreNetwork/exocore/x/dogfood/types"
	operatortypes "github.com/ExocoreNetwork/exocore/x/operator/types"
	oracletypes "github.com/ExocoreNetwork/exocore/x/oracle/types"
)

func init() {
	cfg := sdk.GetConfig()
	config.SetBech32Prefixes(cfg)
	config.SetBip44CoinType(cfg)
}

// DefaultTestingAppInit defines the IBC application used for testing
var DefaultTestingAppInit func(chainID string, pruneOpts *pruningtypes.PruningOptions, isPrintLog bool) func() (ibctesting.TestingApp, map[string]json.RawMessage) = SetupTestingApp

// DefaultConsensusParams defines the default Tendermint consensus params used in
// Evmos testing.
var DefaultConsensusParams = &tmproto.ConsensusParams{
	Block: &tmproto.BlockParams{
		MaxBytes: 200000,
		MaxGas:   -1, // no limit
	},
	Evidence: &tmproto.EvidenceParams{
		MaxAgeNumBlocks: 302400,
		MaxAgeDuration:  504 * time.Hour, // 3 weeks is the max duration
		MaxBytes:        10000,
	},
	Validator: &tmproto.ValidatorParams{
		PubKeyTypes: []string{
			tmtypes.ABCIPubKeyTypeEd25519,
		},
	},
}

func init() {
	feemarkettypes.DefaultMinGasPrice = sdk.ZeroDec()
	cfg := sdk.GetConfig()
	config.SetBech32Prefixes(cfg)
	config.SetBip44CoinType(cfg)
}

// Setup initializes a new Exocore. A Nop logger is set in Exocore.
func Setup(
	isCheckTx bool,
	feemarketGenesis *feemarkettypes.GenesisState,
	chainID string,
	isPrintLog bool,
) *ExocoreApp {
	privVal := mock.NewPV()
	pubKey, _ := privVal.GetPubKey()

	// create validator set with single validator
	validator := tmtypes.NewValidator(pubKey, 1)
	valSet := tmtypes.NewValidatorSet([]*tmtypes.Validator{validator})

	// generate genesis account
	senderPrivKey := secp256k1.GenPrivKey()
	acc := authtypes.NewBaseAccount(senderPrivKey.PubKey().Address().Bytes(), senderPrivKey.PubKey(), 0, 0)
	balance := banktypes.Balance{
		Address: acc.GetAddress().String(),
		Coins:   sdk.NewCoins(sdk.NewCoin(utils.BaseDenom, sdk.NewInt(100000000000000))),
	}

	db := dbm.NewMemDB()
	var logger log.Logger
	if isPrintLog {
		logger = log.NewTMLogger(log.NewSyncWriter(os.Stdout))
	} else {
		logger = log.NewNopLogger()
	}
	app := NewExocoreApp(
		logger,
		db, nil, true, map[int64]bool{},
		DefaultNodeHome, 5,
		encoding.MakeConfig(ModuleBasics),
		simtestutil.NewAppOptionsWithFlagHome(DefaultNodeHome),
		baseapp.SetChainID(chainID),
	)
	if !isCheckTx {
		// init chain must be called to stop deliverState from being nil
		genesisState := NewDefaultGenesisState(app.appCodec)
		genesisState = GenesisStateWithValSet(app, genesisState, valSet, []authtypes.GenesisAccount{acc}, balance)

		// Verify feeMarket genesis
		if feemarketGenesis != nil {
			if err := feemarketGenesis.Validate(); err != nil {
				panic(err)
			}
			genesisState[feemarkettypes.ModuleName] = app.AppCodec().MustMarshalJSON(feemarketGenesis)
		}

		stateBytes, err := json.MarshalIndent(genesisState, "", " ")
		if err != nil {
			panic(err)
		}

		// Initialize the chain
		app.InitChain(
			abci.RequestInitChain{
				ChainId:         chainID,
				Validators:      []abci.ValidatorUpdate{},
				ConsensusParams: DefaultConsensusParams,
				AppStateBytes:   stateBytes,
			},
		)
	}

	return app
}

func GenesisStateWithValSet(app *ExocoreApp, genesisState simapp.GenesisState,
	valSet *tmtypes.ValidatorSet, genAccs []authtypes.GenesisAccount,
	balances ...banktypes.Balance,
) simapp.GenesisState {
	// set genesis accounts
	authGenesis := authtypes.NewGenesisState(authtypes.DefaultParams(), genAccs)
	genesisState[authtypes.ModuleName] = app.AppCodec().MustMarshalJSON(authGenesis)

	// x/assets
	clientChains := []assetstypes.ClientChainInfo{
		{
			Name:               "ethereum",
			MetaInfo:           "ethereum blockchain",
			ChainId:            1,
			FinalizationBlocks: 10,
			LayerZeroChainID:   101,
			AddressLength:      20,
		},
	}
	assets := []assetstypes.AssetInfo{
		{
			Name:             "Tether USD",
			Symbol:           "USDT",
			Address:          "0xdAC17F958D2ee523a2206206994597C13D831ec7",
			Decimals:         6,
			LayerZeroChainID: clientChains[0].LayerZeroChainID,
			MetaInfo:         "Tether USD token",
		},
	}

	// x/operator initialization - address only
	privkey, _ := ethsecp256k1.GenerateKey()
	key, _ := privkey.ToECDSA()
	operator := crypto.PubkeyToAddress(key.PublicKey)
	stakerID, _ := assetstypes.GetStakerIDAndAssetIDFromStr(
		clientChains[0].LayerZeroChainID,
		common.Address(operator.Bytes()).String(), "",
	)
	_, assetID := assetstypes.GetStakerIDAndAssetIDFromStr(
		clientChains[0].LayerZeroChainID,
		"", assets[0].Address,
	)
	depositAmount := sdk.TokensFromConsensusPower(1, evmostypes.PowerReduction)
	depositsByStaker := []assetstypes.DepositsByStaker{
		{
			StakerID: stakerID,
			Deposits: []assetstypes.DepositByAsset{
				{
					AssetID: assetID,
					Info: assetstypes.StakerAssetInfo{
						TotalDepositAmount:        depositAmount,
						WithdrawableAmount:        depositAmount,
						PendingUndelegationAmount: sdk.ZeroInt(),
					},
				},
			},
		},
	}
	// x/oracle initialization
	oracleDefaultParams := oracletypes.DefaultParams()
	oracleDefaultParams.TokenFeeders[1].StartBaseBlock = 1
	oracleGenesis := oracletypes.NewGenesisState(oracleDefaultParams)
	genesisState[oracletypes.ModuleName] = app.AppCodec().MustMarshalJSON(oracleGenesis)

	assetsGenesis := assetstypes.NewGenesis(
		assetstypes.DefaultParams(),
		clientChains, []assetstypes.StakingAssetInfo{
			{
				AssetBasicInfo: assets[0],
				// required to be 0, since deposits are handled after token init.
				StakingTotalAmount: sdk.ZeroInt(),
			},
		}, depositsByStaker,
		[]assetstypes.AssetsByOperator{
			{
				Operator: operator.String(),
				AssetsState: []assetstypes.AssetByID{
					{
						AssetID: assetID,
						Info: assetstypes.OperatorAssetInfo{
							TotalAmount:               depositAmount,
							PendingUndelegationAmount: math.NewInt(0),
							TotalShare:                math.LegacyNewDecFromBigInt(depositAmount.BigInt()),
							OperatorShare:             math.LegacyNewDec(0),
						},
					},
				},
			},
		},
	)
	genesisState[assetstypes.ModuleName] = app.AppCodec().MustMarshalJSON(assetsGenesis)
	// operator registration
	operatorInfos := []operatortypes.OperatorDetail{
		{
			OperatorAddress: operator.String(),
			OperatorInfo: operatortypes.OperatorInfo{
				EarningsAddr:     operator.String(),
				OperatorMetaInfo: "operator1",
				ApproveAddr:      operator.String(),
				Commission:       stakingtypes.NewCommission(sdk.ZeroDec(), sdk.ZeroDec(), sdk.ZeroDec()),
			},
		},
	}
	operatorGenesis := operatortypes.NewGenesisState(operatorInfos, nil, nil, nil, nil, nil, nil, nil)
	genesisState[operatortypes.ModuleName] = app.AppCodec().MustMarshalJSON(operatorGenesis)
	// x/delegation
	singleStateKey := assetstypes.GetJoinedStoreKey(stakerID, assetID, operator.String())
	delegationStates := []delegationtypes.DelegationStates{
		{
			Key: string(singleStateKey),
			States: delegationtypes.DelegationAmounts{
				WaitUndelegationAmount: math.NewInt(0),
				UndelegatableShare:     math.LegacyNewDecFromBigInt(depositAmount.BigInt()),
			},
		},
	}
	associations := []delegationtypes.StakerToOperator{
		{
			Operator: operator.String(),
			StakerId: stakerID,
		},
	}
	stakersByOperator := []delegationtypes.StakersByOperator{
		{
			Key: string(assetstypes.GetJoinedStoreKey(operator.String(), assetID)),
			Stakers: []string{
				stakerID,
			},
		},
	}
	delegationGenesis := delegationtypes.NewGenesis(associations, delegationStates, stakersByOperator, nil)
	genesisState[delegationtypes.ModuleName] = app.AppCodec().MustMarshalJSON(delegationGenesis)

	// create a dogfood genesis with just the validator set, that is, the bare
	// minimum valid genesis required to start a chain.
	dogfoodGenesis := dogfoodtypes.NewGenesis(
		dogfoodtypes.DefaultParams(), []dogfoodtypes.GenesisValidator{
			{
				Power:     1,
				PublicKey: hexutil.Encode(valSet.Validators[0].PubKey.Bytes()),
			},
		},
		[]dogfoodtypes.EpochToOperatorAddrs{}, []dogfoodtypes.EpochToConsensusAddrs{},
		[]dogfoodtypes.EpochToUndelegationRecordKeys{},
		math.NewInt(1), // total vote power
	)
	genesisState[dogfoodtypes.ModuleName] = app.AppCodec().MustMarshalJSON(dogfoodGenesis)

	totalSupply := sdk.NewCoins()
	for _, b := range balances {
		// add genesis acc tokens to total supply
		totalSupply = totalSupply.Add(b.Coins...)
	}
	bankGenesis := banktypes.NewGenesisState(
		banktypes.DefaultParams(), balances, totalSupply,
		[]banktypes.Metadata{}, []banktypes.SendEnabled{},
	)
	genesisState[banktypes.ModuleName] = app.AppCodec().MustMarshalJSON(bankGenesis)

	avsGenesis := avstypes.DefaultGenesis()
	genesisState[avstypes.ModuleName] = app.AppCodec().MustMarshalJSON(avsGenesis)

	return genesisState
}

// SetupTestingApp initializes the IBC-go testing application
// need to keep this design to comply with the ibctesting SetupTestingApp func
// and be able to set the chainID for the tests properly
func SetupTestingApp(chainID string, pruneOpts *pruningtypes.PruningOptions, isPrintLog bool) func() (ibctesting.TestingApp, map[string]json.RawMessage) {
	return func() (ibctesting.TestingApp, map[string]json.RawMessage) {
		db := dbm.NewMemDB()
		cfg := encoding.MakeConfig(ModuleBasics)
		logger := log.NewNopLogger()
		if isPrintLog {
			logger = log.NewTMLogger(log.NewSyncWriter(os.Stdout))
		}
		baseAppOptions := make([]func(*baseapp.BaseApp), 0)
		baseAppOptions = append(baseAppOptions, baseapp.SetChainID(chainID))
		if pruneOpts != nil {
			baseAppOptions = append(baseAppOptions, baseapp.SetPruning(*pruneOpts))
		}
		app := NewExocoreApp(
			logger,
			db, nil, true,
			map[int64]bool{},
			DefaultNodeHome, 5, cfg,
			simtestutil.NewAppOptionsWithFlagHome(DefaultNodeHome),
			baseAppOptions...,
		)
		return app, NewDefaultGenesisState(app.appCodec)
	}
}
