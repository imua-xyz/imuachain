package types

const (
	TwoPhasesPrefix = "TwoPhases/"
	FeederPrefix    = TwoPhasesPrefix + "feeder/"
	ValidatorPrefix = TwoPhasesPrefix + "validator/"
)

func TwoPhasesKeyPrefix() []byte {
	return []byte(TwoPhasesPrefix)
}

func TwoPhasesFeederKeyPrefix() []byte {
	return []byte(FeederPrefix)
}

func TwoPhasesValidatorKeyPrefix() []byte {
	return []byte(ValidatorPrefix)
}

func TwoPhasesFeederKey(feederID uint64) []byte {
	var key []byte
	key = append(key, FeederPrefix...)
	key = append(key, Uint64Bytes(feederID)...)
	return key
}

func TwoPhasesValidatorPieceKey(validator string, feederID uint64) []byte {
	var key []byte
	key = append(key, ValidatorPrefix...)
	key = append(key, []byte(validator)...)
	key = append(key, DelimiterForCombinedKey)
	key = append(key, Uint64Bytes(feederID)...)
	return key
}
