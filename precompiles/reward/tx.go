package reward

import (
	"fmt"
	"math/big"
	"strings"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/vm"
	imuacmn "github.com/imua-xyz/imuachain/precompiles/common"
	"github.com/imua-xyz/imuachain/utils"
	assetskeeper "github.com/imua-xyz/imuachain/x/assets/keeper"
	assetstype "github.com/imua-xyz/imuachain/x/assets/types"
	feedistributiontypes "github.com/imua-xyz/imuachain/x/feedistribution/types"
)

const (
	// The following methods are used to claim and withdraw rewards for stakers and operators.
	MethodClaimReward                 = "claimReward"
	MethodWithdrawReward              = "withdrawReward"
	MethodWithdrawIMUATokenReward     = "withdrawIMUATokenReward"
	MethodWithdrawCommission          = "withdrawCommission"
	MethodWithdrawIMUATokenCommission = "withdrawIMUATokenCommission"

	// The following methods are used to set reward assets and distribution information for AVSs.
	MethodRegisterRewardToken          = "registerRewardToken"
	MethodUpdateRewardToken            = "updateRewardToken"
	MethodSetAVSRewardDistribution     = "setAVSRewardDistribution"
	MethodSetAVSEpochReward            = "setAVSEpochReward"
	MethodSetOperatorRewardProportions = "setOperatorRewardProportions"
	MethodSetAVSRewardParams           = "setAVSRewardParams"
	MethodFundAVSReward                = "fundAVSReward"
)

func addressToID(ctx sdk.Context, assetKeeper assetskeeper.Keeper, chainLzID uint32, address []byte) ([]byte, string, error) {
	chainInfo, err := assetKeeper.GetClientChainInfoByIndex(ctx, uint64(chainLzID))
	if err != nil {
		return nil, "", err
	}
	if len(address) < int(chainInfo.AddressLength) {
		return nil, "", fmt.Errorf(imuacmn.ErrInvalidAddrLength, len(address), chainInfo.AddressLength)
	}

	chainLzIDStr := hexutil.EncodeUint64(uint64(chainLzID))
	ID := strings.Join([]string{hexutil.Encode(address[:int(chainInfo.AddressLength)]), chainLzIDStr}, utils.DelimiterForID)

	return address[:int(chainInfo.AddressLength)], ID, nil
}

func packErrorOutput(method *abi.Method) ([]byte, error) {
	switch method.Name {
	case MethodClaimReward, MethodRegisterRewardToken,
		MethodUpdateRewardToken, MethodSetAVSRewardDistribution,
		MethodSetAVSEpochReward, MethodSetOperatorRewardProportions,
		MethodSetAVSRewardParams, MethodFundAVSReward:
		return method.Outputs.Pack(false)
	case MethodWithdrawReward, MethodWithdrawCommission:
		return method.Outputs.Pack(false, new(big.Int))
	case MethodWithdrawIMUATokenReward, MethodWithdrawIMUATokenCommission:
		return method.Outputs.Pack(false, new(big.Int), new(big.Int))
	case MethodIsRegisteredRewardToken:
		return method.Outputs.Pack(false, false)
	default:
		return nil, fmt.Errorf("packErrorOutput: unsupported reward method %s", method.Name)
	}
}

func (p Precompile) ClaimReward(
	ctx sdk.Context,
	_ common.Address,
	contract *vm.Contract,
	_ vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	// check the invalidation of caller contract,the caller must be Imuachain LzApp contract
	authorized, err := p.assetsKeeper.IsAuthorizedGateway(ctx, contract.CallerAddress)
	if err != nil || !authorized {
		return nil, fmt.Errorf(imuacmn.ErrContractCaller)
	}

	var claimRewardArgs ClaimRewardArgs
	if err := method.Inputs.Copy(&claimRewardArgs, args); err != nil {
		return nil, fmt.Errorf("error while unpacking args to ClaimRewardArgs struct: %s", err)
	}

	_, stakerID, err := addressToID(ctx, p.assetsKeeper, claimRewardArgs.ClientChainLzID, claimRewardArgs.StakerAddress)
	if err != nil {
		return nil, err
	}
	// todo: the total claimed rewards might be returned to the caller in future.
	_, err = p.distributionKeeper.ClaimDelegationRewards(ctx, stakerID)
	if err != nil {
		return nil, err
	}
	return method.Outputs.Pack(true)
}

func (p Precompile) WithdrawReward(
	ctx sdk.Context,
	_ common.Address,
	contract *vm.Contract,
	_ vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	// check the invalidation of caller contract,the caller must be Imuachain LzApp contract
	authorized, err := p.assetsKeeper.IsAuthorizedGateway(ctx, contract.CallerAddress)
	if err != nil || !authorized {
		return nil, fmt.Errorf(imuacmn.ErrContractCaller)
	}

	var withdrawRewardArgs WithdrawRewardArgs
	if err := method.Inputs.Copy(&withdrawRewardArgs, args); err != nil {
		return nil, fmt.Errorf("error while unpacking args to WithdrawRewardArgs struct: %s", err)
	}
	if withdrawRewardArgs.OpAmount == nil || withdrawRewardArgs.OpAmount.Cmp(big.NewInt(0)) == -1 {
		return nil, fmt.Errorf("WithdrawReward: invalid withdraw amount:%v", withdrawRewardArgs.OpAmount)
	}
	_, stakerID, err := addressToID(ctx, p.assetsKeeper, withdrawRewardArgs.ClientChainLzID, withdrawRewardArgs.StakerAddress)
	if err != nil {
		return nil, err
	}
	_, rewardAssetID, err := addressToID(ctx, p.assetsKeeper, withdrawRewardArgs.RewardAssetChainLzID, withdrawRewardArgs.AssetAddress)
	if err != nil {
		return nil, err
	}
	if rewardAssetID == assetstype.ImuachainAssetID {
		return nil, fmt.Errorf("reward asset is the IMUA token issued by the dogfood AVS,rewardAssetID:%s", rewardAssetID)
	}
	if withdrawRewardArgs.DoClaim {
		_, err = p.distributionKeeper.ClaimDelegationRewards(ctx, stakerID)
		if err != nil {
			return nil, err
		}
	}

	actualWithdrawAmount, _, err := p.distributionKeeper.WithdrawStakerRewards(ctx, stakerID, rewardAssetID,
		sdk.NewIntFromBigInt(withdrawRewardArgs.OpAmount), nil)
	if err != nil {
		return nil, err
	}
	return method.Outputs.Pack(true, actualWithdrawAmount.BigInt())
}

func (p Precompile) WithdrawIMUATokenReward(
	ctx sdk.Context,
	_ common.Address,
	contract *vm.Contract,
	_ vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	// check the invalidation of caller contract,the caller must be Imuachain LzApp contract
	authorized, err := p.assetsKeeper.IsAuthorizedGateway(ctx, contract.CallerAddress)
	if err != nil || !authorized {
		return nil, fmt.Errorf(imuacmn.ErrContractCaller)
	}

	var withdrawIMUATokenRewardArgs WithdrawIMUATokenRewardArgs
	if err := method.Inputs.Copy(&withdrawIMUATokenRewardArgs, args); err != nil {
		return nil, fmt.Errorf("error while unpacking args to WithdrawIMUATokenRewardArgs struct: %s", err)
	}
	if withdrawIMUATokenRewardArgs.OpAmount == nil || withdrawIMUATokenRewardArgs.OpAmount.Cmp(big.NewInt(0)) == -1 {
		return nil, fmt.Errorf("WithdrawIMUATokenReward: invalid withdraw amount:%v", withdrawIMUATokenRewardArgs.OpAmount)
	}
	if len(withdrawIMUATokenRewardArgs.ReceiptAddress) != common.AddressLength {
		return nil, fmt.Errorf("invalid receipt EVM address, length:%d,expectedLength:%d ",
			len(withdrawIMUATokenRewardArgs.ReceiptAddress), common.AddressLength)
	}
	receiptAccAddr := sdk.AccAddress(withdrawIMUATokenRewardArgs.ReceiptAddress)
	_, stakerID, err := addressToID(ctx, p.assetsKeeper, withdrawIMUATokenRewardArgs.ClientChainLzID,
		withdrawIMUATokenRewardArgs.StakerAddress)
	if err != nil {
		return nil, err
	}

	if withdrawIMUATokenRewardArgs.DoClaim {
		_, err = p.distributionKeeper.ClaimDelegationRewards(ctx, stakerID)
		if err != nil {
			return nil, err
		}
	}

	actualWithdrawAmount, withdrawAmountFromDogfood, err := p.distributionKeeper.WithdrawStakerRewards(
		ctx, stakerID, assetstype.ImuachainAssetID,
		sdk.NewIntFromBigInt(withdrawIMUATokenRewardArgs.OpAmount), receiptAccAddr)
	if err != nil {
		return nil, err
	}
	return method.Outputs.Pack(true, actualWithdrawAmount.BigInt(), withdrawAmountFromDogfood.BigInt())
}

func (p Precompile) WithdrawCommission(
	ctx sdk.Context,
	_ common.Address,
	contract *vm.Contract,
	_ vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	// check the invalidation of caller contract,the caller must be Imuachain LzApp contract
	authorized, err := p.assetsKeeper.IsAuthorizedGateway(ctx, contract.CallerAddress)
	if err != nil || !authorized {
		return nil, fmt.Errorf(imuacmn.ErrContractCaller)
	}

	var withdrawCommissionArgs WithdrawCommissionArgs
	if err := method.Inputs.Copy(&withdrawCommissionArgs, args); err != nil {
		return nil, fmt.Errorf("error while unpacking args to WithdrawRewardArgs struct: %s", err)
	}
	if withdrawCommissionArgs.OpAmount == nil || withdrawCommissionArgs.OpAmount.Cmp(big.NewInt(0)) == -1 {
		return nil, fmt.Errorf("WithdrawCommission: invalid withdraw amount:%v", withdrawCommissionArgs.OpAmount)
	}
	if len(withdrawCommissionArgs.OperatorAddress) != common.AddressLength {
		return nil, fmt.Errorf("invalid operator EVM address, length:%d,expectedLength:%d ",
			len(withdrawCommissionArgs.OperatorAddress), common.AddressLength)
	}
	operatorAccAddr := sdk.AccAddress(withdrawCommissionArgs.OperatorAddress)
	_, rewardAssetID, err := addressToID(ctx, p.assetsKeeper, withdrawCommissionArgs.RewardAssetChainLzID,
		withdrawCommissionArgs.AssetAddress)
	if err != nil {
		return nil, err
	}
	if rewardAssetID == assetstype.ImuachainAssetID {
		return nil, fmt.Errorf("commission asset is the IMUA token issued by the dogfood AVS,rewardAssetID:%s", rewardAssetID)
	}

	actualWithdrawAmount, _, err := p.distributionKeeper.WithdrawOperatorCommission(ctx, rewardAssetID,
		sdk.NewIntFromBigInt(withdrawCommissionArgs.OpAmount), operatorAccAddr, nil)
	if err != nil {
		return nil, err
	}
	return method.Outputs.Pack(true, actualWithdrawAmount.BigInt())
}

func (p Precompile) WithdrawIMUATokenCommission(
	ctx sdk.Context,
	_ common.Address,
	contract *vm.Contract,
	_ vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	// check the invalidation of caller contract,the caller must be Imuachain LzApp contract
	authorized, err := p.assetsKeeper.IsAuthorizedGateway(ctx, contract.CallerAddress)
	if err != nil || !authorized {
		return nil, fmt.Errorf(imuacmn.ErrContractCaller)
	}

	var withdrawIMUATokenCommissionArgs WithdrawIMUATokenCommissionArgs
	if err := method.Inputs.Copy(&withdrawIMUATokenCommissionArgs, args); err != nil {
		return nil, fmt.Errorf("error while unpacking args to WithdrawIMUATokenCommissionArgs struct: %s", err)
	}
	if withdrawIMUATokenCommissionArgs.OpAmount == nil || withdrawIMUATokenCommissionArgs.OpAmount.Cmp(big.NewInt(0)) == -1 {
		return nil, fmt.Errorf("WithdrawIMUATokenCommission: invalid withdraw amount:%v",
			withdrawIMUATokenCommissionArgs.OpAmount)
	}
	if len(withdrawIMUATokenCommissionArgs.OperatorAddress) != common.AddressLength {
		return nil, fmt.Errorf("invalid operator EVM address, length:%d,expectedLength:%d ",
			len(withdrawIMUATokenCommissionArgs.OperatorAddress), common.AddressLength)
	}
	operatorAccAddr := sdk.AccAddress(withdrawIMUATokenCommissionArgs.OperatorAddress)

	if len(withdrawIMUATokenCommissionArgs.ReceiptAddress) != common.AddressLength {
		return nil, fmt.Errorf("invalid receipt EVM address, length:%d,expectedLength:%d ",
			len(withdrawIMUATokenCommissionArgs.ReceiptAddress), common.AddressLength)
	}
	receiptAccAddr := sdk.AccAddress(withdrawIMUATokenCommissionArgs.ReceiptAddress)

	actualWithdrawAmount, withdrawAmountFromDogfood, err := p.distributionKeeper.WithdrawOperatorCommission(
		ctx, assetstype.ImuachainAssetID,
		sdk.NewIntFromBigInt(withdrawIMUATokenCommissionArgs.OpAmount), operatorAccAddr, receiptAccAddr)
	if err != nil {
		return nil, err
	}
	return method.Outputs.Pack(true, actualWithdrawAmount.BigInt(), withdrawAmountFromDogfood.BigInt())
}

// The following methods are used to set reward assets and distribution information for AVSs.

func (p Precompile) RegisterRewardToken(
	ctx sdk.Context,
	_ common.Address,
	contract *vm.Contract,
	_ vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	var registerRewardTokenArgs RegisterRewardTokenArgs
	if err := method.Inputs.Copy(&registerRewardTokenArgs, args); err != nil {
		return nil, fmt.Errorf("error while unpacking args to RegisterRewardTokenArgs struct: %s", err)
	}
	avsAddr := strings.ToLower(contract.CallerAddress.String())
	// check the input args
	rewardAssetAddr, _, err := addressToID(ctx, p.assetsKeeper, registerRewardTokenArgs.ClientChainID, registerRewardTokenArgs.Token)
	if err != nil {
		return nil, err
	}
	if registerRewardTokenArgs.Decimals > assetstype.MaxDecimal {
		return nil, fmt.Errorf(imuacmn.ErrInvalidDecimal, registerRewardTokenArgs.Decimals, assetstype.MaxDecimal)
	}
	if len(registerRewardTokenArgs.Name) > assetstype.MaxChainTokenNameLength {
		return nil, fmt.Errorf(imuacmn.ErrInvalidNameLength, registerRewardTokenArgs.Name, len(registerRewardTokenArgs.Name), assetstype.MaxChainTokenNameLength)
	}
	if len(registerRewardTokenArgs.MetaData) > assetstype.MaxChainTokenMetaInfoLength {
		return nil, fmt.Errorf(imuacmn.ErrInvalidMetaInfoLength, registerRewardTokenArgs.MetaData,
			len(registerRewardTokenArgs.MetaData), assetstype.MaxChainTokenMetaInfoLength)
	}
	err = p.distributionKeeper.SetAVSRewardAssets(ctx, avsAddr, []assetstype.AssetInfo{
		{
			Name:             registerRewardTokenArgs.Name,
			Symbol:           registerRewardTokenArgs.Symbol,
			Address:          hexutil.Encode(rewardAssetAddr),
			Decimals:         uint32(registerRewardTokenArgs.Decimals),
			LayerZeroChainID: uint64(registerRewardTokenArgs.ClientChainID),
			MetaInfo:         registerRewardTokenArgs.MetaData,
		},
	})
	if err != nil {
		return nil, err
	}
	return method.Outputs.Pack(true)
}

func (p Precompile) UpdateRewardToken(
	ctx sdk.Context,
	_ common.Address,
	contract *vm.Contract,
	_ vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	var updateRewardTokenArgs UpdateRewardTokenArgs
	if err := method.Inputs.Copy(&updateRewardTokenArgs, args); err != nil {
		return nil, fmt.Errorf("error while unpacking args to UpdateRewardTokenArgs struct: %s", err)
	}
	avsAddr := strings.ToLower(contract.CallerAddress.String())
	// check the input args
	if len(updateRewardTokenArgs.MetaData) > assetstype.MaxChainTokenMetaInfoLength {
		return nil, fmt.Errorf(imuacmn.ErrInvalidMetaInfoLength, updateRewardTokenArgs.MetaData,
			len(updateRewardTokenArgs.MetaData), assetstype.MaxChainTokenMetaInfoLength)
	}
	_, assetID, err := addressToID(ctx, p.assetsKeeper, updateRewardTokenArgs.ClientChainID, updateRewardTokenArgs.Token)
	if err != nil {
		return nil, err
	}
	err = p.distributionKeeper.UpdateAVSRewardAssetMetaInfo(ctx, avsAddr, assetID, updateRewardTokenArgs.MetaData)
	if err != nil {
		return nil, err
	}
	return method.Outputs.Pack(true)
}

func (p Precompile) SetAVSRewardDistribution(
	ctx sdk.Context,
	_ common.Address,
	contract *vm.Contract,
	_ vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	var rewardDistributionInfoArgs AVSRewardDistributionInfoArgs
	if err := method.Inputs.Copy(&rewardDistributionInfoArgs, args); err != nil {
		return nil, fmt.Errorf("error while unpacking args to AVSRewardDistributionInfoArgs struct: %s", err)
	}
	avsAddr := strings.ToLower(contract.CallerAddress.String())

	protoRewardCoins, err := ABIRewardCoins(rewardDistributionInfoArgs.RewardCoins).ToProtoStruct(
		ctx, avsAddr, p.distributionKeeper)
	if err != nil {
		return nil, err
	}
	protoOperatorRewardProportions, err := ABIOperatorRewardProportions(rewardDistributionInfoArgs.OperatorRewardProportions).ToProtoStruct()
	if err != nil {
		return nil, err
	}
	err = p.distributionKeeper.SetAVSRewardDistribution(ctx, avsAddr, feedistributiontypes.AVSRewardDistribution{
		Rewards:                   protoRewardCoins,
		OperatorRewardProportions: protoOperatorRewardProportions,
	})
	if err != nil {
		return nil, err
	}
	return method.Outputs.Pack(true)
}

func (p Precompile) SetAVSEpochReward(
	ctx sdk.Context,
	_ common.Address,
	contract *vm.Contract,
	_ vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	var setAVSEpochRewardArgs SetAVSEpochRewardArgs
	if err := method.Inputs.Copy(&setAVSEpochRewardArgs, args); err != nil {
		return nil, fmt.Errorf("error while unpacking args to SetAVSEpochRewardArgs struct: %s", err)
	}
	avsAddr := strings.ToLower(contract.CallerAddress.String())

	protoRewardCoins, err := ABIRewardCoins(setAVSEpochRewardArgs.EpochRewards).ToProtoStruct(
		ctx, avsAddr, p.distributionKeeper)
	if err != nil {
		return nil, err
	}
	err = p.distributionKeeper.SetAVSEpochRewardExclusive(ctx, avsAddr, protoRewardCoins)
	if err != nil {
		return nil, err
	}
	return method.Outputs.Pack(true)
}

func (p Precompile) SetOperatorRewardProportions(
	ctx sdk.Context,
	_ common.Address,
	contract *vm.Contract,
	_ vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	var setOperatorRewardProportionsArgs SetOperatorRewardProportionsArgs
	if err := method.Inputs.Copy(&setOperatorRewardProportionsArgs, args); err != nil {
		return nil, fmt.Errorf("error while unpacking args to SetOperatorRewardProportionsArgs struct: %s", err)
	}
	avsAddr := strings.ToLower(contract.CallerAddress.String())

	protoOperatorRewardProportions, err := ABIOperatorRewardProportions(
		setOperatorRewardProportionsArgs.OperatorRewardProportions).ToProtoStruct()
	if err != nil {
		return nil, err
	}
	err = p.distributionKeeper.SetAVSRewardProportionsExclusive(ctx, avsAddr, protoOperatorRewardProportions)
	if err != nil {
		return nil, err
	}
	return method.Outputs.Pack(true)
}

func (p Precompile) SetAVSRewardParams(
	ctx sdk.Context,
	_ common.Address,
	contract *vm.Contract,
	_ vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	var setAVSRewardParams SetAVSRewardParamsArgs
	if err := method.Inputs.Copy(&setAVSRewardParams, args); err != nil {
		return nil, fmt.Errorf("error while unpacking args to SetAVSRewardParamsArgs struct: %s", err)
	}
	avsAddr := strings.ToLower(contract.CallerAddress.String())

	err := p.distributionKeeper.SetAVSRewardParam(ctx, avsAddr, feedistributiontypes.AVSRewardParam{
		CustomRewardInflation: setAVSRewardParams.IsCustomRewardInflation,
		CustomOperatorRatio:   setAVSRewardParams.IsCustomOperatorRatio,
	})
	if err != nil {
		return nil, err
	}
	return method.Outputs.Pack(true)
}

func (p Precompile) FundAVSReward(
	ctx sdk.Context,
	_ common.Address,
	contract *vm.Contract,
	_ vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	// check the invalidation of caller contract,the caller must be Imuachain LzApp contract
	authorized, err := p.assetsKeeper.IsAuthorizedGateway(ctx, contract.CallerAddress)
	if err != nil || !authorized {
		return nil, fmt.Errorf(imuacmn.ErrContractCaller)
	}

	var fundAVSRewardArgs FundAVSRewardArgs
	if err := method.Inputs.Copy(&fundAVSRewardArgs, args); err != nil {
		return nil, fmt.Errorf("error while unpacking args to FundAVSRewardArgs struct: %s", err)
	}
	if fundAVSRewardArgs.OpAmount == nil || !(fundAVSRewardArgs.OpAmount.Cmp(big.NewInt(0)) == 1) {
		return nil, fmt.Errorf("FundAVSReward: invalid fund amount:%v", fundAVSRewardArgs.OpAmount)
	}
	_, rewardAssetID, err := addressToID(ctx, p.assetsKeeper, fundAVSRewardArgs.RewardAssetChainLzID, fundAVSRewardArgs.AssetAddress)
	if err != nil {
		return nil, err
	}
	if rewardAssetID == assetstype.ImuachainAssetID {
		return nil, fmt.Errorf("can't fund the IMUA token for dogfood AVS,rewardAssetID:%s", rewardAssetID)
	}
	avsAddr := strings.ToLower(fundAVSRewardArgs.AVSAddress.String())
	rewardAssetInfo, err := p.distributionKeeper.GetAVSRewardAssetInfo(ctx, avsAddr, rewardAssetID)
	if err != nil {
		return nil, fmt.Errorf("can't find the reward asset for the input avs,rewardAssetID:%s,avs:%s", rewardAssetID, avsAddr)
	}
	fundAmountDec := feedistributiontypes.ScaleIntByDecimals(
		sdkmath.NewIntFromBigInt(fundAVSRewardArgs.OpAmount), rewardAssetInfo.AssetBasicInfo.Decimals)
	if !fundAmountDec.IsPositive() {
		return nil, fmt.Errorf("FundAVSReward: invalid fund amount after converting to decimal:%s", fundAmountDec)
	}
	err = p.distributionKeeper.UpdateAVSRewardAssetState(ctx,
		strings.ToLower(fundAVSRewardArgs.AVSAddress.String()),
		rewardAssetID,
		&feedistributiontypes.DeltaAVSRewardAssetState{
			RewardPoolBalance: fundAmountDec,
			RewardPoolTotal:   fundAmountDec,
		})
	if err != nil {
		return nil, err
	}
	return method.Outputs.Pack(true)
}
