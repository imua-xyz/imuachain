package assets_test

import (
	"encoding/hex"
	"fmt"
	"math"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/evmos/evmos/v16/crypto/ethsecp256k1"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	evmtypes "github.com/evmos/evmos/v16/x/evm/types"
	"github.com/imua-xyz/imuachain/precompiles/assets/testdata"
	testutilcontracts "github.com/imua-xyz/imuachain/precompiles/testutil/contracts"
	"github.com/imua-xyz/imuachain/testutil"
	testutiltx "github.com/imua-xyz/imuachain/testutil/tx"
	assetstypes "github.com/imua-xyz/imuachain/x/assets/types"
)

var (
	// StorageSlotCounter is the storage slot of the counter object in the Gateway contract.
	StorageSlotCounter = common.Big1
	// CounterStartingValue is the starting value of the counter.
	CounterStartingValue = uint64(1)
)

// ContractDeploymentData is a struct to define all relevant data to deploy a smart contract.
type ContractDeploymentData struct {
	// Contract is the compiled contract to deploy.
	Contract evmtypes.CompiledContract
	// ConstructorArgs are the arguments to pass to the constructor.
	ConstructorArgs []interface{}
}

// prepareTestContracts prepares the test contracts for the tests in this file.
// It returns the gateway caller and callee addresses.
func (s *AssetsPrecompileSuite) prepareTestContracts() (common.Address, common.Address) {
	// set the base fee to 1; the lowest possible
	s.App.FeeMarketKeeper.SetBaseFee(s.Ctx, big.NewInt(1))

	// deploy the gateway callee contract
	gatewayCalleeAddr, err := s.DeployContract(testdata.GatewayCalleeContract)
	s.Require().NoError(err)

	// deploy the gateway contract
	constructorArgs := []interface{}{gatewayCalleeAddr}
	gatewayAddr, err := s.DeployContractWithArgs(
		ContractDeploymentData{
			Contract:        testdata.GatewayContract,
			ConstructorArgs: constructorArgs,
		},
	)
	s.Require().NoError(err)
	// add it as an authorized gateway
	authorizedGateways, err := s.App.AssetsKeeper.GetParams(s.Ctx)
	s.Require().NoError(err)
	authorizedGateways.Gateways = append(authorizedGateways.Gateways, gatewayAddr.String())
	err = s.App.AssetsKeeper.SetParams(s.Ctx, authorizedGateways)
	s.Require().NoError(err)

	// deploy the gateway caller contract
	constructorArgs = []interface{}{gatewayAddr}
	gatewayCallerAddr, err := s.DeployContractWithArgs(
		ContractDeploymentData{
			Contract:        testdata.GatewayCallerContract,
			ConstructorArgs: constructorArgs,
		},
	)
	s.Require().NoError(err)

	return gatewayCallerAddr, gatewayAddr
}

func (s *AssetsPrecompileSuite) getCounterValue(gatewayAddr common.Address) uint64 {
	value := s.App.EvmKeeper.GetState(s.Ctx, gatewayAddr, common.BigToHash(StorageSlotCounter))
	return value.Big().Uint64()
}

// TestWrappedRevert tests the wrapped revert scenario in which a failed call is wrapped in a
// try catch block and prevented from bubbling to the Cosmos level. In this situation, the
// revert of asset state should take effect while the transaction should still succeed.
func (s *AssetsPrecompileSuite) TestWrappedRevert() {
	gatewayCallerAddr, gatewayAddr := s.prepareTestContracts()
	value := s.getCounterValue(gatewayAddr)
	s.Require().Equal(CounterStartingValue, value)

	// Setup common test parameters
	clientChainLzID := uint32(101)
	usdtAddress := common.FromHex(s.Assets[0].Address)
	decimals := s.Assets[0].Decimals
	factor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	paddedUsdtAddress := paddingClientChainAddress(
		usdtAddress, assetstypes.GeneralClientChainAddrLength,
	)
	stakerAddress := testutiltx.GenerateAddress()
	paddedStakerAddress := paddingClientChainAddress(
		stakerAddress[:], assetstypes.GeneralClientChainAddrLength,
	)
	stakerID, assetID := assetstypes.GetStakerIDAndAssetID(
		uint64(clientChainLzID), stakerAddress[:], usdtAddress[:],
	)

	// Create base call arguments
	callArgs := testutilcontracts.CallArgs{
		ContractAddr: gatewayCallerAddr,
		ContractABI:  testdata.GatewayCallerContract.ABI,
		PrivKey:      s.PrivKey,
	}

	// Helper function to check balance
	checkBalance := func(expectedAmount *big.Int) {
		stakerAssetInfo, err := s.App.AssetsKeeper.GetStakerSpecifiedAssetInfo(
			s.Ctx, stakerID, assetID,
		)
		s.Require().NoError(err)
		s.Equal(expectedAmount, stakerAssetInfo.TotalDepositAmount.BigInt())
	}

	// deposit using the gateway caller contract
	opAmount := new(big.Int).Mul(big.NewInt(100), factor)
	args := callArgs.WithMethodName("depositLST").WithArgs(
		clientChainLzID,
		paddedUsdtAddress,
		paddedStakerAddress,
		opAmount,
	)
	// call the depositLST function
	_, _, err := testutilcontracts.Call(s.Ctx, s.App, args)
	s.Require().NoError(err)
	checkBalance(opAmount)

	// Perform withdraw
	withdrawAmount := new(big.Int).Mul(big.NewInt(1), factor)
	args = callArgs.WithMethodName("withdrawLST").WithArgs(
		clientChainLzID,
		paddedUsdtAddress,
		paddedStakerAddress,
		withdrawAmount,
	)
	// call the withdrawLST function
	_, _, err = testutilcontracts.Call(s.Ctx, s.App, args)
	s.Require().NoError(err)

	value = s.getCounterValue(gatewayAddr)
	s.Require().Equal(CounterStartingValue+1, value)

	// Update expected amount and check balance
	opAmount = new(big.Int).Sub(opAmount, withdrawAmount)
	checkBalance(opAmount)
	// change block to reset count of precompile calls
	s.Commit()

	/*
	**** BASE TESTS FINISHED ****
	 */

	// Test withdraw with revert scenarios
	testCases := []struct {
		name                 string
		revertCount          *big.Int
		gasLimit             uint64
		methodName           string
		expectedAmount       *big.Int
		commitBefore         bool
		expectedCounterValue uint64
	}{
		{
			// case 1: try { withdraw; revert; } catch { } one time does not
			// cause deposited amount to change
			name:           "Verify that a wrapped revert undoes the state change",
			revertCount:    big.NewInt(1),
			gasLimit:       math.MaxInt64 - 1,
			methodName:     "withdrawLSTAndThenRevertXTimes",
			expectedAmount: opAmount,
			// reset everything before the test
			commitBefore: true,
			// the revert happens in the gateway depth, which holds the counter
			// hence, counter value must not change.
			// the same revert propagates up to the gateway caller depth, which
			// catches the revert allowing the transaction to succeed.
			expectedCounterValue: CounterStartingValue + 1,
		},
		{
			// case 2: check that loop > N times with try { withdraw; revert; } catch { }
			// will not cause deposited amount to change
			name:        "Verify that more than N wrapped reverts undo the state change",
			revertCount: big.NewInt(int64(evmtypes.MaxPrecompileCalls) + 2),
			// maximum possible gas limit to ensure that it's not the limiting factor
			gasLimit:       math.MaxInt64 - 1,
			methodName:     "withdrawLSTAndThenRevertXTimes",
			expectedAmount: opAmount,
			// do not commit the block, so that the number of precompile calls is not reset
			commitBefore:         false,
			expectedCounterValue: CounterStartingValue + 1,
		},
		{
			// case 3: check that try { withdraw; } catch {} for > N times is only
			// effective for N times
			name: "Verify that more than N successful withdrawals are capped at N",
			// we can go higher than 9 but this number is sufficient to prove the point
			// plus, gas limit concerns will come into play at some point
			revertCount: big.NewInt(int64(evmtypes.MaxPrecompileCalls) + 2),
			// maximum possible gas limit to ensure that it's not the limiting factor
			gasLimit: math.MaxInt64 - 1,
			// we will withdraw 9 times in a try/catch
			// the eighth and ninth calls will fail during AddJournalEntries and
			// be caught by the try/catch block; however, they will still revert
			// the state but save the transaction failure from bubbling up to the
			// Cosmos level
			methodName: "withdrawLSTXTimesInTryCatch",
			// hence, the expected amount is only 7 withdrawal amounts lower
			// than the initial amount and not 9
			expectedAmount: new(big.Int).Sub(
				opAmount,
				new(big.Int).Mul(
					withdrawAmount,
					big.NewInt(int64(evmtypes.MaxPrecompileCalls)),
				),
			),
			// reset everything before the test, particularly the number of precompile calls
			commitBefore: true,
			// the revert happens in the precompile, so that is the gateway depth.
			// however, it happens after N tries have succeeded, so the counter
			// value must be incremented by N
			// In simpler words, we start with 2 (1 in the constructor, 1 in the first withdraw)
			// then, we make 7 successful withdrawals, attempt the eighth one which fails
			// so total is 2 + 7 = 9.
			expectedCounterValue: CounterStartingValue + 1 + uint64(evmtypes.MaxPrecompileCalls),
		},
	}

	for _, tc := range testCases {
		if tc.commitBefore {
			s.Commit()
		}
		// ensure that gas is not the blocker
		err = testutil.FundAccountWithBaseDenom(
			s.Ctx, s.App.BankKeeper, s.Address[:], math.MaxInt64,
		)
		s.Require().NoError(err)
		args = callArgs.WithMethodName(tc.methodName).WithArgs(
			clientChainLzID,
			paddedUsdtAddress,
			paddedStakerAddress,
			withdrawAmount,
			tc.revertCount,
		).WithGasLimit(tc.gasLimit)
		_, _, err = testutilcontracts.Call(s.Ctx, s.App, args)
		s.Require().NoError(err)
		value := s.getCounterValue(gatewayAddr)
		s.Equal(tc.expectedCounterValue, value, fmt.Sprintf("counter value mismatch for %s", tc.name))

		// Balance should remain unchanged after reverts
		checkBalance(tc.expectedAmount)
	}
}

func (s *AssetsPrecompileSuite) TestGasStarvation() {
	hexKey := "D196DCA836F8AC2FFF45B3C9F0113825CCBB33FA1B39737B948503B263ED75AE"
	privKeyDecoded, err := hex.DecodeString(hexKey)
	s.Require().NoError(err)
	privKey := &ethsecp256k1.PrivKey{Key: privKeyDecoded}
	key, err := privKey.ToECDSA()
	s.Require().NoError(err)
	s.PrivKey = privKey
	s.Address = crypto.PubkeyToAddress(key.PublicKey)
	account := s.App.AccountKeeper.GetAccount(s.Ctx, sdk.AccAddress(s.Address.Bytes()))
	if account == nil {
		account = types.NewBaseAccount(s.Address[:], nil, 0, 0)
	}
	account.SetSequence(4)
	s.App.AccountKeeper.SetAccount(s.Ctx, account)
	// test constants, must match contracts
	const (
		TEST_CHAIN_ID = uint32(99)
	)
	var (
		VIRTUAL_TOKEN  = common.HexToAddress("0xbBbBBBBbbBBBbbbBbbBbbbbBBbBbbbbBbBbbBBbB")
		DEPOSIT_AMOUNT = big.NewInt(5)
	)
	deployer := testutiltx.GenerateAddress()
	staker := testutiltx.GenerateAddress()
	stakerID, ASSET_ID := assetstypes.GetStakerIDAndAssetID(
		uint64(TEST_CHAIN_ID), staker[:], VIRTUAL_TOKEN[:],
	)
	paddedStaker := paddingClientChainAddress(staker[:], assetstypes.GeneralClientChainAddrLength)
	paddedAsset := paddingClientChainAddress(VIRTUAL_TOKEN[:], assetstypes.GeneralClientChainAddrLength)
	// fund deployer and staker
	err = testutil.FundAccountWithBaseDenom(
		s.Ctx, s.App.BankKeeper, s.Address[:], 1000000000000000000,
	)
	s.Require().NoError(err)
	err = testutil.FundAccountWithBaseDenom(
		s.Ctx, s.App.BankKeeper, deployer[:], 1000000000000000000,
	)
	s.Require().NoError(err)
	err = testutil.FundAccountWithBaseDenom(
		s.Ctx, s.App.BankKeeper, staker[:], 1000000000000000000,
	)
	s.Require().NoError(err)
	// deploy the third party callee contract
	thirdPartyCalleeAddr, err := s.DeployContract(testdata.ThirdPartyCalleeContract)
	s.Require().NoError(err)
	// deploy the precompile caller contract
	reverterContractAddr, err := s.DeployContractWithArgs(
		ContractDeploymentData{
			Contract:        testdata.PrecompileCallerThatRevertsContract,
			ConstructorArgs: []interface{}{thirdPartyCalleeAddr},
		},
	)
	s.Require().NoError(err)
	// deploy the try catch caller contract
	tryCatchCallerAddr, err := s.DeployContract(testdata.TryCatchCallerContract)
	_ = tryCatchCallerAddr
	s.Require().NoError(err)
	// mark the precompile caller as an authorized gateway
	authorizedGateways, err := s.App.AssetsKeeper.GetParams(s.Ctx)
	s.Require().NoError(err)
	authorizedGateways.Gateways = append(authorizedGateways.Gateways, reverterContractAddr.String())
	err = s.App.AssetsKeeper.SetParams(s.Ctx, authorizedGateways)
	s.Require().NoError(err)
	// now, we do reverterContract.activateStakingForTestChain()
	args := testutilcontracts.CallArgs{
		ContractAddr: reverterContractAddr,
		ContractABI:  testdata.PrecompileCallerThatRevertsContract.ABI,
		PrivKey:      s.PrivKey,
	}.WithMethodName("activateStakingForTestChain")
	_, _, err = testutilcontracts.Call(s.Ctx, s.App, args)
	s.Require().NoError(err)
	s.Commit()
	// check if we are now registered
	// 1. chain
	s.Require().True(s.App.AssetsKeeper.ClientChainExists(s.Ctx, uint64(TEST_CHAIN_ID)))
	// 2. token
	s.Require().True(s.App.AssetsKeeper.IsStakingAsset(s.Ctx, ASSET_ID))
	// get balance of staker
	checkBalance := func(expectedAmount *big.Int) *big.Int {
		stakerAssetInfo, err := s.App.AssetsKeeper.GetStakerSpecifiedAssetInfo(
			s.Ctx, stakerID, ASSET_ID,
		)
		if err != nil {
			s.Equal(expectedAmount, big.NewInt(0))
			return big.NewInt(0)
		}
		s.Equal(expectedAmount, stakerAssetInfo.TotalDepositAmount.BigInt())
		return stakerAssetInfo.TotalDepositAmount.BigInt()
	}
	checkBalance(big.NewInt(0))
	// get initial nonce
	checkNonce := func(expectedNonce uint64, msgAndArgs ...interface{}) uint64 {
		x := s.App.EvmKeeper.GetState(s.Ctx, reverterContractAddr, common.BigToHash(common.Big1)).Big().Uint64()
		s.Equal(expectedNonce, x, msgAndArgs...)
		return x
	}
	nonce1 := checkNonce(0, "initial nonce should be 0")
	// make a real deposit
	args = testutilcontracts.CallArgs{
		ContractAddr: reverterContractAddr,
		ContractABI:  testdata.PrecompileCallerThatRevertsContract.ABI,
		PrivKey:      s.PrivKey,
	}.WithMethodName("callPrecompileAndNotRevert").WithArgs(
		TEST_CHAIN_ID,
		paddedAsset,
		paddedStaker,
		DEPOSIT_AMOUNT,
	)
	_, _, err = testutilcontracts.Call(s.Ctx, s.App, args)
	s.Require().NoError(err)
	s.Commit()
	checkBalance(DEPOSIT_AMOUNT)
	nonce2 := checkNonce(nonce1+1, "nonce should increase by 1")

	lowLevelSlot := common.BigToHash(common.Big2)
	previousCount := s.App.EvmKeeper.GetState(s.Ctx, tryCatchCallerAddr, lowLevelSlot).Big().Uint64()
	args = testutilcontracts.CallArgs{
		ContractAddr: tryCatchCallerAddr,
		ContractABI:  testdata.TryCatchCallerContract.ABI,
		PrivKey:      s.PrivKey,
	}.WithMethodName("callWithTryCatch").WithArgs(
		reverterContractAddr,
		TEST_CHAIN_ID,
		paddedAsset,
		paddedStaker,
		DEPOSIT_AMOUNT,
	).WithGasLimit(78_000)
	_, _, err = testutilcontracts.Call(s.Ctx, s.App, args)
	s.Require().NoError(err)
	s.Commit()
	checkNonce(nonce2, "nonce should not increase")
	nextCount := s.App.EvmKeeper.GetState(s.Ctx, tryCatchCallerAddr, lowLevelSlot).Big().Uint64()
	s.Equal(previousCount+1, nextCount, "success count should increase by 1")
}

// DeployContractWithArgs is a helper function to deploy a contract with constructor arguments.
func (s *AssetsPrecompileSuite) DeployContractWithArgs(
	deploymentData ContractDeploymentData,
) (common.Address, error) {
	return testutil.DeployContract(
		s.Ctx, s.App, s.PrivKey, s.QueryClientEVM,
		deploymentData.Contract, deploymentData.ConstructorArgs...,
	)
}
