package oracle

import (
	"context"
	"math/big"
	"math/rand"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/imua-xyz/imuachain/testutil/network"
	testutiltx "github.com/imua-xyz/imuachain/testutil/tx"
	oracletypes "github.com/imua-xyz/imuachain/x/oracle/types"
)

func (s *E2ETestSuite) testTwoPhaseNST(_ int64) {
	s.testCreatePriceNST()

	s.moveToAndCheck(11)
	clientChainID := uint32(101)
	opAmount := big.NewInt(32)
	nonce := uint64(0)
	var err error
	stakerAddrs := make([]string, 1, 31)
	stakerAddrs[0] = ""
	seenAddrs := make(map[string]struct{})
	duplicatedCount := 0
	for b := 0; b < 1; b++ {
		for i := 0; i < 30; i++ {
			stakerAddrStr := testutiltx.GenerateAddress().String()
			stakerAddr, _ := hexutil.Decode(stakerAddrStr)
			validatorPubkey := []byte{byte(i)}
			// deposit 32 NSTETH to staker from beaconchain_validatro_1
			nonce, err = s.network.SendPrecompileTxWithNonce(network.ASSETS, "depositNST", nonce, clientChainID, validatorPubkey, stakerAddr, opAmount)
			s.Require().NoError(err)
			stakerAddrs = append(stakerAddrs, stakerAddrStr)
			if _, ok := seenAddrs[stakerAddrStr]; ok {
				duplicatedCount++
			} else {
				seenAddrs[stakerAddrStr] = struct{}{}
			}
		}
	}

	mt, changes := getNstRootAndPiecesWithParams(30, 31, 32)
	s.moveToAndCheck(27)

	ctx := context.Background()
	_, ps := priceNST1.generateRealTimeStructs("1", 1)
	ps.Prices[0].Price = string(mt.RootHash())
	ps.Prices[0].DetID = strconv.Itoa(int(mt.LeafCount()))
	msg0 := oracletypes.NewMsgCreatePrice2Phase(creator0.String(), 2, []*oracletypes.PriceSource{&ps}, 27, 1)
	msg1 := oracletypes.NewMsgCreatePrice2Phase(creator1.String(), 2, []*oracletypes.PriceSource{&ps}, 27, 1)
	msg2 := oracletypes.NewMsgCreatePrice2Phase(creator2.String(), 2, []*oracletypes.PriceSource{&ps}, 27, 1)
	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg0}, "valconskey0", kr0)
	s.Require().NoError(err)
	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg1}, "valconskey1", kr1)
	s.Require().NoError(err)
	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg2}, "valconskey2", kr2)
	s.Require().NoError(err)
	_ = changes

	s.moveToAndCheck(29)
	pieces, _ := mt.CollectedPieces()
	for i, piece := range pieces {
		// #nosec G115
		proof := mt.MinimalProofByIndex(uint32(i))
		idxStr, hashStr := proof.FlattenString()
		_, ps := priceNST1.generateRealTimeStructs("1", 1)
		ps.Prices[0].Price = string(piece)
		ps.Prices[0].DetID = strconv.Itoa(i)
		if len(idxStr) > 0 {

			ps.Prices = append(ps.Prices, &oracletypes.PriceTimeDetID{
				Price:     hashStr,
				DetID:     idxStr,
				Timestamp: now,
			})
		}

		msg0 = oracletypes.NewMsgCreatePrice2Phase2(creator0.String(), 2, []*oracletypes.PriceSource{&ps}, 27, 1)
		err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg0}, "valconskey0", kr0)
		s.moveToAndCheck(int64(30 + i))
		//		s.Require().NoError(err)
	}

	// pick 10 random stakers to check its balance is updated as expected
	for i := 0; i < 10; i++ {
		idx := rand.Int63n(30) + 1
		resStakerInfo, err3 := s.network.QueryOracle().StakerInfo(ctx, &oracletypes.QueryStakerInfoRequest{AssetId: network.NativeAssetID, StakerAddr: stakerAddrs[idx]})
		s.Require().NoError(err3)
		bl := resStakerInfo.StakerInfo.BalanceList
		l := len(bl) - 1
		s.Require().Equal(changes[idx].Balance, bl[l].Balance)
	}
}
