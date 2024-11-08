package bank

import (
	sdkmath "cosmossdk.io/math"
	"github.com/ExocoreNetwork/exocore/tests/e2e"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

func (s *E2ETestSuite) TestSendCoin() {
	kr := s.network.Validators[0].ClientCtx.Keyring
	// generate a new account with ethsecp256k1 to recieve/send native coins (hua)
	toAddr, err := e2e.GenerateAccAddress(kr, "user1")
	s.Require().NoError(err)
	// generate sendCoin msg
	fromAddr := s.network.Validators[0].Address
	msg := banktypes.NewMsgSend(fromAddr, toAddr, sdk.NewCoins(sdk.NewCoin(s.network.Config.NativeDenom, sdkmath.NewInt(2000000))))

	// send sendCoinMsg
	err = s.network.SendTx([]sdk.Msg{msg}, s.network.Validators[0].ClientCtx.FromName, kr)
	s.Require().NoError(err)

	// wait to next block for tx to be included
	err = s.network.WaitForNextBlock()
	s.Require().NoError(err)

	// check user1's balance
	res, err := e2e.QueryNativeCoinBalance(toAddr, s.network)
	s.Require().NoError(err)
	s.Require().Equal(e2e.NewNativeCoin(sdkmath.NewInt(2000000), s.network), *res.Balance)

	toAddr2, _ := e2e.GenerateAccAddress(kr, "user2")

	msg = banktypes.NewMsgSend(toAddr, toAddr2, sdk.NewCoins(sdk.NewCoin(s.network.Config.NativeDenom, sdkmath.NewInt(100))))
	// send sendCoinMsg
	err = s.network.SendTx([]sdk.Msg{msg}, "user1", kr)
	s.Require().NoError(err)

	// wait to next block for tx to be included
	err = s.network.WaitForNextBlock()
	s.Require().NoError(err)

	// check user2's balance
	res, err = e2e.QueryNativeCoinBalance(toAddr2, s.network)
	s.Require().NoError(err)
	s.Require().Equal(e2e.NewNativeCoin(sdkmath.NewInt(100), s.network), *res.Balance)
}
