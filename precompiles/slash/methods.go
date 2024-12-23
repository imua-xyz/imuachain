package slash

import (
	"fmt"

	exocmn "github.com/ExocoreNetwork/exocore/precompiles/common"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

const (
	// MethodSlash defines the ABI method name for the slash
	//  transaction.
	MethodSlash = "submitSlash"
)

// SubmitSlash Slash assets to the staker, that will change the state in slash module.
func (p Precompile) SubmitSlash(
	ctx sdk.Context,
	_ common.Address,
	contract *vm.Contract,
	_ vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	// check the invalidation of caller contract
	authorized, err := p.assetsKeeper.IsAuthorizedGateway(ctx, contract.CallerAddress)
	if err != nil || !authorized {
		return nil, fmt.Errorf(exocmn.ErrContractCaller)
	}

	slashParam, err := p.GetSlashParamsFromInputs(ctx, args)
	if err != nil {
		return nil, err
	}

	err = p.slashKeeper.Slash(ctx, slashParam)
	if err != nil {
		return nil, err
	}
	return method.Outputs.Pack(true)
}
