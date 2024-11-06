package types

const (
	EventTypeCreatePrice    = "create_price"
	EventTypeOracleLiveness = "oracle_liveness"
	EventTypeOracleSlash    = "oracle_slash"

	AttributeKeyFeederID          = "feeder_id"
	AttributeKeyTokenID           = "token_id"
	AttributeKeyBasedBlock        = "based_block"
	AttributeKeyRoundID           = "round_id"
	AttributeKeyProposer          = "proposer"
	AttributeKeyFinalPrice        = "final_price"
	AttributeKeyPriceUpdated      = "price_update"
	AttributeKeyParamsUpdated     = "params_update"
	AttributeKeyFeederIDs         = "feeder_ids"
	AttributeKeyNativeTokenUpdate = "native_token_update"
	AttributeKeyNativeTokenChange = "native_token_change"
	AttributeKeyValidatorKey      = "validator_key"
	AttributeKeyMissedRounds      = "missed_rounds"
	AttributeKeyHeight            = "height"
	AttributeKeyPower             = "power"
	AttributeKeyReason            = "reason"
	AttributeKeyJailed            = "jailed"
	AttributeKeyBurnedCoins       = "burned_coins"

	AttributeValuePriceUpdatedSuccess  = "success"
	AttributeValueParamsUpdatedSuccess = "success"
	AttributeValueNativeTokenUpdate    = "update"
	AttributeValueNativeTokenDeposit   = "deposit"
	AttributeValueNativeTokenWithdraw  = "withdraw"
	AttributeValueMissingReportPrice   = "missing_report_price"
	AttributeValueMaliciousReportPrice = "malicious_report_price"
)
