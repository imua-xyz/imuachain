package keeper

import (
	"fmt"
	"sort"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/imua-xyz/imuachain/x/oracle/types"
)

// SetNextPieceIndex sets the next-piece-index of feederID for 'node-recovery'
// func (k Keeper) SetNextPieceIndexForFeeder(ctx sdk.Context, feederID uint64, pieceIndex uint32) {
// 	store := ctx.KVStore(k.storeKey)
// 	key := types.TwoPhasesFeederKey(feederID)
// 	store.Set(key, types.Uint32Bytes(pieceIndex))
// }

func (k Keeper) Setup2ndPhase(ctx sdk.Context, feederID uint64, validators []string, leafCount uint32, rootHash []byte) {
	k.Setup2ndPhaseNextPieceIndex(ctx, feederID, validators)
	k.SetFeederTreeInfo(ctx, feederID, leafCount, rootHash)
}

// we group the validators by feederID instead of the opposite way because when we set up or clear,
// we do this for all validators under the feederID. When changes happen to a single validator, we enter "forceSeal,"
// which removes all validators under that feederID.
// Therefore, we use feederID→[]{validators,index}, not validator→[]{feederID, index} or feederID/validator→index.
// While the latter approach would make "checkAndIncrease" faster when querying the index for a specific
// validator under a specific feederID, it trades off many I/O operations with memory iteration, which isn't optimal.
func (k Keeper) Setup2ndPhaseNextPieceIndex(ctx sdk.Context, feederID uint64, validators []string) {
	store := ctx.KVStore(k.storeKey)
	key := types.TwoPhasesFeederValidatorsKey(feederID)
	validatorIndexList := make([]*types.ValidatorIndex, 0, len(validators))
	// set next piece index for all input validators to 0 under the input feederID
	if len(validators) > 0 {
		for _, validator := range validators {
			validatorIndexList = append(validatorIndexList, &types.ValidatorIndex{Validator: validator, NextIndex: 0})
		}
		bz := k.cdc.MustMarshal(&types.FeederValidatorsIndex{
			ValidatorIndexList: validatorIndexList,
		})
		store.Set(key, bz)
	}
	// set next piece index for feederID to 0
	key = types.TwoPhasesFeederKey(feederID)
	store.Set(key, types.Uint32Bytes(0))
}

func (k Keeper) Clear2ndPhaseNextPieceIndex(ctx sdk.Context, feederID uint64) {
	store := ctx.KVStore(k.storeKey)
	key := types.TwoPhasesFeederKey(feederID)
	// delete nextPieceIndex for feederID
	store.Delete(key)
	key = types.TwoPhasesFeederValidatorsKey(feederID)
	// delete nextPieceIndex for all validators with feederID
	store.Delete(key)
}

func (k Keeper) SetNextPieceIndexForFeeder(ctx sdk.Context, feederID uint64, nextPieceIndex uint32) {
	store := ctx.KVStore(k.storeKey)
	key := types.TwoPhasesFeederKey(feederID)
	store.Set(key, types.Uint32Bytes(nextPieceIndex))
}

// func (k Keeper) ClearNextPieceIndexForFeeder(ctx sdk.Context, feederID uint64) {
// 	store := ctx.KVStore(k.storeKey)
// 	key := types.TwoPhasesFeederKey(feederID)
// 	store.Delete(key)
// }

// TODO: remove this
// NextPieceIndexByFeederID read directly from memory and return the next-piece-index of input feederID
func (k Keeper) NextPieceIndexByFeederID(ctx sdk.Context, feederID uint64) (uint32, bool) {
	// query from mem-cache
	return k.FeederManager.NextPieceIndexByFeederID(feederID)
}

// CheckAndIncreasePieceIndex checks and increase the 'nextPieceIndex' of a specific validator and feederID
// valid pieceIndex starts from 0
// returns (nextPieceIndexAfterIncreased, error)
func (k Keeper) CheckAndIncreaseNextPieceIndex(ctx sdk.Context, validator string, feederID uint64, nextPieceIndex uint32) (uint32, error) {
	maxPieceIndex, ok := k.FeederManager.MaxPieceIndexForTokenFeederID(feederID)
	if !ok {
		return 0, fmt.Errorf("max piece index not found for feederID: %d", feederID)
	}
	if nextPieceIndex > maxPieceIndex {
		return 0, fmt.Errorf("piece_index_check_failed: feederID:%d, max_piece_index:%d, got:%d", feederID, maxPieceIndex, nextPieceIndex)
	}
	store := ctx.KVStore(k.storeKey)
	// key := types.TwoPhasesValidatorPieceKey(validator, feederID)
	key := types.TwoPhasesFeederValidatorsKey(feederID)
	bz := store.Get(key)
	if bz == nil {
		return 0, fmt.Errorf("piece_index_check_failed: validator_not_found: validator:%s, feeder_id:%d", validator, feederID)
	}
	feederValidatorsIndex := &types.FeederValidatorsIndex{}
	k.cdc.MustUnmarshal(bz, feederValidatorsIndex)
	for _, validatorIndex := range feederValidatorsIndex.ValidatorIndexList {
		if validatorIndex.Validator == validator {
			if validatorIndex.NextIndex == nextPieceIndex {
				validatorIndex.NextIndex++
				bz = k.cdc.MustMarshal(feederValidatorsIndex)
				store.Set(key, bz)
				return nextPieceIndex + 1, nil
			}
			return 0, fmt.Errorf("piece_index_check_failed: non_conseecutive: expected:%d, recived:%d", validatorIndex.NextIndex, nextPieceIndex)
		}
	}
	return 0, fmt.Errorf("piece_index_check_failed: next_piece_index not found for valdiator:%s", validator)
}

// used for recovery, otherwise mem-cache are used directly for reading
func (k Keeper) SetRawDataPiece(ctx sdk.Context, feederID uint64, pieceIndex uint32, rawData []byte) {
	store := ctx.KVStore(k.storeKey)
	key := types.TwoPhasesFeederRawDataKey(feederID, pieceIndex)
	store.Set(key, rawData)
}

func (k Keeper) GetRawDataPieces(ctx sdk.Context, feederID uint64) ([][]byte, error) {
	store := ctx.KVStore(k.storeKey)
	key := types.TwoPhasesFeederKey(feederID)
	bz := store.Get(key)
	if bz == nil {
		return nil, nil
	}
	nextPieceIndex := types.BytesToUint32(bz)
	if nextPieceIndex == 0 {
		return nil, nil
	}
	ret := make([][]byte, 0, nextPieceIndex)
	for i := uint32(0); i < nextPieceIndex; i++ {
		key = types.TwoPhasesFeederRawDataKey(feederID, i)
		bz = store.Get(key)
		if bz == nil {
			// this should not happen, we got something wrong in db
			return nil, fmt.Errorf("there's something wrong in db, miss piece:%d of rawData for feederID:%d", i, feederID)
		}
		ret = append(ret, bz)
	}
	return ret, nil
}

func (k Keeper) SetFeederTreeInfo(ctx sdk.Context, feederID uint64, count uint32, rootHash []byte) {
	if count == 0 || len(rootHash) != common.HashLength {
		return
	}
	store := ctx.KVStore(k.storeKey)
	key := types.TwoPhaseFeederTreeInfoKey(feederID)
	treeInfo := &types.TreeInfo{
		LeafCount: count,
		RootHash:  rootHash,
	}
	bz := k.cdc.MustMarshal(treeInfo)
	store.Set(key, bz)
}

func (k Keeper) GetFeederTreeInfo(ctx sdk.Context, feederID uint64) (uint32, []byte) {
	store := ctx.KVStore(k.storeKey)
	key := types.TwoPhaseFeederTreeInfoKey(feederID)
	bz := store.Get(key)
	if bz == nil {
		return 0, nil
	}
	treeInfo := &types.TreeInfo{}
	k.cdc.MustUnmarshal(bz, treeInfo)
	return treeInfo.LeafCount, treeInfo.RootHash
}

// used for recovery, otherwise, mem-cache are used directly for reading
func (k Keeper) AddNodesToMerkleTree(ctx sdk.Context, feederID uint64, proof []*types.HashNode) {
	if len(proof) == 0 {
		return
	}
	store := ctx.KVStore(k.storeKey)
	key := types.TwoPhasesFeederProofKey(feederID)
	bz := store.Get(key)
	merkle := &types.FlattenTree{}
	k.cdc.MustUnmarshal(bz, merkle)
	nodes := merkle.Nodes
	sort.Slice(proof, func(i, j int) bool { return proof[i].Index < proof[j].Index })
	uniqueOrderedProof := make([]*types.HashNode, 0, len(proof))
	for i := 0; i < len(proof); i++ {
		if i == 0 || proof[i].Index != proof[i-1].Index {
			uniqueOrderedProof = append(uniqueOrderedProof, proof[i])
		}
	}
	l1 := len(nodes)
	l2 := len(uniqueOrderedProof)
	i := 0
	j := 0
	newList := make([]*types.HashNode, 0, l1+l2)
	for i < l1 && j < l2 {
		switch {
		case nodes[i].Index == uniqueOrderedProof[j].Index:
			newList = append(newList, nodes[i])
			i++
			j++
		case nodes[i].Index < uniqueOrderedProof[j].Index:
			newList = append(newList, nodes[i])
			i++
		case nodes[i].Index > uniqueOrderedProof[j].Index:
			newList = append(newList, proof[j])
			j++
		}
	}
	if i < l1 {
		newList = append(newList, nodes[i:]...)
	} else {
		newList = append(newList, uniqueOrderedProof[j:]...)
	}
	merkle.Nodes = newList
	bz = k.cdc.MustMarshal(merkle)
	store.Set(key, bz)
}

func (k Keeper) GetNodesFromMerkleTree(ctx sdk.Context, feederID uint64) []*types.HashNode {
	store := ctx.KVStore(k.storeKey)
	key := types.TwoPhasesFeederProofKey(feederID)
	bz := store.Get(key)
	if bz == nil {
		return nil
	}
	mt := &types.FlattenTree{}
	k.cdc.MustUnmarshal(bz, mt)
	return mt.Nodes
}

// clear feederID/, clear:
// 1. rawData
// 2. proof
func (k Keeper) Clear2ndPhase(ctx sdk.Context, feederID uint64, rootIndex uint32) {
	store := ctx.KVStore(k.storeKey)
	// clear rawData
	for i := uint32(0); i <= rootIndex; i++ {
		store.Delete(types.TwoPhasesFeederRawDataKey(feederID, i))
	}
	// clear proof
	store.Delete(types.TwoPhasesFeederProofKey(feederID))
	// clear indexex
	k.Clear2ndPhaseNextPieceIndex(ctx, feederID)
}
