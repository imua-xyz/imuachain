package keeper_test

import (
	"testing"

	keepertest "github.com/imua-xyz/imuachain/testutil/keeper"
	"github.com/imua-xyz/imuachain/x/oracle/keeper"
	"github.com/imua-xyz/imuachain/x/oracle/types"
	"github.com/stretchr/testify/require"
)

func TestGetStartBaseBlock(t *testing.T) {
	tests := []struct {
		name               string
		firstQuotingHeight uint64
		window             uint64
		interval           uint64
		tokenFeeders       []*types.TokenFeeder
		expected           uint64
		description        string
	}{
		{
			name:               "basic scenario - two assets A and B",
			firstQuotingHeight: 5,
			window:             3,
			interval:           30,
			tokenFeeders: []*types.TokenFeeder{
				{
					TokenID:        1,
					StartBaseBlock: 1,
					Interval:       30,
					EndBlock:       0, // running feeder
				},
				{
					TokenID:        2,
					StartBaseBlock: 2,
					Interval:       30,
					EndBlock:       0, // running feeder
				},
			},
			expected:    1, // Should return offset 1, so startBaseBlock = 5 + 1 - 1 = 5, first quoting block = 6
			description: "Assets A (startBaseBlock=1) and B (startBaseBlock=2) with window=3, interval=30. Their quoting windows are {2,3,4} and {3,4,5}. With offset=1, startBaseBlock=5, first quoting block=6 which has no conflicts.",
		},
		{
			name:               "empty token feeders",
			firstQuotingHeight: 10,
			window:             5,
			interval:           20,
			tokenFeeders:       []*types.TokenFeeder{},
			expected:           0, // Should return offset 0, so startBaseBlock = 10 + 0 - 1 = 9, first quoting block = 10
			description:        "No existing feeders, so any block is available. Should return offset 0.",
		},
		{
			name:               "single token feeder",
			firstQuotingHeight: 15,
			window:             4,
			interval:           25,
			tokenFeeders: []*types.TokenFeeder{
				{
					TokenID:        1,
					StartBaseBlock: 10,
					Interval:       25,
					EndBlock:       0, // running feeder
				},
			},
			expected:    0, // Should return offset 0, so startBaseBlock = 15 + 0 - 1 = 14, first quoting block = 15
			description: "Single feeder with startBaseBlock=10, interval=25. Its quoting windows are {11,12,13,14}, {36,37,38,39}, etc. Block 15 is available.",
		},
		{
			name:               "multiple feeders with different intervals",
			firstQuotingHeight: 20,
			window:             3,
			interval:           15,
			tokenFeeders: []*types.TokenFeeder{
				{
					TokenID:        1,
					StartBaseBlock: 5,
					Interval:       10,
					EndBlock:       0, // running feeder
				},
				{
					TokenID:        2,
					StartBaseBlock: 8,
					Interval:       12,
					EndBlock:       0, // running feeder
				},
				{
					TokenID:        3,
					StartBaseBlock: 12,
					Interval:       15,
					EndBlock:       0, // running feeder
				},
			},
			expected:    0, // Should return offset 0, so startBaseBlock = 20 + 0 - 1 = 19, first quoting block = 20
			description: "Multiple feeders with different intervals. Need to find the first available block.",
		},
		{
			name:               "feeders with ended feeders",
			firstQuotingHeight: 50,
			window:             5,
			interval:           20,
			tokenFeeders: []*types.TokenFeeder{
				{
					TokenID:        1,
					StartBaseBlock: 10,
					Interval:       20,
					EndBlock:       45, // ended feeder
				},
				{
					TokenID:        2,
					StartBaseBlock: 15,
					Interval:       20,
					EndBlock:       0, // running feeder
				},
			},
			expected:    0, // Should return offset 0, so startBaseBlock = 50 + 0 - 1 = 49, first quoting block = 50
			description: "One ended feeder (EndBlock=45) and one running feeder. Ended feeder should be ignored.",
		},
		{
			name:               "all feeders ended before firstQuotingHeight",
			firstQuotingHeight: 100,
			window:             3,
			interval:           25,
			tokenFeeders: []*types.TokenFeeder{
				{
					TokenID:        1,
					StartBaseBlock: 10,
					Interval:       20,
					EndBlock:       50, // ended feeder
				},
				{
					TokenID:        2,
					StartBaseBlock: 20,
					Interval:       25,
					EndBlock:       80, // ended feeder
				},
			},
			expected:    0, // Should return offset 0, so startBaseBlock = 100 + 0 - 1 = 99, first quoting block = 100
			description: "All feeders ended before firstQuotingHeight, so any block is available.",
		},
		{
			name:               "dense scheduling - find minimum conflict",
			firstQuotingHeight: 10,
			window:             2,
			interval:           5,
			tokenFeeders: []*types.TokenFeeder{
				{
					TokenID:        1,
					StartBaseBlock: 5,
					Interval:       5,
					EndBlock:       0, // running feeder
				},
				{
					TokenID:        2,
					StartBaseBlock: 6,
					Interval:       5,
					EndBlock:       0, // running feeder
				},
				{
					TokenID:        3,
					StartBaseBlock: 7,
					Interval:       5,
					EndBlock:       0, // running feeder
				},
			},
			expected:    0, // Should return offset 0, so startBaseBlock = 10 + 0 - 1 = 9, first quoting block = 10
			description: "Dense scheduling with multiple feeders. Need to find the block with minimum conflicts.",
		},
		{
			name:               "large interval scenario",
			firstQuotingHeight: 1000,
			window:             10,
			interval:           100,
			tokenFeeders: []*types.TokenFeeder{
				{
					TokenID:        1,
					StartBaseBlock: 100,
					Interval:       200,
					EndBlock:       0, // running feeder
				},
				{
					TokenID:        2,
					StartBaseBlock: 150,
					Interval:       300,
					EndBlock:       0, // running feeder
				},
			},
			expected:    0, // Should return offset 0, so startBaseBlock = 1000 + 0 - 1 = 999, first quoting block = 1000
			description: "Large intervals and window. Need to find available block in the range.",
		},
		{
			name:               "edge case - firstQuotingHeight equals feeder start",
			firstQuotingHeight: 21,
			window:             3,
			interval:           15,
			tokenFeeders: []*types.TokenFeeder{
				{
					TokenID:        1,
					StartBaseBlock: 20,
					Interval:       15,
					EndBlock:       0, // running feeder
				},
			},
			expected:    3, // Should return offset 3, so startBaseBlock = 21 + 3 - 1 = 23, first quoting block = 24
			description: "firstQuotingHeight equals feeder startBaseBlock. Feeder's first quoting window is {21,22,23}. New asset needs offset 3 to avoid conflict.",
		},
		{
			name:               "complex overlapping scenario",
			firstQuotingHeight: 30,
			window:             4,
			interval:           20,
			tokenFeeders: []*types.TokenFeeder{
				{
					TokenID:        1,
					StartBaseBlock: 10,
					Interval:       15,
					EndBlock:       0, // running feeder
				},
				{
					TokenID:        2,
					StartBaseBlock: 15,
					Interval:       20,
					EndBlock:       0, // running feeder
				},
				{
					TokenID:        3,
					StartBaseBlock: 20,
					Interval:       25,
					EndBlock:       0, // running feeder
				},
			},
			expected:    0, // Should return offset 0, so startBaseBlock = 30 + 0 - 1 = 29, first quoting block = 30
			description: "Complex overlapping scenario with multiple feeders having different intervals.",
		},
	}

	for _, tt := range tests {
		//	if !strings.HasPrefix(tt.name, "edge case - firstQuotingHeight equals") {
		//		continue
		//	}
		t.Run(tt.name, func(t *testing.T) {
			result := keeper.GetStartBaseBlock(tt.firstQuotingHeight, tt.window, tt.interval, tt.tokenFeeders)
			require.Equal(t, tt.expected, result, tt.description)
		})
	}
}

// TestGetStartBaseBlockDetailed tests the function with detailed block-by-block analysis
func TestGetStartBaseBlockDetailed(t *testing.T) {
	// Test case: Assets A (startBaseBlock=1, interval=30) and B (startBaseBlock=2, interval=30)
	// Window = 3, firstQuotingHeight = 5, interval = 30
	//
	// Asset A quoting windows: {2,3,4}, {32,33,34}, {62,63,64}, ...
	// Asset B quoting windows: {3,4,5}, {33,34,35}, {63,64,65}, ...
	//
	// For new asset C with firstQuotingHeight=5, interval=30:
	// We need to find offset such that startBaseBlock = firstQuotingHeight + offset - 1
	// and first quoting block (startBaseBlock+1) doesn't conflict with existing assets.
	//
	// If offset=1, then startBaseBlock = 5 + 1 - 1 = 5, first quoting block=6, window={6,7,8}
	// This doesn't conflict with A or B, so it should work.

	tokenFeeders := []*types.TokenFeeder{
		{
			TokenID:        1,
			StartBaseBlock: 1,
			Interval:       30,
			EndBlock:       0,
		},
		{
			TokenID:        2,
			StartBaseBlock: 2,
			Interval:       30,
			EndBlock:       0,
		},
	}

	offset := keeper.GetStartBaseBlock(5, 3, 30, tokenFeeders)

	// The result is the offset from firstQuotingHeight
	// So if offset = 1, then startBaseBlock = firstQuotingHeight + offset - 1 = 5 + 1 - 1 = 5
	// First quoting block = startBaseBlock + 1 = 5 + 1 = 6, window = {6, 7, 8}
	startBaseBlock := 5 + offset - 1
	firstQuotingBlock := startBaseBlock + 1

	t.Logf("Offset: %d", offset)
	t.Logf("Calculated startBaseBlock: %d", startBaseBlock)
	t.Logf("First quoting block: %d", firstQuotingBlock)
	t.Logf("Quoting window: {%d, %d, %d}", firstQuotingBlock, firstQuotingBlock+1, firstQuotingBlock+2)

	// Verify that the calculated window doesn't conflict with existing assets
	// Asset A windows: {2,3,4}, {32,33,34}, {62,63,64}, ...
	// Asset B windows: {3,4,5}, {33,34,35}, {63,64,65}, ...

	// The new window should not overlap with any existing windows
	require.True(t, firstQuotingBlock > 5, "First quoting block should be after the conflicting range")
	require.Equal(t, uint64(1), offset, "Expected offset 1 for this scenario")
}

func TestRegisterNewTokenAndSetTokenFeeder_StartBaseBlock_NoOverlap(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)
	ctx = ctx.WithBlockHeight(10)

	// Configure one existing running feeder whose quoting window occupies {21,22,23} when StartBaseBlock=20 and MaxNonce=3.
	p := k.GetParams(ctx)
	p.MaxNonce = 3
	p.TokenFeeders = []*types.TokenFeeder{
		{}, // index 0 reserved
		{
			TokenID:        1,
			RuleID:         1,
			StartRoundID:   1,
			StartBaseBlock: 20,
			Interval:       30,
			EndBlock:       0,
		},
	}
	k.SetParams(ctx, p)

	// Add a new token at height=10. Default startAfterBlocks=10 => initial startBaseBlock=20 (first quoting height=21),
	// which conflicts with the existing feeder. The new feeder should be shifted so that:
	// - new StartBaseBlock == 23
	// - first quoting block == 24 (no overlap with {21,22,23})
	oInfo := &types.OracleInfo{
		AssetID: "0xdeadbeef_0x9ce1",
	}
	oInfo.Chain.Name = "Ethereum"
	oInfo.Chain.Desc = "-"
	oInfo.Token.Name = "TEST_TOKEN_1"
	oInfo.Token.Decimal = "18"
	oInfo.Token.Contract = "0x"
	oInfo.Token.AssetID = oInfo.AssetID
	// Keep interval empty to use defaultInterval=30 in keeper.
	oInfo.Feeder.Interval = ""

	err := k.RegisterNewTokenAndSetTokenFeeder(ctx, oInfo)
	require.NoError(t, err)

	p2 := k.GetParams(ctx)
	require.GreaterOrEqual(t, len(p2.TokenFeeders), 3)
	newTF := p2.TokenFeeders[len(p2.TokenFeeders)-1]
	require.NotNil(t, newTF)
	require.EqualValues(t, 23, newTF.StartBaseBlock)

	// Sanity: the first quoting block is startBaseBlock+1
	require.EqualValues(t, uint64(24), newTF.StartBaseBlock+1)

	// Ensure the chosen quoting window does not overlap the existing one.
	existingWindow := map[uint64]struct{}{21: {}, 22: {}, 23: {}}
	for h := newTF.StartBaseBlock + 1; h <= newTF.StartBaseBlock+uint64(p2.MaxNonce); h++ {
		_, overlap := existingWindow[h]
		require.False(t, overlap, "unexpected quoting window overlap at height %d", h)
	}
}
