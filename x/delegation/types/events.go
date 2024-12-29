package types

const (
	// delegation of exo native asset, since UpdateStakerAssetState us not called for this case
	EventTypeNativeDelegation = "native_delegation"
	AttributeKeyStaker        = "staker"
	AttributeKeyOperator      = "operator"
	AttributeKeyAmount        = "amount"
	// undelegation started for exo native asset
	EventTypeNativeUndelegationStarted = "native_undelegation_started"
	// undelegation completed for exo native asset
	EventTypeNativeUndelegationCompleted = "native_undelegation_completed"

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
)
