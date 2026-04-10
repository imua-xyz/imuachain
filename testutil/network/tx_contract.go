package network

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/imua-xyz/imuachain/precompiles/assets/testdata"
)

// DeployOracleGatewayContract deploys the minimal oracle gateway used in e2e tests.
func (n Network) DeployOracleGatewayContract(oracleCaller common.Address) (common.Address, error) {
	ctorArgs, err := testdata.OracleGatewayContract.ABI.Pack("", oracleCaller)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to pack constructor args: %w", err)
	}
	data := append(append([]byte{}, testdata.OracleGatewayContract.Bin...), ctorArgs...)
	return n.deployContract(data)
}

func (n Network) deployContract(data []byte) (common.Address, error) {
	ctx := context.Background()
	ethC := n.Validators[0].JSONRPCClient

	chainID, err := ethC.ChainID(ctx)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to get chainID: %w", err)
	}

	nonce, err := ethC.NonceAt(ctx, callAddr, nil)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to get nonce for address %s: %w", callAddr.Hex(), err)
	}

	gasPrice, err := ethC.SuggestGasPrice(ctx)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to get suggested gas price: %w", err)
	}
	if bal, err := ethC.BalanceAt(ctx, callAddr, nil); err == nil && bal.Sign() == 0 {
		return common.Address{}, fmt.Errorf("insufficient deployer balance: 0")
	}

	est, err := ethC.EstimateGas(ctx, ethereum.CallMsg{From: callAddr, Data: data})
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to estimate deploy gas: %w", err)
	}
	gasLimit := est + 50_000
	tx := types.NewTx(&types.LegacyTx{
		Nonce:    nonce,
		To:       nil,
		Value:    big.NewInt(0),
		Gas:      gasLimit,
		GasPrice: gasPrice,
		Data:     data,
	})
	signer := types.LatestSignerForChainID(chainID)
	signTx, err := types.SignTx(tx, signer, sk)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to sign transaction: %w", err)
	}

	if err := ethC.SendTransaction(ctx, signTx); err != nil {
		return common.Address{}, fmt.Errorf("failed to send transaction: %w", err)
	}

	// Wait for the deployment to be included.
	var (
		receiptStatus uint64
		receiptFound  bool
		receiptGas    uint64
	)
	for i := 0; i < 40; i++ {
		receipt, err := ethC.TransactionReceipt(ctx, signTx.Hash())
		if err == nil {
			receiptStatus = receipt.Status
			receiptGas = receipt.GasUsed
			receiptFound = true
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	if !receiptFound {
		return common.Address{}, fmt.Errorf("contract deployment receipt not found")
	}
	if receiptStatus != types.ReceiptStatusSuccessful {
		return common.Address{}, fmt.Errorf("contract deployment failed (status=%d gas_used=%d gas_limit=%d)", receiptStatus, receiptGas, gasLimit)
	}

	return crypto.CreateAddress(callAddr, nonce), nil
}
