package network

import (
	"context"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
)

// SendTx construct and sign that tx with input msgs
func (n *Network) SendTx(msgs []sdk.Msg, keyName string, keyring keyring.Keyring) error {
	txf, ctx, err := generateTxf(n.Validators[0].ClientCtx, keyName, keyring, 1.5, n.Config.MinGasPrices)
	if err != nil {
		return err
	}
	return tx.BroadcastTx(ctx, txf, msgs...)
}

// SendTxOracleCreatePrice consturct and sign that tx with input msgs, it's different from SendTx, since when we use ed25519 for oracle senario, we allowed that signer is an unexists account, this implementation skip the 'accoutn exists' related checks
// Also, if you want to sign some normal message (not oracle-create-price) with ed25519, just use SendTx is fine, we support ed25519 signing in keyring
func (n *Network) SendTxOracleCreateprice(msgs []sdk.Msg, keyName string, keyring keyring.Keyring) error {
	txf, ctx, err := generateTxf(n.Validators[0].ClientCtx, keyName, keyring, 1.5, n.Config.MinGasPrices)
	if err != nil {
		return err
	}
	// calculate gas
	txBytes, err := txf.BuildSimTx(msgs...)
	if err != nil {
		return err
	}
	txClient := txtypes.NewServiceClient(ctx)
	simRes, err := txClient.Simulate(context.Background(), &txtypes.SimulateRequest{
		TxBytes: txBytes,
	})
	if err != nil {
		return err
	}
	gasAdjusted := uint64(txf.GasAdjustment() * float64(simRes.GasInfo.GasUsed))
	txf.WithGas(gasAdjusted)
	transaction, err := txf.BuildUnsignedTx(msgs...)
	if err != nil {
		return err
	}
	if err := tx.Sign(txf, ctx.GetFromName(), transaction, true); err != nil {
		return err
	}
	txBytes, err = ctx.TxConfig.TxEncoder()(transaction.GetTx())
	if err != nil {
		return err
	}
	res, err := ctx.BroadcastTx(txBytes)
	if err != nil {
		return err
	}
	return ctx.PrintProto(res)
}

func generateTxf(ctx client.Context, keyName string, kr keyring.Keyring, adjustment float64, minGasPrice string) (tx.Factory, client.Context, error) {
	var txf tx.Factory
	record, err := kr.Key(keyName)
	if err != nil {
		return txf, ctx, err
	}
	acc, err := record.GetAddress()
	if err != nil {
		return txf, ctx, err
	}
	ctx.FromAddress = acc
	ctx.FromName = keyName
	ctx.SkipConfirm = true
	txf = tx.Factory{}.
		WithChainID(ctx.ChainID).
		WithKeybase(kr).
		WithTxConfig(ctx.TxConfig).
		WithSignMode(signing.SignMode_SIGN_MODE_DIRECT).
		WithGasAdjustment(adjustment).
		WithAccountRetriever(ctx.AccountRetriever).
		WithGasPrices(minGasPrice).
		WithSimulateAndExecute(true)

	ctx.BroadcastMode = flags.BroadcastSync
	return txf, ctx, nil
}
