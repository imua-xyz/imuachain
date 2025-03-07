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
func OracleCreatePriceTx(tx sdk.Tx) (msgsO []*oracletypes.MsgCreatePrice, validOracle, validRawData bool) {
	msgs := tx.GetMsgs()
	if len(msgs) == 0 {
		return nil, false, false
	}
	msgsO = make([]*oracletypes.MsgCreatePrice, 0, len(msgs))
	for _, msg := range msgs {
		msgO, ok := msg.(*oracletypes.MsgCreatePrice)
		if !ok {
			return nil, false, false
		}
		if msgO.IsPhaseTwo() {
			if len(msgs) > 1 {
				return nil, false, false
			}
			msgsO = append(msgsO, msgO)
			return msgsO, true, true
		}
		msgsO = append(msgsO, msgO)
	}
	return msgsO, true, false
}
