package types

// x/assets events
const (
	// staker asset state updated
	EventTypeUpdatedStakerAsset           = "staker_asset_updated"
	AttributeKeyStakerID                  = "staker_id"
	AttributeKeyAssetID                   = "asset_id"
	AttributeKeyDepositAmount             = "deposit_amount"
	AttributeKeyWithdrawableAmount        = "withdrawable_amount"
	AttributeKeyPendingUndelegationAmount = "pending_undelegation_amount"

	// client chain addition or update
	EventTypeNewClientChain        = "client_chain_added"
	EventTypeUpdatedClientChain    = "client_chain_updated"
	AttributeKeyName               = "name"
	AttributeKeyMetaInfo           = "meta_info"
	AttributeKeyChainID            = "chain_id"
	AttributeKeyExocoreChainIdx    = "exocore_chain_index"
	AttributeKeyFinalizationBlocks = "finalization_blocks"
	AttributeKeyLZID               = "layer_zero_chain_id"
	AttributeKeySigType            = "signature_type"
	AttributeKeyAddrLength         = "address_length"

	// token addition
	EventTypeNewToken    = "token_added"
	AttributeKeySymbol   = "symbol"
	AttributeKeyAddress  = "address"
	AttributeKeyDecimals = "decimals"

	// token update
	EventTypeUpdatedToken = "token_updated"

	// operator asset state updated
	EventTypeUpdatedOperatorAsset = "operator_asset_updated"
	AttributeKeyOperatorAddress   = "operator_address"
	AttributeKeyTotalAmount       = "total_amount"
	AttributeKeyTotalShare        = "total_share"
	AttributeKeyOperatorShare     = "operator_share"

	// token deposit amount updated; useful for tracking total deposited of an asset.
	// note that this amount includes lifetime slashed quantity of that token.
	EventTypeUpdatedStakingTotalAmount = "staking_total_amount_updated"
)
