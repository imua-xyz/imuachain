package delegation_test

import (
	"math/big"

	assetskeeper "github.com/ExocoreNetwork/exocore/x/assets/keeper"

	operatortypes "github.com/ExocoreNetwork/exocore/x/operator/types"

	sdkmath "cosmossdk.io/math"

	"github.com/ExocoreNetwork/exocore/app"
	"github.com/ExocoreNetwork/exocore/precompiles/delegation"
	"github.com/ExocoreNetwork/exocore/x/assets/types"
	assetstype "github.com/ExocoreNetwork/exocore/x/assets/types"
	delegationtype "github.com/ExocoreNetwork/exocore/x/delegation/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/evmos/evmos/v16/x/evm/statedb"
	evmtypes "github.com/evmos/evmos/v16/x/evm/types"
)

func (s *DelegationPrecompileSuite) TestIsTransaction() {
	testCases := []struct {
		name   string
		method string
		isTx   bool
	}{
		{
			delegation.MethodDelegate,
			s.precompile.Methods[delegation.MethodDelegate].Name,
			true,
		},
		{
			delegation.MethodUndelegate,
			s.precompile.Methods[delegation.MethodUndelegate].Name,
			true,
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

// TestRun tests Delegate method through calling Run function.
func (s *DelegationPrecompileSuite) TestRunDelegate() {
	// deposit params for test
	exocoreLzAppAddress := "0x3fC91A3afd70395Cd496C647d5a6CC9D4B2b7FAD"
	usdtAddress := common.FromHex("0xdAC17F958D2ee523a2206206994597C13D831ec7")
	opAccAddr := "exo13h6xg79g82e2g2vhjwg7j4r2z2hlncelwutkjr"
	clientChainLzID := 101
	lzNonce := 0
	delegationAmount := big.NewInt(50)
	depositAmount := big.NewInt(100)
	smallDepositAmount := big.NewInt(20)
	assetAddr := paddingClientChainAddress(usdtAddress, types.GeneralClientChainAddrLength)
	depositAsset := func(staker []byte, depositAmount sdkmath.Int) {
		// deposit asset for delegation test
		params := &assetskeeper.DepositWithdrawParams{
			ClientChainLzID: 101,
			Action:          types.DepositLST,
			StakerAddress:   staker,
			AssetsAddress:   usdtAddress,
			OpAmount:        depositAmount,
		}
		err := s.App.AssetsKeeper.PerformDepositOrWithdraw(s.Ctx, params)
		s.Require().NoError(err)
	}
	registerOperator := func() {
		registerReq := &operatortypes.RegisterOperatorReq{
			FromAddress: opAccAddr,
			Info: &operatortypes.OperatorInfo{
				EarningsAddr: opAccAddr,
			},
		}
		_, err := s.OperatorMsgServer.RegisterOperator(s.Ctx, registerReq)
		s.NoError(err)
	}
	commonMalleate := func() (common.Address, []byte) {
		// prepare the call input for delegation test
		input, err := s.precompile.Pack(
			delegation.MethodDelegate,
			uint32(clientChainLzID),
			uint64(lzNonce),
			assetAddr,
			paddingClientChainAddress(s.Address.Bytes(), types.GeneralClientChainAddrLength),
			[]byte(opAccAddr),
			delegationAmount,
		)
		s.Require().NoError(err, "failed to pack input")
		return s.Address, input
	}
	successRet, err := s.precompile.Methods[delegation.MethodDelegate].Outputs.Pack(true)
	s.Require().NoError(err)

	failureRet, err := s.precompile.Methods[delegation.MethodDelegate].Outputs.Pack(false)
	s.Require().NoError(err)

	testcases := []struct {
		name        string
		malleate    func() (common.Address, []byte)
		readOnly    bool
		expPass     bool
		errContains string
		returnBytes []byte
	}{
		{
			name: "fail - delegateToThroughClientChain transaction will fail because the exocoreLzAppAddress is mismatched",
			malleate: func() (common.Address, []byte) {
				return commonMalleate()
			},
			readOnly:    false,
			expPass:     false,
			returnBytes: failureRet,
		},
		{
			name: "fail - delegateToThroughClientChain transaction will fail because the contract caller isn't the exoCoreLzAppAddr",
			malleate: func() (common.Address, []byte) {
				depositModuleParam := &assetstype.Params{
					Gateways: []string{exocoreLzAppAddress},
				}
				err := s.App.AssetsKeeper.SetParams(s.Ctx, depositModuleParam)
				s.Require().NoError(err)
				return commonMalleate()
			},
			readOnly:    false,
			expPass:     false,
			returnBytes: failureRet,
		},
		{
			name: "fail - delegateToThroughClientChain transaction will fail because the delegated operator hasn't been registered",
			malleate: func() (common.Address, []byte) {
				depositModuleParam := &assetstype.Params{
					Gateways: []string{s.Address.String()},
				}
				err := s.App.AssetsKeeper.SetParams(s.Ctx, depositModuleParam)
				s.Require().NoError(err)
				return commonMalleate()
			},
			readOnly:    false,
			expPass:     false,
			returnBytes: failureRet,
		},
		{
			name: "fail - delegateToThroughClientChain transaction will fail because the delegated asset hasn't been deposited",
			malleate: func() (common.Address, []byte) {
				depositModuleParam := &assetstype.Params{
					Gateways: []string{s.Address.String()},
				}
				err := s.App.AssetsKeeper.SetParams(s.Ctx, depositModuleParam)
				s.Require().NoError(err)
				registerOperator()
				return commonMalleate()
			},
			readOnly:    false,
			expPass:     false,
			returnBytes: failureRet,
		},
		{
			name: "fail - delegateToThroughClientChain transaction will fail because the delegation amount is bigger than the canWithdraw amount",
			malleate: func() (common.Address, []byte) {
				depositModuleParam := &assetstype.Params{
					Gateways: []string{s.Address.String()},
				}
				err := s.App.AssetsKeeper.SetParams(s.Ctx, depositModuleParam)
				s.Require().NoError(err)
				registerOperator()
				depositAsset(s.Address.Bytes(), sdkmath.NewIntFromBigInt(smallDepositAmount))
				return commonMalleate()
			},
			readOnly:    false,
			expPass:     false,
			returnBytes: failureRet,
		},
		{
			name: "pass - delegateToThroughClientChain transaction",
			malleate: func() (common.Address, []byte) {
				assetsModuleParam := &assetstype.Params{
					Gateways: []string{s.Address.String()},
				}
				err := s.App.AssetsKeeper.SetParams(s.Ctx, assetsModuleParam)
				s.Require().NoError(err)
				registerOperator()
				depositAsset(s.Address.Bytes(), sdkmath.NewIntFromBigInt(depositAmount))
				return commonMalleate()
			},
			returnBytes: successRet,
			readOnly:    false,
			expPass:     true,
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
				// for failed cases we expect it returns bool value instead of error
				// this is a workaround because the error returned by precompile can not be caught in EVM
				// see https://github.com/ExocoreNetwork/exocore/issues/70
				// TODO: we should figure out root cause and fix this issue to make precompiles work normally
				s.Require().NoError(err, "expected no error when running the precompile")
				s.Require().Equal(tc.returnBytes, bz, "expected returned bytes to be nil")
			}
		})
	}
}

// TestRun tests Delegate method through calling Run function.
func (s *DelegationPrecompileSuite) TestRunUnDelegate() {
	// deposit params for test
	usdtAddress := common.FromHex("0xdAC17F958D2ee523a2206206994597C13D831ec7")
	operatorAddr := "exo13h6xg79g82e2g2vhjwg7j4r2z2hlncelwutkjr"
	clientChainLzID := 101
	lzNonce := uint64(0)
	delegationAmount := big.NewInt(50)
	depositAmount := big.NewInt(100)
	assetAddr := paddingClientChainAddress(usdtAddress, types.GeneralClientChainAddrLength)
	depositAsset := func(staker []byte, depositAmount sdkmath.Int) {
		// deposit asset for delegation test
		params := &assetskeeper.DepositWithdrawParams{
			ClientChainLzID: 101,
			Action:          types.DepositLST,
			StakerAddress:   staker,
			AssetsAddress:   usdtAddress,
			OpAmount:        depositAmount,
		}
		err := s.App.AssetsKeeper.PerformDepositOrWithdraw(s.Ctx, params)
		s.Require().NoError(err)
	}

	delegateAsset := func(staker []byte, delegateAmount sdkmath.Int) {
		// deposit asset for delegation test
		delegateToParams := &delegationtype.DelegationOrUndelegationParams{
			ClientChainID: 101,
			Action:        types.DelegateTo,
			StakerAddress: staker,
			AssetsAddress: usdtAddress,
			OpAmount:      delegateAmount,
		}
		opAccAddr, err := sdk.AccAddressFromBech32(operatorAddr)
		s.Require().NoError(err)
		delegateToParams.OperatorAddress = opAccAddr
		err = s.App.DelegationKeeper.DelegateTo(s.Ctx, delegateToParams)
		s.Require().NoError(err)
	}
	registerOperator := func() {
		registerReq := &operatortypes.RegisterOperatorReq{
			FromAddress: operatorAddr,
			Info: &operatortypes.OperatorInfo{
				EarningsAddr: operatorAddr,
			},
		}
		_, err := s.OperatorMsgServer.RegisterOperator(s.Ctx, registerReq)
		s.NoError(err)
	}
	commonMalleate := func() (common.Address, []byte) {
		// prepare the call input for delegation test
		input, err := s.precompile.Pack(
			delegation.MethodUndelegate,
			uint32(clientChainLzID),
			lzNonce+1,
			assetAddr,
			paddingClientChainAddress(s.Address.Bytes(), types.GeneralClientChainAddrLength),
			[]byte(operatorAddr),
			delegationAmount,
		)
		s.Require().NoError(err, "failed to pack input")
		return s.Address, input
	}
	successRet, err := s.precompile.Methods[delegation.MethodUndelegate].Outputs.Pack(true)
	s.Require().NoError(err)

	testcases := []struct {
		name        string
		malleate    func() (common.Address, []byte)
		readOnly    bool
		expPass     bool
		errContains string
		returnBytes []byte
	}{
		{
			name: "pass - undelegateFromThroughClientChain transaction",
			malleate: func() (common.Address, []byte) {
				depositModuleParam := &assetstype.Params{
					Gateways: []string{s.Address.String()},
				}
				err := s.App.AssetsKeeper.SetParams(s.Ctx, depositModuleParam)
				s.Require().NoError(err)
				registerOperator()
				depositAsset(s.Address.Bytes(), sdkmath.NewIntFromBigInt(depositAmount))
				delegateAsset(s.Address.Bytes(), sdkmath.NewIntFromBigInt(delegationAmount))
				return commonMalleate()
			},
			returnBytes: successRet,
			readOnly:    false,
			expPass:     true,
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

			// set txHash for delegation module
			s.Ctx = s.Ctx.WithValue(delegation.CtxKeyTxHash, common.HexToHash(msgEthereumTx.Hash))
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
				// for failed cases we expect it returns bool value instead of error
				// this is a workaround because the error returned by precompile can not be caught in EVM
				// see https://github.com/ExocoreNetwork/exocore/issues/70
				// TODO: we should figure out root cause and fix this issue to make precompiles work normally
				s.Require().NoError(err, "expected no error when running the precompile")
				s.Require().Equal(tc.returnBytes, bz, "expected returned bytes to be nil")
			}
		})
	}
}
