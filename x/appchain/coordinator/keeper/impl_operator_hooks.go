package keeper

import (
	exocoretypes "github.com/ExocoreNetwork/exocore/types/keys"
	"github.com/ExocoreNetwork/exocore/utils"
	operatortypes "github.com/ExocoreNetwork/exocore/x/operator/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// OperatorHooksWrapper is the wrapper structure that implements the operator hooks for the
// coordinator keeper.
type OperatorHooksWrapper struct {
	keeper *Keeper
}

// Interface guards
var _ operatortypes.OperatorHooks = OperatorHooksWrapper{}

func (k *Keeper) OperatorHooks() OperatorHooksWrapper {
	return OperatorHooksWrapper{k}
}

// AfterOperatorKeySet is the implementation of the operator hooks.
func (h OperatorHooksWrapper) AfterOperatorKeySet(
	sdk.Context, sdk.AccAddress, string, exocoretypes.WrappedConsKey,
) {
	// no-op
}

// AfterOperatorKeyReplaced is the implementation of the operator hooks.
func (h OperatorHooksWrapper) AfterOperatorKeyReplaced(
	ctx sdk.Context, _ sdk.AccAddress,
	oldKey exocoretypes.WrappedConsKey, _ exocoretypes.WrappedConsKey,
	chainID string,
) {
	if chainID != utils.ChainIDWithoutRevision(ctx.ChainID()) {
		consAddr := oldKey.ToConsAddr()
		_, found := h.keeper.GetSubscriberValidatorForChain(ctx, chainID, consAddr)
		if found {
			// schedule this consensus address for pruning at the maturity of the packet containing this vscID that will
			// go out at the end of this epoch.
			nextVscID := h.keeper.GetVscIDForChain(ctx, chainID) + 1
			h.keeper.AppendConsAddrToPrune(ctx, chainID, nextVscID, consAddr)
			// reverse lookup
			h.keeper.SetMaturityVscIDForChainIDConsAddr(ctx, chainID, consAddr, nextVscID)
		} else {
			// delete the reverse lookup of old cons addr + chain id -> operator addr, since it was never an active
			// validator.
			h.keeper.operatorKeeper.DeleteOperatorAddressForChainIDAndConsAddr(
				ctx, chainID, consAddr,
			)
		}
	}
}

// AfterOperatorKeyRemovalInitiated is the implementation of the operator hooks.
func (h OperatorHooksWrapper) AfterOperatorKeyRemovalInitiated(
	ctx sdk.Context, _ sdk.AccAddress, chainID string, key exocoretypes.WrappedConsKey,
) {
	if chainID != utils.ChainIDWithoutRevision(ctx.ChainID()) {
		consAddr := key.ToConsAddr()
		_, found := h.keeper.GetSubscriberValidatorForChain(ctx, chainID, consAddr)
		if found {
			// schedule this consensus address for pruning at the maturity of the packet containing this vscID that will
			// go out at the end of this epoch.
			nextVscID := h.keeper.GetVscIDForChain(ctx, chainID) + 1
			h.keeper.AppendConsAddrToPrune(ctx, chainID, nextVscID, consAddr)
			// reverse lookup
			h.keeper.SetMaturityVscIDForChainIDConsAddr(ctx, chainID, consAddr, nextVscID)
		} else {
			// delete the reverse lookup of old cons addr + chain id -> operator addr, since it was never an active
			// validator.
			h.keeper.operatorKeeper.DeleteOperatorAddressForChainIDAndConsAddr(
				ctx, chainID, consAddr,
			)
		}
	}
}
