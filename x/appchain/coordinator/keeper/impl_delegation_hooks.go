package keeper

import (
	"fmt"

	"github.com/ExocoreNetwork/exocore/utils"
	delegationtypes "github.com/ExocoreNetwork/exocore/x/delegation/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
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
	logger := wrapper.keeper.Logger(ctx)
	// given the operator, find the chainIDs for which they are (1) opted in, or (2) in the process of opting out.
	// (1) simply let the undelegation mature when it matures on the subscriber chain.
	// (2) the undelegation should mature when the operator's opt out matures on the subscriber chain.
	// within the undelegation situation, the previous keys don't matter, because
	// they will be replaced anyway. hence, we only need to check the current keys.
	chainIDs, err := wrapper.keeper.operatorKeeper.GetChainIDsForOperator(ctx, operator.String())
	if err != nil {
		logger.Error(
			"error getting chainIDs for operator",
			"operator", operator,
			"recordKey", fmt.Sprintf("%x", recordKey),
		)
		// do not return an error because that would indicate an undelegation failure.
		// TODO: verify the above claim and check the impact of actually returning the error
		return nil
	}
	// TODO: above only returns the chainIDs for which the operator is opted-in, but
	// not those for which the operator is in the process of opting out. this will be
	// resolved in the unbonding duration calculation pull request and hopefully,
	// meaningfully unified.
	for _, chainID := range chainIDs {
		if chainID != utils.ChainIDWithoutRevision(ctx.ChainID()) {
			found, wrappedKey, _ := wrapper.keeper.operatorKeeper.GetOperatorConsKeyForChainID(
				ctx, operator, chainID,
			)
			if !found {
				logger.Debug(
					"operator not opted in; ignoring",
					"operator", operator,
					"chainID", chainID,
				)
				continue
			}
			var nextVscID uint64
			if wrapper.keeper.operatorKeeper.IsOperatorRemovingKeyFromChainID(
				ctx, operator, chainID,
			) {
				nextVscID = wrapper.keeper.GetMaturityVscIDForChainIDConsAddr(ctx, chainID, wrappedKey.ToConsAddr())
				if nextVscID == 0 {
					logger.Error(
						"undelegation maturity epoch not set",
						"operator", operator,
						"chainID", chainID,
						"consAddr", wrappedKey.ToConsAddr(),
						"recordKey", fmt.Sprintf("%x", recordKey),
					)
					// move on to the next chainID
					continue
				}
			} else {
				nextVscID = wrapper.keeper.GetVscIDForChain(ctx, chainID) + 1
			}
			wrapper.keeper.AppendUndelegationToRelease(ctx, chainID, nextVscID, recordKey)
			// increment the count for each such chainID
			if err := wrapper.keeper.delegationKeeper.IncrementUndelegationHoldCount(ctx, recordKey); err != nil {
				return err
			}
		}
	}
	return nil
}
