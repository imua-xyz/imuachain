package types

import (
	"fmt"
	"sort"
	"strings"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type (
	DeltaAVSRewardAssetState AVSRewardAssetState
)

type OperatorRewardProportions []OperatorRewardProportion

type (
	CommonAVSRewards           []CommonAVSRewardData
	EpochRewardsAndProportions struct {
		Rewards                   sdk.DecCoins
		OperatorRewardProportions []OperatorRewardProportion
	}
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
		pair := strings.SplitN(part, ":", 2)
		if len(pair) != 2 {
			return nil, fmt.Errorf("invalid format for operator proportion: %q", part)
		}

		addr := strings.TrimSpace(pair[0])
		propStr := strings.TrimSpace(pair[1])

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
			return fmt.Errorf("avsReward %s amount is not positive", crs[0])
		}
		return nil
	default:
		// check single avsReward case
		if err := (CommonAVSRewards{crs[0]}).Validate(); err != nil {
			return err
		}

		lowAVSAddr := crs[0].AVSAddress
		seenAVSAddr := make(map[string]bool)
		seenAVSAddr[lowAVSAddr] = true

		for _, avsReward := range crs[1:] {
			if seenAVSAddr[avsReward.AVSAddress] {
				return fmt.Errorf("duplicate avs address %s", avsReward.AVSAddress)
			}
			if avsReward.AVSAddress <= lowAVSAddr {
				return fmt.Errorf("avs address %s is not sorted", avsReward.AVSAddress)
			}
			if !avsReward.IsPositive() {
				return fmt.Errorf("avsReward %s amount is not positive", avsReward.AVSAddress)
			}

			// we compare each avsReward against the last avs address
			lowAVSAddr = avsReward.AVSAddress
			seenAVSAddr[avsReward.AVSAddress] = true
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

		case 0: // avs reward A address == avs reward B address
			res := avsRewardA.Add(avsRewardB)
			if !res.IsZeroRewards() {
				sum = append(sum, res)
			}

			indexA++
			indexB++

		case 1: // coin A denom > coin B denom
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

// CalculateRewards calculates the rewards, the receiver of this function should be the rewards ratio.
func (crs CommonAVSRewards) CalculateRewards(delegatedAmount sdk.Dec) (CommonAVSRewards, error) {
	if delegatedAmount.IsNegative() {
		return nil, ErrInvalidInputParameter.Wrapf("CalculateRewards, the delegated amount is negative, value:%s", delegatedAmount)
	}
	ret := make([]CommonAVSRewardData, 0)
	if delegatedAmount.IsZero() {
		return ret, nil
	}
	for _, avsRewardRatio := range crs {
		// note: necessary to truncate so we don't allow withdrawing more rewards than owed
		rewards := avsRewardRatio.Rewards.MulDecTruncate(delegatedAmount)
		ret = append(ret, CommonAVSRewardData{
			AVSAddress: avsRewardRatio.AVSAddress,
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
