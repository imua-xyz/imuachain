package feedermanagement

import (
	"math/big"
	"sort"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/imua-xyz/imuachain/x/oracle/keeper/common"
	oracletypes "github.com/imua-xyz/imuachain/x/oracle/types"
)

type Submitter interface {
	SetValidatorUpdateForCache(sdk.Context, oracletypes.ValidatorUpdateBlock)
	SetParamsForCache(sdk.Context, oracletypes.RecentParams)
	SetMsgItemsForCache(sdk.Context, oracletypes.RecentMsg)
}

type CacheReader interface {
	GetPowerForValidator(validator string) (*big.Int, bool)
	GetTotalPower() (totalPower *big.Int)
	GetValidators() []string
	IsRuleV1(feederID int64) bool
	IsDeterministic(sourceID int64) (bool, error)
	GetThreshold() *threshold
}

// used to track validator change
type cacheValidator struct {
	validators map[string]*big.Int
	update     bool
	totalPower *big.Int
}

// used to track params change
type cacheParams struct {
	params *oracletypes.Params
	update bool
}

type cacheMsgs []*oracletypes.MsgItem

type caches struct {
	k Submitter

	msg        *cacheMsgs
	validators *cacheValidator
	params     *cacheParams
}

type MsgItem struct {
	FeederID     int64
	Validator    string
	Power        *big.Int
	PriceSources []*priceSource
}

// PriceInfo is the price information including price, decimal, time, and detID
// this is defined as a internal type as alias of oracletypes.PriceTimeDetID
type PriceInfo oracletypes.PriceTimeDetID

// PricePower wraps PriceInfo with power and validators
// Validators indicates the validators who provided the price
// Power indicates the accumulated power of all validators who provided the price
type PricePower struct {
	Price      *PriceInfo
	Power      *big.Int
	Validators map[string]struct{}
}

// PriceResult is the final price information including price, decimal, time, and detID
// this is defined as a internal type as alias of PriceInfo
type PriceResult PriceInfo

// PriceSource describes a specific source of related prices
// deteministic indicates whether the source is deterministic
// finalPrice indicates the final price of the source aggregated from all prices
// sourceID indicates the source ID
// detIDs indicates the detIDs of all prices, detID means the unique identifier of a price defined in the deterministic itself
// prices indicates all prices of the source, it's defined in slice instead of map to keep the order
type priceSource struct {
	deterministic bool
	finalPrice    *PriceResult
	sourceID      int64
	detIDs        map[string]struct{}
	// ordered by detID
	prices []*PriceInfo
}

// priceValiadtor describes a specific validator of related priceSources(each source could has multiple prices)
// finalPrice indicates the final price of the validator aggregated from all prices of each source
// validator indicates the validator address
// power indicates the power of the validator
// priceSources indicates all priceSources of the validator, the map key is sourceID
type priceValidator struct {
	finalPrice *PriceResult
	validator  string
	power      *big.Int
	// each source will get a single final price independently, the order of sources does not matter, map is safe
	priceSources map[int64]*priceSource
}

// recordsValidators is the price records for all validators, the record item is priceValidator
// finalPrice indicates the final price of all validators aggregated from all prices of each validator
// finalPrices indicates the final price of each validator aggregated from all of its prices of each source
// accumulatedPower indicates the accumulated power of all validators
// records indicates all priceValidators, the map key is validator address
type recordsValidators struct {
	finalPrice  *PriceResult
	finalPrices map[string]*PriceResult
	// TODO: V2: accumulatedValidPower only includes validators who providing all sources required by rules(defined in oracle.Params)
	// accumulatedValidVpower: map[string]*big.Int
	accumulatedPower *big.Int
	// each validator will get a single final price independently, the order of validators does not matter, map is safe
	records map[string]*priceValidator
}

// recordsDS is the price records of a deterministic source
// finalPrice indicates the final price of a specific detID chosen out of all prices with different detIDs which pass the threshold
// finalDetID indicates the final detID chosen out of all detIDs
// accumulatedPowers indicates the accumulated power of all validators who provided the price for this source
// valiators indicates all validators who provided the price for this source
// records indicates all PricePower of this source provided by different validators, the slice is ordered by detID
type recordsDS struct {
	finalPrice        *PriceResult
	finalDetID        string
	accumulatedPowers *big.Int
	validators        map[string]struct{}
	// ordered by detID
	records []*PricePower
}

// each source will get a final price independently, the order of sources does not matter, map is safe
// recordsDSs is the price records for all deterministic sources
// threshold indicates the threshold defined to decide final price for each source
type recordsDSs struct {
	t     *threshold
	dsMap map[int64]*recordsDS
}

// threshold is defined as (thresholdA * thresholdB) / totalPower
// when do compare with power, it should be (thresholdB * power) > (thresholdA * totalPower) to avoid decimal calculation
type threshold struct {
	totalPower *big.Int
	thresholdA *big.Int
	thresholdB *big.Int
}

func (t *threshold) Equals(t2 *threshold) bool {
	if t == nil || t2 == nil {
		return t == t2
	}
	return t.totalPower.Cmp(t2.totalPower) == 0 && t.thresholdA.Cmp(t2.thresholdA) == 0 && t.thresholdB.Cmp(t2.thresholdB) == 0
}

func (t *threshold) Cpy() *threshold {
	return &threshold{
		totalPower: new(big.Int).Set(t.totalPower),
		thresholdA: new(big.Int).Set(t.thresholdA),
		thresholdB: new(big.Int).Set(t.thresholdB),
	}
}

func (t *threshold) Exceeds(power *big.Int) bool {
	return new(big.Int).Mul(t.thresholdB, power).Cmp(new(big.Int).Mul(t.thresholdA, t.totalPower)) > 0
}

// aggregator is the price aggregator for a specific round
// t is the threshold definition for price consensus for the round
// finalPrice indicates the final price of the round
// v is the price records for all validators with prices they provided
// ds is the price records for all deterministic sources which could have prices with multiple detIDs provided by validators
// algo is the aggregation algorithm for the round, currently we use 'Median' as default
type aggregator struct {
	t          *threshold
	finalPrice *PriceResult
	v          *recordsValidators
	ds         *recordsDSs
	algo       AggAlgorithm
}

// roundStatus indicates the status of a round
type roundStatus int32

const (
	// define closed as default value 0
	// close: the round is closed for price submission and no valid price to commit
	roundStatusClosed roundStatus = iota
	// open: the round is open for price submission
	roundStatusOpen
	// committable: the round is closed for price submission and available for price to commit
	roundStatusCommittable
)

// round is the price round for a specific tokenFeeder corresponding to the price feed progress of a specific token
type round struct {
	// startBaseBlock is the start block height of corresponding tokenFeeder
	startBaseBlock int64
	// startRoundID is the round ID of corresponding tokenFeeder
	startRoundID int64
	// endBlock is the end block height of the corresponding tokenFeeder
	endBlock int64
	// interval is the interval of the corresponding tokenFeeder
	interval int64
	// quoteWindowSize is the quote window size of the corresponding tokenFeeder
	quoteWindowSize int64

	// feederID is the feeder ID of the corresponding tokenFeeder
	feederID int64
	// tokenID is the token ID of the corresponding tokenFeeder
	tokenID int64

	// roundBaseBlock is the round base block of current round
	roundBaseBlock int64
	// roundID is the round ID of current round
	roundID int64
	// status indicates the status of current round
	status roundStatus
	// aggregator is the price aggregator for current round
	a *aggregator
	// cache is the cache reader for current round to provide params, validators information
	cache CacheReader
	// algo is the aggregation algorithm for current round to get final price
	algo AggAlgorithm
}

type orderedSliceInt64 []int64

func (osi orderedSliceInt64) Equals(o2 orderedSliceInt64) bool {
	if len(osi) == 0 || len(o2) == 0 {
		return len(osi) == len(o2)
	}

	for idx, v := range osi {
		if v != (o2)[idx] {
			return false
		}
	}
	return true
}

func (osi *orderedSliceInt64) add(i int64) {
	result := append(*osi, i)
	sort.Slice(result, func(i, j int) bool {
		return result[i] < result[j]
	})
	*osi = result
}

func (osi *orderedSliceInt64) remove(i int64) {
	for idx, v := range *osi {
		if v == i {
			*osi = append((*osi)[:idx], (*osi)[idx+1:]...)
			return
		}
	}
}

func (osi *orderedSliceInt64) sort() {
	sort.Slice(*osi, func(i, j int) bool {
		return (*osi)[i] < (*osi)[j]
	})
}

// FeederManager is the manager for the price feed progress of all token feeders
type FeederManager struct {
	// fCheckTx is a copy of FeederManager used for mode of checkTx to simulate transactions
	fCheckTx *FeederManager
	// k is the oracle keeper
	k common.KeeperOracle
	// sortedFeederIDs is the ordered feeder IDs corresponding to all the rounds included in FeederManager
	sortedFeederIDs orderedSliceInt64
	// rounds is the map of all rounds included in FeederManager, the key is the feeder ID
	// TODO: change type of key from int64 to uint64
	rounds map[int64]*round
	cs     *caches
	// paramsUpdated indicates whether the params are updated in current block
	paramsUpdated bool
	// validatorsUpdated indicates whether the validators are updated in current block
	validatorsUpdated bool
	// forceSeal indicates whether it's satisfied to force seal all the rounds
	// when the validators are updated in current block or some related params are updated, this will be set to true.
	forceSeal bool
	// restSlashing indicates whether it's satisfied to reset slashing
	// when the slashing params is changed in current block, this will be set to true.
	resetSlashing bool
}
