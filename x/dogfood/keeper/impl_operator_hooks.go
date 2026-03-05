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

// afterValidatorCreated is a helper function to call the hook in a cached context.
// the hook is primarily handled in the SDK's x/slashing, which simply stores a
// lookup of consensus address to public key.
func (h OperatorHooksWrapper) afterValidatorCreated(
	ctx sdk.Context, accAddress sdk.AccAddress,
) {
	cc, writeFunc := ctx.CacheContext()
	if err := h.keeper.Hooks().AfterValidatorCreated(
		cc, sdk.ValAddress(accAddress),
	); err != nil {
		h.keeper.Logger(ctx).Error("error in AfterValidatorCreated", "error", err)
	} else {
		writeFunc()
	}
}

// AfterOperatorKeySet is the implementation of the operator hooks.
// CONTRACT: an operator cannot set their key if they are already in the process of removing it.
func (h OperatorHooksWrapper) AfterOperatorKeySet(
	ctx sdk.Context, accAddress sdk.AccAddress, chainID string, _ keytypes.WrappedConsKey,
) {
	// we batch vote power changes at the end of the epoch, so nothing to do with those.
	// we should, however, let the x/slashing module know of this change.
	if chainID == utils.ChainIDWithoutRevision(ctx.ChainID()) {
		h.afterValidatorCreated(ctx, accAddress)
	}
}

// AfterOperatorKeyReplaced is the implementation of the operator hooks.
// CONTRACT: key replacement is not allowed if the operator is in the process of removing their
// key.
// CONTRACT: key replacement from newKey to oldKey is not allowed, after a replacement from
// oldKey to newKey.
func (h OperatorHooksWrapper) AfterOperatorKeyReplaced(
	ctx sdk.Context, accAddress sdk.AccAddress, oldKey keytypes.WrappedConsKey,
	newKey keytypes.WrappedConsKey, chainID string,
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
		oldConsAddr := oldKey.ToConsAddr()
		newConsAddr := newKey.ToConsAddr()

		// Map standard cosmos hooks (call AfterValidatorCreated to register the new key).
		// This creates a blank ValidatorSigningInfo internally.
		h.afterValidatorCreated(ctx, accAddress)
		// copy from old cons addr to new cons addr
		h.keeper.CopyValidatorSigningInfo(ctx, oldConsAddr, newConsAddr)
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
			// this should never happen because a consensus key must be set for this operator
			// and avs combination for it to show up in the operator + affected AVS list
			if !found || err != nil {
				h.keeper.Logger(ctx).Error(
					"could not find consensus key for operator",
					"operatorAddr", operator,
					"chainIDWithoutRevision", chainIDWithoutRevision,
					"err", err,
				)
				return
			}
			// if it is an unjail request, we update the validator set at the end of the block
			// TODO: unjailing should be performed only at the epoch end or when there is
			// another jailing
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
