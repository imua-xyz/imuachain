package batch

import (
	"context"
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	sdkmath "cosmossdk.io/math"

	"github.com/ExocoreNetwork/exocore/cmd/config"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/cosmos/cosmos-sdk/client"
	sdktx "github.com/cosmos/cosmos-sdk/client/tx"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"golang.org/x/xerrors"
)

const GasAdjustment = 2

// WaitForEvmTxReceipt Wait for the transaction receipt and confirm if it's mined successfully
func WaitForEvmTxReceipt(client *ethclient.Client, txHash common.Hash, waitDuration, waitExpiration time.Duration) (*types.Receipt, error) {
	startTime := time.Now() // Record the start time of the function
	for {
		// Check if the wait time has exceeded the expiration limit
		if time.Since(startTime) > waitExpiration {
			return nil, fmt.Errorf("exceeded wait expiration time of %v，txHash:%s", waitExpiration, txHash)
		}

		// Attempt to retrieve the transaction receipt
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		receipt, err := client.TransactionReceipt(ctx, txHash)
		cancel()
		if err == nil {
			// If the receipt is found, return it
			return receipt, nil
		}

		// Log a message indicating the receipt is not yet available
		logger.Info("can't get the receipt of EVM transaction, continue waiting", "waitDuration", waitDuration, "err", err)

		// Wait for the specified duration before retrying
		time.Sleep(waitDuration)
	}
}

// SignAndSendEvmTx : If an address exists in the sync map, we use the nonce maintained in the sync map.
// If it does not exist, the nonce strategy is determined based on the input parameters.
// For AVS, operators and faucet addresses, their nonce values are maintained in the sync map.
// However, for subsequent test transactions such as deposit, delegation, undelegation, and withdrawal sent
// by staker addresses, the nonce is managed independently during the transaction process.
// Thus, this method can be used in three scenarios:
// 1. Directly retrieving the nonce within this function.
// 2. Using the nonce maintained in the sync map.
// 3. For staker-related batch tests, relying on the external nonce automatically maintained by the caller.
func (m *Manager) SignAndSendEvmTx(txInfo *EvmTxInQueue) (common.Hash, error) {
	msg := ethereum.CallMsg{
		From: txInfo.From,
		To:   txInfo.ToAddr,
		Data: txInfo.Data,
	}
	sk := txInfo.Sk
	nonce := txInfo.Nonce
	ethC := m.NodeEVMHTTPClients[DefaultNodeIndex]
	// load the nonce from the sync map
	var nonceFromSyncMap bool
	loadNonce, err := m.LoadSequence(crypto.PubkeyToAddress(sk.PublicKey))
	if err == nil {
		// The nonce of the faucet, AVSs, and operators have been stored in the sync map,
		// so load nonce from the sync map.
		nonceFromSyncMap = true
		nonce = loadNonce
	}
	if !txInfo.UseExternalNonce && !nonceFromSyncMap {
		nonce, err = ethC.NonceAt(m.ctx, msg.From, nil)
		if err != nil {
			return common.Hash{}, err
		}
	}

	estimateGas, err := ethC.EstimateGas(m.ctx, msg)
	if err != nil {
		return common.Hash{}, err
	}
	gasPrice, err := ethC.SuggestGasPrice(m.ctx)
	if err != nil {
		return common.Hash{}, err
	}

	retTx := types.NewTx(&types.LegacyTx{
		Nonce:    nonce,
		To:       msg.To,
		Value:    msg.Value,
		Gas:      uint64(math.Round(float64(estimateGas) * GasAdjustment)),
		GasPrice: gasPrice,
		Data:     msg.Data,
	})
	logger.Info("SignAndSendEvmTx", "from", strings.ToLower(txInfo.From.String()), "to", retTx.To(),
		"nonce", retTx.Nonce(), "value", retTx.Value(), "gasPrice", retTx.GasPrice(),
		"gas", retTx.Gas(), "dataLength", len(retTx.Data()), "time", time.Now().String())
	signTx, err := types.SignTx(retTx, m.EthSigner, sk)
	if err != nil {
		return common.Hash{}, err
	}

	err = ethC.SendTransaction(m.ctx, signTx)
	if err != nil {
		return common.Hash{}, err
	}

	if nonceFromSyncMap {
		// update the sequence if we use faucet sk to sign the tx
		m.Sequences.Store(crypto.PubkeyToAddress(sk.PublicKey), loadNonce+1)
	}
	return signTx.Hash(), nil
}

func (m *Manager) SignSendEvmTxAndWait(txInfo *EvmTxInQueue) error {
	txHash, err := m.SignAndSendEvmTx(txInfo)
	if err != nil {
		return err
	}
	receipt, err := WaitForEvmTxReceipt(m.NodeEVMHTTPClients[DefaultNodeIndex], txHash, m.WaitDuration, m.WaitExpiration)
	if err != nil {
		return err
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		logger.Info(" the evm tx has been on chain but the execution is Failed", "txHash", txHash)
		return xerrors.Errorf("Failed evm tx receipt, txID:%s", txHash)
	}
	logger.Info("the evm tx has been on chain successfully", "blockNumber", receipt.BlockNumber, "txID", txHash)
	return nil
}

// BroadcastTxWithRes is copied from the BroadcastTx of the SDK.
// The difference is that it removes the confirmation and returns
// the result to enable waiting for on-chain confirmation.
func BroadcastTxWithRes(clientCtx client.Context, txf sdktx.Factory, msgs ...sdktypes.Msg) (res *sdktypes.TxResponse, err error) {
	txf, err = txf.Prepare(clientCtx)
	if err != nil {
		return nil, err
	}

	if txf.SimulateAndExecute() || clientCtx.Simulate {
		if clientCtx.Offline {
			return nil, xerrors.New("cannot estimate gas in offline mode")
		}

		_, adjusted, err := sdktx.CalculateGas(clientCtx, txf, msgs...)
		if err != nil {
			return nil, err
		}

		txf = txf.WithGas(adjusted)
		_, _ = fmt.Fprintf(os.Stderr, "%s\n", sdktx.GasEstimateResponse{GasEstimate: txf.Gas()})
	}

	if clientCtx.Simulate {
		return nil, nil
	}

	tx, err := txf.BuildUnsignedTx(msgs...)
	if err != nil {
		return nil, err
	}

	err = sdktx.Sign(txf, clientCtx.GetFromName(), tx, true)
	if err != nil {
		return nil, err
	}

	txBytes, err := clientCtx.TxConfig.TxEncoder()(tx.GetTx())
	if err != nil {
		return nil, err
	}

	// broadcast to a Tendermint node
	return clientCtx.BroadcastTx(txBytes)
}

// SignAndSendMultiMsgs all messages are signed by one private key
func (m *Manager) SignAndSendMultiMsgs(
	clientCtx client.Context, fromName string,
	broadcastMode string, msgs ...sdktypes.Msg,
) ([]*sdktypes.TxResponse, error) {
	keyRecord, err := clientCtx.Keyring.Key(fromName)
	if err != nil {
		return nil, xerrors.Errorf("SignAndSendMultiMsgs, can't get key record,fromName:%s,err:%w", fromName, err)
	}
	fromAddr, err := keyRecord.GetAddress()
	if err != nil {
		return nil, xerrors.Errorf("SignAndSendMultiMsgs, can't get address from the key record,fromName:%s,err:%w", fromName, err)
	}
	logger.Info("from name and Addr is:", "fromName", fromName, "fromAddr", fromAddr.String())
	clientCtx = clientCtx.
		WithFromName(fromName).
		WithFromAddress(fromAddr).
		WithBroadcastMode(broadcastMode)

	txFactory := sdktx.Factory{}
	txFactory = txFactory.
		WithChainID(clientCtx.ChainID).
		WithKeybase(clientCtx.Keyring).
		WithTxConfig(clientCtx.TxConfig).
		WithGasAdjustment(GasAdjustment).
		WithSimulateAndExecute(true).
		WithAccountRetriever(clientCtx.AccountRetriever)

	if fromName == FaucetSKName ||
		strings.HasPrefix(fromName, OperatorNamePrefix) ||
		strings.HasPrefix(fromName, AVSNamePrefix) {
		sequence, err := m.LoadSequence(common.BytesToAddress(fromAddr))
		if err != nil {
			return nil, err
		}
		logger.Info("the sequence loaded from the sync map is:",
			"fromAddr", common.BytesToAddress(fromAddr), "sequence", sequence)
		txFactory = txFactory.WithSequence(sequence)
	}
	responseList := make([]*sdktypes.TxResponse, 0)
	for _, msg := range msgs {
		// get gas price
		suggestGasPrice, err := m.GasPrice()
		if err != nil {
			return nil, xerrors.Errorf("Failed to get suggest gas price, error:%s", err.Error())
		}
		gasPrices := sdktypes.NewDecCoins(
			sdktypes.NewDecCoin(
				config.BaseDenom,
				sdkmath.NewIntFromBigInt(suggestGasPrice),
			),
		).String()
		txFactory = txFactory.
			WithGasPrices(gasPrices).
			WithFeePayer(fromAddr)
		res, err := BroadcastTxWithRes(clientCtx, txFactory, msg)
		if err != nil {
			return nil, xerrors.Errorf("Failed to broadcast tx, error:%s", err.Error())
		}
		if res.Code != 0 {
			return nil, xerrors.Errorf("Failed to broadcast tx, response code:%v", res.Code)
		}
		responseList = append(responseList, res)
		// clientCtx.PrintProto(res)

		// increase the sequence and sleep before sending the next cosmos tx
		nextSequence := txFactory.Sequence() + 1
		txFactory = txFactory.WithSequence(nextSequence)
		if fromName == FaucetSKName ||
			strings.HasPrefix(fromName, OperatorNamePrefix) ||
			strings.HasPrefix(fromName, AVSNamePrefix) {
			logger.Info("save the sequence",
				"fromAddr", common.BytesToAddress(fromAddr).String(), "nextSequence", nextSequence)
			m.Sequences.Store(common.BytesToAddress(fromAddr), nextSequence)
		}
		sleepDur := time.Duration(1000/m.config.TxNumberPerSec) * time.Millisecond
		time.Sleep(sleepDur)
	}
	return responseList, nil
}

func (m *Manager) WaitForCosmosTxs(responseList []*sdktypes.TxResponse, waitDuration, waitExpiration time.Duration) error {
	for _, res := range responseList {
		startTime := time.Now() // Record the start time of the function
		for {
			// Check if the wait time has exceeded the expiration limit
			if time.Since(startTime) > waitExpiration {
				return fmt.Errorf("WaitForCosmosTxs: exceeded wait expiration time of %v", waitExpiration)
			}

			queryRes, err := tx.QueryTx(m.NodeClientCtx[DefaultNodeIndex], res.TxHash)
			if err != nil {
				logger.Info("can't query the cosmos tx, continue waiting", "txHash", res.TxHash, "err", err)
				// Wait for the specified duration before retrying
				// Log a message indicating the receipt is not yet available
				time.Sleep(waitDuration)
				continue
			}
			if queryRes.Code == 0 {
				// the tx has been on-chain successfully
				logger.Info("the cosmos tx has been on chain successfully", "txHash", res.TxHash)
			} else {
				logger.Info("the cosmos tx has been on chain successfully, but the execution is Failed", "txHash", res.TxHash, "code", queryRes.Code, "log", queryRes.RawLog)
			}
			break
		}
	}
	return nil
}

func (m *Manager) SignSendMultiMsgsAndWait(
	clientCtx client.Context, fromName string,
	broadcastMode string, msgs ...sdktypes.Msg,
) error {
	resList, err := m.SignAndSendMultiMsgs(clientCtx, fromName, broadcastMode, msgs...)
	if err != nil {
		return err
	}
	return m.WaitForCosmosTxs(resList,
		m.WaitDuration,
		m.WaitExpiration)
}
