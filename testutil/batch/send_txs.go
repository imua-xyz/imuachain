package batch

import (
	"context"
	"crypto/ecdsa"
	"math"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"golang.org/x/xerrors"
)

const GasAdjustment = 1.25

type BasicEvmTxRequirements struct {
	ctx          context.Context
	sk           *ecdsa.PrivateKey
	caller       common.Address
	signer       types.Signer
	ethC         *ethclient.Client
	WaitDuration time.Duration
}

// WaitForEvmTxReceipt Wait for the transaction receipt and confirm if it's mined successfully
func WaitForEvmTxReceipt(client *ethclient.Client, txHash common.Hash, waitDuration time.Duration) (*types.Receipt, error) {
	for {
		// Check if the transaction receipt is available
		receipt, err := client.TransactionReceipt(context.Background(), txHash)
		if err == nil {
			// If the receipt is found, return it
			return receipt, nil
		}
		logger.Info("can't get the receipt of evm tx, continue waiting", "err", err)
		// If the receipt is not available yet, wait and try again
		time.Sleep(waitDuration)
	}
}

func SignAndSendEvmTx(basicInfo *BasicEvmTxRequirements, txInfo *EvmTxInQueue) (common.Hash, error) {
	msg := ethereum.CallMsg{
		From: txInfo.From,
		To:   txInfo.ToAddr,
		Data: txInfo.Data,
	}
	sk := txInfo.sk
	if sk == nil {
		// using default sk to send this transaction
		msg.From = basicInfo.caller
		sk = basicInfo.sk
	}
	estimateGas, err := basicInfo.ethC.EstimateGas(basicInfo.ctx, msg)
	if err != nil {
		return common.Hash{}, err
	}
	gasPrice, err := basicInfo.ethC.SuggestGasPrice(basicInfo.ctx)
	if err != nil {
		return common.Hash{}, err
	}

	nonce := txInfo.Nonce
	if !txInfo.UseExternalNonce {
		nonce, err = basicInfo.ethC.NonceAt(basicInfo.ctx, msg.From, nil)
		if err != nil {
			return common.Hash{}, err
		}
	}

	retTx := types.NewTx(&types.LegacyTx{
		Nonce:    nonce,
		To:       msg.To,
		Value:    msg.Value,
		Gas:      uint64(math.Round(float64(estimateGas) * GasAdjustment)),
		GasPrice: gasPrice,
		Data:     msg.Data,
	})
	signTx, err := types.SignTx(retTx, basicInfo.signer, sk)
	if err != nil {
		return common.Hash{}, err
	}

	err = basicInfo.ethC.SendTransaction(basicInfo.ctx, signTx)
	if err != nil {
		return common.Hash{}, err
	}
	return signTx.Hash(), nil
}

func SignSendEvmTxAndWait(basicInfo *BasicEvmTxRequirements, txInfo *EvmTxInQueue) error {
	txHash, err := SignAndSendEvmTx(basicInfo, txInfo)
	if err != nil {
		return err
	}
	receipt, err := WaitForEvmTxReceipt(basicInfo.ethC, txHash, basicInfo.WaitDuration)
	if err != nil {
		return err
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return xerrors.Errorf("failed evm tx receipt, txID:%s", txHash)
	}
	return nil
}

// SignAndSendMultiMsgs all messages are signed by one private key
func SignAndSendMultiMsgs(
	clientCtx client.Context, fromName string,
	broadcastMode string, msgs ...sdktypes.Msg,
) error {
	txFactory := tx.Factory{}
	txFactory = txFactory.
		WithChainID(clientCtx.ChainID).
		WithKeybase(clientCtx.Keyring).
		WithTxConfig(clientCtx.TxConfig).
		WithGasAdjustment(GasAdjustment).
		WithSimulateAndExecute(true)
	clientCtx.WithFrom(fromName)
	clientCtx.WithBroadcastMode(broadcastMode)
	for _, msg := range msgs {
		err := tx.BroadcastTx(clientCtx, txFactory, msg)
		if err != nil {
			return xerrors.Errorf("failed to broadcast tx, error:%s", err.Error())
		}
	}
	return nil
}
