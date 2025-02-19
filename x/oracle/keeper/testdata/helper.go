package testdata

import "github.com/imua-xyz/imuachain/x/oracle/types"

const t = "2024-05-01 01:01:01"

func newPTD(detID, price string) *types.PriceTimeDetID {
	return &types.PriceTimeDetID{
		Price:     price,
		Decimal:   18,
		Timestamp: t,
		DetID:     detID,
	}
}

func newPS(sourceID uint64, prices ...*types.PriceTimeDetID) *types.PriceSource {
	return &types.PriceSource{
		SourceID: sourceID,
		Prices:   prices,
	}
}

func newPTR(price string, roundID uint64) *types.PriceTimeRound {
	return &types.PriceTimeRound{
		Price:     price,
		Decimal:   18,
		Timestamp: t,
		RoundID:   roundID,
	}
}

func newPrices(tokenID uint64, nextRoundID uint64, pList ...*types.PriceTimeRound) types.Prices {
	return types.Prices{
		TokenID:     tokenID,
		NextRoundID: nextRoundID,
		PriceList:   pList,
	}
}
