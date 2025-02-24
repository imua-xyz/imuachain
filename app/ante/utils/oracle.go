package utils

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	oracletypes "github.com/imua-xyz/imuachain/x/oracle/types"
)

// TxSizeLimit limits max size of a create-price tx, this is calculated based on one nativeTokenbalance message of worst case(max size), which will need 576 bytes for balance update
const TxSizeLimit = 1000

func IsOracleCreatePriceTx(tx sdk.Tx) bool {
	msgs := tx.GetMsgs()
	if len(msgs) == 0 {
		return false
	}
	for _, msg := range msgs {
		if _, ok := msg.(*oracletypes.MsgCreatePrice); !ok {
			return false
		}
	}
	return true
}
