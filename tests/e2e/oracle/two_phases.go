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
	stakerAddrStrs := make([]string, 1, 31)
	stakerAddrStrs[0] = ""
	seenAddrs := make(map[string]struct{})
	duplicatedCount := 0
	// TODO(leonz): expand this to result more than 10000 stakers after nst updating its storage, current nst storage cost too much gas when do deposit
	for b := 0; b < 1; b++ {
		for i := 0; i < 60; i++ {
			stakerAddrStr := testutiltx.GenerateAddress().String()
			stakerAddr, _ := hexutil.Decode(stakerAddrStr)
			validatorPubkey := []byte{byte(i)}
			// deposit 32 NSTETH to staker from beaconchain_validatro_1
			nonce, err = s.network.SendPrecompileTxWithNonce(network.ASSETS, "depositNST", nonce, clientChainID, validatorPubkey, stakerAddr, opAmount)
			s.Require().NoError(err)
			stakerAddrStrs = append(stakerAddrStrs, stakerAddrStr)
			if _, ok := seenAddrs[stakerAddrStr]; ok {
				duplicatedCount++
			} else {
				seenAddrs[stakerAddrStr] = struct{}{}
			}
		}
	}

	s.updateNSTBalance(32, 61, 30, stakerAddrStrs)

	s.updateNSTBalance(57, 61, 60, stakerAddrStrs)
}

// start should be the base block of an nst-feeder round
func (s *E2ETestSuite) updateNSTBalance(start uint64, version uint64, stakerCount uint32, stakerAddrStrs []string) {
	mt, changes := getNstRootAndPiecesWithParams(version, stakerCount, 32)

	// #nosec G115
	startHeight := int64(start)
	s.moveToAndCheck(startHeight)

	ctx := context.Background()
	_, ps := priceNST1.generateRealTimeStructs("1", 1)
	ps.Prices[0].Price = string(mt.RootHash())
	ps.Prices[0].DetID = strconv.Itoa(int(mt.LeafCount()))
	msg0 := oracletypes.NewMsgCreatePrice2Phase(creator0.String(), 2, []*oracletypes.PriceSource{&ps}, start, 1)
	msg1 := oracletypes.NewMsgCreatePrice2Phase(creator1.String(), 2, []*oracletypes.PriceSource{&ps}, start, 1)
	msg2 := oracletypes.NewMsgCreatePrice2Phase(creator2.String(), 2, []*oracletypes.PriceSource{&ps}, start, 1)
	err := s.network.SendTxOracleCreateprice([]sdk.Msg{msg0}, "valconskey0", kr0)
	s.Require().NoError(err)
	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg1}, "valconskey1", kr1)
	s.Require().NoError(err)
	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg2}, "valconskey2", kr2)
	s.Require().NoError(err)

	s.moveToAndCheck(startHeight + 2)
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

		msg0 = oracletypes.NewMsgCreatePrice2Phase2(creator0.String(), 2, []*oracletypes.PriceSource{&ps}, start, 1)
		err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg0}, "valconskey0", kr0)
		s.Require().NoError(err)
		s.moveToAndCheck(startHeight + int64(3+i))
	}

	//	resStakerInfos, _ := s.network.QueryOracle().StakerInfos(ctx, &oracletypes.QueryStakerInfosRequest{AssetId: network.NativeAssetID})
	// pick 10 random stakers to check its balance is updated as expected
	maxSize := 20
	if int(stakerCount) < maxSize {
		maxSize = int(stakerCount)
	}
	for i := 0; i < maxSize-1; i++ {
		idx := rand.Int63n(int64(maxSize)) + 1
		resStakerInfo, err3 := s.network.QueryOracle().StakerInfo(ctx, &oracletypes.QueryStakerInfoRequest{AssetId: network.NativeAssetID, StakerAddr: stakerAddrStrs[idx]})
		s.Require().NoError(err3)
		bl := resStakerInfo.StakerInfo.BalanceList
		l := len(bl) - 1
		s.Require().Equal(changes[idx].Balance, bl[l].Balance)
	}
}
