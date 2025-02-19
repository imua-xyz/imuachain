package aggregator

import (
	"math/big"

	"github.com/imua-xyz/imuachain/x/oracle/types"
)

var (
	one     = big.NewInt(1)
	zero    = big.NewInt(0)
	ten     = big.NewInt(10)
	eleven  = big.NewInt(11)
	fifteen = big.NewInt(15)
	twenty  = big.NewInt(20)
)

var (
	pTD1  = newPTD("1", "10")
	pTD2  = newPTD("2", "12")
	pTD3  = newPTD("3", "15")
	pTD2M = newPTD("2", "11")
	pTD3M = newPTD("3", "19")
	// 1-10, 2-12
	pS1 = []*types.PriceSource{newPS(1, pTD1, pTD2)}
	// 2-12, 3-1
	pS2 = []*types.PriceSource{newPS(1, pTD3, pTD2)}
	// 1-10, 2-11(m)
	pS3 = []*types.PriceSource{newPS(1, pTD1, pTD2M)}
	// 2-12, 3-19(m)
	pS4 = []*types.PriceSource{newPS(1, pTD2, pTD3M)}
	// 1-10, 3-19(m)
	pS5 = []*types.PriceSource{newPS(1, pTD1, pTD3M)}

	pS6 = []*types.PriceSource{newPS(2, pTD1)}

	// 1-10, 2-12
	pS21 = []*types.PriceSource{newPS(1, pTD1, pTD2), newPS(2, pTD1, pTD3)}
	// 2-12, 3-15
	pS22 = []*types.PriceSource{newPS(1, pTD3, pTD2), newPS(2, pTD2, pTD3)}
	// 1-10, 2-11(m)
	pS23 = []*types.PriceSource{newPS(1, pTD1, pTD2M), newPS(2, pTD2M, pTD1)}
	// 2-12, 3-19(m)
	pS24 = []*types.PriceSource{newPS(1, pTD2, pTD3M), newPS(2, pTD3, pTD2M)}
	// 1-10, 3-19(m)
	pS25 = []*types.PriceSource{newPS(1, pTD1, pTD3M), newPS(2, pTD2M, pTD3M)}
)

var defaultParams = types.Params{
	Chains:       []*types.Chain{{Name: "-", Desc: "-"}, {Name: "Ethereum", Desc: "-"}},
	Tokens:       []*types.Token{{}, {Name: "eth", ChainID: 1, ContractAddress: "0xabc", Decimal: 18, Active: true, AssetID: ""}},
	Sources:      []*types.Source{{}, {Name: "chainLink", Entry: &types.Endpoint{}, Valid: true, Deterministic: true}},
	Rules:        []*types.RuleSource{{}, {SourceIDs: []uint64{1}}},
	TokenFeeders: []*types.TokenFeeder{{}, {TokenID: 1, RuleID: 1, StartRoundID: 1, StartBaseBlock: 0, Interval: 10, EndBlock: 0}},
	MaxNonce:     3,
	ThresholdA:   2,
	ThresholdB:   3,
	Mode:         1,
	MaxDetId:     5,
}
