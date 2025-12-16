package keeper

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"

	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/ethereum/go-ethereum/common"

	tmbytes "github.com/cometbft/cometbft/libs/bytes"
	tmtypes "github.com/cometbft/cometbft/types"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	"github.com/armon/go-metrics"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"

	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/evmos/evmos/v16/x/evm/types"
	imuachainevmtypes "github.com/imua-xyz/imuachain/x/evm/types"
)

var (
	_ types.MsgServer             = &Keeper{}
	_ imuachainevmtypes.MsgServer = &Keeper{}
)

// EthereumTx implements the gRPC MsgServer interface. It receives a transaction which is then
// executed (i.e applied) against the go-ethereum EVM. The provided SDK Context is set to the Keeper
// so that it can implements and call the StateDB methods without receiving it as a function
// parameter.
func (k *Keeper) EthereumTx(goCtx context.Context, msg *types.MsgEthereumTx) (*types.MsgEthereumTxResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	sender := msg.From
	tx := msg.AsTransaction()
	txIndex := k.GetTxIndexTransient(ctx)

	labels := []metrics.Label{
		telemetry.NewLabel("tx_type", fmt.Sprintf("%d", tx.Type())),
	}
	if tx.To() == nil {
		labels = append(labels, telemetry.NewLabel("execution", "create"))
	} else {
		labels = append(labels, telemetry.NewLabel("execution", "call"))
	}

	response, err := k.ApplyTransaction(ctx, tx)
	if err != nil {
		return nil, errorsmod.Wrap(err, "failed to apply transaction")
	}

	err = k.postProcessResponse(
		ctx, labels, response,
		sender, txIndex, tx.Gas(), tx.Value(), tx.To(), tx.Type(),
	)
	if err != nil {
		// leave it unchanged since callee has already wrapped it
		return nil, err
	}
	return response, nil
}

func (k *Keeper) postProcessResponse(
	ctx sdk.Context, labels []metrics.Label, response *types.MsgEthereumTxResponse,
	sender string, txIndex uint64, txGas uint64, txValue *big.Int, txTo *common.Address,
	txType uint8,
) error {
	defer func() {
		telemetry.IncrCounterWithLabels(
			[]string{"tx", "msg", "ethereum_tx", "total"},
			1,
			labels,
		)

		if response.GasUsed != 0 {
			telemetry.IncrCounterWithLabels(
				[]string{"tx", "msg", "ethereum_tx", "gas_used", "total"},
				float32(response.GasUsed),
				labels,
			)

			// Observe which users define a gas limit >> gas used. Note, that
			// gas_limit and gas_used are always > 0
			gasLimit := math.LegacyNewDec(int64(txGas))                           // #nosec G115
			gasRatio, err := gasLimit.QuoInt64(int64(response.GasUsed)).Float64() // #nosec G115
			if err == nil {
				telemetry.SetGaugeWithLabels(
					[]string{"tx", "msg", "ethereum_tx", "gas_limit", "per", "gas_used"},
					float32(gasRatio),
					labels,
				)
			}
		}
	}()

	attrs := []sdk.Attribute{
		sdk.NewAttribute(sdk.AttributeKeyAmount, txValue.String()),
		// add event for ethereum transaction hash format
		sdk.NewAttribute(types.AttributeKeyEthereumTxHash, response.Hash),
		// add event for index of valid ethereum tx
		sdk.NewAttribute(types.AttributeKeyTxIndex, strconv.FormatUint(txIndex, 10)),
		// add event for eth tx gas used, we can't get it from cosmos tx result when it contains multiple eth tx msgs.
		sdk.NewAttribute(types.AttributeKeyTxGasUsed, strconv.FormatUint(response.GasUsed, 10)),
	}

	if len(ctx.TxBytes()) > 0 {
		// add event for tendermint transaction hash format
		hash := tmbytes.HexBytes(tmtypes.Tx(ctx.TxBytes()).Hash())
		attrs = append(attrs, sdk.NewAttribute(types.AttributeKeyTxHash, hash.String()))
	}

	if txTo != nil {
		attrs = append(attrs, sdk.NewAttribute(types.AttributeKeyRecipient, txTo.Hex()))
	}

	if response.Failed() {
		attrs = append(attrs, sdk.NewAttribute(types.AttributeKeyEthereumTxFailed, response.VmError))
	}

	txLogAttrs := make([]sdk.Attribute, len(response.Logs))
	for i, log := range response.Logs {
		value, err := json.Marshal(log)
		if err != nil {
			return errorsmod.Wrap(err, "failed to encode log")
		}
		txLogAttrs[i] = sdk.NewAttribute(types.AttributeKeyTxLog, string(value))
	}

	// emit events
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeEthereumTx,
			attrs...,
		),
		sdk.NewEvent(
			types.EventTypeTxLog,
			txLogAttrs...,
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
			sdk.NewAttribute(sdk.AttributeKeySender, sender),
			sdk.NewAttribute(types.AttributeKeyTxType, fmt.Sprintf("%d", txType)),
		),
	})

	return nil
}

// UpdateParams implements the gRPC MsgServer interface. When an UpdateParams
// proposal passes, it updates the module parameters. The update can only be
// performed if the requested authority is the Cosmos SDK governance module
// account.
func (k *Keeper) UpdateParams(goCtx context.Context, req *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if k.authority.String() != req.Authority {
		return nil, errorsmod.Wrapf(govtypes.ErrInvalidSigner, "invalid authority, expected %s, got %s", k.authority.String(), req.Authority)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := k.SetParams(ctx, req.Params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}

// CallContract implements the gRPC MsgServer interface.
// It receives a request to call a contract and applies it to the EVM.
func (k *Keeper) CallContract(
	goCtx context.Context,
	req *imuachainevmtypes.MsgCallContract,
) (response *types.MsgEthereumTxResponse, retErr error) {
	if k.authority.String() != req.Authority {
		return nil, errorsmod.Wrapf(
			govtypes.ErrInvalidSigner,
			"invalid authority, expected %s, got %s",
			k.authority.String(), req.Authority,
		)
	}

	// use the same logic as k.EthereumTx(); do not cache the context
	ctx := sdk.UnwrapSDKContext(goCtx)
	nonce := k.GetNonce(ctx, common.BytesToAddress(k.authority.Bytes()))
	if nonce > 0 {
		// nonce is already incremented by the AnteHandler when execution reaches here.
		// to get the nonce of this transaction, we need to decrement it.
		nonce--
	} else {
		// a value of 0 is not possible since it is already incremented.
		return nil, errorsmod.Wrapf(
			errortypes.ErrInvalidRequest, "nonce is 0",
		)
	}

	// since our message has gas fee cap and gas tip cap, we use dynamic fee tx type
	txType := uint8(ethtypes.DynamicFeeTxType)
	labels := []metrics.Label{
		telemetry.NewLabel("tx_type", fmt.Sprintf("%d", txType)),
	}
	ctx = ctx.WithGasMeter(sdk.NewInfiniteGasMeter())
	var to *common.Address
	if req.To != "" {
		// caller must provide a valid contract address
		if !common.IsHexAddress(req.To) {
			return nil, errorsmod.Wrapf(
				errortypes.ErrInvalidRequest, "invalid contract address",
			)
		}
		addr := common.HexToAddress(req.To)
		to = &addr
		labels = append(labels, telemetry.NewLabel("execution", "call"))
	} else {
		// or it can be nil to indicate contract creation
		to = nil
		labels = append(labels, telemetry.NewLabel("execution", "create"))
	}
	data := req.Data

	gasLimit := uint64(0)
	if req.GasLimit != 0 {
		gasLimit = req.GasLimit
	} else {
		// if caller forgets to provide a gas limit, use the block max gas limit
		// we can do this safely since this function is only called by the authority
		params := ctx.ConsensusParams()
		if params != nil && params.Block != nil && params.Block.MaxGas > 0 {
			gasLimit = uint64(params.Block.MaxGas)
		}
	}
	if gasLimit == 0 {
		// if caller didn't provide a limit and a system-wide value is not set,
		// quit.
		return nil, errorsmod.Wrapf(
			errortypes.ErrInvalidRequest, "gas limit is 0",
		)
	}
	txIndex := k.GetTxIndexTransient(ctx)
	sender := common.BytesToAddress(k.authority.Bytes())
	value := big.NewInt(0)
	if req.Amount != nil {
		// internally this takes care of req.Amount.IsNil()
		value = req.Amount.BigInt()
	}
	msg := ethtypes.NewMessage(
		sender,
		to,
		nonce,
		value,
		gasLimit,
		// the gas prices are EVM constructs and this is a Cosmos execution
		// so they are not relevant.
		big.NewInt(0), // gas price
		big.NewInt(0), // gas fee cap
		big.NewInt(0), // gas tip cap
		data,
		nil,
		false,
	)
	response, retErr = k.ApplyMessage(ctx, msg, types.NewNoOpTracer(), true)
	if retErr != nil {
		return nil, retErr
	}
	retErr = k.postProcessResponse(
		ctx, labels, response,
		sender.Hex(), txIndex, msg.Gas(), msg.Value(), msg.To(), txType,
	)
	if retErr != nil {
		// leave it unchanged since callee has already wrapped it
		return nil, retErr
	}
	return response, nil
}
