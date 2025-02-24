package oracle

import (
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/imua-xyz/imuachain/testutil/network"
	oracletypes "github.com/imua-xyz/imuachain/x/oracle/types"
)

// the test cases run with 'devmode' flag, we try to elaborate all cases to check the recovery logic works fine in each scenario
// this could take some time since we will run for many tokenfeeder rounds to cover many cases
//
// comments explain:
//
//	1{1} means the first block includes one valid quote
//	1{1-} means the first block includes on invalid quote which is different with the expected final price(detID, price)
func (s *E2ETestSuite) testRecoveryCases(start int64) {
	// 1.successfully aggregated,
	//   1.1 all prices provided are the same(detID, price, decimal)
	// 1{3}, 2{1}, 3
	s.moveToAndCheck(start)
	// #nosec G115 -- block height is positive
	startUint := uint64(start)
	msg0 := oracletypes.NewMsgCreatePrice(creator0.String(), 1, []*oracletypes.PriceSource{&priceRecovery1}, startUint, 1)
	msg1 := oracletypes.NewMsgCreatePrice(creator1.String(), 1, []*oracletypes.PriceSource{&priceRecovery1}, startUint, 1)
	msg2 := oracletypes.NewMsgCreatePrice(creator2.String(), 1, []*oracletypes.PriceSource{&priceRecovery1}, startUint, 1)
	msg3 := oracletypes.NewMsgCreatePrice(creator3.String(), 1, []*oracletypes.PriceSource{&priceRecovery1}, startUint, 1)

	err := s.network.SendTxOracleCreateprice([]sdk.Msg{msg0}, "valconsKey0", kr0)
	s.Require().NoError(err)

	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg1}, "valconsKey1", kr1)
	s.Require().NoError(err)

	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg2}, "valconsKey2", kr2)
	s.Require().NoError(err)

	s.moveToAndCheck(start + 1)
	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg3}, "valconsKey3", kr3)
	s.Require().NoError(err)

	s.moveToAndCheck(start + 2)
	res, err := s.network.QueryOracle().LatestPrice(ctxWithHeight(start+1), &oracletypes.QueryGetLatestPriceRequest{TokenId: 1})
	s.NoError(err)
	s.Require().Equal(res.Price.Price, priceRecovery1.Prices[0].Price)

	// 1{1}, 2{3}, 3
	// init_start + 10
	start += 10
	startUint = uint64(start)
	msg0_1 := oracletypes.NewMsgCreatePrice(creator0.String(), 1, []*oracletypes.PriceSource{&priceRecovery2}, startUint, 1)
	msg1_1 := oracletypes.NewMsgCreatePrice(creator1.String(), 1, []*oracletypes.PriceSource{&priceRecovery2}, startUint, 1)
	msg2_1 := oracletypes.NewMsgCreatePrice(creator2.String(), 1, []*oracletypes.PriceSource{&priceRecovery2}, startUint, 1)
	msg3_1 := oracletypes.NewMsgCreatePrice(creator3.String(), 1, []*oracletypes.PriceSource{&priceRecovery2}, startUint, 1)

	s.moveToAndCheck(start)

	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg0_1}, "valconsKey0", kr0)
	s.Require().NoError(err)

	s.moveToAndCheck(start + 1)

	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg1_1}, "valconsKey1", kr1)
	s.Require().NoError(err)

	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg2_1}, "valconsKey2", kr2)
	s.Require().NoError(err)

	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg3_1}, "valconsKey3", kr3)
	s.Require().NoError(err)

	s.moveToAndCheck(start + 3)

	res, err = s.network.QueryOracle().LatestPrice(ctxWithHeight(start+1), &oracletypes.QueryGetLatestPriceRequest{TokenId: 1})
	s.NoError(err)
	// price not updated yet
	s.Require().Equal(res.Price.Price, priceRecovery1.Prices[0].Price)

	res, err = s.network.QueryOracle().LatestPrice(ctxWithHeight(start+2), &oracletypes.QueryGetLatestPriceRequest{TokenId: 1})
	s.NoError(err)
	// price updated from priceRecovery1 to priceRecovery2
	s.Require().Equal(res.Price.Price, priceRecovery2.Prices[0].Price)

	// 1{1}, 2, 3{3}
	// init_start + 20
	start += 10
	startUint = uint64(start)

	s.moveToAndCheck(start)
	msg0.BasedBlock = startUint
	msg1.BasedBlock = startUint
	msg2.BasedBlock = startUint
	msg3.BasedBlock = startUint

	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg0}, "valconsKey0", kr0)
	s.Require().NoError(err)

	s.moveToAndCheck(start + 2)

	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg1}, "valconsKey1", kr1)
	s.Require().NoError(err)

	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg2}, "valconsKey2", kr2)
	s.Require().NoError(err)

	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg3}, "valconsKey3", kr3)
	s.Require().NoError(err)

	s.moveToAndCheck(start + 4)

	res, err = s.network.QueryOracle().LatestPrice(ctxWithHeight(start+2), &oracletypes.QueryGetLatestPriceRequest{TokenId: 1})
	s.NoError(err)
	// price not updated yet
	s.Require().Equal(res.Price.Price, priceRecovery2.Prices[0].Price)

	res, err = s.network.QueryOracle().LatestPrice(ctxWithHeight(start+3), &oracletypes.QueryGetLatestPriceRequest{TokenId: 1})
	s.NoError(err)
	// price updated from priceRecovery2 to priceRecovery1
	s.Require().Equal(res.Price.Price, priceRecovery1.Prices[0].Price)

	// 1{1}, 2{1}, 3{1}
	// init_start + 30
	start += 10
	startUint = uint64(start)
	s.moveToAndCheck(start)
	msg1_1.BasedBlock = startUint
	msg2_1.BasedBlock = startUint
	msg3_1.BasedBlock = startUint

	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg1_1}, "valconsKey1", kr1)
	s.Require().NoError(err)

	s.moveToAndCheck(start + 1)

	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg2_1}, "valconsKey2", kr2)
	s.Require().NoError(err)

	s.moveToAndCheck(start + 2)

	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg3_1}, "valconsKey3", kr3)
	s.Require().NoError(err)

	s.moveToAndCheck(start + 4)

	res, err = s.network.QueryOracle().LatestPrice(ctxWithHeight(start+2), &oracletypes.QueryGetLatestPriceRequest{TokenId: 1})
	s.Require().NoError(err)
	// price not updated yet
	s.Require().Equal(res.Price.Price, priceRecovery1.Prices[0].Price)

	res, err = s.network.QueryOracle().LatestPrice(ctxWithHeight(start+3), &oracletypes.QueryGetLatestPriceRequest{TokenId: 1})
	s.Require().NoError(err)
	// price updated from priceRecovery1 to priceRecovery2
	s.Require().Equal(res.Price.Price, priceRecovery2.Prices[0].Price)

	// 1{1}, 2{1}, 3{2}
	// init_start+40
	start += 10
	startUint = uint64(start)
	s.moveToAndCheck(start)
	msg1.BasedBlock = startUint
	msg2.BasedBlock = startUint
	msg3.BasedBlock = startUint
	msg0.BasedBlock = startUint

	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg1}, "valconsKey1", kr1)
	s.Require().NoError(err)

	s.moveToAndCheck(start + 1)

	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg2}, "valconsKey2", kr2)
	s.Require().NoError(err)

	s.moveToAndCheck(start + 2)

	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg3}, "valconsKey3", kr3)
	s.Require().NoError(err)

	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg0}, "valconsKey0", kr0)
	s.Require().NoError(err)

	s.moveToAndCheck(start + 4)

	res, err = s.network.QueryOracle().LatestPrice(ctxWithHeight(start+2), &oracletypes.QueryGetLatestPriceRequest{TokenId: 1})
	s.Require().NoError(err)
	// price not updated yet
	s.Require().Equal(res.Price.Price, priceRecovery2.Prices[0].Price)

	res, err = s.network.QueryOracle().LatestPrice(ctxWithHeight(start+3), &oracletypes.QueryGetLatestPriceRequest{TokenId: 1})
	s.Require().NoError(err)
	// price updated from priceRecovery2 to priceRecovery1
	s.Require().Equal(res.Price.Price, priceRecovery1.Prices[0].Price)

	// 1{1}, 2{2}, 3{1}
	// init_start+50
	start += 10
	startUint = uint64(start)
	s.moveToAndCheck(start)
	msg1_1.BasedBlock = startUint
	msg2_1.BasedBlock = startUint
	msg3_1.BasedBlock = startUint
	msg0_1.BasedBlock = startUint

	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg1_1}, "valconsKey1", kr1)
	s.Require().NoError(err)

	s.moveToAndCheck(start + 1)

	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg2_1}, "valconsKey2", kr2)
	s.Require().NoError(err)

	s.moveToAndCheck(start + 2)

	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg3_1}, "valconsKey3", kr3)
	s.Require().NoError(err)

	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg0_1}, "valconsKey0", kr0)
	s.Require().NoError(err)

	s.moveToAndCheck(start + 4)

	res, err = s.network.QueryOracle().LatestPrice(ctxWithHeight(start+2), &oracletypes.QueryGetLatestPriceRequest{TokenId: 1})
	s.Require().NoError(err)
	// price not updated yet
	s.Require().Equal(res.Price.Price, priceRecovery1.Prices[0].Price)

	res, err = s.network.QueryOracle().LatestPrice(ctxWithHeight(start+3), &oracletypes.QueryGetLatestPriceRequest{TokenId: 1})
	s.Require().NoError(err)
	// price updated from priceRecovery1 to priceRecovery2
	s.Require().Equal(res.Price.Price, priceRecovery2.Prices[0].Price)

	// 1{1}, 2{2}, 3{1}
	// init_start+60
	start += 10
	startUint = uint64(start)
	s.moveToAndCheck(start)
	msg1.BasedBlock = startUint
	msg2.BasedBlock = startUint
	msg0.BasedBlock = startUint
	msg3.BasedBlock = startUint

	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg1}, "valconsKey1", kr1)
	s.Require().NoError(err)

	s.moveToAndCheck(start + 1)

	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg2}, "valconsKey2", kr2)
	s.Require().NoError(err)

	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg0}, "valconsKey0", kr0)
	s.Require().NoError(err)

	s.moveToAndCheck(start + 2)

	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg3}, "valconsKey3", kr3)
	s.Require().NoError(err)

	s.moveToAndCheck(start + 3)

	res, err = s.network.QueryOracle().LatestPrice(ctxWithHeight(start+1), &oracletypes.QueryGetLatestPriceRequest{TokenId: 1})
	s.Require().NoError(err)
	// price not updated yet
	s.Require().Equal(res.Price.Price, priceRecovery2.Prices[0].Price)

	res, err = s.network.QueryOracle().LatestPrice(ctxWithHeight(start+2), &oracletypes.QueryGetLatestPriceRequest{TokenId: 1})
	s.Require().NoError(err)
	// price updated from priceRecovery2 to priceRecovery1
	s.Require().Equal(res.Price.Price, priceRecovery1.Prices[0].Price)

	// 1{2}, 2{2}, mixed prices
	// init_start+70
	start += 10
	startUint = uint64(start)
	s.moveToAndCheck(start)
	msg1_2 := oracletypes.NewMsgCreatePrice(creator1.String(), 1, []*oracletypes.PriceSource{&priceRecovery1_3}, startUint, 1)
	msg2_2 := oracletypes.NewMsgCreatePrice(creator2.String(), 1, []*oracletypes.PriceSource{&priceRecovery1_2}, startUint, 1)
	msg3_2 := oracletypes.NewMsgCreatePrice(creator3.String(), 1, []*oracletypes.PriceSource{&priceRecovery3}, startUint, 1)
	//	msg0_2 := oracletypes.NewMsgCreatePrice(creator0.String(), 1, []*oracletypes.PriceSource{&priceRecovery3}, startUint, 1)
	msg0_1.BasedBlock = startUint

	// id:1,2,3
	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg1_2}, "valconsKey1", kr1)
	s.Require().NoError(err)

	// id:3
	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg3_2}, "valconsKey3", kr3)
	s.Require().NoError(err)

	s.moveToAndCheck(start + 1)

	// id:1,2
	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg2_2}, "valconsKey2", kr2)
	s.Require().NoError(err)

	// id:2
	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg0_1}, "valconsKey0", kr0)
	s.Require().NoError(err)

	s.moveToAndCheck(start + 3)

	res, err = s.network.QueryOracle().LatestPrice(ctxWithHeight(start+1), &oracletypes.QueryGetLatestPriceRequest{TokenId: 1})
	s.Require().NoError(err)
	// price not updated yet
	s.Require().Equal(res.Price.Price, priceRecovery1.Prices[0].Price)

	res, err = s.network.QueryOracle().LatestPrice(ctxWithHeight(start+2), &oracletypes.QueryGetLatestPriceRequest{TokenId: 1})
	s.Require().NoError(err)
	// price updated from priceRecovery1 to priceRecovery2
	s.Require().Equal(res.Price.Price, priceRecovery2.Prices[0].Price)

	// 1{2}, 2, 3{2}
	// init_start+80
	start += 10
	startUint = uint64(start)
	s.moveToAndCheck(start)
	msg3_2.BasedBlock = startUint
	msg1_2.BasedBlock = startUint
	msg2_2.BasedBlock = startUint
	msg0.BasedBlock = startUint
	// id:3
	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg3_2}, "valconsKey3", kr3)
	s.Require().NoError(err)

	// id:1,2,3
	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg1_2}, "valconsKey1", kr1)
	s.Require().NoError(err)

	s.moveToAndCheck(start + 2)

	// id:1,2
	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg2_2}, "valconsKey2", kr2)
	s.Require().NoError(err)

	// id:1
	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg0}, "valconsKey0", kr0)
	s.Require().NoError(err)

	s.moveToAndCheck(start + 4)

	res, err = s.network.QueryOracle().LatestPrice(ctxWithHeight(start+1), &oracletypes.QueryGetLatestPriceRequest{TokenId: 1})
	s.Require().NoError(err)
	// price not updated yet
	s.Require().Equal(res.Price.Price, priceRecovery2.Prices[0].Price)

	res, err = s.network.QueryOracle().LatestPrice(ctxWithHeight(start+3), &oracletypes.QueryGetLatestPriceRequest{TokenId: 1})
	s.Require().NoError(err)
	// price updated from priceRecovery2 to priceRecovery1
	s.Require().Equal(res.Price.Price, priceRecovery1.Prices[0].Price)

	// 1{2}, 2{2}, 3. mixed prices
	// init_start+90
	start += 10
	startUint = uint64(start)
	s.moveToAndCheck(start)
	msg3_2.BasedBlock = startUint
	msg1_2.BasedBlock = startUint
	msg2_2.BasedBlock = startUint
	msg0_2 := oracletypes.NewMsgCreatePrice(creator0.String(), 1, []*oracletypes.PriceSource{&priceRecovery3}, startUint, 1)
	// msg0_2.BasedBlock = startUint
	// id:3
	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg3_2}, "valconsKey3", kr3)
	s.Require().NoError(err)

	// id:1,2,3
	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg1_2}, "valconsKey1", kr1)
	s.Require().NoError(err)

	s.moveToAndCheck(start + 1)

	// id:1,2
	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg2_2}, "valconsKey2", kr2)
	s.Require().NoError(err)

	// id:3
	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg0_2}, "valconsKey0", kr0)
	s.Require().NoError(err)

	s.moveToAndCheck(start + 3)

	res, err = s.network.QueryOracle().LatestPrice(ctxWithHeight(start+1), &oracletypes.QueryGetLatestPriceRequest{TokenId: 1})
	s.Require().NoError(err)
	// price not updated yet
	s.Require().Equal(res.Price.Price, priceRecovery1.Prices[0].Price)

	res, err = s.network.QueryOracle().LatestPrice(ctxWithHeight(start+2), &oracletypes.QueryGetLatestPriceRequest{TokenId: 1})
	s.Require().NoError(err)
	// price updated from priceRecovery1 to priceRecovery3
	s.Require().Equal(res.Price.Price, priceRecovery3.Prices[0].Price)

	// 2.failed to aggregate
	//    2.1 all prices provided are the same(detID, price, decimal), failed for not enough power
	// init_start+100
	start += 10
	startUint = uint64(start)
	s.moveToAndCheck(start)
	msg3.BasedBlock = startUint
	msg1.BasedBlock = startUint
	msg2_1.BasedBlock = startUint
	msg0_1.BasedBlock = startUint

	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg3}, "valconsKey3", kr3)
	s.Require().NoError(err)

	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg1}, "valconsKey1", kr1)
	s.Require().NoError(err)

	s.moveToAndCheck(start + 1)

	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg2_1}, "valconsKey2", kr2)
	s.Require().NoError(err)

	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg0_1}, "valconsKey0", kr0)
	s.Require().NoError(err)

	s.moveToAndCheck(start + 4)

	res, err = s.network.QueryOracle().LatestPrice(ctxWithHeight(start+3), &oracletypes.QueryGetLatestPriceRequest{TokenId: 1})
	s.Require().NoError(err)
	// price not updated yet
	s.Require().Equal(res.Price.Price, priceRecovery3.Prices[0].Price)

	//    2.2 mixed with some different prices(detID, price)
	// init_start+110
	start += 10
	startUint = uint64(start)
	s.moveToAndCheck(start)
	msg3_2.BasedBlock = startUint
	msg1_2.BasedBlock = startUint
	msg2_2.BasedBlock = startUint
	msg0_2.BasedBlock = startUint
	// msg0_2.BasedBlock = startUint
	// id:3
	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg3_2}, "valconsKey3", kr3)
	s.Require().NoError(err)

	// id:1,2,3
	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg1_2}, "valconsKey1", kr1)
	s.Require().NoError(err)

	s.moveToAndCheck(start + 1)

	// id:1,2
	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg2_2}, "valconsKey2", kr2)
	s.Require().NoError(err)

	// id:3
	//	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg0_2}, "valconsKey0", kr0)
	//	s.Require().NoError(err)

	s.moveToAndCheck(start + 3)

	res, err = s.network.QueryOracle().LatestPrice(ctxWithHeight(start+2), &oracletypes.QueryGetLatestPriceRequest{TokenId: 1})
	s.Require().NoError(err)
	// price not updated yet
	s.Require().Equal(res.Price.Price, priceRecovery3.Prices[0].Price)

	//    2.3 failed for forceSeal by paramsUpdate
	// TODO: for now all paramsUpdate related forceSeal are not supported (the related fields are not allowed to be updated by msgUpdateParms)
	// we comment out this case for now
	//	start += 10
	//	startUint = uint64(start)
	//	msg0.BasedBlock = startUint
	//	msg1.BasedBlock = startUint
	//	msg2.BasedBlock = startUint
	//	msgUpdateParams := oracletypes.NewMsgUpdateParams("creator", `{"max_nonce":5}`)
	//	s.moveNAndCheck(start)
	//
	//	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg0}, "valconsKey0", kr0)
	//	s.Require().NoError(err)
	//
	//	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg1}, "valconsKey1", kr1)
	//	s.Require().NoError(err)
	//
	//	// send updateParams msg to forceSeal current round
	//	err = s.network.SendTx([]sdk.Msg{msgUpdateParams}, s.network.Validators[0].ClientCtx.FromName, kr0)
	//	s.Require().NoError(err)
	//	s.moveToAndCheck(start + 1)
	//
	//	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg2}, "valconsKey2", kr2)
	//	s.Require().NoError(err)
	//
	//	s.moveToAndCheck(start + 3)
	//
	//	res, err = s.network.QueryOracle().LatestPrice(ctxWithHeight(start+2), &oracletypes.QueryGetLatestPriceRequest{TokenId: 1})
	//	s.Require().NoError(err)
	//	// price failed to update
	//	s.Require().Equal(res.Price.Price, priceRecovery3.Prices[0].Price)

	//    2.4 failed for forceSeal by validatorSetUpdate: we use an old timestamp in genesisfile to setup the network so that the epoch end will be triggered on each block
	start += 10
	startUint = uint64(start)

	msg0.BasedBlock = startUint
	msg1.BasedBlock = startUint
	msg2.BasedBlock = startUint
	//		msgUpdateParams := oracletypes.NewMsgUpdateParams(s.network.Validators[0].Address.String(), `{"max_nonce":5}`)
	s.moveToAndCheck(start)

	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg0}, "valconsKey0", kr0)
	s.Require().NoError(err)

	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg1}, "valconsKey1", kr1)
	s.Require().NoError(err)

	// delegate to change validator set, we set genesis time to a history time so that the validator set update will be triggered every block
	clientChainID := uint32(101)
	lzNonce := uint64(0)
	assetAddr, _ := hexutil.Decode(network.ETHAssetAddress)
	stakerAddr := []byte(s.network.Validators[0].Address)
	operatorAddr := []byte(s.network.Validators[0].Address.String())
	opAmount := big.NewInt(90000000)
	// deposit 32 NSTETH to staker from beaconchain_validatro_1
	err = s.network.SendPrecompileTx(network.DELEGATION, "delegate", clientChainID, lzNonce, assetAddr, stakerAddr, operatorAddr, opAmount)
	s.Require().NoError(err)

	// power will be updated at endBlock of start+2, it would force seal this round
	s.moveToAndCheck(start + 2)

	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg2}, "valconsKey2", kr2)
	s.Require().NotNil(err)

	s.moveToAndCheck(start + 3)

	res, err = s.network.QueryOracle().LatestPrice(ctxWithHeight(start+2), &oracletypes.QueryGetLatestPriceRequest{TokenId: 1})
	s.NoError(err)
	s.Require().Equal(res.Price.Price, priceRecovery3.Prices[0].Price)

	s.moveToAndCheck(start + 20)
}
