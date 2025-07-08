package reward

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

const (
	MethodIsRegisteredRewardToken = "isRegisteredRewardToken"
)

func (p Precompile) IsRegisteredRewardToken(
	ctx sdk.Context,
	_ common.Address,
	_ *vm.Contract,
	_ vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	var reqArgs IsRegisterRewardTokenArgs
	if err := method.Inputs.Copy(&reqArgs, args); err != nil {
		return nil, fmt.Errorf("error while unpacking args to IsRegisterRewardTokenArgs struct: %s", err)
	}
	_, rewardAssetID, err := addressToID(ctx, p.assetsKeeper, reqArgs.ClientChainID, reqArgs.Token)
	if err != nil {
		return nil, err
	}

	isRegistered := p.distributionKeeper.IsRegisteredRewardAsset(ctx, rewardAssetID)
	return method.Outputs.Pack(true, isRegistered)
}
