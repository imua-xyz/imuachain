package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// AppendMany appends a variable number of byte slices together
func AppendMany(byteses ...[]byte) (out []byte) {
	for _, bytes := range byteses {
		out = append(out, bytes...)
	}
	return out
}

// ChainIDWithLenKey returns the key with the following format:
// bytePrefix | len(chainId) | chainId
// This is similar to Solidity's ABI encoding.
func ChainIDWithLenKey(chainID string) []byte {
	chainIDL := len(chainID)
	return AppendMany(
		// Append the chainID length
		// #nosec G701
		sdk.Uint64ToBigEndian(uint64(chainIDL)),
		// Append the chainID
		[]byte(chainID),
	)
}

func GetSpecifiedVotingPower(operator string, votingPowerSet []*OperatorVotingPower) *OperatorVotingPower {
	for idx := range votingPowerSet {
		tmpVp := votingPowerSet[idx]
		if tmpVp.OperatorAddr == operator {
			return tmpVp
		}
	}
	return nil
}
