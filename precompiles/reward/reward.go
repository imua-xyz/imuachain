package reward

import (
	"bytes"
	"embed"
	"fmt"
	"math/big"

	"github.com/cometbft/cometbft/libs/log"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	cmn "github.com/evmos/evmos/v16/precompiles/common"
	assetskeeper "github.com/imua-xyz/imuachain/x/assets/keeper"
	rewardkeeper "github.com/imua-xyz/imuachain/x/reward/keeper"
)

var _ vm.PrecompiledContract = &Precompile{}

// Embed abi json file to the executable binary. Needed when importing as dependency.
//
//go:embed abi.json
var f embed.FS

// Precompile defines the precompiled contract for reward.
type Precompile struct {
	cmn.Precompile
	assetsKeeper assetskeeper.Keeper
	rewardKeeper rewardkeeper.Keeper
}

// NewPrecompile creates a new reward Precompile instance as a
// PrecompiledContract interface.
func NewPrecompile(
	stakingStateKeeper assetskeeper.Keeper,
	rewardKeeper rewardkeeper.Keeper,
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
		},
		rewardKeeper: rewardKeeper,
		assetsKeeper: stakingStateKeeper,
	}, nil
}

// Address defines the address of the reward compile contract.
// address: 0x0000000000000000000000000000000000000806
func (p Precompile) Address() common.Address {
	return common.HexToAddress("0x0000000000000000000000000000000000000806")
}

// RequiredGas calculates the precompiled contract's base gas rate.
func (p Precompile) RequiredGas(input []byte) uint64 {
	methodID := input[:4]

	method, err := p.MethodById(methodID)
	if err != nil {
		// This should never happen since this method is going to fail during Run
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

	if err := stateDB.Commit(); err != nil {
		return nil, err
	}

	if method.Name == MethodReward {
		bz, err = p.Reward(ctx, evm.Origin, contract, stateDB, method, args)
	}

	if err != nil {
		// for failed cases we expect it returns bool value instead of error
		// this is a workaround because the error returned by precompile can not be caught in EVM
		// see https://github.com/imua-xyz/imuachain/issues/70
		// TODO: we should figure out root cause and fix this issue to make precompiles work normally
		return method.Outputs.Pack(false, new(big.Int))
	}

	cost := ctx.GasMeter().GasConsumed() - initialGas

	if !contract.UseGas(cost) {
		return nil, vm.ErrOutOfGas
	}

	return bz, nil
}

// IsTransaction checks if the given methodID corresponds to a transaction or query.
//
// Available reward transactions are:
//   - reward
func (Precompile) IsTransaction(methodID string) bool {
	switch methodID {
	case MethodReward:
		return true
	default:
		return false
	}
}

// Logger returns a precompile-specific logger.
func (p Precompile) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("Imuachain module", "reward")
}
