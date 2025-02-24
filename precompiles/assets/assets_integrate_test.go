package assets_test

import (
	"math/big"

	"github.com/imua-xyz/imuachain/precompiles/assets"

	"github.com/ethereum/go-ethereum/common"
	"github.com/imua-xyz/imuachain/precompiles/testutil"
	"github.com/imua-xyz/imuachain/precompiles/testutil/contracts"
	assetstype "github.com/imua-xyz/imuachain/x/assets/types"
)

// General variables used for integration tests
var (
	// defaultCallArgs  are the default arguments for calling the smart contract
	//
	// NOTE: this has to be populated in a BeforeEach block because the contractAddr would otherwise be a nil address.
	defaultCallArgs contracts.CallArgs

	// defaultLogCheck instantiates a log check arguments struct with the precompile ABI events populated.
	defaultLogCheck testutil.LogCheckArgs

	// passCheck defines the arguments to check if the precompile returns no error
	passCheck testutil.LogCheckArgs
)

func (s *AssetsPrecompileSuite) TestCallDepositLSTFromEOA() {
	// deposit params for test
	imuaLzAppAddress := "0x3fC91A3afd70395Cd496C647d5a6CC9D4B2b7FAD"
	depositParams := assetstype.Params{
		Gateways: []string{imuaLzAppAddress},
	}
	assetAddress := common.FromHex("0xdAC17F958D2ee523a2206206994597C13D831ec7")
	paddingAssetAddress := paddingClientChainAddress(assetAddress, assetstype.GeneralClientChainAddrLength)
	clientChainLzID := 101
	stakerAddr := paddingClientChainAddress(s.Address.Bytes(), assetstype.GeneralClientChainAddrLength)
	opAmount := big.NewInt(100)
	method := assets.MethodDepositLST

	beforeEach := func() {
		s.SetupTest()
		// set the default call arguments
		defaultCallArgs = contracts.CallArgs{
			ContractAddr: s.precompile.Address(),
			ContractABI:  s.precompile.ABI,
			PrivKey:      s.PrivKey,
		}

		defaultLogCheck = testutil.LogCheckArgs{
			ABIEvents: s.precompile.ABI.Events,
		}
		passCheck = defaultLogCheck.WithExpPass(true)
	}

	prepareFunc := func(params *assetstype.Params, method string) contracts.CallArgs {
		err := s.App.AssetsKeeper.SetParams(s.Ctx, params)
		s.Require().NoError(err)
		defaultDepositArgs := defaultCallArgs.WithMethodName(method)
		return defaultDepositArgs.WithArgs(
			uint32(clientChainLzID),
			paddingAssetAddress,
			stakerAddr,
			opAmount)
	}

	// test caller error
	beforeEach()
	setDepositToArgs := prepareFunc(&depositParams, method)
	_, response, err := contracts.CallContractAndCheckLogs(s.Ctx, s.App, setDepositToArgs, passCheck)
	// s.Require().ErrorContains(err, assetstype.ErrNotEqualToLzAppAddr.Error())
	result, err := s.precompile.ABI.Unpack(assets.MethodDepositLST, response.Ret)
	s.Require().NoError(err)
	s.Require().Equal(len(result), 2)
	success, ok := result[0].(bool)
	s.Require().True(ok)
	s.Require().False(success)

	// test success
	beforeEach()
	depositParams.Gateways = []string{s.Address.String()}
	setDepositToArgs = prepareFunc(&depositParams, method)
	_, ethRes, err := contracts.CallContractAndCheckLogs(s.Ctx, s.App, setDepositToArgs, passCheck)
	successRet, err := s.precompile.Methods[assets.MethodDepositLST].Outputs.Pack(true, opAmount)
	s.Require().NoError(err)
	s.Require().Equal(successRet, ethRes.Ret)
}

func (s *AssetsPrecompileSuite) TestCallDepositToFromContract() {
	// deposit params for test
	lzAppAddress := "0x3fC91A3afd70395Cd496C647d5a6CC9D4B2b7FAD"
	depositParams := assetstype.Params{
		Gateways: []string{lzAppAddress},
	}

	assetAddress := common.FromHex("0xdAC17F958D2ee523a2206206994597C13D831ec7")
	paddingAssetAddress := paddingClientChainAddress(assetAddress, assetstype.GeneralClientChainAddrLength)
	clientChainLzID := 101
	stakerAddr := paddingClientChainAddress(s.Address.Bytes(), assetstype.GeneralClientChainAddrLength)
	opAmount := big.NewInt(100)

	// contractAddr is the address of the smart contract that will be deployed
	var contractAddr common.Address
	var err error

	// deploy the caller contract
	s.SetupTest()
	contractAddr, err = s.DeployContract(contracts.DepositCallerContract)
	s.Require().NoError(err)
	// NextBlock the smart contract
	s.NextBlock()

	// check contract was correctly deployed
	cAcc := s.App.EvmKeeper.GetAccount(s.Ctx, contractAddr)
	s.Require().NotNil(cAcc)
	s.Require().True(cAcc.IsContract())

	beforeEach := func() {
		s.SetupTest()
		// populate default call args
		defaultCallArgs = contracts.CallArgs{
			ContractAddr: contractAddr,
			ContractABI:  contracts.DepositCallerContract.ABI,
			PrivKey:      s.PrivKey,
		}

		// default log check arguments
		defaultLogCheck = testutil.LogCheckArgs{ABIEvents: contracts.DepositCallerContract.ABI.Events}
		passCheck = defaultLogCheck.WithExpPass(true)
	}

	prepareFunc := func(params *assetstype.Params, method string) contracts.CallArgs {
		err := s.App.AssetsKeeper.SetParams(s.Ctx, params)
		s.Require().NoError(err)
		defaultDepositArgs := defaultCallArgs.WithMethodName(method)
		return defaultDepositArgs.WithArgs(
			uint32(clientChainLzID),
			paddingAssetAddress,
			stakerAddr,
			opAmount)
	}

	// testDepositTo
	beforeEach()
	depositParams.Gateways = []string{contractAddr.String()}
	setDepositToArgs := prepareFunc(&depositParams, "testDepositTo")
	_, _, err = contracts.CallContractAndCheckLogs(s.Ctx, s.App, setDepositToArgs, passCheck)
	s.Require().NoError(err)
	// todo: need to find why the ethRet is nil when called by contract
	/*	successRet, err := contracts.DepositCallerContract.ABI.Methods["testDepositTo"].Outputs.Pack(true, opAmount)
		s.Require().NoError(err)
		s.Require().Equal(successRet, ethRes.Ret)*/

	// testCallDepositToAndEmitEvent
	beforeEach()
	setDepositToArgs = prepareFunc(&depositParams, "testCallDepositToAndEmitEvent")
	// todo: need to check why can't get the ethereum log
	// eventCheck := passCheck.WithExpEvents("callDepositToResult")
	_, _, err = contracts.CallContractAndCheckLogs(s.Ctx, s.App, setDepositToArgs, passCheck)
	s.Require().NoError(err)
	/*	successRet, err = contracts.DepositCallerContract.ABI.Methods["testCallDepositToAndEmitEvent"].Outputs.Pack(true, opAmount)
		s.Require().NoError(err)
		s.Require().Equal(successRet, ethRes.Ret)*/

	// testCallDepositToWithTryCatch
	beforeEach()
	depositParams.Gateways = []string{lzAppAddress}
	setDepositToArgs = prepareFunc(&depositParams, "testCallDepositToWithTryCatch")
	// eventCheck = passCheck.WithExpEvents("ErrorOccurred")
	// todo: need to check the ethereum log
	_, _, err = contracts.CallContractAndCheckLogs(s.Ctx, s.App, setDepositToArgs, passCheck)
	s.Require().NoError(err)
	/*	successRet, err = contracts.DepositCallerContract.ABI.Methods["testCallDepositToWithTryCatch"].Outputs.Pack(false, big.NewInt(0))
		s.Require().NoError(err)
		s.Require().Equal(successRet, ethRes.Ret)*/
}

func (s *AssetsPrecompileSuite) TestCallWithdrawLSTFromEOA() {
	// withdraw params for test
	imuaLzAppAddress := "0x3fC91A3afd70395Cd496C647d5a6CC9D4B2b7FAD"
	params := assetstype.Params{
		Gateways: []string{imuaLzAppAddress},
	}
	usdtAddress := paddingClientChainAddress(common.FromHex("0xdAC17F958D2ee523a2206206994597C13D831ec7"), assetstype.GeneralClientChainAddrLength)
	clientChainLzID := 101
	stakerAddr := paddingClientChainAddress(s.Address.Bytes(), assetstype.GeneralClientChainAddrLength)
	opAmount := big.NewInt(100)
	assetAddr := usdtAddress
	method := assets.MethodWithdrawLST

	beforeEach := func() {
		s.SetupTest()
		// set the default call arguments
		defaultCallArgs = contracts.CallArgs{
			ContractAddr: s.precompile.Address(),
			ContractABI:  s.precompile.ABI,
			PrivKey:      s.PrivKey,
		}

		defaultLogCheck = testutil.LogCheckArgs{
			ABIEvents: s.precompile.ABI.Events,
		}
		passCheck = defaultLogCheck.WithExpPass(true)
	}

	prepareFunc := func(params *assetstype.Params, method string) contracts.CallArgs {
		err := s.App.AssetsKeeper.SetParams(s.Ctx, params)
		s.Require().NoError(err)
		defaultWithdrawArgs := defaultCallArgs.WithMethodName(method)
		return defaultWithdrawArgs.WithArgs(
			uint32(clientChainLzID),
			assetAddr,
			stakerAddr,
			opAmount)
	}

	beforeEach()
	setWithdrawArgs := prepareFunc(&params, method)
	_, response, err := contracts.CallContractAndCheckLogs(s.Ctx, s.App, setWithdrawArgs, passCheck)

	// for failed cases we expect it returns bool value instead of error
	// this is a workaround because the error returned by precompile can not be caught in EVM
	// see https://github.com/imua-xyz/imuachain/issues/70
	// TODO: we should figure out root cause and fix this issue to make precompiles work normally
	s.Require().NoError(err)

	result, err := setWithdrawArgs.ContractABI.Unpack(method, response.Ret)
	s.Require().NoError((err))

	// solidity: function withdraw(...) returns (bool success, uint256 updatedBalance)
	s.Require().Equal(len(result), 2)

	// the first element should be bool value that indicates whether the withdrawal is successful
	success, ok := result[0].(bool)
	s.Require().True(ok)
	s.Require().False(success)

	// the second element represents updatedBalance and should be 0 since success is false and withdrawal has failed
	updatedBalance, ok := result[1].(*big.Int)
	s.Require().True(ok)
	s.Require().Zero(updatedBalance.Cmp(new(big.Int)))
}
