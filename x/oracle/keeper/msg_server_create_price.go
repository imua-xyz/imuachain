package keeper

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"strconv"
	"strings"
	"time"

	sdkerrors "cosmossdk.io/errors"
	"github.com/ExocoreNetwork/exocore/x/oracle/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	layout          = "2006-01-02 15:04:05"
	maxFutureOffset = 5 * time.Second
)

// CreatePrice proposes price for new round of specific tokenFeeder
func (ms msgServer) CreatePrice(goCtx context.Context, msg *types.MsgCreatePrice) (*types.MsgCreatePriceResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	gasMeter := ctx.GasMeter()
	ctx = ctx.WithGasMeter(sdk.NewInfiniteGasMeter())
	defer func() {
		ctx = ctx.WithGasMeter(gasMeter)
	}()

	logger := ms.Logger(ctx)

	validator, _ := types.ConsAddrStrFromCreator(msg.Creator)
	logQuote := []interface{}{"feederID", msg.FeederID, "baseBlock", msg.BasedBlock, "proposer", validator, "msg-nonce", msg.Nonce, "height", ctx.BlockHeight()}

	if err := checkTimestamp(ctx, msg); err != nil {
		logger.Error("quote has invalid timestamp", append(logQuote, "error", err)...)
		return nil, types.ErrPriceProposalFormatInvalid.Wrap(err.Error())
	}

	if err := ms.ValidateMsg(msg); err != nil {
		logger.Error("failed to validate msg", append(logQuote, "error", err)...)
		return nil, err
	}
	// core logic and functionality of Price Aggregation
	finalPrice, err := ms.ProcessQuote(ctx, msg, ctx.IsCheckTx())
	if err != nil {
		if sdkerrors.IsOf(err, types.ErrQuoteRecorded) {
			// quote is recorded only, this happens when a quoting-window is not availalbe before that window end due to final price aggregated successfully in advance
			// we will still record this msg if it's valid
			logger.Info("recorded quote for oracle-behavior evaluation", logQuote...)
			return &types.MsgCreatePriceResponse{}, nil
		}
		logger.Error("failed to process quote", append(logQuote, "error", err)...)
		return nil, err
	}

	logger.Info("added quote for aggregation", logQuote...)
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

		if len(priceStr) >= 32 {
			hash := sha256.New()
			hash.Write([]byte(priceStr))
			priceStr = base64.StdEncoding.EncodeToString(hash.Sum(nil))
		}

		// emit event to tell price is updated for current round of corresponding feederID
		ctx.EventManager().EmitEvent(sdk.NewEvent(
			types.EventTypeCreatePrice,
			sdk.NewAttribute(types.AttributeKeyRoundID, roundIDStr),
			sdk.NewAttribute(types.AttributeKeyFinalPrice, strings.Join([]string{tokenIDStr, roundIDStr, priceStr, decimalStr}, "_")),
			sdk.NewAttribute(types.AttributeKeyPriceUpdated, types.AttributeValuePriceUpdatedSuccess)),
		)
	}

	return &types.MsgCreatePriceResponse{}, nil
}
