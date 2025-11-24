package types

import (
	"fmt"
	"strconv"
	"strings"
)

// x/feedistribution events
const (
	EventTypeAllocateRewardsToOperator = "allocate_rewards_to_operator"
	AttributeKeyOperatorTotalReward    = "operator_total_reward"
	AttributeKeyOperatorCommission     = "operator_commission"
	// EventTypeWithdrawRewards : withdraw the reward for a staker
	EventTypeWithdrawRewards              = "withdraw_rewards"
	EventTypeWithdrawRewardFromAVS        = "withdraw_reward_from_avs"
	AttributeKeyWithdrawDecCoinsFromAVS   = "withdraw_dec_coins_from_avs"
	AttributeKeyStakerOutstandingRewards  = "staker_outstanding_rewards"
	AttributeKeyStakerWithdrawableRewards = "staker_withdrawable_rewards"

	// EventTypeWithdrawCommission :  withdraw the commission for an operator
	EventTypeWithdrawCommission           = "withdraw_commission"
	EventTypeWithdrawCommissionFromAVS    = "withdraw_commission_from_avs"
	AttributeKeyTotalWithdrawAmount       = "total_withdraw_amount"
	AttributeKeyWithdrawAmountFromDogfood = "withdraw_amount_from_dogfood"
	AttributeKeyStakerID                  = "staker_id"
	AttributeKeyOperator                  = "operator"

	// EventTypeUpdatedAVSRewardAsset : avs reward asset state updated
	EventTypeUpdatedAVSRewardAsset    = "avs_reward_asset_updated"
	AttributeKeyAvsAddress            = "avs_address"
	AttributeKeyAssetID               = "asset_id"
	AttributeKeyRewardPoolBalance     = "reward_pool_balance"
	AttributeKeyRewardPoolTotal       = "reward_pool_total"
	AttributeKeyRewardAllocationTotal = "reward_allocation_total"

	// EventTypeNewAVSRewardAsset : new avs reward asset
	EventTypeNewAVSRewardAsset       = "avs_reward_asset_added"
	AttributeKeyDenomination         = "reward_denomination"
	AttributeKeyDenominationExponent = "reward_denomination_exponent"

	// EventTypeUpdatedRewardAssetMetaInfo : reward asset meta info update
	EventTypeUpdatedRewardAssetMetaInfo = "reward_asset_meta_info_updated"

	// EventTypeAVSRewardDistributionSet : set the avs reward distribution
	EventTypeAVSRewardDistributionSet = "avs_reward_distribution_set"
	EventTypeAVSEpochRewardSet        = "avs_epoch_reward_set"
	EventTypeAVSRewardProportionsSet  = "avs_reward_proportions_set"
	AttributeKeyEpochRewards          = "epoch_rewards"
	AttributeKeyOperatorProportions   = "operator_reward_proportions"
	AttributeKeyEpochIdentifier       = "epoch_identifier"
	AttributeKeyEpochNumber           = "epoch_number"

	// EventTypeAVSRewardParamSet : set the avs reward parameter
	EventTypeAVSRewardParamSet        = "avs_reward_param_set"
	AttributeKeyAVSRewardParam        = "avs_reward_param"
	AttributeKeyCustomRewardInflation = "CustomRewardInflation"
	AttributeKeyCustomOperatorRatio   = "CustomOperatorRatio"
)

func (p *AVSRewardParam) ToEventString() string {
	return fmt.Sprintf("%s:%t,%s:%t",
		AttributeKeyCustomRewardInflation, p.CustomRewardInflation,
		AttributeKeyCustomOperatorRatio, p.CustomOperatorRatio,
	)
}

func ParseAVSRewardParams(event string) (*AVSRewardParam, error) {
	ret := &AVSRewardParam{}
	pairs := strings.Split(event, ",")

	for _, pair := range pairs {
		kv := strings.Split(pair, ":")
		if len(kv) != 2 {
			return nil, fmt.Errorf("invalid AVS reward param pair format: %s", pair)
		}

		key := kv[0]
		valStr := kv[1]

		val, err := strconv.ParseBool(valStr)
		if err != nil {
			return nil, fmt.Errorf("invalid bool value in AVS reward param: %s", valStr)
		}

		switch key {
		case AttributeKeyCustomRewardInflation:
			ret.CustomRewardInflation = val
		case AttributeKeyCustomOperatorRatio:
			ret.CustomOperatorRatio = val
		default:
			return nil, fmt.Errorf("unknown AVS reward param key: %s", key)
		}
	}

	return ret, nil
}
