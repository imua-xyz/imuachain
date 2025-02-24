package testdata

import (
	"time"

	sdkmath "cosmossdk.io/math"
	"github.com/imua-xyz/imuachain/x/oracle/types"
)

func DefaultParamsForTest() types.Params {
	return types.Params{
		Chains: []*types.Chain{
			{Name: "-", Desc: "-"},
			{Name: "Ethereum", Desc: "-"},
		},
		Tokens: []*types.Token{
			{},
			{
				Name:            "ETH",
				ChainID:         1,
				ContractAddress: "0x",
				Decimal:         8,
				Active:          true,
				AssetID:         "0x0b34c4d876cd569129cf56bafabb3f9e97a4ff42_0x9ce1",
			},
		},
		// source defines where to fetch the prices
		Sources: []*types.Source{
			{
				Name: "0 position is reserved",
			},
			{
				Name: "Chainlink",
				Entry: &types.Endpoint{
					Offchain: map[uint64]string{0: ""},
				},
				Valid:         true,
				Deterministic: true,
			},
		},
		// rules defines price from which sources are accepted, could be used to proof malicious
		Rules: []*types.RuleSource{
			// 0 is reserved
			{},
			{
				// all sources math
				SourceIDs: []uint64{0},
			},
		},
		// TokenFeeder describes when a token start to be updated with its price, and the frequency, endTime.
		TokenFeeders: []*types.TokenFeeder{
			{},
			{
				TokenID:        1,
				RuleID:         1,
				StartRoundID:   1,
				StartBaseBlock: 20,
				Interval:       10,
			},
		},
		MaxNonce:   3,
		ThresholdA: 2,
		ThresholdB: 3,
		// V1 set mode to 1
		Mode:          types.ConsensusModeASAP,
		MaxDetId:      5,
		MaxSizePrices: 100,
		Slashing: &types.SlashingParams{
			ReportedRoundsWindow:        100,
			MinReportedPerWindow:        sdkmath.LegacyNewDec(1).Quo(sdkmath.LegacyNewDec(2)),
			OracleMissJailDuration:      600 * time.Second,
			OracleMaliciousJailDuration: 30 * 24 * time.Hour,
			SlashFractionMalicious:      sdkmath.LegacyNewDec(1).Quo(sdkmath.LegacyNewDec(10)),
		},
	}
}
