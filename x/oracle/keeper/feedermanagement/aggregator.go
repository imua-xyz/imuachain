package feedermanagement

import (
	"fmt"
	"math/big"
	"reflect"
	"slices"

	"golang.org/x/exp/maps"
)

type sourceChecker interface {
	IsDeterministic(sourceID int64) (bool, error)
}

func newAggregator(t *threshold, algo AggAlgorithm) *aggregator {
	return &aggregator{
		t:          t,
		finalPrice: nil,
		v:          newRecordsValidators(),
		ds:         newRecordsDSs(t),
		algo:       algo,
	}
}

func (a *aggregator) Equals(a2 *aggregator) bool {
	if a == nil || a2 == nil {
		return a == a2
	}

	if !reflect.DeepEqual(a.finalPrice, a2.finalPrice) {
		return false
	}

	if !a.t.Equals(a2.t) {
		return false
	}

	if !a.v.Equals(a2.v) {
		return false
	}

	if !a.ds.Equals(a2.ds) {
		return false
	}

	if !a.algo.Equals(a2.algo) {
		return false
	}

	return true
}

func (a *aggregator) CopyForCheckTx() *aggregator {
	if a == nil {
		return nil
	}
	var finalPrice *PriceResult
	if a.finalPrice != nil {
		tmp := *a.finalPrice
		finalPrice = &tmp
	}
	return &aggregator{
		t:          a.t.Cpy(),
		finalPrice: finalPrice,
		v:          a.v.Cpy(),
		ds:         a.ds.Cpy(),
		algo:       a.algo,
	}
}

func (a *aggregator) GetFinalPrice() (*PriceResult, bool) {
	if a.finalPrice != nil {
		return a.finalPrice, true
	}
	if !a.exceedsThreshold() {
		return nil, false
	}
	finalPrice, ok := a.v.GetFinalPrice(a.algo)
	if ok {
		a.finalPrice = finalPrice
	}
	return finalPrice, ok
}

func (a *aggregator) RecordMsg(msg *MsgItem) error {
	_, err := a.v.RecordMsg(msg)
	return err
}

// AddMsg records the message in a.v and do aggregation in a.ds
func (a *aggregator) AddMsg(msg *MsgItem) error {
	// record into recordsValidators, validation for duplication
	addedMsg, err := a.v.RecordMsg(msg)
	// all prices failed to be recorded
	if err != nil {
		return fmt.Errorf("failed to add quote, error:%w", err)
	}
	// add into recordsDSs for DS aggregation
	for _, ps := range addedMsg.PriceSources {
		if ps.deterministic {
			if a.ds.AddPriceSource(ps, msg.Power, msg.Validator) {
				finalPrice, ok := a.ds.GetFinalPriceForSourceID(ps.sourceID)
				if ok {
					a.v.UpdateFinalPriceForDS(ps.sourceID, finalPrice)
				}
			}
		}
	}
	return nil
}

// TODO: V2: the accumulatedPower should corresponding to all valid validators which provides all sources required by rules(defined in oracle.Params)
func (a *aggregator) exceedsThreshold() bool {
	return a.t.Exceeds(a.v.accumulatedPower)
}

func newRecordsValidators() *recordsValidators {
	return &recordsValidators{
		finalPrice:       nil,
		accumulatedPower: big.NewInt(0),
		records:          make(map[string]*priceValidator),
	}
}

func (rv *recordsValidators) Equals(rv2 *recordsValidators) bool {
	if rv == nil || rv2 == nil {
		return rv == rv2
	}

	if !reflect.DeepEqual(rv.finalPrice, rv2.finalPrice) {
		return false
	}
	if rv.accumulatedPower.Cmp(rv2.accumulatedPower) != 0 {
		return false
	}
	if !reflect.DeepEqual(rv.finalPrices, rv2.finalPrices) {
		return false
	}
	if len(rv.records) != len(rv2.records) {
		return false
	}
	// safe to range map, map compare
	for k, v := range rv.records {
		if v2, ok := rv2.records[k]; !ok || !v.Equals(v2) {
			return false
		}
	}

	return true
}

func (rv *recordsValidators) Cpy() *recordsValidators {
	if rv == nil {
		return nil
	}
	var finalPrice *PriceResult
	if rv.finalPrice != nil {
		tmp := *rv.finalPrice
		finalPrice = &tmp
	}
	var finalPrices map[string]*PriceResult
	if len(rv.finalPrices) > 0 {
		finalPrices = make(map[string]*PriceResult)
		// safe to range map, map copy
		for v, p := range rv.finalPrices {
			price := *p
			finalPrices[v] = &price
		}
	}
	records := make(map[string]*priceValidator)
	// safe to range map, map copy
	for v, pv := range rv.records {
		records[v] = pv.Cpy()
	}
	return &recordsValidators{
		finalPrice:       finalPrice,
		finalPrices:      finalPrices,
		accumulatedPower: new(big.Int).Set(rv.accumulatedPower),
		records:          records,
	}
}

func (rv *recordsValidators) RecordMsg(msg *MsgItem) (*MsgItem, error) {
	record, ok := rv.records[msg.Validator]
	if !ok {
		record = newPriceValidator(msg.Validator, msg.Power)
	}
	rets := &MsgItem{
		FeederID:     msg.FeederID,
		Validator:    msg.Validator,
		Power:        msg.Power,
		PriceSources: make([]*priceSource, 0),
	}

	updated, added, err := record.TryAddPriceSources(msg.PriceSources)
	if err != nil {
		return nil, fmt.Errorf("failed to record msg, error:%w", err)
	}
	record.ApplyAddedPriceSources(updated)
	if !ok {
		rv.records[msg.Validator] = record
		rv.accumulatedPower = new(big.Int).Add(rv.accumulatedPower, msg.Power)
	}
	rets.PriceSources = added
	return rets, nil
}

func (rv *recordsValidators) GetValidatorQuotePricesForSourceID(validator string, sourceID int64) ([]*PriceInfo, bool) {
	record, ok := rv.records[validator]
	if !ok {
		return nil, false
	}
	pSource, ok := record.priceSources[sourceID]
	if !ok {
		return nil, false
	}
	return pSource.prices, true
}

func (rv *recordsValidators) GetFinalPrice(algo AggAlgorithm) (*PriceResult, bool) {
	if rv.finalPrice != nil {
		return rv.finalPrice, true
	}
	if prices, ok := rv.GetFinalPriceForValidators(algo); ok {
		keySlice := make([]string, 0, len(prices))
		// safe to range map, this is used to generate a sorted keySlice
		for validator := range prices {
			keySlice = append(keySlice, validator)
		}
		slices.Sort(keySlice)
		algo.Reset()
		// keys are sorted to make sure algo.Add is called in the same order for deterministic result
		for _, validator := range keySlice {
			if !algo.Add(prices[validator]) {
				algo.Reset()
				return nil, false
			}
		}
		rv.finalPrice = algo.GetResult()
		return rv.finalPrice, rv.finalPrice != nil
	}
	return nil, false
}

func (rv *recordsValidators) GetFinalPriceForValidators(algo AggAlgorithm) (map[string]*PriceResult, bool) {
	if len(rv.finalPrices) > 0 {
		return rv.finalPrices, true
	}
	ret := make(map[string]*PriceResult)
	// the order here is not important, so it's safe to range map here
	// we only return true when all validators have finalPrice
	for validator, pv := range rv.records {
		finalPrice, ok := pv.GetFinalPrice(algo)
		if !ok {
			return nil, false
		}
		ret[validator] = finalPrice
	}
	rv.finalPrices = ret
	return ret, true
}

func (rv *recordsValidators) UpdateFinalPriceForDS(sourceID int64, finalPrice *PriceResult) bool {
	if finalPrice == nil {
		return false
	}
	// it's safe to range map here, order does not matter
	for _, record := range rv.records {
		// ignore the fail cases for updating some pv' DS finalPrice
		record.UpdateFinalPriceForDS(sourceID, finalPrice)
	}
	return true
}

func newRecordsDSs(t *threshold) *recordsDSs {
	return &recordsDSs{
		t:     t,
		dsMap: make(map[int64]*recordsDS),
	}
}

func (rdss *recordsDSs) Equals(rdss2 *recordsDSs) bool {
	if rdss == nil || rdss2 == nil {
		return rdss == rdss2
	}

	if !rdss.t.Equals(rdss2.t) {
		return false
	}
	if len(rdss.dsMap) != len(rdss2.dsMap) {
		return false
	}
	// safe to range map, map compare
	for k, v := range rdss.dsMap {
		if v2, ok := rdss2.dsMap[k]; !ok || !v.Equals(v2) {
			return false
		}
	}

	return true
}

func (rdss *recordsDSs) Cpy() *recordsDSs {
	if rdss == nil {
		return nil
	}
	dsMap := make(map[int64]*recordsDS)
	// safe to range map, map copy
	for id, r := range rdss.dsMap {
		dsMap[id] = r.Cpy()
	}
	return &recordsDSs{
		t:     rdss.t.Cpy(),
		dsMap: dsMap,
	}
}

// AddPriceSource adds prices for DS sources
func (rdss *recordsDSs) AddPriceSource(ps *priceSource, power *big.Int, validator string) bool {
	if !ps.deterministic {
		return false
	}
	price, ok := rdss.dsMap[ps.sourceID]
	if !ok {
		price = newRecordsDS()
		rdss.dsMap[ps.sourceID] = price
	}
	for _, p := range ps.prices {
		price.AddPrice(&PricePower{
			Price:      p,
			Power:      power,
			Validators: map[string]struct{}{validator: {}},
		})
	}
	return true
}

func (rdss *recordsDSs) GetFinalPriceForSourceID(sourceID int64) (*PriceResult, bool) {
	rds, ok := rdss.dsMap[sourceID]
	if !ok {
		return nil, false
	}
	return rds.GetFinalPrice(rdss.t)
}

func (rdss *recordsDSs) GetFinalPriceForSources() (map[int64]*PriceResult, bool) {
	ret := make(map[int64]*PriceResult)
	// safe to range map, the result is a map of 'all or none'
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
		validators:        make(map[string]struct{}),
		finalDetID:        "",
		accumulatedPowers: big.NewInt(0),
		records:           make([]*PricePower, 0),
	}
}

func (rds *recordsDS) Equals(rds2 *recordsDS) bool {
	if rds == nil || rds2 == nil {
		return rds == rds2
	}

	if !reflect.DeepEqual(rds.finalPrice, rds2.finalPrice) {
		return false
	}
	if rds.finalDetID != rds2.finalDetID {
		return false
	}
	if rds.accumulatedPowers.Cmp(rds2.accumulatedPowers) != 0 {
		return false
	}
	if !reflect.DeepEqual(rds.validators, rds2.validators) {
		return false
	}
	if len(rds.records) != len(rds2.records) {
		return false
	}
	for i, r := range rds.records {
		if !r.Equals(rds2.records[i]) {
			return false
		}
	}

	return true
}

func (rds *recordsDS) Cpy() *recordsDS {
	if rds == nil {
		return nil
	}
	var finalPrice *PriceResult
	if rds.finalPrice != nil {
		tmp := *rds.finalPrice
		finalPrice = &tmp
	}
	validators := make(map[string]struct{})
	// safe to range map, map copy
	for v := range rds.validators {
		validators[v] = struct{}{}
	}
	records := make([]*PricePower, 0, len(rds.records))
	for _, r := range rds.records {
		records = append(records, r.Cpy())
	}
	return &recordsDS{
		finalPrice:        finalPrice,
		finalDetID:        rds.finalDetID,
		accumulatedPowers: new(big.Int).Set(rds.accumulatedPowers),
		validators:        validators,
		records:           records,
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

// AddPrice adds a price into recordsDS
// NOTE: the input PricePower should be filtered by recordsValidators before calling this function to make sure the price is not duplicated by detID
func (rds *recordsDS) AddPrice(p *PricePower) {
	validator := maps.Keys(p.Validators)[0]
	i := 0
	l := len(rds.records)
	for ; i < l; i++ {
		record := rds.records[i]
		if record.Price.EqualDS(p.Price) {
			if _, ok := record.Validators[validator]; !ok {
				record.Power.Add(record.Power, p.Power)
				record.Validators[validator] = struct{}{}
			}
			break
		}
	}
	if i >= l {
		p = p.Cpy()
		for i = 0; i < l; i++ {
			record := rds.records[i]
			if p.Price.DetID <= record.Price.DetID {
				// insert before i
				combined := append([]*PricePower{p}, rds.records[i:]...)
				rds.records = append(rds.records[:i], combined...)
				break
			}
		}
		if i >= l {
			rds.records = append(rds.records, p)
		}
	}
	if _, ok := rds.validators[validator]; !ok {
		rds.accumulatedPowers.Add(rds.accumulatedPowers, p.Power)
		rds.validators[validator] = struct{}{}
	}
}
