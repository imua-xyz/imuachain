package oracle

import (
	"time"

	oracletypes "github.com/ExocoreNetwork/exocore/x/oracle/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const layout = "2006-01-02 15:04:05"

func (s *E2ETestSuite) TestCreatePriceLST() {
	kr := s.network.Validators[0].ClientCtx.Keyring
	s.network.WaitForHeight(10)
	creator := sdk.AccAddress(s.network.Validators[0].PubKey.Address())
	priceSource := oracletypes.PriceSource{
		SourceID: 1,
		Prices: []*oracletypes.PriceTimeDetID{
			{
				Price:     "999123",
				Decimal:   3,
				Timestamp: time.Now().UTC().Format(layout),
				DetID:     "9",
			},
		},
	}
	msg := oracletypes.NewMsgCreatePrice(creator.String(), 1, []*oracletypes.PriceSource{&priceSource}, 10, 1)
	s.network.SendTx([]sdk.Msg{msg}, "validator0", kr)

	s.network.WaitForHeight(12)
}

func (s *E2ETestSuite) TestCreatePriceNST() {

}

func (s *E2ETestSuite) TestSlashing() {

}
