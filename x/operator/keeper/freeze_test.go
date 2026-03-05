package keeper_test

import (
	"bytes"
	"sort"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	testutiltx "github.com/imua-xyz/imuachain/testutil/tx"
	operatortypes "github.com/imua-xyz/imuachain/x/operator/types"
)

func (suite *OperatorTestSuite) TestFrozenOperator() {
	commission := stakingtypes.NewCommission(
		sdk.ZeroDec(), sdk.ZeroDec(), sdk.ZeroDec(),
	)
	count := 10
	addrs := make([]sdk.AccAddress, 0, count)
	for i := 0; i < count; i++ {
		addr := sdk.AccAddress(testutiltx.GenerateAddress().Bytes())
		suite.RegisterOperator(
			addr.String(), commission, false,
		)
		// freeze it once successfully
		err := suite.App.OperatorKeeper.FreezeOperator(suite.Ctx, addr)
		suite.Require().NoError(err)
		suite.Require().True(
			suite.App.OperatorKeeper.IsOperatorFrozen(
				suite.Ctx, addr,
			),
		)
		// freeze it again should fail
		err = suite.App.OperatorKeeper.FreezeOperator(suite.Ctx, addr)
		suite.Require().Error(err)
		suite.Require().ErrorIs(err, operatortypes.ErrOperatorAlreadyFrozen)
		addrs = append(addrs, addr)
	}
	sort.SliceStable(
		addrs, func(i, j int) bool {
			return bytes.Compare(addrs[i].Bytes(), addrs[j].Bytes()) < 0
		},
	)
	frozenOperators := suite.App.OperatorKeeper.GetAllFrozenOperators(suite.Ctx)
	suite.Require().Equal(len(addrs), len(frozenOperators))
	for i := range addrs {
		suite.Require().Equal(
			sdk.AccAddress(addrs[i]).String(),
			frozenOperators[i],
		)
	}

}
