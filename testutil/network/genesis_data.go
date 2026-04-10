package network

import (
	"fmt"
	"os"
	"time"

	sdkmath "cosmossdk.io/math"
	assetstypes "github.com/imua-xyz/imuachain/x/assets/types"
	delegationtypes "github.com/imua-xyz/imuachain/x/delegation/types"
	dogfoodtypes "github.com/imua-xyz/imuachain/x/dogfood/types"
	operatortypes "github.com/imua-xyz/imuachain/x/operator/types"
	oracletypes "github.com/imua-xyz/imuachain/x/oracle/types"
)

const (
	// TestEVMChainID represents the LayerZero chain ID for the test EVM chain
	TestEVMChainID = 101
	// EVMAddressLength is the standard length of EVM addresses in bytes
	EVMAddressLength   = 20
	NativeAssetAddress = "0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
	ETHAssetAddress    = "0xdac17f958d2ee523a2206206994597c13d831ec7"
	NativeAssetID      = "0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee_0x65"
	ETHAssetID         = "0xdac17f958d2ee523a2206206994597c13d831ec7_0x65"
)

var (
	// DefaultGenStateAssets only includes two assets, one for ETH and the other for NST ETH
	// For the contract address of asset-ETH we filled with the address of USDT, that's ok for test
	// we bond both tokens to the price of ETH in oracle module
	DefaultGenStateAssets = assetstypes.GenesisState{
		Params: assetstypes.Params{
			Gateways: []string{"0x3e108c058e8066da635321dc3018294ca82ddedf"},
		},
		ClientChains: []assetstypes.ClientChainInfo{
			{
				Name:             "Example EVM chain",
				MetaInfo:         "Example EVM chain metaInfo",
				LayerZeroChainID: TestEVMChainID,
				AddressLength:    EVMAddressLength,
			},
		},
		Tokens: []assetstypes.StakingAssetInfo{
			NewTestToken("ETH", "Ethereum native token", ETHAssetAddress, TestEVMChainID, 0, 5000),
			NewTestToken("NST ETH", "native restaking ETH", NativeAssetAddress, TestEVMChainID, 0, 5000),
		},
	}

	DefaultGenStateOperator = operatortypes.GenesisState{}

	DefaultGenStateDelegation = delegationtypes.GenesisState{}

	DefaultGenStateDogfood = *dogfoodtypes.DefaultGenesis()

	DefaultGenStateOracle = *oracletypes.DefaultGenesis()
)

func init() {
	DefaultGenStateOracle.Params.Chains = append(DefaultGenStateOracle.Params.Chains, &oracletypes.Chain{Name: "Ethereum", Desc: "-"})
	DefaultGenStateOracle.Params.Tokens = append(DefaultGenStateOracle.Params.Tokens, &oracletypes.Token{
		Name:            "ETH",
		ChainID:         1,
		ContractAddress: "0x",
		Decimal:         8,
		Active:          true,
		// bond assetsIDs of ETH, NSTETH to ETH price
		AssetID: fmt.Sprintf("%s,%s", ETHAssetID, NativeAssetID),
	})
	DefaultGenStateOracle.Params.TokenFeeders = append(DefaultGenStateOracle.Params.TokenFeeders, &oracletypes.TokenFeeder{
		TokenID:      1,
		RuleID:       2,
		StartRoundID: 1,
		// set ETH tokenfeeder's 'StartBaseBlock' to 10
		StartBaseBlock: 10,
		Interval:       10,
	})
	// set NSTETH token and tokenFeeder
	DefaultGenStateOracle.Params.Tokens = append(DefaultGenStateOracle.Params.Tokens, &oracletypes.Token{
		Name:            "NSTETH",
		ChainID:         1,
		ContractAddress: "0x",
		Decimal:         0,
		Active:          true,
		AssetID:         "NST_0x65",
	})
	DefaultGenStateOracle.Params.TokenFeeders = append(DefaultGenStateOracle.Params.TokenFeeders, &oracletypes.TokenFeeder{
		TokenID:        2,
		RuleID:         3,
		StartRoundID:   1,
		StartBaseBlock: 7,
		Interval:       10,
	})
	// set slashing_miss window to 4
	DefaultGenStateOracle.Params.Slashing.ReportedRoundsWindow = 4
	// set jailduration of oracle report downtime to 15 seconds for test
	DefaultGenStateOracle.Params.Slashing.OracleMissJailDuration = 15 * time.Second
	// set initial prices to avoid zero voting power at epoch end
	DefaultGenStateOracle.PricesList = []oracletypes.Prices{
		{
			TokenID:     1,
			NextRoundID: 2,
			PriceList: []*oracletypes.PriceTimeRound{
				{Price: "1", Decimal: 0, RoundID: 1},
			},
		},
		{
			TokenID:     2,
			NextRoundID: 2,
			PriceList: []*oracletypes.PriceTimeRound{
				{Price: "1", Decimal: 0, RoundID: 1},
			},
		},
	}
	switch os.Getenv("TEST_OPTION") {
	case "nst-malicious":
		fallthrough
	case "nst":
		DefaultGenStateOracle.Params.PieceSizeByte = 32
		DefaultGenStateOracle.Params.TokenFeeders[2].Interval = 25
	default:
	}
	//	if os.Getenv("TEST_OPTION") == "nst" {
	//		DefaultGenStateOracle.Params.PieceSizeByte = 32
	//		DefaultGenStateOracle.Params.TokenFeeders[2].Interval = 25
	//	}
}

// DefaultOracleGenesisForE2E returns a fresh default oracle genesis with ETH + NST feeders (same as after init()).
// Used by initGenFiles when no override is set, so e2e tests get a consistent 3-feeder genesis without reading
// the possibly mutated DefaultGenStateOracle.
func DefaultOracleGenesisForE2E() oracletypes.GenesisState {
	gs := *oracletypes.DefaultGenesis()
	gs.Params.Chains = append(gs.Params.Chains, &oracletypes.Chain{Name: "Ethereum", Desc: "-"})
	gs.Params.Tokens = append(gs.Params.Tokens, &oracletypes.Token{
		Name:            "ETH",
		ChainID:         1,
		ContractAddress: "0x",
		Decimal:         8,
		Active:          true,
		AssetID:         fmt.Sprintf("%s,%s", ETHAssetID, NativeAssetID),
	})
	gs.Params.TokenFeeders = append(gs.Params.TokenFeeders, &oracletypes.TokenFeeder{
		TokenID:        1,
		RuleID:         2,
		StartRoundID:   1,
		StartBaseBlock: 10,
		Interval:       10,
	})
	gs.Params.Tokens = append(gs.Params.Tokens, &oracletypes.Token{
		Name:            "NSTETH",
		ChainID:         1,
		ContractAddress: "0x",
		Decimal:         0,
		Active:          true,
		AssetID:         "NST_0x65",
	})
	gs.Params.TokenFeeders = append(gs.Params.TokenFeeders, &oracletypes.TokenFeeder{
		TokenID:        2,
		RuleID:         3,
		StartRoundID:   1,
		StartBaseBlock: 7,
		Interval:       10,
	})
	gs.Params.Slashing.ReportedRoundsWindow = 4
	gs.Params.Slashing.OracleMissJailDuration = 15 * time.Second
	gs.PricesList = []oracletypes.Prices{
		{TokenID: 1, NextRoundID: 1, PriceList: []*oracletypes.PriceTimeRound{{Price: "1", Decimal: 0, RoundID: 0}}},
		//		{TokenID: 2, NextRoundID: 2, PriceList: []*oracletypes.PriceTimeRound{{Price: "1", Decimal: 0, RoundID: 1}}},
	}
	switch os.Getenv("TEST_OPTION") {
	case "nst-malicious", "nst":
		gs.Params.PieceSizeByte = 32
		gs.Params.TokenFeeders[2].Interval = 25
	}
	return gs
}

func NewTestToken(name, metaInfo, address string, chainID uint64, decimal uint32, amount int64) assetstypes.StakingAssetInfo {
	if name == "" {
		panic("token name cannot be empty")
	}
	if amount <= 0 {
		panic("staking amount must be positive")
	}
	return assetstypes.StakingAssetInfo{
		AssetBasicInfo: assetstypes.AssetInfo{
			Name:             name,
			MetaInfo:         metaInfo,
			Decimals:         decimal,
			Address:          address,
			LayerZeroChainID: chainID,
		},
		StakingTotalAmount: sdkmath.NewInt(amount),
	}
}
