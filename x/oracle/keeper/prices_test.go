package keeper_test

import (
	"strconv"
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	keepertest "github.com/imua-xyz/imuachain/testutil/keeper"
	"github.com/imua-xyz/imuachain/testutil/nullify"
	"github.com/imua-xyz/imuachain/x/oracle/keeper"
	"github.com/imua-xyz/imuachain/x/oracle/keeper/testdata"
	"github.com/imua-xyz/imuachain/x/oracle/types"
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
	rst.PriceList = rst.PriceList[1:]
	require.Equal(t, pRes, rst)
}

func TestPricesGetMultiAssets(t *testing.T) {
	keeper, ctx := keepertest.OracleKeeper(t)
	keeper.SetPrices(ctx, testdata.P1)
	assets := make(map[string]struct{})
	assets["0x0b34c4d876cd569129cf56bafabb3f9e97a4ff42_0x9ce1"] = struct{}{}
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

	assets["unexistsAsset"] = struct{}{}
	_, err = keeper.GetMultipleAssetsPrices(ctx, assets)
	// require.ErrorIs(t, err, types.ErrGetPriceAssetNotFound.Wrapf("assetID does not exist in oracle %s", "unexistsAsset"))
	require.ErrorIs(t, err, types.ErrGetPriceRoundNotFound.Wrapf("no valid price for assetIDs=%s", "unexistsAsset"))
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

func TestPriceGrowID(t *testing.T) {
	id1Uint := uint64(1)
	id2Uint := uint64(2)
	id3Uint := uint64(3)
	id4Uint := uint64(4)
	id7Uint := uint64(7)
	id8Uint := uint64(8)
	keeper, ctx := keepertest.OracleKeeper(t)
	keeper.SetPrices(ctx, types.Prices{
		TokenID:     1,
		NextRoundID: id3Uint,
		PriceList: []*types.PriceTimeRound{
			{Price: "123", RoundID: id1Uint},
			{Price: "799", RoundID: id2Uint},
		},
	})

	price, latestRoundID := keeper.GrowRoundID(ctx, 1, id2Uint, true)
	require.Equal(t, "799", price)
	require.Equal(t, id2Uint, latestRoundID)

	price, latestRoundID = keeper.GrowRoundID(ctx, 1, id2Uint, false)
	require.Equal(t, "799", price)
	require.Equal(t, id2Uint, latestRoundID)

	price, latestRoundID = keeper.GrowRoundID(ctx, 1, id3Uint, true)
	require.Equal(t, "799", price)
	require.Equal(t, id3Uint, latestRoundID)

	price, latestRoundID = keeper.GrowRoundID(ctx, 1, id4Uint, true)
	require.Equal(t, "799", price)
	require.Equal(t, id4Uint, latestRoundID)

	price, latestRoundID = keeper.GrowRoundID(ctx, 1, id7Uint, true)
	require.Equal(t, "799", price)
	require.Equal(t, id7Uint, latestRoundID)

	price, latestRoundID = keeper.GrowRoundID(ctx, 1, id8Uint, false)
	require.Equal(t, "799", price)
	require.Equal(t, id8Uint, latestRoundID)
}

func TestPriceGrowID_WithAccumulateFalse(t *testing.T) {
	id1Uint := uint64(1)
	id2Uint := uint64(2)
	id3Uint := uint64(3)
	id4Uint := uint64(4)
	keeper, ctx := keepertest.OracleKeeper(t)
	keeper.SetPrices(ctx, types.Prices{
		TokenID:     1,
		NextRoundID: id3Uint,
		PriceList: []*types.PriceTimeRound{
			{Price: "123", RoundID: id1Uint},
			{Price: "799", RoundID: id2Uint},
		},
	})

	// Get initial accumulated price (should not exist)
	_, found := keeper.GetAccumulatedPrice(ctx, 1)
	require.False(t, found, "accumulated price should not exist initially")

	// Grow round ID with accumulate=false
	price, latestRoundID := keeper.GrowRoundID(ctx, 1, id2Uint, false)
	require.Equal(t, "799", price)
	require.Equal(t, id2Uint, latestRoundID)

	// Verify accumulated price still doesn't exist
	_, found = keeper.GetAccumulatedPrice(ctx, 1)
	require.False(t, found, "accumulated price should not exist when accumulate=false")

	// Grow round ID again with accumulate=false
	price, latestRoundID = keeper.GrowRoundID(ctx, 1, id3Uint, false)
	require.Equal(t, "799", price)
	require.Equal(t, id3Uint, latestRoundID)

	// Verify accumulated price still doesn't exist
	_, found = keeper.GetAccumulatedPrice(ctx, 1)
	require.False(t, found, "accumulated price should not exist when accumulate=false")

	// Grow round ID with accumulate=true to create accumulated price
	price, latestRoundID = keeper.GrowRoundID(ctx, 1, id4Uint, true)
	require.Equal(t, "799", price)
	require.Equal(t, id4Uint, latestRoundID)

	// Now accumulated price should exist
	accPrice, found := keeper.GetAccumulatedPrice(ctx, 1)
	require.True(t, found, "accumulated price should exist after accumulate=true")
	require.Equal(t, id2Uint, accPrice.LastRoundID, "accumulated price LastRoundID should match")
	require.Equal(t, id3Uint, accPrice.StartRoundID, "accumulated price StartRoundID should match")

	// Grow round ID again with accumulate=false
	price, latestRoundID = keeper.GrowRoundID(ctx, 1, uint64(5), false)
	require.Equal(t, "799", price)
	require.Equal(t, uint64(5), latestRoundID)

	// Verify accumulated price remains unchanged (still at round 4)
	accPrice2, found := keeper.GetAccumulatedPrice(ctx, 1)
	require.True(t, found, "accumulated price should still exist")
	require.Equal(t, id2Uint, accPrice2.LastRoundID, "accumulated price LastRoundID should remain at 2")
	require.Equal(t, accPrice.Price, accPrice2.Price, "accumulated price should not change when accumulate=false")
}

func TestAppendPriceTR_MixedAccumulateFlags(t *testing.T) {
	keeper, ctx := keepertest.OracleKeeper(t)
	const tokenID uint64 = 1

	// Start from an empty state with NextRoundID=1.
	keeper.SetPrices(ctx, types.Prices{
		TokenID:     tokenID,
		NextRoundID: 1,
		PriceList:   []*types.PriceTimeRound{},
	})

	// round 1 (accumulate doesn't matter; roundID<=1 is skipped in accumulation)
	ok := keeper.AppendPriceTR(ctx, tokenID, types.PriceTimeRound{
		Price:   "100",
		RoundID: 1,
		Decimal: 18,
	}, true)
	require.True(t, ok)
	acc, found := keeper.GetAccumulatedPrice(ctx, tokenID)
	require.False(t, found)

	// round 2, same price, accumulate=true will create the initial accumulator (Price=0, LastRoundID=0, StartRoundID=1)
	ok = keeper.AppendPriceTR(ctx, tokenID, types.PriceTimeRound{
		Price:   "100",
		RoundID: 2,
		Decimal: 18,
	}, true)
	require.True(t, ok)
	acc, found = keeper.GetAccumulatedPrice(ctx, tokenID)
	require.True(t, found)
	require.Equal(t, uint64(1), acc.StartRoundID)
	require.Equal(t, uint64(0), acc.LastRoundID)
	require.Equal(t, "0", acc.Price)

	// round 3, price changes but accumulate=false -> accumulator should remain unchanged
	ok = keeper.AppendPriceTR(ctx, tokenID, types.PriceTimeRound{
		Price:   "110",
		RoundID: 3,
		Decimal: 18,
	}, false)
	require.True(t, ok)
	acc2, found := keeper.GetAccumulatedPrice(ctx, tokenID)
	require.True(t, found)
	require.Equal(t, acc, acc2)

	// round 4, accumulate=true -> should accumulate prev round (round 3) for 3 rounds (from LastRoundID=0 to RoundID=3)
	ok = keeper.AppendPriceTR(ctx, tokenID, types.PriceTimeRound{
		Price:   "120",
		RoundID: 4,
		Decimal: 18,
	}, true)
	require.True(t, ok)
	acc3, found := keeper.GetAccumulatedPrice(ctx, tokenID)
	require.True(t, found)
	require.Equal(t, uint64(1), acc3.StartRoundID)
	require.Equal(t, uint64(3), acc3.LastRoundID)
	require.Equal(t, "330", acc3.Price)

	// round 5, accumulate=false -> accumulator should remain unchanged
	ok = keeper.AppendPriceTR(ctx, tokenID, types.PriceTimeRound{
		Price:   "130",
		RoundID: 5,
		Decimal: 18,
	}, false)
	require.True(t, ok)
	acc4, found := keeper.GetAccumulatedPrice(ctx, tokenID)
	require.True(t, found)
	require.Equal(t, acc3, acc4)
}
