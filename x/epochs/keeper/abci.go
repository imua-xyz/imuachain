package keeper

import (
	"strconv"
	"time"

	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/ExocoreNetwork/exocore/x/epochs/types"
)

// BeginBlocker is used to start or end the epochs, amongst the epochs currently
// in the store.
func (k Keeper) BeginBlocker(ctx sdk.Context) {
	defer telemetry.ModuleMeasureSince(types.ModuleName, time.Now(), telemetry.MetricKeyBeginBlocker)
	logger := k.Logger(ctx)
	k.IterateEpochInfos(
		ctx,
		func(_ int64, epochInfo types.EpochInfo) (stop bool) {
			// even though the epochInfo as validated in AddEpochInfo
			// and setEpochInfoUnchecked is private to this module,
			// we still validate it here, just in case.
			if err := epochInfo.Validate(); err != nil {
				logger.Error(
					"epoch info validation failed, skipping",
					"identifier", epochInfo.Identifier,
					"error", err,
				)
				return false
			}
			if ctx.BlockTime().Before(epochInfo.StartTime) {
				// short circuit if this epoch is not yet scheduled to start
				return false
			}
			epochEndTime := epochInfo.CurrentEpochStartTime.Add(epochInfo.Duration)
			// are we starting this identifier for the first time?
			isFirstTick := !epochInfo.EpochCountingStarted
			// is this the end of the current tick?
			isTickEnding := ctx.BlockTime().After(epochEndTime)
			// if either of those conditions are true, we will start an epoch
			isEpochStart := isTickEnding || isFirstTick

			if !isEpochStart {
				return false
			}
			// if we reach here, we are starting a new epoch. this means, we set its height.
			epochInfo.CurrentEpochStartHeight = ctx.BlockHeight()

			if isFirstTick {
				epochInfo.EpochCountingStarted = true
				// even if the genesis file may start at a different number, we will reset to 1.
				epochInfo.CurrentEpoch = 1
				// serialized to disk as t.Unix(), which is location independent,
				// even if the genesis file has `epochInfo.StartTime` in a different timezone.
				epochInfo.CurrentEpochStartTime = epochInfo.StartTime
				// we don't call BeforeEpochStart here because it is the first epoch.
				// similarly, we don't emit an ending event.
			} else {
				// if we are here, isTickEnding is true but isFirstTick is false.
				// in other words, epoch i is ending and epoch i+1 is starting.
				logger.Info(
					"ending epoch",
					"identifier", epochInfo.Identifier,
					"number", epochInfo.CurrentEpoch,
					"height", ctx.BlockHeight(),
				)
				ctx.EventManager().EmitEvent(
					sdk.NewEvent(
						types.EventTypeEpochEnd,
						sdk.NewAttribute(
							types.AttributeEpochIdentifier, epochInfo.Identifier,
						),
						sdk.NewAttribute(
							types.AttributeEpochNumber,
							strconv.FormatInt(
								epochInfo.CurrentEpoch, 10,
							),
						),
					),
				)
				// NOTE: this hook is called BEFORE the new epoch info is saved.
				k.Hooks().AfterEpochEnd(ctx, epochInfo.Identifier, epochInfo.CurrentEpoch)
				// now, we can increment the epoch number.
				epochInfo.CurrentEpoch++
				// and set the new start time.
				// (1) this time is serialized to disk as t.Unix(), which is location independent.
				// (2) epoch end time is CurrentEpochStartTime + Duration, of which the former
				//     is also similarly serialized to disk as t.Unix().
				//     if it is set in the genesis file with a different time zone, that is thus taken care of.
				//     if it is not provided, it is set to ctx.BlockTime(), which is UTC.
				// hence, we do not need to worry about timezones.
				epochInfo.CurrentEpochStartTime = epochEndTime
			}

			// now we are starting the i+1 epoch, that is the one currently set in epochInfo.
			k.setEpochInfoUnchecked(ctx, epochInfo)

			ctx.EventManager().EmitEvent(
				sdk.NewEvent(
					types.EventTypeEpochStart,
					sdk.NewAttribute(
						types.AttributeEpochIdentifier, epochInfo.Identifier,
					),
					sdk.NewAttribute(
						types.AttributeEpochNumber,
						strconv.FormatInt(
							epochInfo.CurrentEpoch, 10,
						),
					),
					sdk.NewAttribute(
						types.AttributeEpochStartTime,
						strconv.FormatInt(
							epochInfo.CurrentEpochStartTime.Unix(), 10,
						),
					),
				),
			)

			k.Hooks().BeforeEpochStart(ctx, epochInfo.Identifier, epochInfo.CurrentEpoch)

			return false
		},
	)
}
