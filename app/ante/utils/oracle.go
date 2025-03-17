package utils

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	oracletypes "github.com/imua-xyz/imuachain/x/oracle/types"
)

const (
	// TxSizeLimitOraclePrice is the max size of oracle price tx without raw data piece
	TxSizeLimitOraclePrice = 400
	// TxSizeLimitOracleRawData is the max size of oracle price tx with raw data piece
	TxSizeLimitOracleRawData = 49600
)

// OracleCreatePriceTx returns the oracle create price txs in the given tx, and whether the tx is valid oracle tx and valid raw data tx
func OracleCreatePriceTx(tx sdk.Tx) (msgsOracle []*oracletypes.MsgCreatePrice, validOracle, validRawData, mixed bool) {
	msgs := tx.GetMsgs()
	if len(msgs) == 0 {
		return nil, false, false, false
	}
	msgsOracle = make([]*oracletypes.MsgCreatePrice, 0, len(msgs))

	l := len(msgs)
	for i := 0; i < l; i++ {
		msg := msgs[i]
		//	for _, msg := range msgs {
		msgOracle, ok := msg.(*oracletypes.MsgCreatePrice)

		if !ok {
			// when we found one non-oralce message in this tx, this must not be a valid oracle tx
			if i > 0 {
				return nil, false, false, true
			}
			for j := 1; j < l; j++ {
				msg = msgs[j]
				if _, ok = msg.(*oracletypes.MsgCreatePrice); ok {
					return nil, false, false, true
				}
			}
			// this is a valid(format) tx which don't have any oracle message
			return nil, false, false, false
		}

		if msgOracle.IsPhaseTwo() {
			if len(msgs) > 1 {
				return nil, false, false, true
			}
			msgsOracle = append(msgsOracle, msgOracle)
			return msgsOracle, true, true, false
		}
		msgsOracle = append(msgsOracle, msgOracle)
	}
	return msgsOracle, true, false, false
}
