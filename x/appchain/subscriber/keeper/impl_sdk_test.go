package keeper_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ExocoreNetwork/exocore/testutil/keeper"
)

func TestKeeper_GetParams(t *testing.T) {
	k, ctx, _ := keeper.NewSubscriberKeeper(t)

	params := k.GetParams(ctx)
	assert.Equal(t, stakingtypes.Params{}, params)
}

func TestKeeper_IterateValidators(t *testing.T) {
	k, ctx, _ := keeper.NewSubscriberKeeper(t)

	// This should be a no-op, so we're just checking that it doesn't panic
	k.IterateValidators(ctx, func(int64, stakingtypes.ValidatorI) bool {
		return false
	})
}

func TestKeeper_ValidatorByConsAddr(t *testing.T) {
	k, ctx, _ := keeper.NewSubscriberKeeper(t)

	consAddr := sdk.ConsAddress([]byte("test"))
	validator := k.ValidatorByConsAddr(ctx, consAddr)

	assert.NotNil(t, validator)
	assert.NotEqual(t, stakingtypes.Unbonded, validator.GetStatus())
}

func TestKeeper_SlashWithInfractionReason(t *testing.T) {
	k, ctx, _ := keeper.NewSubscriberKeeper(t)

	consAddr := sdk.ConsAddress([]byte("test"))
	infractionHeight := int64(100)
	power := int64(1000)
	slashFactor := sdk.NewDec(1)

	// Test with INFRACTION_UNSPECIFIED
	result := k.SlashWithInfractionReason(ctx, consAddr, infractionHeight, power, slashFactor, stakingtypes.Infraction_INFRACTION_UNSPECIFIED)
	assert.Equal(t, sdk.ZeroInt(), result)

	// Test with a valid infraction
	// Note: This test assumes that QueueSlashPacket is implemented correctly
	result = k.SlashWithInfractionReason(ctx, consAddr, infractionHeight, power, slashFactor, stakingtypes.Infraction_INFRACTION_DOUBLE_SIGN)
	assert.Equal(t, sdk.ZeroInt(), result)
}

func TestKeeper_MaxValidators(t *testing.T) {
	k, ctx, _ := keeper.NewSubscriberKeeper(t)

	maxValidators := k.MaxValidators(ctx)
	assert.Equal(t, uint32(0), maxValidators) // Assuming default is 0
}

func TestKeeper_GetAllValidators(t *testing.T) {
	k, ctx, _ := keeper.NewSubscriberKeeper(t)

	validators := k.GetAllValidators(ctx)
	assert.Empty(t, validators)
}

func TestKeeper_IsValidatorJailed(t *testing.T) {
	k, ctx, _ := keeper.NewSubscriberKeeper(t)

	consAddr := sdk.ConsAddress([]byte("test"))

	// Assuming HasOutstandingDowntime is implemented
	isJailed := k.IsValidatorJailed(ctx, consAddr)
	assert.False(t, isJailed) // Assuming default is false
}

func TestKeeper_UnbondingTime(t *testing.T) {
	k, ctx, _ := keeper.NewSubscriberKeeper(t)

	// Assuming GetUnbondingPeriod is implemented
	unbondingTime := k.UnbondingTime(ctx)
	assert.Equal(t, time.Duration(0), unbondingTime) // Assuming default is 0
}

func TestKeeper_ApplyAndReturnValidatorSetUpdates(t *testing.T) {
	k, ctx, _ := keeper.NewSubscriberKeeper(t)

	updates, err := k.ApplyAndReturnValidatorSetUpdates(ctx)
	require.NoError(t, err)
	assert.Empty(t, updates)
}

// TestKeeper_Panics tests the methods that are expected to panic
func TestKeeper_Panics(t *testing.T) {
	k, ctx, _ := keeper.NewSubscriberKeeper(t)

	assert.Panics(t, func() {
		k.Validator(ctx, sdk.ValAddress{})
	})

	assert.Panics(t, func() {
		k.Delegation(ctx, sdk.AccAddress{}, sdk.ValAddress{})
	})
}
