package feedermanagement

import (
	"math"

	oracletypes "github.com/imua-xyz/imuachain/x/oracle/types"
)

type price oracletypes.MsgItem

func (p *price) withFeederID(feederID uint64) *price {
	ret := *p
	ret.FeederID = feederID
	return &ret
}

func (p *price) withValidator(validator string) *price {
	ret := *p
	ret.Validator = validator
	return &ret
}

func (p *price) msgItem() *oracletypes.MsgItem {
	ret := (oracletypes.MsgItem)(*p)
	return &ret
}

func newPrice(prices []*oracletypes.PriceTimeDetID) *price {
	return &price{
		PSources: []*oracletypes.PriceSource{
			{
				SourceID: 1,
				Prices:   prices,
			},
		},
	}
}

type validatorSet []*price
type powers struct {
	validators map[string]struct{}
	p          int
}
type blocks struct {
	msgItemsInBlocks [][]*price
	idx              int
	accumulated      map[string]*powers
	// for test cases we use int, so this condition is set to >= (equivalent to > for real bigInt cases)
	threshold int
	result    *oracletypes.PriceTimeDetID
}

type Blocks struct {
	MsgItemsInBlocks [][]*price
	Idx              int
	Accumulated      map[string]*powers
	// for test cases we use int, so this condition is set to >= (equivalent to > for real bigInt cases)
	Threshold int
	Result    *oracletypes.PriceTimeDetID
}

func newBlocks(t int) *blocks {
	return &blocks{
		msgItemsInBlocks: make([][]*price, 0),
		accumulated:      make(map[string]*powers),
		threshold:        t,
	}
}

func NewBlocks(t int) *Blocks {
	return &Blocks{
		MsgItemsInBlocks: make([][]*price, 0),
		Accumulated:      make(map[string]*powers),
		Threshold:        t,
	}
}

func (b *Blocks) AddPrices(ps []*price) {
	b.MsgItemsInBlocks = append(b.MsgItemsInBlocks, ps)
}
func (b *Blocks) Next() (ps []*price, result *oracletypes.PriceTimeDetID) {
	if b.Idx >= len(b.MsgItemsInBlocks) {
		return nil, nil
	}

	ret := b.MsgItemsInBlocks[b.Idx]
	b.Idx++
	// skip calculation, just return next msgs and result
	if b.Result != nil {
		return ret, b.Result
	}

	// calculate the expected result(fianlPrice)
	for _, pMsgItem := range ret {
		// TODO: test only, we assume only first element is valid(sourceID=1)
		if pMsgItem == nil {
			break
		}
		if len(pMsgItem.PSources) < 1 || pMsgItem.PSources[0].SourceID != 1 {
			panic("we support v1 test only")
		}
		validator := pMsgItem.Validator
		pPriceTimeDetIDs := pMsgItem.PSources[0].Prices
		for _, pPriceTimeDetID := range pPriceTimeDetIDs {
			acPower := 0
			if item := b.Accumulated[pPriceTimeDetID.DetID]; item != nil {
				if _, ok := item.validators[validator]; !ok {
					item.validators[validator] = struct{}{}
					item.p++
					acPower = item.p
				}
				// we dont update the tmp variable acPower if validator had been seen
			} else {
				b.Accumulated[pPriceTimeDetID.DetID] = &powers{
					validators: map[string]struct{}{validator: {}},
					p:          1,
				}
				acPower = 1
			}
			if acPower >= b.Threshold {
				if b.Result != nil && pPriceTimeDetID.DetID > b.Result.DetID {
					b.Result = pPriceTimeDetID
				}
			}
		}
	}
	return ret, nil
}

func (b *blocks) AddPirces(ps []*price) {
	b.msgItemsInBlocks = append(b.msgItemsInBlocks, ps)
}

func (b *blocks) next() (ps []*price, result *oracletypes.PriceTimeDetID) {
	if b.idx >= len(b.msgItemsInBlocks) {
		return nil, nil
	}

	ret := b.msgItemsInBlocks[b.idx]
	b.idx++
	// skip calculation, just return next msgs and result
	if b.result != nil {
		return ret, b.result
	}

	// calculate the expected result(fianlPrice)
	for _, pMsgItem := range ret {
		// TODO: test only, we assume only first element is valid(sourceID=1)
		if len(pMsgItem.PSources) < 1 || pMsgItem.PSources[0].SourceID != 1 {
			panic("we support v1 test only")
		}
		validator := pMsgItem.Validator
		pPriceTimeDetIDs := pMsgItem.PSources[0].Prices
		for _, pPriceTimeDetID := range pPriceTimeDetIDs {
			acPower := 0
			if item := b.accumulated[pPriceTimeDetID.DetID]; item != nil {
				if _, ok := item.validators[validator]; !ok {
					item.validators[validator] = struct{}{}
					item.p++
					acPower = item.p
				}
				// we dont update the tmp variable acPower if validator had been seen
			} else {
				b.accumulated[pPriceTimeDetID.DetID] = &powers{
					validators: map[string]struct{}{validator: {}},
					p:          1,
				}
				acPower = 1
			}
			if acPower >= b.threshold {
				b.result = pPriceTimeDetID
				return ret, b.result
			}
		}
	}
	return ret, nil
}

func (b *blocks) reset() {
	b.idx = 0
	b.accumulated = make(map[string]*powers)
}

func (b *Blocks) Reset() {
	b.Idx = 0
	b.Accumulated = make(map[string]*powers)
	b.Result = nil
}

func generateAllValidatorSets(ps []*price, validators []string) []validatorSet {
	total := len(validators)
	ret := make([]validatorSet, 0, 8^total)

	vs := make([]*price, total)
	var f func(int, int)
	f = func(depth int, total int) {
		if depth == 0 {
			cpy := make([]*price, total)
			// price never changed, it's fine to just copy the pointers
			copy(cpy, vs)
			ret = append(ret, cpy)
			return
		}
		for _, p := range ps {
			if p != nil {
				vs[total-depth] = p.withFeederID(1).withValidator(validators[total-depth])
			}
			f(depth-1, total)
		}
	}
	f(total, total)
	return ret
}

// this method has some hard coded value for easy listing all cases
// it restrict validatorSet corresponding for 4 validators
// we set this sizeValidators and check for caller to notice
// func generateAllBlocks(validatorSets []validatorSet, quotingWindow int, sizeValidators int) []*blocks {
func generateAllBlocks(validatorSets []validatorSet, quotingWindow int, sizeValidators int) []*Blocks {
	// TODO: support arbitrary size
	if sizeValidators != 4 {
		// this variable is not actually used in the following process for now
		panic("only support for 4 validators for test case")
	}
	// count of possible combinations for one validatorSet
	count := int(math.Pow(math.Pow(2, float64(sizeValidators)), float64(quotingWindow)))
	ret := make([]*Blocks, 0, len(validatorSets)*count)
	for _, vs := range validatorSets {
		// TODO: this should be generated from seizeValidtors instead of hard code
		// but it might break the defination as 'int', we just set 3 here for simplify temporary
		tmpBs := make([][]*price, 0)
		var f func(int, [][]int)

		f = func(depth int, idxs [][]int) {
			if depth == 0 {
				if len(tmpBs) != 3 {
					panic("length not equal to 3")
				}
				//				bs := newBlocks(3)
				bs := NewBlocks(3)
				for _, tmp := range tmpBs {
					cpy := make([]*price, len(tmp))
					copy(cpy, tmp)
					bs.AddPrices(cpy)
				}
				ret = append(ret, bs)
				return
			}
			if idxs == nil {
				//	depth--
				// 1 validators, including nil(zero validator)
				for i := -1; i < 4; i++ {
					if i == -1 {
						tmpBs = append(tmpBs, []*price{nil})
					} else {
						tmpBs = append(tmpBs, []*price{vs[i]})
					}
					f(depth-1, nil)

					if len(tmpBs) > 0 {
						tmpBs = tmpBs[:len(tmpBs)-1]
					}

				}

				// 2 validators

				f(depth, [][]int{{0, 1}, {0, 2}, {0, 3}, {1, 2}, {1, 3}, {2, 3}})

				// 3 validators
				f(depth, [][]int{{0, 1, 2}, {0, 1, 3}, {0, 2, 3}, {1, 2, 3}})

				// 4 ovalidators
				f(depth, [][]int{{0, 1, 2, 3}})
			} else {
				for _, idx := range idxs {
					tmp := make([]*price, 0, len(idx))
					for _, id := range idx {
						tmp = append(tmp, vs[id])
					}
					tmpBs = append(tmpBs, tmp)
					f(depth-1, nil)
					if len(tmpBs) > 0 {
						tmpBs = tmpBs[:len(tmpBs)-1]
					}

				}
			}
		}

		f(3, nil)
	}
	return ret
}

var (
	// only consider about combination
	// TODO: add cases as Permutation ?
	prices []*price = []*price{
		newPrice([]*oracletypes.PriceTimeDetID{
			{Price: "12100000000", Decimal: 8, DetID: "1"},
		}),
		newPrice([]*oracletypes.PriceTimeDetID{
			{Price: "12700000000", Decimal: 8, DetID: "2"},
		}),
		newPrice([]*oracletypes.PriceTimeDetID{
			{Price: "12900000000", Decimal: 8, DetID: "3"},
		}),

		newPrice([]*oracletypes.PriceTimeDetID{
			{Price: "12100000000", Decimal: 8, DetID: "1"},
			{Price: "12700000000", Decimal: 8, DetID: "2"},
		}),
		newPrice([]*oracletypes.PriceTimeDetID{
			{Price: "12100000000", Decimal: 8, DetID: "1"},
			{Price: "12900000000", Decimal: 8, DetID: "3"},
		}),
		newPrice([]*oracletypes.PriceTimeDetID{
			{Price: "12700000000", Decimal: 8, DetID: "2"},
			{Price: "12900000000", Decimal: 8, DetID: "3"},
		}),

		newPrice([]*oracletypes.PriceTimeDetID{
			{Price: "12100000000", Decimal: 8, DetID: "1"},
			{Price: "12700000000", Decimal: 8, DetID: "2"},
			{Price: "12900000000", Decimal: 8, DetID: "3"},
		}),
		// 0 price should be considered
		nil,
	}
)
