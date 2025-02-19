package aggregator

import "github.com/imua-xyz/imuachain/x/oracle/types"

func newPTD(detID, price string) *types.PriceTimeDetID {
	return &types.PriceTimeDetID{
		Price:     price,
		Decimal:   1,
		Timestamp: "-",
		DetID:     detID,
	}
}

func newPS(sourceID uint64, prices ...*types.PriceTimeDetID) *types.PriceSource {
	return &types.PriceSource{
		SourceID: sourceID,
		Prices:   prices,
	}
}
