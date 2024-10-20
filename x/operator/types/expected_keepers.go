package types

import (
	sdkmath "cosmossdk.io/math"
	keytypes "github.com/ExocoreNetwork/exocore/types/keys"
	assetstype "github.com/ExocoreNetwork/exocore/x/assets/types"
	delegationkeeper "github.com/ExocoreNetwork/exocore/x/delegation/keeper"
	delegationtype "github.com/ExocoreNetwork/exocore/x/delegation/types"
	epochstypes "github.com/ExocoreNetwork/exocore/x/epochs/types"
	oracletype "github.com/ExocoreNetwork/exocore/x/oracle/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ OracleKeeper = MockOracle{}

type AssetsKeeper interface {
	GetStakingAssetInfo(
		ctx sdk.Context, assetID string,
	) (info *assetstype.StakingAssetInfo, err error)
	GetAssetsDecimal(
		ctx sdk.Context, assets map[string]interface{},
	) (decimals map[string]uint32, err error)
	IterateAssetsForOperator(
		ctx sdk.Context, isUpdate bool, operator string, assetsFilter map[string]interface{},
		f func(assetID string, state *assetstype.OperatorAssetInfo) error,
	) error
	ClientChainExists(ctx sdk.Context, index uint64) bool
	GetOperatorSpecifiedAssetInfo(ctx sdk.Context, operatorAddr sdk.Address, assetID string) (info *assetstype.OperatorAssetInfo, err error)
	GetAllStakingAssetsInfo(ctx sdk.Context) (allAssets []assetstype.StakingAssetInfo, err error)
}

var _ DelegationKeeper = &delegationkeeper.Keeper{}

type DelegationKeeper interface {
	IterateUndelegationsByOperator(
		ctx sdk.Context, operator string, heightFilter *uint64, isUpdate bool,
		opFunc func(undelegation *delegationtype.UndelegationRecord) error) error
	HasStakerList(ctx sdk.Context, operator, assetID string) bool
	GetStakersByOperator(
		ctx sdk.Context, operator, assetID string,
	) (delegationtype.StakerList, error)
	SetStakerShareToZero(
		ctx sdk.Context, operator, assetID string, stakerList delegationtype.StakerList,
	) error
	DeleteStakersListForOperator(ctx sdk.Context, operator, assetID string) error

	IterateDelegationsForStaker(ctx sdk.Context, stakerID string, opFunc delegationkeeper.DelegationOpFunc) error
}

type PriceChange struct {
	OriginalPrice sdkmath.Int
	NewPrice      sdkmath.Int
	Decimal       uint8
}

// Price represents the expected return type from the price Oracle
// the first field is the value and the second is the decimal
// it's same as the price type of ChainLink
type Price struct {
	Value   sdkmath.Int
	Decimal uint8
}

// OracleKeeper is the oracle interface expected by operator module
// These functions need to be implemented by the oracle module
type OracleKeeper interface {
	// GetSpecifiedAssetsPrice is a function to retrieve the asset price according to the
	// assetID.
	GetSpecifiedAssetsPrice(ctx sdk.Context, assetID string) (oracletype.Price, error)
	// GetMultipleAssetsPrices is a function to retrieve multiple assets prices according to the
	// assetID.
	GetMultipleAssetsPrices(ctx sdk.Context, assets map[string]interface{}) (map[string]oracletype.Price, error)
}

type MockOracle struct{}

func (MockOracle) GetSpecifiedAssetsPrice(_ sdk.Context, _ string) (oracletype.Price, error) {
	return oracletype.Price{
		Value:   sdkmath.NewInt(1),
		Decimal: 0,
	}, nil
}

func (MockOracle) GetMultipleAssetsPrices(_ sdk.Context, assets map[string]interface{}) (map[string]oracletype.Price, error) {
	ret := make(map[string]oracletype.Price, 0)
	for assetID := range assets {
		ret[assetID] = oracletype.Price{
			Value:   sdkmath.NewInt(1),
			Decimal: 0,
		}
	}
	return ret, nil
}

type AVSKeeper interface {
	// GetAVSSupportedAssets The ctx can be historical or current, depending on the state you
	// wish to retrieve. If the caller want to retrieve a historical assets info supported by
	// Avs, it needs to generate a historical context through calling
	// `ContextForHistoricalState` implemented in x/assets/types/general.go
	GetAVSSupportedAssets(ctx sdk.Context, avsAddr string) (map[string]interface{}, error)
	GetAVSSlashContract(ctx sdk.Context, avsAddr string) (string, error)
	// GetChainIDByAVSAddr converts the hex AVS address to the chainID.
	GetChainIDByAVSAddr(ctx sdk.Context, avsAddr string) (string, bool)
	// GetAVSMinimumSelfDelegation returns the USD value of minimum self delegation, which
	// is set for operator
	GetAVSMinimumSelfDelegation(ctx sdk.Context, avsAddr string) (sdkmath.LegacyDec, error)
	// GetEpochEndAVSs returns the AVS list where the current block marks the end of their epoch.
	// todo: maybe the epoch of different AVSs should be implemented in the AVS module,then
	// the other modules implement the EpochsHooks to trigger state updating.
	GetEpochEndAVSs(ctx sdk.Context, epochIdentifier string, epochNumber int64) []string
	// IsAVS returns true if the address is a registered AVS address.
	IsAVS(ctx sdk.Context, addr string) (bool, error)
	IsAVSByChainID(ctx sdk.Context, chainID string) (bool, string)
	GetAVSEpochInfo(ctx sdk.Context, addr string) (*epochstypes.EpochInfo, error)
}

type SlashKeeper interface {
	IsOperatorFrozen(ctx sdk.Context, addr sdk.AccAddress) bool
}

type OperatorHooks interface {
	// This hook is called when an operator declares the consensus key for the provided chain.
	AfterOperatorKeySet(
		ctx sdk.Context, addr sdk.AccAddress, chainID string,
		pubKey keytypes.WrappedConsKey,
	)
	// This hook is called when an operator's consensus key is replaced for a chain.
	AfterOperatorKeyReplaced(
		ctx sdk.Context, addr sdk.AccAddress, oldKey keytypes.WrappedConsKey,
		newKey keytypes.WrappedConsKey, chainID string,
	)
	// This hook is called when an operator initiates the removal of a consensus key for a
	// chain.
	AfterOperatorKeyRemovalInitiated(
		ctx sdk.Context, addr sdk.AccAddress, chainID string, key keytypes.WrappedConsKey,
	)
}
