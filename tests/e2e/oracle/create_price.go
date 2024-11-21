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

/*
	cases:
	  we need more than 2/3 power, so that at least 3 out of 4 validators power should be enough
		1. block_1_1: v1 sendPrice{p1}, [no round_1 price after block_1_1 committed], block_1_2:v2&v3 sendPrice{p1}, [got round_1 price{p1} after block_1_2 committed]
		2. block_2_1: v3 sendPrice{p2}, block_2_2: v1 sendPrice{p2}, [no round_2 price after block_2_2 committed], block_2_3:nothing, [got round_2 price{p1} equals to round_1 after block_2_3 committed]
		3. block_3_1: v1 sendPrice{p1}, block_3_2: v2&v3 sendPrice{p2}, block_3_3: v3 sendPrice{p2}, [got final price{p2} after block_3_3 committed]
		4. block_4_1: v1&v2&v3 sendPrice{p1}, [got round_4 price{p1} after block_4_1 committed]]

		--- nonce:
*/

func (s *E2ETestSuite) TestCreatePriceLST() {
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
	return
	// case_2. failed to update price to p2, keep p1
	// timestamp need to be updated
	priceTest2R2 := price2.updateTimestamp()
	priceTimeDetID2R2 := priceTest2R2.getPriceTimeDetID("10")
	priceSource2R2 := oracletypes.PriceSource{
		SourceID: 1,
		Prices: []*oracletypes.PriceTimeDetID{
			&priceTimeDetID2R2,
		},
	}
	msg0 = oracletypes.NewMsgCreatePrice(creator0.String(), 1, []*oracletypes.PriceSource{&priceSource2R2}, 20, 1)
	msg2 = oracletypes.NewMsgCreatePrice(creator2.String(), 1, []*oracletypes.PriceSource{&priceSource2R2}, 20, 1)

	s.moveToAndCheck(20)
	// send price{p2} from validator-2
	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg2}, "valconskey2", kr2)
	s.Require().NoError(err)
	s.moveToAndCheck(21)
	// send price{p2} from validator-0
	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg0}, "valconskey0", kr0)
	s.Require().NoError(err)
	s.moveToAndCheck(24)
	res, err = s.network.QueryOracle().LatestPrice(context.Background(), &oracletypes.QueryGetLatestPriceRequest{TokenId: 1})
	s.Require().NoError(err)
	// price update fail, round 2 still have price{p1}
	s.Require().Equal(priceTest1R1.getPriceTimeRound(2), res.Price)

	// case_3. update price to p2{reporter:v0,v1,v2, miss:v3}, v3 should be slash for now(require 1 report at least, but got 0)
	// update timestamp
	priceTest2R3 := price2.updateTimestamp()
	priceTimeDetID2R3 := priceTest2R3.getPriceTimeDetID("11")
	priceSource2R3 := oracletypes.PriceSource{
		SourceID: 1,
		Prices: []*oracletypes.PriceTimeDetID{
			&priceTimeDetID2R3,
		},
	}

	msg0 = oracletypes.NewMsgCreatePrice(creator0.String(), 1, []*oracletypes.PriceSource{&priceSource2R3}, 30, 1)
	msg1 = oracletypes.NewMsgCreatePrice(creator1.String(), 1, []*oracletypes.PriceSource{&priceSource2R3}, 30, 1)
	msg2 = oracletypes.NewMsgCreatePrice(creator2.String(), 1, []*oracletypes.PriceSource{&priceSource2R3}, 30, 1)
	s.moveToAndCheck(30)
	// send price{p2} from validator-0
	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg0}, "valconskey0", kr0)
	s.Require().NoError(err)
	s.moveToAndCheck(31)
	// send price{p2} from validator-1
	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg1}, "valconskey1", kr1)
	s.Require().NoError(err)

	// send price{p2} from validator-2
	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg2}, "valconskey2", kr2)
	s.Require().NoError(err)

	s.moveToAndCheck(33)
	res, err = s.network.QueryOracle().LatestPrice(context.Background(), &oracletypes.QueryGetLatestPriceRequest{TokenId: 1})
	s.Require().NoError(err)
	// price updated, round 3 has price{p2}
	s.Require().Equal(priceTest2R3.getPriceTimeRound(3), res.Price)
	// case_slahsing: validator 3 is jailed
	resSigningInfo, err = s.network.QuerySlashing().SigningInfo(context.Background(), &slashingtypes.QuerySigningInfoRequest{ConsAddress: sdk.ConsAddress(s.network.Validators[3].PubKey.Address()).String()})
	s.Require().NoError(err)
	s.Require().True(true, resSigningInfo.ValSigningInfo.JailedUntil.After(time.Now()))
	// chainID := avstypes.ChainIDWithoutRevision(s.network.Config.ChainID)
	// avsAddr := avstypes.GenerateAVSAddr(chainID)
	// resOperatorSlashInfo, err := s.network.QueryOperator().QueryOperatorSlashInfo(context.Background(), &operatortypes.QueryOperatorSlashInfoRequest{OperatorAVSAddress: &operatortypes.OperatorAVSAddress{OperatorAddr: s.network.Validators[3].Address.String(), AvsAddress: avsAddr}})
	// s.Require().NoError(err)
	// slashProportion, _ := sdkmath.LegacyNewDecFromStr("0.05")
	// s.Require().Equal(slashProportion, resOperatorSlashInfo.AllSlashInfo[0].Info.SlashProportion)

	// case_4. update price to p1{reporter:v0,v1,v2, miss:v3}
	s.moveToAndCheck(40)
	priceTest1R4, priceSource1R4 := price1.generateRealTimeStructs("12", 1)
	msg0 = oracletypes.NewMsgCreatePrice(creator0.String(), 1, []*oracletypes.PriceSource{&priceSource1R4}, 40, 1)
	msg1 = oracletypes.NewMsgCreatePrice(creator1.String(), 1, []*oracletypes.PriceSource{&priceSource1R4}, 40, 1)
	msg2 = oracletypes.NewMsgCreatePrice(creator2.String(), 1, []*oracletypes.PriceSource{&priceSource1R4}, 40, 1)
	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg0}, "valconskey0", kr0)
	s.Require().NoError(err)
	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg1}, "valconskey1", kr1)
	s.Require().NoError(err)
	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg2}, "valconskey2", kr2)
	s.Require().NoError(err)
	s.moveToAndCheck(42)
	res, err = s.network.QueryOracle().LatestPrice(context.Background(), &oracletypes.QueryGetLatestPriceRequest{TokenId: 1})
	s.Require().NoError(err)
	// price updated, round 4 has price{p1}
	s.Require().Equal(priceTest1R4.getPriceTimeRound(4), res.Price)
}

func (s *E2ETestSuite) TestCreatePriceNST() {} //nolint:unused

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
