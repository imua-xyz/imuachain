package types_test

import (
	"strings"
	"testing"

	sdkmath "cosmossdk.io/math"
	"github.com/ExocoreNetwork/exocore/x/exomint/types"
	"github.com/cometbft/cometbft/libs/log"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestValidate(t *testing.T) {
	res, _ := sdk.NewIntFromString(types.DefaultEpochRewardStr)
	cases := []struct {
		name      string
		params    types.Params
		expResult bool
		expError  string
	}{
		{
			name:      "valid params",
			params:    types.DefaultParams(),
			expResult: true,
		},
		{
			// Denominations can be 3 ~ 128 characters long and support letters, followed by either
			// a letter, a number or a separator ('/', ':', '.', '_' or '-').
			name: "invalid mint denom",
			params: types.Params{
				MintDenom:       "aa",
				EpochReward:     res,
				EpochIdentifier: "day",
			},
			expResult: false,
			expError:  "invalid denom",
		},
		{
			name: "invalid reward",
			params: types.Params{
				MintDenom:       "aevmos",
				EpochReward:     res.Neg(),
				EpochIdentifier: "day",
			},
			expResult: false,
			expError:  "mint reward must be non-negative",
		},
		{
			name: "nil reward",
			params: types.Params{
				MintDenom:       "aevmos",
				EpochReward:     sdkmath.Int{},
				EpochIdentifier: "day",
			},
			expResult: false,
			expError:  "epoch reward cannot be nil",
		},
		{
			name: "zero reward",
			params: types.Params{
				MintDenom:       "aevmos",
				EpochReward:     sdkmath.ZeroInt(),
				EpochIdentifier: "day",
			},
			expResult: true,
		},
		{
			name: "invalid epoch identifier",
			params: types.Params{
				MintDenom:   "aevmos",
				EpochReward: res,
				// all are valid except blank
				EpochIdentifier: "",
			},
			expResult: false,
			expError:  "epoch identifier",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
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

type NoOpLogger struct{}

func (l NoOpLogger) Debug(msg string, keyvals ...interface{}) {}
func (l NoOpLogger) Info(msg string, keyvals ...interface{})  {}
func (l NoOpLogger) Error(msg string, keyvals ...interface{}) {}
func (l NoOpLogger) With(keyvals ...interface{}) log.Logger   { return l }

var _ log.Logger = NoOpLogger{}

func TestOverrideIfRequired(t *testing.T) {
	cases := []struct {
		name     string
		prev     types.Params
		next     types.Params
		over     types.Params
		malleate func(next *types.Params)
	}{
		{
			name: "no override",
			prev: types.DefaultParams(),
			next: types.DefaultParams(),
			over: types.DefaultParams(),
		},
		{
			name: "mint override",
			prev: types.DefaultParams(),
			next: types.DefaultParams(),
			over: types.DefaultParams(),
			malleate: func(next *types.Params) {
				next.MintDenom = ""
			},
		},
		{
			name: "nil epoch reward",
			prev: types.DefaultParams(),
			next: types.DefaultParams(),
			over: types.DefaultParams(),
			malleate: func(next *types.Params) {
				next.EpochReward = sdkmath.Int{}
			},
		},
		{
			name: "zero epoch reward",
			prev: types.DefaultParams(),
			next: types.DefaultParams(),
			over: types.NewParams(
				types.DefaultMintDenom,
				sdkmath.ZeroInt(), // 0 is not overridden
				types.DefaultEpochIdentifier,
			),
			malleate: func(next *types.Params) {
				next.EpochReward = sdkmath.ZeroInt()
			},
		},
		{
			name: "negative epoch reward",
			prev: types.DefaultParams(),
			next: types.DefaultParams(),
			over: types.DefaultParams(),
			malleate: func(next *types.Params) {
				next.EpochReward = sdkmath.NewInt(-1)
			},
		},
		{
			name: "nil epoch reward",
			prev: types.DefaultParams(),
			next: types.DefaultParams(),
			over: types.DefaultParams(),
			malleate: func(next *types.Params) {
				next.EpochReward = sdkmath.Int{}
			},
		},
		{
			name: "blank epoch identifier",
			prev: types.DefaultParams(),
			next: types.DefaultParams(),
			over: types.DefaultParams(),
			malleate: func(next *types.Params) {
				next.EpochIdentifier = ""
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.malleate != nil {
				tc.malleate(&tc.next)
			}
			over := tc.next.OverrideIfRequired(tc.prev, NoOpLogger{})
			if !over.Equal(tc.over) {
				t.Fatalf("expected %v, got %v", tc.over, over)
			}
		})
	}
}
