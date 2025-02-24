package feedermanagement

import (
	"fmt"

	oracletypes "github.com/imua-xyz/imuachain/x/oracle/types"
)

func newRound(feederID int64, tokenFeeder *oracletypes.TokenFeeder, quoteWindowSize int64, cache CacheReader, algo AggAlgorithm) *round {
	return &round{
		// #nosec G115
		startBaseBlock: int64(tokenFeeder.StartBaseBlock),
		// #nosec G115
		startRoundID: int64(tokenFeeder.StartRoundID),
		// #nosec G115
		endBlock: int64(tokenFeeder.EndBlock),
		// #nosec G115
		interval:        int64(tokenFeeder.Interval),
		quoteWindowSize: quoteWindowSize,
		feederID:        feederID,
		// #nosec G115
		tokenID: int64(tokenFeeder.TokenID),
		cache:   cache,

		// default value
		status:         roundStatusClosed,
		a:              nil,
		roundBaseBlock: 0,
		roundID:        0,
		algo:           algo,
	}
}

func (r *round) Equals(r2 *round) bool {
	if r == nil || r2 == nil {
		return r == r2
	}

	if r.startBaseBlock != r2.startBaseBlock ||
		r.startRoundID != r2.startRoundID ||
		r.endBlock != r2.endBlock ||
		r.interval != r2.interval ||
		r.quoteWindowSize != r2.quoteWindowSize ||
		r.feederID != r2.feederID ||
		r.tokenID != r2.tokenID ||
		r.roundBaseBlock != r2.roundBaseBlock ||
		r.roundID != r2.roundID ||
		r.status != r2.status {
		return false
	}
	if !r.a.Equals(r2.a) {
		return false
	}

	return true
}

func (r *round) CopyForCheckTx() *round {
	// flags has been taken care of
	ret := *r
	// cache does not need to be copied since it's a readonly interface,
	// and there's no race condition since abci requests are not executing concurrntly
	ret.a = ret.a.CopyForCheckTx()
	return &ret
}

func (r *round) getMsgItemFromProto(msg *oracletypes.MsgItem) (*MsgItem, error) {
	power, found := r.cache.GetPowerForValidator(msg.Validator)
	if !found {
		return nil, fmt.Errorf("failed to get power for validator:%s", msg.Validator)
	}
	priceSources := make([]*priceSource, 0, len(msg.PSources))
	for _, ps := range msg.PSources {
		psNew, err := getPriceSourceFromProto(ps, r.cache)
		if err != nil {
			return nil, err
		}
		priceSources = append(priceSources, psNew)
	}
	return &MsgItem{
		// #nosec G115
		FeederID:     int64(msg.FeederID),
		Validator:    msg.Validator,
		Power:        power,
		PriceSources: priceSources,
	}, nil
}

func (r *round) ValidQuotingBaseBlock(height int64) bool {
	return r.IsQuotingWindowOpen() && r.roundBaseBlock == height
}

// Tally process information to get the final price
// it does not verify if the msg is for the corresponding round(roundid/roundBaseBlock)
// TODO: use valid value instead of the original protoMsg in return
func (r *round) Tally(protoMsg *oracletypes.MsgItem) (*PriceResult, *oracletypes.MsgItem, error) {
	if !r.IsQuotingWindowOpen() {
		return nil, nil, fmt.Errorf("quoting window is not open, feederID:%d", r.feederID)
	}

	msg, err := r.getMsgItemFromProto(protoMsg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get msgItem from proto, error:%w", err)
	}
	if !r.IsQuoting() {
		// record msg for 'handlQuotingMisBehavior'
		err := r.a.RecordMsg(msg)
		if err == nil {
			return nil, protoMsg, oracletypes.ErrQuoteRecorded
		}
		return nil, nil, fmt.Errorf("failed to record quote for aggregated round, error:%w", err)
	}

	err = r.a.AddMsg(msg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to add quote for aggregation of feederID:%d, roundID:%d, error:%w", r.feederID, r.roundID, err)
	}

	finalPrice, ok := r.FinalPrice()
	if ok {
		r.status = roundStatusCommittable
		// NOTE: for V1, we need return the DetID as well since chainlink is the only source
		if r.cache.IsRuleV1(r.feederID) {
			detID := r.getFinalDetIDForSourceID(oracletypes.SourceChainlinkID)
			finalPrice.DetID = detID
		}
		return finalPrice, protoMsg, nil
	}

	return nil, protoMsg, nil
}

func (r *round) UpdateParams(tokenFeeder *oracletypes.TokenFeeder, quoteWindowSize int64) {
	// #nosec G115
	r.startBaseBlock = int64(tokenFeeder.StartBaseBlock)
	// #nosec G115
	r.endBlock = int64(tokenFeeder.EndBlock)
	// #nosec G115
	r.interval = int64(tokenFeeder.Interval)
	r.quoteWindowSize = quoteWindowSize
}

// PrepareForNextBlock sets status to Open and create a new aggregator on the block before the first block of quoting
func (r *round) PrepareForNextBlock(currentHeight int64) (open bool) {
	if currentHeight < r.roundBaseBlock && r.IsQuoting() {
		r.closeQuotingWindow()
		return open
	}
	// currentHeight euqls to baseBlock
	if currentHeight == r.roundBaseBlock && !r.IsQuoting() {
		r.openQuotingWindow()
		open = true
		return open
	}
	baseBlock, roundID, delta, expired := r.getPosition(currentHeight)

	if expired && r.IsQuoting() {
		r.closeQuotingWindow()
		return open
	}
	// open a new round
	if baseBlock > r.roundBaseBlock {
		// move to next round
		r.roundBaseBlock = baseBlock
		r.roundID = roundID
		// the first block in the quoting window
		if delta == 0 && !r.IsQuoting() {
			r.openQuotingWindow()
			open = true
		}
	}
	return open
}

func (r *round) openQuotingWindow() {
	r.status = roundStatusOpen
	r.a = newAggregator(r.cache.GetThreshold(), r.algo)
}

// IsQuotingWindowOpen returns if the round is inside its current quoting window including status of {open, committable, close}
func (r *round) IsQuotingWindowOpen() bool {
	// aggregator is set when quoting window open and removed when the window reaches the end or be force sealed
	return r.a != nil
}

func (r *round) IsQuotingWindowEnd(currentHeight int64) bool {
	_, _, delta, _ := r.getPosition(currentHeight)
	return delta == r.quoteWindowSize
}

func (r *round) IsQuoting() bool {
	return r.status == roundStatusOpen
}

func (r *round) FinalPrice() (*PriceResult, bool) {
	if r.a == nil {
		return nil, false
	}
	return r.a.GetFinalPrice()
}

// Close sets round status to roundStatusClosed and remove current aggregator
func (r *round) closeQuotingWindow() {
	r.status = roundStatusClosed
	r.a = nil
}

func (r *round) PerformanceReview(validator string) (miss, malicious bool) {
	finalPrice, ok := r.FinalPrice()
	if !ok {
		return
	}
	if !r.cache.IsRuleV1(r.feederID) {
		// only rulev1 is supported for now
		return
	}
	detID := r.getFinalDetIDForSourceID(oracletypes.SourceChainlinkID)
	price := finalPrice.PriceInfo()
	price.DetID = detID
	prices, ok := r.a.v.GetValidatorQuotePricesForSourceID(validator, oracletypes.SourceChainlinkID)
	if !ok {
		miss = true
		return
	}
	for _, p := range prices {
		if p.EqualDS(price) {
			// duplicated detID had been filtered out, so if an 'euqal' price is found, there will be no 'malicous' price for that detID
			return
		}
		if p.DetID == price.DetID {
			malicious = true
			return
		}
	}
	miss = true
	return
}

//nolint:unparam
func (r *round) getFinalDetIDForSourceID(sourceID int64) string {
	return r.a.ds.GetFinalDetIDForSourceID(sourceID)
}

func (r *round) Committable() bool {
	return r.status == roundStatusCommittable
}

func (r *round) getPosition(currentHeight int64) (baseBlock, roundID, delta int64, expired bool) {
	// endBlock is included
	if r.endBlock > 0 && currentHeight > r.endBlock {
		expired = true
		return
	}
	if currentHeight < r.startBaseBlock {
		return
	}
	delta = currentHeight - r.startBaseBlock
	if r.interval == 0 {
		return
	}
	rounds := delta / r.interval
	roundID = r.startRoundID + rounds
	delta -= rounds * r.interval
	baseBlock = currentHeight - delta
	return
}
