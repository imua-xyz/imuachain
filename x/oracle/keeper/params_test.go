package keeper_test

import (
	"testing"

	testkeeper "github.com/ExocoreNetwork/exocore/testutil/keeper"
	"github.com/ExocoreNetwork/exocore/x/oracle/keeper/testdata"
	"github.com/ExocoreNetwork/exocore/x/oracle/types"
	"github.com/stretchr/testify/require"
)

func TestGetParams(t *testing.T) {
	k, ctx := testkeeper.OracleKeeper(t)
	params := testdata.DefaultParamsForTest()

	k.SetParams(ctx, params)

	require.EqualValues(t, params, k.GetParams(ctx))
}

func TestUpdateTokenFeederUpdate(t *testing.T) {
	tests := []struct {
		name        string
		tokenFeeder types.TokenFeeder
		height      uint64
		err         error
	}{
		// invalid inputs
		// fail when add/update fields, before validation
		{
			name: "invalid update, empty fields to update",
			tokenFeeder: types.TokenFeeder{
				TokenID: 1,
			},
			height: 1,
			err:    types.ErrInvalidParams.Wrap("invalid tokenFeeder to update, no valid field set"),
		},
		{
			name: "invalid udpate, for not-start feeder, set StartbaseBlock to history height",
			tokenFeeder: types.TokenFeeder{
				TokenID: 1,
				// set current height to 100 to test fail case
				StartBaseBlock: 10,
			},
			height: 100,
			err:    types.ErrInvalidParams.Wrap("invalid tokenFeeder to update, invalid StartBaseBlock"),
		},
		{
			name: "invalid update, for running feeder, set EndBlock to history height",
			tokenFeeder: types.TokenFeeder{
				TokenID: 1,
				// set current height to 2000000 to test fail case
				EndBlock: 1500000,
			},
			height: 2000000,
			err:    types.ErrInvalidParams.Wrap("invalid tokenFeeder to update, invalid EndBlock"),
		},
		{
			name: "invalid update, for stopped feeder, restart a feeder with wrong StartRoundID",
			tokenFeeder: types.TokenFeeder{
				TokenID: 2,
				RuleID:  1,
				// set current height to 100
				StartBaseBlock: 1000,
				// should be 4
				StartRoundID: 5,
				Interval:     10,
			},
			height: 100,
			err:    types.ErrInvalidParams.Wrap("invalid tokenFeeder to update"),
		},
		// success adding/updating, but fail validation
		{
			name: "invalid update, for new feeder, EndBlock is not set properly",
			tokenFeeder: types.TokenFeeder{
				TokenID:        3,
				StartBaseBlock: 10,
				StartRoundID:   1,
				Interval:       10,
				EndBlock:       51,
			},
			height: 1,
			err:    types.ErrInvalidParams.Wrap("invalid tokenFeeder, invalid EndBlock"),
		},
		{
			name: "invalid update, for new feeder, tokenID not exists",
			tokenFeeder: types.TokenFeeder{
				TokenID:        4,
				StartBaseBlock: 10,
				StartRoundID:   1,
				Interval:       10,
				EndBlock:       58,
			},
			height: 1,
			err:    types.ErrInvalidParams.Wrap("invalid tokenFeeder, non-exist tokenID referred"),
		},
		{
			name: "invalid udpate, for existing feeder, feeder not started, and set endblock to history height",
			tokenFeeder: types.TokenFeeder{
				TokenID:  2,
				EndBlock: 5,
			},
			height: 6,
			err:    types.ErrInvalidParams.Wrapf("invalid tokenFeeder to update, invalid EndBlock, currentHeight: %d, set: %d", 6, 5),
		},
		{
			name: "invalid update, for existing feeder, feeder started, and set endblock to history height including 0",
			tokenFeeder: types.TokenFeeder{
				TokenID: 2,
			},
			height: 11,
			err:    types.ErrInvalidParams.Wrapf("invalid tokenFeeder to update, invalid EndBlock, currentHeight: %d, set: %d", 11, 0),
		},
		{
			name: "invalid update, add a new feeder, set endblock>0 but <=startbasedBlock",
			tokenFeeder: types.TokenFeeder{
				TokenID:        3,
				StartBaseBlock: 10,
				StartRoundID:   1,
				Interval:       10,
				EndBlock:       9,
			},
			height: 1,
			err:    types.ErrInvalidParams.Wrapf("invalid TokenFeeder, invalid EndBlock to be set, startBaseBlock: %d, endBlock: %d", 10, 9),
		},
		{
			name: "invalid update, resume a stopped feeder, set startroundID incorrectly",
			tokenFeeder: types.TokenFeeder{
				TokenID:        2,
				RuleID:         1,
				StartRoundID:   3, // should be 4
				StartBaseBlock: 51,
				Interval:       10,
				EndBlock:       0,
			},
			height: 50,
			err:    types.ErrInvalidParams.Wrapf("invalid tokenFeeder to update, invalid StartBaseBlock or StartRoundID, currentHeight:%d, set_startBasedBlock:%d, set_StartRoundID:%d", 50, 51, 3),
		},
		{
			name: "invalid update, resume a stopped feeder, set startBasedBlock in history",
			tokenFeeder: types.TokenFeeder{
				TokenID:        2,
				RuleID:         1,
				StartRoundID:   4, // should be 4
				StartBaseBlock: 50,
				Interval:       10,
				EndBlock:       0,
			},
			height: 51,
			err:    types.ErrInvalidParams.Wrapf("invalid tokenFeeder to update, invalid StartBaseBlock or StartRoundID, currentHeight:%d, set_startBasedBlock:%d, set_StartRoundID:%d", 50, 51, 3),
		},

		// valid inputs
		{
			name: "valid update, new feeder",
			tokenFeeder: types.TokenFeeder{
				TokenID:        3,
				StartBaseBlock: 10,
				StartRoundID:   1,
				Interval:       10,
				EndBlock:       19,
			},
			height: 1,
			err:    nil,
		},
		{
			name: "valid update, resume a stopped feeder",
			tokenFeeder: types.TokenFeeder{
				TokenID:        2,
				RuleID:         1,
				StartRoundID:   4, // should be 4
				StartBaseBlock: 51,
				Interval:       10,
				EndBlock:       0,
			},
			height: 50,
			err:    nil,
		},
	}
	p := testdata.DefaultParamsForTest()
	p.Tokens = append(p.Tokens, &types.Token{
		Name:            "TEST",
		ChainID:         1,
		ContractAddress: "0x",
		Decimal:         8,
		Active:          true,
		AssetID:         "",
	})
	p.Tokens = append(p.Tokens, &types.Token{
		Name:            "TEST_NEW",
		ChainID:         1,
		ContractAddress: "0x",
		Decimal:         8,
		Active:          true,
		AssetID:         "",
	})
	p.TokenFeeders = append(p.TokenFeeders, &types.TokenFeeder{
		TokenID:        2,
		RuleID:         1,
		StartRoundID:   1,
		StartBaseBlock: 10,
		Interval:       10,
		EndBlock:       38,
	})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := p.UpdateTokenFeeder(&tt.tokenFeeder, tt.height)
			if err == nil {
				err = p.Validate()
			}
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestParamsValidate(t *testing.T) {
	p := testdata.DefaultParamsForTest()
	p.MaxSizePrices = 0
	err := p.Validate()
	require.ErrorIs(t, err, types.ErrInvalidParams.Wrap("invalid MaxSizePrices"))
}

func TestTokenFeederValidate(t *testing.T) {
	p := testdata.DefaultParamsForTest()
	cases := []struct {
		name         string
		prevEndBlock uint64
		feeder       *types.TokenFeeder
		err          error
	}{
		{
			name:         "invalid case, endblock==0",
			prevEndBlock: 0,
			feeder: &types.TokenFeeder{
				TokenID:        1,
				RuleID:         1,
				StartRoundID:   1,
				StartBaseBlock: 1000000,
				Interval:       10,
			},
			err: types.ErrInvalidParams.Wrap("invalid tokenFeeder, for the same token at most one running or future tokenFeeder without endblock set"),
		},
		{
			name:         "invalid case, overlap",
			prevEndBlock: 1000015,
			feeder: &types.TokenFeeder{
				TokenID:        1,
				RuleID:         1,
				StartRoundID:   1,
				StartBaseBlock: 1000000,
				Interval:       10,
			},
			err: types.ErrInvalidParams.Wrap("invalid tokenFeeder overlap with previous feeder"),
		},
		{
			name:         "invalid case, roundID not continuous",
			prevEndBlock: 1000015,
			feeder: &types.TokenFeeder{
				TokenID:        1,
				RuleID:         1,
				StartRoundID:   2,
				StartBaseBlock: 1000020,
				Interval:       10,
			},
			err: types.ErrInvalidParams.Wrap("invalid tokenFeeder no continuos roundID"),
		},
		{
			name:         "valid case with two feeders",
			prevEndBlock: 35,
			feeder: &types.TokenFeeder{
				TokenID:        1,
				RuleID:         1,
				StartRoundID:   3,
				StartBaseBlock: 1000020,
				Interval:       10,
			},
			err: nil,
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			p = testdata.DefaultParamsForTest()
			p.TokenFeeders[1].EndBlock = testCase.prevEndBlock
			p.TokenFeeders = append(p.TokenFeeders, testCase.feeder)
			err := p.Validate()
			if testCase.err != nil {
				require.ErrorIs(t, err, testCase.err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
