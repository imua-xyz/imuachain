package aggregator

import (
	"math/big"

	"github.com/ExocoreNetwork/exocore/x/oracle/keeper/common"
	"github.com/ExocoreNetwork/exocore/x/oracle/types"
)

type confirmedPrice struct {
	sourceID  uint64
	detID     string
	price     string
	timestamp string
}

// internal struct
type priceAndPower struct {
	price string
	power *big.Int
}

// for a specific DS round, it could have multiple values provided by different validators(should not be true if there's no malicious validator)
type roundPrices struct { // 0 means NS
	detID     string
	prices    []*priceAndPower
	price     string
	timestamp string
	// confirmed bool
}

// udpate priceAndPower for a specific DSRoundID, if the price exists, increase its power with provided data
// return confirmed=true, when detect power exceeds the threshold
func (r *roundPrices) updatePriceAndPower(pw *priceAndPower, totalPower *big.Int) (updated bool, confirmed bool) {
	if len(r.price) > 0 {
		confirmed = true
		return
	}
	for _, item := range r.prices {
		if item.price == pw.price {
			item.power = new(big.Int).Add(item.power, pw.power)
			updated = true
			if common.ExceedsThreshold(item.power, totalPower) {
				r.price = item.price
				confirmed = true
			}
			return
		}
	}
	if len(r.prices) < cap(r.prices) {
		r.prices = append(r.prices, pw)
		updated = true
		if common.ExceedsThreshold(pw.power, totalPower) {
			r.price = pw.price
			confirmed = true
		}
	}
	return
}

// each DS corresponding a roundPriceList to represent its multiple rounds(DS round) in one oracle-round
type roundPricesList struct {
	roundPricesList []*roundPrices
	// each round can have at most roundPricesCount priceAndPower
	roundPricesCount int
}

func (r *roundPricesList) copy4CheckTx() *roundPricesList {
	ret := &roundPricesList{
		roundPricesList:  make([]*roundPrices, 0, len(r.roundPricesList)),
		roundPricesCount: r.roundPricesCount,
	}

	for _, v := range r.roundPricesList {
		tmpRP := &roundPrices{
			detID:     v.detID,
			price:     v.price,
			prices:    make([]*priceAndPower, 0, len(v.prices)),
			timestamp: v.timestamp,
		}
		for _, pNP := range v.prices {
			tmpPNP := *pNP
			// power will be modified during execution
			tmpPNP.power = copyBigInt(pNP.power)
			tmpRP.prices = append(tmpRP.prices, &tmpPNP)
		}

		ret.roundPricesList = append(ret.roundPricesList, tmpRP)
	}
	return ret
}

// to tell if any round of this DS has reached consensus/confirmed
func (r *roundPricesList) hasConfirmedDetID() bool {
	for _, round := range r.roundPricesList {
		if len(round.price) > 0 {
			return true
		}
	}
	return false
}

// get the roundPriceList correspond to specifid detID of a DS
// if no required data and the pricesList has not reach its limitation, we will add a new slot for this detId
func (r *roundPricesList) getOrNewRound(detID string, timestamp string) (round *roundPrices) {
	for _, round = range r.roundPricesList {
		if round.detID == detID {
			if len(round.price) > 0 {
				round = nil
			}
			return
		}
	}

	if len(r.roundPricesList) < cap(r.roundPricesList) {
		round = &roundPrices{
			detID:     detID,
			prices:    make([]*priceAndPower, 0, r.roundPricesCount),
			timestamp: timestamp,
		}
		r.roundPricesList = append(r.roundPricesList, round)
		return
	}
	return
}

// calculator used to get consensus on deterministic source based data from validator set reports of price
type calculator struct {
	// sourceId->{[]{roundId, []{price,power}, confirmed}}, confirmed value will be set in [0]
	deterministicSource map[uint64]*roundPricesList
	validatorLength     int
	totalPower          *big.Int
}

func (c *calculator) copy4CheckTx() *calculator {
	ret := newCalculator(c.validatorLength, c.totalPower)

	// copy deterministicSource
	for k, v := range c.deterministicSource {
		ret.deterministicSource[k] = v.copy4CheckTx()
	}

	return ret
}

func (c *calculator) newRoundPricesList() *roundPricesList {
	return &roundPricesList{
		roundPricesList: make([]*roundPrices, 0, int(common.MaxDetID)*c.validatorLength),
		// for each DS-roundId, the count of prices provided is the number of validators at most
		roundPricesCount: c.validatorLength,
	}
}

func (c *calculator) getOrNewSourceID(sourceID uint64) *roundPricesList {
	rounds := c.deterministicSource[sourceID]
	if rounds == nil {
		rounds = c.newRoundPricesList()
		c.deterministicSource[sourceID] = rounds
	}
	return rounds
}

// fillPrice called upon new MsgCreatPrice arrived, to trigger the calculation to get to consensus on the same roundID_of_deterministic_source
// v1 use mode1, TODO: switch modes
func (c *calculator) fillPrice(pSources []*types.PriceSource, _ string, power *big.Int) (confirmedRounds []*confirmedPrice) {
	for _, pSource := range pSources {
		rounds := c.getOrNewSourceID(pSource.SourceID)
		if rounds.hasConfirmedDetID() {
			// TODO: this skip is just for V1 to do fast calculation and release EndBlocker pressure, may lead to 'not latest detId' be chosen
			break
		}
		for _, pDetID := range pSource.Prices {
			round := rounds.getOrNewRound(pDetID.DetID, pDetID.Timestamp)
			if round == nil {
				// this sourceId has reach the limitation of different detId, or has confirmed
				continue
			}

			updated, confirmed := round.updatePriceAndPower(&priceAndPower{pDetID.Price, power}, c.totalPower)
			if updated && confirmed {
				// sourceId, detId, price
				confirmedRounds = append(confirmedRounds, &confirmedPrice{pSource.SourceID, round.detID, round.price, round.timestamp}) // TODO: just in v1 with mode==1, we use asap, so we just ignore any further data from this DS, even higher detId may get to consensus, in this way, in most case, we can complete the calculation in the transaction execution process. Release the pressure in EndBlocker
				// TODO: this may delay to current block finish
				break
			}
		}
	}
	return
}

func newCalculator(validatorSetLength int, totalPower *big.Int) *calculator {
	return &calculator{
		deterministicSource: make(map[uint64]*roundPricesList),
		validatorLength:     validatorSetLength,
		totalPower:          totalPower,
	}
}
