package types

import (
	"fmt"
	"math/big"
	"strings"
)

func (p PriceAcc) ToPriceTR() PriceTimeRound {
	return PriceTimeRound{
		Price:     p.Price,
		RoundID:   p.LastRoundID,
		Timestamp: p.Timestamp,
		Decimal:   p.Decimal,
	}
}

func (p PriceAcc) AccumulatePriceTR(pTR PriceTimeRound) (PriceAcc, error) {
	tmpAccumulated, err := p.ToPriceTR().Accumulate(pTR)
	if err != nil {
		return p, err
	}
	return tmpAccumulated.ToPriceAcc(p.StartRoundID), nil
}

func (p PriceTimeRound) ToPriceAcc(startRoundID uint64) PriceAcc {
	return PriceAcc{
		Price:        p.Price,
		StartRoundID: startRoundID,
		LastRoundID:  p.RoundID,
		Timestamp:    p.Timestamp,
		Decimal:      p.Decimal,
	}
}

func (p PriceTimeRound) Equal(other PriceTimeRound) bool {
	if p.Decimal != other.Decimal {
		p, other = UniformDecimals(p, other)
	}
	return p.Price == other.Price
}

func (p PriceTimeRound) Accumulate(other PriceTimeRound) (PriceTimeRound, error) {
	if other.RoundID <= p.RoundID {
		return p, fmt.Errorf("cannot accumulate price with lower or equal round ID: %d <= %d", other.RoundID, p.RoundID)
	}
	rounds := other.RoundID - p.RoundID
	if p.Decimal != other.Decimal {
		p, other = UniformDecimals(p, other)
	}
	pBigInt, ok := new(big.Int).SetString(p.Price, 10)
	if !ok {
		return p, fmt.Errorf("invalid price format: %s", p.Price)
	}
	otherBigInt, ok := new(big.Int).SetString(other.Price, 10)
	if !ok {
		return p, fmt.Errorf("invalid price format: %s", other.Price)
	}
	otherBigInt = otherBigInt.Mul(otherBigInt, new(big.Int).SetUint64(rounds))
	pBigInt = pBigInt.Add(pBigInt, otherBigInt)
	return PriceTimeRound{
		Price:     pBigInt.String(),
		RoundID:   other.RoundID,
		Timestamp: p.Timestamp,
		Decimal:   p.Decimal,
	}, nil
}

func UniformDecimals(p1, p2 PriceTimeRound) (PriceTimeRound, PriceTimeRound) {
	if p1.Decimal == p2.Decimal {
		return p1, p2
	}
	if p1.Decimal > p2.Decimal {
		p2.Price += strings.Repeat("0", int(p1.Decimal-p2.Decimal))
		p2.Decimal = p1.Decimal
	} else {
		p1.Price += strings.Repeat("0", int(p2.Decimal-p1.Decimal))
		p1.Decimal = p2.Decimal
	}
	return p1, p2
}
