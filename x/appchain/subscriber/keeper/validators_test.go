package keeper_test

import (
	"testing"

	"github.com/ExocoreNetwork/exocore/testutil/keeper"
	testutiltx "github.com/ExocoreNetwork/exocore/testutil/tx"
	commontypes "github.com/ExocoreNetwork/exocore/x/appchain/common/types"
	"github.com/stretchr/testify/require"
)

func TestKeeper_ValsetUpdateID(t *testing.T) {
	k, ctx, _ := keeper.NewSubscriberKeeper(t)

	t.Run("Set and Get ValsetUpdateID", func(t *testing.T) {
		height := int64(100)
		expectedID := uint64(12345)

		k.SetValsetUpdateIDForHeight(ctx, height, expectedID)
		actualID := k.GetValsetUpdateIDForHeight(ctx, height)

		require.Equal(t, expectedID, actualID)
	})

	t.Run("Get non-existent ValsetUpdateID", func(t *testing.T) {
		height := int64(200)
		actualID := k.GetValsetUpdateIDForHeight(ctx, height)

		require.Equal(t, uint64(0), actualID)
	})
}

func TestKeeper_SubscriberValidator(t *testing.T) {
	k, ctx, _ := keeper.NewSubscriberKeeper(t)

	t.Run("Set, Get, and Delete SubscriberValidator", func(t *testing.T) {
		consAddr := testutiltx.GenerateConsAddress()
		pubKey := testutiltx.GenerateConsensusKey()
		power := int64(1000)

		validator, err := commontypes.NewSubscriberValidator(consAddr, power, pubKey.ToSdkKey())
		require.NoError(t, err)

		// Set validator
		k.SetSubscriberValidator(ctx, validator)

		// Get validator
		gotValidator, found := k.GetSubscriberValidator(ctx, consAddr)
		require.True(t, found)
		require.Equal(t, validator, gotValidator)

		// Delete validator
		k.DeleteSubscriberValidator(ctx, consAddr)

		// Try to get deleted validator
		_, found = k.GetSubscriberValidator(ctx, consAddr)
		require.False(t, found)
	})

	t.Run("Delete non-existent SubscriberValidator", func(t *testing.T) {
		consAddr := testutiltx.GenerateConsAddress()

		// This should not panic
		k.DeleteSubscriberValidator(ctx, consAddr)
	})
}

func TestKeeper_GetAllSubscriberValidators(t *testing.T) {
	k, ctx, _ := keeper.NewSubscriberKeeper(t)

	t.Run("Get all SubscriberValidators", func(t *testing.T) {
		// Create and set multiple validators
		validators := []commontypes.SubscriberValidator{}
		for i := 0; i < 5; i++ {
			consAddr := testutiltx.GenerateConsAddress()
			pubKey := testutiltx.GenerateConsensusKey()
			power := int64(1000 + i)

			validator, err := commontypes.NewSubscriberValidator(consAddr, power, pubKey.ToSdkKey())
			require.NoError(t, err)

			k.SetSubscriberValidator(ctx, validator)
			validators = append(validators, validator)
		}

		// Get all validators
		gotValidators := k.GetAllSubscriberValidators(ctx)

		require.Equal(t, len(validators), len(gotValidators))
		for _, validator := range validators {
			require.Contains(t, gotValidators, validator)
		}
	})
}
