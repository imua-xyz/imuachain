package oracle

import (
	"strings"

	"github.com/imua-xyz/imuachain/testutil/network"
	oracletypes "github.com/imua-xyz/imuachain/x/oracle/types"
)

// PrepareXChainOracleGenesis returns a copy of the default oracle genesis with xchain token/feeder
// and feeders limited to xchain only. It does not mutate network.DefaultGenStateOracle, so other
// suites (e.g. CreatePrice) that run in the same process still get the default genesis.
func PrepareXChainOracleGenesis(cfg *network.Config) oracletypes.GenesisState {
	bz := cfg.Codec.MustMarshalJSON(&network.DefaultGenStateOracle)
	var copy oracletypes.GenesisState
	cfg.Codec.MustUnmarshalJSON(bz, &copy)
	params := &copy.Params

	// Find or add xchain token and feeder
	var xchainFeeder *oracletypes.TokenFeeder
	for i, token := range params.Tokens {
		if token != nil && strings.HasPrefix(strings.ToLower(token.AssetID), oracletypes.XChainIDPrefix) {
			for _, tf := range params.TokenFeeders {
				if tf != nil && int(tf.TokenID) == i {
					xchainFeeder = tf
					break
				}
			}
			break
		}
	}
	if xchainFeeder == nil {
		tokenID := uint64(len(params.Tokens))
		params.Tokens = append(params.Tokens, &oracletypes.Token{
			Name:            "XCHAIN_TEST",
			ChainID:         1,
			ContractAddress: "0x",
			Decimal:         0,
			Active:          true,
			AssetID:         "xchain_101",
		})
		params.TokenFeeders = append(params.TokenFeeders, &oracletypes.TokenFeeder{
			TokenID:        tokenID,
			RuleID:         3,
			StartRoundID:   1,
			StartBaseBlock: 5,
			Interval:       10,
		})
		xchainFeeder = params.TokenFeeders[len(params.TokenFeeders)-1]
		// ensure xchain token has initial price
		hasPrice := false
		for _, p := range copy.PricesList {
			if p.TokenID == tokenID {
				hasPrice = true
				break
			}
		}
		if !hasPrice {
			copy.PricesList = append(copy.PricesList, oracletypes.Prices{
				TokenID:     tokenID,
				NextRoundID: 1,
				PriceList: []*oracletypes.PriceTimeRound{
					{Price: "1", Decimal: 0, RoundID: 0},
				},
			})
		}
	}

	// Limit feeders to xchain only (same as limitOracleFeedersToXChain but on our copy)
	params.TokenFeeders = []*oracletypes.TokenFeeder{{}, xchainFeeder}
	return copy
}

// ensureXChainGatewayGenesis adds the oracle gateway to assets params. Mutates network.DefaultGenStateAssets only.
// Called by xchain tests so the network has the gateway; CreatePrice is unaffected by this.
func ensureXChainGatewayGenesis() {
	gateway := strings.ToLower(network.ExpectedOracleGatewayAddress().Hex())
	params := &network.DefaultGenStateAssets.Params
	if len(params.Gateways) == 0 {
		params.Gateways = []string{gateway}
		return
	}
	for _, addr := range params.Gateways {
		if strings.ToLower(addr) == gateway {
			return
		}
	}
	newGateways := []string{gateway}
	for _, addr := range params.Gateways {
		if strings.ToLower(addr) != gateway {
			newGateways = append(newGateways, addr)
		}
	}
	params.Gateways = newGateways
}
