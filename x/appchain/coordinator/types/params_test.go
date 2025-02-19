package types_test

import (
	"strings"
	"testing"
	time "time"

	"github.com/imua-xyz/imuachain/x/appchain/coordinator/types"
	epochstypes "github.com/imua-xyz/imuachain/x/epochs/types"
)

func TestValidate(t *testing.T) {
	cases := []struct {
		name      string
		params    types.Params
		expResult bool
		expError  string
		malleate  func(params *types.Params)
	}{
		{
			name:      "default params",
			params:    types.DefaultParams(),
			expResult: true,
		},
		{
			name:   "nil client",
			params: types.DefaultParams(),
			malleate: func(params *types.Params) {
				params.TemplateClient = nil
			},
			expResult: false,
			expError:  "template client is nil",
		},
		{
			name:   "invalid client",
			params: types.DefaultParams(),
			malleate: func(params *types.Params) {
				params.TemplateClient.UpgradePath = []string{
					"", // empty string is invalid
				}
			},
			expResult: false,
			expError:  "key in upgrade path at index 0 cannot be empty",
		},
		{
			name:   "invalid trust period fraction",
			params: types.DefaultParams(),
			malleate: func(params *types.Params) {
				params.TrustingPeriodFraction = "1.5"
			},
			expResult: false,
			expError:  "trusting period fraction is invalid",
		},
		{
			name:   "invalid IBC timeout duration",
			params: types.DefaultParams(),
			malleate: func(params *types.Params) {
				params.IBCTimeoutPeriod = time.Duration(-1)
			},
			expResult: false,
			expError:  "IBC timeout period is invalid",
		},
		{
			name:   "invalid init timeout period",
			params: types.DefaultParams(),
			malleate: func(params *types.Params) {
				params.InitTimeoutPeriod = epochstypes.NewEpoch(0, "week")
			},
			expResult: false,
			expError:  "init timeout period is invalid",
		},
		{
			name:   "invalid BSC timeout period",
			params: types.DefaultParams(),
			malleate: func(params *types.Params) {
				params.VSCTimeoutPeriod = epochstypes.NewEpoch(0, "week")
			},
			expResult: false,
			expError:  "VSC timeout period is invalid",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.malleate != nil {
				tc.malleate(&tc.params)
			}
			// first, validate the test case itself
			if tc.expResult && tc.expError != "" {
				t.Fatal("invalid test case: expected success but got error")
			} else if !tc.expResult && tc.expError == "" {
				t.Fatal("invalid test case: expected error but got success")
			}
			// then run the test case
			err := tc.params.Validate()
			if tc.expResult && err != nil {
				t.Fatalf("expected no error, got %s", err)
			}
			if !tc.expResult {
				if err == nil {
					t.Fatal("expected error, got none")
				} else {
					if !strings.Contains(err.Error(), tc.expError) {
						t.Fatalf("expected error %q, got %q", tc.expError, err.Error())
					}
				}
			}
		})
	}
}
