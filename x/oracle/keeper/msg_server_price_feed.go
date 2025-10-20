package keeper

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/imua-xyz/imuachain/x/oracle/types"
)

const (
	layout          = "2006-01-02 15:04:05"
	maxFutureOffset = 30 * time.Second
	maxPriceLength  = 32
)

// CreatePrice proposes price for new round of specific tokenFeeder
func (ms msgServer) CreatePrice(goCtx context.Context, msg *types.MsgCreatePrice) (*types.MsgCreatePriceResponse, error) {
	start := time.Now()
	ctx := sdk.UnwrapSDKContext(goCtx)

	gasMeter := ctx.GasMeter()
	ctx = ctx.WithGasMeter(sdk.NewInfiniteGasMeter())
	defer func() {
		if !ctx.IsCheckTx() {
			ms.addTotald(time.Since(start))
			ms.increaseCount()
		}
		ctx = ctx.WithGasMeter(gasMeter)
	}()

	logger := ms.Logger(ctx)

	validator, _ := types.ConsAddrStrFromCreator(msg.Creator)
	logQuote := []any{"feederID", msg.FeederID, "baseBlock", msg.BasedBlock, "proposer", validator, "msg-nonce", msg.Nonce, "height", ctx.BlockHeight()}

	if err := checkTimestamp(ctx, msg); err != nil {
		logger.Error("quote has invalid timestamp", append(logQuote, "error", err)...)
		return nil, types.ErrPriceProposalFormatInvalid.Wrap(err.Error())
	}

	// goto rawData process which needs no 'aggragation', we just verify the provided piece with recorded root which got consensus
	if msg.IsPhaseTwo() {
		cachedIndex, cachedRawData, err := ms.ProcessRawData(ctx, msg, ctx.IsCheckTx())
		if err == nil {
			logger.Info("quote of 2nd-phase added rawData piece", append(logQuote, "rootHash", hex.EncodeToString(cachedRawData), "piece-index", msg.Prices[0].Prices[0].DetID)...)
			ctx.EventManager().EmitEvent(sdk.NewEvent(
				types.EventTypeCreatePrice,
				sdk.NewAttribute(types.AttributeKeyNSTPieceUpdate, types.AttributeValueTrue),
				sdk.NewAttribute(types.AttributeKeyNSTPieceChange, fmt.Sprintf("%d_%d", msg.FeederID, cachedIndex)),
			))
			return &types.MsgCreatePriceResponse{}, nil
		}
		logger.Error("quote of 2nd-phase for rawData piece failed", append(logQuote, "error", err)...)
		return nil, err
	}

	// core logic and functionality of Price Aggregation for 1st phase including
	// - price data
	// - hash for big data
	finalPrice, err := ms.ProcessQuote(ctx, msg, ctx.IsCheckTx())
	if err != nil {
		if sdkerrors.IsOf(err, types.ErrQuoteRecorded) {
			// quote is recorded only, this happens when a quoting-window is not availalbe before that window end due to final price aggregated successfully in advance
			// we will still record this msg if it's valid
			logger.Info("recorded quote for oracle-behavior evaluation", append(logQuote, "msg", msg)...)
			return &types.MsgCreatePriceResponse{}, nil
		}
		logger.Error("failed to process quote", append(logQuote, "error", err)...)
		return nil, err
	}

	logger.Info("added quote for aggregation", append(logQuote, "msg", msg, "isCheckTx", ctx.IsCheckTx())...)
	// TODO: use another type
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeCreatePrice,
		sdk.NewAttribute(types.AttributeKeyFeederID, strconv.FormatUint(msg.FeederID, 10)),
		sdk.NewAttribute(types.AttributeKeyBasedBlock, strconv.FormatUint(msg.BasedBlock, 10)),
		sdk.NewAttribute(types.AttributeKeyProposer, validator),
	))

	if finalPrice != nil {
		logger.Info("final price successfully aggregated", "price", finalPrice, "feederID", msg.FeederID, "height", ctx.BlockHeight())
		decimalStr := strconv.FormatInt(int64(finalPrice.Decimal), 10)
		// #nosec G115
		tokenID, _ := ms.GetTokenIDForFeederID(int64(msg.FeederID))
		tokenIDStr := strconv.FormatInt(tokenID, 10)
		roundIDStr := strconv.FormatUint(finalPrice.RoundID, 10)
		priceStr := finalPrice.Price

		if msg.IsPhaseOne() {
			// fot two-phases aggregation, the price represents for the rootHash of merkleTree derived from rawData, we need to encode it to base64 for identity transfer via websocket
			priceStr = base64.StdEncoding.EncodeToString([]byte(priceStr))
			// two-phases must from deterministic source, so we can append the detID to roundIDf for events, and finalPrice is finalized on current price, so its deteministic is correct
			roundIDStr = fmt.Sprintf("%s|%s", roundIDStr, msg.Prices[0].Prices[0].DetID)
		}

		// emit event to tell price is updated for current round of corresponding feederID
		ctx.EventManager().EmitEvent(sdk.NewEvent(
			types.EventTypeCreatePrice,
			sdk.NewAttribute(types.AttributeKeyRoundID, roundIDStr),
			sdk.NewAttribute(types.AttributeKeyFinalPrice, strings.Join([]string{tokenIDStr, roundIDStr, priceStr, decimalStr}, "_")),
			sdk.NewAttribute(types.AttributeKeyPriceUpdated, types.AttributeValueTrue)),
		)
	}

	return &types.MsgCreatePriceResponse{}, nil
}
