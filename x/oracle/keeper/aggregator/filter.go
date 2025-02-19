package aggregator

import (
	"strconv"

	"github.com/imua-xyz/imuachain/x/oracle/keeper/common"
	"github.com/imua-xyz/imuachain/x/oracle/types"
)

type filter struct {
	maxNonce int
	maxDetID int
	// nonce start from 1
	validatorNonce map[string]*common.Set[int32]
	// validator_sourceId -> roundID, NS use 0
	validatorSource map[string]*common.Set[string]
}

func newFilter(maxNonce, maxDetID int) *filter {
	return &filter{
		maxNonce:        maxNonce,
		maxDetID:        maxDetID,
		validatorNonce:  make(map[string]*common.Set[int32]),
		validatorSource: make(map[string]*common.Set[string]),
	}
}

func (f *filter) copy4CheckTx() *filter {
	ret := *f
	ret.validatorNonce = make(map[string]*common.Set[int32], len(f.validatorNonce))
	ret.validatorSource = make(map[string]*common.Set[string], len(f.validatorSource))

	for k, v := range f.validatorNonce {
		ret.validatorNonce[k] = v.Copy()
	}

	for k, v := range f.validatorSource {
		ret.validatorSource[k] = v.Copy()
	}

	return &ret
}

func (f *filter) newVNSet() *common.Set[int32] {
	return common.NewSet[int32](f.maxNonce)
}

func (f *filter) newVSSet() *common.Set[string] {
	return common.NewSet[string](f.maxDetID)
}

// add priceWithSource into calculator list and aggregator list depends on the source type(deterministic/non-deterministic)
func (f *filter) addPSource(pSources []*types.PriceSource, validator string) (list4Calculator []*types.PriceSource, list4Aggregator []*types.PriceSource) {
	for _, pSource := range pSources {
		// check conflicts or duplicate data for the same roundID within the same source
		if len(pSource.Prices[0].DetID) > 0 {
			// #nosec G115
			k := validator + strconv.Itoa(int(pSource.SourceID))
			detIDs := f.validatorSource[k]
			if detIDs == nil {
				detIDs = f.newVSSet()
				f.validatorSource[k] = detIDs
			}

			pSourceTmp := &types.PriceSource{
				SourceID: pSource.SourceID,
				Prices:   make([]*types.PriceTimeDetID, 0, len(pSource.Prices)),
				Desc:     pSource.Desc,
			}

			for _, pDetID := range pSource.Prices {
				if ok := detIDs.Add(pDetID.DetID); ok {
					// deterministic id has not seen in filter and limitation of ids this souce has not reached
					pSourceTmp.Prices = append(pSourceTmp.Prices, pDetID)
				}
			}
			if len(pSourceTmp.Prices) > 0 {
				list4Calculator = append(list4Calculator, pSourceTmp)
				list4Aggregator = append(list4Aggregator, pSourceTmp)
			}
		} else {
			// add non-deterministic pSource value into aggregator list
			list4Aggregator = append(list4Aggregator, pSource)
		}
	}
	return list4Calculator, list4Aggregator
}

// filtrate checks data from MsgCreatePrice, and will drop the conflict or duplicate data, it will then fill data into calculator(for deterministic source data to get to consensus) and aggregator (for both deterministic and non0-deterministic source data run 2-layers aggregation to get the final price)
func (f *filter) filtrate(price *types.MsgCreatePrice) (list4Calculator []*types.PriceSource, list4Aggregator []*types.PriceSource) {
	validator := price.Creator
	nonces := f.validatorNonce[validator]
	if nonces == nil {
		nonces = f.newVNSet()
		f.validatorNonce[validator] = nonces
	}

	if ok := nonces.Add(price.Nonce); ok {
		list4Calculator, list4Aggregator = f.addPSource(price.Prices, validator)
	}
	return
}
