package keeper_test

import (
	assetstype "github.com/imua-xyz/imuachain/x/assets/types"
)

func (suite *StakingAssetsTestSuite) TestParams() {
	params := &assetstype.Params{
		Gateways: []string{"0x3fC91A3afd70395Cd496C647d5a6CC9D4B2b7FAD"},
	}
	err := suite.App.AssetsKeeper.SetParams(suite.Ctx, params)
	suite.NoError(err)

	getParams, err := suite.App.AssetsKeeper.GetParams(suite.Ctx)
	suite.NoError(err)
	suite.Equal(*params, *getParams)
}

// TestParamsBusinessRules tests business logic validation in Keeper
func (suite *StakingAssetsTestSuite) TestParamsBusinessRules() {
	testCases := []struct {
		name        string
		gateways    []string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid gateway address",
			gateways:    []string{"0x3fC91A3afd70395Cd496C647d5a6CC9D4B2b7FAD"},
			expectError: false,
		},
		{
			name:        "empty gateways list is allowed",
			gateways:    []string{},
			expectError: false,
		},
		{
			name:        "zero address should be rejected by business rules",
			gateways:    []string{"0x0000000000000000000000000000000000000000"},
			expectError: true,
			errorMsg:    "address is in forbidden list",
		},
		{
			name:        "precompile address should be rejected by business rules",
			gateways:    []string{"0x0000000000000000000000000000000000000001"},
			expectError: true,
			errorMsg:    "address is in forbidden list",
		},
		{
			name:        "multiple precompile addresses should be rejected",
			gateways:    []string{"0x0000000000000000000000000000000000000001", "0x0000000000000000000000000000000000000002"},
			expectError: true,
			errorMsg:    "address is in forbidden list",
		},
		{
			name:        "valid multiple gateways",
			gateways:    []string{"0x3fC91A3afd70395Cd496C647d5a6CC9D4B2b7FAD", "0x1234567890123456789012345678901234567890"},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			params := &assetstype.Params{
				Gateways: tc.gateways,
			}
			err := suite.App.AssetsKeeper.SetParams(suite.Ctx, params)
			if tc.expectError {
				suite.Error(err)
				suite.Contains(err.Error(), tc.errorMsg)
			} else {
				suite.NoError(err)
				// Verify params were stored correctly
				getParams, err := suite.App.AssetsKeeper.GetParams(suite.Ctx)
				suite.NoError(err)
				suite.Equal(len(tc.gateways), len(getParams.Gateways))
			}
		})
	}
}
