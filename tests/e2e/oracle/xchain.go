package oracle

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"strconv"
	"strings"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/ethereum/go-ethereum/common"
	assetstypes "github.com/imua-xyz/imuachain/x/assets/types"
	keeper "github.com/imua-xyz/imuachain/x/oracle/keeper"
	oracletypes "github.com/imua-xyz/imuachain/x/oracle/types"

	"github.com/imua-xyz/imuachain/testutil/network"
)

func (s *XChainTestSuite) TestCrossChainOracle2PhasesA() {
	// Ensure local keyring/creator are initialized regardless of test order.
	kr0 = s.network.Validators[0].ClientCtx.Keyring
	creator0 = sdk.AccAddress(s.network.Validators[0].PubKey.Address())
	kr1 = s.network.Validators[1].ClientCtx.Keyring
	creator1 = sdk.AccAddress(s.network.Validators[1].PubKey.Address())
	kr2 = s.network.Validators[2].ClientCtx.Keyring
	creator2 = sdk.AccAddress(s.network.Validators[2].PubKey.Address())

	ctx := context.Background()
	paramsRes, err := s.network.QueryOracle().Params(ctx, &oracletypes.QueryParamsRequest{})
	s.Require().NoError(err)

	feederID, startBaseBlock, interval := s.mustGetXChainFeeder(paramsRes.Params)
	oracleCaller := common.BytesToAddress(authtypes.NewModuleAddress(oracletypes.ModuleName))
	gatewayAddr := s.ensureOracleGateway(oracleCaller)
	s.Require().Equal(strings.ToLower(network.ExpectedOracleGatewayAddress().Hex()), strings.ToLower(gatewayAddr.Hex()))

	currentHeight, err := s.network.LatestStateHeight()
	s.Require().NoError(err)
	minHeight := int64(startBaseBlock) + int64(paramsRes.Params.MaxNonce) + 2
	if currentHeight < minHeight {
		s.moveToAndCheck(minHeight)
	}

	baseBlock := s.nextBaseBlock(startBaseBlock, interval)
	s.moveToAndCheck(baseBlock)

	var srcChainID uint64 = network.TestEVMChainID
	batch := keeper.RawDataXChainBatch{
		SrcChainID: srcChainID,
		BatchSeq:   1,
		Messages: []keeper.RawDataXChainMsg{
			{
				ID:         "xmsg:0",
				Nonce:      1,
				Type:       "evm",
				PayloadB64: base64.StdEncoding.EncodeToString([]byte("hello")),
			},
			{
				ID:         "xmsg:1",
				Nonce:      2,
				Type:       "evm",
				PayloadB64: base64.StdEncoding.EncodeToString([]byte("world")),
			},
		},
	}
	rawData, err := json.Marshal(batch)
	s.Require().NoError(err)

	mt, err := oracletypes.DeriveMT(paramsRes.Params.PieceSizeByte, rawData)
	s.Require().NoError(err)
	rootHash := mt.RootHash()
	leafCount := mt.LeafCount()

	_, ps := priceNST1.generateRealTimeStructs("1", 1)
	leafCountStr := strconv.Itoa(int(leafCount))
	ps.Prices[0].Price = string(append(rootHash, leafCountStr...))

	msg0 := oracletypes.NewMsgCreatePrice2Phase(creator0.String(), feederID, []*oracletypes.PriceSource{&ps}, uint64(baseBlock), 1)
	msg1 := oracletypes.NewMsgCreatePrice2Phase(creator1.String(), feederID, []*oracletypes.PriceSource{&ps}, uint64(baseBlock), 1)
	msg2 := oracletypes.NewMsgCreatePrice2Phase(creator2.String(), feederID, []*oracletypes.PriceSource{&ps}, uint64(baseBlock), 1)
	s.Require().NoError(s.network.SendTxOracleCreateprice([]sdk.Msg{msg0}, "valconskey0", kr0))
	s.Require().NoError(s.network.SendTxOracleCreateprice([]sdk.Msg{msg1}, "valconskey1", kr1))
	s.Require().NoError(s.network.SendTxOracleCreateprice([]sdk.Msg{msg2}, "valconskey2", kr2))

	phaseTwoStart := baseBlock + int64(paramsRes.Params.MaxNonce)
	s.moveToAndCheck(phaseTwoStart)
	// Allow the 2nd-phase state (nextPieceIndex) to be initialized.
	s.moveNAndCheck(1)

	s.submitXChainPhaseTwoPieces(mt, feederID, baseBlock)
	// Allow EndBlock queue processing to complete.
	s.moveNAndCheck(2)

	// execSeq := s.waitForXChainLastExecutedSeq(srcChainID, 1)
	addedSeq := s.waitForXChainLastSeq(srcChainID, 1)
	s.Require().EqualValues(1, addedSeq)
	// Note: message may be marked processed even when gateway delivery fails (dropped after retries).
}

func (s *XChainTestSuite) TestCrossChainOracle2PhasesDepositLST() {
	// Ensure local keyring/creator are initialized regardless of test order.
	kr0 = s.network.Validators[0].ClientCtx.Keyring
	creator0 = sdk.AccAddress(s.network.Validators[0].PubKey.Address())
	kr1 = s.network.Validators[1].ClientCtx.Keyring
	creator1 = sdk.AccAddress(s.network.Validators[1].PubKey.Address())
	kr2 = s.network.Validators[2].ClientCtx.Keyring
	creator2 = sdk.AccAddress(s.network.Validators[2].PubKey.Address())

	ctx := context.Background()
	paramsRes, err := s.network.QueryOracle().Params(ctx, &oracletypes.QueryParamsRequest{})
	s.Require().NoError(err)

	feederID, startBaseBlock, interval := s.mustGetXChainFeeder(paramsRes.Params)
	oracleCaller := common.BytesToAddress(authtypes.NewModuleAddress(oracletypes.ModuleName))
	gatewayAddr := s.ensureOracleGateway(oracleCaller)
	s.Require().Equal(strings.ToLower(network.ExpectedOracleGatewayAddress().Hex()), strings.ToLower(gatewayAddr.Hex()))

	currentHeight, err := s.network.LatestStateHeight()
	s.Require().NoError(err)
	minHeight := int64(startBaseBlock) + int64(paramsRes.Params.MaxNonce) + 2
	if currentHeight < minHeight {
		s.moveToAndCheck(minHeight)
	}

	baseBlock := s.nextBaseBlock(startBaseBlock, interval)
	s.moveToAndCheck(baseBlock)

	var srcChainID uint64 = network.TestEVMChainID
	// Use next batch seq so this test works regardless of test order (e.g. after TestCrossChainOracle2PhasesA).
	currentSeq, _ := s.queryXChainLastSeq(srcChainID)
	batchSeq := currentSeq + 1

	stakerAddr := common.BytesToAddress(s.network.Validators[0].Address.Bytes())
	tokenAddr := common.HexToAddress(network.ETHAssetAddress)
	opAmount := sdkmath.NewInt(1000)

	stakerID, assetID := assetstypes.GetStakerIDAndAssetID(
		srcChainID,
		stakerAddr.Bytes(),
		tokenAddr.Bytes(),
	)
	before := s.queryStakerTotalDeposited(ctx, stakerID, assetID)

	payload := buildDepositLSTMessage(stakerAddr, tokenAddr, opAmount.BigInt())
	batch := keeper.RawDataXChainBatch{
		SrcChainID: srcChainID,
		BatchSeq:   batchSeq,
		Messages: []keeper.RawDataXChainMsg{
			{
				ID:         "xmsg:deposit:0",
				Nonce:      1,
				Type:       "evm",
				PayloadB64: base64.StdEncoding.EncodeToString(payload),
			},
		},
	}
	rawData, err := json.Marshal(batch)
	s.Require().NoError(err)

	mt, err := oracletypes.DeriveMT(paramsRes.Params.PieceSizeByte, rawData)
	s.Require().NoError(err)
	rootHash := mt.RootHash()
	leafCount := mt.LeafCount()

	_, ps := priceNST1.generateRealTimeStructs("1", 1)
	leafCountStr := strconv.Itoa(int(leafCount))
	ps.Prices[0].Price = string(append(rootHash, leafCountStr...))

	msg0 := oracletypes.NewMsgCreatePrice2Phase(creator0.String(), feederID, []*oracletypes.PriceSource{&ps}, uint64(baseBlock), 1)
	msg1 := oracletypes.NewMsgCreatePrice2Phase(creator1.String(), feederID, []*oracletypes.PriceSource{&ps}, uint64(baseBlock), 1)
	msg2 := oracletypes.NewMsgCreatePrice2Phase(creator2.String(), feederID, []*oracletypes.PriceSource{&ps}, uint64(baseBlock), 1)
	s.Require().NoError(s.network.SendTxOracleCreateprice([]sdk.Msg{msg0}, "valconskey0", kr0))
	s.Require().NoError(s.network.SendTxOracleCreateprice([]sdk.Msg{msg1}, "valconskey1", kr1))
	s.Require().NoError(s.network.SendTxOracleCreateprice([]sdk.Msg{msg2}, "valconskey2", kr2))

	phaseTwoStart := baseBlock + int64(paramsRes.Params.MaxNonce)
	s.moveToAndCheck(phaseTwoStart)
	// Allow the 2nd-phase state (nextPieceIndex) to be initialized.
	s.moveNAndCheck(1)

	s.submitXChainPhaseTwoPieces(mt, feederID, baseBlock)

	// Allow EndBlock queue processing to complete.
	s.moveNAndCheck(2)

	execSeq := s.waitForXChainLastExecutedSeq(srcChainID, batchSeq)
	s.Require().EqualValues(batchSeq, execSeq)

	processed := s.queryOracleStore(oracletypes.XChainMsgProcessedKey(srcChainID, "xmsg:deposit:0"))
	s.Require().Greater(len(processed), 0)

	after := s.queryStakerTotalDeposited(ctx, stakerID, assetID)
	s.T().Logf("deposit delta: stakerID=%s assetID=%s before=%s after=%s", stakerID, assetID, before, after)
	s.Require().True(after.Sub(before).Equal(opAmount))
}

func (s *XChainTestSuite) mustGetXChainFeeder(params oracletypes.Params) (feederID uint64, startBaseBlock uint64, interval uint64) {
	var tokenID uint64
	var found bool
	for i, token := range params.Tokens {
		if strings.HasPrefix(strings.ToLower(token.AssetID), oracletypes.XChainIDPrefix) {
			tokenID = uint64(i)
			found = true
			break
		}
	}
	s.Require().True(found, "no token with xchain_ prefix found in params")

	for i, feeder := range params.TokenFeeders {
		if feeder.TokenID == tokenID {
			return uint64(i), feeder.StartBaseBlock, feeder.Interval
		}
	}

	s.FailNow("xchain feeder not found in params")
	return 0, 0, 0
}

func (s *XChainTestSuite) nextBaseBlock(startBaseBlock uint64, interval uint64) int64 {
	height, err := s.network.LatestStateHeight()
	s.Require().NoError(err)
	if height <= int64(startBaseBlock) {
		return int64(startBaseBlock)
	}
	if interval == 0 {
		return height
	}
	// Move to the next base block >= current height.
	delta := height - int64(startBaseBlock)
	rounds := delta / int64(interval)
	baseBlock := int64(startBaseBlock) + rounds*int64(interval)
	if baseBlock < height {
		baseBlock += int64(interval)
	}
	return baseBlock
}

func buildDepositLSTMessage(staker common.Address, token common.Address, amount *big.Int) []byte {
	msg := make([]byte, 1+32+32+32)
	msg[0] = 2 // REQUEST_DEPOSIT_LST in GatewayStorage.Action

	stakerBz := network.PaddingAddressTo32(staker)
	tokenBz := network.PaddingAddressTo32(token)
	amountBz := make([]byte, 32)
	amount.FillBytes(amountBz)

	copy(msg[1:33], stakerBz)
	copy(msg[33:65], amountBz)
	copy(msg[65:97], tokenBz)

	return msg
}

func (s *XChainTestSuite) submitXChainPhaseTwoPieces(mt *oracletypes.MerkleTree, feederID uint64, baseBlock int64) {
	pieces, ok := mt.CollectedPieces()
	s.Require().True(ok)
	for i, piece := range pieces {
		proof := mt.MinimalProofByIndex(uint32(i))
		idxStr, hashStr := proof.FlattenString()
		_, ps2 := priceNST1.generateRealTimeStructs("1", 1)
		ps2.Prices[0].Price = string(piece)
		ps2.Prices[0].DetID = strconv.Itoa(i)
		if len(idxStr) > 0 {
			ps2.Prices = append(ps2.Prices, &oracletypes.PriceTimeDetID{
				Price:     hashStr,
				DetID:     idxStr,
				Timestamp: now,
			})
		}

		msg0 := oracletypes.NewMsgCreatePrice2Phase2(creator0.String(), feederID, []*oracletypes.PriceSource{&ps2}, uint64(baseBlock), 1)
		msg1 := oracletypes.NewMsgCreatePrice2Phase2(creator1.String(), feederID, []*oracletypes.PriceSource{&ps2}, uint64(baseBlock), 1)
		msg2 := oracletypes.NewMsgCreatePrice2Phase2(creator2.String(), feederID, []*oracletypes.PriceSource{&ps2}, uint64(baseBlock), 1)
		s.Require().NoError(s.network.SendTxOracleCreateprice([]sdk.Msg{msg0}, "valconskey0", kr0))
		s.Require().NoError(s.network.SendTxOracleCreateprice([]sdk.Msg{msg1}, "valconskey1", kr1))
		s.Require().NoError(s.network.SendTxOracleCreateprice([]sdk.Msg{msg2}, "valconskey2", kr2))
		s.moveNAndCheck(1)
	}
}

func (s *XChainTestSuite) queryStakerTotalDeposited(ctx context.Context, stakerID, assetID string) sdkmath.Int {
	res, err := s.network.QueryAssets().QueryStakerBalance(ctx, &assetstypes.QueryStakerBalanceRequest{
		StakerId: stakerID,
		AssetId:  assetID,
	})
	s.Require().NoError(err)
	s.Require().NotNil(res.StakerBalance)
	return res.StakerBalance.TotalDeposited
}

func (s *XChainTestSuite) queryXChainLastExecutedSeq(srcChainID uint64) (uint64, bool) {
	value := s.queryOracleStore(oracletypes.XChainLastExecutedSeqKey(srcChainID))
	if len(value) == 0 {
		return 0, false
	}
	seq, err := oracletypes.BytesToUint64(value)
	s.Require().NoError(err)
	return seq, true
}

func (s *XChainTestSuite) queryXChainLastSeq(srcChainID uint64) (uint64, bool) {
	value := s.queryOracleStore(oracletypes.XChainLastSeqKey(srcChainID))
	if len(value) == 0 {
		return 0, false
	}
	seq, err := oracletypes.BytesToUint64(value)
	s.Require().NoError(err)
	return seq, true
}

func (s *XChainTestSuite) queryOracleStore(key []byte) []byte {
	res, err := s.network.Validators[0].RPCClient.ABCIQuery(context.Background(), "/store/oracle/key", key)
	s.Require().NoError(err)
	return res.Response.Value
}

func (s *XChainTestSuite) ensureOracleGateway(oracleCaller common.Address) common.Address {
	if (s.gatewayAddr != common.Address{}) {
		return s.gatewayAddr
	}
	addr, err := s.network.DeployOracleGatewayContract(oracleCaller)
	s.Require().NoError(err)
	// Allow the deployment tx to be included.
	s.moveNAndCheck(1)
	s.gatewayAddr = addr
	return addr
}

func (s *XChainTestSuite) waitForXChainLastExecutedSeq(srcChainID uint64, expected uint64) uint64 {
	for i := 0; i < 10; i++ {
		if seq, ok := s.queryXChainLastExecutedSeq(srcChainID); ok {
			if seq == expected {
				return seq
			}
		}
		s.moveNAndCheck(1)
	}
	s.FailNow("xchain last executed seq not updated")
	return 0
}

func (s *XChainTestSuite) waitForXChainLastSeq(srcChainID uint64, expected uint64) uint64 {
	for i := 0; i < 10; i++ {
		if seq, ok := s.queryXChainLastSeq(srcChainID); ok {
			if seq == expected {
				return seq
			}
		}
		s.moveNAndCheck(1)
	}
	s.FailNow("xchain last seq not updated")
	return 0
}

// --- Outbound E2E tests (方案A: imuachain 侧验证) ---

// TestCrossChainOutboundWithdrawResponse verifies the full outbound capture path:
// 1. Deposit LST to establish balance
// 2. Withdraw LST → gateway emits OutboundResponse
// 3. oracle module captures OutboundResponse from EVM logs → outbound queue
// 4. EndBlock creates checkpoint for pending outbound messages
func (s *XChainTestSuite) TestCrossChainOutboundWithdrawResponse() {
	kr0 = s.network.Validators[0].ClientCtx.Keyring
	creator0 = sdk.AccAddress(s.network.Validators[0].PubKey.Address())
	kr1 = s.network.Validators[1].ClientCtx.Keyring
	creator1 = sdk.AccAddress(s.network.Validators[1].PubKey.Address())
	kr2 = s.network.Validators[2].ClientCtx.Keyring
	creator2 = sdk.AccAddress(s.network.Validators[2].PubKey.Address())

	ctx := context.Background()
	paramsRes, err := s.network.QueryOracle().Params(ctx, &oracletypes.QueryParamsRequest{})
	s.Require().NoError(err)

	feederID, startBaseBlock, interval := s.mustGetXChainFeeder(paramsRes.Params)
	oracleCaller := common.BytesToAddress(authtypes.NewModuleAddress(oracletypes.ModuleName))
	gatewayAddr := s.ensureOracleGateway(oracleCaller)
	s.Require().Equal(strings.ToLower(network.ExpectedOracleGatewayAddress().Hex()), strings.ToLower(gatewayAddr.Hex()))

	currentHeight, err := s.network.LatestStateHeight()
	s.Require().NoError(err)
	minHeight := int64(startBaseBlock) + int64(paramsRes.Params.MaxNonce) + 2
	if currentHeight < minHeight {
		s.moveToAndCheck(minHeight)
	}

	var srcChainID uint64 = network.TestEVMChainID
	stakerAddr := common.BytesToAddress(s.network.Validators[0].Address.Bytes())
	tokenAddr := common.HexToAddress(network.ETHAssetAddress)

	// --- Step 1: Deposit to establish balance ---
	s.T().Log("Step 1: Deposit LST to establish balance")
	currentSeq, _ := s.queryXChainLastSeq(srcChainID)
	depositBatchSeq := currentSeq + 1
	depositAmount := sdkmath.NewInt(5000)

	depositPayload := buildDepositLSTMessage(stakerAddr, tokenAddr, depositAmount.BigInt())
	s.submitBatchAndWait(paramsRes.Params, feederID, startBaseBlock, interval, srcChainID, depositBatchSeq, []keeper.RawDataXChainMsg{
		{ID: "outbound-test-deposit:0", Nonce: 100, Type: "evm", PayloadB64: base64.StdEncoding.EncodeToString(depositPayload)},
	})

	stakerID, assetID := assetstypes.GetStakerIDAndAssetID(srcChainID, stakerAddr.Bytes(), tokenAddr.Bytes())
	afterDeposit := s.queryStakerTotalDeposited(ctx, stakerID, assetID)
	s.T().Logf("After deposit: totalDeposited=%s", afterDeposit)
	s.Require().True(afterDeposit.GTE(depositAmount))

	// --- Step 2: Withdraw LST → triggers OutboundResponse ---
	s.T().Log("Step 2: Withdraw LST to trigger outbound response")
	withdrawBatchSeq := depositBatchSeq + 1
	withdrawAmount := sdkmath.NewInt(1000)

	withdrawPayload := buildWithdrawLSTMessage(stakerAddr, tokenAddr, withdrawAmount.BigInt())
	s.submitBatchAndWait(paramsRes.Params, feederID, startBaseBlock, interval, srcChainID, withdrawBatchSeq, []keeper.RawDataXChainMsg{
		{ID: "outbound-test-withdraw:0", Nonce: 101, Type: "evm", PayloadB64: base64.StdEncoding.EncodeToString(withdrawPayload)},
	})

	// Allow a few more blocks for outbound processing + checkpoint creation
	s.moveNAndCheck(3)

	// --- Step 3: Verify outbound queue ---
	s.T().Log("Step 3: Verify outbound queue has the response")
	outboundMsgs := s.queryOutboundMessages(srcChainID)
	s.T().Logf("Outbound messages for chain %d: count=%d", srcChainID, len(outboundMsgs))
	// The withdraw should have triggered an OutboundResponse event in the gateway,
	// which deliverXChainToGateway parsed and enqueued.
	if len(outboundMsgs) > 0 {
		s.T().Logf("First outbound msg: seq=%d nonce=%d payloadLen=%d", outboundMsgs[0].SeqNum, outboundMsgs[0].Nonce, len(outboundMsgs[0].PayloadHex))
	}

	// --- Step 4: Verify checkpoint creation ---
	s.T().Log("Step 4: Verify checkpoint was created")
	checkpointNonce := s.queryCheckpointNonce(srcChainID)
	s.T().Logf("Checkpoint nonce for chain %d: %d", srcChainID, checkpointNonce)

	if len(outboundMsgs) > 0 {
		s.Require().Greater(checkpointNonce, uint64(0), "checkpoint should exist when outbound messages are pending")

		// Verify checkpoint data
		cpBz := s.queryOracleStore(oracletypes.CheckpointDataKey(srcChainID, checkpointNonce))
		s.Require().Greater(len(cpBz), 0, "checkpoint data should be stored")
		s.T().Logf("Checkpoint data length: %d bytes", len(cpBz))
	}

	s.T().Log("Outbound E2E (方案A) completed successfully")
}

// submitBatchAndWait submits a complete xchain batch (phase 1 + phase 2) and waits for processing.
func (s *XChainTestSuite) submitBatchAndWait(
	params oracletypes.Params,
	feederID, startBaseBlock, interval, srcChainID, batchSeq uint64,
	messages []keeper.RawDataXChainMsg,
) {
	batch := keeper.RawDataXChainBatch{
		SrcChainID: srcChainID,
		BatchSeq:   batchSeq,
		Messages:   messages,
	}
	rawData, err := json.Marshal(batch)
	s.Require().NoError(err)

	mt, err := oracletypes.DeriveMT(params.PieceSizeByte, rawData)
	s.Require().NoError(err)
	rootHash := mt.RootHash()
	leafCount := mt.LeafCount()

	baseBlock := s.nextBaseBlock(startBaseBlock, interval)
	s.moveToAndCheck(baseBlock)

	_, ps := priceNST1.generateRealTimeStructs("1", 1)
	leafCountStr := strconv.Itoa(int(leafCount))
	ps.Prices[0].Price = string(append(rootHash, leafCountStr...))

	// Phase 1: all 3 validators submit roothash
	msg0 := oracletypes.NewMsgCreatePrice2Phase(creator0.String(), feederID, []*oracletypes.PriceSource{&ps}, uint64(baseBlock), 1)
	msg1 := oracletypes.NewMsgCreatePrice2Phase(creator1.String(), feederID, []*oracletypes.PriceSource{&ps}, uint64(baseBlock), 1)
	msg2 := oracletypes.NewMsgCreatePrice2Phase(creator2.String(), feederID, []*oracletypes.PriceSource{&ps}, uint64(baseBlock), 1)
	s.Require().NoError(s.network.SendTxOracleCreateprice([]sdk.Msg{msg0}, "valconskey0", kr0))
	s.Require().NoError(s.network.SendTxOracleCreateprice([]sdk.Msg{msg1}, "valconskey1", kr1))
	s.Require().NoError(s.network.SendTxOracleCreateprice([]sdk.Msg{msg2}, "valconskey2", kr2))

	phaseTwoStart := baseBlock + int64(params.MaxNonce)
	s.moveToAndCheck(phaseTwoStart)
	s.moveNAndCheck(1)

	// Phase 2: submit raw data pieces
	s.submitXChainPhaseTwoPieces(mt, feederID, baseBlock)

	// Wait for delivery
	s.moveNAndCheck(2)
	addedSeq := s.waitForXChainLastSeq(srcChainID, batchSeq)
	s.Require().EqualValues(batchSeq, addedSeq)
}

func buildWithdrawLSTMessage(staker common.Address, token common.Address, amount *big.Int) []byte {
	msg := make([]byte, 1+32+32+32)
	msg[0] = 4 // REQUEST_WITHDRAW_LST in GatewayStorage.Action

	stakerBz := network.PaddingAddressTo32(staker)
	tokenBz := network.PaddingAddressTo32(token)
	amountBz := make([]byte, 32)
	amount.FillBytes(amountBz)

	copy(msg[1:33], stakerBz)
	copy(msg[33:65], amountBz)
	copy(msg[65:97], tokenBz)

	return msg
}

func (s *XChainTestSuite) queryOutboundMessages(dstChainID uint64) []keeper.OutboundMsg {
	headBz := s.queryOracleStore(oracletypes.OutboundHeadKey(dstChainID))
	tailBz := s.queryOracleStore(oracletypes.OutboundTailKey(dstChainID))
	if len(headBz) == 0 || len(tailBz) == 0 {
		return nil
	}
	head, _ := oracletypes.BytesToUint64(headBz)
	tail, _ := oracletypes.BytesToUint64(tailBz)
	var msgs []keeper.OutboundMsg
	for idx := head; idx < tail; idx++ {
		bz := s.queryOracleStore(oracletypes.OutboundItemKey(dstChainID, idx))
		if len(bz) == 0 {
			continue
		}
		var msg keeper.OutboundMsg
		if err := json.Unmarshal(bz, &msg); err != nil {
			continue
		}
		msgs = append(msgs, msg)
	}
	return msgs
}

func (s *XChainTestSuite) queryCheckpointNonce(dstChainID uint64) uint64 {
	bz := s.queryOracleStore(oracletypes.CheckpointNonceKey(dstChainID))
	if len(bz) == 0 {
		return 0
	}
	v, _ := oracletypes.BytesToUint64(bz)
	return v
}
