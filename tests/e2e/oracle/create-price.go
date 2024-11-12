package oracle

import (
	"context"

	oracletypes "github.com/ExocoreNetwork/exocore/x/oracle/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const layout = "2006-01-02 15:04:05"

func (s *E2ETestSuite) TestCreatePriceLST() {
	kr := s.network.Validators[0].ClientCtx.Keyring
	creator := sdk.AccAddress(s.network.Validators[0].PubKey.Address())
	priceTest1 := price1.updateTimestamp()
	priceTimeDetID := priceTest1.getPriceTimeDetID("9")
	priceSource := oracletypes.PriceSource{
		SourceID: 1,
		Prices: []*oracletypes.PriceTimeDetID{
			&priceTimeDetID,
		},
	}
	msg := oracletypes.NewMsgCreatePrice(creator.String(), 1, []*oracletypes.PriceSource{&priceSource}, 10, 1)

	s.moveToAndCheck(10)

	// send create-price from validator-0
	err := s.network.SendTxOracleCreateprice([]sdk.Msg{msg}, "valconskey0", kr)
	s.Require().NoError(err)

	// final price not aggregated
	_, err = s.network.QueryOracle().LatestPrice(context.Background(), &oracletypes.QueryGetLatestPriceRequest{TokenId: 1})
	errStatus, _ := status.FromError(err)
	s.Require().Equal(codes.NotFound, errStatus.Code())

	s.moveToAndCheck(11)
	// send create-price from validator-1
	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg}, "valconskey1", kr)

	s.moveToAndCheck(12)

	res, err := s.network.QueryOracle().LatestPrice(context.Background(), &oracletypes.QueryGetLatestPriceRequest{TokenId: 1})
	s.Require().NoError(err)
	s.Require().Equal(priceTest1.getPriceTimeRound(1), res.Price)
}

func (s *E2ETestSuite) TestCreatePriceNST() {

}

func (s *E2ETestSuite) TestSlashing() {

}

func (s *E2ETestSuite) moveToAndCheck(height int64) {
	_, err := s.network.WaitForHeight(height)
	s.Require().NoError(err)
}

func (s *E2ETestSuite) moveNAndCheck(n int64) {
	for i := int64(0); i < n; i++ {
		err := s.network.WaitForNextBlock()
		s.Require().NoError(err)
	}
}
