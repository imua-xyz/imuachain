package feedermanagement

import (
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"slices"

	oracletypes "github.com/ExocoreNetwork/exocore/x/oracle/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
)

type ItemV map[string]*big.Int

const v1RuleID = 1

func (c *caches) CpyForSimulation() *caches {
	ret := *c
	msg := *(c.msg)
	params := *(c.params)
	// it's safe to do shallow copy on msg, params
	ret.msg = &msg
	ret.params = &params
	validators := make(map[string]*big.Int)
	// safe to range map, map copy
	for v, p := range c.validators.validators {
		validators[v] = new(big.Int).Set(p)
	}
	ret.validators = &cacheValidator{
		validators: validators,
		update:     c.validators.update,
		totalPower: new(big.Int).Set(c.validators.totalPower),
	}

	return &ret
}

func (c *caches) Equals(c2 *caches) bool {
	if c == nil || c2 == nil {
		return c == c2
	}
	if !c.msg.Equals(c2.msg) {
		return false
	}
	if !c.validators.Equals(c2.validators) {
		return false
	}
	if !c.params.Equals(c2.params) {
		return false
	}
	return true
}

func (c *caches) Init(k Submitter, params *oracletypes.Params, validators map[string]*big.Int) {
	c.ResetCaches()
	c.k = k

	c.params.add(params)

	c.validators.add(validators)
}

func (c *caches) GetDecimalFromFeederID(feederID uint64) (int32, error) {
	p := c.params.params
	if feederID <= 0 || feederID > uint64(len(p.TokenFeeders)) {
		return 0, errors.New("feederID not exists")
	}
	tf := p.TokenFeeders[feederID]
	return p.Tokens[tf.TokenID].Decimal, nil
}

func (c *caches) GetMaxNonce() int32 {
	return c.params.params.GetMaxNonce()
}

func (c *caches) GetMaxSizePrices() int32 {
	return c.params.params.GetMaxSizePrices()
}

func (c *caches) IsDeterministic(sourceID int64) (bool, error) {
	sources := c.params.params.Sources
	if sourceID >= int64(len(sources)) || sourceID <= 0 {
		return false, errors.New("invalid sourceID")
	}
	return sources[sourceID].Deterministic, nil
}

// RuleV1, we restrict the source to be Chainlink and only that source is acceptable
func (c *caches) IsRuleV1(feederID int64) bool {
	ruleID := c.params.params.TokenFeeders[feederID].RuleID
	return ruleID == v1RuleID && len(c.params.params.Sources) == 2 && c.params.params.Sources[1].Name == oracletypes.SourceChainlinkName
}

func (c *caches) GetTokenIDForFeederID(feederID int64) (int64, bool) {
	tf, ok := c.GetTokenFeederForFeederID(feederID)
	if !ok {
		return 0, false
	}
	// #nosec G115  // tokenID is index of slice
	return int64(tf.TokenID), true
}

// GetValidators return current validator set as ordered slice
func (c *caches) GetValidators() []string {
	return c.validators.slice()
}

func (cm *cacheMsgs) Equals(cm2 *cacheMsgs) bool {
	if cm == nil || cm2 == nil {
		return cm == cm2
	}
	for idx, v := range *cm {
		v2 := (*cm2)[idx]
		if !reflect.DeepEqual(v, v2) {
			return false
		}
	}
	return true
}

func (cm *cacheMsgs) Cpy() *cacheMsgs {
	ret := make([]*oracletypes.MsgItem, 0, len(*cm))
	for _, msg := range *cm {
		msgCpy := *msg
		ret = append(ret, &msgCpy)
	}
	cmNew := cacheMsgs(ret)
	return &cmNew
}

func (cm *cacheMsgs) add(item *oracletypes.MsgItem) {
	*cm = append(*cm, item)
}

func (cm *cacheMsgs) commit(ctx sdk.Context, k Submitter) {
	if len(*cm) == 0 {
		return
	}
	recentMsgs := oracletypes.RecentMsg{
		// #nosec G115  // height is not negative
		Block: uint64(ctx.BlockHeight()),
		Msgs:  *cm,
	}

	k.SetMsgItemsForCache(ctx, recentMsgs)

	*cm = make([]*oracletypes.MsgItem, 0)
}

func (cv *cacheValidator) Equals(cv2 *cacheValidator) bool {
	if cv == nil || cv2 == nil {
		return cv == cv2
	}
	if cv.update != cv2.update {
		return false
	}
	if len(cv.validators) != len(cv2.validators) {
		return false
	}
	if cv.totalPower.Cmp(cv2.totalPower) != 0 {
		return false
	}
	// safe to range map, map compare
	for k, v := range cv.validators {
		if v2, ok := cv2.validators[k]; !ok {
			return false
		} else if v.Cmp(v2) != 0 {
			return false
		}
	}
	return true
}

func (cv *cacheValidator) add(validators map[string]*big.Int) {
	// safe to range map, check and update all KVs with another map
	for operator, newPower := range validators {
		power, ok := cv.validators[operator]
		if !ok {
			power = common.Big0
		}
		if power.Cmp(newPower) != 0 {
			cv.update = true
			// only do sub when power>0
			if ok {
				cv.totalPower.Sub(cv.totalPower, power)
			}
			// use < 1 to keep it the same as 'applyValidatorChange' in dogfood
			if newPower.Cmp(common.Big1) < 0 {
				delete(cv.validators, operator)
				continue
			}
			cv.totalPower.Add(cv.totalPower, newPower)
			if !ok {
				cv.validators[operator] = new(big.Int)
			}
			cv.validators[operator].Set(newPower)
		}
	}
}

func (cv *cacheValidator) commit(ctx sdk.Context, k Submitter) {
	if !cv.update {
		return
	}
	// #nosec blockHeight is not negative
	// TODO: consider change the define of all height types in proto to int64(since cosmossdk defined block height as int64) to get avoid all these conversion
	k.SetValidatorUpdateForCache(ctx, oracletypes.ValidatorUpdateBlock{Block: uint64(ctx.BlockHeight())})
	cv.update = false
}

func (cv *cacheValidator) size() int {
	return len(cv.validators)
}

// returned slice is ordered
func (cv *cacheValidator) slice() []string {
	if cv.size() == 0 {
		return nil
	}
	validators := make([]string, 0, cv.size())
	// safe to range map, this range is used to generate a sorted slice
	for validator := range cv.validators {
		validators = append(validators, validator)
	}
	slices.Sort(validators)
	return validators
}

func (cp *cacheParams) Equals(cp2 *cacheParams) bool {
	if cp == nil || cp2 == nil {
		return cp == cp2
	}
	if cp.update != cp2.update {
		return false
	}
	p1 := cp.params
	p2 := cp2.params
	return reflect.DeepEqual(p1, p2)
}

func (cp *cacheParams) add(p *oracletypes.Params) {
	cp.params = p
	cp.update = true
}

func (cp *cacheParams) commit(ctx sdk.Context, k Submitter) {
	if !cp.update {
		return
	}
	k.SetParamsForCache(ctx, oracletypes.RecentParams{
		// #nosec G115 blockheight is not negative
		Block:  uint64(ctx.BlockHeight()),
		Params: cp.params,
	})
	cp.update = false
}

// memory cache
func (c *caches) AddCache(i any) error {
	switch item := i.(type) {
	case *oracletypes.MsgItem:
		c.msg.add(item)
	case *oracletypes.Params:
		c.params.add(item)
	case ItemV:
		c.validators.add(item)
	default:
		return fmt.Errorf("unsuppported caceh type: %T", i)
	}
	return nil
}

// Read reads the cache
func (c *caches) Read(i any) bool {
	switch item := i.(type) {
	case ItemV:
		if item == nil {
			return false
		}
		// safe to range map, map copy
		for addr, power := range c.validators.validators {
			item[addr] = power
		}
		return c.validators.update
	case *oracletypes.Params:
		if item == nil {
			return false
		}
		*item = *c.params.params
		return c.params.update
	case *[]*oracletypes.MsgItem:
		if item == nil {
			return false
		}
		*item = *c.msg
		return len(*c.msg) > 0
	default:
		return false
	}
}

func (c *caches) GetThreshold() *threshold {
	params := &oracletypes.Params{}
	c.Read(params)
	return &threshold{
		totalPower: c.GetTotalPower(),
		thresholdA: big.NewInt(int64(params.ThresholdA)),
		thresholdB: big.NewInt(int64(params.ThresholdB)),
	}
}

// GetPowerForValidator returns the power of a validator
func (c *caches) GetPowerForValidator(validator string) (power *big.Int, found bool) {
	if c.validators != nil &&
		len(c.validators.validators) > 0 {
		power = c.validators.validators[validator]
		if power != nil {
			found = true
		}
	}
	// if caches not filled yet, we just return not-found instead of fetching from keeper
	return
}

// GetTotalPower returns the total power of all validators
func (c *caches) GetTotalPower() *big.Int {
	return new(big.Int).Set(c.validators.totalPower)
}

// GetTokenFeederForFeederID returns the token feeder for a feederID
func (c *caches) GetTokenFeederForFeederID(feederID int64) (tokenFeeder *oracletypes.TokenFeeder, found bool) {
	if c.params != nil &&
		c.params.params != nil &&
		int64(len(c.params.params.TokenFeeders)) > feederID {
		tokenFeeder = c.params.params.TokenFeeders[feederID]
		found = true
	}
	return
}

// SkipCommit skip real commit by setting the update flag to false
func (c *caches) SkipCommit() {
	c.validators.update = false
	c.params.update = false
}

// Commit commits the cache to the KVStore
func (c *caches) Commit(ctx sdk.Context, reset bool) (msgUpdated, validatorsUpdated, paramsUpdated bool) {
	if len(*(c.msg)) > 0 {
		c.msg.commit(ctx, c.k)
		msgUpdated = true
	}

	if c.validators.update {
		c.validators.commit(ctx, c.k)
		validatorsUpdated = true
	}

	if c.params.update {
		c.params.commit(ctx, c.k)
		paramsUpdated = true
	}
	if reset {
		c.ResetCaches()
	}
	return
}

func (c *caches) ResetCaches() {
	*c = *(newCaches())
}

func newCaches() *caches {
	return &caches{
		msg: new(cacheMsgs),
		validators: &cacheValidator{
			validators: make(map[string]*big.Int),
			totalPower: big.NewInt(0),
		},
		params: &cacheParams{},
	}
}
