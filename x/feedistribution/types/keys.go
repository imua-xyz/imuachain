package types

import (
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
	prefixParams byte = iota + 1
	prefixAVSRewardAssets
	prefixAVSRewardAssetByDenomination
	prefixAVSRewardParam
	prefixFeePools
	prefixAVSRewardDistribution
	prefixOperatorUnclaimedRewards
	prefixDelegatorWithdrawAddr
	prefixStakeChangeDelegations
	prefixDelegationStartingInfo
	prefixOperatorHistoricalRewards
	prefixOperatorCurrentRewards
	prefixOperatorCommission
	prefixOperatorSlashEvent
	prefixStakerClaimedRewards
	prefixStakerRewardParams
)

var (
	KeyPrefixParams = []byte{prefixParams}

	// KeyPrefixAVSRewardAssets :
	// avsAddr + '/' + assetID -> types.AVSRewardAsset
	// Key for the avs reward assets, it supports multiple reward assets for an AVS.
	// The reward assets should be registered by the AVS.
	KeyPrefixAVSRewardAssets = []byte{prefixAVSRewardAssets}

	// KeyPrefixAVSRewardAssetByDenomination :
	// avsAddr + '/' + denomination -> assetID
	// It's used to query the reward asset by its denomination.
	KeyPrefixAVSRewardAssetByDenomination = []byte{prefixAVSRewardAssetByDenomination}

	// KeyPrefixAVSRewardParam :
	// avsAddr -> types.AVSRewardParam
	// Key for the reward parameters of all AVSs, the avs can choose the distribution strategy by it.
	KeyPrefixAVSRewardParam = []byte{prefixAVSRewardParam}

	// KeyPrefixFeePools :
	// avsAddr -> types.FeePool
	// Key for the fee pools of all AVSs; it will track multiple community reward pools for different AVSs,
	// unlike the cosmos-sdk.
	KeyPrefixFeePools = []byte{prefixFeePools}

	// KeyPrefixAVSRewardDistribution :
	// avsAddr -> AVSRewardDistribution
	// key for avs rewards distribution, it will track the reward distribution information of multiple
	// AVSs in the current epoch.
	KeyPrefixAVSRewardDistribution = []byte{prefixAVSRewardDistribution}

	// KeyPrefixOperatorUnclaimedRewards :
	// operator + '/' + AVSAddr -> OperatorUnclaimedRewards
	// key for unclaimed rewards, it will track multiple unclaimed rewards from different AVSs
	KeyPrefixOperatorUnclaimedRewards = []byte{prefixOperatorUnclaimedRewards}

	// KeyPrefixStakeChangeDelegations :
	// epochIdentifier + '/' + operatorAddr + '/' + assetID -> DelegationChangeInfo
	// In imua chain, the F1 distribution is integrated per epoch, so we need to track the
	// delegations whose stake has changed each epoch. Then, we trigger the distribution lazily at
	// the end of epoch and update their startingInfos for future distributions. The slices need to
	// be cleared after finishing the distribution at the end of epoch, then they can be used for
	// the next epoch.
	KeyPrefixStakeChangeDelegations = []byte{prefixStakeChangeDelegations}

	// KeyPrefixDelegationStartingInfo :
	// delegationKey = restakerID +'/'+assetID+'/'+operatorAddr
	// delegationKey + '/' + epochIdentifier -> DelegationStartingInfo
	// key for delegation starting info, it will be used to track the starting info for a delegation reward
	// period.
	// Due to different epoch configurations for different AVSs, the startingInfo for the same delegation
	// will vary across different epochs. Therefore, it is necessary to store the startingInfo for all AVS
	// epochs related to this delegation. Of course, since multiple AVSs in the system may share the same
	// epoch configuration, it is sufficient to store the startingInfo for each epoch separately, rather
	// than storing it individually for each AVS. All AVSs with the same epoch configuration can share the
	// startingInfo for that epoch.
	KeyPrefixDelegationStartingInfo = []byte{prefixDelegationStartingInfo}

	// KeyPrefixOperatorHistoricalRewards :
	// operator + '/' +assetID + '/'+ epochIdentifier + '/' + period -> OperatorHistoricalRewards
	// key for historical operators rewards / stake
	KeyPrefixOperatorHistoricalRewards = []byte{prefixOperatorHistoricalRewards}

	// KeyPrefixOperatorCurrentRewards :
	// operator + '/' +assetID + '/'+ epochIdentifier -> OperatorCurrentRewards
	// key for current operator rewards
	KeyPrefixOperatorCurrentRewards = []byte{prefixOperatorCurrentRewards}

	// KeyPrefixOperatorCommission :
	// operator + '/' + AVSAddr -> OperatorCommission
	// key for accumulated operator commission
	KeyPrefixOperatorCommission = []byte{prefixOperatorCommission}

	// KeyPrefixOperatorSlashEvent :
	// operator + '/' + assetID + '/' + epochIdentifier + '/' + epochNumber + '/' + blockHeight-> OperatorSlashEvent
	// key for operator slash fraction, the periods of different epochs will differ when a
	// slash event occurs, so the slash event should be recorded for all epochs.
	// todo: We defer implementing the deletion mechanism, as the expected number of slash events
	// is low and unlikely to lead to significant state accumulation.
	KeyPrefixOperatorSlashEvent = []byte{prefixOperatorSlashEvent}

	// KeyPrefixStakerClaimedRewards :
	// stakerID + '/' + AVSAddr -> StakerClaimedRewards
	// key for claimed rewards of staker, including the outstanding and withdrawn rewards.
	// Unlike the F1 distribution in Cosmos SDK, the reward vault for restakers in the Imua
	// protocol may be distributed across different client chains as well as the Imua chain
	// itself. Therefore, when a staker claims rewards, we handle it in two steps. The first
	// step is similar to the F1 withdraw process, which is passive and triggered when the stake
	// changes. In this step, the reward withdrawal is first recorded under this specific key.
	// Afterward, the staker can initiate a claim transaction on the chain where the reward vault
	// is located. The transaction will check the validity of the claim based on the recorded reward
	// status and execute the reward distribution accordingly.
	KeyPrefixStakerClaimedRewards = []byte{prefixStakerClaimedRewards}

	// KeyPrefixDelegatorWithdrawAddr
	// key for delegator withdraw address
	// todo: The reward claim process seems to be initiated on the Imua chain and routed through the
	// cross-chain protocol to execute on multiple reward vaults, which appears to be the most reasonable approach.
	// However, this process is also applicable to delegation. Since all of our current design process entry points
	// are initiated from the client chain, this method may be considered for optimization in the future, especially
	// when considering the transaction fees on the client chain.
	// Additionally, with this approach, stakers whose addresses are not compatible with EVM will need to set up an
	// EVM address on the Imua chain to execute the above operations. Of course, for such stakers, if they participate
	// and receive rewards on the Imua chain, such as rewards provided by dogfood, they will also need such an address
	// when claiming rewards. Therefore, we may later provide an interface for all address-incompatible stakers to
	// configure their EVM addresses on Imua. However, the transaction to configure this address should be initiated
	// on the client chain and authorized by the staker's native address.
	// This address can actually be shared with the `KeyPrefixReStakerImuaAddr` defined under the asset module.
	// However, if we need to separate the staker's operation address from the reward address, then they cannot be
	// shared. This part will be decided when we actually use it in the future, and the issue only needs to be
	// addressed when distributing Imua rewards to stakers with address incompatibility on the chain.
	KeyPrefixDelegatorWithdrawAddr = []byte{prefixDelegatorWithdrawAddr}

	// KeyPrefixStakerRewardParams stakerID -> StakerRewardParams
	KeyPrefixStakerRewardParams = []byte{prefixStakerRewardParams}
)
