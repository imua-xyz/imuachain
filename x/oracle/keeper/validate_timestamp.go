package keeper

import (
	"context"
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/imua-xyz/imuachain/x/oracle/types"
)

func checkTimestamp(goCtx context.Context, msg *types.MsgCreatePrice) error {
	ctx := sdk.UnwrapSDKContext(goCtx)
	now := ctx.BlockTime().UTC()
	for _, ps := range msg.Prices {
		for _, price := range ps.Prices {
			ts := price.Timestamp
			if len(ts) == 0 {
				return fmt.Errorf("timestamp should not be empty, blockTime:%s, got:%s", now.Format(layout), ts)
			}
			t, err := time.ParseInLocation(layout, ts, time.UTC)
			if err != nil {
				return fmt.Errorf("timestamp format invalid, blockTime:%s, got:%s", now.Format(layout), ts)
			}
			if now.Add(maxFutureOffset).Before(t) {
				return fmt.Errorf("timestamp is in the future, blockTime:%s, got:%s", now.Format(layout), ts)
			}
		}
	}
	return nil
}
