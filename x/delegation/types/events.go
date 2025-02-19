package types

const (
	// delegation of IMUA native asset, since UpdateStakerAssetState is not called for this case
	EventTypeImuaAssetDelegation = "imua_asset_delegation"
	AttributeKeyOperator         = "operator"
	AttributeKeyAmount           = "amount"

	// delegation state
	EventTypeDelegationStateUpdated    = "delegation_state_updated"
	AttributeKeyStakerID               = "staker_id"
	AttributeKeyAssetID                = "asset_id"
	AttributeKeyOperatorAddr           = "operator"
	AttributeKeyUndelegatableShare     = "undelegatable_share"
	AttributeKeyWaitUndelegationAmount = "wait_undelegation_amount"

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

	// undelegation matured
	EventTypeUndelegationMatured          = "undelegation_matured"
	AttributeKeyWithdrawableAmount        = "withdrawable_amount"
	AttributeKeyPendingUndelegationAmount = "pending_undelegation_amount"

	// undelegation held back or released
	EventTypeUndelegationHoldCountChanged = "undelegation_hold_count_changed"
	AttributeKeyHoldCount                 = "hold_count"
)
