package keeper

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/imua-xyz/imuachain/x/oracle/types"
)

func TestAccumulatePriceTR_Internal(t *testing.T) {
	tests := []struct {
		name           string
		setup          func(*Keeper, sdk.Context, uint64)
		input          types.PriceTimeRound
		expectedError  bool
		expectedResult *types.PriceAcc
	}{
		{
			name: "skip accumulation for roundID <= 1",
			setup: func(k *Keeper, ctx sdk.Context, tokenID uint64) {
				// No setup needed
			},
			input: types.PriceTimeRound{
				Price:   "100",
				RoundID: 1,
				Decimal: 18,
			},
			expectedError:  false,
			expectedResult: nil, // No accumulation should happen
		},
		{
			name: "error when previous round not found",
			setup: func(k *Keeper, ctx sdk.Context, tokenID uint64) {
				// No setup - no previous round exists
			},
			input: types.PriceTimeRound{
				Price:   "100",
				RoundID: 3,
				Decimal: 18,
			},
			expectedError:  true,
			expectedResult: nil,
		},
		{
			name: "create initial accumulated price when prices are equal",
			setup: func(k *Keeper, ctx sdk.Context, tokenID uint64) {
				k.AppendPriceTR(ctx, tokenID, types.PriceTimeRound{
					Price:   "100",
					RoundID: 1,
					Decimal: 18,
				}, true)
			},
			input: types.PriceTimeRound{
				Price:   "100",
				RoundID: 2,
				Decimal: 18,
			},
			expectedError: false,
			expectedResult: &types.PriceAcc{
				Price:        "0",
				StartRoundID: 1,
				LastRoundID:  0,
				Decimal:      0,
			},
		},
		{
			name: "accumulate price when prices are different",
			setup: func(k *Keeper, ctx sdk.Context, tokenID uint64) {
				k.AppendPriceTR(ctx, tokenID, types.PriceTimeRound{
					Price:   "100",
					RoundID: 1,
					Decimal: 18,
				}, true)
				k.AppendPriceTR(ctx, tokenID, types.PriceTimeRound{
					Price:   "100",
					RoundID: 2,
					Decimal: 18,
				}, true)
			},
			input: types.PriceTimeRound{
				Price:   "110",
				RoundID: 3,
				Decimal: 18,
			},
			expectedError: false,
			expectedResult: &types.PriceAcc{
				Price:        "200",
				StartRoundID: 1,
				LastRoundID:  2,
				Decimal:      18,
			},
		},
		{
			name: "continue accumulation with existing accumulated price",
			setup: func(k *Keeper, ctx sdk.Context, tokenID uint64) {
				k.AppendPriceTR(ctx, tokenID, types.PriceTimeRound{
					Price:   "100",
					RoundID: 1,
					Decimal: 18,
				}, true)
				k.AppendPriceTR(ctx, tokenID, types.PriceTimeRound{
					Price:   "100",
					RoundID: 2,
					Decimal: 18,
				}, true)
				k.AppendPriceTR(ctx, tokenID, types.PriceTimeRound{
					Price:   "110",
					RoundID: 3,
					Decimal: 18,
				}, true)
			},
			input: types.PriceTimeRound{
				Price:   "120",
				RoundID: 4,
				Decimal: 18,
			},
			expectedError: false,
			expectedResult: &types.PriceAcc{
				Price:        "310",
				StartRoundID: 1,
				LastRoundID:  3,
				Decimal:      18,
			},
		},
		{
			name: "skip accumulation when previous round is already accumulated",
			setup: func(k *Keeper, ctx sdk.Context, tokenID uint64) {
				k.AppendPriceTR(ctx, tokenID, types.PriceTimeRound{
					Price:   "100",
					RoundID: 1,
					Decimal: 18,
				}, true)
				k.AppendPriceTR(ctx, tokenID, types.PriceTimeRound{
					Price:   "100",
					RoundID: 2,
					Decimal: 18,
				}, true)
				k.AppendPriceTR(ctx, tokenID, types.PriceTimeRound{
					Price:   "110",
					RoundID: 3,
					Decimal: 18,
				}, true)
				// Set up accumulated price that already includes round 3
				accPrice := types.PriceAcc{
					Price:        "210",
					StartRoundID: 2,
					LastRoundID:  3,
					Decimal:      18,
				}
				store := ctx.KVStore(k.storeKey)
				key := types.PricesAccumulatedKey(tokenID)
				store.Set(key, k.cdc.MustMarshal(&accPrice))
			},
			input: types.PriceTimeRound{
				Price:   "120",
				RoundID: 4,
				Decimal: 18,
			},
			expectedError: false,
			expectedResult: &types.PriceAcc{
				Price:        "210",
				StartRoundID: 2,
				LastRoundID:  3,
				Decimal:      18,
			},
		},
		{
			name: "handle different decimals correctly",
			setup: func(k *Keeper, ctx sdk.Context, tokenID uint64) {
				k.AppendPriceTR(ctx, tokenID, types.PriceTimeRound{
					Price:   "100",
					RoundID: 1,
					Decimal: 6,
				}, true)
				k.AppendPriceTR(ctx, tokenID, types.PriceTimeRound{
					Price:   "100",
					RoundID: 2,
					Decimal: 8,
				}, true)
			},
			input: types.PriceTimeRound{
				Price:   "110",
				RoundID: 3,
				Decimal: 18,
			},
			expectedError: false,
			expectedResult: &types.PriceAcc{
				Price:        "10100",
				StartRoundID: 1,
				LastRoundID:  2,
				Decimal:      8,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fresh keeper and context for each test case
			keeper, ctx := MockOracleKeeper(t)
			tokenID := uint64(1)

			// Run setup
			tt.setup(keeper, ctx, tokenID)

			// Execute the function under test
			err := keeper.accumulatePriceTR(ctx, tokenID, tt.input)

			// Check error expectation
			if tt.expectedError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			// Check result expectation
			if tt.expectedResult == nil {
				// No accumulation should have happened
				_, found := keeper.GetAccumulatedPrice(ctx, tokenID)
				require.False(t, found)
			} else {
				// Check accumulated price matches expectation
				accPrice, found := keeper.GetAccumulatedPrice(ctx, tokenID)
				require.True(t, found)
				require.Equal(t, tt.expectedResult.Price, accPrice.Price)
				require.Equal(t, tt.expectedResult.StartRoundID, accPrice.StartRoundID)
				require.Equal(t, tt.expectedResult.LastRoundID, accPrice.LastRoundID)
				require.Equal(t, tt.expectedResult.Decimal, accPrice.Decimal)
			}
		})
	}
}
