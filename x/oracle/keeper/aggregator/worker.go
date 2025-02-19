package aggregator

import (
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/imua-xyz/imuachain/x/oracle/keeper/common"
	"github.com/imua-xyz/imuachain/x/oracle/types"
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
}

func (w *worker) do(msg *types.MsgCreatePrice) []*types.PriceSource {
	accAddress, _ := sdk.AccAddressFromBech32(msg.Creator)
	validator := sdk.ConsAddress(accAddress).String()
	power := w.ctx.validatorsPower[validator]
	list4Calculator, list4Aggregator := w.f.filtrate(msg)
	if list4Aggregator != nil {
		w.a.fillPrice(list4Aggregator, validator, power)
		if confirmedRounds := w.c.fillPrice(list4Calculator, validator, power); confirmedRounds != nil {
			w.a.confirmDSPrice(confirmedRounds)
		}
	}
	return list4Aggregator
}

func (w *worker) aggregate() *big.Int {
	return w.a.aggregate()
}

// not concurrency safe
func (w *worker) seal() {
	if w.sealed {
		return
	}
	w.sealed = true
	w.price = w.a.aggregate().String()
	w.f = nil
	w.c = nil
	w.a = nil
}

// newWorker new a instance for a tokenFeeder's specific round
func newWorker(feederID uint64, agc *AggregatorContext) *worker {
	return &worker{
		f:       newFilter(int(common.MaxNonce), int(common.MaxDetID)),
		c:       newCalculator(len(agc.validatorsPower), agc.totalPower),
		a:       newAggregator(len(agc.validatorsPower), agc.totalPower),
		decimal: agc.params.GetTokenInfo(feederID).Decimal,
		ctx:     agc,
	}
}
