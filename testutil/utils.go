package testutil

import (
	"encoding/json"
	"time"

	avstypes "github.com/ExocoreNetwork/exocore/x/avs/types"

	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	pruningtypes "github.com/cosmos/cosmos-sdk/store/pruning/types"
	"github.com/evmos/evmos/v16/testutil"
	"github.com/stretchr/testify/suite"
	"golang.org/x/exp/rand"

	testutiltx "github.com/ExocoreNetwork/exocore/testutil/tx"
	oracletypes "github.com/ExocoreNetwork/exocore/x/oracle/types"

	exocoreapp "github.com/ExocoreNetwork/exocore/app"
	"github.com/ExocoreNetwork/exocore/utils"
	assetstypes "github.com/ExocoreNetwork/exocore/x/assets/types"
	distributiontypes "github.com/ExocoreNetwork/exocore/x/feedistribution/types"

	delegationtypes "github.com/ExocoreNetwork/exocore/x/delegation/types"
	dogfoodtypes "github.com/ExocoreNetwork/exocore/x/dogfood/types"
	operatorkeeper "github.com/ExocoreNetwork/exocore/x/operator/keeper"
	operatortypes "github.com/ExocoreNetwork/exocore/x/operator/types"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/crypto/tmhash"
	tmtypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	evmostypes "github.com/evmos/evmos/v16/types"
	"github.com/evmos/evmos/v16/x/evm/statedb"
	evmtypes "github.com/evmos/evmos/v16/x/evm/types"
)

type BaseTestSuite struct {
	suite.Suite

	Ctx        sdk.Context
	App        *exocoreapp.ExocoreApp
	Address    common.Address
	AccAddress sdk.AccAddress
	StakerAddr string

	PrivKey   cryptotypes.PrivKey
	Signer    keyring.Signer
	EthSigner ethtypes.Signer

	// construct genesis state from this info
	// x/assets
	ClientChains []assetstypes.ClientChainInfo
	Assets       []assetstypes.AssetInfo
	AssetIDs     []string
	// for tracking validator across blocks
	ValSet    *tmtypes.ValidatorSet
	Operators []sdk.AccAddress
	Powers    []int64

	StateDB        *statedb.StateDB
	QueryClientEVM evmtypes.QueryClient

	InitTime          time.Time
	OperatorMsgServer operatortypes.MsgServer
}

func (suite *BaseTestSuite) SetupTest() {
	suite.DoSetupTest()
}

// SetupWithGenesisValSet initializes a new ExocoreApp with a validator set and genesis accounts
// that also act as delegators.
func (suite *BaseTestSuite) SetupWithGenesisValSet(genAccs []authtypes.GenesisAccount, balances ...banktypes.Balance) {
	pruneOpts := pruningtypes.NewPruningOptionsFromString(pruningtypes.PruningOptionDefault)
	appI, genesisState := exocoreapp.SetupTestingApp(utils.DefaultChainID, &pruneOpts, false)()
	app, ok := appI.(*exocoreapp.ExocoreApp)
	suite.Require().True(ok)

	// set genesis accounts
	authGenesis := authtypes.NewGenesisState(authtypes.DefaultParams(), genAccs)
	genesisState[authtypes.ModuleName] = app.AppCodec().MustMarshalJSON(authGenesis)

	// x/operator initialization - address only
	operator1 := sdk.AccAddress(testutiltx.GenerateAddress().Bytes())
	operator2 := sdk.AccAddress(testutiltx.GenerateAddress().Bytes())
	suite.Operators = []sdk.AccAddress{operator1, operator2}
	stakerID1, _ := assetstypes.GetStakerIDAndAssetIDFromStr(
		suite.ClientChains[0].LayerZeroChainID,
		common.Address(operator1.Bytes()).String(), "",
	)
	suite.StakerAddr = common.Address(operator1.Bytes()).String()
	stakerID2, _ := assetstypes.GetStakerIDAndAssetIDFromStr(
		suite.ClientChains[0].LayerZeroChainID,
		common.Address(operator2.Bytes()).String(), "",
	)
	_, assetID := assetstypes.GetStakerIDAndAssetIDFromStr(
		suite.ClientChains[0].LayerZeroChainID,
		"", suite.Assets[0].Address,
	)
	suite.AssetIDs = append(suite.AssetIDs, assetID)
	// x/assets initialization - deposits (client chains and tokens are from caller)
	power := int64(101)
	power2 := int64(100)
	suite.Powers = []int64{power, power2}
	depositAmount := math.NewIntWithDecimal(power, 6)
	depositAmount2 := math.NewIntWithDecimal(power2, 6)
	usdValue := math.LegacyNewDec(power)
	usdValue2 := math.LegacyNewDec(power2)
	depositsByStaker := []assetstypes.DepositsByStaker{
		{
			StakerID: stakerID1,
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
		{
			StakerID: stakerID2,
			Deposits: []assetstypes.DepositByAsset{
				{
					AssetID: assetID,
					Info: assetstypes.StakerAssetInfo{
						TotalDepositAmount:        depositAmount2,
						WithdrawableAmount:        depositAmount2,
						PendingUndelegationAmount: sdk.ZeroInt(),
					},
				},
			},
		},
	}
	operatorAssets := []assetstypes.AssetsByOperator{
		{
			Operator: operator1.String(),
			AssetsState: []assetstypes.AssetByID{
				{
					AssetID: assetID,
					Info: assetstypes.OperatorAssetInfo{
						TotalAmount:               depositAmount,
						PendingUndelegationAmount: sdk.ZeroInt(),
						TotalShare:                sdk.NewDecFromBigInt(depositAmount.BigInt()),
						OperatorShare:             sdk.NewDecFromBigInt(depositAmount.BigInt()),
					},
				},
			},
		},
		{
			Operator: operator2.String(),
			AssetsState: []assetstypes.AssetByID{
				{
					AssetID: assetID,
					Info: assetstypes.OperatorAssetInfo{
						TotalAmount:               depositAmount2,
						PendingUndelegationAmount: sdk.ZeroInt(),
						TotalShare:                sdk.NewDecFromBigInt(depositAmount2.BigInt()),
						OperatorShare:             sdk.NewDecFromBigInt(depositAmount2.BigInt()),
					},
				},
			},
		},
	}
	assetsGenesis := assetstypes.NewGenesis(
		assetstypes.DefaultParams(),
		suite.ClientChains, []assetstypes.StakingAssetInfo{
			{
				AssetBasicInfo:     suite.Assets[0],
				StakingTotalAmount: depositAmount.Add(depositAmount2),
			},
			//	{
			//		AssetBasicInfo:     suite.Assets[1],
			//		StakingTotalAmount: depositAmount.Add(math.NewInt(132)),
			//	},
		}, depositsByStaker, operatorAssets,
	)
	genesisState[assetstypes.ModuleName] = app.AppCodec().MustMarshalJSON(assetsGenesis)

	// x/oracle initialization
	oracleDefaultParams := oracletypes.DefaultParams()
	oracleDefaultParams.Tokens[1].AssetID = "0xdac17f958d2ee523a2206206994597c13d831ec7_0x65"
	oracleDefaultParams.TokenFeeders[1].StartBaseBlock = 1
	oracleDefaultParams.Tokens = append(oracleDefaultParams.Tokens, &oracletypes.Token{
		Name:            "USDT",
		ChainID:         1,
		ContractAddress: "0x",
		Decimal:         0,
		Active:          true,
		AssetID:         "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48_0x65",
	})
	oracleDefaultParams.TokenFeeders = append(oracleDefaultParams.TokenFeeders, &oracletypes.TokenFeeder{
		TokenID:        2,
		RuleID:         1,
		StartRoundID:   1,
		StartBaseBlock: 1,
		Interval:       10,
	})
	oracleGenesis := oracletypes.NewGenesisState(oracleDefaultParams)
	oracleGenesis.PricesList = []oracletypes.Prices{
		{TokenID: 1, NextRoundID: 2, PriceList: []*oracletypes.PriceTimeRound{{Price: "1", Decimal: 0, RoundID: 1}}},
		{TokenID: 2, NextRoundID: 2, PriceList: []*oracletypes.PriceTimeRound{{Price: "1", Decimal: 0, RoundID: 1}}},
	}
	genesisState[oracletypes.ModuleName] = app.AppCodec().MustMarshalJSON(oracleGenesis)

	// x/operator registration
	operatorInfos := []operatortypes.OperatorDetail{
		{
			OperatorAddress: operator1.String(),
			OperatorInfo: operatortypes.OperatorInfo{
				EarningsAddr:     operator1.String(),
				OperatorMetaInfo: "operator1",
				Commission:       stakingtypes.NewCommission(sdk.ZeroDec(), sdk.ZeroDec(), sdk.ZeroDec()),
			},
		},
		{
			OperatorAddress: operator2.String(),
			OperatorInfo: operatortypes.OperatorInfo{
				EarningsAddr:     operator2.String(),
				OperatorMetaInfo: "operator2",
				Commission:       stakingtypes.NewCommission(sdk.ZeroDec(), sdk.ZeroDec(), sdk.ZeroDec()),
			},
		},
	}
	// generate validator private/public key
	pubKey := testutiltx.GenerateConsensusKey()
	suite.Require().NotNil(pubKey)
	pubKey2 := testutiltx.GenerateConsensusKey()
	suite.Require().NotNil(pubKey2)
	chainIDWithoutRevision := avstypes.ChainIDWithoutRevision(utils.DefaultChainID)
	operatorConsKeys := []operatortypes.OperatorConsKeyRecord{
		{
			OperatorAddress: operator1.String(),
			Chains: []operatortypes.ChainDetails{
				{
					ChainID:      chainIDWithoutRevision,
					ConsensusKey: pubKey.ToHex(),
				},
			},
		},
		{
			OperatorAddress: operator2.String(),
			Chains: []operatortypes.ChainDetails{
				{
					ChainID:      chainIDWithoutRevision,
					ConsensusKey: pubKey2.ToHex(),
				},
			},
		},
	}
	avsAddr := avstypes.GenerateAVSAddr(chainIDWithoutRevision)
	optStates := []operatortypes.OptedState{
		{
			Key: string(assetstypes.GetJoinedStoreKey(operator1.String(), avsAddr)),
			OptInfo: operatortypes.OptedInfo{
				OptedInHeight:  1,
				OptedOutHeight: operatortypes.DefaultOptedOutHeight,
			},
		},
		{
			Key: string(assetstypes.GetJoinedStoreKey(operator2.String(), avsAddr)),
			OptInfo: operatortypes.OptedInfo{
				OptedInHeight:  1,
				OptedOutHeight: operatortypes.DefaultOptedOutHeight,
			},
		},
	}
	operatorUSDValues := []operatortypes.OperatorUSDValue{
		{
			Key: string(assetstypes.GetJoinedStoreKey(avsAddr, operator1.String())),
			OptedUSDValue: operatortypes.OperatorOptedUSDValue{
				SelfUSDValue:   usdValue,
				TotalUSDValue:  usdValue,
				ActiveUSDValue: usdValue,
			},
		},
		{
			Key: string(assetstypes.GetJoinedStoreKey(avsAddr, operator2.String())),
			OptedUSDValue: operatortypes.OperatorOptedUSDValue{
				SelfUSDValue:   usdValue2,
				TotalUSDValue:  usdValue2,
				ActiveUSDValue: usdValue2,
			},
		},
	}
	avsUSDValues := []operatortypes.AVSUSDValue{
		{
			AVSAddr: avsAddr,
			Value: operatortypes.DecValueField{
				Amount: usdValue.Add(usdValue2),
			},
		},
	}
	operatorGenesis := operatortypes.NewGenesisState(operatorInfos, operatorConsKeys, optStates, operatorUSDValues, avsUSDValues, nil, nil, nil)
	genesisState[operatortypes.ModuleName] = app.AppCodec().MustMarshalJSON(operatorGenesis)

	// x/delegation
	delegationStates := []delegationtypes.DelegationStates{
		{
			Key: string(assetstypes.GetJoinedStoreKey(stakerID1, assetID, operator1.String())),
			States: delegationtypes.DelegationAmounts{
				WaitUndelegationAmount: math.NewInt(0),
				UndelegatableShare:     math.LegacyNewDecFromBigInt(depositAmount.BigInt()),
			},
		},
		{
			Key: string(assetstypes.GetJoinedStoreKey(stakerID2, assetID, operator2.String())),
			States: delegationtypes.DelegationAmounts{
				WaitUndelegationAmount: math.NewInt(0),
				UndelegatableShare:     math.LegacyNewDecFromBigInt(depositAmount2.BigInt()),
			},
		},
	}
	associations := []delegationtypes.StakerToOperator{
		{
			Operator: operator1.String(),
			StakerID: stakerID1,
		},
		{
			Operator: operator2.String(),
			StakerID: stakerID2,
		},
	}
	stakersByOperator := []delegationtypes.StakersByOperator{
		{
			Key: string(assetstypes.GetJoinedStoreKey(operator1.String(), assetID)),
			Stakers: []string{
				stakerID1,
			},
		},
		{
			Key: string(assetstypes.GetJoinedStoreKey(operator2.String(), assetID)),
			Stakers: []string{
				stakerID2,
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
				PublicKey: pubKey.ToHex(),
				Power:     power,
			},
			{
				PublicKey: pubKey2.ToHex(),
				Power:     power2,
			},
		},
		[]dogfoodtypes.EpochToOperatorAddrs{},
		[]dogfoodtypes.EpochToConsensusAddrs{},
		[]dogfoodtypes.EpochToUndelegationRecordKeys{},
		math.NewInt(power+power2), // must match total vote power
	)
	dogfoodGenesis.Params.MinSelfDelegation = math.NewInt(100)
	genesisState[dogfoodtypes.ModuleName] = app.AppCodec().MustMarshalJSON(dogfoodGenesis)
	distributionGenesis := distributiontypes.NewGenesisState(
		distributiontypes.DefaultParams(),
	)
	genesisState[distributiontypes.ModuleName] = app.AppCodec().MustMarshalJSON(distributionGenesis)

	suite.ValSet = tmtypes.NewValidatorSet([]*tmtypes.Validator{
		tmtypes.NewValidator(pubKey.ToTmKey(), 1),
		tmtypes.NewValidator(pubKey2.ToTmKey(), 1),
	})

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

	stateBytes, err := json.MarshalIndent(genesisState, "", " ")
	suite.Require().NoError(err)

	// init chain will set the validator set and initialize the genesis accounts
	suite.InitTime = time.Now().UTC()
	app.InitChain(abci.RequestInitChain{
		Time:            suite.InitTime,
		ChainId:         utils.DefaultChainID,
		Validators:      []abci.ValidatorUpdate{},
		ConsensusParams: exocoreapp.DefaultConsensusParams,
		AppStateBytes:   stateBytes,
	})
	// committing the chain now is not required. doing so will skip the first block.

	// instantiate new header
	header := testutil.NewHeader(
		1,
		suite.InitTime.Add(time.Second),
		utils.DefaultChainID,
		pubKey.ToConsAddr(),
		tmhash.Sum([]byte("App")),
		tmhash.Sum([]byte("Validators")),
	)

	app.BeginBlock(abci.RequestBeginBlock{
		Header: header,
	})

	suite.Ctx = app.BaseApp.NewContext(false, header)
	suite.App = app
	suite.OperatorMsgServer = operatorkeeper.NewMsgServerImpl(app.OperatorKeeper)

	// at this point, we have reached the genesis state and we are in the middle of the first block.
	// BeginBlock of block 1 has been done, and we can process txs.
	// EndBlock is called after that.
}

func (suite *BaseTestSuite) DoSetupTest() {
	// create AccAddress for test
	pubBz := make([]byte, ed25519.PubKeySize)
	pub := &ed25519.PubKey{Key: pubBz}
	_, err := rand.Read(pub.Key)
	suite.Require().NoError(err)
	suite.AccAddress = sdk.AccAddress(pub.Address())

	// generate genesis account
	addr, priv := testutiltx.NewAddrKey()
	suite.PrivKey = priv
	suite.Address = addr
	suite.Signer = testutiltx.NewSigner(priv)
	baseAcc := authtypes.NewBaseAccount(priv.PubKey().Address().Bytes(), priv.PubKey(), 0, 0)
	acc := &evmostypes.EthAccount{
		BaseAccount: baseAcc,
		CodeHash:    common.BytesToHash(evmtypes.EmptyCodeHash).Hex(),
	}
	// set amount for genesis account
	amount := sdk.TokensFromConsensusPower(5, evmostypes.PowerReduction)
	balance := banktypes.Balance{
		Address: acc.GetAddress().String(),
		Coins:   sdk.NewCoins(sdk.NewCoin(utils.BaseDenom, amount)),
	}
	// Exocore modules genesis
	// x/assets
	suite.ClientChains = []assetstypes.ClientChainInfo{
		{
			Name:               "ethereum",
			MetaInfo:           "ethereum blockchain",
			ChainId:            1,
			FinalizationBlocks: 10,
			LayerZeroChainID:   101,
			AddressLength:      20,
		},
	}
	suite.Assets = []assetstypes.AssetInfo{
		{
			Name:             "Tether USD",
			Symbol:           "USDT",
			Address:          "0xdAC17F958D2ee523a2206206994597C13D831ec7",
			Decimals:         6,
			LayerZeroChainID: suite.ClientChains[0].LayerZeroChainID,
			MetaInfo:         "Tether USD token",
		},
	}

	// Initialize an ExocoreApp for test
	suite.SetupWithGenesisValSet(
		[]authtypes.GenesisAccount{acc}, balance,
	)

	// Create StateDB
	suite.StateDB = statedb.New(suite.Ctx, suite.App.EvmKeeper, statedb.NewEmptyTxConfig(common.BytesToHash(suite.Ctx.HeaderHash().Bytes())))

	suite.EthSigner = ethtypes.LatestSignerForChainID(suite.App.EvmKeeper.ChainID())

	queryHelperEvm := baseapp.NewQueryServerTestHelper(suite.Ctx, suite.App.InterfaceRegistry())
	evmtypes.RegisterQueryServer(queryHelperEvm, suite.App.EvmKeeper)
	suite.QueryClientEVM = evmtypes.NewQueryClient(queryHelperEvm)
}

// DeployContract deploys a contract that calls the deposit precompile's methods for testing purposes.
func (suite *BaseTestSuite) DeployContract(contract evmtypes.CompiledContract) (addr common.Address, err error) {
	addr, err = DeployContract(
		suite.Ctx,
		suite.App,
		suite.PrivKey,
		suite.QueryClientEVM,
		contract,
	)
	return
}

// NextBlock commits the current block and sets up the next block at a time t + 1 second.
func (suite *BaseTestSuite) NextBlock() {
	suite.CommitAfter(time.Second)
}

// Commit commits the current block and sets up the next block at a time t + 1 nanosecond.
func (suite *BaseTestSuite) Commit() {
	suite.CommitAfter(time.Nanosecond)
}

// CommitAfter commits the current block and sets up the next block at a time t + d.
func (suite *BaseTestSuite) CommitAfter(d time.Duration) {
	var err error
	// do not use an uncached ctx here
	suite.Ctx, err = CommitAndCreateNewCtx(suite.Ctx, suite.App, d, nil, false)
	suite.Require().NoError(err)
}
