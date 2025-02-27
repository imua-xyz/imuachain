package types

const (
	TwoPhasesPrefix        = "TwoPhases/"
	FeederPrefix           = TwoPhasesPrefix + "feeder/"
	FeederValidatorsPrefix = TwoPhasesPrefix + "validators/"
	FeederRawDataPrefix    = TwoPhasesPrefix + "rawData/"
	FeederProofPrefix      = TwoPhasesPrefix + "proof/"
	FeederTreeInfoPrefix   = TwoPhasesPrefix + "treeInfo/"
)

func TwoPhasesKeyPrefix() []byte {
	return []byte(TwoPhasesPrefix)
}

func TwoPhasesFeederKeyPrefix() []byte {
	return []byte(FeederPrefix)
}

func TwoPhasesFeederValidatorsKeyPrefix() []byte {
	return []byte(FeederValidatorsPrefix)
}

func TwoPhasesFeederKey(feederID uint64) []byte {
	var key []byte
	key = append(key, FeederPrefix...)
	key = append(key, Uint64Bytes(feederID)...)
	return key
}

func TwoPhasesFeederValidatorsKey(feederID uint64) []byte {
	var key []byte
	key = append(key, FeederValidatorsPrefix...)
	key = append(key, Uint64Bytes(feederID)...)
	return key
}

func TwoPhasesFeederRawDataKeyPrefix(feederID uint64) []byte {
	var key []byte
	key = append(key, FeederRawDataPrefix...)
	key = append(key, DelimiterForCombinedKey)
	key = append(key, Uint64Bytes(feederID)...)
	key = append(key, DelimiterForCombinedKey)
	return key
}

func TwoPhasesFeederRawDataKey(feederID uint64, index uint32) []byte {
	var key []byte
	key = append(key, TwoPhasesFeederRawDataKeyPrefix(feederID)...)
	return append(key, Uint32Bytes(index)...)
}

func TwoPhasesFeederProofKeyPrefix() []byte {
	return []byte(FeederProofPrefix)
}

func TwoPhasesFeederProofKey(feederID uint64) []byte {
	var key []byte
	key = append(key, FeederProofPrefix...)
	key = append(key, Uint64Bytes(feederID)...)
	return key
}

func TwoPhaseFeederTreeInfoKeyPrefix() []byte {
	return []byte(FeederTreeInfoPrefix)
}

func TwoPhaseFeederTreeInfoKey(feederID uint64) []byte {
	var key []byte
	key = append(key, FeederTreeInfoPrefix...)
	key = append(key, Uint64Bytes(feederID)...)
	return key
}
