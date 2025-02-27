package types

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
