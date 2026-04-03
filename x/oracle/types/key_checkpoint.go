package types

const (
	// CheckpointPrefix is the top-level prefix for all checkpoint state.
	CheckpointPrefix = "Checkpoint/"
	// CheckpointNoncePrefix stores the next checkpoint nonce per dstChainID.
	CheckpointNoncePrefix = CheckpointPrefix + "nonce/"
	// CheckpointDataPrefix stores checkpoint data by (dstChainID, nonce).
	CheckpointDataPrefix = CheckpointPrefix + "data/"
	// CheckpointSigPrefix stores validator signatures by (dstChainID, nonce, validatorAddr).
	CheckpointSigPrefix = CheckpointPrefix + "sig/"
	// CheckpointSignedPowerPrefix stores the accumulated signed power per (dstChainID, nonce).
	CheckpointSignedPowerPrefix = CheckpointPrefix + "power/"

	// ValsetCheckpointPrefix is the top-level prefix for validator set checkpoints.
	ValsetCheckpointPrefix = "ValsetCheckpoint/"
	// ValsetCheckpointNonceKey stores the next valset checkpoint nonce.
	ValsetCheckpointNonceKey = ValsetCheckpointPrefix + "nonce"
	// ValsetCheckpointDataPrefix stores valset checkpoint data by nonce.
	ValsetCheckpointDataPrefix = ValsetCheckpointPrefix + "data/"
	// ValsetCheckpointSigPrefix stores signatures for valset checkpoints.
	ValsetCheckpointSigPrefix = ValsetCheckpointPrefix + "sig/"
)

func CheckpointNonceKey(dstChainID uint64) []byte {
	return append([]byte(CheckpointNoncePrefix), Uint64Bytes(dstChainID)...)
}

func CheckpointDataKey(dstChainID, nonce uint64) []byte {
	key := make([]byte, 0, len(CheckpointDataPrefix)+16)
	key = append(key, []byte(CheckpointDataPrefix)...)
	key = append(key, Uint64Bytes(dstChainID)...)
	key = append(key, Uint64Bytes(nonce)...)
	return key
}

func CheckpointSigKey(dstChainID, nonce uint64, validator []byte) []byte {
	key := make([]byte, 0, len(CheckpointSigPrefix)+16+len(validator))
	key = append(key, []byte(CheckpointSigPrefix)...)
	key = append(key, Uint64Bytes(dstChainID)...)
	key = append(key, Uint64Bytes(nonce)...)
	key = append(key, validator...)
	return key
}

func CheckpointSignedPowerKey(dstChainID, nonce uint64) []byte {
	key := make([]byte, 0, len(CheckpointSignedPowerPrefix)+16)
	key = append(key, []byte(CheckpointSignedPowerPrefix)...)
	key = append(key, Uint64Bytes(dstChainID)...)
	key = append(key, Uint64Bytes(nonce)...)
	return key
}

func ValsetCheckpointDataKey(nonce uint64) []byte {
	return append([]byte(ValsetCheckpointDataPrefix), Uint64Bytes(nonce)...)
}

func ValsetCheckpointSigKey(nonce uint64, validator []byte) []byte {
	key := make([]byte, 0, len(ValsetCheckpointSigPrefix)+8+len(validator))
	key = append(key, []byte(ValsetCheckpointSigPrefix)...)
	key = append(key, Uint64Bytes(nonce)...)
	key = append(key, validator...)
	return key
}
