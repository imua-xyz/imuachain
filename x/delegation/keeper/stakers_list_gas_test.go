package keeper_test

import (
	"fmt"
	"math"

	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	assetstypes "github.com/imua-xyz/imuachain/x/assets/types"
)

// This test is meant to confirm the suspected source of gas fluctuation:
// `AppendStakerForOperator` reads + unmarshals + appends + marshals + writes the full staker list,
// so the KV write size grows with the list length.
func (suite *DelegationTestSuite) TestAppendStakerForOperator_GasScalesWithListSize() {
	suite.basicPrepare()

	_, assetID := assetstypes.GetStakerIDAndAssetIDFromStr(
		suite.clientChainLzID,
		"",
		suite.assetAddr.String(),
	)
	operator := suite.opAccAddr.String()

	lengths := []int{1, 10, 50, 100, 200, 500, 1000, 10000}
	gasUsed := make([]uint64, 0, len(lengths))

	for _, n := range lengths {
		// Prepare: build an existing list of size n.
		for i := 0; i < n; i++ {
			err := suite.App.DelegationKeeper.AppendStakerForOperator(
				suite.Ctx,
				operator,
				assetID,
				fmt.Sprintf("staker-%d", i),
			)
			suite.NoError(err)
		}

		// Reset gas meter to isolate the cost of a single append.
		ctx := suite.Ctx.WithGasMeter(storetypes.NewGasMeter(1_000_000_000_000))

		start := ctx.GasMeter().GasConsumed()
		err := suite.App.DelegationKeeper.AppendStakerForOperator(
			ctx,
			operator,
			assetID,
			fmt.Sprintf("new-staker-%d", n),
		)
		suite.NoError(err)
		delta := ctx.GasMeter().GasConsumed() - start

		suite.T().Logf("AppendStakerForOperator listLen=%d -> gasDelta=%d", n, delta)
		gasUsed = append(gasUsed, delta)

		// Clean up the list so each measurement starts from a controlled list length.
		// (Otherwise later iterations would include the previous iterations' entries.)
		err = suite.App.DelegationKeeper.DeleteStakersListForOperator(suite.Ctx, operator, assetID)
		suite.NoError(err)
	}

	// The gas difference should be less than 100
	for i := 1; i < len(gasUsed); i++ {
		suite.True(
			math.Abs(float64(gasUsed[i]-gasUsed[i-1])) <= 100.0,
			"expected O(1) gas: len=%d gas=%d < len=%d gas=%d",
			lengths[i], gasUsed[i], lengths[i-1], gasUsed[i-1],
		)
	}
	// With the sentinel approach, the gas difference between consecutive lengths is less
	// than 100. However, this does not provide any information about the gas consumption
	// between the lowest and the highest lengths. Hence, we do not assert this.
}
