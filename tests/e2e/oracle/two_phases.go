package oracle

import (
	"bytes"
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
	stakerAddrStrs := s.depositNST(3, 60)
	// #nosec G115
	stakerCount := uint32(len(stakerAddrStrs))
	version := uint64(stakerCount)
	c1 := uint32(30)
	c2 := uint32(60)
	if c1 > stakerCount {
		c1 = stakerCount
		c2 = stakerCount
	} else if c2 > stakerCount {
		c2 = stakerCount
	}
	s.updateNSTBalance(7, version, c1, stakerAddrStrs, true)

	s.updateNSTBalance(32, version, c2, stakerAddrStrs, true)
}

func (s *E2ETestSuite) testTwoPhaseNSTMalicious(_ uint64) {
	stakerAddrStrs := s.depositNST(3, 60)
	// #nosec G115
	stakerCount := uint32(len(stakerAddrStrs))
	version := uint64(stakerCount)
	mt, _ := getNstRootAndPiecesWithParams(version, stakerCount, 32)
	s.Require().Greater(mt.LeafCount(), uint32(3))
	s.sendRawDataRoot(7, mt.RootHash(), mt.LeafCount())

	s.moveToAndCheck(10)

	pieces, _ := mt.CollectedPieces()
	for i := 0; i < 4; i++ {
		var piece []byte
		var proof oracletypes.Proof
		if i <= 1 {
			piece = pieces[i]
			proof = mt.MinimalProofByIndex(uint32(i))
		} else {
			piece = pieces[i-1]
			// #nosec G115
			proof = mt.MinimalProofByIndex(uint32(i - 1))
		}
		// #nosec G115
		idxStr, hashStr := proof.FlattenString()
		_, ps := priceNST1.generateRealTimeStructs("1", 1)
		// modify to make it to be incorrect piece which will fail the merkle proof verify
		if i > 1 {
			ps.Prices[0].DetID = strconv.Itoa(i - 1)
			ps.Prices[0].Price = string(pieces[i-1])
		} else {
			ps.Prices[0].DetID = strconv.Itoa(i)
			if i == 1 {
				// this will pass anteHandler to be included into
				lp := len(piece)
				s.Require().Greater(lp, 2)
				fake := make([]byte, lp)
				copy(fake, piece)
				fBytes := []byte{1, 2}
				if bytes.Equal(fake[:2], fBytes) {
					fBytes = []byte{2, 3}
				}
				fake = append(fake[2:], fBytes...)
				ps.Prices[0].Price = string(fake)
			} else {
				ps.Prices[0].Price = string(piece)
			}
		}

		if len(idxStr) > 0 {
			ps.Prices = append(ps.Prices, &oracletypes.PriceTimeDetID{
				Price:     hashStr,
				DetID:     idxStr,
				Timestamp: now,
			})
		}

		switch i {
		case 0:
			msg0 := oracletypes.NewMsgCreatePrice2Phase2(creator0.String(), 2, []*oracletypes.PriceSource{&ps}, 7, 1)
			err := s.network.SendTxOracleCreateprice([]sdk.Msg{msg0}, "valconskey0", kr0)
			s.Require().NoError(err)
		case 1:
			msg0 := oracletypes.NewMsgCreatePrice2Phase2(creator0.String(), 2, []*oracletypes.PriceSource{&ps}, 7, 1)
			err := s.network.SendTxOracleCreateprice([]sdk.Msg{msg0}, "valconskey0", kr0)
			s.Require().NoError(err)
		case 2:
			msg1 := oracletypes.NewMsgCreatePrice2Phase2(creator1.String(), 2, []*oracletypes.PriceSource{&ps}, 7, 1)
			err := s.network.SendTxOracleCreateprice([]sdk.Msg{msg1}, "valconskey1", kr1)
			s.Require().NoError(err)
		case 3:
			msg1 := oracletypes.NewMsgCreatePrice2Phase2(creator1.String(), 2, []*oracletypes.PriceSource{&ps}, 7, 1)
			err := s.network.SendTxOracleCreateprice([]sdk.Msg{msg1}, "valconskey1", kr1)
			s.Require().ErrorContains(err, "no valid nextPieceIndex for feederID:2")
		}

		s.moveToAndCheck(int64(11 + i))
	}
}

// return value returns the number of valid stakers that had successfully deposited NST
func (s *E2ETestSuite) depositNST(start int64, stakerCount int) []string {
	s.moveToAndCheck(start)
	clientChainID := uint32(101)
	opAmount := big.NewInt(32)
	nonce := uint64(0)
	var err error
	stakerAddrStrs := make([]string, 0, stakerCount)
	seenAddrs := make(map[string]struct{})
	duplicatedCount := 0
	// TODO(leonz): expand this to result more than 10000 stakers after nst updating its storage, current nst storage cost too much gas when do deposit
	for b := 0; b < 1; b++ {
		for i := 0; i < stakerCount; i++ {
			stakerAddrStr := testutiltx.GenerateAddress().String()
			stakerAddr, _ := hexutil.Decode(stakerAddrStr)
			validatorPubkey := []byte{byte(i)}
			// deposit 32 NSTETH to staker from beaconchain_validatro_1
			nonce, err = s.network.SendPrecompileTxWithNonce(network.ASSETS, "depositNST", nonce, clientChainID, validatorPubkey, stakerAddr, opAmount)
			s.Require().NoError(err)
			if _, ok := seenAddrs[stakerAddrStr]; ok {
				duplicatedCount++
			} else {
				stakerAddrStrs = append(stakerAddrStrs, stakerAddrStr)
				seenAddrs[stakerAddrStr] = struct{}{}
			}
		}
	}
	return stakerAddrStrs
}

// start should be the base block of an nst-feeder round
func (s *E2ETestSuite) updateNSTBalance(start uint64, version uint64, stakerCount uint32, stakerAddrStrs []string, checkPieceError bool) {
	mt, changes := getNstRootAndPiecesWithParams(version, stakerCount, 32)

	s.sendRawDataRoot(start, mt.RootHash(), mt.LeafCount())
	// #nosec G115
	startHeight := int64(start)

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

		msg0 := oracletypes.NewMsgCreatePrice2Phase2(creator0.String(), 2, []*oracletypes.PriceSource{&ps}, start, 1)
		err := s.network.SendTxOracleCreateprice([]sdk.Msg{msg0}, "valconskey0", kr0)
		if checkPieceError {
			s.Require().NoError(err)
		}
		s.moveToAndCheck(startHeight + int64(3+i))
	}

	s.moveToAndCheck(startHeight + int64(3+len(pieces)))
	//	resStakerInfos, _ := s.network.QueryOracle().StakerInfos(ctx, &oracletypes.QueryStakerInfosRequest{AssetId: network.NativeAssetID})
	// pick 10 random stakers to check its balance is updated as expected
	maxSize := 20
	if int(stakerCount) < maxSize {
		maxSize = int(stakerCount)
	}

	ctx := context.Background()
	for i := 0; i < maxSize-1; i++ {
		idx := rand.Int63n(int64(maxSize)) + 1
		resStakerInfo, err3 := s.network.QueryOracle().StakerInfo(ctx, &oracletypes.QueryStakerInfoRequest{AssetId: network.NativeAssetID, StakerAddr: stakerAddrStrs[idx]})
		s.Require().NoError(err3)
		bl := resStakerInfo.StakerInfo.BalanceList
		l := len(bl) - 1
		s.Require().Equal(changes[idx].Balance, bl[l].Balance)
	}
}

func (s *E2ETestSuite) sendRawDataRoot(start uint64, root []byte, count uint32) {
	// #nosec G115
	startHeight := int64(start)
	s.moveToAndCheck(startHeight)
	//	ctx := context.Background()
	_, ps := priceNST1.generateRealTimeStructs("1", 1)
	ps.Prices[0].Price = string(root)
	ps.Prices[0].DetID = strconv.Itoa(int(count))
	msg0 := oracletypes.NewMsgCreatePrice2Phase(creator0.String(), 2, []*oracletypes.PriceSource{&ps}, start, 1)
	msg1 := oracletypes.NewMsgCreatePrice2Phase(creator1.String(), 2, []*oracletypes.PriceSource{&ps}, start, 1)
	msg2 := oracletypes.NewMsgCreatePrice2Phase(creator2.String(), 2, []*oracletypes.PriceSource{&ps}, start, 1)
	err := s.network.SendTxOracleCreateprice([]sdk.Msg{msg0}, "valconskey0", kr0)
	s.Require().NoError(err)
	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg1}, "valconskey1", kr1)
	s.Require().NoError(err)
	err = s.network.SendTxOracleCreateprice([]sdk.Msg{msg2}, "valconskey2", kr2)
	s.Require().NoError(err)
}
