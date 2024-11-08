package bank

import (
	"github.com/ExocoreNetwork/exocore/tests/e2e"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (s *E2ETestSuite) TestQeuryBalance() {
	res, err := e2e.QueryNativeCoinBalance(s.network.Validators[0].Address, s.network)
	s.Require().NoError(err)
	s.Require().Equal(sdk.NewCoin(s.network.Config.NativeDenom, s.network.Config.AccountTokens), *res.Balance)
	s.Require().Equal(e2e.NewNativeCoin(s.network.Config.AccountTokens, s.network), *res.Balance)
}
