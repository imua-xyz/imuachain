package types

import (
	"fmt"
	"sort"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type (
	CommonAVSRewards   []CommonAVSRewardData
	CompoundingRewards []CompoundingRewardsPerAsset
)

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

// NegativeDecCoins returns a set of coins with all amount negative.
func NegativeDecCoins(coins sdk.DecCoins) sdk.DecCoins {
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
// nolint: dupl
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

// Negative returns a set of CommonAVSRewardData with all rewards amount negative.
func (crs CommonAVSRewards) Negative() CommonAVSRewards {
	res := make([]CommonAVSRewardData, 0, len(crs))
	for _, avsReward := range crs {
		res = append(res, CommonAVSRewardData{
			AVSAddress: avsReward.AVSAddress,
			Rewards:    NegativeDecCoins(avsReward.Rewards),
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
	diff := crs.safeAdd(avsRewardsB.Negative())
	return diff, diff.IsAnyNegative()
}

// CalculateRewardRatio calculates the rewards ratio， the receiver of this function should be the total rewards.
func (crs CommonAVSRewards) CalculateRewardRatio(totalDelegatedAmount sdk.Dec) (CommonAVSRewards, error) {
	if !totalDelegatedAmount.IsPositive() {
		return nil, fmt.Errorf("CalculateRewardRatio, total delegated amount isn't positive, value:%s", totalDelegatedAmount)
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
		return nil, fmt.Errorf("MulDecTruncate, the multiplier is negative, value:%s", multiplier)
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
// nolint: dupl
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
			Rewards:            CommonAVSRewards(compoundingReward.Rewards).Negative(),
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
