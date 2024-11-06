package common

import (
	"math/big"
	"sort"

	"github.com/ExocoreNetwork/exocore/x/oracle/types"
)

var (
	// maxNonce indicates how many messages a validator can submit in a single round to offer price
	// current we use this as a mock distance
	MaxNonce int32 = 3

	// these two threshold value used to set the threshold to tell when the price had come to consensus and was able to get a final price of that round
	ThresholdA int32 = 2
	ThresholdB int32 = 3

	// maxDetId each validator can submit, so the calculator can cache maximum of maxDetId*count(validators) values, this is for resistance of malicious validator submmiting invalid detId
	MaxDetID int32 = 5

	// for each token at most MaxSizePrices round of prices will be keep in store
	MaxSizePrices = 100

	// consensus mode: v1: as soon as possbile
	Mode types.ConsensusMode = types.ConsensusModeASAP
)

type Set[T comparable] struct {
	size  int
	slice []T
}

func (s *Set[T]) Copy() *Set[T] {
	ret := NewSet[T](s.Length())
	ret.slice = append(ret.slice, s.slice...)
	return ret
}

func (s *Set[T]) Add(value T) bool {
	if len(s.slice) == s.size {
		return false
	}
	for _, v := range s.slice {
		if v == value {
			return false
		}
	}
	s.slice = append(s.slice, value)
	return true
}

func (s *Set[T]) Has(value T) bool {
	for _, v := range s.slice {
		if v == value {
			return true
		}
	}
	return false
}

func (s *Set[T]) Length() int {
	return s.size
}

func NewSet[T comparable](length int) *Set[T] {
	return &Set[T]{
		size:  length,
		slice: make([]T, 0, length),
	}
}

func ExceedsThreshold(power *big.Int, totalPower *big.Int) bool {
	return new(big.Int).Mul(power, big.NewInt(int64(ThresholdB))).Cmp(new(big.Int).Mul(totalPower, big.NewInt(int64(ThresholdA)))) > 0
}

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
