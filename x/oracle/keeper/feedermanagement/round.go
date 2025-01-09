package feedermanagement

import (
	"fmt"

	oracletypes "github.com/ExocoreNetwork/exocore/x/oracle/types"
)

func newRound(feederID int64, tokenFeeder *oracletypes.TokenFeeder, quoteWindowSize int64, cache CacheReader) *round {
	return &round{
		startBaseBlock:  int64(tokenFeeder.StartBaseBlock),
		startRoundID:    int64(tokenFeeder.StartRoundID),
		endBlock:        int64(tokenFeeder.EndBlock),
		interval:        int64(tokenFeeder.Interval),
		quoteWindowSize: quoteWindowSize,

		feederID: feederID,
		tokenID:  int64(tokenFeeder.TokenID),
		cache:    cache,

		// default value
		status:         roundStatusClosed,
		a:              nil,
		roundBaseBlock: 0,
		roundID:        0,
	}
}

func (r *round) CopyForCheckTx() *round {
	// flags has been taken care of
	ret := *r
	// cache does not need to be copied since it's a readonly interface,
	// and there's no race condition since abci requests are not executing concurrntly
	ret.a = ret.a.CopyForCheckTx()
	return &ret
}

func (r *round) getMsgItemFromProto(msg *oracletypes.MsgItem) *MsgItem {
	power, _ := r.cache.GetPowerForValidator(msg.Validator)
	priceSources := make([]*priceSource, 0, len(msg.PSources))
	for _, ps := range msg.PSources {
		priceSources = append(priceSources, GetPriceSourceFromProto(ps, r.cache))
	}
	return &MsgItem{
		FeederID:     int64(msg.FeederID),
		Validator:    msg.Validator,
		Power:        power,
		PriceSources: priceSources,
	}
}

func (r *round) ValidQuotingBaseBlock(height int64) bool {
	return r.IsQuotingWindowOpen() && r.roundBaseBlock == height
}

// Tally process information to get the final price
// it does not verify if the msg is for the corresponding round(roundid/roundBaseBlock)
func (r *round) Tally(protoMsg *oracletypes.MsgItem) (*PriceResult, *oracletypes.MsgItem, error) {
	if !r.IsQuotingWindowOpen() {
		return nil, nil, fmt.Errorf("quoting window is not open, feederID:%d", r.feederID)
	}

	msg := r.getMsgItemFromProto(protoMsg)
	if !r.IsQuoting() {
		// record msg for 'handlQuotingMisBehavior'
		err := r.a.RecordMsg(msg)
		if err == nil {
			return nil, protoMsg, oracletypes.ErrQuoteRecorded
		}
		return nil, nil, fmt.Errorf("failed to record quote for aggregated round, error:%w", err)
	}

	err := r.a.AddMsg(msg)
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
	r.startBaseBlock = int64(tokenFeeder.StartBaseBlock)
	r.endBlock = int64(tokenFeeder.EndBlock)
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
	r.a = newAggregator(r.cache.GetThreshold())
}

func (r *round) IsQuotingWindowOpen() bool {
	// either open or committable means the round is inside the living quoting window
	return r.status != roundStatusClosed
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
	miss = true
	detID := r.getFinalDetIDForSourceID(oracletypes.SourceChainlinkID)
	price := finalPrice.PriceInfo()
	price.DetID = detID
	prices, ok := r.a.v.GetValidatorQuotePricesForSourceID(validator, oracletypes.SourceChainlinkID)
	if !ok {
		return
	}
	for _, p := range prices {
		if p.EqualDS(price) {
			miss = false
		} else if p.DetID == price.DetID {
			miss = false
			malicious = true
		}
	}
	return
}

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
	rounds := delta / r.interval
	roundID = r.startRoundID + rounds
	delta -= rounds * r.interval
	baseBlock = currentHeight - delta
	return
}
