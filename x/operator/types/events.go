package types

const (
	EventTypeOptIn                  = "opt_in"
	AttributeKeyOperator            = "operator"
	AttributeKeyAVSAddr             = "avs_addr"
	EventTypeOptOut                 = "opt_out"
	EventTypeRegisterOperator       = "register_operator"
	EventTpesSetConsKey             = "set_cons_key"
	AttributeKeyChainID             = "chain_id"
	AttributeKeyConsensusAddress    = "consensus_address"
	EventTypeInitRemoveConsKey      = "init_remove_cons_key"
	EventTypeEndRemoveConsKey       = "end_remove_cons_key"
	EventTypeSetPrevConsKey         = "set_prev_cons_key"
	EventTypeUpdateOperatorUSDValue = "update_operator_usd_value"
	AttributeSelfUSDValue           = "self_usd_value"
	AttributeTotalUSDValue          = "total_usd_value"
	AttributeActiveUSDValue         = "active_usd_value"
	EventTypeDeleteOperatorUSDValue = "delete_operator_usd_value"
	EventTypeUpdateAVSUSDValue      = "update_avs_usd_value"
	EventTypeDeleteAVSUSDValue      = "delete_avs_usd_value"
)
