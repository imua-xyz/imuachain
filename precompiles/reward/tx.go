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
	MethodSetStakerRewardParams       = "setStakerRewardParams"
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
	MethodUndelegateReward             = "undelegateReward"
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
		MethodSetAVSRewardParams, MethodFundAVSReward,
		MethodSetStakerRewardParams, MethodUndelegateReward:
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

	var wrapper WithdrawRewardArgsWrapper
	if err := method.Inputs.Copy(&wrapper, args); err != nil {
		return nil, fmt.Errorf("error while unpacking args to WithdrawRewardArgsWrapper struct: %s", err)
	}
	withdrawRewardArgs := wrapper.Params
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

	var wrapper WithdrawIMUATokenRewardArgsWrapper
	if err := method.Inputs.Copy(&wrapper, args); err != nil {
		return nil, fmt.Errorf("error while unpacking args to WithdrawIMUATokenRewardArgsWrapper struct: %s", err)
	}
	withdrawIMUATokenRewardArgs := wrapper.Params
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

func (p Precompile) SetStakerRewardParams(
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
	var setStakerRewardParamsArgs SetStakerRewardParamsArgs
	if err := method.Inputs.Copy(&setStakerRewardParamsArgs, args); err != nil {
		return nil, fmt.Errorf("error while unpacking args to SetStakerRewardParamsArgs struct: %s", err)
	}
	_, stakerID, err := addressToID(ctx, p.assetsKeeper, setStakerRewardParamsArgs.ClientChainLzID,
		setStakerRewardParamsArgs.StakerAddress)
	if err != nil {
		return nil, err
	}
	rewardParams := feedistributiontypes.StakerRewardParams{
		RedelegateReward:       setStakerRewardParamsArgs.RedelegateReward,
		RedelegateOperatorAddr: setStakerRewardParamsArgs.RedelegateOperator,
	}
	err = rewardParams.Validate()
	if err != nil {
		return nil, fmt.Errorf("invalid staker reward parameters, stakerID:%s,err:%s", stakerID, err)
	}
	err = p.distributionKeeper.SetStakerRewardParams(ctx, stakerID, rewardParams)
	if err != nil {
		return nil, fmt.Errorf("failed to set staker reward parameters, stakerID:%s,err:%s", stakerID, err)
	}

	return method.Outputs.Pack(true)
}

func (p Precompile) UndelegateReward(
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
	var wrapper UndelegateRewardArgsWrapper
	if err := method.Inputs.Copy(&wrapper, args); err != nil {
		return nil, fmt.Errorf("error while unpacking args to UndelegateRewardArgsWrapper struct: %s", err)
	}
	undelegateRewardArgs := wrapper.Params
	if undelegateRewardArgs.OpAmount == nil || undelegateRewardArgs.OpAmount.Cmp(big.NewInt(0)) <= 0 {
		return nil, fmt.Errorf("UndelegateReward: invalid undelegation amount:%v", undelegateRewardArgs.OpAmount)
	}
	_, stakerID, err := addressToID(ctx, p.assetsKeeper, undelegateRewardArgs.ClientChainLzID, undelegateRewardArgs.StakerAddress)
	if err != nil {
		return nil, err
	}
	_, rewardAssetID, err := addressToID(ctx, p.assetsKeeper, undelegateRewardArgs.RewardAssetChainLzID, undelegateRewardArgs.AssetAddress)
	if err != nil {
		return nil, err
	}
	// the input operator address is cosmos accAddress type,so we need to check the length and decode it through Bench32
	if len(undelegateRewardArgs.OperatorAddr) != assetstype.ImuachainOperatorAddrLength {
		return nil, fmt.Errorf(imuacmn.ErrInputOperatorAddrLength, len(undelegateRewardArgs.OperatorAddr), assetstype.ImuachainOperatorAddrLength)
	}
	operatorAccAddr, err := sdk.AccAddressFromBech32(undelegateRewardArgs.OperatorAddr)
	if err != nil {
		return nil, fmt.Errorf("error occurred when parse acc address from Bech32,the addr is:%s, error:%s", undelegateRewardArgs.OperatorAddr, err.Error())
	}
	err = p.distributionKeeper.UndelegateClaimedRewards(ctx, stakerID, rewardAssetID, operatorAccAddr, undelegateRewardArgs.InstantUnbond, sdkmath.NewIntFromBigInt(undelegateRewardArgs.OpAmount))
	if err != nil {
		return nil, err
	}
	return method.Outputs.Pack(true)
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
		return nil, fmt.Errorf("error while unpacking args to WithdrawCommissionArgs struct: %s", err)
	}
	if withdrawCommissionArgs.OpAmount == nil || withdrawCommissionArgs.OpAmount.Cmp(big.NewInt(0)) == -1 {
		return nil, fmt.Errorf("WithdrawCommission: invalid withdraw amount:%v", withdrawCommissionArgs.OpAmount)
	}
	// the input operator address is EVM address type
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
	// the input operator address is EVM address type
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
	var wrapper RegisterRewardTokenArgsWrapper
	if err := method.Inputs.Copy(&wrapper, args); err != nil {
		return nil, fmt.Errorf("error while unpacking args to RegisterRewardTokenArgsWrapper struct: %s", err)
	}
	registerRewardTokenArgs := wrapper.Params
	avsAddr := strings.ToLower(contract.CallerAddress.String())
	// check the input args
	rewardAssetAddr, rewardAssetID, err := addressToID(ctx, p.assetsKeeper, registerRewardTokenArgs.ClientChainID, registerRewardTokenArgs.Token)
	if err != nil {
		return nil, err
	}
	if registerRewardTokenArgs.DenominationExponent > assetstype.MaxDecimal {
		return nil, fmt.Errorf(imuacmn.ErrInvalidDenominationExponent, registerRewardTokenArgs.DenominationExponent, assetstype.MaxDecimal)
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
	if p.assetsKeeper.IsStakingAsset(ctx, rewardAssetID) {
		stakingAssetInfo, err := p.assetsKeeper.GetStakingAssetInfo(ctx, rewardAssetID)
		if err != nil {
			return nil, err
		}

		if stakingAssetInfo.AssetBasicInfo.Decimals != uint32(registerRewardTokenArgs.Decimals) ||
			stakingAssetInfo.AssetBasicInfo.Name != registerRewardTokenArgs.Name ||
			stakingAssetInfo.AssetBasicInfo.Symbol != registerRewardTokenArgs.Symbol {
			return nil, fmt.Errorf(imuacmn.ErrAssetBasicInfoMismatch, registerRewardTokenArgs.Name, registerRewardTokenArgs.Symbol, registerRewardTokenArgs.Decimals)
		}
	}
	err = p.distributionKeeper.SetAVSRewardAssets(ctx, avsAddr, []feedistributiontypes.AVSRewardAssetInfo{
		{
			AssetInfo: assetstype.AssetInfo{
				Name:             registerRewardTokenArgs.Name,
				Symbol:           registerRewardTokenArgs.Symbol,
				Address:          hexutil.Encode(rewardAssetAddr),
				Decimals:         uint32(registerRewardTokenArgs.Decimals),
				LayerZeroChainID: uint64(registerRewardTokenArgs.ClientChainID),
				MetaInfo:         registerRewardTokenArgs.MetaData,
			},
			RewardDenomination:   registerRewardTokenArgs.Denomination,
			DenominationExponent: uint32(registerRewardTokenArgs.DenominationExponent),
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
	var wrapper AVSRewardDistributionInfoArgsWrapper
	if err := method.Inputs.Copy(&wrapper, args); err != nil {
		return nil, fmt.Errorf("error while unpacking args to AVSRewardDistributionInfoArgsWrapper struct: %s", err)
	}
	rewardDistributionInfoArgs := wrapper.RewardDistribution
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
	rewardAssetInfo, err := p.distributionKeeper.GetAVSRewardAsset(ctx, avsAddr, rewardAssetID)
	if err != nil {
		return nil, fmt.Errorf("can't find the reward asset for the input avs,rewardAssetID:%s,avs:%s", rewardAssetID, avsAddr)
	}
	fundAmountDec := feedistributiontypes.ScaleIntByDecimals(
		sdkmath.NewIntFromBigInt(fundAVSRewardArgs.OpAmount), rewardAssetInfo.RewardAssetInfo.DenominationExponent)
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
