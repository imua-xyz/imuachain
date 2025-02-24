package bech32_test

import (
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	evmtypes "github.com/evmos/evmos/v16/x/evm/types"
	"github.com/imua-xyz/imuachain/app"
	"github.com/imua-xyz/imuachain/cmd/config"
	"github.com/imua-xyz/imuachain/precompiles/bech32"
	testutiltx "github.com/imua-xyz/imuachain/testutil/tx"
)

// TestRun tests the precompile's Run method.
func (s *Bech32PrecompileSuite) TestRun() {
	inputAddr := testutiltx.GenerateAddress()
	contract := vm.NewPrecompile(
		vm.AccountRef(inputAddr), s.precompile, big.NewInt(0), uint64(1000000),
	)

	testCases := []struct {
		name        string
		malleate    func() *vm.Contract
		postCheck   func(data []byte)
		expPass     bool
		errContains string
	}{
		{
			"fail - invalid method",
			func() *vm.Contract {
				contract.Input = []byte("invalid")
				return contract
			},
			nil,
			false,
			"no method with id",
		},
		{
			"fail - error during unpack",
			func() *vm.Contract {
				contract.Input = s.precompile.Methods[bech32.MethodHexToBech32].ID
				return contract
			},
			nil,
			false,
			"abi: attempting to unmarshall an empty string while arguments are expected",
		},
		{
			"fail - HexToBech32 method error",
			func() *vm.Contract {
				input, err := s.precompile.Pack(bech32.MethodHexToBech32, inputAddr, "")
				s.Require().NoError(err, "failed to pack input")
				contract.Input = input
				return contract
			},
			nil,
			false,
			"empty bech32 prefix provided, expected a non-empty string",
		},
		{
			"pass - hex to bech32 account (im)",
			func() *vm.Contract {
				input, err := s.precompile.Pack(
					bech32.MethodHexToBech32, inputAddr, config.Bech32Prefix,
				)
				s.Require().NoError(err, "failed to pack input")
				contract.Input = input
				return contract
			},
			func(data []byte) {
				args, err := s.precompile.Unpack(bech32.MethodHexToBech32, data)
				s.Require().NoError(err, "failed to unpack output")
				s.Require().Len(args, 1)
				addr, ok := args[0].(string)
				s.Require().True(ok)
				s.Require().Equal(sdk.AccAddress(inputAddr.Bytes()).String(), addr)
			},
			true,
			"",
		},
		{
			"pass - hex to bech32 validator operator (imvaloper)",
			func() *vm.Contract {
				input, err := s.precompile.Pack(
					bech32.MethodHexToBech32, inputAddr, config.Bech32PrefixValAddr,
				)
				s.Require().NoError(err, "failed to pack input")
				contract.Input = input
				return contract
			},
			func(data []byte) {
				args, err := s.precompile.Unpack(bech32.MethodHexToBech32, data)
				s.Require().NoError(err, "failed to unpack output")
				s.Require().Len(args, 1)
				addr, ok := args[0].(string)
				s.Require().True(ok)
				s.Require().Equal(sdk.ValAddress(inputAddr.Bytes()).String(), addr)
			},
			true,
			"",
		},
		{
			"pass - hex to bech32 consensus address (imvalcons)",
			func() *vm.Contract {
				input, err := s.precompile.Pack(
					bech32.MethodHexToBech32, inputAddr, config.Bech32PrefixConsAddr,
				)
				s.Require().NoError(err, "failed to pack input")
				contract.Input = input
				return contract
			},
			func(data []byte) {
				args, err := s.precompile.Unpack(bech32.MethodHexToBech32, data)
				s.Require().NoError(err, "failed to unpack output")
				s.Require().Len(args, 1)
				addr, ok := args[0].(string)
				s.Require().True(ok)
				s.Require().Equal(sdk.ConsAddress(inputAddr.Bytes()).String(), addr)
			},
			true,
			"",
		},
		{
			"pass - bech32 to hex account address",
			func() *vm.Contract {
				input, err := s.precompile.Pack(
					bech32.MethodBech32ToHex, sdk.AccAddress(inputAddr.Bytes()).String(),
				)
				s.Require().NoError(err, "failed to pack input")
				contract.Input = input
				return contract
			},
			func(data []byte) {
				args, err := s.precompile.Unpack(bech32.MethodBech32ToHex, data)
				s.Require().NoError(err, "failed to unpack output")
				s.Require().Len(args, 1)
				addr, ok := args[0].(common.Address)
				s.Require().True(ok)
				s.Require().Equal(inputAddr, addr)
			},
			true,
			"",
		},
		{
			"pass - bech32 to hex validator address",
			func() *vm.Contract {
				input, err := s.precompile.Pack(
					bech32.MethodBech32ToHex, sdk.ValAddress(inputAddr.Bytes()).String(),
				)
				s.Require().NoError(err, "failed to pack input")
				contract.Input = input
				return contract
			},
			func(data []byte) {
				args, err := s.precompile.Unpack(bech32.MethodBech32ToHex, data)
				s.Require().NoError(err, "failed to unpack output")
				s.Require().Len(args, 1)
				addr, ok := args[0].(common.Address)
				s.Require().True(ok)
				s.Require().Equal(inputAddr, addr)
			},
			true,
			"",
		},
		{
			"pass - bech32 to hex consensus address",
			func() *vm.Contract {
				input, err := s.precompile.Pack(
					bech32.MethodBech32ToHex, sdk.ValAddress(inputAddr.Bytes()).String(),
				)
				s.Require().NoError(err, "failed to pack input")
				contract.Input = input
				return contract
			},
			func(data []byte) {
				args, err := s.precompile.Unpack(bech32.MethodBech32ToHex, data)
				s.Require().NoError(err, "failed to unpack output")
				s.Require().Len(args, 1)
				addr, ok := args[0].(common.Address)
				s.Require().True(ok)
				s.Require().Equal(inputAddr, addr)
			},
			true,
			"",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			contract := tc.malleate()
			evm := s.getEvm(contract)
			bz, err := s.precompile.Run(evm, contract, true)
			// Check results
			if tc.expPass {
				// check test validity
				s.Require().Empty(tc.errContains)
				s.Require().NoError(err, "expected no error when running the precompile")
				s.Require().NotNil(bz, "expected returned bytes not to be nil")
				if tc.postCheck != nil {
					tc.postCheck(bz)
				}
			} else {
				s.Require().Error(err, "expected error to be returned when running the precompile")
				s.Require().Nil(bz, "expected returned bytes to be nil")
				s.Require().ErrorContains(err, tc.errContains)
				s.Require().Nil(tc.postCheck)
			}
		})
	}
}

func (s *Bech32PrecompileSuite) getEvm(contract *vm.Contract) *vm.EVM {
	baseFee := s.App.FeeMarketKeeper.GetBaseFee(s.Ctx)
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
	return evm
}
