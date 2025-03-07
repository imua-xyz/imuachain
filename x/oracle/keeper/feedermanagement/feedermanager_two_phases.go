package feedermanagement

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	oracletypes "github.com/imua-xyz/imuachain/x/oracle/types"
)

func (f *FeederManager) NextPieceIndexByFeederID(feederID uint64) (uint32, bool) {
	// #nosec G115
	r, ok := f.rounds[int64(feederID)]
	if !ok || r.m == nil {
		return 0, false
	}
	return r.m.NextPieceIndex()
}

func (f *FeederManager) MaxPieceIndexForTokenFeederID(feederID uint64) (uint32, bool) {
	// #nosec G115
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
// if the merkle tree for the feederID had been completed, the rootHash will be returned, and verified will be false
func (f *FeederManager) VerifyPieceProofsForTokenFeederID(feederID uint64, targetPiece *oracletypes.PieceWithProof) ([]byte, bool) {
	// #nosec G115
	r, ok := f.rounds[int64(feederID)]
	if !ok || r.m == nil {
		return nil, false
	}
	if r.m.Completed() {
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

	// #nosec G115
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
	if err != nil {
		return nil, false
	}

	pieceIndex := uint32(tmp)
	if pieceIndex >= pieceCount {
		return nil, false
	}

	ret := &oracletypes.PieceWithProof{
		Index:     pieceIndex,
		RawData:   []byte(pieceStr),
		BaseBlock: msg.BasedBlock,
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
			if err != nil {
				return nil, false
			}

			index := uint32(tmp)
			if index > rootIndex {
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
	// #nosec G115
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
func (f *FeederManager) FeederIDsCollectingRawData() map[uint64]uint64 {
	return f.phaseTwoCollectingFeederIDs
}

func (f *FeederManager) phaseTwoCollected(feederID uint64) {
	delete(f.phaseTwoCollectingFeederIDs, feederID)
}

func (f *FeederManager) resetPhaseTwoCollectingFeederIDs() {
	f.phaseTwoCollectingFeederIDs = make(map[uint64]uint64)
	for _, feederID := range f.sortedFeederIDs {
		r := f.rounds[feederID]
		if r.m != nil && !r.m.Completed() {
			// #nosec G115
			f.phaseTwoCollectingFeederIDs[uint64(feederID)] = uint64(r.roundID)
		}
	}
}

func (f *FeederManager) resetPhaseTwoMaliciousTx() {
	f.phaseTwoMaliciousTx = make(map[uint64]string)
}

// ProcessRawData verify the submitted piece of rawData with proof against the expected root and cached the result if it passded the verification
// return (cached rawData piece, error)
func (f *FeederManager) ProcessRawData(ctx sdk.Context, msg *oracletypes.MsgCreatePrice, isCheckTx bool) ([]byte, error) {
	if isCheckTx {
		f = f.getCheckTx()
	}
	var r *round
	var err error
	if r, err = f.validateMsg(ctx, msg); err != nil {
		return nil, oracletypes.ErrInvalidMsg.Wrap(err.Error())
	}
	// we skip the verification in simulation mode, this is necessary for caching future pieces in mempool which hlep proposer to ensure including necessary piece to get avoid of missing count
	if isCheckTx {
		return nil, nil
	}
	// #nosec G115
	f.phaseTwoCollected(msg.FeederID)
	// this is ensured to get an non nil piece by anteHandler
	piece, _ := f.GetPieceWithProof(msg)
	// we don't check the 1st return value to see if this input proof is of the minimal, that's the duty of anteHandler, and 'verified' pieceWithProof will not fail the tx execution
	cachedProof, ok := r.m.VerifyAndCacheOrdered(piece.Index, piece.RawData, piece.Proof)
	if !ok {
		validator, _ := oracletypes.ConsAddrStrFromCreator(msg.Creator)
		f.phaseTwoMaliciousTx[msg.FeederID] = validator
		return nil, fmt.Errorf("failed to verify piece of index %d provided within message for feederID:%d against root:%s", piece.Index, msg.FeederID, hex.EncodeToString(r.m.RootHash()))
	}
	// we don't need to cache the proof for state updating if the merkle tree have collected all rawData
	if !r.m.Completed() {
		r.cachedProofForBlock = append(r.cachedProofForBlock, cachedProof...)
	}
	// we don't do no state update in tx exexuting, the postHandler and all state update will be handled in EndBlock
	//		// post handle rawData registered for the feederID
	//		// clear all caching pieces from stateDB
	//		// remove/reset merkleTree
	//		// remove merkleTree
	// persist piece for recovery (with memory-cache update into merkleTree)
	// save this piece and proof to db for recovery, for nodes without running,
	// this process only causes additional: two write to stateDB(piece, proof), one read from the stateDB(piece)

	return piece.RawData, nil
}
