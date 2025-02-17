package types

// x/dogfood events
const (
	// since the AVS is created at genesis, we manually emit this event.
	EventTypeDogfoodAvsCreated    = "dogfood_avs_created"
	AttributeKeyChainIDWithoutRev = "chain_id"
	AttributeKeyAvsAddress        = "avs_address"

	// emitted when the last total power is set, also, implying that
	// the validator set has changed.
	EventTypeLastTotalPowerUpdated = "last_total_power_updated"
	AttributeKeyLastTotalPower     = "last_total_power"

	// emitted when an operator opts out and will be finished at the end of the provided epoch.
	EventTypeOptOutBegan     = "opt_out_began"
	AttributeKeyEpoch        = "epoch"
	AttributeKeyOperator     = "operator"
	EventTypeOptOutsFinished = "opt_outs_finished"

	// emitted when a consensus address is added to the list of consensus addresses to prune
	// at the end of the epoch.
	EventTypeConsAddrPruningScheduled = "cons_addr_pruning_scheduled"
	AttributeKeyConsAddr              = "cons_addr"
	EventTypeConsAddrsPruned          = "cons_addrs_pruned"

	// emitted when an undelegation is added to the list of undelegations to mature
	// at the end of the epoch.
	EventTypeUndelegationMaturityScheduled = "undelegation_maturity_scheduled"
	AttributeKeyRecordID                   = "record_id"
	EventTypeUndelegationsMatured          = "undelegations_matured"
)
