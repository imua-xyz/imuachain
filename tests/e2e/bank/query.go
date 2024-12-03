package bank

import (
	"github.com/ExocoreNetwork/exocore/tests/e2e"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// TestQueryBalance verifies that the native coin balance query returns the expected
// account balance for validators. It checks that:
// - The balance matches the network configuration
// - The returned coin denomination and amount are correct
// - The balance can be properly parsed into a native coin
func (s *E2ETestSuite) TestQueryBalance() {
	res, err := e2e.QueryNativeCoinBalance(s.network.Validators[0].Address, s.network)
	s.Require().NoError(err)
	s.Require().Equal(sdk.NewCoin(s.network.Config.NativeDenom, s.network.Config.AccountTokens), *res.Balance)
	s.Require().Equal(e2e.NewNativeCoin(s.network.Config.AccountTokens, s.network), *res.Balance)
}
