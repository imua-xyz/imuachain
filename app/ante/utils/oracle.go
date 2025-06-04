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

// IsOraclePhaseTwoTx checks if the tx is a phase two oracle tx which means it includes exactly one oracle message of raw data phase two
func IsOraclePhaseTwoTx(tx sdk.Tx) bool {
	msgs := tx.GetMsgs()
	if len(msgs) != 1 {
		return false
	}
	msg, ok := msgs[0].(*oracletypes.MsgCreatePrice)
	if !ok {
		return false
	}
	return msg.IsPhaseTwo()
}

// IsValidOracleTx checks wether the tx is a valid oracle tx
// return values:
// isOracle: all messages in the tx are oracle messages and non of them is phase two
// isPhaseTwo: the tx includes exactly one oracle message and it is phase two
// mixed:  mixed message types in the tx which has both oracle types and non-oracle types
func IsValidOracleTx(tx sdk.Tx) (msgOracles []*oracletypes.MsgCreatePrice, isOracle bool, isPhaseTwo bool, mixed bool) {
	msgs := tx.GetMsgs()
	l := len(msgs)
	if l == 0 {
		return nil, false, false, false
	}

	hasOracleMsg := false
	for _, msg := range msgs {
		msgOracle, ok := msg.(*oracletypes.MsgCreatePrice)
		if ok {
			msgOracles = append(msgOracles, msgOracle)
			hasOracleMsg = true
			if msgOracle.IsPhaseTwo() {
				if l == 1 {
					return msgOracles, true, true, false
				}
				return nil, false, false, true
			}
		} else if hasOracleMsg {
			return nil, false, false, true
		}
	}
	return msgOracles, hasOracleMsg, false, false
}
