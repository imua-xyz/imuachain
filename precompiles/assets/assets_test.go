package assets_test

import (
	"math/big"
	"strings"

	"cosmossdk.io/math"
	sdkmath "cosmossdk.io/math"
	assetsprecompile "github.com/ExocoreNetwork/exocore/precompiles/assets"
	assetskeeper "github.com/ExocoreNetwork/exocore/x/assets/keeper"
	assetstypes "github.com/ExocoreNetwork/exocore/x/assets/types"
	"github.com/ExocoreNetwork/exocore/x/oracle/types"
	"github.com/evmos/evmos/v16/x/evm/statedb"

	"github.com/ExocoreNetwork/exocore/app"
	assetstype "github.com/ExocoreNetwork/exocore/x/assets/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	evmtypes "github.com/evmos/evmos/v16/x/evm/types"
)

func (s *AssetsPrecompileSuite) TestIsTransaction() {
	testCases := []struct {
		name   string
		method string
		isTx   bool
	}{
		{
			assetsprecompile.MethodDepositLST,
			s.precompile.Methods[assetsprecompile.MethodDepositLST].Name,
			true,
		},
		{
			assetsprecompile.MethodWithdrawLST,
			s.precompile.Methods[assetsprecompile.MethodWithdrawLST].Name,
			true,
		},
		{
			assetsprecompile.MethodDepositNST,
			s.precompile.Methods[assetsprecompile.MethodDepositNST].Name,
			true,
		},
		{
			assetsprecompile.MethodWithdrawNST,
			s.precompile.Methods[assetsprecompile.MethodWithdrawNST].Name,
			true,
		},
		{
			assetsprecompile.MethodGetClientChains,
			s.precompile.Methods[assetsprecompile.MethodGetClientChains].Name,
			false,
		},
		{
			"invalid",
			"invalid",
			false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.Require().Equal(s.precompile.IsTransaction(tc.method), tc.isTx)
		})
	}
}

func paddingClientChainAddress(input []byte, outputLength int) []byte {
	if len(input) < outputLength {
		padding := make([]byte, outputLength-len(input))
		return append(input, padding...)
	}
	return input
}

// TestRunDepositTo tests DepositOrWithdraw method through calling Run function..
func (s *AssetsPrecompileSuite) TestRunDeposit() {
	// assetsprecompile params for test
	exocoreLzAppAddress := "0x3fC91A3afd70395Cd496C647d5a6CC9D4B2b7FAD"
	usdtAddress := paddingClientChainAddress(common.FromHex("0xdAC17F958D2ee523a2206206994597C13D831ec7"), assetstype.GeneralClientChainAddrLength)
	usdcAddress := paddingClientChainAddress(common.FromHex("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"), assetstype.GeneralClientChainAddrLength)
	clientChainLzID := 101
	stakerAddr := paddingClientChainAddress(s.Address.Bytes(), assetstype.GeneralClientChainAddrLength)
	stakerAddrStr := strings.ToLower(s.Address.String())
	NSTAssetAddr := assetstypes.GenerateNSTAddr(s.ClientChains[0].AddressLength)
	stakerID, assetID := assetstype.GetStakerIDAndAssetID(s.ClientChains[0].LayerZeroChainID, s.Address.Bytes(), NSTAssetAddr)

	opAmount := big.NewInt(100)
	opAmount32, _ := new(big.Int).SetString("32000000000000000000", 10)
	assetAddr := usdtAddress
	assetAddrNST := paddingClientChainAddress(assetstype.GenerateNSTAddr(s.ClientChains[0].AddressLength), assetstype.GeneralClientChainAddrLength)
	commonMalleate := func(method string, assetAddr []byte, opAmount *big.Int) (common.Address, []byte) {
		input, err := s.precompile.Pack(
			method,
			uint32(clientChainLzID),
			assetAddr,
			stakerAddr,
			opAmount,
		)
		s.Require().NoError(err, "failed to pack input")
		return s.Address, input
	}
	successRet, err := s.precompile.Methods[assetsprecompile.MethodDepositLST].Outputs.Pack(true, opAmount)
	successRetNST, err := s.precompile.Methods[assetsprecompile.MethodDepositNST].Outputs.Pack(true, opAmount32)
	s.Require().NoError(err)

	testcases := []struct {
		name        string
		malleate    func() (common.Address, []byte)
		readOnly    bool
		expPass     bool
		errContains string
		returnBytes []byte
		extra       func()
	}{
		{
			name: "fail - depositTo transaction will fail because the exocoreLzAppAddress is mismatched",
			malleate: func() (common.Address, []byte) {
				return commonMalleate(assetsprecompile.MethodDepositLST, assetAddr, opAmount)
			},
			readOnly:    false,
			expPass:     false,
			errContains: assetstype.ErrNotAuthorizedGateway.Error(),
		},
		{
			name: "fail - depositTo transaction will fail because the contract caller isn't the exoCoreLzAppAddr",
			malleate: func() (common.Address, []byte) {
				depositModuleParam := &assetstype.Params{
					Gateways: []string{exocoreLzAppAddress},
				}
				err := s.App.AssetsKeeper.SetParams(s.Ctx, depositModuleParam)
				s.Require().NoError(err)
				return commonMalleate(assetsprecompile.MethodDepositLST, assetAddr, opAmount)
			},
			readOnly:    false,
			expPass:     false,
			errContains: assetstype.ErrNotAuthorizedGateway.Error(),
		},
		{
			name: "fail - depositTo transaction will fail because the staked assetsprecompile hasn't been registered",
			malleate: func() (common.Address, []byte) {
				depositModuleParam := &assetstype.Params{
					Gateways: []string{s.Address.String()},
				}
				err := s.App.AssetsKeeper.SetParams(s.Ctx, depositModuleParam)
				s.Require().NoError(err)
				assetAddr = usdcAddress
				return commonMalleate(assetsprecompile.MethodDepositLST, assetAddr, opAmount)
			},
			readOnly:    false,
			expPass:     false,
			errContains: assetstype.ErrNoClientChainAssetKey.Error(),
		},
		{
			name: "pass - depositTo transaction",
			malleate: func() (common.Address, []byte) {
				depositModuleParam := &assetstype.Params{
					Gateways: []string{s.Address.String()},
				}
				assetAddr = usdtAddress
				err := s.App.AssetsKeeper.SetParams(s.Ctx, depositModuleParam)
				s.Require().NoError(err)
				return commonMalleate(assetsprecompile.MethodDepositLST, assetAddr, opAmount)
			},
			returnBytes: successRet,
			readOnly:    false,
			expPass:     true,
		},
		{
			name: "pass - depositNST",
			malleate: func() (common.Address, []byte) {
				depositModuleParam := &assetstype.Params{
					Gateways: []string{s.Address.String()},
				}
				assetAddr = usdtAddress
				err := s.App.AssetsKeeper.SetParams(s.Ctx, depositModuleParam)
				s.Require().NoError(err)
				return commonMalleate(assetsprecompile.MethodDepositNST, assetAddrNST, opAmount32)
			},
			returnBytes: successRetNST,
			readOnly:    false,
			expPass:     true,
			extra: func() {
				amount32 := sdkmath.NewIntWithDecimal(32, 18)
				// check depositNST successfully updated stakerAssetInfo in assets_module
				stakerAssetInfo, _ := s.App.AssetsKeeper.GetStakerSpecifiedAssetInfo(s.Ctx, stakerID, assetID)
				s.Equal(&assetstypes.StakerAssetInfo{
					TotalDepositAmount:        amount32,
					WithdrawableAmount:        amount32,
					PendingUndelegationAmount: sdkmath.ZeroInt(),
				}, stakerAssetInfo)

				// check depositNST successfully updated stakerList in oracle_module
				stakerList := s.App.OracleKeeper.GetStakerList(s.Ctx, assetID)
				s.Equal(stakerList.StakerAddrs[0], stakerAddrStr)
				// check depositNST successfully update stakerInfo with correct validatorPubkey
				stakerInfo := s.App.OracleKeeper.GetStakerInfo(s.Ctx, assetID, stakerAddrStr)
				s.Equal(types.BalanceInfo{
					Block:   1,
					RoundID: 0,
					Change:  types.Action_ACTION_DEPOSIT,
					Balance: 32,
				}, *stakerInfo.BalanceList[0])
			},
		},
	}

	for _, tc := range testcases {
		tc := tc
		s.Run(tc.name, func() {
			// setup basic test suite
			s.SetupTest()

			baseFee := s.App.FeeMarketKeeper.GetBaseFee(s.Ctx)

			// malleate testcase
			caller, input := tc.malleate()

			contract := vm.NewPrecompile(vm.AccountRef(caller), s.precompile, big.NewInt(0), uint64(1e6))
			contract.Input = input

			contractAddr := contract.Address()
			// Build and sign Ethereum transaction
			txArgs := evmtypes.EvmTxArgs{
				ChainID:   s.App.EvmKeeper.ChainID(),
				Nonce:     0,
				To:        &contractAddr,
				Amount:    nil,
				GasLimit:  100000,
				GasPrice:  app.MainnetMinGasPrices.BigInt(),
				GasFeeCap: baseFee,
				GasTipCap: big.NewInt(1),
				Accesses:  &ethtypes.AccessList{},
			}
			msgEthereumTx := evmtypes.NewTx(&txArgs)

			msgEthereumTx.From = s.Address.String()
			err := msgEthereumTx.Sign(s.EthSigner, s.Signer)
			s.Require().NoError(err, "failed to sign Ethereum message")

			// Instantiate config
			proposerAddress := s.Ctx.BlockHeader().ProposerAddress
			cfg, err := s.App.EvmKeeper.EVMConfig(s.Ctx, proposerAddress, s.App.EvmKeeper.ChainID())
			s.Require().NoError(err, "failed to instantiate EVM config")

			msg, err := msgEthereumTx.AsMessage(s.EthSigner, baseFee)
			s.Require().NoError(err, "failed to instantiate Ethereum message")

			// Instantiate EVM
			evm := s.App.EvmKeeper.NewEVM(
				s.Ctx, msg, cfg, nil, s.StateDB,
			)

			params := s.App.EvmKeeper.GetParams(s.Ctx)
			activePrecompiles := params.GetActivePrecompilesAddrs()
			precompileMap := s.App.EvmKeeper.Precompiles(activePrecompiles...)
			err = vm.ValidatePrecompiles(precompileMap, activePrecompiles)
			s.Require().NoError(err, "invalid precompiles", activePrecompiles)
			evm.WithPrecompiles(precompileMap, activePrecompiles)

			// Run precompiled contract
			bz, err := s.precompile.Run(evm, contract, tc.readOnly)

			// Check results
			if tc.expPass {
				s.Require().NoError(err, "expected no error when running the precompile")
				s.Require().Equal(tc.returnBytes, bz, "the return doesn't match the expected result")
			} else {
				/*		s.Require().Error(err, "expected error to be returned when running the precompile")
						s.Require().Nil(bz, "expected returned bytes to be nil")
						s.Require().ErrorContains(err, tc.errContains)*/
				// for failed cases we expect it returns bool value instead of error
				// this is a workaround because the error returned by precompile can not be caught in EVM
				// see https://github.com/ExocoreNetwork/exocore/issues/70
				// TODO: we should figure out root cause and fix this issue to make precompiles work normally
				result, err := s.precompile.ABI.Unpack(assetsprecompile.MethodDepositLST, bz)
				s.Require().NoError(err)
				s.Require().Equal(len(result), 2)
				success, ok := result[0].(bool)
				s.Require().True(ok)
				s.Require().False(success)
			}
			if tc.extra != nil {
				// run extra logic/checking for this test case
				tc.extra()
			}
		})
	}
}

// TestRun tests the precompiled Run method withdraw.
func (s *AssetsPrecompileSuite) TestRunWithdrawPrincipal() {
	// deposit params for test
	usdtAddress := common.FromHex("0xdAC17F958D2ee523a2206206994597C13D831ec7")
	clientChainLzID := 101
	withdrawAmount := big.NewInt(10)
	depositAmount := big.NewInt(100)
	amount32, _ := new(big.Int).SetString("32000000000000000000", 10)
	assetAddr := paddingClientChainAddress(usdtAddress, assetstype.GeneralClientChainAddrLength)
	assetAddrNST := paddingClientChainAddress(assetstype.GenerateNSTAddr(s.ClientChains[0].AddressLength), assetstype.GeneralClientChainAddrLength)
	NSTAddress := assetstype.GenerateNSTAddr(s.ClientChains[0].AddressLength)
	depositAsset := func(staker []byte, depositAmount sdkmath.Int, assetAddress []byte) {
		// deposit asset for withdraw test
		params := &assetskeeper.DepositWithdrawParams{
			ClientChainLzID: 101,
			Action:          assetstype.DepositLST,
			StakerAddress:   staker,
			// AssetsAddress:   usdtAddress,
			AssetsAddress: assetAddress,
			OpAmount:      depositAmount,
		}
		err := s.App.AssetsKeeper.PerformDepositOrWithdraw(s.Ctx, params)
		s.Require().NoError(err)
	}

	commonMalleate := func(method string, assetAddr []byte, withdrawAmount *big.Int) (common.Address, []byte) {
		// Prepare the call input for withdraw test
		input, err := s.precompile.Pack(
			method,
			uint32(clientChainLzID),
			assetAddr,
			paddingClientChainAddress(s.Address.Bytes(), assetstype.GeneralClientChainAddrLength),
			withdrawAmount,
		)
		s.Require().NoError(err, "failed to pack input")
		return s.Address, input
	}
	successRet, err := s.precompile.Methods[assetsprecompile.MethodWithdrawLST].Outputs.Pack(true, new(big.Int).Sub(depositAmount, withdrawAmount))
	successRetNST, err := s.precompile.Methods[assetsprecompile.MethodWithdrawNST].Outputs.Pack(true, big.NewInt(0))
	s.Require().NoError(err)
	testcases := []struct {
		name        string
		malleate    func() (common.Address, []byte)
		readOnly    bool
		expPass     bool
		errContains string
		returnBytes []byte
		extra       func()
	}{
		{
			name: "pass - withdraw via pre-compiles, LST",
			malleate: func() (common.Address, []byte) {
				depositModuleParam := &assetstype.Params{
					Gateways: []string{s.Address.String()},
				}
				err := s.App.AssetsKeeper.SetParams(s.Ctx, depositModuleParam)
				s.Require().NoError(err)
				depositAsset(s.Address.Bytes(), sdkmath.NewIntFromBigInt(depositAmount), usdtAddress)
				return commonMalleate(assetsprecompile.MethodWithdrawLST, assetAddr, withdrawAmount)
			},
			returnBytes: successRet,
			readOnly:    false,
			expPass:     true,
		},
		{
			name: "pass - withdraw via pre-compiles, NST",
			malleate: func() (common.Address, []byte) {
				depositModuleParam := &assetstype.Params{
					Gateways: []string{s.Address.String()},
				}
				err := s.App.AssetsKeeper.SetParams(s.Ctx, depositModuleParam)
				s.Require().NoError(err)
				depositAsset(s.Address.Bytes(), sdkmath.NewIntFromBigInt(amount32), NSTAddress)
				return commonMalleate(assetsprecompile.MethodWithdrawLST, assetAddrNST, amount32)
			},
			returnBytes: successRetNST,
			readOnly:    false,
			expPass:     true,
			extra: func() {
				stakerID, assetID := assetstype.GetStakerIDAndAssetID(s.ClientChains[0].LayerZeroChainID, s.Address.Bytes(), NSTAddress)
				// check depositNST successfully updated stakerAssetInfo in assets_module
				stakerAssetInfo, _ := s.App.AssetsKeeper.GetStakerSpecifiedAssetInfo(s.Ctx, stakerID, assetID)
				s.Equal(&assetstypes.StakerAssetInfo{
					TotalDepositAmount:        sdkmath.ZeroInt(),
					WithdrawableAmount:        sdkmath.ZeroInt(),
					PendingUndelegationAmount: sdkmath.ZeroInt(),
				}, stakerAssetInfo)

				// check depositNST successfully updated stakerList in oracle_module
				stakerList := s.App.OracleKeeper.GetStakerList(s.Ctx, assetID)
				s.Equal(len(stakerList.StakerAddrs), 0)
				// check depositNST successfully update stakerInfo with correct validatorPubkey
				stakerInfo := s.App.OracleKeeper.GetStakerInfo(s.Ctx, assetID, s.Address.String())
				s.Equal(0, len(stakerInfo.BalanceList))
			},
		},
	}
	for _, tc := range testcases {
		tc := tc
		s.Run(tc.name, func() {
			// setup basic test suite
			s.SetupTest()

			baseFee := s.App.FeeMarketKeeper.GetBaseFee(s.Ctx)

			// malleate testcase
			caller, input := tc.malleate()

			contract := vm.NewPrecompile(vm.AccountRef(caller), s.precompile, big.NewInt(0), uint64(1e6))
			contract.Input = input

			contractAddr := contract.Address()
			// Build and sign Ethereum transaction
			txArgs := evmtypes.EvmTxArgs{
				ChainID:   s.App.EvmKeeper.ChainID(),
				Nonce:     0,
				To:        &contractAddr,
				Amount:    nil,
				GasLimit:  100000,
				GasPrice:  app.MainnetMinGasPrices.BigInt(),
				GasFeeCap: baseFee,
				GasTipCap: big.NewInt(1),
				Accesses:  &ethtypes.AccessList{},
			}
			msgEthereumTx := evmtypes.NewTx(&txArgs)

			msgEthereumTx.From = s.Address.String()
			err := msgEthereumTx.Sign(s.EthSigner, s.Signer)
			s.Require().NoError(err, "failed to sign Ethereum message")

			// Instantiate config
			proposerAddress := s.Ctx.BlockHeader().ProposerAddress
			cfg, err := s.App.EvmKeeper.EVMConfig(s.Ctx, proposerAddress, s.App.EvmKeeper.ChainID())
			s.Require().NoError(err, "failed to instantiate EVM config")

			msg, err := msgEthereumTx.AsMessage(s.EthSigner, baseFee)
			s.Require().NoError(err, "failed to instantiate Ethereum message")

			// Create StateDB
			s.StateDB = statedb.New(s.Ctx, s.App.EvmKeeper, statedb.NewEmptyTxConfig(common.BytesToHash(s.Ctx.HeaderHash().Bytes())))
			// Instantiate EVM
			evm := s.App.EvmKeeper.NewEVM(
				s.Ctx, msg, cfg, nil, s.StateDB,
			)
			params := s.App.EvmKeeper.GetParams(s.Ctx)
			activePrecompiles := params.GetActivePrecompilesAddrs()
			precompileMap := s.App.EvmKeeper.Precompiles(activePrecompiles...)
			err = vm.ValidatePrecompiles(precompileMap, activePrecompiles)
			s.Require().NoError(err, "invalid precompiles", activePrecompiles)
			evm.WithPrecompiles(precompileMap, activePrecompiles)

			// Run precompiled contract
			bz, err := s.precompile.Run(evm, contract, tc.readOnly)

			// Check results
			if tc.expPass {
				s.Require().NoError(err, "expected no error when running the precompile")
				s.Require().Equal(tc.returnBytes, bz, "the return doesn't match the expected result")
			} else {
				s.Require().Error(err, "expected error to be returned when running the precompile")
				s.Require().Nil(bz, "expected returned bytes to be nil")
				s.Require().ErrorContains(err, tc.errContains)
			}
			if tc.extra != nil {
				tc.extra()
			}
		})
	}
}

func (s *AssetsPrecompileSuite) TestGetClientChains() {
	input, err := s.precompile.Pack("getClientChains")
	s.Require().NoError(err, "failed to pack input")
	output, err := s.precompile.Methods["getClientChains"].Outputs.Pack(true, []uint32{101})
	s.Require().NoError(err, "failed to pack output")
	s.Run("get client chains", func() {
		s.SetupTest()
		baseFee := s.App.FeeMarketKeeper.GetBaseFee(s.Ctx)
		contract := vm.NewPrecompile(
			vm.AccountRef(s.Address),
			s.precompile,
			big.NewInt(0),
			uint64(1e6),
		)
		contract.Input = input
		contractAddr := contract.Address()
		txArgs := evmtypes.EvmTxArgs{
			ChainID:   s.App.EvmKeeper.ChainID(),
			Nonce:     0,
			To:        &contractAddr,
			Amount:    nil,
			GasLimit:  100000,
			GasPrice:  app.MainnetMinGasPrices.BigInt(),
			GasFeeCap: baseFee,
			GasTipCap: big.NewInt(1),
			Accesses:  &ethtypes.AccessList{},
		}
		msgEthereumTx := evmtypes.NewTx(&txArgs)
		msgEthereumTx.From = s.Address.String()
		err := msgEthereumTx.Sign(s.EthSigner, s.Signer)
		s.Require().NoError(err, "failed to sign Ethereum message")
		proposerAddress := s.Ctx.BlockHeader().ProposerAddress
		cfg, err := s.App.EvmKeeper.EVMConfig(
			s.Ctx, proposerAddress, s.App.EvmKeeper.ChainID(),
		)
		s.Require().NoError(err, "failed to instantiate EVM config")
		msg, err := msgEthereumTx.AsMessage(s.EthSigner, baseFee)
		s.Require().NoError(err, "failed to instantiate Ethereum message")
		evm := s.App.EvmKeeper.NewEVM(
			s.Ctx, msg, cfg, nil, s.StateDB,
		)
		params := s.App.EvmKeeper.GetParams(s.Ctx)
		activePrecompiles := params.GetActivePrecompilesAddrs()
		precompileMap := s.App.EvmKeeper.Precompiles(activePrecompiles...)
		err = vm.ValidatePrecompiles(precompileMap, activePrecompiles)
		s.Require().NoError(err, "invalid precompiles", activePrecompiles)
		evm.WithPrecompiles(precompileMap, activePrecompiles)
		bz, err := s.precompile.Run(evm, contract, true)
		s.Require().NoError(
			err, "expected no error when running the precompile",
		)
		s.Require().Equal(
			output, bz, "the return doesn't match the expected result",
		)
	})
}

func (s *AssetsPrecompileSuite) TestUpdateAuthorizedGateways() {
	testcases := []struct {
		name        string
		malleate    func() (common.Address, []byte)
		readOnly    bool
		expPass     bool
		errContains string
		expResult   bool
	}{
		{
			name: "fail - update gateways for mainnet, authority mismatch",
			malleate: func() (common.Address, []byte) {
				newGateways := []common.Address{
					common.HexToAddress("0x3fC91A3afd70395Cd496C647d5a6CC9D4B2b7FAD"),
					common.HexToAddress("0x4fC91A3afd70395Cd496C647d5a6CC9D4B2b7FAE"),
				}
				input, err := s.precompile.Pack(
					"updateAuthorizedGateways",
					newGateways,
				)
				s.Require().NoError(err)
				return s.Address, input
			},
			readOnly:  false,
			expPass:   true,
			expResult: false,
		},
	}

	for _, tc := range testcases {
		tc := tc
		s.Run(tc.name, func() {
			s.SetupTest() // Reset state for each test case

			// Get caller address and input data
			caller, input := tc.malleate()

			// set up EVM environment
			contract, evm, err := s.setupEVMEnvironment(caller, input, big.NewInt(0))
			s.Require().NoError(err)

			// Execute precompile
			bz, err := s.precompile.Run(evm, contract, tc.readOnly)

			s.Require().NoError(err)
			// we expect no error for both success and failure cases
			success, err := s.precompile.Unpack("updateAuthorizedGateways", bz)

			if tc.expPass {
				s.Require().NoError(err)
				s.Require().Equal(tc.expResult, success[0].(bool))
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *AssetsPrecompileSuite) TestIsAuthorizedGateway() {
	testcases := []struct {
		name        string
		malleate    func() (common.Address, []byte)
		readOnly    bool
		expPass     bool
		expResult   bool
		errContains string
	}{
		{
			name: "pass - gateway is authorized",
			malleate: func() (common.Address, []byte) {
				// First set up an authorized gateway
				params := &assetstypes.Params{
					Gateways: []string{s.Address.String()},
				}
				err := s.App.AssetsKeeper.SetParams(s.Ctx, params)
				s.Require().NoError(err)

				input, err := s.precompile.Pack(
					"isAuthorizedGateway",
					s.Address,
				)
				s.Require().NoError(err)
				return s.Address, input
			},
			readOnly:  true,
			expPass:   true,
			expResult: true,
		},
		{
			name: "pass - gateway is not authorized",
			malleate: func() (common.Address, []byte) {
				// Set up different authorized gateway
				params := &assetstypes.Params{
					Gateways: []string{
						"0x1234567890123456789012345678901234567890",
					},
				}
				err := s.App.AssetsKeeper.SetParams(s.Ctx, params)
				s.Require().NoError(err)

				input, err := s.precompile.Pack(
					"isAuthorizedGateway",
					s.Address,
				)
				s.Require().NoError(err)
				return s.Address, input
			},
			readOnly:  true,
			expPass:   true,
			expResult: false,
		},
		{
			name: "pass - check with empty gateway list",
			malleate: func() (common.Address, []byte) {
				// Set empty gateway list
				params := &assetstypes.Params{
					Gateways: []string{},
				}
				err := s.App.AssetsKeeper.SetParams(s.Ctx, params)
				s.Require().NoError(err)

				input, err := s.precompile.Pack(
					"isAuthorizedGateway",
					s.Address,
				)
				s.Require().NoError(err)
				return s.Address, input
			},
			readOnly:  true,
			expPass:   true,
			expResult: false,
		},
		{
			name: "pass - check zero address",
			malleate: func() (common.Address, []byte) {
				input, err := s.precompile.Pack(
					"isAuthorizedGateway",
					common.Address{},
				)
				s.Require().NoError(err)
				return s.Address, input
			},
			readOnly:  true,
			expPass:   true,
			expResult: false,
		},
	}

	for _, tc := range testcases {
		tc := tc
		s.Run(tc.name, func() {
			s.SetupTest() // Reset state for each test case

			// Get caller address and input data
			caller, input := tc.malleate()

			// Setup EVM environment
			contract, evm, err := s.setupEVMEnvironment(caller, input, big.NewInt(0))
			s.Require().NoError(err)

			// Execute precompile
			bz, err := s.precompile.Run(evm, contract, tc.readOnly)

			if tc.expPass {
				s.Require().NoError(err)
				// Unpack and verify the result
				result, err := s.precompile.Unpack("isAuthorizedGateway", bz)
				s.Require().NoError(err)
				s.Require().Equal(true, result[0].(bool))
				s.Require().Equal(tc.expResult, result[1].(bool))
			} else {
				s.Require().Error(err)
				s.Require().Contains(err.Error(), tc.errContains)
			}
		})
	}
}

func (s *AssetsPrecompileSuite) TestGetTokenInfo() {
	testcases := []struct {
		name        string
		malleate    func() (common.Address, []byte)
		readOnly    bool
		expPass     bool
		returnCheck func([]byte) bool
		errContains string
	}{
		{
			name: "pass - get existing token info (NST)",
			malleate: func() (common.Address, []byte) {
				// NST token is already set up in SetupTest()
				tokenAddr := paddingClientChainAddress(
					common.FromHex("0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"),
					assetstype.GeneralClientChainAddrLength,
				)

				input, err := s.precompile.Pack(
					"getTokenInfo",
					uint32(s.ClientChains[0].LayerZeroChainID),
					tokenAddr,
				)
				s.Require().NoError(err)
				return s.Address, input
			},
			readOnly: true,
			expPass:  true,
			returnCheck: func(bz []byte) bool {
				result, err := s.precompile.Unpack("getTokenInfo", bz)
				s.Require().NoError(err)
				success := result[0].(bool)
				s.Require().True(success)

				tokenInfo := result[1].(struct {
					Name          string   `json:"name"`
					Symbol        string   `json:"symbol"`
					ClientChainID uint32   `json:"clientChainID"`
					TokenID       []byte   `json:"tokenID"`
					Decimals      uint8    `json:"decimals"`
					TotalStaked   *big.Int `json:"totalStaked"`
				})
				return tokenInfo.Name == "Native Restaking ETH" &&
					tokenInfo.Symbol == "NSTETH" &&
					tokenInfo.Decimals == 18 &&
					tokenInfo.TotalStaked.Cmp(s.nstStaked.BigInt()) == 0
			},
		},
		{
			name: "pass - get existing token info (LST)",
			malleate: func() (common.Address, []byte) {
				// Setup LST token first
				s.lstStaked = math.NewInt(100)
				lstToken := &assetstypes.StakingAssetInfo{
					AssetBasicInfo: assetstypes.AssetInfo{
						Name:             "Liquid Staking Token",
						Symbol:           "LST",
						Address:          "0x1234567890123456789012345678901234567890",
						Decimals:         6,
						LayerZeroChainID: uint64(101),
						MetaInfo:         "liquid staking token",
					},
					StakingTotalAmount: s.lstStaked,
				}
				err := s.App.AssetsKeeper.SetStakingAssetInfo(s.Ctx, lstToken)
				s.Require().NoError(err)

				tokenAddr := paddingClientChainAddress(
					common.FromHex(lstToken.AssetBasicInfo.Address),
					assetstype.GeneralClientChainAddrLength,
				)

				input, err := s.precompile.Pack(
					"getTokenInfo",
					uint32(lstToken.AssetBasicInfo.LayerZeroChainID),
					tokenAddr,
				)
				s.Require().NoError(err)
				return s.Address, input
			},
			readOnly: true,
			expPass:  true,
			returnCheck: func(bz []byte) bool {
				result, err := s.precompile.Unpack("getTokenInfo", bz)
				s.Require().NoError(err)
				success := result[0].(bool)
				s.Require().True(success)

				tokenInfo := result[1].(struct {
					Name          string   `json:"name"`
					Symbol        string   `json:"symbol"`
					ClientChainID uint32   `json:"clientChainID"`
					TokenID       []byte   `json:"tokenID"`
					Decimals      uint8    `json:"decimals"`
					TotalStaked   *big.Int `json:"totalStaked"`
				})
				return tokenInfo.Name == "Liquid Staking Token" &&
					tokenInfo.Symbol == "LST" &&
					tokenInfo.Decimals == 6 &&
					tokenInfo.TotalStaked.Cmp(s.lstStaked.BigInt()) == 0
			},
		},
		{
			name: "fail - non-existent token",
			malleate: func() (common.Address, []byte) {
				input, err := s.precompile.Pack(
					"getTokenInfo",
					uint32(999),
					paddingClientChainAddress(
						common.FromHex("0x1234567890123456789012345678901234567890"),
						assetstype.GeneralClientChainAddrLength,
					),
				)
				s.Require().NoError(err)
				return s.Address, input
			},
			readOnly: true,
			expPass:  true, // The call succeeds but returns false
			returnCheck: func(bz []byte) bool {
				result, err := s.precompile.Unpack("getTokenInfo", bz)
				s.Require().NoError(err)
				success := result[0].(bool)
				return !success // Expect false for non-existent token
			},
		},
		{
			name: "fail - invalid chain ID",
			malleate: func() (common.Address, []byte) {
				input, err := s.precompile.Pack(
					"getTokenInfo",
					uint32(0), // invalid chain ID
					paddingClientChainAddress(
						common.FromHex("0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"),
						assetstype.GeneralClientChainAddrLength,
					),
				)
				s.Require().NoError(err)
				return s.Address, input
			},
			readOnly: true,
			expPass:  true, // The call succeeds but returns false
			returnCheck: func(bz []byte) bool {
				result, err := s.precompile.Unpack("getTokenInfo", bz)
				s.Require().NoError(err)
				success := result[0].(bool)
				return !success // Expect false for invalid chain ID
			},
		},
	}

	for _, tc := range testcases {
		tc := tc
		s.Run(tc.name, func() {
			s.SetupTest() // Reset state for each test case

			// Get caller address and input data
			caller, input := tc.malleate()

			// Setup EVM environment
			contract, evm, err := s.setupEVMEnvironment(caller, input, big.NewInt(0))
			s.Require().NoError(err)

			// Execute precompile
			bz, err := s.precompile.Run(evm, contract, tc.readOnly)
			s.Require().NoError(err) // All calls should succeed

			if tc.returnCheck != nil {
				s.Require().True(tc.returnCheck(bz))
			}
		})
	}
}

func (s *AssetsPrecompileSuite) TestGetStakerBalanceByToken() {
	testcases := []struct {
		name        string
		malleate    func() (common.Address, []byte)
		readOnly    bool
		expPass     bool
		returnCheck func([]byte) bool
	}{
		{
			name: "pass - get balance with only asset state",
			malleate: func() (common.Address, []byte) {
				clientChainID := uint32(101)
				tokenAddr := paddingClientChainAddress(
					common.FromHex("0x1234567890123456789012345678901234567890"),
					20,
				)
				stakerAddr := paddingClientChainAddress(
					s.Address.Bytes(),
					20,
				)

				// Setup token
				token := &assetstypes.StakingAssetInfo{
					AssetBasicInfo: assetstypes.AssetInfo{
						Name:             "Test Token",
						Symbol:           "TEST",
						Address:          hexutil.Encode(tokenAddr),
						Decimals:         18,
						LayerZeroChainID: uint64(clientChainID),
					},
					StakingTotalAmount: sdkmath.NewInt(100),
				}
				err := s.App.AssetsKeeper.SetStakingAssetInfo(s.Ctx, token)
				s.Require().NoError(err)

				// Setup staker asset state
				stakerID, assetID := assetstypes.GetStakerIDAndAssetID(uint64(clientChainID), stakerAddr, tokenAddr)
				assetDelta := assetstypes.DeltaStakerSingleAsset{
					TotalDepositAmount:        sdkmath.NewInt(100),
					WithdrawableAmount:        sdkmath.NewInt(70),
					PendingUndelegationAmount: sdkmath.NewInt(30),
				}
				err = s.App.AssetsKeeper.UpdateStakerAssetState(s.Ctx, stakerID, assetID, assetDelta)
				s.Require().NoError(err)

				input, err := s.precompile.Pack(
					"getStakerBalanceByToken",
					clientChainID,
					stakerAddr,
					tokenAddr,
				)
				s.Require().NoError(err)
				return s.Address, input
			},
			readOnly: true,
			expPass:  true,
			returnCheck: func(bz []byte) bool {
				result, err := s.precompile.Unpack("getStakerBalanceByToken", bz)
				s.Require().NoError(err)
				success := result[0].(bool)
				s.Require().True(success)

				balance := result[1].(struct {
					ClientChainID      uint32   `json:"clientChainID"`
					StakerAddress      []byte   `json:"stakerAddress"`
					TokenID            []byte   `json:"tokenID"`
					Balance            *big.Int `json:"balance"`
					Withdrawable       *big.Int `json:"withdrawable"`
					Delegated          *big.Int `json:"delegated"`
					PendingUndelegated *big.Int `json:"pendingUndelegated"`
					TotalDeposited     *big.Int `json:"totalDeposited"`
				})

				return balance.Balance.Cmp(big.NewInt(100)) == 0 && // TotalDepositAmount
					balance.Withdrawable.Cmp(big.NewInt(70)) == 0 && // WithdrawableAmount
					balance.Delegated.Sign() == 0 && // No delegations
					balance.PendingUndelegated.Cmp(big.NewInt(30)) == 0 && // PendingUndelegationAmount
					balance.TotalDeposited.Cmp(big.NewInt(100)) == 0 // TotalDepositAmount
			},
		},
		{
			name: "pass - non-existent token",
			malleate: func() (common.Address, []byte) {
				input, err := s.precompile.Pack(
					"getStakerBalanceByToken",
					uint32(999),
					paddingClientChainAddress(
						common.FromHex("0x1234567890123456789012345678901234567890"),
						assetstype.GeneralClientChainAddrLength,
					),
					paddingClientChainAddress(
						common.FromHex("0xdAC17F958D2ee523a2206206994597C13D831ec7"),
						assetstype.GeneralClientChainAddrLength,
					),
				)
				s.Require().NoError(err)
				return s.Address, input
			},
			readOnly: true,
			expPass:  true,
			returnCheck: func(bz []byte) bool {
				result, err := s.precompile.Unpack("getStakerBalanceByToken", bz)
				s.Require().NoError(err)
				success := result[0].(bool)
				return !success // Should return false for non-existent token
			},
		},
		{
			name: "pass - invalid chain ID",
			malleate: func() (common.Address, []byte) {
				input, err := s.precompile.Pack(
					"getStakerBalanceByToken",
					uint32(0), // invalid chain ID
					paddingClientChainAddress(
						s.Address.Bytes(),
						assetstype.GeneralClientChainAddrLength,
					),
					paddingClientChainAddress(
						common.FromHex("0xdAC17F958D2ee523a2206206994597C13D831ec7"),
						assetstype.GeneralClientChainAddrLength,
					),
				)
				s.Require().NoError(err)
				return s.Address, input
			},
			readOnly: true,
			expPass:  true,
			returnCheck: func(bz []byte) bool {
				result, err := s.precompile.Unpack("getStakerBalanceByToken", bz)
				s.Require().NoError(err)
				success := result[0].(bool)
				return !success // Should return false for invalid chain ID
			},
		},
	}

	for _, tc := range testcases {
		tc := tc
		s.Run(tc.name, func() {
			s.SetupTest() // Reset state for each test case

			// Get caller address and input data
			caller, input := tc.malleate()

			// Setup EVM environment
			contract, evm, err := s.setupEVMEnvironment(caller, input, big.NewInt(0))
			s.Require().NoError(err)

			// Execute precompile
			bz, err := s.precompile.Run(evm, contract, tc.readOnly)
			s.Require().NoError(err) // All calls should succeed

			if tc.returnCheck != nil {
				s.Require().True(tc.returnCheck(bz))
			}
		})
	}
}

// setupEVMEnvironment creates an EVM environment and returns the contract and EVM instance
func (s *AssetsPrecompileSuite) setupEVMEnvironment(
	caller common.Address,
	input []byte,
	value *big.Int,
) (*vm.Contract, *vm.EVM, error) {
	// Create contract
	contract := vm.NewPrecompile(vm.AccountRef(caller), s.precompile, value, uint64(1e6))
	contract.Input = input

	// Get base fee for tx execution
	baseFee := s.App.FeeMarketKeeper.GetBaseFee(s.Ctx)

	// Build and sign Ethereum transaction
	contractAddr := contract.Address()
	txArgs := evmtypes.EvmTxArgs{
		ChainID:   s.App.EvmKeeper.ChainID(),
		Nonce:     0,
		To:        &contractAddr,
		Amount:    nil,
		GasLimit:  100000,
		GasPrice:  app.MainnetMinGasPrices.BigInt(),
		GasFeeCap: baseFee,
		GasTipCap: big.NewInt(1),
		Accesses:  &ethtypes.AccessList{},
	}
	msgEthereumTx := evmtypes.NewTx(&txArgs)
	msgEthereumTx.From = caller.String()
	err := msgEthereumTx.Sign(s.EthSigner, s.Signer)
	if err != nil {
		return nil, nil, err
	}

	// Prepare EVM execution
	proposerAddress := s.Ctx.BlockHeader().ProposerAddress
	cfg, err := s.App.EvmKeeper.EVMConfig(s.Ctx, proposerAddress, s.App.EvmKeeper.ChainID())
	if err != nil {
		return nil, nil, err
	}

	msg, err := msgEthereumTx.AsMessage(s.EthSigner, baseFee)
	if err != nil {
		return nil, nil, err
	}

	// Create EVM instance
	evm := s.App.EvmKeeper.NewEVM(s.Ctx, msg, cfg, nil, s.StateDB)

	// Setup precompiles
	params := s.App.EvmKeeper.GetParams(s.Ctx)
	activePrecompiles := params.GetActivePrecompilesAddrs()
	precompileMap := s.App.EvmKeeper.Precompiles(activePrecompiles...)
	err = vm.ValidatePrecompiles(precompileMap, activePrecompiles)
	if err != nil {
		return nil, nil, err
	}
	evm.WithPrecompiles(precompileMap, activePrecompiles)

	return contract, evm, nil
}
