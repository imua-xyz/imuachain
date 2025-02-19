package testdata

import (
	"math/big"

	"github.com/imua-xyz/imuachain/x/oracle/types"
)

var (
	One     = big.NewInt(1)
	Zero    = big.NewInt(0)
	Ten     = big.NewInt(10)
	Eleven  = big.NewInt(11)
	Fifteen = big.NewInt(15)
	Twenty  = big.NewInt(20)
)

var (
	PTD1  = newPTD("1", "10")
	PTD2  = newPTD("2", "12")
	PTD3  = newPTD("3", "15")
	PTD2M = newPTD("2", "11")
	PTD3M = newPTD("3", "19")
	// 1-10, 2-12
	PS1 = []*types.PriceSource{newPS(1, PTD1, PTD2)}
	// 2-12, 3-1
	PS2 = []*types.PriceSource{newPS(1, PTD3, PTD2)}
	// 1-10, 2-11(m)
	PS3 = []*types.PriceSource{newPS(1, PTD1, PTD2M)}
	// 2-12, 3-19(m)
	PS4 = []*types.PriceSource{newPS(1, PTD2, PTD3M)}
	// 1-10, 3-19(m)
	PS5 = []*types.PriceSource{newPS(1, PTD1, PTD3M)}

	PS6 = []*types.PriceSource{newPS(2, PTD1)}

	// 1-10, 2-12
	PS21 = []*types.PriceSource{newPS(1, PTD1, PTD2), newPS(2, PTD1, PTD3)}
	// 2-12, 3-15
	PS22 = []*types.PriceSource{newPS(1, PTD3, PTD2), newPS(2, PTD2, PTD3)}
	// 1-10, 2-11(m)
	PS23 = []*types.PriceSource{newPS(1, PTD1, PTD2M), newPS(2, PTD2M, PTD1)}
	// 2-12, 3-19(m)
	PS24 = []*types.PriceSource{newPS(1, PTD2, PTD3M), newPS(2, PTD3, PTD2M)}
	// 1-10, 3-19(m)
	PS25 = []*types.PriceSource{newPS(1, PTD1, PTD3M), newPS(2, PTD2M, PTD3M)}
)

var (
	PTR1 = newPTR("100", 1)
	PTR2 = newPTR("109", 2)
	PTR3 = newPTR("117", 3)
	PTR4 = newPTR("129", 4)
	PTR5 = newPTR("121", 5)
	P1   = newPrices(1, 6, PTR1, PTR2, PTR3, PTR4, PTR5)
)

var DefaultParams = types.Params{
	Chains:       []*types.Chain{{Name: "-", Desc: "-"}, {Name: "Ethereum", Desc: "-"}},
	Tokens:       []*types.Token{{}, {Name: "eth", ChainID: 1, ContractAddress: "0xabc", Decimal: 18, Active: true}},
	Sources:      []*types.Source{{}, {Name: "chainLink", Entry: &types.Endpoint{}, Valid: true, Deterministic: true}},
	Rules:        []*types.RuleSource{{}, {SourceIDs: []uint64{1}}},
	TokenFeeders: []*types.TokenFeeder{{}, {TokenID: 1, RuleID: 1, StartRoundID: 1, StartBaseBlock: 0, Interval: 10, EndBlock: 0}},
}
