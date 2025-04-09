package reward

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

const (
	// MethodReward defines the ABI method name for the reward
	//  transaction.
	MethodReward = "claimReward"
)

// Reward assets to the staker, that will change the state in reward module.
func (p Precompile) Reward(
	ctx sdk.Context,
	_ common.Address,
	contract *vm.Contract,
	_ vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	return nil, nil
}
