package reward

import (
	"bytes"
	"embed"
	"fmt"

	feedistribution "github.com/imua-xyz/imuachain/x/feedistribution/keeper"

	"github.com/cometbft/cometbft/libs/log"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	cmn "github.com/evmos/evmos/v16/precompiles/common"
	imuacmn "github.com/imua-xyz/imuachain/precompiles/common"
	assetskeeper "github.com/imua-xyz/imuachain/x/assets/keeper"
)

var _ vm.PrecompiledContract = &Precompile{}

// Embed abi json file to the executable binary. Needed when importing as dependency.
//
//go:embed abi.json
var f embed.FS

// Precompile defines the precompiled contract for reward.
type Precompile struct {
	cmn.Precompile
	assetsKeeper       assetskeeper.Keeper
	distributionKeeper feedistribution.Keeper
}

// NewPrecompile creates a new reward Precompile instance as a
// PrecompiledContract interface.
func NewPrecompile(
	stakingStateKeeper assetskeeper.Keeper,
	distributionKeeper feedistribution.Keeper,
	authzKeeper authzkeeper.Keeper,
) (*Precompile, error) {
	abiBz, err := f.ReadFile("abi.json")
	if err != nil {
		return nil, fmt.Errorf("error loading the reward ABI %s", err)
	}

	newAbi, err := abi.JSON(bytes.NewReader(abiBz))
	if err != nil {
		return nil, fmt.Errorf(cmn.ErrInvalidABI, err)
	}

	return &Precompile{
		Precompile: cmn.Precompile{
			ABI:                  newAbi,
			AuthzKeeper:          authzKeeper,
			KvGasConfig:          storetypes.KVGasConfig(),
			TransientKVGasConfig: storetypes.TransientGasConfig(),
			ApprovalExpiration:   cmn.DefaultExpirationDuration, // should be configurable in the future.
			Addr:                 common.HexToAddress("0x0000000000000000000000000000000000000806"),
		},
		distributionKeeper: distributionKeeper,
		assetsKeeper:       stakingStateKeeper,
	}, nil
}

// RequiredGas calculates the precompiled contract's base gas rate.
func (p Precompile) RequiredGas(input []byte) uint64 {
	method, err := p.MethodById(input)
	if err != nil {
		return 0
	}

	return p.Precompile.RequiredGas(input, p.IsTransaction(method.Name))
}

// Run executes the precompiled contract reward methods defined in the ABI.
func (p Precompile) Run(evm *vm.EVM, contract *vm.Contract, readOnly bool) (bz []byte, err error) {
	ctx, stateDB, method, initialGas, args, err := p.RunSetup(evm, contract, readOnly, p.IsTransaction)
	if err != nil {
		return nil, err
	}

	// This handles any out of gas errors that may occur during the execution of a precompile tx or query.
	// It avoids panics and returns the out of gas error so the EVM can continue gracefully.
	defer cmn.HandleGasError(ctx, contract, initialGas, &err)()

	var logError error
	cc := ctx
	writeFunc := func() {}
	if p.IsTransaction(method.Name) {
		cc, writeFunc = ctx.CacheContext()
	}
	var precompileCommonFunc imuacmn.PrecompileCommonTxFunc
	switch method.Name {
	// transactions
	case MethodClaimReward:
		precompileCommonFunc = p.ClaimReward
	case MethodWithdrawReward:
		precompileCommonFunc = p.WithdrawReward
	case MethodWithdrawIMUATokenReward:
		precompileCommonFunc = p.WithdrawIMUATokenReward
	case MethodWithdrawCommission:
		precompileCommonFunc = p.WithdrawCommission
	case MethodWithdrawIMUATokenCommission:
		precompileCommonFunc = p.WithdrawIMUATokenCommission
	case MethodRegisterRewardToken:
		precompileCommonFunc = p.RegisterRewardToken
	case MethodUpdateRewardToken:
		precompileCommonFunc = p.UpdateRewardToken
	case MethodSetAVSRewardDistribution:
		precompileCommonFunc = p.SetAVSRewardDistribution
	case MethodSetAVSEpochReward:
		precompileCommonFunc = p.SetAVSEpochReward
	case MethodSetOperatorRewardProportions:
		precompileCommonFunc = p.SetOperatorRewardProportions
	case MethodSetAVSRewardParams:
		precompileCommonFunc = p.SetAVSRewardParams
	case MethodFundAVSReward:
		precompileCommonFunc = p.FundAVSReward
	// queries
	case MethodIsRegisteredRewardToken:
		precompileCommonFunc = p.IsRegisteredRewardToken
	default:
		return nil, fmt.Errorf("unsupported reward method %s", method.Name)
	}

	bz, err = precompileCommonFunc(cc, evm.Origin, contract, stateDB, method, args)
	if err != nil {
		logError = err
		// for failed cases we expect it returns bool value instead of error
		// this is a workaround because the error returned by precompile can not be caught in EVM
		// see https://github.com/imua-xyz/imuachain/issues/70
		bz, err = packErrorOutput(method)
	}

	if logError != nil {
		ctx.Logger().Error(
			"return error when calling reward precompile",
			"module", "reward precompile",
			"method", method.Name,
			"err", logError,
		)
	} else {
		writeFunc()
	}

	cost := ctx.GasMeter().GasConsumed() - initialGas
	if !contract.UseGas(cost) {
		return nil, vm.ErrOutOfGas
	}

	return bz, nil
}

// IsTransaction checks if the given methodName corresponds to a transaction or query.
func (Precompile) IsTransaction(methodName string) bool {
	switch methodName {
	case MethodClaimReward, MethodWithdrawReward,
		MethodWithdrawIMUATokenReward, MethodWithdrawCommission,
		MethodWithdrawIMUATokenCommission,
		MethodRegisterRewardToken, MethodUpdateRewardToken,
		MethodSetAVSRewardDistribution, MethodSetAVSEpochReward,
		MethodSetOperatorRewardProportions, MethodSetAVSRewardParams,
		MethodFundAVSReward:
		return true
	case MethodIsRegisteredRewardToken:
		return false
	default:
		// this panic is safe to perform because the `init` function
		// below forces developers to add all methods to the switch statement.
		panic(fmt.Sprintf("unknown method: %s", methodName))
	}
}

func init() {
	// dummy instance
	var p Precompile
	if err := imuacmn.ValidateIsTx(f, p.IsTransaction); err != nil {
		panic(err)
	}
}

// Logger returns a precompile-specific logger.
func (p Precompile) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("Imuachain module", "reward")
}
