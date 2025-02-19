package types

// DONTCOVER

// x/operator events
const (
	EventTypeRegisterOperator        = "register_operator"
	AttributeKeyOperator             = "operator"
	AttributeKeyMetaInfo             = "meta_info"
	AttributeKeyMaxCommissionRate    = "max_commission_rate"
	AttributeKeyMaxChangeRate        = "max_change_rate"
	AttributeKeyCommissionUpdateTime = "commission_update_time"

	EventTypeOptIn            = "opt_in"
	AttributeKeyAVSAddr       = "avs_addr"
	AttributeKeySlashContract = "slash_contract"
	AttributeKeyOptInHeight   = "opt_in_height"
	AttributeKeyOptOutHeight  = "opt_out_height"
	AttributeKeyJailed        = "jailed"

	EventTypeOptInfoUpdated = "update_opt_info"

	EventTypeSetConsKey          = "set_cons_key"
	AttributeKeyChainID          = "chain_id"
	AttributeKeyConsensusAddress = "consensus_address"
	AttributeKeyConsKeyHex       = "cons_key_hex"

	EventTypeInitRemoveConsKey = "init_remove_cons_key"

	EventTypeEndRemoveConsKey = "end_remove_cons_key"

	EventTypeSetPrevConsKey    = "set_prev_cons_key"
	EventTypeRemovePrevConsKey = "remove_prev_cons_key"

	EventTypeUpdateOperatorUSDValue = "update_operator_usd_value"
	AttributeKeySelfUSDValue        = "self_usd_value"
	AttributeKeyTotalUSDValue       = "total_usd_value"
	AttributeKeyActiveUSDValue      = "active_usd_value"

	EventTypeDeleteOperatorUSDValue = "delete_operator_usd_value"

	EventTypeUpdateAVSUSDValue = "update_avs_usd_value"

	EventTypeDeleteAVSUSDValue = "delete_avs_usd_value"

	EventTypeUndelegationSlashed = "undelegation_slashed"
	AttributeKeyRecordID         = "record_id"
	AttributeKeyAmount           = "amount"
	AttributeKeySlashAmount      = "slash_amount"

	EventTypeOperatorAssetSlashed = "operator_asset_slashed"
	AttributeKeyAssetID           = "asset_id"
)
