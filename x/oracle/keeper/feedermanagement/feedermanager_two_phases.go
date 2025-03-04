package feedermanagement

import (
	"encoding/base64"
	"sort"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	oracletypes "github.com/imua-xyz/imuachain/x/oracle/types"
)

func (f *FeederManager) NextPieceIndexByFeederID(feederID uint64) (uint32, bool) {
	r, ok := f.rounds[int64(feederID)]
	if !ok || r.m == nil {
		return 0, false
	}
	return r.m.NextPieceIndex()
}

func (f *FeederManager) MaxPieceIndexForTokenFeederID(feederID uint64) (uint32, bool) {
	r, ok := f.rounds[int64(feederID)]
	if !ok {
		return 0, false
	}
	if r.m == nil || r.m.LeafCount() < 1 {
		return 0, false
	}
	return r.m.LeafCount() - 1, true
}

// VerifyPieceProofsForTokenFeederID verifies targetPiece against feederID's corresponding merkle root, it might need proof nodes from pieces
// return (rootHash, verified)
func (f *FeederManager) VerifyPieceProofsForTokenFeederID(feederID uint64, targetPiece *oracletypes.PieceWithProof) ([]byte, bool) {
	r, ok := f.rounds[int64(feederID)]
	if !ok {
		return r.m.RootHash(), false
	}
	if r.m != nil || r.m.Completed() {
		return r.m.RootHash(), false
	}
	// the proof has been verified to be minimal in anteHandler so we dont need to check the first returned value
	_, verified := r.m.VerifyAndCacheOrdered(targetPiece.Index, targetPiece.RawData, targetPiece.Proof)
	return r.m.RootHash(), verified
}

// GetPieceWithProof verify the message is a valid rawData message and parse the piece of rawData
func (f *FeederManager) GetPieceWithProof(msg *oracletypes.MsgCreatePrice) (*oracletypes.PieceWithProof, bool) {
	if !msg.IsPhaseTwo() {
		return nil, false
	}
	if !f.cs.IsRule2PhasesByFeederID(msg.FeederID) {
		return nil, false
	}

	r := f.rounds[int64(msg.FeederID)]
	if r == nil {
		return nil, false
	}
	pieceCount := r.PieceCount()

	if len(msg.Prices) != 1 || len(msg.Prices[0].Prices) < 1 || len(msg.Prices[0].Prices) > 2 {
		return nil, false
	}

	pieceStr := msg.Prices[0].Prices[0].Price
	if len(pieceStr) == 0 || len(pieceStr) > int(f.cs.RawDataPieceSize()) {
		return nil, false
	}

	tmp, err := strconv.ParseUint(msg.Prices[0].Prices[0].DetID, 10, 32)
	pieceIndex := uint32(tmp)
	if err != nil || pieceIndex >= pieceCount { // || pieceIndex < 1 {
		return nil, false
	}

	ret := &oracletypes.PieceWithProof{
		Index:   pieceIndex,
		RawData: []byte(pieceStr),
	}

	if len(msg.Prices[0].Prices) == 2 {
		joinedHashesBase64 := msg.Prices[0].Prices[1].Price
		joinedIndexes := msg.Prices[0].Prices[1].DetID
		if len(joinedHashesBase64) == 0 || len(joinedIndexes) == 0 {
			return nil, false
		}

		hashesBase64 := strings.Split(joinedHashesBase64, oracletypes.DelimiterForBase64)
		indexes := strings.Split(joinedIndexes, oracletypes.DelimiterForBase64)

		if len(hashesBase64) == 0 || len(hashesBase64) != len(indexes) {
			return nil, false
		}

		proof := make([]*oracletypes.HashNode, 0, len(hashesBase64))
		rootIndex := r.m.RootIndex()
		for i, hashBase64 := range hashesBase64 {
			hashBytes, err := base64.StdEncoding.DecodeString(hashBase64)
			if err != nil || len(hashBytes) != common.HashLength {
				return nil, false
			}
			tmp, err := strconv.ParseUint(indexes[i], 10, 32)
			index := uint32(tmp)
			if err != nil || index > rootIndex {
				return nil, false
			}
			proof = append(proof, &oracletypes.HashNode{Index: index, Hash: hashBytes})
		}
		ret.Proof = proof
	}
	return ret, true
}

// GetTidyProofPathByIndex return the proof path with unseen nodes that under condition pieces comes as index from 0 to n, and cached all seen proof nodes
func (f *FeederManager) MinimalProofPathByIndex(feederID uint64, index uint32) []uint32 {
	r, ok := f.rounds[int64(feederID)]
	if !ok {
		return nil
	}
	if r.m == nil {
		return nil
	}
	return r.m.MinimalProofPathByIndex(index)
}

// FeederIDsCollectingRawData returns the list of feederIDs that are currently collecting raw data
// the list is sorted in ascending order
func (f *FeederManager) FeederIDsCollectingRawData() []uint64 {
	// TODO(leonz): implement me
	ret := make([]uint64, 0)
	for feederID, r := range f.rounds {
		if r.m != nil && !r.m.Completed() {
			ret = append(ret, uint64(feederID))
		}
	}
	sort.Slice(ret, func(i, j int) bool {
		return ret[i] < ret[j]
	})
	return ret
}
