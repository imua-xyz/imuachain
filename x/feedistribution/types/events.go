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
	AttributeKeyOperator        = "operator"
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
	EventTypeAVSEpochRewardSet        = "avs_epoch_reward_set"
	EventTypeAVSRewardProportionsSet  = "avs_reward_proportions_set"
	AttributeKeyEpochRewards          = "epoch_rewards"
	AttributeKeyOperatorProportions   = "operator_reward_proportions"

	// EventTypeAVSRewardParamSet : set the avs reward parameter
	EventTypeAVSRewardParamSet = "avs_reward_param_set"
	AttributeKeyAVSRewardParam = "avs_reward_param"

	// EventTypeStakeChangedDelegationsSet : set the delegations with changed stake
	EventTypeStakeChangedDelegationsSet = "stake_change_delegations_set"
	AttributeKeyStakers                 = "stakers"
	AttributeKeyPreDelegatedTotalAmount = "pre_delegated_total_amount"

	// EventTypeStakeChangedDelegationsDelete : delete the delegations with changed stake by epoch
	EventTypeStakeChangedDelegationsDelete = "stake_change_delegations_delete"
	AttributeKeyEpochIdentifier            = "epoch_identifier"
)
