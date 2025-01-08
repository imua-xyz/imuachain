package types

import (
	"github.com/ExocoreNetwork/exocore/utils"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ChainIDWithLenKey returns the key with the following format:
// bytePrefix | len(chainId) | chainId
// This is similar to Solidity's ABI encoding.
func ChainIDWithLenKey(chainID string) []byte {
	chainIDL := len(chainID)
	return utils.AppendMany(
		// Append the chainID length
		// #nosec G701
		sdk.Uint64ToBigEndian(uint64(chainIDL)),
		// Append the chainID
		[]byte(chainID),
	)
}

func GetSpecifiedVotingPower(operator string, votingPowerSet []*OperatorVotingPower) *OperatorVotingPower {
	for _, vp := range votingPowerSet {
		if vp.OperatorAddr == operator {
			return vp
		}
	}
	return nil
}
