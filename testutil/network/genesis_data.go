package network

import (
	"fmt"
	"time"

	sdkmath "cosmossdk.io/math"
	assetstypes "github.com/ExocoreNetwork/exocore/x/assets/types"
	delegationtypes "github.com/ExocoreNetwork/exocore/x/delegation/types"
	dogfoodtypes "github.com/ExocoreNetwork/exocore/x/dogfood/types"
	operatortypes "github.com/ExocoreNetwork/exocore/x/operator/types"
	oracletypes "github.com/ExocoreNetwork/exocore/x/oracle/types"
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
	// DefaultGenStateOperator = *operatortypes.DefaultGenesis()

	DefaultGenStateDelegation = delegationtypes.GenesisState{}
	// DefaultGenStateDelegation = *delegationtypes.DefaultGenesis()

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
	DefaultGenStateOracle.Params.Sources = append(DefaultGenStateOracle.Params.Sources, &oracletypes.Source{
		Name: "Chainlink",
		Entry: &oracletypes.Endpoint{
			Offchain: map[uint64]string{0: ""},
		},
		Valid:         true,
		Deterministic: true,
	})
	DefaultGenStateOracle.Params.Rules = append(DefaultGenStateOracle.Params.Rules, &oracletypes.RuleSource{
		// all sources math
		SourceIDs: []uint64{0},
	})
	DefaultGenStateOracle.Params.TokenFeeders = append(DefaultGenStateOracle.Params.TokenFeeders, &oracletypes.TokenFeeder{
		TokenID:      1,
		RuleID:       1,
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
		RuleID:         1,
		StartRoundID:   1,
		StartBaseBlock: 7,
		Interval:       10,
	})
	// set slashing_miss window to 4
	DefaultGenStateOracle.Params.Slashing.ReportedRoundsWindow = 4
	// set jailduration of oracle report downtime to 15 seconds for test
	DefaultGenStateOracle.Params.Slashing.OracleMissJailDuration = 15 * time.Second
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
