package reward_test

import (
	"math/big"

	assetskeeper "github.com/ExocoreNetwork/exocore/x/assets/keeper"

	sdkmath "cosmossdk.io/math"
	"github.com/ExocoreNetwork/exocore/app"
	"github.com/ExocoreNetwork/exocore/precompiles/reward"
	assetstype "github.com/ExocoreNetwork/exocore/x/assets/types"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/evmos/evmos/v16/x/evm/statedb"
	evmtypes "github.com/evmos/evmos/v16/x/evm/types"
)

func (s *RewardPrecompileTestSuite) TestIsTransaction() {
	testCases := []struct {
		name   string
		method string
		isTx   bool
	}{
		{
			reward.MethodReward,
			s.precompile.Methods[reward.MethodReward].Name,
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

// TestRun tests the precompiled Run method reward.
func (s *RewardPrecompileTestSuite) TestRunRewardThroughClientChain() {
	// deposit params for test
	exocoreLzAppEventTopic := "0xc6a377bfc4eb120024a8ac08eef205be16b817020812c73223e81d1bdb9708ec"
	usdtAddress := common.FromHex("0xdAC17F958D2ee523a2206206994597C13D831ec7")
	clientChainLzID := 101
	withdrawAmount := big.NewInt(10)
	depositAmount := big.NewInt(100)
	assetAddr := paddingClientChainAddress(usdtAddress, assetstype.GeneralClientChainAddrLength)
	depositAsset := func(staker []byte, depositAmount sdkmath.Int) {
		// deposit asset for reward test
		params := &assetskeeper.DepositWithdrawParams{
			ClientChainLzID: 101,
			Action:          assetstype.DepositLST,
			StakerAddress:   staker,
			AssetsAddress:   usdtAddress,
			OpAmount:        depositAmount,
		}
		err := s.App.AssetsKeeper.PerformDepositOrWithdraw(s.Ctx, params)
		s.Require().NoError(err)
	}

	commonMalleate := func() (common.Address, []byte) {
		// Prepare the call input for reward test
		input, err := s.precompile.Pack(
			reward.MethodReward,
			uint32(clientChainLzID),
			assetAddr,
			paddingClientChainAddress(s.Address.Bytes(), assetstype.GeneralClientChainAddrLength),
			withdrawAmount,
		)
		s.Require().NoError(err, "failed to pack input")
		return s.Address, input
	}
	// successRet, err := s.precompile.Methods[reward.MethodReward].Outputs.Pack(true, new(big.Int).Add(depositAmount, withdrawAmount))
	// TODO: reward precompile is disabled, so it always errors and returns fail
	successRet, err := s.precompile.Methods[reward.MethodReward].Outputs.Pack(false, new(big.Int))
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
			name: "pass - reward via pre-compiles",
			malleate: func() (common.Address, []byte) {
				depositModuleParam := &assetstype.Params{
					ExocoreLzAppAddress:    s.Address.String(),
					ExocoreLzAppEventTopic: exocoreLzAppEventTopic,
				}
				err := s.App.AssetsKeeper.SetParams(s.Ctx, depositModuleParam)
				s.Require().NoError(err)
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
