package reward_test

import (
	"bytes"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/imua-xyz/imuachain/precompiles/reward"
)

// TestWrapperStructsDecoding tests that all wrapper structs correctly decode struct-based parameters
// using method.Inputs.Copy(). This verifies the wrapper approach works for all struct-based functions.
func (s *RewardPrecompileTestSuite) TestWrapperStructsDecoding() {
	testCases := []struct {
		name     string
		method   string
		packFunc func() ([]byte, error)
		verify   func(interface{}) bool
	}{
		{
			name:   "WithdrawRewardArgsWrapper",
			method: reward.MethodWithdrawReward,
			packFunc: func() ([]byte, error) {
				params := reward.WithdrawRewardArgs{
					DoClaim:              true,
					ClientChainLzID:      1,
					RewardAssetChainLzID: 1,
					AssetAddress:         paddingClientChainAddress(common.HexToAddress("0x1234567890123456789012345678901234567890").Bytes(), 32),
					StakerAddress:        paddingClientChainAddress(common.HexToAddress("0xabcdefabcdefabcdefabcdefabcdefabcdefabcd").Bytes(), 32),
					OpAmount:             big.NewInt(1000),
				}
				return s.precompile.ABI.Pack(reward.MethodWithdrawReward, params)
			},
			verify: func(v interface{}) bool {
				wrapper, ok := v.(reward.WithdrawRewardArgsWrapper)
				if !ok {
					return false
				}
				args := wrapper.Params
				return args.DoClaim == true &&
					args.ClientChainLzID == 1 &&
					args.RewardAssetChainLzID == 1 &&
					args.OpAmount.Cmp(big.NewInt(1000)) == 0 &&
					bytes.Equal(args.AssetAddress, paddingClientChainAddress(common.HexToAddress("0x1234567890123456789012345678901234567890").Bytes(), 32)) &&
					bytes.Equal(args.StakerAddress, paddingClientChainAddress(common.HexToAddress("0xabcdefabcdefabcdefabcdefabcdefabcdefabcd").Bytes(), 32))
			},
		},
		{
			name:   "WithdrawIMUATokenRewardArgsWrapper",
			method: reward.MethodWithdrawIMUATokenReward,
			packFunc: func() ([]byte, error) {
				params := reward.WithdrawIMUATokenRewardArgs{
					DoClaim:         true,
					ClientChainLzID: 1,
					StakerAddress:   paddingClientChainAddress(common.HexToAddress("0xabcdefabcdefabcdefabcdefabcdefabcdefabcd").Bytes(), 32),
					ReceiptAddress:  common.HexToAddress("0x1111111111111111111111111111111111111111").Bytes(),
					OpAmount:        big.NewInt(2000),
				}
				return s.precompile.ABI.Pack(reward.MethodWithdrawIMUATokenReward, params)
			},
			verify: func(v interface{}) bool {
				wrapper, ok := v.(reward.WithdrawIMUATokenRewardArgsWrapper)
				if !ok {
					return false
				}
				args := wrapper.Params
				return args.DoClaim == true &&
					args.ClientChainLzID == 1 &&
					args.OpAmount.Cmp(big.NewInt(2000)) == 0 &&
					bytes.Equal(args.StakerAddress, paddingClientChainAddress(common.HexToAddress("0xabcdefabcdefabcdefabcdefabcdefabcdefabcd").Bytes(), 32)) &&
					bytes.Equal(args.ReceiptAddress, common.HexToAddress("0x1111111111111111111111111111111111111111").Bytes())
			},
		},
		{
			name:   "UndelegateRewardArgsWrapper",
			method: reward.MethodUndelegateReward,
			packFunc: func() ([]byte, error) {
				operatorAddr := make([]byte, 32)
				copy(operatorAddr, []byte("operator1234567890123456789012345"))
				params := reward.UndelegateRewardArgs{
					ClientChainLzID:      1,
					RewardAssetChainLzID: 1,
					AssetAddress:         paddingClientChainAddress(common.HexToAddress("0x1234567890123456789012345678901234567890").Bytes(), 32),
					StakerAddress:        paddingClientChainAddress(common.HexToAddress("0xabcdefabcdefabcdefabcdefabcdefabcdefabcd").Bytes(), 32),
					OperatorAddr:         string(operatorAddr),
					OpAmount:             big.NewInt(3000),
					InstantUnbond:        false,
				}
				return s.precompile.ABI.Pack(reward.MethodUndelegateReward, params)
			},
			verify: func(v interface{}) bool {
				wrapper, ok := v.(reward.UndelegateRewardArgsWrapper)
				if !ok {
					return false
				}
				args := wrapper.Params
				operatorAddr := make([]byte, 32)
				copy(operatorAddr, []byte("operator1234567890123456789012345"))
				return args.ClientChainLzID == 1 &&
					args.RewardAssetChainLzID == 1 &&
					args.OpAmount.Cmp(big.NewInt(3000)) == 0 &&
					args.InstantUnbond == false &&
					bytes.Equal(args.AssetAddress, paddingClientChainAddress(common.HexToAddress("0x1234567890123456789012345678901234567890").Bytes(), 32)) &&
					bytes.Equal(args.StakerAddress, paddingClientChainAddress(common.HexToAddress("0xabcdefabcdefabcdefabcdefabcdefabcdefabcd").Bytes(), 32)) &&
					args.OperatorAddr == string(operatorAddr)
			},
		},
		{
			name:   "RegisterRewardTokenArgsWrapper",
			method: reward.MethodRegisterRewardToken,
			packFunc: func() ([]byte, error) {
				params := reward.RegisterRewardTokenArgs{
					ClientChainID:        1,
					Token:                paddingClientChainAddress(common.HexToAddress("0x1234567890123456789012345678901234567890").Bytes(), 32),
					Decimals:             18,
					Name:                 "TestToken",
					Symbol:               "TEST",
					MetaData:             "Test metadata",
					Denomination:         "test",
					DenominationExponent: 0,
				}
				return s.precompile.ABI.Pack(reward.MethodRegisterRewardToken, params)
			},
			verify: func(v interface{}) bool {
				wrapper, ok := v.(reward.RegisterRewardTokenArgsWrapper)
				if !ok {
					return false
				}
				args := wrapper.Params
				return args.ClientChainID == 1 &&
					args.Decimals == 18 &&
					args.Name == "TestToken" &&
					args.Symbol == "TEST" &&
					args.Denomination == "test" &&
					bytes.Equal(args.Token, paddingClientChainAddress(common.HexToAddress("0x1234567890123456789012345678901234567890").Bytes(), 32)) &&
					args.MetaData == "Test metadata" &&
					args.DenominationExponent == 0
			},
		},
		{
			name:   "AVSRewardDistributionInfoArgsWrapper",
			method: reward.MethodSetAVSRewardDistribution,
			packFunc: func() ([]byte, error) {
				params := reward.AVSRewardDistributionInfoArgs{
					RewardCoins: []reward.ABIRewardCoin{
						{
							Denomination: "test",
							Amount:       big.NewInt(1000),
						},
					},
					OperatorRewardProportions: []reward.ABIOperatorRewardProportion{
						{
							Operator:    "operator1",
							Numerator:   big.NewInt(1),
							Denominator: big.NewInt(2),
						},
					},
				}
				return s.precompile.ABI.Pack(reward.MethodSetAVSRewardDistribution, params)
			},
			verify: func(v interface{}) bool {
				wrapper, ok := v.(reward.AVSRewardDistributionInfoArgsWrapper)
				if !ok {
					return false
				}
				args := wrapper.RewardDistribution
				return len(args.RewardCoins) == 1 &&
					args.RewardCoins[0].Denomination == "test" &&
					args.RewardCoins[0].Amount.Cmp(big.NewInt(1000)) == 0 &&
					len(args.OperatorRewardProportions) == 1 &&
					args.OperatorRewardProportions[0].Operator == "operator1" &&
					args.OperatorRewardProportions[0].Numerator.Cmp(big.NewInt(1)) == 0 &&
					args.OperatorRewardProportions[0].Denominator.Cmp(big.NewInt(2)) == 0
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Pack the struct parameters using ABI
			packed, err := tc.packFunc()
			s.Require().NoError(err, "failed to pack struct parameters")

			// Get the method from ABI
			method, ok := s.precompile.ABI.Methods[tc.method]
			s.Require().True(ok, "method %s not found in ABI", tc.method)

			// Unpack the input (skip the 4-byte method selector)
			args, err := method.Inputs.Unpack(packed[4:])
			s.Require().NoError(err, "failed to unpack struct parameters")
			s.Require().Len(args, 1, "should have one argument (the struct)")

			// Test Copy() with wrapper structs
			switch tc.method {
			case reward.MethodWithdrawReward:
				var wrapper reward.WithdrawRewardArgsWrapper
				err := method.Inputs.Copy(&wrapper, args)
				s.Require().NoError(err, "method.Inputs.Copy() should succeed for WithdrawRewardArgsWrapper")
				s.Require().True(tc.verify(wrapper), "decoded wrapper should match original values")

			case reward.MethodWithdrawIMUATokenReward:
				var wrapper reward.WithdrawIMUATokenRewardArgsWrapper
				err := method.Inputs.Copy(&wrapper, args)
				s.Require().NoError(err, "method.Inputs.Copy() should succeed for WithdrawIMUATokenRewardArgsWrapper")
				s.Require().True(tc.verify(wrapper), "decoded wrapper should match original values")

			case reward.MethodUndelegateReward:
				var wrapper reward.UndelegateRewardArgsWrapper
				err := method.Inputs.Copy(&wrapper, args)
				s.Require().NoError(err, "method.Inputs.Copy() should succeed for UndelegateRewardArgsWrapper")
				s.Require().True(tc.verify(wrapper), "decoded wrapper should match original values")

			case reward.MethodRegisterRewardToken:
				var wrapper reward.RegisterRewardTokenArgsWrapper
				err := method.Inputs.Copy(&wrapper, args)
				s.Require().NoError(err, "method.Inputs.Copy() should succeed for RegisterRewardTokenArgsWrapper")
				s.Require().True(tc.verify(wrapper), "decoded wrapper should match original values")

			case reward.MethodSetAVSRewardDistribution:
				var wrapper reward.AVSRewardDistributionInfoArgsWrapper
				err := method.Inputs.Copy(&wrapper, args)
				s.Require().NoError(err, "method.Inputs.Copy() should succeed for AVSRewardDistributionInfoArgsWrapper")
				s.Require().True(tc.verify(wrapper), "decoded wrapper should match original values")
			}
		})
	}
}
