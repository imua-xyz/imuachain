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
	prefixFeePool
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

	//
	FeePoolKey                          = []byte{prefixFeePool}
	OperatorOutstandingRewardsPrefix    = []byte{prefixOperatorOutstandingRewards}    // key for outstanding rewards
	DelegatorWithdrawAddrPrefix         = []byte{prefixDelegatorWithdrawAddr}         // key for delegator withdraw address
	DelegatorStartingInfoPrefix         = []byte{prefixDelegatorStartingInfo}         // key for delegator starting info
	OperatorHistoricalRewardsPrefix     = []byte{prefixOperatorHistoricalRewards}     // key for historical operators rewards / stake
	OperatorCurrentRewardsPrefix        = []byte{prefixOperatorCurrentRewards}        // key for current operator rewards
	OperatorAccumulatedCommissionPrefix = []byte{prefixOperatorAccumulatedCommission} // key for accumulated operator commission
	OperatorSlashEventPrefix            = []byte{prefixOperatorSlashEvent}            // key for operator slash fraction
	StakerOutstandingRewardsPrefix      = []byte{prefixStakerOutstandingRewards}      // key for outstanding rewards of staker
)

// GetOperatorAccumulatedCommissionKey creates the key for a validator's current commission.
func GetOperatorAccumulatedCommissionKey(v sdk.ValAddress) []byte {
	return append(OperatorAccumulatedCommissionPrefix, address.MustLengthPrefix(v.Bytes())...)
}

// GetOperatorCurrentRewardsKey creates the key for a validator's current rewards.
func GetOperatorCurrentRewardsKey(v sdk.ValAddress) []byte {
	return append(OperatorCurrentRewardsPrefix, address.MustLengthPrefix(v.Bytes())...)
}

// GetOperatorOutstandingRewardsKey creates the outstanding rewards key for a validator.
func GetOperatorOutstandingRewardsKey(valAddr sdk.ValAddress) []byte {
	return append(OperatorOutstandingRewardsPrefix, address.MustLengthPrefix(valAddr.Bytes())...)
}

// GetStakerOutstandingRewardsKey creates the outstanding rewards key for the staker.
func GetStakerOutstandingRewardsKey(staker string) []byte {
	return append(StakerOutstandingRewardsPrefix, address.MustLengthPrefix([]byte(staker))...)
}
