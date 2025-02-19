package reward

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	exocmn "github.com/imua-xyz/imuachain/precompiles/common"
	"github.com/imua-xyz/imuachain/x/assets/types"
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
	// check the invalidation of caller contract
	err := p.assetsKeeper.CheckExocoreGatewayAddr(ctx, contract.CallerAddress)
	if err != nil {
		return nil, fmt.Errorf(exocmn.ErrContractCaller, err.Error())
	}

	rewardParam, err := p.GetRewardParamsFromInputs(ctx, args)
	if err != nil {
		return nil, err
	}

	err = p.rewardKeeper.RewardForWithdraw(ctx, rewardParam)
	if err != nil {
		return nil, err
	}
	// get the latest asset state of staker to return.
	stakerID, assetID := types.GetStakerIDAndAssetID(rewardParam.ClientChainLzID, rewardParam.WithdrawRewardAddress, rewardParam.AssetsAddress)
	info, err := p.assetsKeeper.GetStakerSpecifiedAssetInfo(ctx, stakerID, assetID)
	if err != nil {
		return nil, err
	}
	return method.Outputs.Pack(true, info.TotalDepositAmount.BigInt())
}
