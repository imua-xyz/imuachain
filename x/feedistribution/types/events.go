package types

// x/feedistribution events
const (
	EventTypeCommission         = "commission"
	EventTypeSetWithdrawAddress = "set_withdraw_address"
	EventTypeRewards            = "rewards"
	// EventTypeWithdrawRewards : withdraw the reward for a staker
	EventTypeWithdrawRewards = "withdraw_rewards"

	// EventTypeWithdrawCommission :  withdraw the commission for an operator
	EventTypeWithdrawCommission             = "withdraw_commission"
	AttributeKeyAllAVSActualWithdrawAmounts = "all_avs_actual_withdraw_amounts"
	AttributeKeyTotalWithdrawAmount         = "total_withdraw_amount"
	AttributeKeyWithdrawAmountFromDogfood   = "withdraw_amount_from_dogfood"
	AttributeKeyStakerID                    = "staker_id"

	// EventTypeWithdrawCommissionFromDogfood :  withdraw all commissions only from dogfood.
	EventTypeWithdrawCommissionFromDogfood = "withdraw_commission_from_dogfood"
	AttributeKeyCommissionAmount           = "commission_amount"

	EventTypeProposerReward = "proposer_reward"

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

	// EventTypeUpdatedRewardAssetMetaInfo : reward asset meta info update
	EventTypeUpdatedRewardAssetMetaInfo = "reward_asset_meta_info_updated"

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
	AttributeKeyEpochNumber                = "epoch_number"
)
