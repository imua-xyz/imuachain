package keeper_test

import (
	"strconv"
	"testing"

	sdkmath "cosmossdk.io/math"
	keepertest "github.com/ExocoreNetwork/exocore/testutil/keeper"
	"github.com/ExocoreNetwork/exocore/testutil/nullify"
	"github.com/ExocoreNetwork/exocore/x/oracle/keeper"
	"github.com/ExocoreNetwork/exocore/x/oracle/keeper/testdata"
	"github.com/ExocoreNetwork/exocore/x/oracle/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func createNPrices(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.Prices {
	items := make([]types.Prices, n)
	for i := range items {
		items[i].TokenID = uint64(i + 1)
		items[i] = types.Prices{
			TokenID:     uint64(i + 1),
			NextRoundID: 2,
			PriceList: []*types.PriceTimeRound{
				testdata.PTR1,
				testdata.PTR2,
				testdata.PTR3,
				testdata.PTR4,
				testdata.PTR5,
			},
		}
		keeper.SetPrices(ctx, items[i])
	}
	return items
}

func TestPricesGet(t *testing.T) {
	keeper, ctx := keepertest.OracleKeeper(t)
	keeper.SetPrices(ctx, testdata.P1)
	rst, found := keeper.GetPrices(ctx, 1)
	require.True(t, found)
	pRes := testdata.P1
	require.Equal(t, pRes, rst)
}

func TestPricesGetMultiAssets(t *testing.T) {
	keeper, ctx := keepertest.OracleKeeper(t)
	keeper.SetPrices(ctx, testdata.P1)
	assets := make(map[string]interface{})
	assets["0x0b34c4d876cd569129cf56bafabb3f9e97a4ff42_0x9ce1"] = new(interface{})
	prices, err := keeper.GetMultipleAssetsPrices(ctx, assets)
	expectedPrices := make(map[string]types.Price)
	v, _ := sdkmath.NewIntFromString(testdata.PTR5.Price)
	// v, _ := sdkmath.NewIntFromString(testdata.PTR5.Price)
	expectedPrices["0x0b34c4d876cd569129cf56bafabb3f9e97a4ff42_0x9ce1"] = types.Price{
		Value:   v,
		Decimal: uint8(testdata.PTR5.Decimal),
	}
	require.NoError(t, err)
	require.Equal(t, expectedPrices, prices)

	assets["unexistsAsset"] = new(interface{})
	_, err = keeper.GetMultipleAssetsPrices(ctx, assets)
	require.ErrorIs(t, err, types.ErrGetPriceAssetNotFound.Wrapf("assetID does not exist in oracle %s", "unexistsAsset"))
}

func TestPricesRemove(t *testing.T) {
	keeper, ctx := keepertest.OracleKeeper(t)
	items := createNPrices(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemovePrices(ctx,
			item.TokenID,
		)
		_, found := keeper.GetPrices(ctx,
			item.TokenID,
		)
		require.False(t, found)
	}
}

func TestPricesGetAll(t *testing.T) {
	keeper, ctx := keepertest.OracleKeeper(t)
	items := createNPrices(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllPrices(ctx)),
	)
}
