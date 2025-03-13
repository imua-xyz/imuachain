package assets_test

import (
	"math"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	evmtypes "github.com/evmos/evmos/v16/x/evm/types"
	"github.com/imua-xyz/imuachain/precompiles/assets/testdata"
	testutilcontracts "github.com/imua-xyz/imuachain/precompiles/testutil/contracts"
	"github.com/imua-xyz/imuachain/testutil"
	testutiltx "github.com/imua-xyz/imuachain/testutil/tx"
	assetstypes "github.com/imua-xyz/imuachain/x/assets/types"
)

// ContractDeploymentData is a struct to define all relevant data to deploy a smart contract.
type ContractDeploymentData struct {
	// Contract is the compiled contract to deploy.
	Contract evmtypes.CompiledContract
	// ConstructorArgs are the arguments to pass to the constructor.
	ConstructorArgs []interface{}
}

// TestWrappedRevert tests the wrapped revert scenario in which a failed call is wrapped in a
// try catch block and prevented from bubbling to the Cosmos level. In this situation, the
// revert of asset state should take effect while the transaction should still succeed.
func (s *AssetsPrecompileSuite) TestWrappedRevert() {
	// set the base fee to 1; the lowest possible
	s.App.FeeMarketKeeper.SetBaseFee(s.Ctx, big.NewInt(1))

	// deploy the gateway contract
	gatewayAddr, err := s.DeployContract(testdata.GatewayContract)
	s.Require().NoError(err)
	// add it as an authorized gateway
	authorizedGateways, err := s.App.AssetsKeeper.GetParams(s.Ctx)
	s.Require().NoError(err)
	authorizedGateways.Gateways = append(authorizedGateways.Gateways, gatewayAddr.String())
	err = s.App.AssetsKeeper.SetParams(s.Ctx, authorizedGateways)
	s.Require().NoError(err)

	// deploy the gateway caller contract
	constructorArgs := []interface{}{gatewayAddr}
	gatewayCallerAddr, err := s.DeployContractWithArgs(
		ContractDeploymentData{
			Contract:        testdata.GatewayCallerContract,
			ConstructorArgs: constructorArgs,
		},
	)
	s.Require().NoError(err)

	// Setup common test parameters
	clientChainLzID := uint32(101)
	usdtAddress := common.FromHex(s.Assets[0].Address)
	decimals := s.Assets[0].Decimals
	factor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	paddedUsdtAddress := paddingClientChainAddress(usdtAddress, assetstypes.GeneralClientChainAddrLength)
	stakerAddress := testutiltx.GenerateAddress()
	paddedStakerAddress := paddingClientChainAddress(stakerAddress[:], assetstypes.GeneralClientChainAddrLength)
	stakerID, assetID := assetstypes.GetStakerIDAndAssetID(uint64(clientChainLzID), stakerAddress[:], usdtAddress[:])

	// Create base call arguments
	callArgs := testutilcontracts.CallArgs{
		ContractAddr: gatewayCallerAddr,
		ContractABI:  testdata.GatewayCallerContract.ABI,
		PrivKey:      s.PrivKey,
	}

	// Helper function to check balance
	checkBalance := func(expectedAmount *big.Int) {
		stakerAssetInfo, err := s.App.AssetsKeeper.GetStakerSpecifiedAssetInfo(s.Ctx, stakerID, assetID)
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
	_, _, err = testutilcontracts.Call(s.Ctx, s.App, args)
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

	// Update expected amount and check balance
	opAmount = new(big.Int).Sub(opAmount, withdrawAmount)
	checkBalance(opAmount)
	// change block to reset count of precompile calls
	s.Commit()

	// Test withdraw with revert scenarios
	testCases := []struct {
		name        string
		revertCount *big.Int
		gasLimit    uint64
	}{
		{"Single revert", big.NewInt(1), 0},
		{"Exceeding max reverts", big.NewInt(int64(evmtypes.MaxPrecompileCalls) + 2), math.MaxInt64 - 1},
	}

	for _, tc := range testCases {
		err = testutil.FundAccountWithBaseDenom(s.Ctx, s.App.BankKeeper, s.Address[:], math.MaxInt64)
		s.Require().NoError(err)
		args = callArgs.WithMethodName("withdrawLSTAndThenRevertXTimes").WithArgs(
			clientChainLzID,
			paddedUsdtAddress,
			paddedStakerAddress,
			withdrawAmount,
			tc.revertCount,
		).WithGasLimit(tc.gasLimit)
		_, _, err = testutilcontracts.Call(s.Ctx, s.App, args)
		s.Require().NoError(err)
		// do not commit the block, so that we can check within it

		// Balance should remain unchanged after reverts
		checkBalance(opAmount)
	}

	// reset block gas limit
	s.Commit()
	// reset base fee to 1 in case it changed in the previous block
	s.App.FeeMarketKeeper.SetBaseFee(s.Ctx, big.NewInt(1))
	// then fund the address
	err = testutil.FundAccountWithBaseDenom(s.Ctx, s.App.BankKeeper, s.Address[:], math.MaxInt64)
	s.Require().NoError(err)
	args = callArgs.WithMethodName("withdrawLSTXTimesInTryCatch").WithArgs(
		clientChainLzID,
		paddedUsdtAddress,
		paddedStakerAddress,
		withdrawAmount,
		// the eighth call will fail during AddJournalEntries
		// it will be caught by the try catch block in Solidity
		big.NewInt(int64(evmtypes.MaxPrecompileCalls)+2),
	).WithGasLimit(math.MaxInt64 - 1)
	_, _, err = testutilcontracts.Call(s.Ctx, s.App, args)
	s.Require().NoError(err)
	// update the expected amount such that the +2 is not counted
	opAmount = new(big.Int).Sub(opAmount, new(big.Int).Mul(withdrawAmount, big.NewInt(int64(evmtypes.MaxPrecompileCalls))))

	// Balance should remain unchanged after reverts
	checkBalance(opAmount)
}

func (s *AssetsPrecompileSuite) DeployContractWithArgs(
	deploymentData ContractDeploymentData,
) (common.Address, error) {
	return testutil.DeployContract(
		s.Ctx, s.App, s.PrivKey, s.QueryClientEVM,
		deploymentData.Contract, deploymentData.ConstructorArgs...,
	)
}
