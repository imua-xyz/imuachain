package types

import "fmt"

type DeltaAVSRewardAssetState AVSRewardAssetState

type OperatorRewardProportions []*OperatorRewardProportion

// String implements the Stringer interface for OperatorRewardProportions. It returns a
// human-readable representation of operator reward proportions
func (ps OperatorRewardProportions) String() string {
	if len(ps) == 0 {
		return ""
	}

	out := ""
	for _, p := range ps {
		proportionStr := fmt.Sprintf("%v:%v", p.OperatorAddr, p.RewardProportion.String())
		out += fmt.Sprintf("%v,", proportionStr)
	}

	return out[:len(out)-1]
}

// AppendUniqueDelegationKey appends a new delegation key to StakeChangeDelegations
// only if it's not already present.
func (s *StakeChangeDelegations) AppendUniqueDelegationKey(newKey string) {
	// Check if the newKey already exists in the slice
	for _, key := range s.DelegationKeys {
		if key == newKey {
			// If the key already exists, do not append it
			return
		}
	}
	// Append the newKey if it's not already present
	s.DelegationKeys = append(s.DelegationKeys, newKey)
}
