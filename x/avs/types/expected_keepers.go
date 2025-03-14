package types

import (
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/evmos/evmos/v16/x/evm/statedb"
	assetstype "github.com/imua-xyz/imuachain/x/assets/types"
	epochstypes "github.com/imua-xyz/imuachain/x/epochs/types"
	operatortypes "github.com/imua-xyz/imuachain/x/operator/types"
)

// EpochsKeeper represents the expected keeper interface for the epochs module.
type EpochsKeeper interface {
	GetEpochInfo(sdk.Context, string) (epochstypes.EpochInfo, bool)
}

// AccountKeeper defines the expected account keeper used for simulations (noalias)
type AccountKeeper interface {
	GetAccount(ctx sdk.Context, addr sdk.AccAddress) types.AccountI
	GetModuleAddress(name string) sdk.AccAddress
}

// BankKeeper defines the expected interface needed to retrieve account balances.
type BankKeeper interface {
	SpendableCoins(ctx sdk.Context, addr sdk.AccAddress) sdk.Coins
	SendCoinsFromModuleToAccount(
		ctx sdk.Context,
		senderModule string,
		recipientAddr sdk.AccAddress,
		amt sdk.Coins,
	) error
	SendCoinsFromModuleToModule(
		ctx sdk.Context,
		senderModule, recipientModule string,
		amt sdk.Coins,
	) error
	SendCoinsFromAccountToModule(
		ctx sdk.Context,
		senderAddr sdk.AccAddress,
		recipientModule string,
		amt sdk.Coins,
	) error
}

// EVMKeeper defines the expected EVM keeper interface used on erc20
type EVMKeeper interface {
	SetAccount(ctx sdk.Context, addr common.Address, account statedb.Account) error
	SetCode(ctx sdk.Context, codeHash, code []byte)
	GetNewContractNonce(sdk.Context) uint64
}

// OperatorKeeper represents the expected keeper interface for the operator module.
type OperatorKeeper interface {
	IsOperator(ctx sdk.Context, addr sdk.AccAddress) bool
	OptIn(ctx sdk.Context, operatorAddress sdk.AccAddress, avsAddr string) error
	OptOut(ctx sdk.Context, operatorAddress sdk.AccAddress, avsAddr string) (err error)
	GetOptedInOperatorListByAVS(ctx sdk.Context, avsAddr string) ([]string, error)
	GetOperatorOptedUSDValue(ctx sdk.Context, avsAddr, operatorAddr string) (operatortypes.OperatorOptedUSDValue, error)
	GetAVSUSDValue(ctx sdk.Context, avsAddr string) (sdkmath.LegacyDec, error)
	SetOperatorInfo(ctx sdk.Context, addr string, info *operatortypes.OperatorInfo) (err error)
	OperatorInfo(ctx sdk.Context, addr string) (info *operatortypes.OperatorInfo, err error)
}

// AssetsKeeper represents the expected keeper interface for the assets module.
type AssetsKeeper interface {
	GetStakingAssetInfo(
		ctx sdk.Context, assetID string,
	) (info *assetstype.StakingAssetInfo, err error)
	IsStakingAsset(sdk.Context, string) bool
}
