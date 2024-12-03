package aggregator

import (
	"github.com/ExocoreNetwork/exocore/x/oracle/keeper/common"
	"github.com/ExocoreNetwork/exocore/x/oracle/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// worker is the actual instance used to calculate final price for each tokenFeeder's round. Which means, every tokenFeeder corresponds to a specified token, and for that tokenFeeder, each round we use a worker instance to calculate the final price
type worker struct {
	sealed  bool
	price   string
	decimal int32
	// mainly used for deterministic source data to check conflicts and validation
	f *filter
	// used to get to consensus on deterministic source's data
	c *calculator
	// when enough data(exceeds threshold) collected, aggregate to conduct the final price
	a   *aggregator
	ctx *AggregatorContext
	// TODO: move outside into context through .ctx
	records recordMsg
}

// recordKey used to retrieve messages from records to evaluate that if a validator report proper price for a specific feederID+sourceID
type recordKey struct {
	validator string
	feederID  uint64
	sourceID  uint64
}

// recordMsg define wrap the map for fast access to validator's message info
type recordMsg map[recordKey][]*types.PriceTimeDetID

func newRecordMsg() recordMsg {
	return make(map[recordKey][]*types.PriceTimeDetID)
}

func (r recordMsg) get(validator string, feederID, sourceID uint64) []*types.PriceTimeDetID {
	v := r[recordKey{validator, feederID, sourceID}]
	return v
}

func (r recordMsg) check(validator string, feederID, sourceID uint64, price, detID string) (exist, matched bool) {
	prices := r.get(validator, feederID, sourceID)
	for _, p := range prices {
		if p.DetID == detID {
			exist = true
			if p.Price == price {
				matched = true
				return
			}
		}
	}
	return
}

func (r recordMsg) set(creator string, feederID uint64, priceSources []*types.PriceSource) {
	accAddress, _ := sdk.AccAddressFromBech32(creator)
	validator := sdk.ConsAddress(accAddress).String()
	for _, price := range priceSources {
		r[recordKey{validator, feederID, price.SourceID}] = price.Prices
	}
}

// GetFinalPriceList relies requirement to aggregator inside them to get final price list
// []{feederID, sourceID, detID, price} in asc order of {soruceID}
func (w *worker) getFinalPriceList(feederID uint64) []*types.AggFinalPrice {
	return w.a.getFinalPriceList(feederID)
}

func (w *worker) filtrate(msg *types.MsgCreatePrice) (list4Calculator []*types.PriceSource, list4Aggregator []*types.PriceSource) {
	return w.f.filtrate(msg)
}

func (w *worker) recordMessage(creator string, feederID uint64, priceSources []*types.PriceSource) {
	w.records.set(creator, feederID, priceSources)
}

func (w *worker) check(validator string, feederID, sourceID uint64, price, detID string) (exist, matched bool) {
	return w.records.check(validator, feederID, sourceID, price, detID)
}

func (w *worker) do(msg *types.MsgCreatePrice) []*types.PriceSource {
	list4Calculator, list4Aggregator := w.f.filtrate(msg)
	if list4Aggregator != nil {
		accAddress, _ := sdk.AccAddressFromBech32(msg.Creator)
		validator := sdk.ConsAddress(accAddress).String()
		power := w.ctx.validatorsPower[validator]
		w.a.fillPrice(list4Aggregator, validator, power)
		if confirmedRounds := w.c.fillPrice(list4Calculator, validator, power); confirmedRounds != nil {
			w.a.confirmDSPrice(confirmedRounds)
		}
	}
	return list4Aggregator
}

func (w *worker) aggregate() string {
	return w.a.aggregate()
}

// not concurrency safe
func (w *worker) seal() {
	if w.sealed {
		return
	}
	w.sealed = true
	w.price = w.a.aggregate()
}

// newWorker new a instance for a tokenFeeder's specific round
func newWorker(feederID uint64, agc *AggregatorContext) *worker {
	return &worker{
		f:       newFilter(int(common.MaxNonce), int(common.MaxDetID)),
		c:       newCalculator(len(agc.validatorsPower), agc.totalPower),
		a:       newAggregator(len(agc.validatorsPower), agc.totalPower),
		decimal: agc.params.GetTokenInfo(feederID).Decimal,
		ctx:     agc,
		records: newRecordMsg(),
	}
}
