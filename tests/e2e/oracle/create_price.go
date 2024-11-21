package oracle

import (
	"context"
	"time"

	sdkmath "cosmossdk.io/math"
	avstypes "github.com/ExocoreNetwork/exocore/x/avs/types"
	operatortypes "github.com/ExocoreNetwork/exocore/x/operator/types"
	oracletypes "github.com/ExocoreNetwork/exocore/x/oracle/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const layout = "2006-01-02 15:04:05"

func (s *E2ETestSuite) TestCreatePriceSlashing() {
	kr0 := s.network.Validators[0].ClientCtx.Keyring
	creator0 := sdk.AccAddress(s.network.Validators[0].PubKey.Address())

	kr1 := s.network.Validators[1].ClientCtx.Keyring
	creator1 := sdk.AccAddress(s.network.Validators[1].PubKey.Address())

	kr2 := s.network.Validators[2].ClientCtx.Keyring
	creator2 := sdk.AccAddress(s.network.Validators[2].PubKey.Address())

	kr3 := s.network.Validators[3].ClientCtx.Keyring
	creator3 := sdk.AccAddress(s.network.Validators[3].PubKey.Address())

	//	kr3 := s.network.Validators[2].ClientCtx.Keyring
	//	creator3 := sdk.AccAddress(s.network.Validators[2].PubKey.Address())

	priceTest1R1 := price1.updateTimestamp()
	priceTimeDetID1R1 := priceTest1R1.getPriceTimeDetID("9")
	priceSource1R1 := oracletypes.PriceSource{
		SourceID: 1,
		Prices: []*oracletypes.PriceTimeDetID{
			&priceTimeDetID1R1,
		},
	}

	// case_1. update price to p1 {reporter: v0, v1, v2. miss:v3}
	s.moveToAndCheck(10)
	// send create-price from validator-0
	msg0 := oracletypes.NewMsgCreatePrice(creator0.String(), 1, []*oracletypes.PriceSource{&priceSource1R1}, 10, 1)
	err := s.network.SendTxOracleCreateprice([]sdk.Msg{msg0}, "valconskey0", kr0)
	s.Require().NoError(err)

	// query final price
	_, err = s.network.QueryOracle().LatestPrice(context.Background(), &oracletypes.QueryGetLatestPriceRequest{TokenId: 1})
	errStatus, _ := status.FromError(err)
	s.Require().Equal(codes.NotFound, errStatus.Code())

	s.moveToAndCheck(11)
	// send create-price from validator-1
	msg1 := oracletypes.NewMsgCreatePrice(creator1.String(), 1, []*oracletypes.PriceSource{&priceSource1R1}, 10, 1)
	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg1}, "valconskey1", kr1)
	s.Require().NoError(err)

	// send create-price from validator-2
	msg2 := oracletypes.NewMsgCreatePrice(creator2.String(), 1, []*oracletypes.PriceSource{&priceSource1R1}, 10, 1)
	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg2}, "valconskey2", kr2)
	s.Require().NoError(err)

	// send create-price with 'malicious' price from validator-3
	priceSource1R1.Prices[0].Price = "123"
	msg3 := oracletypes.NewMsgCreatePrice(creator3.String(), 1, []*oracletypes.PriceSource{&priceSource1R1}, 10, 1)
	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg3}, "valconskey3", kr3)
	s.Require().NoError(err)

	s.moveToAndCheck(13)
	// query final price
	res, err := s.network.QueryOracle().LatestPrice(context.Background(), &oracletypes.QueryGetLatestPriceRequest{TokenId: 1})
	s.Require().NoError(err)
	s.Require().Equal(priceTest1R1.getPriceTimeRound(1), res.Price)

	resSigningInfo, err := s.network.QuerySlashing().SigningInfo(context.Background(), &slashingtypes.QuerySigningInfoRequest{ConsAddress: sdk.ConsAddress(s.network.Validators[3].PubKey.Address()).String()})
	s.Require().NoError(err)
	s.Require().True(true, resSigningInfo.ValSigningInfo.JailedUntil.After(time.Now()))
	chainID := avstypes.ChainIDWithoutRevision(s.network.Config.ChainID)
	avsAddr := avstypes.GenerateAVSAddr(chainID)
	resOperatorSlashInfo, err := s.network.QueryOperator().QueryOperatorSlashInfo(context.Background(), &operatortypes.QueryOperatorSlashInfoRequest{OperatorAVSAddress: &operatortypes.OperatorAVSAddress{OperatorAddr: s.network.Validators[3].Address.String(), AvsAddress: avsAddr}})
	s.Require().NoError(err)
	slashProportion, _ := sdkmath.LegacyNewDecFromStr("0.1")
	s.Require().Equal(slashProportion, resOperatorSlashInfo.AllSlashInfo[0].Info.SlashProportion)
}

func (s *E2ETestSuite) moveToAndCheck(height int64) {
	_, err := s.network.WaitForHeightWithTimeout(height, 30*time.Second)
	s.Require().NoError(err)
}

func (s *E2ETestSuite) moveNAndCheck(n int64) {
	for i := int64(0); i < n; i++ {
		err := s.network.WaitForNextBlock()
		s.Require().NoError(err)
	}
}
