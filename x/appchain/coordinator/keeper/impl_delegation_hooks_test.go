package keeper_test

import (
	"errors"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	testkeeper "github.com/ExocoreNetwork/exocore/testutil/keeper"
	testutiltx "github.com/ExocoreNetwork/exocore/testutil/tx"
	keepermod "github.com/ExocoreNetwork/exocore/x/appchain/coordinator/keeper"
)

func TestAfterDelegation(t *testing.T) {
	keeper, ctx, _ := testkeeper.NewCoordinatorKeeper(t)
	hooks := keeper.DelegationHooks()

	// AfterDelegation should do nothing, so we just ensure it doesn't panic
	require.NotPanics(t, func() {
		hooks.AfterDelegation(ctx, sdk.AccAddress{})
	})
}

func TestAfterUndelegationStarted(t *testing.T) {
	keeper, ctx, mocks := testkeeper.NewCoordinatorKeeper(t)
	hooks := keeper.DelegationHooks()

	operator := sdk.AccAddress("testoperator")
	recordKey := []byte("testrecordkey")
	chainIDs := []string{"chain1", "chain2"}

	// Setup mocks
	mocks.OperatorKeeper.EXPECT().GetChainIDsForOperator(ctx, operator.String()).Return(chainIDs, nil)

	for _, chainID := range chainIDs {
		wrappedKey := testutiltx.GenerateConsensusKey()
		mocks.OperatorKeeper.EXPECT().GetOperatorConsKeyForChainID(ctx, operator, chainID).
			Return(true, wrappedKey, nil)
		mocks.OperatorKeeper.EXPECT().IsOperatorRemovingKeyFromChainID(ctx, operator, chainID).
			Return(false)
		mocks.DelegationKeeper.EXPECT().IncrementUndelegationHoldCount(ctx, recordKey).Return(nil)
	}

	// Test AfterUndelegationStarted
	err := hooks.AfterUndelegationStarted(ctx, operator, recordKey)
	require.NoError(t, err)

	// Additional test case: operator removing key from chainID
	t.Run("Operator removing key", func(t *testing.T) {
		keeper, ctx, mocks := testkeeper.NewCoordinatorKeeper(t)
		hooks := keeper.DelegationHooks()

		chainID := "chain3"
		mocks.OperatorKeeper.EXPECT().GetChainIDsForOperator(ctx, operator.String()).Return([]string{chainID}, nil)
		wrappedKey := testutiltx.GenerateConsensusKey()
		mocks.OperatorKeeper.EXPECT().GetOperatorConsKeyForChainID(ctx, operator, chainID).
			Return(true, wrappedKey, nil)
		mocks.OperatorKeeper.EXPECT().IsOperatorRemovingKeyFromChainID(ctx, operator, chainID).
			Return(true)

		nextVscID := uint64(5)
		keeper.SetMaturityVscIDForChainIDConsAddr(ctx, chainID, wrappedKey.ToConsAddr(), nextVscID)

		mocks.DelegationKeeper.EXPECT().IncrementUndelegationHoldCount(ctx, recordKey).Return(nil)

		err := hooks.AfterUndelegationStarted(ctx, operator, recordKey)
		require.NoError(t, err)
	})

	// Test case: Error incrementing undelegation hold count
	t.Run("Error incrementing hold count", func(t *testing.T) {
		expErr := errors.New("error incrementing undelegation hold count")
		keeper, ctx, mocks := testkeeper.NewCoordinatorKeeper(t)
		hooks := keeper.DelegationHooks()

		mocks.OperatorKeeper.EXPECT().GetChainIDsForOperator(ctx, operator.String()).Return(chainIDs, nil)
		wrappedKey := testutiltx.GenerateConsensusKey()
		mocks.OperatorKeeper.EXPECT().GetOperatorConsKeyForChainID(ctx, operator, chainIDs[0]).
			Return(true, wrappedKey, nil)
		mocks.OperatorKeeper.EXPECT().IsOperatorRemovingKeyFromChainID(ctx, operator, chainIDs[0]).
			Return(false)
		mocks.DelegationKeeper.EXPECT().IncrementUndelegationHoldCount(ctx, recordKey).
			Return(expErr)

		err := hooks.AfterUndelegationStarted(ctx, operator, recordKey)
		require.Error(t, err)
		require.ErrorIs(t, err, expErr)
	})
}

func TestDelegationHooks(t *testing.T) {
	keeper, _, _ := testkeeper.NewCoordinatorKeeper(t)
	hooks := keeper.DelegationHooks()

	require.NotNil(t, hooks)
	require.IsType(t, keepermod.DelegationHooksWrapper{}, hooks)
}
