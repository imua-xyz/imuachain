package common

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

type PrecompileCommonTxFunc func(ctx sdk.Context, origin common.Address, contract *vm.Contract,
	stateDB vm.StateDB, method *abi.Method, args []interface{}) ([]byte, error)
