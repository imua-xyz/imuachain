package types

import (
	"errors"
	"strings"
	"time"

	sdkmath "cosmossdk.io/math"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"gopkg.in/yaml.v2"
)

var (
	KeyChains       = []byte("Chains")
	KeyTokens       = []byte("Tokens")
	KeySources      = []byte("Sources")
	KeyRules        = []byte("Rules")
	KeyTokenFeeders = []byte("TokenFeeders")
	oneDec          = sdkmath.LegacyNewDec(1)
)

var _ paramtypes.ParamSet = (*Params)(nil)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams() Params {
	return Params{
		// maximum number of transactions be submitted in one round from a validator
		MaxNonce: 1,
		// maximum number of deteministic-source price can be submitted in one round from a validator
		MaxDetId: 1,
		// Mode is set to 1 for V1, means:
		// For deteministic source, use consensus to find out valid final price, for non-deteministic source, use the latest price
		// Final price will be confirmed as soon as the threshold is reached, and will ignore any furthur messages submitted with prices
		Mode:         ConsensusModeASAP,
		ThresholdA:   2,
		ThresholdB:   3,
		Chains:       []*Chain{{}},
		Tokens:       []*Token{{}},
		Sources:      []*Source{{}},
		Rules:        []*RuleSource{{}},
		TokenFeeders: []*TokenFeeder{{}},
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return Params{
		Chains: []*Chain{
			{Name: "-", Desc: "-"},
		},
		Tokens: []*Token{
			{},
		},
		// source defines where to fetch the prices
		Sources: []*Source{
			{
				Name: "0 position is reserved",
			},
		},
		// rules defines price from which sources are accepted, could be used to proof malicious
		Rules: []*RuleSource{
			// 0 is reserved
			{},
		},
		// TokenFeeder describes when a token start to be updated with its price, and the frequency, endTime.
		TokenFeeders: []*TokenFeeder{
			{},
		},
		MaxNonce:   3,
		ThresholdA: 2,
		ThresholdB: 3,
		// V1 set mode to 1
		Mode:          ConsensusModeASAP,
		MaxDetId:      5,
		MaxSizePrices: 100,
		Slashing: &SlashingParams{
			ReportedRoundsWindow:        100,
			MinReportedPerWindow:        sdkmath.LegacyNewDec(1).Quo(sdkmath.LegacyNewDec(2)),
			OracleMissJailDuration:      600 * time.Second,
			OracleMaliciousJailDuration: 30 * 24 * time.Hour,
			SlashFractionMiss:           sdkmath.LegacyNewDec(1).Quo(sdkmath.LegacyNewDec(20)),
			SlashFractionMalicious:      sdkmath.LegacyNewDec(1).Quo(sdkmath.LegacyNewDec(10)),
		},
	}
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyChains, &p.Chains, func(_ interface{}) error { return nil }),
		paramtypes.NewParamSetPair(KeyTokens, &p.Tokens, func(_ interface{}) error { return nil }),
		paramtypes.NewParamSetPair(KeySources, &p.Sources, func(_ interface{}) error { return nil }),
		paramtypes.NewParamSetPair(KeyRules, &p.Rules, func(_ interface{}) error { return nil }),
		paramtypes.NewParamSetPair(KeyTokenFeeders, &p.TokenFeeders, func(_ interface{}) error { return nil }),
	}
}

// TODO: consider to parallel verifications of chains, tokens, rules, ..., to improve performance.
// Validate validates the set of params
func (p Params) Validate() error {
	// Some basic configure params validation:
	// Maxnonce: This tells how many transactions for one round can a validator send, This also restrict how many blocks a window lasts for one round to collect transactions
	// MaxDetID: This only works for DS, to tell how many continuous roundID_from_DS could be accept at most for one round of exorcore_oracle
	// ThresholdA/ThresholdB: represents the threshold of voting power to confirm a price as final price
	// Mode: tells how and when to confirm a final price, expect for voting power threshold, v1 set this value to 1 means final price will be confirmed as soon as it has reached the threshold of total voting power, and just ignore any remaining transactions followed for current round.
	if p.MaxNonce < 1 || p.MaxDetId < 1 || p.ThresholdA < 1 || p.ThresholdB < p.ThresholdA || p.Mode != ConsensusModeASAP || p.MaxSizePrices < 1 {
		return ErrInvalidParams.Wrapf("invalid maxNonce/maxDetID/Threshold/Mode/MaxSizePrices: %d, %d, %d, %d, %d, %d", p.MaxNonce, p.MaxDetId, p.ThresholdA, p.ThresholdB, p.Mode, p.MaxSizePrices)
	}

	slashing := p.Slashing

	if slashing == nil {
		return ErrInvalidParams.Wrap("slashing params must not be nil")
	}

	if slashing.ReportedRoundsWindow < 1 {
		return ErrInvalidParams.Wrapf("ReportedRoundsWindow must be at least 1, got %d", slashing.ReportedRoundsWindow)
	}
	if slashing.MinReportedPerWindow.GT(oneDec) || !slashing.MinReportedPerWindow.IsPositive() {
		return ErrInvalidParams.Wrapf("MinReportedPerWindow must be in (0, 1], got %v", slashing.MinReportedPerWindow)
	}
	if slashing.SlashFractionMiss.GT(oneDec) || !slashing.SlashFractionMiss.IsPositive() {
		return ErrInvalidParams.Wrapf("SlashFractionMiss must be in (0, 1], got %v", slashing.SlashFractionMiss)
	}
	if slashing.SlashFractionMalicious.GT(oneDec) || !slashing.SlashFractionMalicious.IsPositive() {
		return ErrInvalidParams.Wrapf("SlashFractionMalicious must be in (0, 1], got %v", slashing.SlashFractionMalicious)
	}
	if slashing.OracleMissJailDuration <= 0 {
		return ErrInvalidParams.Wrapf("OracleMissJailDuration must be positive, got %v", slashing.OracleMissJailDuration)
	}
	if slashing.OracleMaliciousJailDuration <= 0 {
		return ErrInvalidParams.Wrapf("OracleMaliciousJailDuration must be positive, got %v", slashing.OracleMaliciousJailDuration)
	}

	// validate tokenFeeders
	feeders := make(map[uint64]*TokenFeeder)
	for fID, feeder := range p.TokenFeeders {
		// id==0 is reserved
		if fID == 0 {
			continue
		}
		if err := feeder.validate(); err != nil {
			return err
		}
		// If Endblock is set, it should not be in the window of one round
		if feeder.EndBlock > 0 && (feeder.EndBlock-feeder.StartBaseBlock)%feeder.Interval < uint64(p.MaxNonce) {
			return ErrInvalidParams.Wrap("invalid tokenFeeder, invalid EndBlock")
		}
		// Interval should be long enough, make it more than twice pricing window of one round
		if feeder.Interval < 2*uint64(p.MaxNonce) {
			return ErrInvalidParams.Wrap("invalid tokenFeeder, invalid interval")
		}
		// cross validation with tokens
		if feeder.TokenID >= uint64(len(p.Tokens)) {
			return ErrInvalidParams.Wrap("invalid tokenFeeder, non-exist tokenID referred")
		}
		// cross validation with rules
		if feeder.RuleID >= uint64(len(p.Rules)) {
			return ErrInvalidParams.Wrap("invalid tokenFeeder, non-exist ruleID referred")
		}
		// at most on running or future tokenfeeder
		//	if feederPrev, exists := feeders[feeder.TokenID]; exists {
		// prev.endBlock > current.startBasedBlock, 1. current.endBlock.]>prev.endBlock invalid. 2. current.endBlock < prev.startBasedBlock invalid:in this case if we want to set an earlier feeder, we should just modify the previous one's startBaseddBlock
		if prev, exists := feeders[feeder.TokenID]; exists {
			if prev.EndBlock == 0 {
				return ErrInvalidParams.Wrap("invalid tokenFeeder, for the same token at most one running or future tokenFeeder without endblock set")
			}
			if prev.EndBlock >= feeder.StartBaseBlock {
				// if one token has more than one feeders including stopped feeders, they should be in order of startBaseBlock
				return ErrInvalidParams.Wrap("invalid tokenFeeder, for the same token the range of token feeder can't overlap")
			}
			// this is one feeder that succeeded the previous one, so we should check the roundID is continuoslly increased
			prevEndRoundID := prev.StartRoundID + (prev.EndBlock-prev.StartBaseBlock)/prev.Interval
			if feeder.StartRoundID != prevEndRoundID+1 {
				return ErrInvalidParams.Wrap("invalid tokenFeeder, for the same token the roundID should be continuous")
			}
		}
		feeders[feeder.TokenID] = feeder
	}
	//}
	// validate chain
	for cID, chain := range p.Chains {
		// id==0 is reserved
		if cID == 0 {
			continue
		}
		if err := chain.validate(); err != nil {
			return err
		}
	}

	// validate token
	for tID, token := range p.Tokens {
		// id==0 is reserved
		if tID == 0 {
			continue
		}
		if err := token.validate(); err != nil {
			return err
		}
		// cross validation with chain
		if token.ChainID >= uint64(len(p.Chains)) {
			return ErrInvalidParams.Wrap("invalid token, non-exist chainID referred")
		}
	}

	// validate rules
	for rID, rule := range p.Rules {
		if rID == 0 {
			continue
		}
		if err := rule.validate(); err != nil {
			return err
		}
		// cross validation with sources
		for _, id := range rule.SourceIDs {
			if id >= uint64(len(p.Rules)) {
				return ErrInvalidParams.Wrap("invalid rule")
			}
		}
		if rule.Nom != nil {
			for _, id := range rule.Nom.SourceIDs {
				if id < 1 || id >= uint64(len(p.Rules)) {
					return ErrInvalidParams.Wrap("invalid rule")
				}
			}
		}
	}
	// validete sources
	for sID, source := range p.Sources {
		if sID == 0 {
			continue
		}
		if err := source.validate(); err != nil {
			return err
		}
	}
	return nil
}

// AddSources adds new sources to tell where to fetch prices
func (p Params) AddSources(sources ...*Source) (Params, error) {
	sNames := make(map[string]struct{})
	for _, source := range p.Sources {
		sNames[source.Name] = struct{}{}
	}
	for _, s := range sources {
		if !s.Valid {
			return p, ErrInvalidParams.Wrap("invalid source to add, new source should be valid")
		}
		if _, exists := sNames[s.Name]; exists {
			return p, ErrInvalidParams.Wrap("invalid source to add, duplicated")
		}
		sNames[s.Name] = struct{}{}
		p.Sources = append(p.Sources, s)
	}
	return p, nil
}

// AddChains adds new chains on which tokens are deployed
func (p Params) AddChains(chains ...*Chain) (Params, error) {
	cNames := make(map[string]struct{})
	for _, chain := range p.Chains {
		cNames[chain.Name] = struct{}{}
	}
	for _, c := range chains {
		if _, exists := cNames[c.Name]; exists {
			return p, ErrInvalidParams.Wrap("invalid chain to add, duplicated")
		}
		p.Chains = append(p.Chains, c)
	}
	return p, nil
}

// UpdateTokens upates token info
// Since we don't allow to add any new token with the same Name&ChainID existed, so all fileds except the those are able to be modified
// contractAddress and decimal are only allowed before any tokenFeeder of that token had been started
// assetID is allowed to be modified no matter any tokenFeeder is started
func (p Params) UpdateTokens(currentHeight uint64, tokens ...*Token) (Params, error) {
	for _, t := range tokens {
		update := false
		for tokenID := 1; tokenID < len(p.Tokens); tokenID++ {
			token := p.Tokens[tokenID]
			token.AssetID = strings.ToLower(token.AssetID)
			if token.ChainID == t.ChainID && token.Name == t.Name {
				// modify existing token
				update = true
				// update assetID
				if len(t.AssetID) > 0 {
					token.AssetID = t.AssetID
				}
				if !p.TokenStarted(uint64(tokenID), currentHeight) {
					// contractAddres is mainly used as a description information
					if len(t.ContractAddress) > 0 {
						token.ContractAddress = t.ContractAddress
					}
					// update Decimal, token.Decimal is allowed to modified to at least 1
					if t.Decimal > 0 {
						token.Decimal = t.Decimal
					}
				}
				// any other modification will be ignored
				break
			}
		}
		// add a new token
		if !update {
			p.Tokens = append(p.Tokens, t)
		}
	}
	return p, nil
}

// TokenStarted returns if any tokenFeeder had been started for the specified token identified by tokenID
func (p Params) TokenStarted(tokenID, height uint64) bool {
	for _, feeder := range p.TokenFeeders {
		if feeder.TokenID == tokenID && height >= feeder.StartBaseBlock {
			return true
		}
	}
	return false
}

// AddRules adds a new RuleSource defining which sources and how many of the defined source are needed to be valid for a price to be provided
func (p Params) AddRules(rules ...*RuleSource) (Params, error) {
	p.Rules = append(p.Rules, rules...)
	return p, nil
}

func (p Params) UpdateMaxPriceCount(count int32) (Params, error) {
	if count < 0 {
		return p, ErrInvalidParams.Wrap("invalid maxPriceCount")
	}
	if count > 0 {
		p.MaxSizePrices = count
	}
	return p, nil
}

// UpdateTokenFeeder updates tokenfeeder info, validation first
func (p Params) UpdateTokenFeeder(tf *TokenFeeder, currentHeight uint64) (Params, error) {
	tfIDs := p.GetFeederIDsByTokenID(tf.TokenID)
	if len(tfIDs) == 0 {
		// first tokenfeeder for this token
		p.TokenFeeders = append(p.TokenFeeders, tf)
		return p, nil
	}
	tfIdx := tfIDs[len(tfIDs)-1]
	tokenFeeder := p.TokenFeeders[tfIdx]

	// latest feeder is not started yet
	if tokenFeeder.StartBaseBlock > currentHeight {
		// fields can be modified: startBaseBlock, interval, endBlock
		update := false
		if tf.StartBaseBlock > 0 {
			// Set startBlock to some height in history is not allowed
			if tf.StartBaseBlock <= currentHeight {
				return p, ErrInvalidParams.Wrapf("invalid tokenFeeder to update, invalid StartBaseBlock, currentHeight: %d, set: %d", currentHeight, tf.StartBaseBlock)
			}
			update = true
			tokenFeeder.StartBaseBlock = tf.StartBaseBlock
		}
		if tf.Interval > 0 {
			tokenFeeder.Interval = tf.Interval
			update = true
		}
		if tf.EndBlock > 0 {
			// EndBlock must be set to some height in the future
			if tf.EndBlock <= currentHeight {
				return p, ErrInvalidParams.Wrapf("invalid tokenFeeder to update, invalid EndBlock, currentHeight: %d, set: %d", currentHeight, tf.EndBlock)
			}
			tokenFeeder.EndBlock = tf.EndBlock
			update = true
		}
		// TODO: or we can just ignore this case instead of report an error
		if !update {
			return p, ErrInvalidParams.Wrap("invalid tokenFeeder to update, no valid field set")
		}
		p.TokenFeeders[tfIdx] = tokenFeeder
		return p, nil
	}

	// latest feeder is running
	if tokenFeeder.EndBlock == 0 || tokenFeeder.EndBlock > currentHeight {
		// fields can be modified: endBlock
		if tf.EndBlock == 0 || tf.EndBlock <= currentHeight {
			return p, ErrInvalidParams.Wrapf("invalid tokenFeeder to update, invalid EndBlock, currentHeight: %d, set: %d", currentHeight, tf.EndBlock)
		}
		tokenFeeder.EndBlock = tf.EndBlock
		p.TokenFeeders[tfIdx] = tokenFeeder
		return p, nil
	}

	// latest feeder is stopped, this is actually a new feeder to resume the latest one for the same token
	latestRoundID := tokenFeeder.StartRoundID + (tokenFeeder.EndBlock-tokenFeeder.StartBaseBlock)/tokenFeeder.Interval
	if tf.StartBaseBlock <= currentHeight || tf.StartRoundID != latestRoundID+1 {
		return p, ErrInvalidParams.Wrapf("invalid tokenFeeder to update, invalid StartBaseBlock or StartRoundID, currentHeight:%d, set_startBasedBlock:%d, set_StartRoundID:%d", currentHeight, tf.StartBaseBlock, tf.StartRoundID)
	}
	p.TokenFeeders = append(p.TokenFeeders, tf)

	return p, nil
}

// String implements the Stringer interface.
func (p Params) String() string {
	out, _ := yaml.Marshal(p)
	return string(out)
}

// GetSourceIDByName returns sourceID related to the specified name
func (p Params) GetSourceIDByName(n string) int {
	for i, s := range p.Sources {
		if n == s.Name {
			return i
		}
	}
	return 0
}

// GetFeederIDsByTokenID returns all feederIDs related to the specified token
func (p Params) GetFeederIDsByTokenID(tID uint64) []int {
	ret := make([]int, 0)
	for fID, f := range p.TokenFeeders {
		// feeder list is ordered, so the slice returned is in the order of the feeders updated for the same token
		if f.TokenID == tID {
			ret = append(ret, fID)
		}
	}
	return ret
}

// validate validation on field Chain
func (c Chain) validate() error {
	// Name must be set
	if len(c.Name) == 0 {
		return ErrInvalidParams.Wrap("invalid Chain, name not set")
	}
	return nil
}

// validation on field Token
func (t Token) validate() error {
	// Name must be set, and chainID must start from 1
	if len(t.Name) == 0 || t.ChainID < 1 {
		return ErrInvalidParams.Wrap("invalid Token, name not set or ChainID<1")
	}
	return nil
}

// validate validation on field RuleSource
func (r RuleSource) validate() error {
	// at least one of SourceIDs and Nom has to be set
	if len(r.SourceIDs) == 0 && (r.Nom == nil || len(r.Nom.SourceIDs) == 0) {
		return ErrInvalidParams.Wrap("invalid RuleSource")
	}
	if r.Nom != nil &&
		r.Nom.Minimum > uint64(len(r.Nom.SourceIDs)) {
		return ErrInvalidParams.Wrap("invalid RuleSource")
	}
	return nil
}

// validate validation on filed Source
func (s Source) validate() error {
	// Name must be set
	if len(s.Name) == 0 {
		return ErrInvalidParams.Wrap("invalid Source, name not set")
	}
	return nil
}

// validate validation on field TokenFeeder
func (f TokenFeeder) validate() error {
	// IDs must start from 1
	if f.TokenID < 1 ||
		f.StartRoundID < 1 ||
		f.Interval < 1 ||
		f.StartBaseBlock < 1 {
		return ErrInvalidParams.Wrapf("invalid TokenFeeder, tokenID/startRoundID/interval/startBaseBlock: %d, %d, %d, %d", f.TokenID, f.StartRoundID, f.Interval, f.StartBaseBlock)
	}

	// if EndBlock is set, it must be bigger than startBaseBlock
	if f.EndBlock > 0 && f.StartBaseBlock >= f.EndBlock {
		return ErrInvalidParams.Wrapf("invalid TokenFeeder, invalid EndBlock to be set, startBaseBlock: %d, endBlock: %d", f.StartBaseBlock, f.EndBlock)
	}

	return nil
}

func (p Params) GetTokenIDFromAssetID(assetID string) int {
	for id, token := range p.Tokens {
		assetIDs := strings.Split(token.AssetID, ",")
		for _, aID := range assetIDs {
			if aID == assetID {
				return id
			}
		}
	}
	return 0
}

func (p Params) GetAssetIDsFromTokenID(tokenID uint64) []string {
	if tokenID >= uint64(len(p.Tokens)) {
		return nil
	}
	return strings.Split(p.Tokens[tokenID].AssetID, ",")
}

func (p Params) IsDeterministicSource(sourceID uint64) bool {
	return p.Sources[sourceID].Deterministic
}

func (p Params) IsValidSource(sourceID uint64) bool {
	if sourceID == 0 {
		// custom defined source
		return true
	}
	return p.Sources[sourceID].Valid
}

func (p Params) GetTokenFeeder(feederID uint64) *TokenFeeder {
	for k, v := range p.TokenFeeders {
		if uint64(k) == feederID {
			return v
		}
	}
	return nil
}

func (p Params) GetTokenInfo(feederID uint64) *Token {
	for k, v := range p.TokenFeeders {
		if uint64(k) == feederID {
			return p.Tokens[v.TokenID]
		}
	}
	return nil
}

func (p Params) CheckRules(feederID uint64, prices []*PriceSource) (bool, error) {
	feeder := p.TokenFeeders[feederID]
	rule := p.Rules[feeder.RuleID]
	// specified sources set, v1 use this rule to set `chainlink` as official source
	if len(rule.SourceIDs) > 0 {
		if len(rule.SourceIDs) != len(prices) {
			return false, errors.New("count prices should match rule")
		}
		notFound := false
		if rule.SourceIDs[0] == 0 {
			// match all sources listed
			for sID, source := range p.Sources {
				if sID == 0 {
					continue
				}
				if source.Valid {
					notFound = true
					for _, p := range prices {
						if p.SourceID == uint64(sID) {
							notFound = false
							break
						}
					}

				}
			}
		} else {
			for _, source := range rule.SourceIDs {
				notFound = true
				for _, p := range prices {
					if p.SourceID == source {
						notFound = false
						break
					}
				}
			}
		}
		if notFound {
			return false, errors.New("price source not match with rule")
		}
	}

	// TODO: check NOM
	// return true if no rule set, we will accept any source
	return true, nil
}

// CheckDecimal checks the decimal with feederID equals to decimal set in params, this check should be called after tokenfeeder valid check.
func (p Params) CheckDecimal(feederID uint64, decimal int32) bool {
	feeder := p.TokenFeeders[feederID]
	token := p.Tokens[feeder.TokenID]
	return token.Decimal == decimal
}
