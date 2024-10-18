package types

import (
	time "time"

	keytypes "github.com/ExocoreNetwork/exocore/types/keys"
	avstypes "github.com/ExocoreNetwork/exocore/x/avs/types"
	epochstypes "github.com/ExocoreNetwork/exocore/x/epochs/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/common"
)

// AVSKeeper represents the expected keeper interface for the AVS module.
type AVSKeeper interface {
	RegisterAVSWithChainID(
		sdk.Context, *avstypes.AVSRegisterOrDeregisterParams,
	) (common.Address, error)
	IsAVSByChainID(sdk.Context, string) (bool, string)
	DeleteAVSInfo(sdk.Context, string) error
	GetEpochEndChainIDs(sdk.Context, string, int64) []string
}

// EpochsKeeper represents the expected keeper interface for the epochs module.
type EpochsKeeper interface {
	GetEpochInfo(sdk.Context, string) (epochstypes.EpochInfo, bool)
}

// StakingKeeper represents the expected keeper interface for the staking module.
type StakingKeeper interface {
	UnbondingTime(sdk.Context) time.Duration
}

// OperatorKeeper represents the expected keeper interface for the operator module.
type OperatorKeeper interface {
	GetOperatorConsKeyForChainID(sdk.Context, sdk.AccAddress, string) (bool, keytypes.WrappedConsKey, error)
	IsOperatorRemovingKeyFromChainID(sdk.Context, sdk.AccAddress, string) bool
	GetActiveOperatorsForChainID(sdk.Context, string) ([]sdk.AccAddress, []keytypes.WrappedConsKey)
	GetVotePowerForChainID(sdk.Context, []sdk.AccAddress, string) ([]int64, error)
	GetOperatorAddressForChainIDAndConsAddr(
		sdk.Context, string, sdk.ConsAddress,
	) (bool, sdk.AccAddress)
	DeleteOperatorAddressForChainIDAndConsAddr(
		ctx sdk.Context, chainID string, consAddr sdk.ConsAddress,
	)
	// compared to slashing forwarded by Tendermint, this function doesn't have the vote power parameter.
	// instead it contains the avs address for which the slashing is being executed. the interface is
	// subject to change during implementation. It should check that the validator isn't permanently
	// kicked, and it should jail the validator for the provided duration.
	ApplySlashForHeight(
		ctx sdk.Context, operatorAccAddress sdk.AccAddress, avsAddress string,
		height uint64, fraction sdk.Dec, infraction stakingtypes.Infraction,
		jailDuration time.Duration,
	) error
	GetChainIDsForOperator(sdk.Context, string) ([]string, error)
}

// DelegationKeeper represents the expected keeper interface for the delegation module.
type DelegationKeeper interface {
	IncrementUndelegationHoldCount(sdk.Context, []byte) error
	DecrementUndelegationHoldCount(sdk.Context, []byte) error
}
