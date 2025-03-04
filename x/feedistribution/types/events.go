package types

// x/feedistribution events
const (
	EventTypeCommission         = "commission"
	EventTypeSetWithdrawAddress = "set_withdraw_address"
	EventTypeRewards            = "rewards"
	EventTypeWithdrawRewards    = "withdraw_rewards"
	EventTypeWithdrawCommission = "withdraw_commission"
	EventTypeProposerReward     = "proposer_reward"

	AttributeKeyWithdrawAddress = "withdraw_address"
	AttributeKeyValidator       = "validator"
	AttributeKeyDelegator       = "delegator"

	// EventTypeUpdatedAVSRewardAsset : avs reward asset state updated
	EventTypeUpdatedAVSRewardAsset    = "avs_reward_asset_updated"
	AttributeKeyAvsAddress            = "avs_address"
	AttributeKeyAssetID               = "asset_id"
	AttributeKeyRewardPoolBalance     = "reward_pool_balance"
	AttributeKeyRewardPoolTotal       = "reward_pool_total"
	AttributeKeyRewardAllocationTotal = "reward_allocation_total"

	// EventTypeNewAVSRewardAsset : new avs reward asset
	EventTypeNewAVSRewardAsset = "avs_reward_asset_added"

	// EventTypeUpdatedRewardAsset : reward asset update
	EventTypeUpdatedRewardAsset = "avs_reward_asset_updated"

	// EventTypeAVSRewardDistributionSet : set the avs reward distribution
	EventTypeAVSRewardDistributionSet = "avs_reward_distribution_set"
	AttributeKeyEpochRewards          = "epoch_rewards"
	AttributeKeyOperatorProportions   = "operator_reward_proportions"
)
