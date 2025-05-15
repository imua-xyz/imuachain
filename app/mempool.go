package app

import (
	"context"
	"errors"
	"sort"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/mempool"

	"github.com/imua-xyz/imuachain/app/ante/utils"
	oraclekeeper "github.com/imua-xyz/imuachain/x/oracle/keeper"
	oracletypes "github.com/imua-xyz/imuachain/x/oracle/types"
)

// ImuaMempool is a mempool for oracle module, it caches rawData txs and provides a way to select txs for block proposal
// we only take care of rawData txs with rawData message which is used for 2-phases aggregation of oracle module
// rawData txs are txs with MsgCreatePrice message which inlude raw data piece and proof used to submit oracle data to chain
// rawData pieces is required to be submitted in order, and the piece index is verified in anteHandler
// in ImuaMempool, we pre cache rawData pieces for each feederID(the proposer would have more chance to include a valid rawData piece in block to get avoid of being punished by miss-count)
// Note: This implementation assumes single-threaded usage as it doesn't implement concurrency protection.
type ImuaMempool struct {
	// feederID -> pieceIndex->[]PieceWithProof, cached pieceWithProof for feederID
	cachedPieces map[uint64]map[uint32][]*oracletypes.PieceWithProof
	k            *oraclekeeper.Keeper
	count        int
	// load from config file as a config for mempool
	txDecoder sdk.TxDecoder
}

var _ mempool.Mempool = &ImuaMempool{}

// Insert inserts a tx into mempool. While it accepts all transaction types, it only processes and caches
// raw data transactions related to 2-phases aggregation of the oracle module. Non-oracle transactions
// are simply passed through without any processing.
func (em *ImuaMempool) Insert(_ context.Context, tx sdk.Tx) error {
	// we don't filter tx not with message of rawData type, those tx will just be added into tendermint's txpool
	if !em.includesMsgOracle(tx) {
		return nil
	}

	piece, msgOracle, _, isRawData := em.GetPieceWithProof(tx)
	// we don't filter tx not with message of rawData type
	if !isRawData {
		return nil
	}
	if piece == nil {
		return errors.New("failed to parse pieceWithProof from tx with oracle rawData message")
	}
	fID := msgOracle.FeederID

	piecesCached, ok := em.cachedPieces[fID]
	if !ok {
		em.cachedPieces[fID] = make(map[uint32][]*oracletypes.PieceWithProof)
		em.cachedPieces[fID][piece.Index] = []*oracletypes.PieceWithProof{piece}
		em.count++
		return nil
	}

	piecesIndexCached, ok := piecesCached[piece.Index]
	if !ok {
		piecesCached[piece.Index] = []*oracletypes.PieceWithProof{piece}
		em.count++
		return nil
	}
	for _, pieceCached := range piecesIndexCached {
		if pieceCached.EqualsTo(piece) {
			return errors.New("piece exists in mempool")
		}
	}
	piecesCached[piece.Index] = append(piecesIndexCached, piece)
	em.count++
	return nil
}

// Select selects txs for block proposal, we only select rawData txs with rawData message which is used for 2-phases aggregation of oracle module, for other txs we just return them
// we only keep one tx for each feederID, and we only keep the tx with the piece index expected by the feederID
func (em *ImuaMempool) Select(ctx context.Context, txList [][]byte) mempool.Iterator {
	// Remove all expired transactions. For example, when selecting for block 100,
	// all transactions belonging to block 99 or earlier should be removed since they're
	// no longer relevant for the current selection round.

	collectingFeederIDs := em.k.FeederManager.FeederIDsCollectingRawData()
	if len(collectingFeederIDs) == 0 {
		// remove all cached pieces since no collectingFeederIDs available
		em.reset()
		txDecodedList := make([]sdk.Tx, 0, len(txList))
		for _, txBytes := range txList {
			tx, err := em.txDecoder(txBytes)
			if err != nil {
				continue
			}
			if utils.IsOraclePhaseTwoTx(tx) {
				// remove rawData from request txList if there's no feederID collecting rawData pieces
				continue
			}
			txDecodedList = append(txDecodedList, tx)
		}
		if len(txDecodedList) > 0 {
			return IteratorFromSlice(txDecodedList)
		}
		return nil
	}

	// remove all expired txs from cache
	em.clearExpiredFeederIDcache(collectingFeederIDs)

	seenFeederIDs := make(map[uint64]struct{}, len(collectingFeederIDs))
	keep := make([]sdk.Tx, 0, len(txList))
	for _, txBytes := range txList {
		tx, err := em.txDecoder(txBytes)
		if err != nil {
			// skip undecoded bytes - they will be rechecked when re-broadcast
			continue
		}
		msgs, _, isRawData, _ := utils.IsValidOracleTx(tx)
		if !isRawData {
			// we don't need to check isOracle, it's already checked in anteHandler, and if some proposer filled in any invalid message it will be reject by anteHandler again in runTx
			keep = append(keep, tx)
			continue
		}
		msgOracle := msgs[0]
		if _, ok := seenFeederIDs[msgOracle.FeederID]; ok {
			// we only keep one tx for each feederID, we don't check if the piece index is expected, it's handled by anteHandler
			continue
		}
		if _, ok := collectingFeederIDs[msgOracle.FeederID]; !ok {
			// the piece included in this tx is not expected, we just remove it
			continue
		}
		keep = append(keep, tx)
		seenFeederIDs[msgOracle.FeederID] = struct{}{}
	}
	// fill txs from imua-mempool for missed txs which is required by 'collectinFeederIDs'
	if len(seenFeederIDs) < len(collectingFeederIDs) {
		cFIDKeys := make([]uint64, 0, len(collectingFeederIDs))
		for fID := range collectingFeederIDs {
			cFIDKeys = append(cFIDKeys, fID)
		}
		sort.Slice(cFIDKeys, func(i, j int) bool { return cFIDKeys[i] < cFIDKeys[j] })
		for _, expectedFeederID := range cFIDKeys {
			if _, ok := seenFeederIDs[expectedFeederID]; ok {
				continue
			}
			seenFeederIDs[expectedFeederID] = struct{}{}
			nextPieceID, ok := em.k.NextPieceIndexByFeederID(sdk.UnwrapSDKContext(ctx), expectedFeederID)
			if !ok {
				// this should not happen since the 'expectedFeederID' is valid for collectinng rawData
				continue
			}
			if tx := em.getTxByFeederIDPieceIndex(expectedFeederID, nextPieceID); tx != nil {
				keep = append(keep, tx)
			}
		}
	}

	if len(keep) > 0 {
		return IteratorFromSlice(keep)
	}

	return nil
}

// CountTx returns the number of txs in mempool
func (em *ImuaMempool) CountTx() int {
	return em.count
}

// Remove removes tx from mempool
func (em *ImuaMempool) Remove(tx sdk.Tx) error {
	// remove all pieces with indexes <= piece index of input msg
	piece, msgOracle, _, isRawData := em.GetPieceWithProof(tx)
	// we don't process tx not with rawdata
	if !isRawData {
		return nil
	}
	piecesCached, ok := em.cachedPieces[msgOracle.FeederID]
	if !ok {
		return nil
	}
	newPiecesCached := make(map[uint32][]*oracletypes.PieceWithProof)
	// it's safe to range over piecesCached since we keep the same piece slice for each piece index
	for index, piecesIndexCached := range piecesCached {
		if index > piece.Index {
			newPiecesCached[index] = piecesIndexCached
		} else {
			em.count -= len(piecesIndexCached)
		}
	}
	if len(newPiecesCached) == 0 {
		delete(em.cachedPieces, msgOracle.FeederID)
	} else if len(newPiecesCached) < len(piecesCached) {
		em.cachedPieces[msgOracle.FeederID] = newPiecesCached
	}
	return nil
}

// includesMsgOracle returns true if tx includes MsgCreatePrice
func (em *ImuaMempool) includesMsgOracle(tx sdk.Tx) bool {
	msgs := tx.GetMsgs()
	for _, msg := range msgs {
		if _, ok := msg.(*oracletypes.MsgCreatePrice); ok {
			return true
		}
	}
	return false
}

// GetPieceWithProof returns pieceWithProof and msgCreatePrice from tx
func (em *ImuaMempool) GetPieceWithProof(tx sdk.Tx) (pieceWithProof *oracletypes.PieceWithProof, msgO *oracletypes.MsgCreatePrice, isOracle, isRawData bool) {
	var msgs []*oracletypes.MsgCreatePrice
	msgs, isOracle, isRawData, _ = utils.IsValidOracleTx(tx)
	if !isRawData {
		return
	}
	msgO = msgs[0]

	pieceWithProof, _ = em.k.FeederManager.GetPieceWithProof(msgO)
	if pieceWithProof != nil {
		pieceWithProof.Tx = tx
	}
	return
}

// getTxByFeederIDPieceIndex returns the tx with the piece index and feederID
func (em *ImuaMempool) getTxByFeederIDPieceIndex(feederID uint64, pieceIndex uint32) sdk.Tx {
	pwf, ok := em.cachedPieces[feederID]
	if !ok {
		return nil
	}
	piecesCached, ok := pwf[pieceIndex]
	if !ok || len(piecesCached) == 0 {
		return nil
	}

	if len(piecesCached) > 1 {
		// we got different tx for the same piece, there must be at least one invalid piece
		// imua-mempool don't do the verify, we just pick the first one in cache and move it to the end of the list for that pieceIndex
		// we just remove the picked to the list end instead of deleting it since that't the duty of 'Remove'
		piecesCached = append(piecesCached[1:], piecesCached[0])
		pwf[pieceIndex] = piecesCached
	}
	// the first cached piece had been moved to the end
	return piecesCached[len(piecesCached)-1].Tx
}

// reset resets the mempool
func (em *ImuaMempool) reset() {
	em.cachedPieces = make(map[uint64]map[uint32][]*oracletypes.PieceWithProof)
	em.count = 0
}

// clearExpiredFeederIDcache clears the expired feederID cache
// if a feederID is not in 'collectingFeederIDs', we remove all pieces with that feederID from cache
// or if a feederID is in 'collectingFeederIDs', we remove all pieces with that feederID and piece index <= startBaseBlock
func (em *ImuaMempool) clearExpiredFeederIDcache(collectingFeederIDs map[uint64]uint64) {
	newCachedPieces := make(map[uint64]map[uint32][]*oracletypes.PieceWithProof)
	for feederID := range em.cachedPieces {
		startBaseBlock, ok := collectingFeederIDs[feederID]
		if !ok {
			continue
		}
		pieceMap, ok := em.cachedPieces[feederID]
		if !ok {
			continue
		}
		for pieceIdx, pwpList := range pieceMap {
			newPwpList := make([]*oracletypes.PieceWithProof, 0, len(pwpList))
			for _, pwp := range pwpList {
				if pwp.BaseBlock == startBaseBlock {
					newPwpList = append(newPwpList, pwp)
				} else {
					em.count--
				}
			}
			if len(newPwpList) > 0 {
				if len(newCachedPieces[feederID]) == 0 {
					newCachedPieces[feederID] = make(map[uint32][]*oracletypes.PieceWithProof)
				}
				newCachedPieces[feederID][pieceIdx] = newPwpList
			}
		}
	}
	if len(newCachedPieces) > 0 {
		em.cachedPieces = newCachedPieces
	}
}

// IteratorFromSlice returns an iterator from a slice of txs
func IteratorFromSlice(txList []sdk.Tx) *ImuaMemIterator {
	return &ImuaMemIterator{txList: txList}
}

type ImuaMemIterator struct {
	txList []sdk.Tx
	cursor int
}

func (ii *ImuaMemIterator) Next() mempool.Iterator {
	if ii.cursor >= len(ii.txList)-1 {
		return nil
	}
	ii.cursor++
	return ii
}

func (ii *ImuaMemIterator) Tx() sdk.Tx {
	if ii.cursor < len(ii.txList) {
		return ii.txList[ii.cursor]
	}
	return nil
}

func NewImuaMempool(oKeeper *oraclekeeper.Keeper, decoder sdk.TxDecoder) *ImuaMempool {
	return &ImuaMempool{
		cachedPieces: make(map[uint64]map[uint32][]*oracletypes.PieceWithProof),
		k:            oKeeper,
		txDecoder:    decoder,
	}
}
