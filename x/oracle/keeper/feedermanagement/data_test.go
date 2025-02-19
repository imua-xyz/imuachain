package feedermanagement

import oracletypes "github.com/ExocoreNetwork/exocore/x/oracle/types"

var (
	ps1 = &priceSource{
		deterministic: true,
		sourceID:      1,
		prices:        []*PriceInfo{{Price: "999", Decimal: 8, DetID: "1"}},
	}
	ps1_2 = &priceSource{
		deterministic: true,
		sourceID:      1,
		prices:        []*PriceInfo{{Price: "999", Decimal: 8, DetID: "1"}, {Price: "998", Decimal: 8, DetID: "1"}},
	}
	ps1_3 = &priceSource{
		deterministic: true,
		sourceID:      1,
		prices:        []*PriceInfo{{Price: "999", Decimal: 8, DetID: "1"}, {Price: "999", Decimal: 8, DetID: "2"}},
	}
	ps2 = &priceSource{
		deterministic: true,
		sourceID:      1,
		prices:        []*PriceInfo{{Price: "999", Decimal: 8, DetID: "2"}},
	}
	ps3 = &priceSource{
		deterministic: true,
		sourceID:      2,
		prices:        []*PriceInfo{{Price: "999", Decimal: 8, DetID: "3"}},
	}
	ps3_2 = &priceSource{
		deterministic: true,
		sourceID:      1,
		prices:        []*PriceInfo{{Price: "999", Decimal: 8, DetID: "3"}},
	}
	ps4 = &priceSource{
		deterministic: true,
		sourceID:      1,
		prices:        []*PriceInfo{{Price: "999", Decimal: 8, DetID: "2"}, {Price: "999", Decimal: 8, DetID: "2"}},
	}
	ps5 = &priceSource{
		deterministic: true,
		sourceID:      1,
		detIDs:        map[string]struct{}{"1": {}, "2": {}},
		prices:        []*PriceInfo{{Price: "999", Decimal: 8, DetID: "1"}, {Price: "999", Decimal: 8, DetID: "2"}},
	}
	ps6 = &priceSource{
		deterministic: true,
		sourceID:      1,
		detIDs:        map[string]struct{}{"1": {}, "2": {}, "3": {}},
		prices:        []*PriceInfo{{Price: "999", Decimal: 8, DetID: "1"}, {Price: "999", Decimal: 8, DetID: "2"}, {Price: "999", Decimal: 8, DetID: "3"}},
	}
	msgItem1 = &MsgItem{
		FeederID:     1,
		Validator:    "validator1",
		Power:        big1,
		PriceSources: []*priceSource{ps1_2, ps2},
	}
	msgItem1_2 = &MsgItem{
		FeederID:     1,
		Validator:    "validator1",
		Power:        big1,
		PriceSources: []*priceSource{ps1, ps2},
	}
	msgItem1_3 = &MsgItem{
		FeederID:     1,
		Validator:    "validator1",
		Power:        big1,
		PriceSources: []*priceSource{ps1},
	}
	msgItem2 = &MsgItem{
		FeederID:     1,
		Validator:    "validator2",
		Power:        big1,
		PriceSources: []*priceSource{ps1_2, ps2},
	}
	msgItem2_2 = &MsgItem{
		FeederID:     1,
		Validator:    "validator2",
		Power:        big1,
		PriceSources: []*priceSource{ps1, ps2},
	}
	msgItem3 = &MsgItem{
		FeederID:     1,
		Validator:    "validator3",
		Power:        big1,
		PriceSources: []*priceSource{ps2},
	}
	protoMsgItem1   = newTestProtoMsgItem(1, "validator1", "999", "1")
	protoMsgItem2   = newTestProtoMsgItem(1, "validator1", "999", "2")
	protoMsgItem3   = newTestProtoMsgItem(1, "validator2", "999", "2")
	protoMsgItem4   = newTestProtoMsgItem(1, "validator3", "999", "2")
	protoMsgItem4_2 = newTestProtoMsgItem(1, "validator3", "777", "2")
	protoMsgItem5   = newTestProtoMsgItem(1, "validator4", "999", "2")

	pr1 = &PriceResult{
		Price:     "999",
		Decimal:   8,
		DetID:     "1",
		Timestamp: timestamp,
	}
	pr1_2 = &PriceResult{
		Price:   "999",
		Decimal: 8,
	}
	pw1 = &PricePower{
		Price: &PriceInfo{
			Price:     "999",
			Decimal:   8,
			DetID:     "1",
			Timestamp: timestamp,
		},
		Power:      big1,
		Validators: map[string]struct{}{"validator1": {}},
	}
	pw2 = &PricePower{
		Price: &PriceInfo{
			Price:     "999",
			Decimal:   8,
			DetID:     "2",
			Timestamp: timestamp,
		},
		Power:      big1,
		Validators: map[string]struct{}{"validator1": {}},
	}
	pw2_2 = &PricePower{
		Price: &PriceInfo{
			Price:     "999",
			Decimal:   8,
			DetID:     "2",
			Timestamp: timestamp,
		},
		Power:      big2,
		Validators: map[string]struct{}{"validator1": {}, "validator2": {}},
	}

	pw3 = &PricePower{
		Price: &PriceInfo{
			Price:     "999",
			Decimal:   8,
			DetID:     "2",
			Timestamp: timestamp,
		},
		Power:      big1,
		Validators: map[string]struct{}{"validator2": {}},
	}
	pw3_2 = &PricePower{
		Price: &PriceInfo{
			Price:     "999",
			Decimal:   8,
			DetID:     "2",
			Timestamp: timestamp,
		},
		Power:      big3,
		Validators: map[string]struct{}{"validator1": {}, "validator2": {}, "validator3": {}},
	}

	pw4 = &PricePower{
		Price: &PriceInfo{
			Price:     "999",
			Decimal:   8,
			DetID:     "2",
			Timestamp: timestamp,
		},
		Power:      big1,
		Validators: map[string]struct{}{"validator3": {}},
	}
	pw5 = &PricePower{
		Price: &PriceInfo{
			Price:     "999",
			Decimal:   8,
			DetID:     "2",
			Timestamp: timestamp,
		},
		Power:      big1,
		Validators: map[string]struct{}{"validator4": {}},
	}
)

func newTestProtoMsgItem(feederID uint64, validator string, price string, detID string) *oracletypes.MsgItem {
	return &oracletypes.MsgItem{
		FeederID: feederID,
		PSources: []*oracletypes.PriceSource{{
			SourceID: 1,
			Prices: []*oracletypes.PriceTimeDetID{{
				Price:     price,
				Decimal:   8,
				DetID:     detID,
				Timestamp: timestamp,
			}},
		}},
		Validator: validator,
	}
}
