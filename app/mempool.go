package app

import (
	"context"
	"errors"
	"slices"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/mempool"

	"github.com/imua-xyz/imuachain/app/ante/utils"
	oraclekeeper "github.com/imua-xyz/imuachain/x/oracle/keeper"
	oracletypes "github.com/imua-xyz/imuachain/x/oracle/types"
)

type ImuaMempool struct {
	// feederID -> []PieceWithProof, cached peiceWithProof for feederID
	cachedPieces map[uint64]map[uint32][]*oracletypes.PieceWithProof
	k            *oraclekeeper.Keeper
	count        int
	// load from config file as a config for mempool
	//cacheWindow int
	txDecoder sdk.TxDecoder
}

// Insert inserts a tx into mempool, currently only used for rawData from tx related to 2-phases aggregation of oracle module
func (em *ImuaMempool) Insert(ctx context.Context, tx sdk.Tx) error {
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

func (em *ImuaMempool) Select(ctx context.Context, txList [][]byte) mempool.Iterator {
	// remove all expired tx, when Select for block 100, all txs belongs to 99 or before should be removed

	// feederIDS:[]uint64, which are expecting rawData
	// []Tx, each feederID must have one tx
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
			if _, _, isRawData := utils.OracleCreatePriceTx(tx); isRawData {
				continue
			}
			txDecodedList = append(txDecodedList, tx)
		}
		return IteratorFromSlice(txDecodedList)
	}

	em.clearExpiredFeederIDcache(collectingFeederIDs)

	seenFeederIDs := make(map[uint64]struct{})
	keep := make([]sdk.Tx, 0, len(txList))
	for _, txBytes := range txList {
		tx, err := em.txDecoder(txBytes)
		if err != nil {
			keep = append(keep, tx)
			continue
		}
		// TODO:(leonz) remove this
		if !em.includesMsgOracle(tx) {
			keep = append(keep, tx)
			continue
		}
		msgs, _, isRawData := utils.OracleCreatePriceTx(tx)
		if !isRawData {
			// we don't need to check isOracle, it's already checked in anteHandler, and if some proposer filled in any invalid message it will be reject by anteHandler again in runTx
			keep = append(keep, tx)
			continue
		}
		msgOracle := msgs[0]
		// check this before collectingFeederIds 'contain', since it's faster for map than slice
		if _, ok := seenFeederIDs[msgOracle.FeederID]; ok {
			// we only keep one tx for each feederID, we don't check if the piece index is expected, it's handled by anteHandler
			continue
		}
		if !slices.Contains(collectingFeederIDs, msgOracle.FeederID) {
			// the piece included in this tx is not expected, we just remove it
			continue
		}
		keep = append(keep, tx)
		seenFeederIDs[msgOracle.FeederID] = struct{}{}
	}
	// fill txs from imua-mempool for missed txs which is required by 'collectinFeederIDs'
	if len(seenFeederIDs) < len(collectingFeederIDs) {
		for _, expectedFeederID := range collectingFeederIDs {
			if _, ok := seenFeederIDs[expectedFeederID]; ok {
				continue
			}
			seenFeederIDs[expectedFeederID] = struct{}{}
			nextPieceID, ok := em.k.NextPieceIndexByFeederID(sdk.UnwrapSDKContext(ctx), expectedFeederID)
			if !ok {
				// this should not happen since the 'expectedFeederID' is valid for collectinng rawData
				continue
			}
			seenFeederIDs[expectedFeederID] = struct{}{}
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

func (em *ImuaMempool) CountTx() int {
	return em.count
}

// Remove removes tx from mempool
func (em *ImuaMempool) Remove(tx sdk.Tx) error {
	// TODO(leonz): clear history sealed round on block change
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
	msgs, isOracle, isRawData = utils.OracleCreatePriceTx(tx)
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
		// imua-mempool don't do the verify, we just pick the first one in cache and move it the the end of the list for that pieceIndex
		// we just remove the picked to the list end instead of deleting it since that't the duty of 'Remove'
		piecesCached = append(piecesCached[1:], piecesCached[0])
	}
	// the first cached piece had been moved to the end
	return piecesCached[len(piecesCached)-1].Tx
}

func (em *ImuaMempool) reset() {
	em.cachedPieces = make(map[uint64]map[uint32][]*oracletypes.PieceWithProof)
	em.count = 0
}

func (em *ImuaMempool) clearExpiredFeederIDcache(collectingFeederIDs []uint64) {
	feederIDs := make(map[uint64]struct{})
	for _, feederID := range collectingFeederIDs {
		feederIDs[feederID] = struct{}{}
	}

	for feederID := range em.cachedPieces {
		if _, ok := feederIDs[feederID]; !ok {
			delete(em.cachedPieces, feederID)
		}
	}
}

func IteratorFromSlice(txList []sdk.Tx) *ImuaMemIterator {
	// TODO:(leonz) implement me
	return &ImuaMemIterator{txList: txList}
	return nil
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
	return nil
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
