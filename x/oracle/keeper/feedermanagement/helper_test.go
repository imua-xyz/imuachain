package feedermanagement

import (
	"math/big"
	"math/rand"

	"github.com/ExocoreNetwork/exocore/x/oracle/keeper/testdata"
	oracletypes "github.com/ExocoreNetwork/exocore/x/oracle/types"
)

type Test struct {
}

var (
	tData     *Test
	params    = testdata.DefaultParamsForTest()
	r         = rand.New(rand.NewSource(1))
	timestamp = "2025-01-01 00:01:02"
	decimal   = int32(8)
	big1      = big.NewInt(1)
	big2      = big.NewInt(2)
	big3      = big.NewInt(3)
	big4      = big.NewInt(4)
	th        = &threshold{big4, big2, big3}
)

func (t *Test) NewFeederManager(cs CacheReader) *FeederManager {
	f := NewFeederManager(nil)
	round := t.NewRound(cs)
	f.rounds[round.feederID] = round
	// prepare this Round
	round.PrepareForNextBlock(int64(params.TokenFeeders[int(round.feederID)].StartBaseBlock))
	return f
}

func (t *Test) NewPricePower() *PricePower {
	return &PricePower{
		Price:      t.NewPriceInfo("999", "1"),
		Power:      big1,
		Validators: map[string]struct{}{"validator1": {}},
	}
}

func (t *Test) NewPriceSource(deterministic bool, filled bool) *priceSource {
	ret := newPriceSource(oracletypes.SourceChainlinkID, deterministic)
	if filled {
		price := t.NewPriceInfo("999", "1")
		ret.prices = append(ret.prices, price)
	}
	return ret
}

func (t *Test) NewPriceValidator(filled bool) *priceValidator {
	ret := newPriceValidator("validator1", big1)
	if filled {
		ps := t.NewPriceSource(true, true)
		ret.priceSources[oracletypes.SourceChainlinkID] = ps
	}
	return ret
}

func (t *Test) NewRecordsDS(filled bool) *recordsDS {
	ret := newRecordsDS()
	if filled {
		ret.validators["validtors"] = struct{}{}
		ret.accumulatedPowers = big1
		ret.records = append(ret.records, t.NewPricePower())
	}
	return ret
}

func (t *Test) NewRecordsDSs(filled bool) *recordsDSs {
	ret := newRecordsDSs(th)
	if filled {
		rds := t.NewRecordsDS(filled)
		ret.dsMap[oracletypes.SourceChainlinkID] = rds
	}
	return nil
}

func (t *Test) NewRecordsValidators(filled bool) *recordsValidators {
	ret := newRecordsValidators()
	if filled {
		ret.accumulatedPower = big1
		ret.records["validtor1"] = t.NewPriceValidator(filled)
	}
	return nil
}

func (t *Test) NewAggregator(filled bool) *aggregator {
	ret := newAggregator(th, defaultAggMedian)
	if filled {
		ret.v = t.NewRecordsValidators(filled)
		ret.ds = t.NewRecordsDSs(filled)
	}
	return ret
}

func (t *Test) NewRound(cs CacheReader) *round {
	feederID := r.Intn(len(params.TokenFeeders)-1) + 1
	round := newRound(int64(feederID), params.TokenFeeders[feederID], int64(params.MaxNonce), cs, defaultAggMedian)
	return round
}

func (t *Test) NewRoundWithFeederID(cs CacheReader, feederID int64) *round {
	round := newRound(feederID, params.TokenFeeders[feederID], int64(params.MaxNonce), cs, defaultAggMedian)
	return round
}

func (f *Test) NewPriceInfo(price string, detID string) *PriceInfo {
	return &PriceInfo{
		Price:     price,
		Decimal:   decimal,
		DetID:     detID,
		Timestamp: timestamp,
	}
}
