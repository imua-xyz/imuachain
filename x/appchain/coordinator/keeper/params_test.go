package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	testkeeper "github.com/ExocoreNetwork/exocore/testutil/keeper"
	"github.com/ExocoreNetwork/exocore/x/appchain/coordinator/types"
	epochstypes "github.com/ExocoreNetwork/exocore/x/epochs/types"
)

func TestGetParams(t *testing.T) {
	k, ctx, _ := testkeeper.NewCoordinatorKeeper(t)
	params := types.DefaultParams()

	k.SetParams(ctx, params)

	require.EqualValues(t, params, k.GetParams(ctx))
}

func TestSetParams(t *testing.T) {
	k, ctx, _ := testkeeper.NewCoordinatorKeeper(t)

	// Create custom params
	customParams := types.Params{
		TemplateClient:         nil, // You may want to create a mock ClientState here
		TrustingPeriodFraction: "0.8",
		IBCTimeoutPeriod:       24 * time.Hour,
		InitTimeoutPeriod:      epochstypes.Epoch{EpochIdentifier: "day", EpochNumber: 1},
		VSCTimeoutPeriod:       epochstypes.Epoch{EpochIdentifier: "day", EpochNumber: 2},
	}

	// Set custom params
	k.SetParams(ctx, customParams)

	// Get params and verify they match the custom params
	retrievedParams := k.GetParams(ctx)
	require.EqualValues(t, customParams, retrievedParams)

	// Verify individual fields
	require.Equal(t, customParams.TrustingPeriodFraction, retrievedParams.TrustingPeriodFraction)
	require.Equal(t, customParams.IBCTimeoutPeriod, retrievedParams.IBCTimeoutPeriod)
	require.Equal(t, customParams.InitTimeoutPeriod, retrievedParams.InitTimeoutPeriod)
	require.Equal(t, customParams.VSCTimeoutPeriod, retrievedParams.VSCTimeoutPeriod)
}

func TestGetParams_Default(t *testing.T) {
	k, ctx, _ := testkeeper.NewCoordinatorKeeper(t)

	// Get params without setting them first
	params := k.GetParams(ctx)

	// Verify that default params are returned
	require.EqualValues(t, types.DefaultParams(), params)
}
