package feedermanagement

import (
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"sort"

	oracletypes "github.com/ExocoreNetwork/exocore/x/oracle/types"
)

func GetPriceInfoFromProtoPriceTimeDetID(p *oracletypes.PriceTimeDetID) *PriceInfo {
	return (*PriceInfo)(p)
}

func (p *PriceInfo) ProtoPriceTimeDetID() *oracletypes.PriceTimeDetID {
	return (*oracletypes.PriceTimeDetID)(p)
}

func (p *PriceInfo) EqualDS(pi *PriceInfo) bool {
	return p.Price == pi.Price && p.DetID == pi.DetID && p.Decimal == pi.Decimal
}

func (p *PriceInfo) PriceResult() *PriceResult {
	return (*PriceResult)(p)
}

func (p *PriceResult) PriceInfo() *PriceInfo {
	return (*PriceInfo)(p)
}

func (p *PriceResult) ProtoPriceTimeRound(roundID int64, timestamp string) *oracletypes.PriceTimeRound {
	return &oracletypes.PriceTimeRound{
		Price:     p.Price,
		Decimal:   p.Decimal,
		Timestamp: timestamp,
		RoundID:   uint64(roundID),
	}
}

func getPriceSourceFromProto(ps *oracletypes.PriceSource, checker sourceChecker) *priceSource {
	prices := make([]*PriceInfo, 0, len(ps.Prices))
	for _, p := range ps.Prices {
		prices = append(prices, GetPriceInfoFromProtoPriceTimeDetID(p))
	}
	return &priceSource{
		// #nosec G115
		deterministic: checker.IsDeterministic(int64(ps.SourceID)),
		// #nosec G115
		sourceID: int64(ps.SourceID),
		prices:   prices,
	}
}

func newPriceValidator(validator string, power *big.Int) *priceValidator {
	return &priceValidator{
		finalPrice:   nil,
		validator:    validator,
		power:        new(big.Int).Set(power),
		priceSources: make(map[int64]*priceSource),
	}
}

func (pv *priceValidator) Cpy() *priceValidator {
	if pv == nil {
		return nil
	}
	var finalPrice *PriceResult
	if pv.finalPrice != nil {
		tmp := *pv.finalPrice
		finalPrice = &tmp
	}
	priceSources := make(map[int64]*priceSource)
	for id, ps := range pv.priceSources {
		priceSources[id] = ps.Cpy()
	}
	return &priceValidator{
		finalPrice:   finalPrice,
		validator:    pv.validator,
		power:        new(big.Int).Set(pv.power),
		priceSources: priceSources,
	}
}

func (pv *priceValidator) Equals(pv2 *priceValidator) bool {
	if pv == nil && pv2 == nil {
		return true
	}
	if pv == nil || pv2 == nil {
		return false
	}
	if pv.validator != pv2.validator || pv.power.Cmp(pv2.power) != 0 {
		return false
	}
	if len(pv.priceSources) != len(pv2.priceSources) {
		return false
	}
	for id, ps := range pv.priceSources {
		ps2, ok := pv2.priceSources[id]
		if !ok || !ps.Equals(ps2) {
			return false
		}
	}
	return true
}

func (pv *priceValidator) GetPSCopy(sourceID int64, deterministic bool) *priceSource {
	if ps, ok := pv.priceSources[sourceID]; ok {
		return ps.Cpy()
	}
	return newPriceSource(sourceID, deterministic)
}

func (pv *priceValidator) TryAddPriceSources(pSs []*priceSource) (updated map[int64]*priceSource, added []*priceSource, err error) {
	var es errorStr
	updated = make(map[int64]*priceSource)
	for _, psNew := range pSs {
		ps, ok := updated[psNew.sourceID]
		if !ok {
			ps, ok = pv.priceSources[psNew.sourceID]
			if !ok {
				ps = newPriceSource(psNew.sourceID, psNew.deterministic)
			} else {
				ps = ps.Cpy()
			}
		}
		psAdded, err := ps.Add(psNew)
		if err != nil {
			es.add(fmt.Sprintf("sourceID:%d, error:%s", psNew.sourceID, err.Error()))
		} else {
			updated[psNew.sourceID] = ps
			added = append(added, psAdded)
		}
	}
	if len(updated) > 0 {
		return updated, added, nil
	}
	return nil, nil, fmt.Errorf("failed to add priceSource listi, error:%s", es)
}

func (pv *priceValidator) ApplyAddedPriceSources(psMap map[int64]*priceSource) {
	for id, ps := range psMap {
		pv.priceSources[id] = ps
	}
}

// TODO: V2: check valdiator has provided all sources required by rules(defined in oracle.params)
func (pv *priceValidator) GetFinalPrice() (*PriceResult, bool) {
	if pv.finalPrice != nil {
		return pv.finalPrice, true
	}
	if len(pv.priceSources) == 0 {
		return nil, false
	}
	for _, price := range pv.priceSources {
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

func (pv *priceValidator) UpdateFinalPriceForDS(sourceID int64, finalPrice *PriceResult) bool {
	if finalPrice == nil {
		return false
	}
	if price, ok := pv.priceSources[sourceID]; ok {
		price.finalPrice = finalPrice
		return true
	}
	return false
}

func newPriceSource(sourceID int64, deterministic bool) *priceSource {
	return &priceSource{
		deterministic: deterministic,
		finalPrice:    nil,
		sourceID:      sourceID,
		detIDs:        make(map[string]struct{}),
		prices:        make([]*PriceInfo, 0),
	}
}

func (ps *priceSource) Equals(ps2 *priceSource) bool {
	if ps == nil && ps2 == nil {
		return true
	}
	if ps == nil || ps2 == nil {
		return false
	}
	if ps.sourceID != ps2.sourceID || ps.deterministic != ps2.deterministic {
		return false
	}
	if !reflect.DeepEqual(ps.detIDs, ps2.detIDs) {
		return false
	}
	if !reflect.DeepEqual(ps.finalPrice, ps2.finalPrice) {
		return false
	}
	if len(ps.prices) != len(ps2.prices) {
		return false
	}
	if !reflect.DeepEqual(ps.prices, ps2.prices) {
		return false
	}
	return true
}

func (ps *priceSource) Cpy() *priceSource {
	if ps == nil {
		return nil
	}
	var finalPrice *PriceResult
	if ps.finalPrice != nil {
		tmp := *ps.finalPrice
		finalPrice = &tmp
	}
	// deterministic, sourceID
	detIDs := make(map[string]struct{})
	for detID := range ps.detIDs {
		detIDs[detID] = struct{}{}
	}
	prices := make([]*PriceInfo, 0, len(ps.prices))
	for _, p := range ps.prices {
		pCpy := *p
		prices = append(prices, &pCpy)
	}
	return &priceSource{
		deterministic: ps.deterministic,
		finalPrice:    finalPrice,
		sourceID:      ps.sourceID,
		detIDs:        detIDs,
		prices:        prices,
	}
}

// Add adds prices of a source from priceSource
// we don't verify the input is DS or NS, it's just handled under the rule restrict by p.deterministic
func (ps *priceSource) Add(psNew *priceSource) (*priceSource, error) {
	if ps.sourceID != psNew.sourceID {
		return nil, fmt.Errorf("failed to add priceSource, sourceID mismatch, expected:%d, got:%d", ps.sourceID, psNew.sourceID)
	}

	if !ps.deterministic {
		// this is not ds, then just set the final price or overwrite if the input has a later timestamp
		if ps.finalPrice == nil {
			ps.finalPrice = psNew.prices[0].PriceResult()
			ps.prices = append(ps.prices, psNew.prices[0])
			psNew.prices = psNew.prices[:1]
			return psNew, nil
		}
		// equivalent to After, just overwrite the old value
		if psNew.prices[0].Timestamp > ps.finalPrice.Timestamp {
			ps.finalPrice = psNew.prices[0].PriceResult()
			ps.prices = append(ps.prices, psNew.prices[0])
			psNew.prices = psNew.prices[:1]
			return ps, nil
		}
		return nil, errors.New("failed to add ProtoPriceSource for NS, timestamp is old")
	}

	var es errorStr
	added := false
	ret := &priceSource{
		deterministic: ps.deterministic,
		sourceID:      ps.sourceID,
		prices:        make([]*PriceInfo, 0),
	}
	for _, pNew := range psNew.prices {
		if _, ok := ps.detIDs[pNew.DetID]; ok {
			es.add(fmt.Sprintf("duplicated DetID:%s", pNew.DetID))
			continue
		}
		added = true
		ps.detIDs[pNew.DetID] = struct{}{}
		ps.prices = append(ps.prices, pNew)
		ret.prices = append(ret.prices, pNew)
	}

	if !added {
		return nil, fmt.Errorf("failed to add ProtoPriceSource, sourceID:%d, errors:%s", ps.sourceID, es)
	}

	sort.Slice(ps.prices, func(i, j int) bool {
		return ps.prices[i].DetID < ps.prices[j].DetID
	})
	return ret, nil
}

func (p *PricePower) Equals(p2 *PricePower) bool {
	if p == nil && p2 == nil {
		return true
	}
	if p == nil || p2 == nil {
		return false
	}
	if !reflect.DeepEqual(p.Price, p2.Price) || p.Power.Cmp(p2.Power) != 0 {
		return false
	}
	if len(p.Validators) != len(p2.Validators) {
		return false
	}
	for v := range p.Validators {
		if _, ok := p2.Validators[v]; !ok {
			return false
		}
	}
	return true
}

func (p *PricePower) Cpy() *PricePower {
	price := *p.Price
	validators := make(map[string]struct{})
	for v := range p.Validators {
		validators[v] = struct{}{}
	}
	return &PricePower{
		Price:      &price,
		Power:      new(big.Int).Set(p.Power),
		Validators: validators,
	}
}

type errorStr string

func (e *errorStr) add(s string) {
	es := string(*e)
	*e = errorStr(fmt.Sprintf("%s[%s]", es, s))
}
