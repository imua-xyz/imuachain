package oracle

import (
	"time"

	oracletypes "github.com/ExocoreNetwork/exocore/x/oracle/types"
)

const layout = "2006-01-02 15:04:05"

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
	t := time.Now().UTC().Format(layout)
	p.Timestamp = t
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
		Timestamp: time.Now().UTC().Format(layout),
	}
}

var (
	price1 = priceTime{
		Price:     "199999",
		Decimal:   18,
		Timestamp: time.Now().UTC().Format(layout),
	}
	price2 = priceTime{
		Price:     "299999",
		Decimal:   18,
		Timestamp: time.Now().UTC().Format(layout),
	}

	stakerChanges1 = [][]int{{0, -4}}
	priceNST1      = generateNSTPriceTime(stakerChanges1)
)
