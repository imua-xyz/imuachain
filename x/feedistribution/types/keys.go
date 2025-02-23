package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/ethereum/go-ethereum/common"
)

const (
	// ModuleName defines the module name
	ModuleName = "feedistribution"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// RouterKey is the message route for distribution
	RouterKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey            = "mem_feedistribution"
	ProtocolPoolModuleName = "protocolpool"
)

// ModuleAddress is the native module address for EVM
var ModuleAddress common.Address

func init() {
	ModuleAddress = common.BytesToAddress(authtypes.NewModuleAddress(ModuleName).Bytes())
}

const (
	// EpochIdentifier defines the epoch identifier for fee distribution module
	prefixParams byte = iota + 1
	prefixEpochIdentifier
	prefixFeePools
	prefixOperatorOutstandingRewards
	prefixDelegatorWithdrawAddr
	prefixDelegatorStartingInfo
	prefixOperatorHistoricalRewards
	prefixOperatorCurrentRewards
	prefixOperatorAccumulatedCommission
	prefixOperatorSlashEvent
	prefixStakerOutstandingRewards
)

var (
	KeyPrefixParams          = []byte{prefixParams}
	KeyPrefixEpochIdentifier = []byte{prefixEpochIdentifier}

	// KeyPrefixFeePools avsAddr -> types.FeePool
	// Key for the fee pools of all AVSs; it will track multiple reward pools for different AVSs,
	// unlike the cosmos-sdk.
	KeyPrefixFeePools = []byte{prefixFeePools}
	// KeyPrefixOperatorOutstandingRewards operator + '/' + AVSAddr -> OperatorOutstandingRewards
	// key for outstanding rewards, it will track multiple outstanding rewards from different AVSs
	KeyPrefixOperatorOutstandingRewards = []byte{prefixOperatorOutstandingRewards}

	// KeyPrefixDelegatorWithdrawAddr
	// key for delegator withdraw address
	KeyPrefixDelegatorWithdrawAddr = []byte{prefixDelegatorWithdrawAddr}

	// KeyPrefixDelegatorStartingInfo
	// key for delegator starting info,
	KeyPrefixDelegatorStartingInfo         = []byte{prefixDelegatorStartingInfo}
	KeyPrefixOperatorHistoricalRewards     = []byte{prefixOperatorHistoricalRewards}     // key for historical operators rewards / stake
	KeyPrefixOperatorCurrentRewards        = []byte{prefixOperatorCurrentRewards}        // key for current operator rewards
	KeyPrefixOperatorAccumulatedCommission = []byte{prefixOperatorAccumulatedCommission} // key for accumulated operator commission
	KeyPrefixOperatorSlashEvent            = []byte{prefixOperatorSlashEvent}            // key for operator slash fraction
	KeyPrefixStakerOutstandingRewards      = []byte{prefixStakerOutstandingRewards}      // key for outstanding rewards of staker
)

// GetOperatorAccumulatedCommissionKey creates the key for a validator's current commission.
func GetOperatorAccumulatedCommissionKey(v sdk.ValAddress) []byte {
	return append(KeyPrefixOperatorAccumulatedCommission, address.MustLengthPrefix(v.Bytes())...)
}

// GetOperatorCurrentRewardsKey creates the key for a validator's current rewards.
func GetOperatorCurrentRewardsKey(v sdk.ValAddress) []byte {
	return append(KeyPrefixOperatorCurrentRewards, address.MustLengthPrefix(v.Bytes())...)
}

// GetOperatorOutstandingRewardsKey creates the outstanding rewards key for a validator.
func GetOperatorOutstandingRewardsKey(valAddr sdk.ValAddress) []byte {
	return append(KeyPrefixOperatorOutstandingRewards, address.MustLengthPrefix(valAddr.Bytes())...)
}

// GetStakerOutstandingRewardsKey creates the outstanding rewards key for the staker.
func GetStakerOutstandingRewardsKey(staker string) []byte {
	return append(KeyPrefixStakerOutstandingRewards, address.MustLengthPrefix([]byte(staker))...)
}
