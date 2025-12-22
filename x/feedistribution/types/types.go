package types

import (
	"fmt"
	"sort"
	"strings"

	oracletypes "github.com/imua-xyz/imuachain/x/oracle/types"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type (
	DeltaAVSRewardAssetState      AVSRewardAssetState
	DeltaOperatorUnclaimedRewards OperatorUnclaimedRewards
	DeltaStakerClaimedRewards     StakerClaimedRewards
	OperatorRewardProportions     []OperatorRewardProportion

	CommonAVSRewards           []CommonAVSRewardData
	EpochRewardsAndProportions struct {
		Rewards                   sdk.DecCoins
		OperatorRewardProportions []OperatorRewardProportion
	}

	CompoundingRewards        []CompoundingRewardsPerAsset
	CompoundingRewardsWithAVS struct {
		AVS                string
		CompoundingRewards CompoundingRewards
	}

	RewardsDelegationShares []RewardsDelegationShare
)

// String implements the Stringer interface for OperatorRewardProportions. It returns a
// human-readable representation of operator reward proportions
func (op OperatorRewardProportions) String() string {
	if len(op) == 0 {
		return ""
	}

	out := ""
	for _, p := range op {
		proportionStr := fmt.Sprintf("%v:%v", p.OperatorAddr, p.RewardProportion.String())
		out += fmt.Sprintf("%v,", proportionStr)
	}

	if out != "" {
		out = out[:len(out)-1]
	}
	return out
}

// ParseOperatorRewardProportions parses a string representation like "addr1:0.7,addr2:0.3"
// into a slice of OperatorRewardProportion structs.
func ParseOperatorRewardProportions(s string) ([]OperatorRewardProportion, error) {
	if strings.TrimSpace(s) == "" {
		return nil, nil
	}

	parts := strings.Split(s, ",")
	result := make([]OperatorRewardProportion, 0, len(parts))

	for _, part := range parts {
		addr, propStr, ok := strings.Cut(part, ":")
		if !ok {
			return nil, fmt.Errorf("invalid format for operator proportion: %q", part)
		}

		addr = strings.TrimSpace(addr)
		propStr = strings.TrimSpace(propStr)

		prop, err := sdk.NewDecFromStr(propStr)
		if err != nil {
			return nil, fmt.Errorf("invalid decimal for operator %s: %w", addr, err)
		}

		result = append(result, OperatorRewardProportion{
			OperatorAddr:     addr,
			RewardProportion: prop,
		})
	}

	return result, nil
}

// AppendUniqueStakerID appends a new stakerID to the staker list in DelegationChangeInfo
// only if it's not already present.
// return true if the stake is appended
func (d *DelegationChangeInfo) AppendUniqueStakerID(stakerID string, preDelegatedAmount sdkmath.Int, assetDecimal uint32) bool {
	// Check if the newKey already exists in the slice
	for _, stakerDelegationChange := range d.StakerDelegationChanges {
		if stakerDelegationChange.StakerId == stakerID {
			// If the staker already exists, do not append it
			return false
		}
	}
	// Append the newKey if it's not already present
	d.StakerDelegationChanges = append(d.StakerDelegationChanges, StakerDelegationChange{
		StakerId: stakerID,
		PreviousDelegatedAmount: ScaleIntByDecimals(
			preDelegatedAmount, assetDecimal),
	})
	return true
}

func (d *DelegationChangeInfo) DelegationChangesByStaker() map[string]sdk.Dec {
	ret := make(map[string]sdk.Dec)
	for _, changedDelegation := range d.StakerDelegationChanges {
		ret[changedDelegation.StakerId] = changedDelegation.PreviousDelegatedAmount
	}
	return ret
}

// HasAVSReward checks whether the avs reward exists.
func (o *OperatorCurrentRewards) HasAVSReward(avsAddr string) bool {
	return CommonAVSRewards(o.Rewards).RewardsOf(avsAddr) != nil
}

func (o *OperatorCurrentRewards) UpdateReward(isIncrease bool, deltaRewards CommonAVSRewardData) error {
	if isIncrease {
		o.Rewards = CommonAVSRewards(o.Rewards).Add(deltaRewards)
	} else {
		newRewards, isAnyNegative := CommonAVSRewards(o.Rewards).SafeSub(CommonAVSRewards{deltaRewards})
		if isAnyNegative {
			return ErrNegativeCoinAmount.
				Wrapf("failed to update the current reward for specific AVS, avsAddr:%s", deltaRewards.AVSAddress)
		}
		o.Rewards = newRewards
	}
	return nil
}

// This implementation refers to the DecCoins in cosmos-sdk.
// The CommonAVSRewardData entries are sorted by avsAddr when added to CommonAVSRewards.

func (cr CommonAVSRewardData) IsZeroRewards() bool {
	if len(cr.Rewards) == 0 {
		return true
	}
	return cr.Rewards.IsZero()
}

func (cr CommonAVSRewardData) IsPositive() bool {
	return !cr.IsZeroRewards() && cr.Rewards.IsAllPositive()
}

func (cr CommonAVSRewardData) Add(avsRewardB CommonAVSRewardData) CommonAVSRewardData {
	if cr.AVSAddress != avsRewardB.AVSAddress {
		return cr
	}
	return CommonAVSRewardData{
		AVSAddress: cr.AVSAddress,
		Rewards:    cr.Rewards.Add(avsRewardB.Rewards...),
	}
}

// Sorting

var _ sort.Interface = CommonAVSRewards{}

// Len implements sort.Interface for CommonAVSRewards
func (crs CommonAVSRewards) Len() int { return len(crs) }

// Less implements sort.Interface for CommonAVSRewards
func (crs CommonAVSRewards) Less(i, j int) bool { return crs[i].AVSAddress < crs[j].AVSAddress }

// Swap implements sort.Interface for CommonAVSRewards
func (crs CommonAVSRewards) Swap(i, j int) { crs[i], crs[j] = crs[j], crs[i] }

// Sort is a helper function to sort the set of CommonAVSRewards in-place.
func (crs CommonAVSRewards) Sort() CommonAVSRewards {
	sort.Sort(crs)
	return crs
}

// NewCommonAVSRewards constructs a new CommonAVSRewardData set.
// The provided CommonAVSRewardData will be sanitized by removing
// zero rewards and sorting the CommonAVSRewardData set. A panic will occur if the
// CommonAVSRewardData set is not valid.
func NewCommonAVSRewards(avsRewards ...CommonAVSRewardData) CommonAVSRewards {
	newAVSRewards := sanitizeCommonAVSRewards(avsRewards)
	if err := newAVSRewards.Validate(); err != nil {
		panic(fmt.Errorf("invalid avs reward set %s: %w", newAVSRewards, err))
	}

	return newAVSRewards
}

func sanitizeCommonAVSRewards(avsRewards []CommonAVSRewardData) CommonAVSRewards {
	// remove zeroes
	newAVSRewards := removeZeroAVSReward(avsRewards)
	if len(newAVSRewards) == 0 {
		return CommonAVSRewards{}
	}

	return newAVSRewards.Sort()
}

func removeZeroAVSReward(avsRewards CommonAVSRewards) CommonAVSRewards {
	result := make([]CommonAVSRewardData, 0, len(avsRewards))

	for _, avsReward := range avsRewards {
		if !avsReward.IsZeroRewards() {
			result = append(result, avsReward)
		}
	}

	return result
}

// negative returns a set of coins with all amount negative.
func negativeDecCoins(coins sdk.DecCoins) sdk.DecCoins {
	res := make([]sdk.DecCoin, 0, len(coins))
	for _, coin := range coins {
		res = append(res, sdk.DecCoin{
			Denom:  coin.Denom,
			Amount: coin.Amount.Neg(),
		})
	}
	return res
}

// Validate checks that the CommonAVSRewards are sorted, have positive rewards, with a unique
// avs address (i.e no duplicates). Otherwise, it returns an error.
// we don't validate the avs address here, because the input avs addresses are always valid when
// handling reward distribution.
func (crs CommonAVSRewards) Validate() error {
	switch len(crs) {
	case 0:
		return nil

	case 1:
		if !crs[0].IsPositive() {
			return fmt.Errorf("avsReward amount is not positive,avs:%s,rewards:%s", crs[0].AVSAddress, crs[0].Rewards)
		}
		return nil
	default:
		// check single avsReward case
		if err := (CommonAVSRewards{crs[0]}).Validate(); err != nil {
			return err
		}

		lowAVS := crs[0].AVSAddress
		for _, avsReward := range crs[1:] {
			if avsReward.AVSAddress <= lowAVS {
				return fmt.Errorf("avs address %s is not sorted", avsReward.AVSAddress)
			}
			if !avsReward.IsPositive() {
				return fmt.Errorf("avsReward %s amount is not positive", avsReward.AVSAddress)
			}

			// we compare each avsReward against the last avs address
			lowAVS = avsReward.AVSAddress
		}

		return nil
	}
}

// Add adds two sets of CommonAVSRewardData.
//
// NOTE: Add operates under the invariant that CommonAVSRewardData are sorted by
// avsAddr.
//
// CONTRACT: Add will never return CommonAVSRewards where one CommonAVSRewardData has a non-positive
// rewards. In otherwords, IsValid will always return true.
func (crs CommonAVSRewards) Add(avsRewards ...CommonAVSRewardData) CommonAVSRewards {
	return crs.safeAdd(avsRewards)
}

// safeAdd will perform addition of two CommonAVSRewards sets. If both CommonAVSRewards sets are
// empty, then an empty set is returned. If only a single set is empty, the
// other set is returned. Otherwise, the CommonAVSRewards are compared in order of their
// avs address and addition only occurs when the address match, otherwise
// the CommonAVSRewards is simply added to the sum assuming it's not zero.
// nolint:dupl
func (crs CommonAVSRewards) safeAdd(avsRewardsB CommonAVSRewards) CommonAVSRewards {
	sum := ([]CommonAVSRewardData)(nil)
	indexA, indexB := 0, 0
	lenA, lenB := len(crs), len(avsRewardsB)

	for {
		if indexA == lenA {
			if indexB == lenB {
				// return nil coins if both sets are empty
				return sum
			}

			// return set B (excluding zero rewards) if set A is empty
			return append(sum, removeZeroAVSReward(avsRewardsB[indexB:])...)
		} else if indexB == lenB {
			// return set A (excluding zero rewards) if set B is empty
			return append(sum, removeZeroAVSReward(crs[indexA:])...)
		}

		avsRewardA, avsRewardB := crs[indexA], avsRewardsB[indexB]

		switch strings.Compare(avsRewardA.AVSAddress, avsRewardB.AVSAddress) {
		case -1: // avs A address < avs B address
			if !avsRewardA.IsZeroRewards() {
				sum = append(sum, avsRewardA)
			}

			indexA++

		case 0: // avs A address == avs B address
			res := avsRewardA.Add(avsRewardB)
			if !res.IsZeroRewards() {
				sum = append(sum, res)
			}

			indexA++
			indexB++

		case 1: // avs A address > avs B address
			if !avsRewardB.IsZeroRewards() {
				sum = append(sum, avsRewardB)
			}

			indexB++
		}
	}
}

// negative returns a set of CommonAVSRewardData with all rewards amount negative.
func (crs CommonAVSRewards) negative() CommonAVSRewards {
	res := make([]CommonAVSRewardData, 0, len(crs))
	for _, avsReward := range crs {
		res = append(res, CommonAVSRewardData{
			AVSAddress: avsReward.AVSAddress,
			Rewards:    negativeDecCoins(avsReward.Rewards),
		})
	}
	return res
}

// IsAnyNegative returns true if there is at least one coin of the avs rewards whose amount
// is negative; returns false otherwise. It returns false if the CommonAVSRewards set
// is empty too.
func (crs CommonAVSRewards) IsAnyNegative() bool {
	for _, avsReward := range crs {
		if avsReward.Rewards.IsAnyNegative() {
			return true
		}
	}
	return false
}

func (crs CommonAVSRewards) IsZeroRewards() bool {
	for _, rewardPerAVS := range crs {
		if !rewardPerAVS.IsZeroRewards() {
			return false
		}
	}
	return true
}

// Sub subtracts a set of CommonAVSRewards from another (adds the inverse).
func (crs CommonAVSRewards) Sub(avsRewardsB CommonAVSRewards) CommonAVSRewards {
	diff, hasNeg := crs.SafeSub(avsRewardsB)
	if hasNeg {
		panic("negative avs rewards")
	}

	return diff
}

// SafeSub performs the same arithmetic as Sub but returns a boolean if any
// negative avs rewards amount was returned.
func (crs CommonAVSRewards) SafeSub(avsRewardsB CommonAVSRewards) (CommonAVSRewards, bool) {
	diff := crs.safeAdd(avsRewardsB.negative())
	return diff, diff.IsAnyNegative()
}

// CalculateRewardRatio calculates the rewards ratio， the receiver of this function should be the total rewards.
func (crs CommonAVSRewards) CalculateRewardRatio(totalDelegatedAmount sdk.Dec) (CommonAVSRewards, error) {
	if !totalDelegatedAmount.IsPositive() {
		return nil, ErrInvalidInputParameter.Wrapf("CalculateRewardRatio, total delegated amount isn't positive, value:%s", totalDelegatedAmount)
	}
	ret := make([]CommonAVSRewardData, 0)
	for _, avsRewards := range crs {
		// note: necessary to truncate, so we don't allow withdrawing more currentRewards than owed
		rewardRito := avsRewards.Rewards.QuoDecTruncate(totalDelegatedAmount)
		ret = append(ret, CommonAVSRewardData{
			AVSAddress: avsRewards.AVSAddress,
			Rewards:    rewardRito,
		})
	}
	return ret, nil
}

func (crs CommonAVSRewards) MulDecTruncate(multiplier sdk.Dec) (CommonAVSRewards, error) {
	if multiplier.IsNegative() {
		return nil, ErrInvalidInputParameter.Wrapf("MulDecTruncate, the multiplier is negative, value:%s", multiplier)
	}
	ret := make([]CommonAVSRewardData, 0)
	if multiplier.IsZero() {
		return ret, nil
	}
	for _, avsReward := range crs {
		// note: necessary to truncate so we don't allow withdrawing more rewards than owed
		rewards := avsReward.Rewards.MulDecTruncate(multiplier)
		ret = append(ret, CommonAVSRewardData{
			AVSAddress: avsReward.AVSAddress,
			Rewards:    rewards,
		})
	}
	return ret, nil
}

func (crs CommonAVSRewards) RewardsOf(avsAddr string) sdk.DecCoins {
	for _, avsRewards := range crs {
		if avsAddr == avsRewards.AVSAddress {
			return avsRewards.Rewards
		}
	}
	return nil
}

func (cra CompoundingRewardsPerAsset) IsZeroRewards() bool {
	return CommonAVSRewards(cra.Rewards).IsZeroRewards()
}

func (cra CompoundingRewardsPerAsset) IsPositive() bool {
	return !cra.IsZeroRewards() && !CommonAVSRewards(cra.Rewards).IsAnyNegative()
}

func (cra CompoundingRewardsPerAsset) Add(compoundingRewardB CompoundingRewardsPerAsset) CompoundingRewardsPerAsset {
	if cra.RewardDenomination != compoundingRewardB.RewardDenomination {
		return cra
	}
	return CompoundingRewardsPerAsset{
		RewardDenomination: cra.RewardDenomination,
		Rewards:            CommonAVSRewards(cra.Rewards).Add(compoundingRewardB.Rewards...),
	}
}

// Sorting
var _ sort.Interface = CompoundingRewards{}

// Len implements sort.Interface for CompoundingRewards
func (cmr CompoundingRewards) Len() int { return len(cmr) }

// Less implements sort.Interface for CompoundingRewards
func (cmr CompoundingRewards) Less(i, j int) bool {
	return cmr[i].RewardDenomination < cmr[j].RewardDenomination
}

// Swap implements sort.Interface for CompoundingRewards
func (cmr CompoundingRewards) Swap(i, j int) { cmr[i], cmr[j] = cmr[j], cmr[i] }

// Sort is a helper function to sort the set of CompoundingRewards in-place.
func (cmr CompoundingRewards) Sort() CompoundingRewards {
	sort.Sort(cmr)
	return cmr
}

// NewCompoundingRewards constructs a new CompoundingRewardsPerAsset set.
// The provided CompoundingRewardsPerAsset will be sanitized by removing
// zero rewards and sorting the CompoundingRewardsPerAsset set. An empty
// CompoundingRewards will be returned if the CompoundingRewardsPerAsset
// set is not valid.
func NewCompoundingRewards(compoundingRewards ...CompoundingRewardsPerAsset) CompoundingRewards {
	newAVSRewards := sanitizeCompoundingRewards(compoundingRewards)
	if err := newAVSRewards.Validate(); err != nil {
		return CompoundingRewards{}
	}

	return newAVSRewards
}

func sanitizeCompoundingRewards(compoundingRewards []CompoundingRewardsPerAsset) CompoundingRewards {
	// remove zeroes
	newCompoundingRewards := removeZeroCompoundingRewards(compoundingRewards)
	if len(newCompoundingRewards) == 0 {
		return CompoundingRewards{}
	}

	return newCompoundingRewards.Sort()
}

func removeZeroCompoundingRewards(compoundingRewards CompoundingRewards) CompoundingRewards {
	result := make([]CompoundingRewardsPerAsset, 0, len(compoundingRewards))

	for _, compoundingReward := range compoundingRewards {
		if !compoundingReward.IsZeroRewards() {
			result = append(result, compoundingReward)
		}
	}

	return result
}

// Validate checks that the CompoundingRewards are sorted, have positive rewards, with a unique
// symbol (i.e no duplicates). Otherwise, it returns an error.
func (cmr CompoundingRewards) Validate() error {
	switch len(cmr) {
	case 0:
		return nil

	case 1:
		if !cmr[0].IsPositive() {
			return fmt.Errorf("rewardsPerAsset amount is not positive,rewardDenomination:%s", cmr[0].RewardDenomination)
		}
		return nil
	default:
		// check single compounding reward case
		if err := (CompoundingRewards{cmr[0]}).Validate(); err != nil {
			return err
		}

		lowRewardDenomination := cmr[0].RewardDenomination
		for _, rewardsPerAsset := range cmr[1:] {
			if rewardsPerAsset.RewardDenomination <= lowRewardDenomination {
				return fmt.Errorf("rewardDenomination %s is not sorted", rewardsPerAsset.RewardDenomination)
			}
			if !rewardsPerAsset.IsPositive() {
				return fmt.Errorf("rewardDenomination %s amount is not positive", rewardsPerAsset.RewardDenomination)
			}

			// we compare each rewardsPerAsset against the last avs address
			lowRewardDenomination = rewardsPerAsset.RewardDenomination
		}

		return nil
	}
}

func (cmr CompoundingRewards) RewardsOf(symbol string) CommonAVSRewards {
	for _, assetRewards := range cmr {
		if symbol == assetRewards.RewardDenomination {
			return assetRewards.Rewards
		}
	}
	return nil
}

func (cmr CompoundingRewards) Add(compoundingRewards ...CompoundingRewardsPerAsset) CompoundingRewards {
	return cmr.safeAdd(compoundingRewards)
}

// safeAdd will perform addition of two CompoundingRewards sets. If both CompoundingRewards sets are
// empty, then an empty set is returned. If only a single set is empty, the
// other set is returned. Otherwise, the CompoundingRewards are compared in order of their
// symbols and addition only occurs when the symbol match, otherwise
// the CompoundingRewards is simply added to the sum assuming it's not zero.
// nolint:dupl
func (cmr CompoundingRewards) safeAdd(compoundingRewardsB CompoundingRewards) CompoundingRewards {
	sum := ([]CompoundingRewardsPerAsset)(nil)
	indexA, indexB := 0, 0
	lenA, lenB := len(cmr), len(compoundingRewardsB)

	for {
		if indexA == lenA {
			if indexB == lenB {
				// return nil coins if both sets are empty
				return sum
			}

			// return set B (excluding zero rewards) if set A is empty
			return append(sum, removeZeroCompoundingRewards(compoundingRewardsB[indexB:])...)
		} else if indexB == lenB {
			// return set A (excluding zero rewards) if set B is empty
			return append(sum, removeZeroCompoundingRewards(cmr[indexA:])...)
		}

		compoundingRewardA, compoundingRewardB := cmr[indexA], compoundingRewardsB[indexB]

		switch strings.Compare(compoundingRewardA.RewardDenomination, compoundingRewardB.RewardDenomination) {
		case -1: // coin A rewardDenomination < coin B rewardDenomination
			if !compoundingRewardA.IsZeroRewards() {
				sum = append(sum, compoundingRewardA)
			}

			indexA++

		case 0: // coin A rewardDenomination = coin B rewardDenomination
			res := compoundingRewardA.Add(compoundingRewardB)
			if !res.IsZeroRewards() {
				sum = append(sum, res)
			}

			indexA++
			indexB++

		case 1: // coin A rewardDenomination > coin B rewardDenomination
			if !compoundingRewardB.IsZeroRewards() {
				sum = append(sum, compoundingRewardB)
			}

			indexB++
		}
	}
}

// IsAnyNegative returns true if there is at least one coin of the compounding rewards whose amount
// is negative; returns false otherwise. It returns false if the CompoundingRewards set
// is empty too.
func (cmr CompoundingRewards) IsAnyNegative() bool {
	for _, avsReward := range cmr {
		if CommonAVSRewards(avsReward.Rewards).IsAnyNegative() {
			return true
		}
	}
	return false
}

// negative returns a set of CompoundingRewardsPerAsset with all rewards amount negative.
func (cmr CompoundingRewards) negative() CompoundingRewards {
	res := make([]CompoundingRewardsPerAsset, 0, len(cmr))
	for _, compoundingReward := range cmr {
		res = append(res, CompoundingRewardsPerAsset{
			RewardDenomination: compoundingReward.RewardDenomination,
			Rewards:            CommonAVSRewards(compoundingReward.Rewards).negative(),
		})
	}
	return res
}

// Sub subtracts a set of CompoundingRewards from another (adds the inverse).
func (cmr CompoundingRewards) Sub(compoundingRewardsB CompoundingRewards) CompoundingRewards {
	diff, hasNeg := cmr.SafeSub(compoundingRewardsB)
	if hasNeg {
		panic("negative compounding rewards")
	}

	return diff
}

// SafeSub performs the same arithmetic as Sub but returns a boolean if any
// negative avs rewards amount was returned.
func (cmr CompoundingRewards) SafeSub(avsRewardsB CompoundingRewards) (CompoundingRewards, bool) {
	diff := cmr.safeAdd(avsRewardsB.negative())
	return diff, diff.IsAnyNegative()
}

func (rd RewardsDelegationShare) IsZeroShares() bool {
	if len(rd.Shares) == 0 {
		return true
	}
	return rd.Shares.IsZero()
}

func (rd RewardsDelegationShare) IsPositive() bool {
	return !rd.IsZeroShares() && rd.Shares.IsAllPositive()
}

func (rd RewardsDelegationShare) Add(delegationShareB RewardsDelegationShare) RewardsDelegationShare {
	if rd.OperatorAddr != delegationShareB.OperatorAddr {
		return rd
	}
	return RewardsDelegationShare{
		OperatorAddr: rd.OperatorAddr,
		Shares:       rd.Shares.Add(delegationShareB.Shares...),
	}
}

// Sorting

var _ sort.Interface = RewardsDelegationShares{}

// Len implements sort.Interface for CommonAVSRewards
func (rds RewardsDelegationShares) Len() int { return len(rds) }

// Less implements sort.Interface for CommonAVSRewards
func (rds RewardsDelegationShares) Less(i, j int) bool {
	return rds[i].OperatorAddr < rds[j].OperatorAddr
}

// Swap implements sort.Interface for CommonAVSRewards
func (rds RewardsDelegationShares) Swap(i, j int) { rds[i], rds[j] = rds[j], rds[i] }

// Sort is a helper function to sort the set of CommonAVSRewards in-place.
func (rds RewardsDelegationShares) Sort() RewardsDelegationShares {
	sort.Sort(rds)
	return rds
}

// NewRewardsDelegationShares constructs a new RewardsDelegationShare set.
// The provided RewardsDelegationShare will be sanitized by removing
// zero rewards and sorting the RewardsDelegationShare set. A panic will occur if the
// RewardsDelegationShare set is not valid.
func NewRewardsDelegationShares(delegationShares ...RewardsDelegationShare) RewardsDelegationShares {
	newRewardDelegationShares := sanitizeRewardsDelegationShares(delegationShares)
	if err := newRewardDelegationShares.Validate(); err != nil {
		panic(fmt.Errorf("invalid reward delegation shares set %s: %w", newRewardDelegationShares, err))
	}

	return newRewardDelegationShares
}

func sanitizeRewardsDelegationShares(delegationShares []RewardsDelegationShare) RewardsDelegationShares {
	// remove zeroes
	newDelegationShares := removeZeroRewardsDelegationShare(delegationShares)
	if len(newDelegationShares) == 0 {
		return RewardsDelegationShares{}
	}

	return newDelegationShares.Sort()
}

func removeZeroRewardsDelegationShare(delegationShares RewardsDelegationShares) RewardsDelegationShares {
	result := make([]RewardsDelegationShare, 0, len(delegationShares))

	for _, delegationShare := range delegationShares {
		if !delegationShare.IsZeroShares() {
			result = append(result, delegationShare)
		}
	}

	return result
}

// Validate checks that the RewardsDelegationShares are sorted, have positive shares, with a unique
// operator address (i.e no duplicates). Otherwise, it returns an error.
// we don't validate the operator address here, because the input operator addresses are always valid when
// handling reward distribution.
func (rds RewardsDelegationShares) Validate() error {
	switch len(rds) {
	case 0:
		return nil

	case 1:
		if !rds[0].IsPositive() {
			return fmt.Errorf("reward delegation shares is not positive,operator:%s,shares:%s", rds[0].OperatorAddr, rds[0].Shares)
		}
		return nil
	default:
		// check single delegationShares case
		if err := (RewardsDelegationShares{rds[0]}).Validate(); err != nil {
			return err
		}

		lowOperator := rds[0].OperatorAddr
		for _, delegationShares := range rds[1:] {
			if delegationShares.OperatorAddr <= lowOperator {
				return fmt.Errorf("operator address %s is not sorted", delegationShares.OperatorAddr)
			}
			if !delegationShares.IsPositive() {
				return fmt.Errorf("reward delegation shares %s amount is not positive", delegationShares.OperatorAddr)
			}

			// we compare each delegationShares against the last operator address
			lowOperator = delegationShares.OperatorAddr
		}

		return nil
	}
}

// Add adds two sets of RewardsDelegationShare.
//
// NOTE: Add operates under the invariant that RewardsDelegationShare are sorted by
// avsAddr.
//
// CONTRACT: Add will never return RewardsDelegationShares where one RewardsDelegationShare has a non-positive
// rewards. In otherwords, IsValid will always return true.
func (rds RewardsDelegationShares) Add(delegationShares ...RewardsDelegationShare) RewardsDelegationShares {
	return rds.safeAdd(delegationShares)
}

// safeAdd will perform addition of two RewardsDelegationShares sets. If both RewardsDelegationShares sets are
// empty, then an empty set is returned. If only a single set is empty, the
// other set is returned. Otherwise, the RewardsDelegationShares are compared in order of their
// operator address and addition only occurs when the address match, otherwise
// the RewardsDelegationShares is simply added to the sum assuming it's not zero.
// nolint:dupl
func (rds RewardsDelegationShares) safeAdd(delegationSharesB RewardsDelegationShares) RewardsDelegationShares {
	sum := ([]RewardsDelegationShare)(nil)
	indexA, indexB := 0, 0
	lenA, lenB := len(rds), len(delegationSharesB)

	for {
		if indexA == lenA {
			if indexB == lenB {
				// return nil coins if both sets are empty
				return sum
			}

			// return set B (excluding zero shares) if set A is empty
			return append(sum, removeZeroRewardsDelegationShare(delegationSharesB[indexB:])...)
		} else if indexB == lenB {
			// return set A (excluding zero shares) if set B is empty
			return append(sum, removeZeroRewardsDelegationShare(rds[indexA:])...)
		}

		delegationShareA, delegationShareB := rds[indexA], delegationSharesB[indexB]

		switch strings.Compare(delegationShareA.OperatorAddr, delegationShareB.OperatorAddr) {
		case -1: // operator A address < operator B address
			if !delegationShareA.IsZeroShares() {
				sum = append(sum, delegationShareA)
			}

			indexA++

		case 0: // operator A address == operator B address
			res := delegationShareA.Add(delegationShareB)
			if !res.IsZeroShares() {
				sum = append(sum, res)
			}

			indexA++
			indexB++

		case 1: // operator A address > operator B address
			if !delegationShareB.IsZeroShares() {
				sum = append(sum, delegationShareB)
			}

			indexB++
		}
	}
}

// negative returns a set of RewardsDelegationShare with all rewards delegation shares negative.
func (rds RewardsDelegationShares) negative() RewardsDelegationShares {
	res := make([]RewardsDelegationShare, 0, len(rds))
	for _, delegationShare := range rds {
		res = append(res, RewardsDelegationShare{
			OperatorAddr: delegationShare.OperatorAddr,
			Shares:       negativeDecCoins(delegationShare.Shares),
		})
	}
	return res
}

// IsAnyNegative returns true if there is at least one coin of the delegation share whose amount
// is negative; returns false otherwise. It returns false if the RewardsDelegationShares set
// is empty too.
func (rds RewardsDelegationShares) IsAnyNegative() bool {
	for _, avsReward := range rds {
		if avsReward.Shares.IsAnyNegative() {
			return true
		}
	}
	return false
}

func (rds RewardsDelegationShares) IsZeroShares() bool {
	for _, sharePerOperator := range rds {
		if !sharePerOperator.IsZeroShares() {
			return false
		}
	}
	return true
}

// Sub subtracts a set of RewardsDelegationShares from another (adds the inverse).
func (rds RewardsDelegationShares) Sub(delegationSharesB RewardsDelegationShares) RewardsDelegationShares {
	diff, hasNeg := rds.SafeSub(delegationSharesB)
	if hasNeg {
		panic("negative rewards delegation shares")
	}

	return diff
}

// SafeSub performs the same arithmetic as Sub but returns a boolean if any
// negative delegation shares were returned.
func (rds RewardsDelegationShares) SafeSub(delegationSharesB RewardsDelegationShares) (RewardsDelegationShares, bool) {
	diff := rds.safeAdd(delegationSharesB.negative())
	return diff, diff.IsAnyNegative()
}

func (rds RewardsDelegationShares) DelegationSharesOf(operator string) sdk.DecCoins {
	for _, delegationShares := range rds {
		if operator == delegationShares.OperatorAddr {
			return delegationShares.Shares
		}
	}
	return nil
}

func (srp StakerRewardParams) Validate() error {
	if srp.RedelegateReward {
		_, err := sdk.AccAddressFromBech32(srp.RedelegateOperatorAddr)
		if err != nil {
			return fmt.Errorf("invalid redelegated operator address:%s,err:%s", srp.RedelegateOperatorAddr, err)
		}
	}
	return nil
}

func DefaultStakerClaimedRewards() StakerClaimedRewards {
	return StakerClaimedRewards{
		OutstandingRewards:         sdk.NewDecCoins(),
		WithdrawnRewards:           sdk.NewDecCoins(),
		HistoricalTotalRewards:     sdk.NewDecCoins(),
		DelegationRewardsShares:    NewRewardsDelegationShares(),
		PendingUndelegationRewards: sdk.NewDecCoins(),
		PendingSlashedRewards:      sdk.NewDecCoins(),
		WithdrawableRewards:        sdk.NewDecCoins(),
	}
}

func UpdateDecCoins(valueToUpdate sdk.DecCoins, deltaValue sdk.DecCoins) (sdk.DecCoins, error) {
	if len(deltaValue) == 0 {
		return valueToUpdate, nil
	}
	sum := valueToUpdate.Add(deltaValue...)
	if sum.IsAnyNegative() {
		return valueToUpdate, fmt.Errorf(
			"decCoins have negative values after the update, valueToUpdate:%s, deltaValue:%s",
			valueToUpdate, deltaValue,
		)
	}
	return sum, nil
}

func ScaleIntByDecimals(amount sdkmath.Int, decimals uint32) sdk.Dec {
	if decimals == 0 {
		return sdk.NewDecFromInt(amount)
	}
	divisor := sdkmath.NewIntWithDecimal(1, int(decimals)) // #nosec G115
	return sdk.NewDecFromInt(amount).QuoInt(divisor)
}

func UnscaleDecToInt(dec sdk.Dec, decimals uint32) sdkmath.Int {
	if decimals == 0 {
		return dec.TruncateInt()
	}
	multiplier := sdkmath.NewIntWithDecimal(1, int(decimals)) // 10^decimals
	return dec.MulInt(multiplier).TruncateInt()
}

// TruncateSDKDec truncates a sdk.Dec value to the specified number of decimal places without rounding.
// For example, truncating 1.23456789 to 4 decimal places will return 1.2345.
// This function is useful when only a fixed precision is allowed and rounding is not desired.
func TruncateSDKDec(dec sdk.Dec, decimal uint32) sdk.Dec {
	// Compute the multiplier: 10^decimal
	multiplier := sdkmath.NewIntWithDecimal(1, int(decimal))

	// Multiply the original decimal to shift the decimal point to the right
	decMultiplied := dec.MulInt(multiplier)

	// Truncate the result to remove all digits beyond the decimal
	truncated := decMultiplied.TruncateInt()

	// Divide back by the multiplier to restore the decimal point at the correct position
	return sdk.NewDecFromInt(truncated).QuoInt(multiplier)
}

// ValidateRewardAssetDenomination is the default validation function for the denomination of reward asset.
func ValidateRewardAssetDenomination(denomination string) error {
	// check if it contains the combined delimiter `/`, because denomination might be used in
	// a combined key.
	if strings.IndexByte(denomination, oracletypes.DelimiterForCombinedKey) >= 0 {
		return fmt.Errorf("invalid denomination %q: contains combined delimiter %q",
			denomination, string(oracletypes.DelimiterForCombinedKey))
	}
	return sdk.ValidateDenom(denomination)
}
