package types

import (
	"cosmossdk.io/math"
	keytypes "github.com/ExocoreNetwork/exocore/types/keys"
	avstypes "github.com/ExocoreNetwork/exocore/x/avs/types"
	delegationtype "github.com/ExocoreNetwork/exocore/x/delegation/types"
	epochsTypes "github.com/ExocoreNetwork/exocore/x/epochs/types"
	operatortypes "github.com/ExocoreNetwork/exocore/x/operator/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/common"
)

// EpochsKeeper represents the expected keeper interface for the epochs module.
type EpochsKeeper interface {
	GetEpochInfo(sdk.Context, string) (epochsTypes.EpochInfo, bool)
}

// DogfoodHooks represents the event hooks for dogfood module. Ideally, these should
// match those of the staking module but for now it is only a subset of them. The side effects
// of calling the other hooks are not relevant to running the chain, so they can be skipped.
type DogfoodHooks interface {
	AfterValidatorBonded(
		sdk.Context, sdk.ConsAddress, sdk.ValAddress,
	) error
	AfterValidatorRemoved(
		sdk.Context, sdk.ConsAddress, sdk.ValAddress,
	) error
	AfterValidatorCreated(
		sdk.Context, sdk.ValAddress,
	) error
}

// OperatorKeeper represents the expected keeper interface for the operator module.
type OperatorKeeper interface {
	// use a shorted undelegation period if the operator is opting out
	IsOperatorRemovingKeyFromChainID(sdk.Context, sdk.AccAddress, string) bool
	// complete the removal when done
	CompleteOperatorKeyRemovalForChainID(sdk.Context, sdk.AccAddress, string) error
	// reverse lookup for slashing
	GetOperatorAddressForChainIDAndConsAddr(
		sdk.Context, string, sdk.ConsAddress,
	) (bool, sdk.AccAddress)
	// impl_sdk
	IsOperatorJailedForChainID(sdk.Context, sdk.ConsAddress, string) bool
	Jail(sdk.Context, sdk.ConsAddress, string)
	Unjail(sdk.Context, sdk.ConsAddress, string)
	SlashWithInfractionReason(
		sdk.Context, sdk.AccAddress, int64,
		int64, sdk.Dec, stakingtypes.Infraction,
	) math.Int
	ValidatorByConsAddrForChainID(
		ctx sdk.Context, consAddr sdk.ConsAddress, chainID string,
	) (stakingtypes.Validator, bool)
	// at each epoch, get the list and create validator update
	GetActiveOperatorsForChainID(
		sdk.Context, string,
	) ([]sdk.AccAddress, []keytypes.WrappedConsKey)
	// get vote power
	GetVotePowerForChainID(
		sdk.Context, []sdk.AccAddress, string,
	) ([]int64, error)
	// prune slashing-related reverse lookup when matured
	DeleteOperatorAddressForChainIDAndConsAddr(
		ctx sdk.Context, chainID string, consAddr sdk.ConsAddress,
	)
	// at each epoch, the current key becomes the "previous" key
	// for further key set function calls
	ClearPreviousConsensusKeys(ctx sdk.Context, chainID string)
	GetOperatorConsKeyForChainID(
		sdk.Context, sdk.AccAddress, string,
	) (bool, keytypes.WrappedConsKey, error)
	GetOperatorPrevConsKeyForChainID(
		sdk.Context, sdk.AccAddress, string,
	) (bool, keytypes.WrappedConsKey, error)
	// OptInWithConsKey is used at genesis to opt in with a consensus key
	OptInWithConsKey(
		sdk.Context, sdk.AccAddress, string, keytypes.WrappedConsKey,
	) error
	// GetOrCalculateOperatorUSDValues is used to get the self staking value for the operator
	GetOrCalculateOperatorUSDValues(sdk.Context, sdk.AccAddress, string) (operatortypes.OperatorOptedUSDValue, error)
	GetOptedInAVSForOperator(ctx sdk.Context, operatorAddr string) ([]string, error)
	CalculateUSDValueForStaker(ctx sdk.Context, stakerID, avsAddr string, operator sdk.AccAddress) (math.LegacyDec, error)
	OperatorInfo(ctx sdk.Context, addr string) (info *operatortypes.OperatorInfo, err error)
	HasSlash(ctx sdk.Context, avsAddr string) bool
}

// DelegationKeeper represents the expected keeper interface for the delegation module.
type DelegationKeeper interface {
	IncrementUndelegationHoldCount(sdk.Context, []byte) error
	DecrementUndelegationHoldCount(sdk.Context, []byte) error
	GetStakersByOperator(ctx sdk.Context, operator, assetID string) (delegationtype.StakerList, error)
}

// AssetsKeeper represents the expected keeper interface for the assets module.
type AssetsKeeper interface {
	IsStakingAsset(sdk.Context, string) bool
}

type AVSKeeper interface {
	RegisterAVSWithChainID(sdk.Context, *avstypes.AVSRegisterOrDeregisterParams) (common.Address, error)
	IsAVSByChainID(ctx sdk.Context, chainID string) (bool, string)
	GetAVSSupportedAssets(ctx sdk.Context, avsAddr string) (map[string]interface{}, error)
	UpdateAVSInfo(ctx sdk.Context, params *avstypes.AVSRegisterOrDeregisterParams) error
}
