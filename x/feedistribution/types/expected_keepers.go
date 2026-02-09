package types

import (
	"context"

	sdkmath "cosmossdk.io/math"
	assetstype "github.com/imua-xyz/imuachain/x/assets/types"
	delegationtype "github.com/imua-xyz/imuachain/x/delegation/types"
	dogfoodtypes "github.com/imua-xyz/imuachain/x/dogfood/types"
	operatortypes "github.com/imua-xyz/imuachain/x/operator/types"
	oracletype "github.com/imua-xyz/imuachain/x/oracle/types"

	epochsTypes "github.com/imua-xyz/imuachain/x/epochs/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
)

// EpochsKeeper represents the expected keeper interface for the epochs module.
type EpochsKeeper interface {
	GetEpochInfo(ctx sdk.Context, identifier string) (epochsTypes.EpochInfo, bool)
}

type FeeDistributionHooks interface{}

// AccountKeeper defines the expected interface for the Account module.
type AccountKeeper interface {
	GetAccount(sdk.Context, sdk.AccAddress) types.AccountI // only used for simulation
	// Methods imported from account should be defined here
	GetModuleAddress(name string) sdk.AccAddress
	GetModuleAccount(ctx sdk.Context, name string) types.ModuleAccountI
	// TODO remove with genesis 2-phases refactor https://github.com/cosmos/cosmos-sdk/issues/2862
	SetModuleAccount(sdk.Context, types.ModuleAccountI)
}

// BankKeeper defines the expected interface for the Bank module.
type BankKeeper interface {
	// MintCoins(ctx sdk.Context, moduleName string, amt sdk.Coins) error
	GetAllBalances(ctx sdk.Context, addr sdk.AccAddress) sdk.Coins

	SpendableCoins(ctx sdk.Context, addr sdk.AccAddress) sdk.Coins

	SendCoinsFromModuleToModule(ctx sdk.Context, senderModule, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx sdk.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromAccountToModule(ctx sdk.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error

	BlockedAddr(addr sdk.AccAddress) bool
	// IsSendEnabledDenom(ctx sdk.Context, denom string) bool
}

// OperatorKeeper represents the expected keeper interface for the operator module.
type OperatorKeeper interface {
	GetImpactfulEpochsAndAVSsForOperator(ctx sdk.Context, operatorAddr string) ([]string, []string, error)
	GetOperatorAddressForChainIDAndConsAddr(ctx sdk.Context, chainID string, consAddr sdk.ConsAddress) (bool, sdk.AccAddress)
	IsOptedOutAndEffective(ctx sdk.Context, operatorAddr, avsAddr string) bool
	GetOptedInfo(ctx sdk.Context, operatorAddr, avsAddr string) (info *operatortypes.OptedInfo, err error)
	GetAVSUSDValue(ctx sdk.Context, avsAddr string) (sdkmath.LegacyDec, error)
	IterateOperatorUSDValuesForAVS(ctx sdk.Context, avsAddr string, isUpdate bool, opFunc func(operator string, optedUSDValues *operatortypes.OperatorOptedUSDValue) error) error
	GetRecentEndedEpochAVSAssets(ctx sdk.Context, avsAddr string) ([]string, error)
	GetOperatorOptedUSDValue(ctx sdk.Context, avsAddr, operatorAddr string) (operatortypes.OperatorOptedUSDValue, error)
	HasOperatorAssetUSDValue(ctx sdk.Context, epochIdentifier, operator, assetID string) bool
	GetOperatorAssetUSDValue(ctx sdk.Context, epochIdentifier, operator, assetID string) (sdkmath.LegacyDec, error)
	IterateOperatorRewardsUSDValue(ctx sdk.Context, receivingAVS, operator string, isUpdate bool,
		opFunc func(avs, symbol string, usdValue *operatortypes.DecValueField) (bool, bool, error),
	) error
	OperatorInfo(ctx sdk.Context, addr string) (info *operatortypes.OperatorInfo, err error)
	SetOperatorRewardUSDValue(ctx sdk.Context, receivingAVS, rewardSourceAVS, operator, symbol string, amount sdkmath.LegacyDec) error
	RemoveAllStaleOperatorRewardUSDs(ctx sdk.Context, receivingAVS, operator string, keysToKeep map[string]interface{}) error
	IsCompoundRewardsDisabled(ctx sdk.Context, addr string) (bool, error)
}

// AVSKeeper represents the expected keeper interface for the avs module.
type AVSKeeper interface {
	GetEpochEndAVSs(ctx sdk.Context, epochIdentifier string, endingEpochNumber int64) []string
	GetEpochsUsedByAllAVSs(ctx sdk.Context) []string
	IsAVS(ctx sdk.Context, addr string) (bool, error)
	GetAVSEpochInfo(ctx sdk.Context, addr string) (*epochsTypes.EpochInfo, error)
}

// AssetsKeeper represents the expected keeper interface for the assets module.
type AssetsKeeper interface {
	GetStakingAssetInfo(ctx sdk.Context, assetID string) (info *assetstype.StakingAssetInfo, err error)
	GetOperatorSpecifiedAssetInfo(ctx sdk.Context, operatorAddr sdk.AccAddress, assetID string) (info *assetstype.OperatorAssetInfo, err error)
	IsStakingAsset(ctx sdk.Context, assetID string) bool
	IsOperatorAssetExist(ctx sdk.Context, operatorAddr sdk.AccAddress, assetID string) bool
}

// DelegationKeeper represents the expected keeper interface for the delegation module.
type DelegationKeeper interface {
	GetDelegationInfoWithAmounts(ctx sdk.Context, stakerID, assetID, operatorAddr string) (*delegationtype.DelegationAmounts, sdkmath.Int, sdkmath.Int, error)
	IterateDelegationsForStaker(ctx sdk.Context, stakerID string, opFunc delegationtype.DelegationOpFunc) error
	GetStakersByOperator(ctx sdk.Context, operator, assetID string) (delegationtype.StakerList, error)
	DelegateTo(ctx sdk.Context, params *delegationtype.DelegationOrUndelegationParams) (sdkmath.LegacyDec, sdkmath.Int, error)
	UndelegateFrom(ctx sdk.Context, params *delegationtype.DelegationOrUndelegationParams) error
}

type SlashKeeper interface {
	IsOperatorFrozen(ctx sdk.Context, opAddr sdk.AccAddress) bool
}

type StakingKeeper interface {
	GetLastTotalPower(ctx sdk.Context) sdkmath.Int
	GetAllImuachainValidators(
		ctx sdk.Context,
	) (validators []dogfoodtypes.ImuachainValidator)
}

type OracleKeeper interface {
	// GetSpecifiedAssetsPrice is a function to retrieve the asset price according to the
	// assetID.
	GetSpecifiedAssetsPrice(ctx sdk.Context, assetID string) (oracletype.Price, error)
}

// ParamSubspace defines the expected Subspace interface for parameters.
type ParamSubspace interface {
	Get(context.Context, []byte, interface{})
	Set(context.Context, []byte, interface{})
}

// PoolKeeper defines the expected interface needed to fund & distribute pool balances.
type PoolKeeper interface {
	FundCommunityPool(ctx context.Context, amount sdk.Coins, sender sdk.AccAddress) error
	DistributeFromCommunityPool(ctx context.Context, amount sdk.Coins, receiveAddr sdk.AccAddress) error
	GetCommunityPool(ctx context.Context) (sdk.Coins, error)
	SetToDistribute(ctx context.Context, amount sdk.Coins, addr string) error
}
