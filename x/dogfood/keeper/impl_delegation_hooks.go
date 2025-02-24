package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	keytypes "github.com/imua-xyz/imuachain/types/keys"
	avstypes "github.com/imua-xyz/imuachain/x/avs/types"
	delegationtypes "github.com/imua-xyz/imuachain/x/delegation/types"
)

// DelegationHooksWrapper is the wrapper structure that implements the delegation hooks for the
// dogfood keeper.
type DelegationHooksWrapper struct {
	keeper *Keeper
}

// Interface guard
var _ delegationtypes.DelegationHooks = DelegationHooksWrapper{}

// DelegationHooks returns the delegation hooks wrapper. It follows the "accept interfaces,
// return concretes" pattern.
func (k *Keeper) DelegationHooks() DelegationHooksWrapper {
	return DelegationHooksWrapper{k}
}

// AfterDelegation is called after a delegation is made.
func (wrapper DelegationHooksWrapper) AfterDelegation(
	sdk.Context, sdk.AccAddress,
) {
	// we do nothing here, since the vote power for all operators is calculated
	// in the end separately. even if we knew the amount of the delegation, the
	// exchange rate at the end of the epoch is unknown.
}

// AfterUndelegationStarted is called after an undelegation is started.
func (wrapper DelegationHooksWrapper) AfterUndelegationStarted(
	ctx sdk.Context, operator sdk.AccAddress, recordKey []byte,
) error {
	chainIDWithoutRevision := avstypes.ChainIDWithoutRevision(ctx.ChainID())
	var unbondingCompletionEpoch int64
	if wrapper.keeper.operatorKeeper.IsOperatorRemovingKeyFromChainID(
		ctx, operator, chainIDWithoutRevision,
	) {
		// if the operator is opting out, we need to use the finish epoch of the opt out.
		unbondingCompletionEpoch = wrapper.keeper.GetOperatorOptOutFinishEpoch(ctx, operator)
		// even if the operator opts back in, the undelegated vote power does not reappear
		// in the picture. slashable events between undelegation and opt in cannot occur
		// because the operator is not in the validator set.
	} else {
		var found bool
		var wrappedKey keytypes.WrappedConsKey
		if found, wrappedKey, _ = wrapper.keeper.operatorKeeper.GetOperatorConsKeyForChainID(
			ctx, operator, chainIDWithoutRevision,
		); !found {
			wrapper.keeper.Logger(ctx).Debug(
				"AfterUndelegationStarted: operator is not opted in; ignoring",
				"operator", operator,
				"recordKey", fmt.Sprintf("%x", recordKey),
			)
			// if the operator has no key set, they are not opted in to this AVS. hence,
			// we do not need to track the undelegation.
			return nil
		}
		// check if the key is active yet
		isValidator := false
		_, isValidator = wrapper.keeper.GetImuachainValidator(
			ctx, wrappedKey.ToConsAddr(),
		)
		if !isValidator {
			// maybe they changed the key. check the previous key.
			hasOldKey, prevKey, _ := wrapper.keeper.operatorKeeper.GetOperatorPrevConsKeyForChainID(
				ctx, operator, chainIDWithoutRevision,
			)
			if hasOldKey {
				_, isValidator = wrapper.keeper.GetImuachainValidator(
					ctx, prevKey.ToConsAddr(),
				)
			}
		}
		if !isValidator {
			wrapper.keeper.Logger(ctx).Debug(
				"AfterUndelegationStarted: operator not yet a validator; ignoring",
				"operator", operator,
				"recordKey", fmt.Sprintf("%x", recordKey),
			)
			// if the key is not active, we do not need to track the undelegation.
			// this can happen, for example, if the operator just opted into
			// the AVS and the epoch hasn't yet ended.
			return nil
		}
		// otherwise, we use the default unbonding completion epoch.
		unbondingCompletionEpoch = wrapper.keeper.GetUnbondingCompletionEpoch(ctx)
		// if the operator opts out after this, the undelegation will mature before the opt out.
		// so this is not a concern.
	}
	wrapper.keeper.AppendUndelegationToMature(ctx, unbondingCompletionEpoch, recordKey)
	wrapper.keeper.SetUndelegationMaturityEpoch(ctx, recordKey, unbondingCompletionEpoch)
	return wrapper.keeper.delegationKeeper.IncrementUndelegationHoldCount(ctx, recordKey)
}
