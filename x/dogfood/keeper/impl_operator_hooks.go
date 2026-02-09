package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	keytypes "github.com/imua-xyz/imuachain/types/keys"
	"github.com/imua-xyz/imuachain/utils"
	operatortypes "github.com/imua-xyz/imuachain/x/operator/types"
)

// OperatorHooksWrapper is the wrapper structure that implements the operator hooks for the
// dogfood keeper.
type OperatorHooksWrapper struct {
	keeper *Keeper
}

// Interface guards
var _ operatortypes.OperatorHooks = OperatorHooksWrapper{}

func (k *Keeper) OperatorHooks() OperatorHooksWrapper {
	return OperatorHooksWrapper{k}
}

// AfterOperatorKeySet is the implementation of the operator hooks.
// CONTRACT: an operator cannot set their key if they are already in the process of removing it.
func (h OperatorHooksWrapper) AfterOperatorKeySet(
	sdk.Context, sdk.AccAddress, string, keytypes.WrappedConsKey,
) {
	// an operator opting in does not meaningfully affect this module, since
	// this information will be fetched at the end of the epoch
	// and the operator's vote power will be calculated then.
}

// AfterOperatorKeyReplaced is the implementation of the operator hooks.
// CONTRACT: key replacement is not allowed if the operator is in the process of removing their
// key.
// CONTRACT: key replacement from newKey to oldKey is not allowed, after a replacement from
// oldKey to newKey.
func (h OperatorHooksWrapper) AfterOperatorKeyReplaced(
	ctx sdk.Context, _ sdk.AccAddress, oldKey keytypes.WrappedConsKey,
	_ keytypes.WrappedConsKey, chainID string,
) {
	// the impact of key replacement is:
	// 1. vote power of old key is 0, which happens automatically at epoch end in EndBlock. this
	// is because the key is in the previous set but not in the new one and our code will queue
	// a validator update of 0 for this.
	// 2. vote power of new key is calculated, which happens automatically at epoch end in
	// EndBlock.
	// 3. X epochs later, the reverse lookup of old cons addr + chain id -> operator addr
	// should be cleared.
	consAddr := oldKey.ToConsAddr()
	if chainID == utils.ChainIDWithoutRevision(ctx.ChainID()) {
		// The reverse lookup (consensus address -> operator address) must be maintained
		// during the unbonding period to allow slashing of validators who misbehaved while
		// they were active, even after they've been removed from the active validator set.
		// This is necessary even if the validator is currently jailed or inactive, as
		// evidence of misbehavior (e.g., double-signing) from past epochs can be reported
		// and processed during the unbonding period. Therefore, we always schedule the
		// pruning for the consensus address at the end of the unbonding completion epoch.
		unbondingEpoch := h.keeper.GetUnbondingCompletionEpoch(ctx)
		h.keeper.AppendConsensusAddrToPrune(ctx, unbondingEpoch, consAddr)
	}
}

// AfterOperatorKeyRemovalInitiated is the implementation of the operator hooks.
func (h OperatorHooksWrapper) AfterOperatorKeyRemovalInitiated(
	ctx sdk.Context, operator sdk.AccAddress, chainID string, _ keytypes.WrappedConsKey,
) {
	// the impact of key removal is:
	// 1. vote power of the operator is 0, which happens automatically at epoch end in EndBlock.
	// this is because GetActiveOperatorsForChainID filters operators who are removing their
	// keys from the chain.
	// 2. X epochs later, the removal is marked complete in the operator module.
	if chainID == utils.ChainIDWithoutRevision(ctx.ChainID()) {
		// see AfterOperatorKeyReplaced for the reasoning behind scheduling the opt out,
		// even if the operator may not be in the active validator set.
		h.keeper.ScheduleOperatorOptOut(ctx, operator)
	}
}

// AfterSlash is the implementation of the operator hooks.
func (h OperatorHooksWrapper) AfterSlash(
	ctx sdk.Context, operator sdk.AccAddress, _ sdk.Dec, affectedAVSList []string,
	_ []operatortypes.SlashAssetAmount, _ []operatortypes.SlashFromUnclaimedRewards,
) {
	h.afterStakingOrJailChange(ctx, operator, false, affectedAVSList)
}

// AfterJail is the implementation of the operator hooks.
func (h OperatorHooksWrapper) AfterJail(
	ctx sdk.Context, operator sdk.AccAddress, isUnjail bool, affectedAVSList []string,
) {
	h.afterStakingOrJailChange(ctx, operator, isUnjail, affectedAVSList)
}

// afterStakingOrJailChange is the helper function to handle the changes in slashing or jailing.
func (h OperatorHooksWrapper) afterStakingOrJailChange(
	ctx sdk.Context, operator sdk.AccAddress, isUnjail bool, affectedAVSList []string,
) {
	chainIDWithoutRevision := utils.ChainIDWithoutRevision(ctx.ChainID())
	dogfoodAVSAddr := utils.GenerateAVSAddress(chainIDWithoutRevision)
	for _, avs := range affectedAVSList {
		// if we are an affected AVS
		if avs == dogfoodAVSAddr {
			// check whether a consensus key exists for the operator
			found, wrappedKey, err := h.keeper.operatorKeeper.GetOperatorConsKeyForChainID(
				ctx, operator, chainIDWithoutRevision,
			)
			// ideally, this will not happen - except if the operator is opting out?
			if !found || err != nil {
				h.keeper.Logger(ctx).Error(
					"could not find consensus key for operator",
					"operatorAddr", operator,
					"chainIDWithoutRevision", chainIDWithoutRevision,
					"err", err,
				)
				return
			}
			// if it is an unjail request, we update the validator set at the end of the block.
			if isUnjail {
				h.keeper.MarkUpdateValidatorSetFlag(ctx)
				break
			}
			// if, however, it is a jail request, we only alter the validator set if said key
			// is in the current validator set.
			isValidator := false
			_, isValidator = h.keeper.GetImuachainValidator(
				ctx, wrappedKey.ToConsAddr(),
			)
			if isValidator {
				h.keeper.MarkUpdateValidatorSetFlag(ctx)
			}
			break
		}
	}
}
