package oracle

import (
	oracletypes "github.com/ExocoreNetwork/exocore/x/oracle/types"
)

var now = "2025-01-01 00:00:00"

type priceTime struct {
	Price     string
	Decimal   int32
	Timestamp string
}

func (p priceTime) getPriceTimeDetID(detID string) oracletypes.PriceTimeDetID {
	return oracletypes.PriceTimeDetID{
		Price:     p.Price,
		Decimal:   p.Decimal,
		Timestamp: p.Timestamp,
		DetID:     detID,
	}
}

func (p priceTime) getPriceTimeRound(roundID uint64) oracletypes.PriceTimeRound {
	return oracletypes.PriceTimeRound{
		Price:     p.Price,
		Decimal:   p.Decimal,
		Timestamp: p.Timestamp,
		RoundID:   roundID,
	}
}

func (p priceTime) updateTimestamp() priceTime {
	p.Timestamp = now
	return p
}

//nolint:all
func (p priceTime) generateRealTimeStructs(detID string, sourceID uint64) (priceTime, oracletypes.PriceSource) {
	retP := p.updateTimestamp()
	pTimeDetID := retP.getPriceTimeDetID(detID)
	return retP, oracletypes.PriceSource{
		SourceID: sourceID,
		Prices: []*oracletypes.PriceTimeDetID{
			&pTimeDetID,
		},
	}
}

func generateNSTPriceTime(sc [][]int) priceTime {
	rawBytes := convertBalanceChangeToBytes(sc)
	return priceTime{
		Price:     string(rawBytes),
		Decimal:   0,
		Timestamp: now,
	}
}

var (
	price1 = priceTime{
		Price:     "19",
		Decimal:   8,
		Timestamp: now,
	}
	price2 = priceTime{
		Price:     "29",
		Decimal:   8,
		Timestamp: now,
	}

	stakerChanges1 = [][]int{{0, -4}}
	priceNST1      = generateNSTPriceTime(stakerChanges1)
)
