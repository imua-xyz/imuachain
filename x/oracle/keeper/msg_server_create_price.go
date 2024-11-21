package keeper

import (
	"context"
	"errors"
	"strconv"
	"time"

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
	logger := ms.Keeper.Logger(ctx)
	if err := checkTimestamp(ctx, msg); err != nil {
		logger.Info("price proposal timestamp check failed", "error", err, "height", ctx.BlockHeight())
		return nil, types.ErrPriceProposalFormatInvalid.Wrap(err.Error())
	}

	agc := ms.Keeper.GetAggregatorContext(ctx)
	newItem, caches, err := agc.NewCreatePrice(ctx, msg)
	if err != nil {
		logger.Info("price proposal failed", "error", err, "height", ctx.BlockHeight(), "feederID", msg.FeederID)
		return nil, err
	}

	logger.Info("add price proposal for aggregation", "feederID", msg.FeederID, "basedBlock", msg.BasedBlock, "proposer", msg.Creator, "height", ctx.BlockHeight())

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeCreatePrice,
		sdk.NewAttribute(types.AttributeKeyFeederID, strconv.FormatUint(msg.FeederID, 10)),
		sdk.NewAttribute(types.AttributeKeyBasedBlock, strconv.FormatUint(msg.BasedBlock, 10)),
		sdk.NewAttribute(types.AttributeKeyProposer, msg.Creator),
	),
	)

	if caches == nil {
		return &types.MsgCreatePriceResponse{}, nil
	}
	if newItem != nil {
		if success := ms.AppendPriceTR(ctx, newItem.TokenID, newItem.PriceTR); !success {
			// This case should not exist, keep this line to avoid consensus fail if this happens
			prevPrice, nextRoundID := ms.GrowRoundID(ctx, newItem.TokenID)
			logger.Error("append new price round fail for mismatch roundID, and will just grow roundID with previous price", "roundID from finalPrice", newItem.PriceTR.RoundID, "expect nextRoundID", nextRoundID, "prevPrice", prevPrice)
		} else {
			logger.Info("final price aggregation done", "feederID", msg.FeederID, "roundID", newItem.PriceTR.RoundID, "price", newItem.PriceTR.Price)
		}

		decimalStr := strconv.FormatInt(int64(newItem.PriceTR.Decimal), 10)
		tokenIDStr := strconv.FormatUint(newItem.TokenID, 10)
		roundIDStr := strconv.FormatUint(newItem.PriceTR.RoundID, 10)
		ctx.EventManager().EmitEvent(sdk.NewEvent(
			types.EventTypeCreatePrice,
			sdk.NewAttribute(types.AttributeKeyRoundID, roundIDStr),
			sdk.NewAttribute(types.AttributeKeyFinalPrice, tokenIDStr+"_"+roundIDStr+"_"+newItem.PriceTR.Price+"_"+decimalStr),
			sdk.NewAttribute(types.AttributeKeyPriceUpdated, types.AttributeValuePriceUpdatedSuccess)),
		)
		if !ctx.IsCheckTx() {
			ms.Keeper.GetCaches().RemoveCache(caches)
		}
		ms.Keeper.AppendUpdatedFeederIDs(msg.FeederID)
	} else if !ctx.IsCheckTx() {
		ms.Keeper.GetCaches().AddCache(caches)
	}

	return &types.MsgCreatePriceResponse{}, nil
}

func checkTimestamp(goCtx context.Context, msg *types.MsgCreatePrice) error {
	ctx := sdk.UnwrapSDKContext(goCtx)
	now := ctx.BlockTime().UTC()
	for _, ps := range msg.Prices {
		for _, price := range ps.Prices {
			ts := price.Timestamp
			if len(ts) == 0 {
				return errors.New("timestamp should not be empty")
			}
			t, err := time.ParseInLocation(layout, ts, time.UTC)
			if err != nil {
				return errors.New("timestamp format invalid")
			}
			if now.Add(maxFutureOffset).Before(t) {
				return errors.New("timestamp is in the future")
			}
		}
	}
	return nil
}
