package feedermanagement

import (
	"math/big"
	"sort"
	"strings"
)

type BigIntList []*big.Int

func (b BigIntList) Len() int {
	return len(b)
}

func (b BigIntList) Less(i, j int) bool {
	return b[i].Cmp(b[j]) < 0
}

func (b BigIntList) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

func (b BigIntList) Median() *big.Int {
	sort.Sort(b)
	l := len(b)
	if l%2 == 1 {
		return b[l/2]
	}
	return new(big.Int).Div(new(big.Int).Add(b[l/2], b[l/2-1]), big.NewInt(2))
}

type AggAlgorithm interface {
	Add(*PriceInfo) bool
	GetResult() *PriceResult
	Reset()
}

type priceType int

const (
	notSet priceType = iota
	number
	notNumber
)

type AggMedian struct {
	t           priceType
	finalString string
	list        []*big.Int
	decimal     int
}

func NewAggMedian() *AggMedian {
	return &AggMedian{
		list: make([]*big.Int, 0),
	}
}

func (a *AggMedian) Add(price *PriceResult) bool {
	priceInt, ok := new(big.Int).SetString(price.Price, 10)
	if ok {
		if a.t == notNumber {
			return false
		}
		if a.t == notSet {
			a.t = number
			a.list = append(a.list, priceInt)
			a.decimal = int(price.Decimal)
			return true
		}
		if a.decimal != int(price.Decimal) {
			if a.decimal > int(price.Decimal) {
				price.Price = price.Price + strings.Repeat("0", a.decimal-int(price.Decimal))
				priceInt, _ = new(big.Int).SetString(price.Price, 10)
			} else {
				delta := big.NewInt(int64(int(price.Decimal) - a.decimal))
				for _, v := range a.list {
					nv := new(big.Int).Mul(v, new(big.Int).Exp(big.NewInt(10), delta, nil))
					*v = *nv
				}
				a.decimal = int(price.Decimal)
			}
		}
		a.list = append(a.list, priceInt)
		return true
	} else {
		// input is a string, not a number
		if a.t == number {
			return false
		}
		if a.t == notSet {
			a.t = notNumber
			a.finalString = price.Price
			return true
		}
		if a.finalString != price.Price {
			return false
		}
		return true
	}
}

func (a *AggMedian) GetResult() *PriceResult {
	defer a.Reset()
	if a.t == notSet {
		return nil
	}
	if a.t == number {
		result := BigIntList(a.list).Median().String()
		decimal := int32(a.decimal)
		return &PriceResult{
			Price:   result,
			Decimal: decimal,
		}
	}
	if len(a.finalString) == 0 {
		return nil
	}
	result := a.finalString
	return &PriceResult{
		Price: result,
	}
}

func (a *AggMedian) Reset() {
	a.list = make([]*big.Int, 0)
	a.t = notSet
	a.decimal = 0
	a.finalString = ""
}

var defaultAggMedian = NewAggMedian()
