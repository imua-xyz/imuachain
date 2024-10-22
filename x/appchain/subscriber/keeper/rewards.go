package keeper

import (
	"strconv"

	"github.com/ExocoreNetwork/exocore/x/appchain/subscriber/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	transfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
)

// EndBlockSendRewards distributes the rewards minted / collected so far amongst the coordinator and the subscriber.
func (k Keeper) EndBlockSendRewards(ctx sdk.Context) {
	k.SplitRewardsInternally(ctx)
	if !k.ShouldSendRewardsToCoordinator(ctx) {
		return
	}
	// Try to send rewards to coordinator
	cachedCtx, writeCache := ctx.CacheContext()
	if err := k.SendRewardsToCoordinator(cachedCtx); err != nil {
		k.Logger(ctx).Error("attempt to sent rewards to coordinator failed", "error", err)
	} else {
		// The cached context is created with a new EventManager so we merge the event
		// into the original context
		ctx.EventManager().EmitEvents(cachedCtx.EventManager().Events())
		// write cache
		writeCache()
	}

	// Update LastRewardTransmissionHeight
	k.SetLastRewardTransmissionHeight(ctx, ctx.BlockHeight())
}

// DistributeRewardsInternally "distributes" the rewards within the subscriber chain by earmarking the rewards for
// the coordinator in a separate account.
func (k Keeper) SplitRewardsInternally(ctx sdk.Context) {
	// source address, the local fee pool
	subscriberFeePoolAddr := k.accountKeeper.GetModuleAccount(
		ctx, k.feeCollectorName,
	).GetAddress()
	// get all tokens in the fee pool - we distribute them all but transfer
	// only the reward denomination
	fpTokens := k.bankKeeper.GetAllBalances(ctx, subscriberFeePoolAddr)
	if fpTokens.Empty() {
		k.Logger(ctx).Error("no tokens in fee pool")
		return
	}
	// fraction
	frac, err := sdk.NewDecFromStr(k.GetSubscriberParams(ctx).SubscriberRedistributionFraction)
	if err != nil {
		// should not happen since we validated this in the params
		panic(err)
	}
	// multiply all tokens by fraction
	decFPTokens := sdk.NewDecCoinsFromCoins(fpTokens...)
	subsRedistrTokens, _ := decFPTokens.MulDec(frac).TruncateDecimal()
	if subsRedistrTokens.Empty() {
		k.Logger(ctx).Error("no tokens (fractional) to distribute")
		// we can safely return from here since nothing has been distributed so far
		// so there is nothing to revert
		return
	}
	// send them from the fee pool to queue for local distribution
	// TODO(mm): implement SubscriberRedistributeName local distribution logic
	// on what basis?
	err = k.bankKeeper.SendCoinsFromModuleToModule(ctx, k.feeCollectorName,
		types.SubscriberRedistributeName, subsRedistrTokens)
	if err != nil {
		// It is the common behavior in cosmos-sdk to panic if SendCoinsFromModuleToModule
		// returns error.
		panic(err)
	}
	// send the remaining tokens to the coordinator fee pool on the subscriber
	remainingTokens := fpTokens.Sub(subsRedistrTokens...)
	err = k.bankKeeper.SendCoinsFromModuleToModule(ctx, k.feeCollectorName,
		types.SubscriberToSendToCoordinatorName, remainingTokens)
	if err != nil {
		// It is the common behavior in cosmos-sdk to panic if SendCoinsFromModuleToModule
		// returns error.
		panic(err)
	}
}

// Check whether it's time to send rewards to coordinator
func (k Keeper) ShouldSendRewardsToCoordinator(ctx sdk.Context) bool {
	bpdt := k.GetSubscriberParams(ctx).BlocksPerDistributionTransmission
	curHeight := ctx.BlockHeight()
	ltbh := k.GetLastRewardTransmissionHeight(ctx)
	diff := curHeight - ltbh
	shouldSend := diff >= bpdt
	return shouldSend
}

// GetLastRewardTransmissionHeight returns the height of the last reward transmission
func (k Keeper) GetLastRewardTransmissionHeight(
	ctx sdk.Context,
) int64 {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.LastRewardTransmissionHeightKey())
	return int64(sdk.BigEndianToUint64(bz))
}

// SetLastRewardTransmissionHeight sets the height of the last reward transmission
func (k Keeper) SetLastRewardTransmissionHeight(
	ctx sdk.Context,
	height int64,
) {
	store := ctx.KVStore(k.storeKey)
	store.Set(
		types.LastRewardTransmissionHeightKey(),
		sdk.Uint64ToBigEndian(uint64(height)),
	)
}

// SendRewardsToCoordinator attempts to send to the coordinator (via IBC)
// all the block rewards allocated for the coordinator
func (k Keeper) SendRewardsToCoordinator(ctx sdk.Context) error {
	sourceChannelID := k.GetDistributionTransmissionChannel(ctx)
	transferChannel, found := k.channelKeeper.GetChannel(
		ctx, transfertypes.PortID, sourceChannelID,
	)
	if !found || transferChannel.State != channeltypes.OPEN {
		k.Logger(ctx).Error("WARNING: cannot send rewards to coordinator;",
			"transmission channel not in OPEN state", "channelID", sourceChannelID)
		return nil
	}
	// due to timing it may happen that the channel is in TRYOPEN state
	// on the counterparty, and in that case the transfer will fail.
	// this is mitigated by having a sufficiently large reward distribution time
	// and setting up the appchain-1 channel before the first distribution takes place.
	// for localnet, i am using a value of 10 which is a bit low, but the subsequent
	// distributions will be fine. another option is to create the channel immediately
	// after a distribution is queued. that way, the channel will be open by the time
	// of the next distribution.

	// get params for sending rewards
	params := k.GetSubscriberParams(ctx)
	toSendToCoordinatorAddr := k.accountKeeper.GetModuleAccount(ctx,
		types.SubscriberToSendToCoordinatorName).GetAddress() // sender address
	coordinatorAddr := params.CoordinatorFeePoolAddrStr // receiver address
	timeoutHeight := clienttypes.ZeroHeight()
	timeoutTimestamp := uint64(ctx.BlockTime().Add(params.IBCTimeoutPeriod).UnixNano())

	denom := params.RewardDenom
	balance := k.bankKeeper.GetBalance(ctx, toSendToCoordinatorAddr, denom)

	// if the balance is not zero,
	if !balance.IsZero() {
		packetTransfer := &transfertypes.MsgTransfer{
			SourcePort:       transfertypes.PortID,
			SourceChannel:    sourceChannelID,
			Token:            balance,
			Sender:           toSendToCoordinatorAddr.String(), // subscriber address to send from
			Receiver:         coordinatorAddr,                  // coordinator fee pool address to send to
			TimeoutHeight:    timeoutHeight,                    // timeout height disabled
			TimeoutTimestamp: timeoutTimestamp,
			Memo:             "subscriber chain rewards distribution",
		}
		// validate MsgTransfer before calling Transfer()
		err := packetTransfer.ValidateBasic()
		if err != nil {
			return err
		}
		_, err = k.ibcTransferKeeper.Transfer(ctx, packetTransfer)
		if err != nil {
			return err
		}
	} else {
		k.Logger(ctx).Error("cannot send rewards to coordinator",
			"balance is zero", "denom", denom)
		return nil
	}

	k.Logger(ctx).Error("sent block rewards to coordinator",
		"amount", balance.String(),
		"denom", denom,
	)
	currentHeight := ctx.BlockHeight()
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeFeeDistribution,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
			sdk.NewAttribute(
				types.AttributeDistributionCurrentHeight,
				strconv.Itoa(int(currentHeight)),
			),
			sdk.NewAttribute(
				types.AttributeDistributionNextHeight,
				strconv.Itoa(int(currentHeight+params.BlocksPerDistributionTransmission)),
			),
			sdk.NewAttribute(
				types.AttributeDistributionFraction,
				params.SubscriberRedistributionFraction,
			),
			sdk.NewAttribute(types.AttributeDistributionValue, balance.String()),
			sdk.NewAttribute(types.AttributeDistributionDenom, denom),
		),
	)

	return nil
}
