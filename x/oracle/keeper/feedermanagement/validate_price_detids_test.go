package feedermanagement

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	oracletypes "github.com/imua-xyz/imuachain/x/oracle/types"
)

// buildMsgCreatePrice constructs a MsgCreatePrice with given feederID, creator acc address,
// one sourceID and a list of detIDs (all with the same dummy price/decimal for simplicity).
func buildMsgCreatePrice(feederID uint64, creator sdk.AccAddress, sourceID uint64, detIDs []string) *oracletypes.MsgCreatePrice {
	prices := make([]*oracletypes.PriceTimeDetID, 0, len(detIDs))
	for _, d := range detIDs {
		prices = append(prices, &oracletypes.PriceTimeDetID{Price: "1", Decimal: 18, DetID: d})
	}
	return &oracletypes.MsgCreatePrice{
		Creator:  creator.String(),
		FeederID: feederID,
		Prices: []*oracletypes.PriceSource{
			{
				SourceID: sourceID,
				Prices:   prices,
			},
		},
	}
}

func TestValidatePriceSourceDetIDs_EdgeCases(t *testing.T) {
	// minimal FeederManager with only rounds populated as needed
	fm := &FeederManager{rounds: make(map[int64]*round)}
	creator := sdk.AccAddress(make([]byte, 20))

	t.Run("returns true when priceSourceDetIDs == nil (FeederID==0)", func(t *testing.T) {
		msg := buildMsgCreatePrice(0, creator, 1, []string{"a"})
		require.True(t, fm.ValidatePriceSourceDetIDs(msg))
	})

	t.Run("returns true when FeederID exceeds len(f.rounds)", func(t *testing.T) {
		// len(rounds)==0, feederID=5 -> 5>0
		msg := buildMsgCreatePrice(5, creator, 1, []string{"a"})
		require.True(t, fm.ValidatePriceSourceDetIDs(msg))
	})

	t.Run("returns true when corresponding round missing despite len >= feederID", func(t *testing.T) {
		// Populate rounds with unrelated keys to make len>=2, but no key == 1
		fm.rounds[10] = &round{}
		fm.rounds[20] = &round{}
		// len(rounds)=2, feederID=1 (1<=2) but key 1 missing -> should return true
		msg := buildMsgCreatePrice(1, creator, 1, []string{"a"})
		require.True(t, fm.ValidatePriceSourceDetIDs(msg))
		// cleanup
		delete(fm.rounds, 10)
		delete(fm.rounds, 20)
	})

	t.Run("returns true when aggregator or validator records missing", func(t *testing.T) {
		// Round exists but aggregator nil
		fm.rounds[1] = &round{}
		msg := buildMsgCreatePrice(1, creator, 1, []string{"a"})
		require.True(t, fm.ValidatePriceSourceDetIDs(msg))

		// Aggregator exists but recordsValidators nil
		fm.rounds[1] = &round{a: &aggregator{}}
		require.True(t, fm.ValidatePriceSourceDetIDs(msg))

		// recordsValidators exists but records map nil
		fm.rounds[1] = &round{a: &aggregator{v: &recordsValidators{}}}
		require.True(t, fm.ValidatePriceSourceDetIDs(msg))
	})

	t.Run("returns true when validator entry exists but is explicitly nil", func(t *testing.T) {
		r := &round{a: &aggregator{v: &recordsValidators{records: make(map[string]*priceValidator)}}}
		validator, _ := oracletypes.ConsAddrStrFromCreator(creator.String())
		r.a.v.records[validator] = nil
		fm.rounds[1] = r
		msg := buildMsgCreatePrice(1, creator, 1, []string{"a"})
		require.True(t, fm.ValidatePriceSourceDetIDs(msg))
	})

	t.Run("returns true when any detID is unseen for its sourceID", func(t *testing.T) {
		// Setup: existing detIDs only has "a"
		validator, _ := oracletypes.ConsAddrStrFromCreator(creator.String())
		ps := &priceSource{sourceID: 1, detIDs: map[string]struct{}{"a": {}}}
		pv := &priceValidator{validator: validator, priceSources: map[int64]*priceSource{1: ps}}
		rec := &recordsValidators{records: map[string]*priceValidator{validator: pv}}
		fm.rounds[1] = &round{a: &aggregator{v: rec}}

		// Msg includes {"a", "b"} -> "b" unseen => should return true
		msg := buildMsgCreatePrice(1, creator, 1, []string{"a", "b"})
		require.True(t, fm.ValidatePriceSourceDetIDs(msg))
	})

	t.Run("returns false when all detIDs are already present for the same sourceID", func(t *testing.T) {
		// Setup: existing detIDs has {"a","b"}
		validator, _ := oracletypes.ConsAddrStrFromCreator(creator.String())
		ps := &priceSource{sourceID: 2, detIDs: map[string]struct{}{"a": {}, "b": {}}}
		pv := &priceValidator{validator: validator, priceSources: map[int64]*priceSource{2: ps}}
		rec := &recordsValidators{records: map[string]*priceValidator{validator: pv}}
		fm.rounds[2] = &round{a: &aggregator{v: rec}}

		// Msg includes only duplicates {"a","b"} for sourceID 2 -> false
		msg := buildMsgCreatePrice(2, creator, 2, []string{"a", "b"})
		require.False(t, fm.ValidatePriceSourceDetIDs(msg))
	})
}
