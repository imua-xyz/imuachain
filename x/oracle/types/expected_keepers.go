package types

import (
	time "time"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/ethereum/go-ethereum/common"
	keytypes "github.com/imua-xyz/imuachain/types/keys"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/vm"
	evmtypes "github.com/evmos/evmos/v16/x/evm/types"
)

// AccountKeeper defines the expected account keeper used for simulations (noalias)
type AccountKeeper interface {
	GetAccount(ctx sdk.Context, addr sdk.AccAddress) types.AccountI
	// Methods imported from account should be defined here
}

// BankKeeper defines the expected interface needed to retrieve account balances.
type BankKeeper interface {
	SpendableCoins(ctx sdk.Context, addr sdk.AccAddress) sdk.Coins
	// Methods imported from bank should be defined here
}

// DelegationKeeper defines the expected interfaces needed to update nst token balance change
type DelegationKeeper interface {
	UpdateNSTBalance(ctx sdk.Context, stakerID, assetID string, amount sdkmath.Int) error
}

type AssetsKeeper interface {
	GetAssetsDecimal(ctx sdk.Context, assets map[string]struct{}) (decimals map[string]uint32, err error)
	// GetGatewayAddresses returns the set of gateway contract addresses on Imuachain (EVM).
	// Version 1 oracle-bridge uses the first gateway as the delivery target.
	GetGatewayAddresses(ctx sdk.Context) ([]common.Address, error)
}

// OperatorKeeper defines the subset of x/operator keeper APIs needed for bridge checkpoint signing.
type OperatorKeeper interface {
	// GetOperatorConsKeyForChainID returns the consensus key registered by an operator for a given chainID.
	GetOperatorConsKeyForChainID(ctx sdk.Context, opAccAddr sdk.AccAddress, chainID string) (bool, keytypes.WrappedConsKey, error)
	// GetOperatorAddressForChainIDAndConsAddr maps (chainID, consensus address) → operator AccAddress.
	GetOperatorAddressForChainIDAndConsAddr(ctx sdk.Context, chainID string, consAddr sdk.ConsAddress) (bool, sdk.AccAddress)
}

type SlashingKeeper interface {
	JailUntil(sdk.Context, sdk.ConsAddress, time.Time)
}

// EVMKeeper defines the subset of x/evm keeper APIs needed for oracle-bridge gateway delivery.
type EVMKeeper interface {
	ApplyMessage(ctx sdk.Context, msg core.Message, tracer vm.EVMLogger, commit bool) (*evmtypes.MsgEthereumTxResponse, error)
}
