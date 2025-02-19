package bls_test

import (
	"math/big"

	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/imua-xyz/imuachain/precompiles/bls"
	"github.com/prysmaticlabs/prysm/v4/crypto/bls/blst"
)

// TestRun tests the precompiles Run method.
func (s *PrecompileTestSuite) TestRun() {
	contract := vm.NewPrecompile(
		vm.AccountRef(s.caller),
		s.precompile,
		big.NewInt(0),
		uint64(1000000),
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
			func(data []byte) {},
			false,
			"no method with id",
		},
		{
			"fail - error during unpack",
			func() *vm.Contract {
				// only pass the method ID to the input
				contract.Input = s.precompile.Methods[bls.MethodVerify].ID
				return contract
			},
			func(data []byte) {},
			false,
			"abi: attempting to unmarshall an empty string while arguments are expected",
		},
		{
			"pass - verify aggregated signature and aggregated public key",
			func() *vm.Contract {
				input, err := s.precompile.Pack(
					bls.MethodVerify,
					s.msg,
					s.signature.Marshal(),
					s.aggregatedPublicKey.Marshal(),
				)
				s.Require().NoError(err, "failed to pack input")
				contract.Input = input
				return contract
			},
			func(data []byte) {
				args, err := s.precompile.Unpack(bls.MethodVerify, data)
				s.Require().NoError(err, "failed to unpack output")
				s.Require().Len(args, 1)
				valid, ok := args[0].(bool)
				s.Require().True(ok)
				s.Require().True(valid)
			},
			true,
			"",
		},
		{
			"pass - verify aggregated signature and corresponding public keys ",
			func() *vm.Contract {
				rawPubkeys := make([][]byte, len(s.publicKeys))
				for i, pubKey := range s.publicKeys {
					rawPubkeys[i] = pubKey.Marshal()
				}
				input, err := s.precompile.Pack(
					bls.MethodFastAggregateVerify,
					s.msg,
					s.signature.Marshal(),
					rawPubkeys,
				)
				s.Require().NoError(err, "failed to pack input")
				contract.Input = input
				return contract
			},
			func(data []byte) {
				args, err := s.precompile.Unpack(bls.MethodFastAggregateVerify, data)
				s.Require().NoError(err, "failed to unpack output")
				s.Require().Len(args, 1)
				valid, ok := args[0].(bool)
				s.Require().True(ok)
				s.Require().True(valid)
			},
			true,
			"",
		},
		{
			"pass - aggregate public keys ",
			func() *vm.Contract {
				rawPubkeys := make([][]byte, len(s.publicKeys))
				for i, pubKey := range s.publicKeys {
					rawPubkeys[i] = pubKey.Marshal()
				}
				input, err := s.precompile.Pack(
					bls.MethodAggregatePubkeys,
					rawPubkeys,
				)
				s.Require().NoError(err, "failed to pack input")
				contract.Input = input
				return contract
			},
			func(data []byte) {
				args, err := s.precompile.Unpack(bls.MethodAggregatePubkeys, data)
				s.Require().NoError(err, "failed to unpack output")
				s.Require().Len(args, 1)
				aggPubkey, ok := args[0].([]byte)
				s.Require().True(ok)
				_, err = blst.PublicKeyFromBytes(aggPubkey)
				s.Require().NoError(err)
			},
			true,
			"",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// setup basic test suite
			s.SetupTest()

			// malleate testcase
			contract := tc.malleate()

			// Run precompiled contract

			// NOTE: we can ignore the EVM and readonly args since it's a stateless-
			// precompiled contract
			bz, err := s.precompile.Run(nil, contract, true)

			// Check results
			if tc.expPass {
				s.Require().NoError(err, "expected no error when running the precompile")
				s.Require().NotNil(bz, "expected returned bytes not to be nil")
				tc.postCheck(bz)
			} else {
				s.Require().Error(err, "expected error to be returned when running the precompile")
				s.Require().Nil(bz, "expected returned bytes to be nil")
				s.Require().ErrorContains(err, tc.errContains)
			}
		})
	}
}
