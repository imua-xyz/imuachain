package feedermanagement

import (
	"math/big"

	"github.com/ExocoreNetwork/exocore/x/oracle/keeper/common"
	oracletypes "github.com/ExocoreNetwork/exocore/x/oracle/types"
	"github.com/cometbft/cometbft/libs/log"
	sdk "github.com/cosmos/cosmos-sdk/types"
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
	IsDeterministic(sournceID int64) bool
	GetThreshold() *threshold
}

// used to track validator change
type cacheValidator struct {
	validators map[string]*big.Int
	update     bool
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

type PriceInfo oracletypes.PriceTimeDetID

type PricePower struct {
	Price      *PriceInfo
	Power      *big.Int
	Validators map[string]struct{}
}

// type PriceResult oracletypes.PriceTimeRound
type PriceResult PriceInfo

type priceSource struct {
	deterministic bool
	finalPrice    *PriceResult
	sourceID      int64
	detIDs        map[string]struct{}
	// ordered by detID
	prices []*PriceInfo
}

type priceValidator struct {
	finalPrice *PriceResult
	validator  string
	power      *big.Int
	// each source will get a single final price independetly, the order of sources does not matter, map is safe
	priceSources map[int64]*priceSource
}

type recordsValidators struct {
	finalPrice  *PriceResult
	finalPrices map[string]*PriceResult
	// TODO: V2: accumulatedValidPower only includes validators who prividing all sources requred by rules(defined in oracle.Params)
	// accumulatedValidVpower: map[string]*big.Int
	accumulatedPower *big.Int
	// each validator will get a single final price independently, the order of validators does not matter, map is safe
	records map[string]*priceValidator
}

// price records for deteministic source
type recordsDS struct {
	finalPrice *PriceResult
	// TODO: remove this
	finalDetID        string
	accumulatedPowers *big.Int
	validators        map[string]struct{}
	// ordered by detID
	records []*PricePower
}

// each source will get a final price independently, the order of sources does not matter, map is safe
// type recordsDSMap map[int64]*recordsDS
type recordsDSs struct {
	t     *threshold
	dsMap map[int64]*recordsDS
}

type threshold struct {
	totalPower *big.Int
	thresholdA *big.Int
	thresholdB *big.Int
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

type aggregator struct {
	t          *threshold
	finalPrice *PriceResult
	v          *recordsValidators
	ds         *recordsDSs
}
type roundStatus int32

const (
	// define closed as default value 0
	roundStatusClosed roundStatus = iota
	roundStatusOpen
	roundStatusCommittable
)

type round struct {
	startBaseBlock  int64
	startRoundID    int64
	endBlock        int64
	interval        int64
	quoteWindowSize int64

	feederID int64
	tokenID  int64

	roundBaseBlock int64
	roundID        int64
	status         roundStatus
	a              *aggregator
	cache          CacheReader
}

type FeederManager struct {
	logger          log.Logger
	k               common.KeeperOracle
	sortedFeederIDs []int64
	// this will not be ranged, map is safe
	rounds            map[int64]*round
	cs                *caches
	successFeederIDs  []int64
	paramsUpdated     bool
	validatorsUpdated bool
	forceSeal         bool
	resetSlashing     bool
}
