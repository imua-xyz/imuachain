package types

const (
	// delegation of IMUA native asset, since UpdateStakerAssetState is not called for this case
	EventTypeImuaAssetDelegation = "imua_asset_delegation"
	AttributeKeyOperator         = "operator"
	AttributeKeyAmount           = "amount"

	// delegation state
	EventTypeDelegationStateUpdated         = "delegation_state_updated"
	AttributeKeyStakerID                    = "staker_id"
	AttributeKeyAssetID                     = "asset_id"
	AttributeKeyOperatorAddr                = "operator"
	AttributeKeyUndelegatableShareDelta     = "undelegatable_share_delta"
	AttributeKeyWaitUndelegationAmountDelta = "wait_undelegation_amount_delta"

	// operator + asset -> staker
	EventTypeStakerAppended    = "staker_appended"
	EventTypeStakerRemoved     = "staker_removed"
	EventTypeAllStakersRemoved = "all_stakers_removed"

	// staker operator association
	EventTypeOperatorAssociated    = "operator_associated"
	EventTypeOperatorDisassociated = "operator_disassociated"

	// undelegation
	EventTypeUndelegationStarted     = "undelegation_started"
	AttributeKeyRecordID             = "record_id"
	AttributeKeyCompletedEpochID     = "completed_epoch_id"
	AttributeKeyCompletedEpochNumber = "completed_epoch_number"
	AttributeKeyUndelegationID       = "undelegation_id"
	AttributeKeyTxHash               = "tx_hash"
	AttributeKeyBlockNumber          = "block_number"
	AttributeKeyInstantUnbonding     = "instant_unbonding"
	AttributeKeyApplyInstantSlash    = "apply_instant_slash"

	// undelegation matured
	EventTypeUndelegationMatured = "undelegation_matured"

	// undelegation held back or released
	EventTypeUndelegationHoldCountChanged = "undelegation_hold_count_changed"
	AttributeKeyHoldCount                 = "hold_count"
)
