package network

import (
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
)

// SendTx construct and sign that tx with input msgs
func (n *Network) SendTx(msgs []sdk.Msg, keyName string, keyring keyring.Keyring) error {
	ctx := n.Validators[0].ClientCtx
	record, err := keyring.Key(keyName)
	if err != nil {
		return err
	}
	acc, err := record.GetAddress()
	if err != nil {
		return err
	}
	ctx.FromAddress = acc
	ctx.FromName = keyName
	ctx.SkipConfirm = true
	txf := tx.Factory{}.
		WithChainID(ctx.ChainID).
		WithKeybase(keyring).
		WithTxConfig(ctx.TxConfig).
		WithSignMode(signing.SignMode_SIGN_MODE_DIRECT).
		WithGasAdjustment(1.5).
		WithAccountRetriever(ctx.AccountRetriever).
		WithGasPrices(n.Config.MinGasPrices).
		WithSimulateAndExecute(true)

	ctx.BroadcastMode = flags.BroadcastSync
	return tx.BroadcastTx(ctx, txf, msgs...)
}
