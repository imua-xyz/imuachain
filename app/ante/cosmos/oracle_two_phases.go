package cosmos

import (
	"errors"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/imua-xyz/imuachain/app/ante/utils"
)

type OracleTwoPhasesDecorator struct {
	ok utils.OracleKeeper
}

func NewOracleTwoPhasesDecorator(oracleKeeper utils.OracleKeeper) OracleTwoPhasesDecorator {
	return OracleTwoPhasesDecorator{
		ok: oracleKeeper,
	}
}

func (otpd OracleTwoPhasesDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	msgs, _, isRawData := utils.OracleCreatePriceTx(tx)
	if isRawData {
		pieceWithProof, ok := otpd.ok.GetPieceWithProof(msgs[0])
		// valid failed when getting pieceWithProof
		if !ok {
			return ctx, errors.New("failed to valid and get pieceWithProof with a tx with oracle rawData")
		}
		proofPath := otpd.ok.MinimalProofPathByIndex(msgs[0].FeederID, uint32(pieceWithProof.Index))
		if len(proofPath) != int(pieceWithProof.ProofSize()) {
			return ctx, fmt.Errorf("rawData proofPath size not match, expected:%d, got:%d", len(proofPath), pieceWithProof.ProofSize())
		}
		// the proofPath need to be exactly the same of both value and order
		for i, index := range proofPath {
			if pieceWithProof.Proof[i].Index != index {
				return ctx, fmt.Errorf("rawData proofPath didn't include necessary index on position:%d of path:%d", i, index)
			}
		}
	}

	return next(ctx, tx, simulate)
}
