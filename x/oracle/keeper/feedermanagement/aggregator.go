package feedermanagement

import (
	"math/big"
	"slices"

	oracletypes "github.com/ExocoreNetwork/exocore/x/oracle/types"
	"golang.org/x/exp/maps"
)

func newAggregator(t *threshold) *aggregator {
	return &aggregator{
		t:          t,
		finalPrice: nil,
		v:          newRecordsValdiators(),
		ds:         newRecordsDSs(t),
	}
}

func (a *aggregator) GetFinalPrice() (*PriceResult, bool) {
	if a.finalPrice != nil {
		return a.finalPrice, true
	}
	if !a.exceedPowerLimit() {
		return nil, false
	}
	finalPrice, ok := a.v.GetFinalPrice()
	if ok {
		a.finalPrice = finalPrice
	}
	return finalPrice, ok
}

// AddMsg records the message in a.v and do aggregation in a.ds
func (a *aggregator) AddMsg(msg *oracletypes.MsgItem) error {
	// TODO: implement me
	return nil
}

// RecordMsg only records the message in a.v
func (a *aggregator) RecordMsg(msg *oracletypes.MsgItem) {

}

// TODO: V2: the accumulatedPower should corresponding to all valid validators which provides all sources required by rules(defined in oracle.Params)
func (a *aggregator) exceedPowerLimit() bool {
	return a.t.Exceeds(a.v.accumulatedPower)
}

func newPriceValidator(validator string, power *big.Int) *priceValidator {
	return &priceValidator{
		finalPrice:   nil,
		validator:    validator,
		power:        new(big.Int).Set(power),
		pricesSource: make(map[int64]*priceSource, 0),
	}
}

// TODO: V2: check valdiator has provided all sources required by rules(defined in oracle.params)
func (pv *priceValidator) GetFinalPrice() (*PriceResult, bool) {
	if pv.finalPrice != nil {
		return pv.finalPrice, true
	}
	if len(pv.pricesSource) == 0 {
		return nil, false
	}
	//	defer defaultAggMedian.Reset()
	for _, price := range pv.pricesSource {
		if price.finalPrice == nil {
			defaultAggMedian.Reset()
			return nil, false
		}
		if !defaultAggMedian.Add(price.finalPrice) {
			defaultAggMedian.Reset()
			return nil, false
		}
	}
	pv.finalPrice = defaultAggMedian.GetResult()
	return pv.finalPrice, true
}

func (pv *priceValidator) UpdateFinalPriceForDetSource(sourceID int64, finalPrice *PriceResult) bool {
	if finalPrice == nil {
		return false
	}
	if price, ok := pv.pricesSource[sourceID]; ok {
		price.finalPrice = finalPrice
		return true
	}
	return false
}

func newRecordsValdiators() *recordsValidators {
	return &recordsValidators{
		finalPrice:       nil,
		accumulatedPower: big.NewInt(0),
		records:          make(map[string]*priceValidator),
	}
}

func (rv *recordsValidators) GetValidatorQuotePricesForSourceID(validator string, sourceID int64) ([]*PriceInfo, bool) {
	record, ok := rv.records[validator]
	if !ok {
		return nil, false
	}
	pSource, ok := record.pricesSource[sourceID]
	if !ok {
		return nil, false
	}
	return pSource.prices, true
}

func (rv *recordsValidators) GetFinalPrice() (*PriceResult, bool) {
	if rv.finalPrice != nil {
		return rv.finalPrice, true
	}
	if prices, ok := rv.GetFinalPriceForValidators(); ok {
		keySlice := make([]string, 0, len(prices))
		for validator, _ := range prices {
			keySlice = append(keySlice, validator)
		}
		slices.Sort(keySlice)
		for _, validator := range keySlice {
			if !defaultAggMedian.Add(prices[validator]) {
				defaultAggMedian.Reset()
				return nil, false
			}
		}
		rv.finalPrice = defaultAggMedian.GetResult()
		return rv.finalPrice, true
	}
	return nil, false
}

func (rv *recordsValidators) GetFinalPriceForValidators() (map[string]*PriceResult, bool) {
	if rv.finalPrices != nil {
		return rv.finalPrices, true
	}
	ret := make(map[string]*PriceResult)
	for validator, pv := range rv.records {
		if finalPrice, ok := pv.GetFinalPrice(); !ok {
			return nil, false
		} else {
			ret[validator] = finalPrice
		}
	}
	rv.finalPrices = ret
	return ret, true
}

func (rv *recordsValidators) UpdateFinalPriceForDetSource(sourceID int64, finalPrice *PriceResult) bool {
	if finalPrice == nil {
		return false
	}
	// it's safe to range map here, order does not matter
	for _, record := range rv.records {
		// ignore the fail cases for updating some pv' DS finalPrice
		record.UpdateFinalPriceForDetSource(sourceID, finalPrice)
	}
	return true
}

func newRecordsDSs(t *threshold) *recordsDSs {
	return &recordsDSs{
		t:     t,
		dsMap: make(map[int64]*recordsDS),
	}
}

func (rdss *recordsDSs) GetFinalPriceForSources() (map[int64]*PriceResult, bool) {
	ret := make(map[int64]*PriceResult)
	for sourceID, rds := range rdss.dsMap {
		if finalPrice, ok := rds.GetFinalPrice(rdss.t); ok {
			ret[sourceID] = finalPrice
		} else {
			return nil, false
		}
	}
	return ret, true
}

func (rdss *recordsDSs) GetFinalDetIDForSourceID(sourceID int64) string {
	if rds, ok := rdss.dsMap[sourceID]; ok {
		if rds.finalPrice != nil {
			return rds.finalDetID
		}
		if _, ok := rds.GetFinalPrice(rdss.t); ok {
			return rds.finalDetID
		}
	}
	return ""
}

func newRecordsDS() *recordsDS {
	return &recordsDS{
		finalPrice:        nil,
		finalDetID:        "",
		accumulatedPowers: big.NewInt(0),
		records:           make([]*PricePower, 0),
	}
}

func (rds *recordsDS) GetFinalPrice(t *threshold) (*PriceResult, bool) {
	if rds.finalPrice != nil {
		return rds.finalPrice, true
	}
	if t.Exceeds(rds.accumulatedPowers) {
		l := len(rds.records)
		for i := l - 1; i >= 0; i-- {
			pPower := rds.records[i]
			if t.Exceeds(pPower.Power) {
				rds.finalPrice = pPower.Price.PriceResult()
				rds.finalDetID = pPower.Price.DetID
				return rds.finalPrice, true
			}
		}
	}
	return nil, false
}

func (rds *recordsDS) AddPrice(p *PricePower) {
	validator := maps.Keys(p.validators)[0]
	biggestDetID := true
	for i, record := range rds.records {
		if record.Price.EqualDS(p.Price) {
			if _, ok := record.validators[validator]; !ok {
				record.Power = record.Power.Add(record.Power, p.Power)
				record.validators[validator] = struct{}{}
			}
			biggestDetID = false
			break
		}
		if p.Price.DetID <= record.Price.DetID {
			// insert before i
			combined := append([]*PricePower{p}, rds.records[i:]...)
			rds.records = append(rds.records[:i], combined...)
			biggestDetID = false
			break
		}
	}
	if _, ok := rds.validators[validator]; !ok {
		rds.accumulatedPowers = rds.accumulatedPowers.Add(rds.accumulatedPowers, p.Power)
		rds.validators[validator] = struct{}{}
	}
	if biggestDetID {
		rds.records = append(rds.records, p)
	}
}
