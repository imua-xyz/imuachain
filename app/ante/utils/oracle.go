package utils

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	oracletypes "github.com/imua-xyz/imuachain/x/oracle/types"
)

// TxSizeLimit limits max size of a create-price tx, this is calculated based on one nativeTokenbalance message of worst case(max size), which will need 576 bytes for balance update
// 48*1024+5*32+6*4 +... TODO(leonz): ensure the proto cost, now use a fixed 200B
const (
	TxSizeLimitOraclePrice   = 400
	TxSizeLimitOracleRawData = 49600
)

// TODO(leonz): return additional error info
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
