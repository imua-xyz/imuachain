package keeper_test

import (
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/imua-xyz/imuachain/x/epochs/types"
)

func (suite *KeeperTestSuite) TestEpochInfos() {
	testCases := []struct {
		name    string
		expPass bool
		req     *types.QueryEpochsInfoRequest
		expRes  *types.QueryEpochsInfoResponse
	}{
		{
			name:    "all genesis epochs",
			expPass: true,
			req:     &types.QueryEpochsInfoRequest{},
			expRes: &types.QueryEpochsInfoResponse{
				// the below is the same as the DefaultGenesis
				// but with CurrentEpoch set to 1
				// the CurrentEpochStartHeight should be the height of the ctx with which BeginBlock was called, which
				// is the suite.Ctx's height.
				// the CurrentEpochStartTime is the originally set start time, which is the EpochInfo.StartTime.
				// since at genesis, it is 0, it goes to the ctx.BlockTime() during Genesis.
				Epochs: []types.EpochInfo{
					{
						Identifier:              types.DayEpochID,
						StartTime:               suite.InitTime,
						Duration:                time.Hour * 24,
						CurrentEpoch:            1,
						CurrentEpochStartTime:   suite.InitTime,
						EpochCountingStarted:    true,
						CurrentEpochStartHeight: suite.Ctx.BlockHeight(),
					},
					{
						Identifier:              types.HourEpochID,
						StartTime:               suite.InitTime,
						Duration:                time.Hour,
						CurrentEpoch:            1,
						CurrentEpochStartTime:   suite.InitTime,
						EpochCountingStarted:    true,
						CurrentEpochStartHeight: suite.Ctx.BlockHeight(),
					},
					{
						Identifier:              types.MinuteEpochID,
						StartTime:               suite.InitTime,
						Duration:                time.Minute,
						CurrentEpoch:            1,
						CurrentEpochStartTime:   suite.InitTime,
						EpochCountingStarted:    true,
						CurrentEpochStartHeight: suite.Ctx.BlockHeight(),
					},
					{
						Identifier:              types.WeekEpochID,
						StartTime:               suite.InitTime,
						Duration:                time.Hour * 24 * 7,
						CurrentEpoch:            1,
						CurrentEpochStartTime:   suite.InitTime,
						EpochCountingStarted:    true,
						CurrentEpochStartHeight: suite.Ctx.BlockHeight(),
					},
				},
				// we make the query with suite.Ctx, which is what
				// this responds with.
				BlockTime: suite.Ctx.BlockTime(),
				// since we didn't send a pagination, the next key is nil.
				Pagination: &query.PageResponse{
					Total:   4,
					NextKey: nil,
				},
			},
		},
	}
	ctx := sdk.WrapSDKContext(suite.Ctx)
	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			res, err := suite.queryClient.EpochInfos(ctx, tc.req)
			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(tc.expRes, res)
				// suite.Require().Equal(tc.expRes.BlockTime, res.BlockTime)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestCurrentEpoch() {
	testCases := []struct {
		name    string
		expPass bool
		req     *types.QueryCurrentEpochRequest
		expRes  *types.QueryCurrentEpochResponse
	}{
		{
			name:    "unknown epoch",
			expPass: false,
			req: &types.QueryCurrentEpochRequest{
				Identifier: "unknown",
			},
			expRes: nil,
		},
		{
			name:    "blank epoch",
			expPass: false,
			req: &types.QueryCurrentEpochRequest{
				Identifier: "",
			},
			expRes: nil,
		},
		{
			name:    "day epoch",
			expPass: true,
			req: &types.QueryCurrentEpochRequest{
				Identifier: types.DayEpochID,
			},
			expRes: &types.QueryCurrentEpochResponse{
				// increased by BeginBlocker after InitGenesis
				CurrentEpoch: 1,
			},
		},
	}
	ctx := sdk.WrapSDKContext(suite.Ctx)
	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			res, err := suite.queryClient.CurrentEpoch(ctx, tc.req)
			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(tc.expRes, res)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
