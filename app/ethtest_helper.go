package app

import (
	"encoding/json"
	"time"

	"cosmossdk.io/math"
	"cosmossdk.io/simapp"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/testutil/mock"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ExocoreNetwork/exocore/utils"
	dbm "github.com/cometbft/cometbft-db"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/log"
	tmtypes "github.com/cometbft/cometbft/proto/tendermint/types"
	cmtypes "github.com/cometbft/cometbft/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/evmos/evmos/v16/crypto/ethsecp256k1"
	"github.com/evmos/evmos/v16/encoding"
	evmostypes "github.com/evmos/evmos/v16/types"

	assetstypes "github.com/ExocoreNetwork/exocore/x/assets/types"
	delegationtypes "github.com/ExocoreNetwork/exocore/x/delegation/types"
	dogfoodtypes "github.com/ExocoreNetwork/exocore/x/dogfood/types"
	operatortypes "github.com/ExocoreNetwork/exocore/x/operator/types"
	oracletypes "github.com/ExocoreNetwork/exocore/x/oracle/types"
)

// EthDefaultConsensusParams defines the default Tendermint consensus params used in
// EvmosApp testing.
var EthDefaultConsensusParams = &tmtypes.ConsensusParams{
	Block: &tmtypes.BlockParams{
		MaxBytes: 200000,
		MaxGas:   -1, // no limit
	},
	Evidence: &tmtypes.EvidenceParams{
		MaxAgeNumBlocks: 302400,
		MaxAgeDuration:  504 * time.Hour, // 3 weeks is the max duration
		MaxBytes:        10000,
	},
	Validator: &tmtypes.ValidatorParams{
		PubKeyTypes: []string{
			cmtypes.ABCIPubKeyTypeEd25519,
		},
	},
}

// EthSetup initializes a new EvmosApp. A Nop logger is set in EvmosApp.
func EthSetup(isCheckTx bool, patchGenesis func(*ExocoreApp, simapp.GenesisState) simapp.GenesisState) *ExocoreApp {
	return EthSetupWithDB(isCheckTx, patchGenesis, dbm.NewMemDB())
}

// EthSetupWithDB initializes a new EvmosApp. A Nop logger is set in EvmosApp.
func EthSetupWithDB(isCheckTx bool, patchGenesis func(*ExocoreApp, simapp.GenesisState) simapp.GenesisState, db dbm.DB) *ExocoreApp {
	chainID := utils.TestnetChainID + "-1"
	app := NewExocoreApp(log.NewNopLogger(),
		db,
		nil,
		true,
		map[int64]bool{},
		DefaultNodeHome,
		5,
		encoding.MakeConfig(ModuleBasics),
		simtestutil.NewAppOptionsWithFlagHome(DefaultNodeHome),
		baseapp.SetChainID(chainID),
	)
	if !isCheckTx {
		// init chain must be called to stop deliverState from being nil
		genesisState := NewTestGenesisState(app.AppCodec())
		if patchGenesis != nil {
			genesisState = patchGenesis(app, genesisState)
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

// NewTestGenesisState generate genesis state with single validator
func NewTestGenesisState(codec codec.Codec) simapp.GenesisState {
	privVal := mock.NewPV()
	pubKey, err := privVal.GetPubKey()
	if err != nil {
		panic(err)
	}
	// create validator set with single validator
	validator := cmtypes.NewValidator(pubKey, 1)
	valSet := cmtypes.NewValidatorSet([]*cmtypes.Validator{validator})

	// generate genesis account
	senderPrivKey := secp256k1.GenPrivKey()
	acc := authtypes.NewBaseAccount(senderPrivKey.PubKey().Address().Bytes(), senderPrivKey.PubKey(), 0, 0)
	balance := banktypes.Balance{
		Address: acc.GetAddress().String(),
		Coins:   sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(100000000000000))),
	}

	genesisState := NewDefaultGenesisState(codec)
	return genesisStateWithValSet(codec, genesisState, valSet, []authtypes.GenesisAccount{acc}, balance)
}

func genesisStateWithValSet(codec codec.Codec, genesisState simapp.GenesisState,
	valSet *cmtypes.ValidatorSet, genAccs []authtypes.GenesisAccount,
	balances ...banktypes.Balance,
) simapp.GenesisState {
	// set genesis accounts
	authGenesis := authtypes.NewGenesisState(authtypes.DefaultParams(), genAccs)
	genesisState[authtypes.ModuleName] = codec.MustMarshalJSON(authGenesis)

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
	assetsGenesis := assetstypes.NewGenesis(
		assetstypes.DefaultParams(),
		clientChains, []assetstypes.StakingAssetInfo{
			{
				AssetBasicInfo:     assets[0],
				StakingTotalAmount: depositAmount,
			},
		}, depositsByStaker, nil,
	)
	genesisState[assetstypes.ModuleName] = codec.MustMarshalJSON(assetsGenesis)

	// x/oracle initialization
	oracleDefaultParams := oracletypes.DefaultParams()
	oracleDefaultParams.TokenFeeders[1].StartBaseBlock = 1
	oracleGenesis := oracletypes.NewGenesisState(oracleDefaultParams)
	genesisState[oracletypes.ModuleName] = codec.MustMarshalJSON(oracleGenesis)

	// operator registration
	operatorInfos := []operatortypes.OperatorDetail{
		{
			OperatorAddress: operator.String(),
			OperatorInfo: operatortypes.OperatorInfo{
				EarningsAddr:     operator.String(),
				OperatorMetaInfo: "operator1",
				Commission:       stakingtypes.NewCommission(sdk.ZeroDec(), sdk.ZeroDec(), sdk.ZeroDec()),
			},
		},
	}
	operatorGenesis := operatortypes.NewGenesisState(operatorInfos, nil, nil, nil, nil, nil, nil, nil)
	genesisState[operatortypes.ModuleName] = codec.MustMarshalJSON(operatorGenesis)
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
	genesisState[delegationtypes.ModuleName] = codec.MustMarshalJSON(delegationGenesis)

	dogfoodGenesis := dogfoodtypes.NewGenesis(
		dogfoodtypes.DefaultParams(), []dogfoodtypes.GenesisValidator{
			{
				// PublicKey: consensusKeyRecords[0].Chains[0].ConsensusKey,
				Power:     1,
				PublicKey: hexutil.Encode(valSet.Validators[0].PubKey.Bytes()),
			},
		},
		[]dogfoodtypes.EpochToOperatorAddrs{}, []dogfoodtypes.EpochToConsensusAddrs{},
		[]dogfoodtypes.EpochToUndelegationRecordKeys{}, math.NewInt(1),
	)
	genesisState[dogfoodtypes.ModuleName] = codec.MustMarshalJSON(dogfoodGenesis)

	totalSupply := sdk.NewCoins()
	for _, b := range balances {
		// add genesis acc tokens to total supply
		totalSupply = totalSupply.Add(b.Coins...)
	}
	bankGenesis := banktypes.NewGenesisState(
		banktypes.DefaultParams(), balances, totalSupply,
		[]banktypes.Metadata{}, []banktypes.SendEnabled{},
	)
	genesisState[banktypes.ModuleName] = codec.MustMarshalJSON(bankGenesis)

	return genesisState
}
