package testutil

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cosmos/cosmos-sdk/testutil/mock"
	"github.com/ethereum/go-ethereum/common/hexutil"
	keytypes "github.com/imua-xyz/imuachain/types/keys"
	assetskeeper "github.com/imua-xyz/imuachain/x/assets/keeper"

	epochstypes "github.com/imua-xyz/imuachain/x/epochs/types"

	"github.com/imua-xyz/imuachain/cmd/config"
	avstypes "github.com/imua-xyz/imuachain/x/avs/types"

	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	pruningtypes "github.com/cosmos/cosmos-sdk/store/pruning/types"
	"github.com/evmos/evmos/v16/testutil"
	"github.com/stretchr/testify/suite"
	"golang.org/x/exp/rand"

	testutiltx "github.com/imua-xyz/imuachain/testutil/tx"
	oracletypes "github.com/imua-xyz/imuachain/x/oracle/types"

	imuaapp "github.com/imua-xyz/imuachain/app"
	"github.com/imua-xyz/imuachain/utils"
	assetstypes "github.com/imua-xyz/imuachain/x/assets/types"
	distributiontypes "github.com/imua-xyz/imuachain/x/feedistribution/types"

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
	delegationtypes "github.com/imua-xyz/imuachain/x/delegation/types"
	dogfoodtypes "github.com/imua-xyz/imuachain/x/dogfood/types"
	operatorkeeper "github.com/imua-xyz/imuachain/x/operator/keeper"
	operatortypes "github.com/imua-xyz/imuachain/x/operator/types"
)

type BaseTestSuite struct {
	suite.Suite

	Ctx            sdk.Context
	App            *imuaapp.ImuachainApp
	Address        common.Address
	AccAddress     sdk.AccAddress
	StakerAddr     string
	DogfoodAVSAddr string

	PrivKey   cryptotypes.PrivKey
	Signer    keyring.Signer
	EthSigner ethtypes.Signer

	// construct genesis state from this info
	// x/assets
	ClientChains []assetstypes.ClientChainInfo
	Assets       []assetstypes.AssetInfo
	AssetIDs     []string
	StakerIDs    []string
	// for tracking validator across blocks
	ValSet     *tmtypes.ValidatorSet
	Operators  []sdk.AccAddress
	Powers     []int64
	TotalPower int64

	StateDB        *statedb.StateDB
	QueryClientEVM evmtypes.QueryClient

	InitTime          time.Time
	OperatorMsgServer operatortypes.MsgServer

	// tests may use this to allocate a genesis balance
	Balances []banktypes.Balance
}

var EpochsForTest = []string{
	epochstypes.MinuteEpochID,
	epochstypes.HourEpochID,
	epochstypes.DayEpochID,
	epochstypes.WeekEpochID,
}

var DefaultTestClientChain = []assetstypes.ClientChainInfo{
	{
		Name:               "ethereum",
		MetaInfo:           "ethereum blockchain",
		ChainId:            1,
		FinalizationBlocks: 10,
		LayerZeroChainID:   101,
		AddressLength:      20,
	},
	// add the imua chain
	{
		Name:               "Imua",
		MetaInfo:           "The (native) Imua chain",
		ChainId:            0,
		FinalizationBlocks: 10,
		LayerZeroChainID:   0,
		AddressLength:      20,
	},
}

var DefaultTestStakingAssets = []assetstypes.AssetInfo{
	{
		Name:             "Tether USD",
		Symbol:           "USDT",
		Address:          "0xdAC17F958D2ee523a2206206994597C13D831ec7",
		Decimals:         6,
		LayerZeroChainID: 101,
		MetaInfo:         "Tether USD token",
	},
	// add the imua token
	{
		Name:             "Native IM token",
		Symbol:           "IM",
		Address:          "0x0000000000000000000000000000000000000000",
		Decimals:         18,
		LayerZeroChainID: 0,
		MetaInfo:         "IM native to the Imua chain",
	},
}

var DefaultIMRewardAsset = assetstypes.AssetInfo{
	// add the imua token
	Name: "Native IM token",
	// using the base denomination as the symbol.
	Symbol:  utils.BaseDenom,
	Address: "0x0000000000000000000000000000000000000000",
	// Decimals should be set to 0 since the symbol represents the minimum denomination.
	Decimals:         0,
	LayerZeroChainID: 0,
	MetaInfo:         "IM native to the Imua chain",
}

var (
	DefaultUnbondingPeriod    = uint64(5)
	DefaultOperatorCommission = stakingtypes.NewCommission(sdk.MustNewDecFromStr("0.1"), sdk.NewDec(1), sdk.MustNewDecFromStr("0.1"))
	DefaultDepositAmount      = int64(200)
	DefaultDelegateAmount     = int64(100)
	DefaultFaucetAmount       = int64(100000000)
	TestBlockNumberPerEpoch   = int64(3)
)

func (suite *BaseTestSuite) SetupTest() {
	suite.DoSetupTest()
}

// SetupWithGenesisValSet initializes a new ImuachainApp with a validator set and genesis accounts
// that also act as delegators.
func (suite *BaseTestSuite) SetupWithGenesisValSet(genAccs []authtypes.GenesisAccount, balances ...banktypes.Balance) {
	pruneOpts := pruningtypes.NewPruningOptionsFromString(pruningtypes.PruningOptionDefault)
	appI, genesisState := imuaapp.SetupTestingApp(utils.DefaultChainID, &pruneOpts, false)()
	app, ok := appI.(*imuaapp.ImuachainApp)
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
	suite.StakerIDs = []string{stakerID1, stakerID2}
	_, assetID := assetstypes.GetStakerIDAndAssetIDFromStr(
		suite.ClientChains[0].LayerZeroChainID,
		"", suite.Assets[0].Address,
	)
	suite.AssetIDs = []string{assetID}
	// x/assets initialization - deposits (client chains and tokens are from caller)
	power := int64(101)
	power2 := int64(100)
	suite.Powers = []int64{power, power2}
	suite.TotalPower = power + power2
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
			{
				AssetBasicInfo:     suite.Assets[1],
				StakingTotalAmount: sdk.NewInt(0),
			},
		}, depositsByStaker, operatorAssets,
	)
	genesisState[assetstypes.ModuleName] = app.AppCodec().MustMarshalJSON(assetsGenesis)

	// x/oracle initialization
	oracleDefaultParams := oracletypes.DefaultParams()
	oracleDefaultParams.Chains = append(oracleDefaultParams.Chains, &oracletypes.Chain{Name: "Ethereum", Desc: "-"})
	oracleDefaultParams.Tokens = append(oracleDefaultParams.Tokens, &oracletypes.Token{
		Name:            "ETH",
		ChainID:         1,
		ContractAddress: "0x",
		Decimal:         18,
		Active:          true,
		AssetID:         "0xdac17f958d2ee523a2206206994597c13d831ec7_0x65",
	},
	)
	oracleDefaultParams.Sources = append(oracleDefaultParams.Sources, &oracletypes.Source{
		Name: "Chainlink",
		Entry: &oracletypes.Endpoint{
			Offchain: map[uint64]string{0: ""},
		},
		Valid:         true,
		Deterministic: true,
	})
	oracleDefaultParams.Rules = append(oracleDefaultParams.Rules, &oracletypes.RuleSource{
		// all sources math
		SourceIDs: []uint64{0},
	})
	oracleDefaultParams.TokenFeeders = append(oracleDefaultParams.TokenFeeders, &oracletypes.TokenFeeder{
		TokenID:        1,
		RuleID:         1,
		StartRoundID:   1,
		StartBaseBlock: 1,
		Interval:       10,
	})
	oracleDefaultParams.Tokens = append(oracleDefaultParams.Tokens, &oracletypes.Token{
		Name:            "USDT",
		ChainID:         1,
		ContractAddress: "0x",
		Decimal:         0,
		Active:          true,
		AssetID:         "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48_0x65",
	},
		&oracletypes.Token{
			Name:            "NSTETH",
			ChainID:         1,
			ContractAddress: "0x",
			Decimal:         0,
			Active:          true,
			AssetID:         "nst_0x65",
		},
	)
	oracleDefaultParams.TokenFeeders = append(oracleDefaultParams.TokenFeeders, &oracletypes.TokenFeeder{
		TokenID:        1,
		RuleID:         1,
		StartRoundID:   1,
		StartBaseBlock: 1,
		Interval:       10,
	},
		&oracletypes.TokenFeeder{
			TokenID:        2,
			RuleID:         1,
			StartRoundID:   1,
			StartBaseBlock: 1,
			Interval:       10,
		},
		&oracletypes.TokenFeeder{
			TokenID:        3,
			RuleID:         1,
			StartRoundID:   1,
			StartBaseBlock: 1,
			Interval:       10,
		},
	)
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
				ApproveAddr:      operator1.String(),
				Commission:       stakingtypes.NewCommission(sdk.ZeroDec(), sdk.ZeroDec(), sdk.ZeroDec()),
			},
		},
		{
			OperatorAddress: operator2.String(),
			OperatorInfo: operatortypes.OperatorInfo{
				EarningsAddr:     operator2.String(),
				OperatorMetaInfo: "operator2",
				ApproveAddr:      operator2.String(),
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
	avsAddr := avstypes.GenerateAVSAddress(chainIDWithoutRevision)
	suite.DogfoodAVSAddr = avsAddr
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
	operatorAssetUSDValues := []operatortypes.OperatorAssetUSDValue{
		{
			Key: string(assetstypes.GetJoinedStoreKey(dogfoodtypes.DefaultEpochIdentifier, operator1.String(), assetID)),
			Value: operatortypes.DecValueField{
				Amount: usdValue,
			},
		},
		{
			Key: string(assetstypes.GetJoinedStoreKey(dogfoodtypes.DefaultEpochIdentifier, operator2.String(), assetID)),
			Value: operatortypes.DecValueField{
				Amount: usdValue2,
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
	operatorGenesis := operatortypes.NewGenesisState(operatorInfos, operatorConsKeys, optStates, operatorUSDValues, avsUSDValues, nil, nil, nil, operatorAssetUSDValues)
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
			StakerId: stakerID1,
		},
		{
			Operator: operator2.String(),
			StakerId: stakerID2,
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
	delegationGenesis := delegationtypes.NewGenesis(delegationtypes.DefaultParams(), associations, delegationStates, stakersByOperator, nil)
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
	dogfoodGenesis.Params.AssetIDs = []string{assetID}
	dogfoodGenesis.Params.MinSelfDelegation = math.NewInt(100)
	genesisState[dogfoodtypes.ModuleName] = app.AppCodec().MustMarshalJSON(dogfoodGenesis)

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

	// x/distribution
	distributionGenesis := distributiontypes.NewGenesisState(
		distributiontypes.DefaultParams(),
	)
	distributionGenesis.AllAvsRewardAssets = []distributiontypes.AVSAddrAndRewardAssets{
		{
			Avs: avsAddr,
			AvsRewardAssets: []distributiontypes.AVSRewardAsset{
				{
					AssetBasicInfo: DefaultIMRewardAsset,
				},
			},
		},
	}
	genesisState[distributiontypes.ModuleName] = app.AppCodec().MustMarshalJSON(distributionGenesis)

	stateBytes, err := json.MarshalIndent(genesisState, "", " ")
	suite.Require().NoError(err)

	// init chain will set the validator set and initialize the genesis accounts
	suite.InitTime = time.Now().UTC()
	app.InitChain(abci.RequestInitChain{
		Time:            suite.InitTime,
		ChainId:         utils.DefaultChainID,
		Validators:      []abci.ValidatorUpdate{},
		ConsensusParams: imuaapp.DefaultConsensusParams,
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
	// Force config initialization at the start of each test
	cfg := sdk.GetConfig()
	config.SetBech32Prefixes(cfg)
	config.SetBip44CoinType(cfg)
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
	amount := sdk.TokensFromConsensusPower(DefaultFaucetAmount, evmostypes.PowerReduction)
	balance := banktypes.Balance{
		Address: acc.GetAddress().String(),
		Coins:   sdk.NewCoins(sdk.NewCoin(utils.BaseDenom, amount)),
	}
	// Imuachain modules genesis
	// x/assets
	// make defensive copies to avoid mutating global test defaults
	suite.ClientChains = append([]assetstypes.ClientChainInfo(nil), DefaultTestClientChain...)
	suite.Assets = append([]assetstypes.AssetInfo(nil), DefaultTestStakingAssets...)

	// Initialize an ImuachainApp for test
	suite.SetupWithGenesisValSet(
		[]authtypes.GenesisAccount{acc}, append(suite.Balances, balance)...,
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

func (suite *BaseTestSuite) RunToEpochEnd(epochIdentifier string) {
	var epochDuration time.Duration
	switch epochIdentifier {
	case epochstypes.MinuteEpochID:
		epochDuration = time.Minute
	case epochstypes.HourEpochID:
		epochDuration = time.Hour
	case epochstypes.DayEpochID:
		epochDuration = 24 * time.Hour
	case epochstypes.WeekEpochID:
		epochDuration = 7 * 24 * time.Hour
	default:
		suite.Failf("invalid epoch identifier: %s", epochIdentifier)
		return
	}
	// Configure 3 blocks per epoch for testing, so the block duration is epochDuration/3
	// so starting from the initial block of the epoch, it takes three blocks to
	// reach the epoch’s end block.
	blockDuration := epochDuration / time.Duration(TestBlockNumberPerEpoch)
	for i := int64(0); i < TestBlockNumberPerEpoch; i++ {
		suite.CommitAfter(blockDuration)
	}
	// Run EndBlocker to trigger endBlock execution, which updates voting power in the dogfood module.
	// Note: the above CommitAfter only executes beginBlock when running to the last block of the target epoch.
	suite.App.EndBlocker(suite.Ctx, abci.RequestEndBlock{
		Height: suite.Ctx.BlockHeight(),
	})
}

func (suite *BaseTestSuite) RunToEpochEndN(epochIdentifier string, number int) {
	for i := 0; i < number; i++ {
		suite.RunToEpochEnd(epochIdentifier)
	}
}

func (suite *BaseTestSuite) DebugPrintObject(object interface{}) {
	bytes, err := json.MarshalIndent(object, " ", " ")
	suite.Require().NoError(err)
	fmt.Println(string(bytes))
}

func (suite *BaseTestSuite) RegisterOperator(operator string, commission stakingtypes.Commission) {
	// register operator
	registerReq := &operatortypes.RegisterOperatorReq{
		FromAddress: operator,
		Info: &operatortypes.OperatorInfo{
			EarningsAddr: operator,
			ApproveAddr:  operator,
			Commission:   commission,
		},
	}
	_, err := suite.OperatorMsgServer.RegisterOperator(suite.Ctx, registerReq)
	suite.Require().NoError(err)
}

func (suite *BaseTestSuite) Deposit(clientChainLzID uint64, stakerAddr, assetAddr common.Address, amount math.Int) (string, string) {
	stakerID, assetID := assetstypes.GetStakerIDAndAssetID(clientChainLzID, stakerAddr[:], assetAddr[:])
	// staking assets
	depositParam := &assetskeeper.DepositWithdrawParams{
		ClientChainLzID: clientChainLzID,
		Action:          assetstypes.DepositLST,
		StakerAddress:   stakerAddr[:],
		OpAmount:        amount,
		AssetsAddress:   assetAddr[:],
	}
	_, err := suite.App.AssetsKeeper.PerformDepositOrWithdraw(suite.Ctx, depositParam)
	suite.Require().NoError(err)
	return stakerID, assetID
}

func (suite *BaseTestSuite) Delegation(isDelegation bool, clientChainLzID uint64, staker, assetAddr common.Address, operator sdk.AccAddress, amount math.Int) {
	param := &delegationtypes.DelegationOrUndelegationParams{
		ClientChainID:   clientChainLzID,
		AssetsAddress:   assetAddr[:],
		OperatorAddress: operator,
		StakerAddress:   staker[:],
		OpAmount:        amount,
		TxHash:          common.HexToHash("0x24c4a315d757249c12a7a1d7b6fb96261d49deee26f06a3e1787d008b445c3ac"),
	}
	var err error
	if isDelegation {
		err = suite.App.DelegationKeeper.DelegateTo(suite.Ctx, param)
	} else {
		err = suite.App.DelegationKeeper.UndelegateFrom(suite.Ctx, param)
	}
	suite.Require().NoError(err)
}

func (suite *BaseTestSuite) RegisterAvs(_ string, avsAddr common.Address, assetIDs []string, epochIdentifier string, unbondingPeriod uint64) {
	err := suite.App.AVSManagerKeeper.UpdateAVSInfo(suite.Ctx, &avstypes.AVSRegisterOrDeregisterParams{
		Action:          avstypes.RegisterAction,
		EpochIdentifier: epochIdentifier,
		AvsAddress:      avsAddr,
		AssetIDs:        assetIDs,
		UnbondingPeriod: unbondingPeriod,
	})
	suite.Require().NoError(err)
}

func (suite *BaseTestSuite) RegisterAVSs(number int, epochIdentifier string) []common.Address {
	avsNamePrefix := "testAVS"
	// add the imua token in the assets list
	_, imuaAssetID := assetstypes.GetStakerIDAndAssetIDFromStr(
		suite.ClientChains[1].LayerZeroChainID,
		"", suite.Assets[1].Address,
	)
	suite.AssetIDs = []string{suite.AssetIDs[0], imuaAssetID}
	avsList := make([]common.Address, 0)
	for i := 0; i < number; i++ {
		addr, _ := testutiltx.NewAddrKey()
		name := fmt.Sprintf("%s%d", avsNamePrefix, i)
		usedEpochIdentifier := epochIdentifier
		if epochIdentifier == "" {
			usedEpochIdentifier = EpochsForTest[i%len(EpochsForTest)]
		}
		suite.RegisterAvs(name, addr, suite.AssetIDs, usedEpochIdentifier, DefaultUnbondingPeriod)
		avsList = append(avsList, addr)
	}
	return avsList
}

func (suite *BaseTestSuite) RegisterOperators(number int) []sdk.AccAddress {
	operators := make([]sdk.AccAddress, 0)
	for i := 0; i < number; i++ {
		addr, _ := testutiltx.NewAccAddressAndKey()
		suite.RegisterOperator(addr.String(), DefaultOperatorCommission)
		operators = append(operators, addr)
	}
	return operators
}

func (suite *BaseTestSuite) OptIntoAVSs(operators []sdk.AccAddress, avsList []common.Address) {
	for _, operator := range operators {
		for _, avs := range avsList {
			err := suite.App.OperatorKeeper.OptIn(suite.Ctx, operator, strings.ToLower(avs.String()))
			suite.Require().NoError(err)
		}
	}
}

func (suite *BaseTestSuite) OptIntoDogfood(operators []sdk.AccAddress) {
	for _, operator := range operators {
		privVal := mock.NewPV()
		pubKey, err := privVal.GetPubKey()
		suite.Require().NoError(err)
		consensusKey := keytypes.NewWrappedConsKeyFromHex(hexutil.Encode(pubKey.Bytes()))
		err = suite.App.OperatorKeeper.OptInWithConsKey(suite.Ctx, operator, suite.DogfoodAVSAddr, consensusKey)
		suite.Require().NoError(err)
	}
}

func (suite *BaseTestSuite) CreateStakers(number int, clientChainLzID uint64) ([]common.Address, []string) {
	stakerAddrs := make([]common.Address, 0)
	stakerIDs := make([]string, 0)
	for i := 0; i < number; i++ {
		addr, _ := testutiltx.NewAddrKey()
		stakerID, _ := assetstypes.GetStakerIDAndAssetID(clientChainLzID, addr[:], nil)
		stakerAddrs = append(stakerAddrs, addr)
		stakerIDs = append(stakerIDs, stakerID)
	}
	return stakerAddrs, stakerIDs
}

func (suite *BaseTestSuite) RegisterAssets(number int, decimal uint32) ([]common.Address, []string) {
	assetAddrs := make([]common.Address, 0)
	assetIDs := make([]string, 0)
	clientChainLzID := suite.ClientChains[0].LayerZeroChainID
	for i := 0; i < number; i++ {
		name := fmt.Sprintf("testAsset%d", i)
		symbol := fmt.Sprintf("test%d", i)
		addr, _ := testutiltx.NewAddrKey()
		_, assetID := assetstypes.GetStakerIDAndAssetID(clientChainLzID, nil, addr[:])
		err := suite.App.AssetsKeeper.RegisterNewTokenAndSetTokenFeeder(suite.Ctx, &oracletypes.OracleInfo{
			Chain: struct {
				Name string
				Desc string
			}{Name: "Ethereum", Desc: "-"},
			Token: struct {
				Name     string `json:"name"`
				Desc     string
				Decimal  string `json:"decimal"`
				Contract string `json:"contract"`
				AssetID  string `json:"asset_id"`
			}{
				Name:     name,
				Desc:     "_",
				Contract: "0x",
				Decimal:  "18",
				AssetID:  assetID,
			},
			AssetID: assetID,
		})
		suite.Require().NoError(err)
		err = suite.App.AssetsKeeper.SetStakingAssetInfo(suite.Ctx, &assetstypes.StakingAssetInfo{
			AssetBasicInfo: assetstypes.AssetInfo{
				Name:             name,
				Symbol:           symbol,
				Address:          strings.ToLower(addr.Hex()),
				Decimals:         decimal,
				LayerZeroChainID: clientChainLzID,
			},
			StakingTotalAmount: sdk.ZeroInt(),
		})
		suite.Require().NoError(err)
		assetAddrs = append(assetAddrs, addr)
		assetIDs = append(assetIDs, assetID)
	}
	return assetAddrs, assetIDs
}

func (suite *BaseTestSuite) DepositAndDelegateToOperators(
	isAssociate bool, clientChainLzID uint64,
	assetAddr common.Address, assetDecimal uint32,
	stakerAddrs []common.Address, operators []sdk.AccAddress, depositAmount, delegateAmount int64,
) {
	multiplier := math.NewIntWithDecimal(1, int(assetDecimal)) // 10^decimals
	depositAmountBigInt := sdk.NewInt(depositAmount).Mul(multiplier)
	delegationAmountBigInt := sdk.NewInt(delegateAmount).Mul(multiplier)

	for i, stakerAddr := range stakerAddrs {
		for j, operator := range operators {
			// Associate the staker and operator at the same index to satisfy the self-delegation requirement during opt-in.
			if isAssociate && i == j {
				err := suite.App.DelegationKeeper.AssociateOperatorWithStaker(suite.Ctx, clientChainLzID, operator, stakerAddr[:])
				suite.Require().NoError(err)
			}
			suite.Deposit(clientChainLzID, stakerAddr, assetAddr, depositAmountBigInt)
			suite.Delegation(true, clientChainLzID, stakerAddr, assetAddr, operator, delegationAmountBigInt)
		}
	}
}

func (suite *BaseTestSuite) DepositAndDelegateIMUAToOperators(stakerAddrs []common.Address, operators []sdk.AccAddress, depositAmount, delegateAmount int64) {
	assetAddr := common.HexToAddress(suite.Assets[1].Address)
	multiplier := math.NewIntWithDecimal(1, int(suite.Assets[1].Decimals)) // 10^decimals
	depositAmountBigInt := sdk.NewInt(depositAmount).Mul(multiplier)
	delegationAmountBigInt := sdk.NewInt(delegateAmount).Mul(multiplier)

	for _, stakerAddr := range stakerAddrs {
		for _, operator := range operators {
			coins := sdk.NewCoins(sdk.NewCoin(utils.BaseDenom, depositAmountBigInt))
			err := suite.App.BankKeeper.SendCoins(suite.Ctx, suite.Address[:], stakerAddr[:], coins)
			suite.Require().NoError(err)
			suite.Delegation(true, 0, stakerAddr, assetAddr, operator, delegationAmountBigInt)
		}
	}
}
