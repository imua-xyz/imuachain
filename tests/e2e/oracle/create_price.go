package oracle

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"time"

	sdkmath "cosmossdk.io/math"
	"github.com/ExocoreNetwork/exocore/testutil/network"
	assetstypes "github.com/ExocoreNetwork/exocore/x/assets/types"
	avstypes "github.com/ExocoreNetwork/exocore/x/avs/types"
	operatortypes "github.com/ExocoreNetwork/exocore/x/operator/types"
	oracletypes "github.com/ExocoreNetwork/exocore/x/oracle/types"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var (
	kr0, kr1, kr2, kr3                     keyring.Keyring
	creator0, creator1, creator2, creator3 sdk.AccAddress
)

// TestCreatePrice run test cases for oracle module including related workflow from other module(assets)
// create-price for LST
// create-price for NST
// registerToken automatically through assets module when precompiled is called
// slashing for downtime
// slashing for malicious price
func (s *E2ETestSuite) TestCreatePrice() {
	kr0 = s.network.Validators[0].ClientCtx.Keyring
	creator0 = sdk.AccAddress(s.network.Validators[0].PubKey.Address())

	kr1 = s.network.Validators[1].ClientCtx.Keyring
	creator1 = sdk.AccAddress(s.network.Validators[1].PubKey.Address())

	kr2 = s.network.Validators[2].ClientCtx.Keyring
	creator2 = sdk.AccAddress(s.network.Validators[2].PubKey.Address())

	kr3 = s.network.Validators[3].ClientCtx.Keyring
	creator3 = sdk.AccAddress(s.network.Validators[3].PubKey.Address())

	// we combine all test cases into one big case to avoid reset the network multiple times, the order can't be changed

	option := os.Getenv("TEST_OPTION")
	if option == "local" {
		s.testRecoveryCases(10)
	} else {
		s.testRegisterTokenThroughPrecompile()
		s.testCreatePriceNST()
		s.testCreatePriceLST()
		s.testSlashing()
		s.testCreatePriceLSTAfterDelegationChangePower()
	}
}

func (s *E2ETestSuite) testCreatePriceLSTAfterDelegationChangePower() {
	s.moveToAndCheck(80)
	priceTest1R1 := price2.updateTimestamp()
	priceTimeDetID1R1 := priceTest1R1.getPriceTimeDetID("9")
	priceSource1R1 := oracletypes.PriceSource{
		SourceID: 1,
		Prices: []*oracletypes.PriceTimeDetID{
			&priceTimeDetID1R1,
		},
	}

	// send create-price from validator-0
	msg0 := oracletypes.NewMsgCreatePrice(creator0.String(), 1, []*oracletypes.PriceSource{&priceSource1R1}, 80, 1)
	err := s.network.SendTxOracleCreateprice([]sdk.Msg{msg0}, "valconskey0", kr0)
	s.Require().NoError(err)

	// send create-price from validator-1
	msg1 := oracletypes.NewMsgCreatePrice(creator1.String(), 1, []*oracletypes.PriceSource{&priceSource1R1}, 80, 1)
	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg1}, "valconskey1", kr1)
	s.Require().NoError(err)

	s.moveToAndCheck(82)
	res, err := s.network.QueryOracle().LatestPrice(ctxWithHeight(81), &oracletypes.QueryGetLatestPriceRequest{TokenId: 1})
	s.NoError(err)
	s.Require().Equal(res.Price.Price, price1.Price)

	s.moveToAndCheck(85)
	clientChainID := uint32(101)
	lzNonce := uint64(0)
	assetAddr, _ := hexutil.Decode(network.ETHAssetAddress)
	stakerAddr := []byte(s.network.Validators[0].Address)
	operatorAddr := []byte(s.network.Validators[0].Address.String())
	opAmount := big.NewInt(90000000)
	// deposit 32 NSTETH to staker from beaconchain_validatro_1
	err = s.network.SendPrecompileTx(network.DELEGATION, "delegate", clientChainID, lzNonce, assetAddr, stakerAddr, operatorAddr, opAmount)
	s.Require().NoError(err)

	// wait for validator set update
	s.moveToAndCheck(120)

	// send create-price from validator-0
	msg0 = oracletypes.NewMsgCreatePrice(creator0.String(), 1, []*oracletypes.PriceSource{&priceSource1R1}, 120, 1)
	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg0}, "valconskey0", kr0)
	s.Require().NoError(err)

	// send create-price from validator-1
	msg1 = oracletypes.NewMsgCreatePrice(creator1.String(), 1, []*oracletypes.PriceSource{&priceSource1R1}, 120, 1)
	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg1}, "valconskey1", kr1)
	s.Require().NoError(err)

	s.moveToAndCheck(122)
	// query final price. query state of 11 on height 12
	res, err = s.network.QueryOracle().LatestPrice(ctxWithHeight(121), &oracletypes.QueryGetLatestPriceRequest{TokenId: 1})
	s.Require().NoError(err)

	ret := priceTest1R1.getPriceTimeRound(12)
	ret.Timestamp = res.Price.Timestamp
	s.Require().Equal(ret, res.Price)
}

/*
cases:

we need more than 2/3 power, so that at least 3 out of 4 validators power should be enough
1. block_1_1: v1 sendPrice{p1}, [no round_1 price after block_1_1 committed], block_1_2:v2&v3 sendPrice{p1}, [got round_1 price{p1} after block_1_2 committed]
2. block_2_1: v3 sendPrice{p2}, block_2_2: v1 sendPrice{p2}, [no round_2 price after block_2_2 committed], block_2_3:nothing, [got round_2 price{p1} equals to round_1 after block_2_3 committed]
3. block_3_1: v1 sendPrice{p1}, block_3_2: v2&v3 sendPrice{p2}, block_3_3: v3 sendPrice{p2}, [got final price{p2} after block_3_3 committed]
4. block_4_1: v1&v2&v3 sendPrice{p1}, [got round_4 price{p1} after block_4_1 committed]]

--- nonce:
*/
func (s *E2ETestSuite) testCreatePriceLST() {
	priceTest1R1 := price1.updateTimestamp()
	priceTimeDetID1R1 := priceTest1R1.getPriceTimeDetID("9")
	priceSource1R1 := oracletypes.PriceSource{
		SourceID: 1,
		Prices: []*oracletypes.PriceTimeDetID{
			&priceTimeDetID1R1,
		},
	}

	// case_1. slashing_{miss_v3:1, window:2} [1.0]
	s.moveToAndCheck(10)

	// send create-price from validator-0
	msg0 := oracletypes.NewMsgCreatePrice(creator0.String(), 1, []*oracletypes.PriceSource{&priceSource1R1}, 10, 1)
	err := s.network.SendTxOracleCreateprice([]sdk.Msg{msg0}, "valconskey0", kr0)
	s.Require().NoError(err)

	s.moveToAndCheck(11)

	// send create-price from validator-1
	msg1 := oracletypes.NewMsgCreatePrice(creator1.String(), 1, []*oracletypes.PriceSource{&priceSource1R1}, 10, 1)
	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg1}, "valconskey1", kr1)
	s.Require().NoError(err)

	// send create-price from validator-2
	msg2 := oracletypes.NewMsgCreatePrice(creator2.String(), 1, []*oracletypes.PriceSource{&priceSource1R1}, 10, 1)
	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg2}, "valconskey2", kr2)
	s.Require().NoError(err)

	s.moveToAndCheck(12)

	// TODO: there might be a small chance that the blockHeight grows to more than 13, try bigger price window(nonce>3) to be more confident
	// send create-price from validator3 to avoid being slashed for downtime
	msg3 := oracletypes.NewMsgCreatePrice(creator3.String(), 1, []*oracletypes.PriceSource{&priceSource1R1}, 10, 1)
	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg3}, "valconskey3", kr3)
	s.Require().NoError(err)

	// query final price. query state of 11 on height 12
	_, err = s.network.QueryOracle().LatestPrice(ctxWithHeight(11), &oracletypes.QueryGetLatestPriceRequest{TokenId: 1})
	errStatus, _ := status.FromError(err)
	s.Require().Equal(codes.NotFound, errStatus.Code())

	s.moveToAndCheck(13)
	// query final price. query state of 12 on height 13
	res, err := s.network.QueryOracle().LatestPrice(ctxWithHeight(12), &oracletypes.QueryGetLatestPriceRequest{TokenId: 1})
	s.Require().NoError(err)
	// NOTE: update timestamp manually to ignore
	ret := priceTest1R1.getPriceTimeRound(1)
	ret.Timestamp = res.Price.Timestamp
	s.Require().Equal(ret, res.Price)

	// case_2. slashing{miss_v3:1, window:2} [1.0]
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
	res, err = s.network.QueryOracle().LatestPrice(ctxWithHeight(23), &oracletypes.QueryGetLatestPriceRequest{TokenId: 1})
	s.Require().NoError(err)
	// price update fail, round 2 still have price{p1}
	// NOTE: update timestamp manually to ignore
	ret = priceTest1R1.getPriceTimeRound(2)
	ret.Timestamp = res.Price.Timestamp
	s.Require().Equal(ret, res.Price)
	// case_3.  slashing_{miss_v3:2, window:3} [1.0.1]
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
	res, err = s.network.QueryOracle().LatestPrice(ctxWithHeight(32), &oracletypes.QueryGetLatestPriceRequest{TokenId: 1})
	s.Require().NoError(err)
	// price updated, round 3 has price{p2}
	// NOTE: update timestamp manually to ignore
	ret = priceTest2R3.getPriceTimeRound(3)
	ret.Timestamp = res.Price.Timestamp
	s.Require().Equal(ret, res.Price)

	// case_4. slashing_{miss_v3:2, window:4}.maxWindow=4 [1.0.1.0]
	// update timestamp
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

	s.moveToAndCheck(41)
	// send create-price from validator3 to avoid being slashed for downtime
	msg3 = oracletypes.NewMsgCreatePrice(creator3.String(), 1, []*oracletypes.PriceSource{&priceSource1R4}, 40, 1)
	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg3}, "valconskey3", kr3)
	s.Require().NoError(err)

	s.moveToAndCheck(42)

	res, err = s.network.QueryOracle().LatestPrice(ctxWithHeight(41), &oracletypes.QueryGetLatestPriceRequest{TokenId: 1})
	s.Require().NoError(err)
	// price updated, round 4 has price{p1}
	// NOTE: update timestamp manually to ignore
	ret = priceTest1R4.getPriceTimeRound(4)
	ret.Timestamp = res.Price.Timestamp
	s.Require().Equal(ret, res.Price)
}

func (s *E2ETestSuite) testCreatePriceNST() {
	clientChainID := uint32(101)
	validatorPubkey := []byte{1}
	// this is just a fake address
	stakerAddrStr := "0x3e108c058e8066da635321dc3018294ca82ccedf"
	stakerAddr, err := hexutil.Decode(stakerAddrStr)
	stakerID := fmt.Sprintf("%s_0x65", stakerAddrStr)
	s.Require().NoError(err)
	// for eth-nst, it should be exactly 32 tokens each time deposit
	opAmount := big.NewInt(32)
	s.moveToAndCheck(5)
	// deposit 32 NSTETH to staker from beaconchain_validatro_1
	err = s.network.SendPrecompileTx(network.ASSETS, "depositNST", clientChainID, validatorPubkey, stakerAddr, opAmount)
	s.Require().NoError(err)
	ctx := context.Background()

	// slashing_{miss_v3:1, window:1} [1]
	s.moveToAndCheck(7)
	_, ps := priceNST1.generateRealTimeStructs("100_1", 1)
	msg0 := oracletypes.NewMsgCreatePrice(creator0.String(), 2, []*oracletypes.PriceSource{&ps}, 7, 1)
	msg1 := oracletypes.NewMsgCreatePrice(creator1.String(), 2, []*oracletypes.PriceSource{&ps}, 7, 1)
	msg2 := oracletypes.NewMsgCreatePrice(creator2.String(), 2, []*oracletypes.PriceSource{&ps}, 7, 1)
	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg0}, "valconskey0", kr0)
	s.Require().NoError(err)
	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg1}, "valconskey1", kr1)
	s.Require().NoError(err)
	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg2}, "valconskey2", kr2)
	s.Require().NoError(err)

	// on height 7, the state from 6 is committed and confirmed
	res, err := s.network.QueryAssets().QueStakerSpecifiedAssetAmount(ctx, &assetstypes.QuerySpecifiedAssetAmountReq{StakerId: stakerID, AssetId: network.NativeAssetID})
	resStakerList, err2 := s.network.QueryOracle().StakerList(ctx, &oracletypes.QueryStakerListRequest{AssetId: network.NativeAssetID})
	resStakerInfo, err3 := s.network.QueryOracle().StakerInfo(ctx, &oracletypes.QueryStakerInfoRequest{AssetId: network.NativeAssetID, StakerAddr: stakerAddrStr})
	s.Require().NoError(err)
	s.Require().Equal(assetstypes.StakerAssetInfo{
		TotalDepositAmount:        sdkmath.NewInt(32),
		WithdrawableAmount:        sdkmath.NewInt(32),
		PendingUndelegationAmount: sdkmath.ZeroInt(),
	}, *res)
	s.Require().NoError(err2)
	s.Require().Equal(oracletypes.StakerList{
		StakerAddrs: []string{
			stakerAddrStr,
		},
	}, *resStakerList.StakerList)
	s.Require().NoError(err3)
	s.Require().Equal(oracletypes.StakerInfo{
		StakerAddr:  stakerAddrStr,
		StakerIndex: 0,
		ValidatorPubkeyList: []string{
			hexutil.Encode(validatorPubkey),
		},
		BalanceList: []*oracletypes.BalanceInfo{
			{
				RoundID: 0,
				Block:   6,
				Index:   0,
				Balance: 32,
				Change:  oracletypes.Action_ACTION_DEPOSIT,
			},
		},
	}, *resStakerInfo.StakerInfo)

	// new block - 9, state of 8 is committed
	s.moveToAndCheck(9)
	resStakerInfo, err = s.network.QueryOracle().StakerInfo(ctx, &oracletypes.QueryStakerInfoRequest{AssetId: network.NativeAssetID, StakerAddr: stakerAddrStr})
	s.Require().NoError(err)
	s.Require().Equal(2, len(resStakerInfo.StakerInfo.BalanceList))
	s.Require().Equal([]*oracletypes.BalanceInfo{
		{
			Block:   6,
			Index:   0,
			Balance: 32,
			Change:  oracletypes.Action_ACTION_DEPOSIT,
		},
		{
			RoundID: 1,
			Index:   1,
			Block:   8,
			Balance: 28,
			Change:  oracletypes.Action_ACTION_SLASH_REFUND,
		},
	}, resStakerInfo.StakerInfo.BalanceList)
}

func (s *E2ETestSuite) testSlashing() {
	// validator3 had already missed two rounds
	// 1. for NST balance change update round 1
	// 2. for LST round round 3, but rounds had reached to 4 which equals to reportWindow
	// two more conjuctive miss will lead validator3 to be slashed
	s.moveToAndCheck(50)
	// slashing_{miss_v3:3, window:5}[1.0.1.0.1] -> {miss_v3:2, window:4} [0.1.0.1]
	priceTest1R5, priceSource1R5 := price1.generateRealTimeStructs("13", 1)
	msg0 := oracletypes.NewMsgCreatePrice(creator0.String(), 1, []*oracletypes.PriceSource{&priceSource1R5}, 50, 1)
	msg1 := oracletypes.NewMsgCreatePrice(creator1.String(), 1, []*oracletypes.PriceSource{&priceSource1R5}, 50, 1)
	msg2 := oracletypes.NewMsgCreatePrice(creator2.String(), 1, []*oracletypes.PriceSource{&priceSource1R5}, 50, 1)
	err := s.network.SendTxOracleCreateprice([]sdk.Msg{msg0}, "valconskey0", kr0)
	s.Require().NoError(err)
	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg1}, "valconskey1", kr1)
	s.Require().NoError(err)
	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg2}, "valconskey2", kr2)
	s.Require().NoError(err)
	s.moveToAndCheck(52)
	// query state of 51 on height 52
	res, err := s.network.QueryOracle().LatestPrice(ctxWithHeight(51), &oracletypes.QueryGetLatestPriceRequest{TokenId: 1})
	s.Require().NoError(err)
	// price updated, round 4 has price{p1}
	// NOTE: update timestamp manually to ignore
	ret := priceTest1R5.getPriceTimeRound(5)
	ret.Timestamp = res.Price.Timestamp
	s.Require().Equal(ret, res.Price)
	s.moveToAndCheck(60)
	// slashing_{miss_v3:3, window:5} [0.1.0.1.1] -> {miss_v3:2, window:4} [1.0.1.1]
	_, priceSource1R6 := price1.generateRealTimeStructs("14", 1)
	msg0 = oracletypes.NewMsgCreatePrice(creator0.String(), 1, []*oracletypes.PriceSource{&priceSource1R6}, 60, 1)
	msg1 = oracletypes.NewMsgCreatePrice(creator1.String(), 1, []*oracletypes.PriceSource{&priceSource1R6}, 60, 1)
	msg2 = oracletypes.NewMsgCreatePrice(creator2.String(), 1, []*oracletypes.PriceSource{&priceSource1R6}, 60, 1)
	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg0}, "valconskey0", kr0)
	s.Require().NoError(err)
	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg1}, "valconskey1", kr1)
	s.Require().NoError(err)
	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg2}, "valconskey2", kr2)
	s.Require().NoError(err)
	s.moveToAndCheck(64)
	// query state of 63 on height 64
	resSigningInfo, err := s.network.QuerySlashing().SigningInfo(ctxWithHeight(63), &slashingtypes.QuerySigningInfoRequest{ConsAddress: sdk.ConsAddress(s.network.Validators[3].PubKey.Address()).String()})
	s.Require().NoError(err)
	// validator3 is jailed
	s.Require().True(resSigningInfo.ValSigningInfo.JailedUntil.After(time.Now()))
	chainID := avstypes.ChainIDWithoutRevision(s.network.Config.ChainID)
	avsAddr := avstypes.GenerateAVSAddress(chainID)
	resOperator, err := s.network.QueryOperator().QueryOptInfo(context.Background(), &operatortypes.QueryOptInfoRequest{OperatorAVSAddress: &operatortypes.OperatorAVSAddress{OperatorAddr: s.network.Validators[3].Address.String(), AvsAddress: avsAddr}})
	s.Require().NoError(err)
	s.Require().True(resOperator.Jailed)
	// wait for validator3 to pass jail duration
	// timeout commit is set to 2 seconds, 10 blocks about 20 seconds
	s.moveToAndCheck(75)
	msgUnjail := slashingtypes.NewMsgUnjail(s.network.Validators[3].ValAddress)
	// unjail validator3
	err = s.network.SendTx([]sdk.Msg{msgUnjail}, "node3", kr3)
	s.Require().NoError(err)
	s.moveNAndCheck(2)
	resOperator, err = s.network.QueryOperator().QueryOptInfo(context.Background(), &operatortypes.QueryOptInfoRequest{OperatorAVSAddress: &operatortypes.OperatorAVSAddress{OperatorAddr: s.network.Validators[3].Address.String(), AvsAddress: avsAddr}})
	s.Require().NoError(err)
	s.Require().False(resOperator.Jailed)
}

func (s *E2ETestSuite) testRegisterTokenThroughPrecompile() {
	s.moveToAndCheck(2)
	clientChainID := uint32(101)
	assetAddr := common.HexToAddress("0xB82381A3fBD3FaFA77B3a7bE693342618240065b")
	token := network.PaddingAddressTo32(assetAddr)
	decimal := uint8(18)
	name := "WSTETH"
	metaData := "WSTETH for sepolia 2"
	oracleInfo := fmt.Sprintf("%s,Ethereum,8,10,0xB82381A3fBD3FaFA77B3a7bE693342618240067b", name)

	// send eth transaction to precompile contract to register a new token
	err := s.network.SendPrecompileTx(network.ASSETS, "registerToken", clientChainID, token, decimal, name, metaData, oracleInfo)
	s.Require().NoError(err)

	s.moveToAndCheck(4)
	// registerToken will automaticlly register that token into oracle module
	res, err := s.network.QueryOracle().Params(ctxWithHeight(3), &oracletypes.QueryParamsRequest{})
	s.Require().NoError(err)
	s.Require().Equal(name, res.Params.Tokens[len(res.Params.Tokens)-1].Name)
}

func (s *E2ETestSuite) moveToAndCheck(height int64) {
	_, err := s.network.WaitForStateHeightWithTimeout(height, 120*time.Second)
	s.Require().NoError(err)
}

func (s *E2ETestSuite) moveNAndCheck(n int64) {
	for i := int64(0); i < n; i++ {
		err := s.network.WaitForStateNextBlock()
		s.Require().NoError(err)
	}
}

func ctxWithHeight(height int64) context.Context {
	md := metadata.Pairs("x-cosmos-block-height", fmt.Sprintf("%d", height))
	return metadata.NewOutgoingContext(context.Background(), md)
}
