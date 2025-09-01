package keeper

import (
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	epochstypes "github.com/imua-xyz/imuachain/x/epochs/types"
)

// EpochsHooksWrapper is the wrapper structure that implements the epochs hooks for the oracle
// keeper.
type EpochsHooksWrapper struct {
	keeper *Keeper
}

// Interface guard
var _ epochstypes.EpochHooks = EpochsHooksWrapper{}

// EpochsHooks returns the epochs hooks wrapper.
func (k *Keeper) EpochsHooks() EpochsHooksWrapper {
	return EpochsHooksWrapper{k}
}

// no-op methods for the epochs hooks interface
func (wrapper EpochsHooksWrapper) BeforeEpochStart(_ sdk.Context, _ string, _ int64) {
}

func (wrapper EpochsHooksWrapper) AfterEpochEnd(ctx sdk.Context, epochIdentifier string, epochNumber int64) {
	params := wrapper.keeper.GetParams(ctx)
	expEpochID := params.EpochIdentifier
	if strings.Compare(epochIdentifier, expEpochID) != 0 {
		return
	}
	// #nosec G115
	height := uint64(ctx.BlockHeight())
	for idx, tf := range params.TokenFeeders {
		if idx == 0 {
			continue
		}
		if (tf.StartBaseBlock > 0 && height <= tf.StartBaseBlock) || (tf.EndBlock > 0 && height > tf.EndBlock) {
			// If the base block is set and the current height is less than equal the start base block,
			// we skip this feeder.
			continue
		}
		_, ok := wrapper.keeper.FeederManager.GetNSTChainIDFromFeederID(uint64(idx))
		if ok {
			continue
		}
		if _, err := wrapper.keeper.CalculateTWAP(ctx, tf.TokenID, epochNumber); err != nil {
			// If the TWAP calculation fails, we log the error and continue with the next feeder.
			wrapper.keeper.Logger(ctx).Error(
				"failed to calculate TWAP",
				"token_id", tf.TokenID,
				"epoch_number", epochNumber,
				"error", err,
			)
		}
		// Reset the accumulated price for the token feeder no matter if the TWAP was calculated successfully or not.
		wrapper.keeper.ResetAccumulatedPrice(ctx, tf.TokenID)
	}
}
