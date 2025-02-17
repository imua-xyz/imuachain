package types

// x/avs events
const (
	// AVS creation. TODO: capture more information in the event.
	EventTypeAvsCreated    = "avs_created"
	AttributeKeyAvsAddress = "avs_address"

	// avs with chain-id
	EventTypeChainAvsCreated = "chain_avs_created"
	AttributeKeyChainID      = "chain_id"
)
