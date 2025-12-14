package keeper

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"

	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
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
	"github.com/ethereum/go-ethereum/core/vm"
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
			gasLimit := math.LegacyNewDec(int64(tx.Gas()))                        // #nosec G115
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
		sdk.NewAttribute(sdk.AttributeKeyAmount, tx.Value().String()),
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

	if to := tx.To(); to != nil {
		attrs = append(attrs, sdk.NewAttribute(types.AttributeKeyRecipient, to.Hex()))
	}

	if response.Failed() {
		attrs = append(attrs, sdk.NewAttribute(types.AttributeKeyEthereumTxFailed, response.VmError))
	}

	txLogAttrs := make([]sdk.Attribute, len(response.Logs))
	for i, log := range response.Logs {
		value, err := json.Marshal(log)
		if err != nil {
			return nil, errorsmod.Wrap(err, "failed to encode log")
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
			sdk.NewAttribute(types.AttributeKeyTxType, fmt.Sprintf("%d", tx.Type())),
		),
	})

	return response, nil
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
) (resp *imuachainevmtypes.MsgCallContractResponse, retErr error) {
	if k.authority.String() != req.Authority {
		return nil, errorsmod.Wrapf(
			govtypes.ErrInvalidSigner,
			"invalid authority, expected %s, got %s",
			k.authority.String(), req.Authority,
		)
	}

	staticCtx := sdk.UnwrapSDKContext(goCtx)
	ctx, writeFunc := staticCtx.CacheContext()
	defer func() {
		if retErr == nil {
			writeFunc()
		}
	}()
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
	ctx = ctx.WithGasMeter(sdk.NewInfiniteGasMeter())
	var contractAddress *common.Address
	if req.ContractAddress != "" {
		addr := common.HexToAddress(req.ContractAddress)
		contractAddress = &addr
	} else {
		contractAddress = nil
	}
	data := common.Hex2Bytes(req.Data)

	var gasLimit int64 = 30_000_000
	params := ctx.ConsensusParams()
	if params != nil && params.Block != nil && params.Block.MaxGas > 0 {
		gasLimit = params.Block.MaxGas
	}

	msg := ethtypes.NewMessage(
		common.BytesToAddress(k.authority.Bytes()),
		contractAddress,
		nonce,
		big.NewInt(0), // value
		uint64(gasLimit),
		big.NewInt(0), // gas price
		big.NewInt(0), // gas fee cap
		big.NewInt(0), // gas tip cap
		data,
		nil,
		false,
	)
	response, err := k.ApplyMessage(ctx, msg, types.NewNoOpTracer(), true)
	if err != nil {
		return nil, err
	}
	if response.Failed() {
		errStr := response.VmError
		if response.VmError == vm.ErrExecutionReverted.Error() {
			if cause, err := abi.UnpackRevert(common.CopyBytes(response.Ret)); err == nil {
				errStr = cause
			}
		}
		return nil, types.ErrVMExecution.Wrap(errStr)
	}
	return &imuachainevmtypes.MsgCallContractResponse{}, nil
}
