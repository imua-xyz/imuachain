package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	keytypes "github.com/imua-xyz/imuachain/types/keys"
	avstypes "github.com/imua-xyz/imuachain/x/avs/types"
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
	if chainID == avstypes.ChainIDWithoutRevision(ctx.ChainID()) {
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
	ctx sdk.Context, operator sdk.AccAddress, chainID string, key keytypes.WrappedConsKey,
) {
	// the impact of key removal is:
	// 1. vote power of the operator is 0, which happens automatically at epoch end in EndBlock.
	// this is because GetActiveOperatorsForChainID filters operators who are removing their
	// keys from the chain.
	// 2. X epochs later, the removal is marked complete in the operator module.
	if chainID == avstypes.ChainIDWithoutRevision(ctx.ChainID()) {
		// see AfterOperatorKeyReplaced for the reasoning behind scheduling the opt out,
		// even if the operator may not be in the active validator set.
		h.keeper.ScheduleOperatorOptOut(ctx, operator)
	}
}

func (h OperatorHooksWrapper) AfterSlash(
	ctx sdk.Context, operator sdk.AccAddress, _ sdk.Dec, affectedAVSList []string,
	_ []operatortypes.SlashFromAssetsPool,
) {
	h.afterStakingOrJailChange(ctx, operator, false, affectedAVSList)
}

func (h OperatorHooksWrapper) AfterJail(
	ctx sdk.Context, operator sdk.AccAddress, isUnjail bool, affectedAVSList []string,
) {
	h.afterStakingOrJailChange(ctx, operator, isUnjail, affectedAVSList)
}

func (h OperatorHooksWrapper) afterStakingOrJailChange(ctx sdk.Context, operator sdk.AccAddress, isUnjail bool, affectedAVSList []string) {
	chainIDWithoutRevision := avstypes.ChainIDWithoutRevision(ctx.ChainID())
	dogfoodAVSAddr := avstypes.GenerateAVSAddress(chainIDWithoutRevision)
	for _, avs := range affectedAVSList {
		if avs == dogfoodAVSAddr {
			found, wrappedKey, err := h.keeper.operatorKeeper.GetOperatorConsKeyForChainID(ctx, operator, chainIDWithoutRevision)
			if !found || err != nil {
				ctx.Logger().Error("AfterSlash the consensus key isn't found by the chainIDWithoutRevision and operator address", "operatorAddr", operator, "chainIDWithoutRevision", chainIDWithoutRevision, "err", err)
				return
			}
			if isUnjail {
				// mark the flag for unjail
				// the validator has been removed from the current active validator set when jailing,
				// so it shouldn't check if it is active when unjail.
				h.keeper.MarkUpdateValidatorSetFlag(ctx)
				break
			}

			// check if the operator is in the current validator set.
			// check if the key is active yet
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
