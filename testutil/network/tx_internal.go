package network

import (
	"context"

	sdkmath "cosmossdk.io/math"
	ctypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdktx "github.com/cosmos/cosmos-sdk/types/tx"
)

func (n *Network) SendTxInternal(msgs []sdk.Msg, keyName string, kr keyring.Keyring) (*ctypes.ResultBroadcastTx, error) {
	signedTxBytes, err := n.genSignedTxInternal(msgs, keyName, kr)
	if err != nil {
		return nil, err
	}
	// txClient.BroadcastTx()
	return n.Validators[0].RPCClient.BroadcastTxSync(context.Background(), signedTxBytes)
}

// GenSignedTx construct a tx with input messages and sign it privatekey stored in keyring
func (n *Network) genSignedTxInternal(msgs []sdk.Msg, keyName string, kr keyring.Keyring) ([]byte, error) {
	val := n.Validators[0]
	ctx := val.ClientCtx

	record, err := kr.Key(keyName)
	if err != nil {
		return nil, err
	}
	acc, err := record.GetAddress()
	if err != nil {
		return nil, err
	}
	txConfig := ctx.TxConfig
	txBuilder := txConfig.NewTxBuilder()
	txClient := sdktx.NewServiceClient(ctx)
	if err = txBuilder.SetMsgs(msgs...); err != nil {
		return nil, err
	}
	txBytes, err := txConfig.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		return nil, err
	}
	// simulate tx to get estimate gasLimit
	gasInfo, err := txClient.Simulate(context.Background(), &sdktx.SimulateRequest{
		TxBytes: txBytes,
	})
	if err != nil {
		return nil, err
	}
	// adjustment
	gasLimit := uint64(float64(gasInfo.GasInfo.GasUsed) * 1.5)
	gasLimitInt := sdkmath.NewIntFromUint64(gasLimit)
	l := len(n.Config.MinGasPrices) - len(n.Config.NativeDenom)
	fee := sdkmath.ZeroInt()
	if l > 0 {
		minGasPriceStr := n.Config.MinGasPrices[:l]
		minGasPrice, _ := sdkmath.NewIntFromString(minGasPriceStr)
		fee = gasLimitInt.Mul(minGasPrice)
	}

	txBuilder.SetGasLimit(gasLimit)
	txBuilder.SetFeeAmount(sdk.Coins{sdk.NewCoin(n.Config.NativeDenom, fee)})
	num, seq, err := ctx.AccountRetriever.GetAccountNumberSequence(ctx, acc)
	if err != nil {
		return nil, err
	}
	txf := tx.Factory{}.
		WithChainID(ctx.ChainID).
		WithKeybase(kr).
		WithTxConfig(txConfig).
		WithAccountNumber(num).
		WithSequence(seq)

	if err := tx.Sign(txf, keyName, txBuilder, true); err != nil {
		return nil, err
	}

	signedTx := txBuilder.GetTx()
	if txBytes, err = txConfig.TxEncoder()(signedTx); err != nil {
		return nil, err
	}

	return txBytes, nil
}
