package keeper_test

import (
	sdkmath "cosmossdk.io/math"
	"github.com/imua-xyz/imuachain/x/immint/types"
)

func (suite *KeeperTestSuite) TestUpdateParams() {
	prevParams := suite.App.ImmintKeeper.GetParams(suite.Ctx)
	cases := []struct {
		name       string
		params     types.Params
		authority  string
		expectPass bool
		expErr     string
		expParams  types.Params
	}{
		{
			name:       "valid params and authority",
			params:     types.NewParams("nextmint", sdkmath.NewInt(1), "week"),
			authority:  suite.App.ImmintKeeper.GetAuthority(),
			expectPass: true,
			expParams:  types.NewParams("nextmint", sdkmath.NewInt(1), "week"),
		},
		{
			name:       "valid params except denom and valid authority",
			params:     types.NewParams("a", sdkmath.NewInt(1), "week"),
			authority:  suite.App.ImmintKeeper.GetAuthority(),
			expectPass: true,
			expParams:  types.NewParams(prevParams.MintDenom, sdkmath.NewInt(1), "week"),
		},
		{
			name:       "valid params except nil reward and valid authority",
			params:     types.NewParams("nextmint", sdkmath.Int{}, "week"),
			authority:  suite.App.ImmintKeeper.GetAuthority(),
			expectPass: true,
			expParams:  types.NewParams("nextmint", prevParams.EpochReward, "week"),
		},
		{
			name:       "valid params except negative reward and valid authority",
			params:     types.NewParams("nextmint", prevParams.EpochReward.Neg(), "week"),
			authority:  suite.App.ImmintKeeper.GetAuthority(),
			expectPass: true,
			expParams:  types.NewParams("nextmint", prevParams.EpochReward, "week"),
		},
		{
			name:       "valid params except blank epoch and valid authority",
			params:     types.NewParams("nextmint", sdkmath.NewInt(1), ""),
			authority:  suite.App.ImmintKeeper.GetAuthority(),
			expectPass: true,
			expParams:  types.NewParams("nextmint", sdkmath.NewInt(1), prevParams.EpochIdentifier),
		},
		{
			name:       "valid params except non-existing epoch and valid authority",
			params:     types.NewParams("nextmint", sdkmath.NewInt(1), "hello_i_am_not_an_epoch"),
			authority:  suite.App.ImmintKeeper.GetAuthority(),
			expectPass: true,
			expParams:  types.NewParams("nextmint", sdkmath.NewInt(1), prevParams.EpochIdentifier),
		},
		// {
		// 	name:       "valid params but invalid authority",
		// 	params:     types.NewParams("nextmint", sdkmath.NewInt(1), "day"),
		// 	authority:  sdk.AccAddress(common.HexToAddress("0x0").Bytes()).String(),
		// 	expectPass: false,
		// 	expErr:     "invalid authority",
		// },
	}
	for _, tc := range cases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			msg := types.MsgUpdateParams{
				Params:    tc.params,
				Authority: tc.authority,
			}
			_, err := suite.App.ImmintKeeper.UpdateParams(suite.Ctx, &msg)
			if tc.expectPass {
				suite.Require().NoError(err)
				suite.Require().True(len(tc.expErr) == 0, tc.expErr)
				suite.Require().Equal(tc.expParams, suite.App.ImmintKeeper.GetParams(suite.Ctx))
			} else {
				suite.Require().Error(err)
				suite.Require().True(len(tc.expErr) > 0, tc.expErr)
				suite.Require().Contains(err.Error(), tc.expErr)
			}
		})
	}
}
