package types

import (
	"encoding/binary"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// OutboundCheckpoint represents a batch of outbound messages awaiting validator signatures.
type OutboundCheckpoint struct {
	Nonce        uint64      `json:"nonce"`
	DstChainID   uint64      `json:"dst_chain_id"`
	MessagesHash common.Hash `json:"messages_hash"` // keccak256 of abi.encode(messages)
	SeqStart     uint64      `json:"seq_start"`     // first outbound seq in this batch
	SeqEnd       uint64      `json:"seq_end"`       // last outbound seq (inclusive)
	Height       int64       `json:"height"`        // block height when created
	Finalized    bool        `json:"finalized"`     // true when 2/3+ power has signed
}

// CheckpointSignature stores a single validator's ECDSA signature on a checkpoint.
type CheckpointSignature struct {
	Validator common.Address `json:"validator"` // validator EVM address
	V         uint8          `json:"v"`
	R         [32]byte       `json:"r"`
	S         [32]byte       `json:"s"`
	Power     int64          `json:"power"` // validator power at signing time
}

// BridgeID is a unique identifier for this bridge instance.
// Used in checkpoint hash computation to prevent cross-bridge replay.
const BridgeID = uint64(1)

// ComputeCheckpointHash computes the keccak256 hash that validators sign.
// Format: keccak256(abi.encode(bridgeID, nonce, dstChainID, messagesHash))
// This must match the hash computation in BridgeVerifier.sol.
func ComputeCheckpointHash(nonce, dstChainID uint64, messagesHash common.Hash) common.Hash {
	// Encode as 32-byte big-endian words (Solidity abi.encode format)
	data := make([]byte, 0, 128) // 4 * 32 bytes
	data = append(data, common.BigToHash(uint64ToBig(BridgeID)).Bytes()...)
	data = append(data, common.BigToHash(uint64ToBig(nonce)).Bytes()...)
	data = append(data, common.BigToHash(uint64ToBig(dstChainID)).Bytes()...)
	data = append(data, messagesHash.Bytes()...)
	return crypto.Keccak256Hash(data)
}

// ComputeMessagesHash computes keccak256 of the concatenated outbound message payloads.
func ComputeMessagesHash(payloads [][]byte) common.Hash {
	// We hash the length-prefixed concatenation for unambiguous encoding.
	var data []byte
	for _, p := range payloads {
		lenBuf := make([]byte, 8)
		binary.BigEndian.PutUint64(lenBuf, uint64(len(p)))
		data = append(data, lenBuf...)
		data = append(data, p...)
	}
	return crypto.Keccak256Hash(data)
}

// ComputeEthSignedMessageHash wraps the hash in the Ethereum signed message format.
// This matches how MetaMask/ethers.js sign messages: keccak256("\x19Ethereum Signed Message:\n32" + hash)
func ComputeEthSignedMessageHash(hash common.Hash) common.Hash {
	prefix := []byte("\x19Ethereum Signed Message:\n32")
	return crypto.Keccak256Hash(append(prefix, hash.Bytes()...))
}

// ValidatorSetCheckpoint represents a validator set update that must be relayed to the client chain.
type ValidatorSetCheckpoint struct {
	Nonce      uint64           `json:"nonce"`
	Validators []common.Address `json:"validators"`
	Powers     []int64          `json:"powers"`
	TotalPower int64            `json:"total_power"`
	Height     int64            `json:"height"`
	Finalized  bool             `json:"finalized"`
}

// ComputeValsetCheckpointHash computes the hash that validators sign for a validator set update.
func ComputeValsetCheckpointHash(nonce uint64, validators []common.Address, powers []int64) common.Hash {
	data := make([]byte, 0, 64+len(validators)*52)
	data = append(data, common.BigToHash(uint64ToBig(BridgeID)).Bytes()...)
	data = append(data, common.BigToHash(uint64ToBig(nonce)).Bytes()...)
	for i, v := range validators {
		// Left-pad address to 32 bytes
		var addrWord [32]byte
		copy(addrWord[12:], v.Bytes())
		data = append(data, addrWord[:]...)
		data = append(data, common.BigToHash(uint64ToBig(uint64(powers[i]))).Bytes()...)
	}
	return crypto.Keccak256Hash(data)
}

func uint64ToBig(v uint64) *big.Int {
	return new(big.Int).SetUint64(v)
}
