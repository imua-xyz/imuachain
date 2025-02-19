package cache

import "github.com/imua-xyz/imuachain/x/oracle/types"

var defaultParams = types.Params{
	Chains:       []*types.Chain{{Name: "-", Desc: "-"}, {Name: "Ethereum", Desc: "-"}},
	Tokens:       []*types.Token{{}, {Name: "eth", ChainID: 1, ContractAddress: "0xabc", Decimal: 18, Active: true}},
	Sources:      []*types.Source{{}, {Name: "chainLink", Entry: &types.Endpoint{}, Valid: true, Deterministic: true}},
	Rules:        []*types.RuleSource{{}, {SourceIDs: []uint64{1}}},
	TokenFeeders: []*types.TokenFeeder{{}, {TokenID: 1, RuleID: 1, StartRoundID: 1, StartBaseBlock: 0, Interval: 10, EndBlock: 0}},
}
