package keeper_test

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/imua-xyz/imuachain/x/immint/keeper"
	"github.com/imua-xyz/imuachain/x/immint/types"
)

func (suite *KeeperTestSuite) TestQueryParams() {
	defaultParams := types.DefaultParams()
	res, err := suite.queryClient.Params(sdk.WrapSDKContext(suite.Ctx), &types.QueryParamsRequest{})
	suite.Require().NoError(err)
	suite.Require().NotNil(res)
	suite.Require().Equal(defaultParams, res.Params)
}

func (suite *KeeperTestSuite) TestQueryEpochMintInfo() {
	params := suite.App.ImmintKeeper.GetParams(suite.Ctx)
	epochInfo, exist := suite.App.EpochsKeeper.GetEpochInfo(suite.Ctx, params.EpochIdentifier)
	suite.Require().True(exist)
	epochDurationSeconds := int64(epochInfo.Duration.Seconds())
	suite.Require().Greater(epochDurationSeconds, int64(0))
	epochNumberInYear := keeper.SecondsInYear / epochDurationSeconds
	// calculate the expected annual inflation ratio when the inflation parameter is disabled
	totalSupply := suite.App.BankKeeper.GetSupply(suite.Ctx, params.MintDenom).Amount
	annualProvisions := params.EpochReward.Mul(sdk.NewInt(epochNumberInYear))
	annualInflation := sdk.NewDecFromInt(annualProvisions).QuoInt(totalSupply)

	testCases := []struct {
		name      string
		setup     func() types.EpochMintInfo
		expErr    bool
		expErrMsg string
	}{
		{
			name: "inflation disabled",
			setup: func() types.EpochMintInfo {
				return types.EpochMintInfo{
					EpochMintAmount: params.EpochReward,
					AnnualInflation: annualInflation,
				}
			},
			expErr: false,
		},
		{
			name: "start time in future",
			setup: func() types.EpochMintInfo {
				tmpParams := suite.App.ImmintKeeper.GetParams(suite.Ctx)
				tmpParams.InflationParams.Enable = true
				tmpParams.InflationParams.StartTime = suite.Ctx.BlockTime().Unix() + 1
				tmpParams.InflationParams.AnnualInflation = []sdk.Dec{
					sdk.NewDecWithPrec(3, 1),
					sdk.NewDecWithPrec(2, 1),
					sdk.NewDecWithPrec(1, 1),
				}
				suite.App.ImmintKeeper.SetParams(suite.Ctx, tmpParams)
				return types.EpochMintInfo{
					EpochMintAmount: params.EpochReward,
					AnnualInflation: annualInflation,
				}
			},
			expErr: false,
		},
		{
			name: "empty annual inflation list",
			setup: func() types.EpochMintInfo {
				tmpParams := suite.App.ImmintKeeper.GetParams(suite.Ctx)
				tmpParams.InflationParams.Enable = true
				tmpParams.InflationParams.StartTime = suite.Ctx.BlockTime().Unix() + 1
				suite.App.ImmintKeeper.SetParams(suite.Ctx, tmpParams)
				return types.EpochMintInfo{
					EpochMintAmount: params.EpochReward,
					AnnualInflation: annualInflation,
				}
			},
			expErr: false,
		},
		{
			name: "normal case with inflation ratio within bounds",
			setup: func() types.EpochMintInfo {
				tmpParams := suite.App.ImmintKeeper.GetParams(suite.Ctx)
				tmpParams.InflationParams.Enable = true
				tmpParams.InflationParams.StartTime = suite.Ctx.BlockTime().Unix()
				tmpParams.InflationParams.AnnualInflation = []sdk.Dec{
					sdk.NewDecWithPrec(3, 1),
					sdk.NewDecWithPrec(2, 1),
					sdk.NewDecWithPrec(1, 1),
				}
				suite.App.ImmintKeeper.SetParams(suite.Ctx, tmpParams)

				expectedAnnualInflation := tmpParams.InflationParams.AnnualInflation[0]
				expectedAnnualProvisions := expectedAnnualInflation.MulInt(totalSupply)
				epochMintAmount := expectedAnnualProvisions.QuoInt64(epochNumberInYear)
				return types.EpochMintInfo{
					EpochMintAmount: epochMintAmount.TruncateInt(),
					AnnualInflation: expectedAnnualInflation,
				}
			},
			expErr: false,
		},
		{
			name: "index out of range uses last inflation ratio",
			setup: func() types.EpochMintInfo {
				tmpParams := suite.App.ImmintKeeper.GetParams(suite.Ctx)
				tmpParams.InflationParams.Enable = true
				tmpParams.InflationParams.StartTime = suite.Ctx.BlockTime().Unix()
				tmpParams.InflationParams.AnnualInflation = []sdk.Dec{
					sdk.NewDecWithPrec(3, 1),
					sdk.NewDecWithPrec(2, 1),
					sdk.NewDecWithPrec(1, 1),
				}
				suite.App.ImmintKeeper.SetParams(suite.Ctx, tmpParams)

				inflationNumber := len(tmpParams.InflationParams.AnnualInflation)
				// run to out of the inflation ratio lists
				seconds := time.Duration(int64(inflationNumber)*keeper.SecondsInYear + 1)
				suite.CommitAfter(seconds * time.Second)

				totalSupply = suite.App.BankKeeper.GetSupply(suite.Ctx, params.MintDenom).Amount
				expectedAnnualInflation := tmpParams.InflationParams.AnnualInflation[inflationNumber-1]
				expectedAnnualProvisions := expectedAnnualInflation.MulInt(totalSupply)
				epochMintAmount := expectedAnnualProvisions.QuoInt64(epochNumberInYear)
				return types.EpochMintInfo{
					EpochMintAmount: epochMintAmount.TruncateInt(),
					AnnualInflation: expectedAnnualInflation,
				}
			},
			expErr: false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// setup basic test suites
			s.SetupTest()
			var expEpochMintInfo types.EpochMintInfo
			if tc.setup != nil {
				expEpochMintInfo = tc.setup()
			}

			resp, err := s.App.ImmintKeeper.EpochMintInfo(suite.Ctx, &types.QueryEpochMintInfoRequest{})
			if tc.expErr {
				s.Require().ErrorContains(err, tc.expErrMsg)
			} else {
				s.Require().NoError(err)
				s.Require().NotNil(resp)
				s.Require().Equal(expEpochMintInfo, resp.EpochMintInfo)
			}
		})
	}
}
