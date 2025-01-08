package keeper

import (
	"context"
	"errors"
	"time"

	"github.com/ExocoreNetwork/exocore/x/oracle/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

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
