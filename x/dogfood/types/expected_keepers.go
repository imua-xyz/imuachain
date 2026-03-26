package types

import (
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/common"
	keytypes "github.com/imua-xyz/imuachain/types/keys"
	avstypes "github.com/imua-xyz/imuachain/x/avs/types"
	delegationtype "github.com/imua-xyz/imuachain/x/delegation/types"
	epochstypes "github.com/imua-xyz/imuachain/x/epochs/types"
	operatortypes "github.com/imua-xyz/imuachain/x/operator/types"
)

// EpochsKeeper represents the expected keeper interface for the epochs module.
type EpochsKeeper interface {
	GetEpochInfo(sdk.Context, string) (epochstypes.EpochInfo, bool)
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
	// GetOrCalculateOperatorUSDValues is used to get the self staking value for the operator
	GetOrCalculateOperatorUSDValues(sdk.Context, sdk.AccAddress, string) (operatortypes.OperatorOptedUSDValue, error)
	InitGenesisVPSnapshot(ctx sdk.Context) error
	FreezeOperator(ctx sdk.Context, addr sdk.AccAddress) error
	IsOperatorFrozen(ctx sdk.Context, addr sdk.AccAddress) bool
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
	RegisterAVSWithChainID(sdk.Context, *avstypes.AVSRegisterOrDeregisterParams) (bool, common.Address, error)
	IsAVSByChainID(ctx sdk.Context, chainID string) (bool, string)
	UpdateAVSInfo(ctx sdk.Context, params *avstypes.AVSRegisterOrDeregisterParams) error
}

type SlashingKeeper interface {
	GetValidatorSigningInfo(
		sdk.Context, sdk.ConsAddress,
	) (slashingtypes.ValidatorSigningInfo, bool)
	SetValidatorSigningInfo(
		sdk.Context, sdk.ConsAddress, slashingtypes.ValidatorSigningInfo,
	)
	SetValidatorMissedBlockBitArray(
		sdk.Context, sdk.ConsAddress, int64, bool,
	)
	IterateValidatorMissedBlockBitArray(
		sdk.Context, sdk.ConsAddress, func(int64, bool) bool,
	)
	DowntimeJailDuration(sdk.Context) time.Duration
	ResetValidatorSigningInfo(sdk.Context, sdk.ConsAddress)
	ClearValidatorMissedBlockBitArray(sdk.Context, sdk.ConsAddress)
	IsTombstoned(sdk.Context, sdk.ConsAddress) bool
}
