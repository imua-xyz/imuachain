package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/imua-xyz/imuachain/testutil/keeper"
	"github.com/imua-xyz/imuachain/testutil/nullify"
	"github.com/imua-xyz/imuachain/x/oracle/keeper"
	"github.com/imua-xyz/imuachain/x/oracle/types"
)

func createTestValidatorUpdateBlock(keeper *keeper.Keeper, ctx sdk.Context) types.ValidatorUpdateBlock {
	item := types.ValidatorUpdateBlock{}
	keeper.SetValidatorUpdateForCache(ctx, item)
	return item
}

func TestValidatorUpdateBlockGet(t *testing.T) {
	keeper, ctx := keepertest.OracleKeeper(t)
	item := createTestValidatorUpdateBlock(keeper, ctx)
	rst, found := keeper.GetValidatorUpdateBlock(ctx)
	require.True(t, found)
	require.Equal(t,
		nullify.Fill(&item),
		nullify.Fill(&rst),
	)
}

func TestValidatorUpdateBlockRemove(t *testing.T) {
	keeper, ctx := keepertest.OracleKeeper(t)
	createTestValidatorUpdateBlock(keeper, ctx)
	keeper.RemoveValidatorUpdateBlock(ctx)
	_, found := keeper.GetValidatorUpdateBlock(ctx)
	require.False(t, found)
}
