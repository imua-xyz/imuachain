package avs_test

import (
	"fmt"
	"math/big"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	epochstypes "github.com/imua-xyz/imuachain/x/epochs/types"
	"github.com/prysmaticlabs/prysm/v4/crypto/bls/blst"

	utiltx "github.com/imua-xyz/imuachain/testutil/tx"

	sdkmath "cosmossdk.io/math"

	avsManagerPrecompile "github.com/imua-xyz/imuachain/precompiles/avs"
	imuacmn "github.com/imua-xyz/imuachain/precompiles/common"
	assetstype "github.com/imua-xyz/imuachain/x/assets/types"
	avstype "github.com/imua-xyz/imuachain/x/avs/types"
	operatorKeeper "github.com/imua-xyz/imuachain/x/operator/keeper"
	"github.com/imua-xyz/imuachain/x/operator/types"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum/go-ethereum/core/vm"
)

type avsTestCases struct {
	name        string
	malleate    func() []interface{}
	postCheck   func(bz []byte)
	gas         uint64
	expErr      bool
	errContains string
}

var baseTestCases = []avsTestCases{
	{
		name: "fail - empty input args",
		malleate: func() []interface{} {
			return []interface{}{}
		},
		postCheck:   func(bz []byte) {},
		gas:         100000,
		expErr:      true,
		errContains: "invalid number of arguments",
	},
	{
		name: "fail - invalid  address",
		malleate: func() []interface{} {
			return []interface{}{
				"invalid",
			}
		},
		postCheck:   func(bz []byte) {},
		gas:         100000,
		expErr:      true,
		errContains: "invalid bech32 string",
	},
}

func (suite *AVSManagerPrecompileSuite) TestGetOptedInOperatorAccAddrs() {
	method := suite.precompile.Methods[avsManagerPrecompile.MethodGetOptInOperators]
	operatorAddress, avsAddress, slashContract :=
		sdk.AccAddress(utiltx.GenerateAddress().Bytes()).String(), suite.Address, "0xDF907c29719154eb9872f021d21CAE6E5025d7aB"

	operatorOptIn := func() {
		optedInfo := &types.OptedInfo{
			SlashContract: slashContract,
			// #nosec G701
			OptedInHeight:  uint64(suite.Ctx.BlockHeight()),
			OptedOutHeight: types.DefaultOptedOutHeight,
		}
		err := suite.App.OperatorKeeper.SetOptedInfo(suite.Ctx, operatorAddress, avsAddress.String(), optedInfo)
		suite.NoError(err)
	}
	testCases := []avsTestCases{
		{
			name: "fail - invalid avs address",
			malleate: func() []interface{} {
				return []interface{}{
					"invalid",
				}
			},
			postCheck:   func(bz []byte) {},
			gas:         100000,
			expErr:      true,
			errContains: fmt.Sprintf(imuacmn.ErrContractInputParamOrType, 0, "common.Address", "0x0000000000000000000000000000000000000000"),
		},
		{
			"success - no operators",
			func() []interface{} {
				return []interface{}{
					suite.Address,
				}
			},
			func(bz []byte) {
				var out []string
				err := suite.precompile.UnpackIntoInterface(&out, avsManagerPrecompile.MethodGetOptInOperators, bz)
				suite.Require().NoError(err, "failed to unpack output", err)
				suite.Require().Equal(0, len(out))
			},
			100000,
			false,
			"",
		},
		{
			"success - existent operators",
			func() []interface{} {
				operatorOptIn()
				return []interface{}{
					suite.Address,
				}
			},
			func(bz []byte) {
				var out []common.Address
				err := suite.precompile.UnpackIntoInterface(&out, avsManagerPrecompile.MethodGetOptInOperators, bz)
				suite.Require().NoError(err, "failed to unpack output", err)
				suite.Require().Equal(1, len(out))
				acc, err := sdk.AccAddressFromBech32(operatorAddress)
				suite.Require().Equal(common.BytesToAddress(acc), out[0])
			},
			100000,
			false,
			"",
		},
	}
	testCases = append(testCases, baseTestCases[0])

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			contract := vm.NewContract(vm.AccountRef(suite.Address), suite.precompile, big.NewInt(0), tc.gas)

			bz, err := suite.precompile.GetOptedInOperatorAccAddresses(suite.Ctx, contract, &method, tc.malleate())

			if tc.expErr {
				suite.Require().Error(err)
				suite.Require().Contains(err.Error(), tc.errContains)
			} else {
				suite.Require().NoError(err)
				suite.Require().NotEmpty(bz)
				tc.postCheck(bz)
			}
		})
	}
}

func (suite *AVSManagerPrecompileSuite) TestAVSUSDValue() {
	method := s.precompile.Methods[avsManagerPrecompile.MethodGetAVSUSDValue]
	expectedUSDvalue := sdkmath.LegacyZeroDec()

	setUp := func() {
		suite.prepare()
		// register the new token
		usdcAddress := common.HexToAddress("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48")
		usdcClientChainAsset := assetstype.AssetInfo{
			Name:             "USD coin",
			Symbol:           "USDC",
			Address:          usdcAddress.String(),
			Decimals:         6,
			LayerZeroChainID: 101,
			MetaInfo:         "USDC",
		}
		err := suite.App.AssetsKeeper.SetStakingAssetInfo(
			suite.Ctx,
			&assetstype.StakingAssetInfo{
				AssetBasicInfo:     usdcClientChainAsset,
				StakingTotalAmount: sdkmath.ZeroInt(),
			},
		)
		suite.NoError(err)
		// register the new AVS
		suite.prepareAvs([]string{"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48_0x65", "0xdac17f958d2ee523a2206206994597c13d831ec7_0x65"}, utiltx.GenerateAddress().String())
		// opt in
		err = suite.App.OperatorKeeper.OptIn(suite.Ctx, suite.operatorAddress, suite.avsAddress)
		suite.NoError(err)
		usdtPrice, err := suite.App.OperatorKeeper.OracleInterface().GetSpecifiedAssetsPrice(suite.Ctx, suite.assetID)
		suite.NoError(err)
		usdtValue := operatorKeeper.CalculateUSDValue(suite.delegationAmount, usdtPrice.Value, suite.assetDecimal, usdtPrice.Decimal)
		// deposit and delegate another asset to the operator
		suite.NoError(err)
		suite.prepareDeposit(usdcAddress, sdkmath.NewInt(1e8))
		usdcPrice, err := suite.App.OperatorKeeper.OracleInterface().GetSpecifiedAssetsPrice(suite.Ctx, suite.assetID)
		suite.NoError(err)
		delegatedAmount := sdkmath.NewIntWithDecimal(8, 7)
		suite.prepareDelegation(true, usdcAddress, delegatedAmount)

		// updating the new voting power
		usdcValue := operatorKeeper.CalculateUSDValue(suite.delegationAmount, usdcPrice.Value, suite.assetDecimal, usdcPrice.Decimal)
		expectedUSDvalue = usdcValue.Add(usdtValue)
		suite.CommitAfter(time.Hour*1 + time.Nanosecond)
		suite.CommitAfter(time.Hour*1 + time.Nanosecond)
		suite.CommitAfter(time.Hour*1 + time.Nanosecond)
	}

	testCases := []avsTestCases{
		{
			"success - existent operators",
			func() []interface{} {
				setUp()
				return []interface{}{
					common.HexToAddress(suite.avsAddress),
				}
			},
			func(bz []byte) {
				var out *big.Int
				err := s.precompile.UnpackIntoInterface(&out, avsManagerPrecompile.MethodGetAVSUSDValue, bz)
				s.Require().NoError(err, "failed to unpack output", err)
				s.Require().Equal(expectedUSDvalue.BigInt(), out)
			},
			100000,
			false,
			"",
		},
	}
	testCases = append(testCases, baseTestCases[0])

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			contract := vm.NewContract(vm.AccountRef(s.Address), s.precompile, big.NewInt(0), tc.gas)

			bz, err := s.precompile.GetAVSUSDValue(s.Ctx, contract, &method, tc.malleate())

			if tc.expErr {
				s.Require().Error(err)
				s.Require().Contains(err.Error(), tc.errContains)
			} else {
				s.Require().NoError(err)
				s.Require().NotEmpty(bz)
				tc.postCheck(bz)
			}
		})
	}
}

func (suite *AVSManagerPrecompileSuite) TestGetOperatorOptedUSDValue() {
	method := s.precompile.Methods[avsManagerPrecompile.MethodGetOperatorOptedUSDValue]
	expectedUSDvalue := sdkmath.LegacyZeroDec()

	setUp := func() {
		suite.prepare()
		// register the new token
		usdcAddress := common.HexToAddress("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48")
		usdcClientChainAsset := assetstype.AssetInfo{
			Name:             "USD coin",
			Symbol:           "USDC",
			Address:          usdcAddress.String(),
			Decimals:         6,
			LayerZeroChainID: 101,
			MetaInfo:         "USDC",
		}
		err := suite.App.AssetsKeeper.SetStakingAssetInfo(
			suite.Ctx,
			&assetstype.StakingAssetInfo{
				AssetBasicInfo:     usdcClientChainAsset,
				StakingTotalAmount: sdkmath.ZeroInt(),
			},
		)
		suite.NoError(err)
		// register the new AVS
		suite.prepareAvs([]string{"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48_0x65", "0xdac17f958d2ee523a2206206994597c13d831ec7_0x65"}, utiltx.GenerateAddress().String())
		// opt in
		err = suite.App.OperatorKeeper.OptIn(suite.Ctx, suite.operatorAddress, suite.avsAddress)
		suite.NoError(err)
		usdtPrice, err := suite.App.OperatorKeeper.OracleInterface().GetSpecifiedAssetsPrice(suite.Ctx, suite.assetID)
		suite.NoError(err)
		usdtValue := operatorKeeper.CalculateUSDValue(suite.delegationAmount, usdtPrice.Value, suite.assetDecimal, usdtPrice.Decimal)
		// deposit and delegate another asset to the operator
		suite.NoError(err)
		suite.prepareDeposit(usdcAddress, sdkmath.NewInt(1e8))
		usdcPrice, err := suite.App.OperatorKeeper.OracleInterface().GetSpecifiedAssetsPrice(suite.Ctx, suite.assetID)
		suite.NoError(err)
		delegatedAmount := sdkmath.NewIntWithDecimal(8, 7)
		suite.prepareDelegation(true, usdcAddress, delegatedAmount)

		// updating the new voting power
		usdcValue := operatorKeeper.CalculateUSDValue(suite.delegationAmount, usdcPrice.Value, suite.assetDecimal, usdcPrice.Decimal)
		expectedUSDvalue = usdcValue.Add(usdtValue)
		suite.CommitAfter(time.Hour*1 + time.Nanosecond)
		suite.CommitAfter(time.Hour*1 + time.Nanosecond)
		suite.CommitAfter(time.Hour*1 + time.Nanosecond)
	}

	testCases := []avsTestCases{
		{
			"success - existent operators",
			func() []interface{} {
				setUp()
				return []interface{}{
					common.HexToAddress(suite.avsAddress),
					common.BytesToAddress(suite.operatorAddress),
				}
			},
			func(bz []byte) {
				var out *big.Int
				err := s.precompile.UnpackIntoInterface(&out, avsManagerPrecompile.MethodGetOperatorOptedUSDValue, bz)
				s.Require().NoError(err, "failed to unpack output", err)
				s.Require().Equal(expectedUSDvalue.BigInt(), out)
			},
			100000,
			false,
			"",
		},
	}
	testCases = append(testCases, baseTestCases[0])

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			contract := vm.NewContract(vm.AccountRef(s.Address), s.precompile, big.NewInt(0), tc.gas)

			bz, err := s.precompile.GetOperatorOptedUSDValue(s.Ctx, contract, &method, tc.malleate())

			if tc.expErr {
				s.Require().Error(err)
				s.Require().Contains(err.Error(), tc.errContains)
			} else {
				s.Require().NoError(err)
				s.Require().NotEmpty(bz)
				tc.postCheck(bz)
			}
		})
	}
}

func (suite *AVSManagerPrecompileSuite) TestGetRegisteredPubkey() {
	method := suite.precompile.Methods[avsManagerPrecompile.MethodGetRegisteredPubKey]
	privateKey, err := blst.RandKey()
	suite.NoError(err)
	address := utiltx.GenerateAddress()
	var operatorAddress sdk.AccAddress = address[:]

	publicKey := privateKey.PublicKey()
	setUp := func() {
		suite.prepareOperator(operatorAddress.String())

		blsPub := &avstype.BlsPubKeyInfo{
			OperatorAddress: operatorAddress.String(),
			PubKey:          publicKey.Marshal(),
			AvsAddress:      "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
		}
		err = suite.App.AVSManagerKeeper.SetOperatorPubKey(suite.Ctx, blsPub)
		suite.NoError(err)
	}
	testCases := []avsTestCases{
		{
			"success - existent pubKey",
			func() []interface{} {
				setUp()
				return []interface{}{
					address,
					common.HexToAddress("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"),
				}
			},
			func(bz []byte) {
				var out []byte
				err := suite.precompile.UnpackIntoInterface(&out, avsManagerPrecompile.MethodGetRegisteredPubKey, bz)
				suite.Require().NoError(err, "failed to unpack output", err)
				suite.Require().Equal(48, len(out))
				suite.Require().Equal(publicKey.Marshal(), out)
			},
			100000,
			false,
			"",
		},
	}
	testCases = append(testCases, baseTestCases[0])

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			contract := vm.NewContract(vm.AccountRef(suite.Address), suite.precompile, big.NewInt(0), tc.gas)

			bz, err := suite.precompile.GetRegisteredPubKey(suite.Ctx, contract, &method, tc.malleate())

			if tc.expErr {
				suite.Require().Error(err)
				suite.Require().Contains(err.Error(), tc.errContains)
			} else {
				suite.Require().NoError(err)
				suite.Require().NotEmpty(bz)
				tc.postCheck(bz)
			}
		})
	}
}

func (suite *AVSManagerPrecompileSuite) TestGetAVSInfo() {
	method := suite.precompile.Methods[avsManagerPrecompile.MethodGetAVSEpochIdentifier]
	avsAddress := "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"
	testAVSUnbondingPeriod := 7
	testMinSelfDelegation := 10
	testMinOptInOperators := 100
	testMinTotalStakeAmount := 1000
	testStartingEpoch := 1
	setUp := func() {
		avsName := "avsTest"
		avsOwnerAddresses := []string{
			sdk.AccAddress(utiltx.GenerateAddress().Bytes()).String(),
			sdk.AccAddress(utiltx.GenerateAddress().Bytes()).String(),
			sdk.AccAddress(utiltx.GenerateAddress().Bytes()).String(),
		}
		assetID := suite.AssetIDs
		avs := &avstype.AVSInfo{
			Name:                avsName,
			AvsAddress:          avsAddress,
			SlashAddress:        utiltx.GenerateAddress().String(),
			AvsOwnerAddresses:   avsOwnerAddresses,
			AssetIDs:            assetID,
			AvsUnbondingPeriod:  uint64(testAVSUnbondingPeriod),
			MinSelfDelegation:   uint64(testMinSelfDelegation),
			EpochIdentifier:     epochstypes.DayEpochID,
			StartingEpoch:       uint64(testStartingEpoch),
			MinOptInOperators:   uint64(testMinOptInOperators),
			MinTotalStakeAmount: uint64(testMinTotalStakeAmount),
			AvsSlash:            sdk.MustNewDecFromStr("0.001"),
			AvsReward:           sdk.MustNewDecFromStr("0.002"),
			TaskAddress:         "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
		}

		err := suite.App.AVSManagerKeeper.SetAVSInfo(suite.Ctx, avs)
		suite.NoError(err)
	}
	testCases := []avsTestCases{
		{
			"success - existent avs",
			func() []interface{} {
				setUp()
				return []interface{}{
					common.HexToAddress(avsAddress),
				}
			},
			func(bz []byte) {
				var out string

				err := suite.precompile.UnpackIntoInterface(&out, avsManagerPrecompile.MethodGetAVSEpochIdentifier, bz)
				suite.Require().NoError(err, "failed to unpack output", err)
				suite.Require().Equal(epochstypes.DayEpochID, out)
			},
			100000,
			false,
			"",
		},
	}
	testCases = append(testCases, baseTestCases[0])

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			contract := vm.NewContract(vm.AccountRef(suite.Address), suite.precompile, big.NewInt(0), tc.gas)

			bz, err := suite.precompile.GetAVSEpochIdentifier(suite.Ctx, contract, &method, tc.malleate())

			if tc.expErr {
				suite.Require().Error(err)
				suite.Require().Contains(err.Error(), tc.errContains)
			} else {
				suite.Require().NoError(err)
				suite.Require().NotEmpty(bz)
				tc.postCheck(bz)
			}
		})
	}
}

func (suite *AVSManagerPrecompileSuite) TestIsoperator() {
	method := suite.precompile.Methods[avsManagerPrecompile.MethodIsOperator]
	opAccAddr := sdk.AccAddress(utiltx.GenerateAddress().Bytes())

	testCases := []avsTestCases{
		{
			"success - existent operator",
			func() []interface{} {
				suite.prepareOperator(opAccAddr.String())
				return []interface{}{
					common.BytesToAddress(opAccAddr),
				}
			},
			func(bz []byte) {
				var out bool
				err := suite.precompile.UnpackIntoInterface(&out, avsManagerPrecompile.MethodIsOperator, bz)
				suite.Require().NoError(err, "failed to unpack output", err)
				suite.Require().Equal(true, out)
			},
			100000,
			false,
			"",
		},
	}
	testCases = append(testCases, baseTestCases[0])

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			contract := vm.NewContract(vm.AccountRef(suite.Address), suite.precompile, big.NewInt(0), tc.gas)

			bz, err := suite.precompile.IsOperator(suite.Ctx, contract, &method, tc.malleate())

			if tc.expErr {
				suite.Require().Error(err)
				suite.Require().Contains(err.Error(), tc.errContains)
			} else {
				suite.Require().NoError(err)
				suite.Require().NotEmpty(bz)
				tc.postCheck(bz)
			}
		})
	}
}

func (suite *AVSManagerPrecompileSuite) TestGetTaskInfo() {
	method := suite.precompile.Methods[avsManagerPrecompile.MethodGetTaskInfo]
	taskAddress := "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"

	setUp := func() {
		info := &avstype.TaskInfo{
			TaskContractAddress:   taskAddress,
			Name:                  "test-avstask-01",
			TaskId:                uint64(3),
			Hash:                  []byte("active"),
			TaskResponsePeriod:    10,
			StartingEpoch:         5,
			TaskStatisticalPeriod: 60,
			TaskTotalPower:        sdk.Dec(sdkmath.ZeroInt()),
		}
		err := suite.App.AVSManagerKeeper.SetTaskInfo(suite.Ctx, info)
		suite.NoError(err)
	}
	testCases := []avsTestCases{
		{
			"success - existent task",
			func() []interface{} {
				setUp()
				return []interface{}{
					common.HexToAddress(taskAddress),
					uint64(3),
				}
			},
			func(bz []byte) {
				result, err := s.precompile.Unpack(avsManagerPrecompile.MethodGetTaskInfo, bz)
				s.Require().NoError(err)
				taskInfo := result[0].(struct {
					TaskContractAddress   common.Address   `json:"taskContractAddress"`
					Name                  string           `json:"name"`
					Hash                  []byte           `json:"hash"`
					TaskID                uint64           `json:"taskID"`
					TaskResponsePeriod    uint64           `json:"taskResponsePeriod"`
					TaskStatisticalPeriod uint64           `json:"taskStatisticalPeriod"`
					TaskChallengePeriod   uint64           `json:"taskChallengePeriod"`
					ThresholdPercentage   uint8            `json:"thresholdPercentage"`
					StartingEpoch         uint64           `json:"startingEpoch"`
					ActualThreshold       string           `json:"actualThreshold"`
					OptInOperators        []common.Address `json:"optInOperators"`
					SignedOperators       []common.Address `json:"signedOperators"`
					NoSignedOperators     []common.Address `json:"noSignedOperators"`
					ErrSignedOperators    []common.Address `json:"errSignedOperators"`
					TaskTotalPower        string           `json:"taskTotalPower"`
					OperatorActivePower   []struct {
						Operator common.Address `json:"operator"`
						Power    *big.Int       `json:"power"`
					} `json:"operatorActivePower"`
					IsExpected              bool             `json:"isExpected"`
					EligibleRewardOperators []common.Address `json:"eligibleRewardOperators"`
					EligibleSlashOperators  []common.Address `json:"eligibleSlashOperators"`
				})
				suite.Require().Equal(taskInfo.TaskContractAddress, common.HexToAddress(taskAddress))
				suite.Require().NoError(err, "failed to unpack output", err)
			},
			100000,
			false,
			"",
		},
	}
	testCases = append(testCases, baseTestCases[0])

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			contract := vm.NewContract(vm.AccountRef(suite.Address), suite.precompile, big.NewInt(0), tc.gas)

			bz, err := suite.precompile.GetTaskInfo(suite.Ctx, contract, &method, tc.malleate())

			if tc.expErr {
				suite.Require().Error(err)
				suite.Require().Contains(err.Error(), tc.errContains)
			} else {
				suite.Require().NoError(err)
				suite.Require().NotEmpty(bz)
				tc.postCheck(bz)
			}
		})
	}
}

func (suite *AVSManagerPrecompileSuite) TestGetCurrentEpoch() {
	method := suite.precompile.Methods[avsManagerPrecompile.MethodGetCurrentEpoch]
	testCases := []avsTestCases{
		{
			"success - existent avs",
			func() []interface{} {
				return []interface{}{
					epochstypes.DayEpochID,
				}
			},
			func(bz []byte) {
				var out int64

				err := suite.precompile.UnpackIntoInterface(&out, avsManagerPrecompile.MethodGetCurrentEpoch, bz)
				suite.Require().NoError(err, "failed to unpack output", err)
				suite.Require().Equal(int64(1), out)
			},
			100000,
			false,
			"",
		},
	}
	testCases = append(testCases, baseTestCases[0])

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			contract := vm.NewContract(vm.AccountRef(suite.Address), suite.precompile, big.NewInt(0), tc.gas)

			bz, err := suite.precompile.GetCurrentEpoch(suite.Ctx, contract, &method, tc.malleate())

			if tc.expErr {
				suite.Require().Error(err)
				suite.Require().Contains(err.Error(), tc.errContains)
			} else {
				suite.Require().NoError(err)
				suite.Require().NotEmpty(bz)
				tc.postCheck(bz)
			}
		})
	}
}

func (suite *AVSManagerPrecompileSuite) TestGetChallengeInfo() {
	method := suite.precompile.Methods[avsManagerPrecompile.MethodGetChallengeInfo]
	taskAddress := "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"
	challengeAddr := sdk.AccAddress(utiltx.GenerateAddress().Bytes()).String()
	taskID := uint64(3)
	setUp := func() {
		suite.App.AVSManagerKeeper.SetTaskChallengedInfo(suite.Ctx, taskID, challengeAddr, common.HexToAddress(taskAddress))
	}
	testCases := []avsTestCases{
		{
			"success - existent ChallengeInfo",
			func() []interface{} {
				setUp()
				return []interface{}{
					common.HexToAddress(taskAddress),
					taskID,
				}
			},
			func(bz []byte) {
				var out common.Address
				err := suite.precompile.UnpackIntoInterface(&out, avsManagerPrecompile.MethodGetChallengeInfo, bz)
				suite.Require().NoError(err, "failed to unpack output", err)
				address, _ := sdk.AccAddressFromBech32(challengeAddr)
				suite.Require().Equal(common.BytesToAddress(address), out)
			},
			100000,
			false,
			"",
		},
	}
	testCases = append(testCases, baseTestCases[0])

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			contract := vm.NewContract(vm.AccountRef(suite.Address), suite.precompile, big.NewInt(0), tc.gas)

			bz, err := suite.precompile.GetChallengeInfo(suite.Ctx, contract, &method, tc.malleate())

			if tc.expErr {
				suite.Require().Error(err)
				suite.Require().Contains(err.Error(), tc.errContains)
			} else {
				suite.Require().NoError(err)
				suite.Require().NotEmpty(bz)
				tc.postCheck(bz)
			}
		})
	}
}

func (suite *AVSManagerPrecompileSuite) TestGetOperatorTaskResponseList() {
	method := suite.precompile.Methods[avsManagerPrecompile.MethodGetOperatorTaskResponseList]
	taskAddress := "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"
	OperatorAddress1 := sdk.AccAddress(utiltx.GenerateAddress().Bytes()).String()
	OperatorAddress2 := sdk.AccAddress(utiltx.GenerateAddress().Bytes()).String()
	setUp := func() {
		info1 := &avstype.TaskResultInfo{
			TaskContractAddress: taskAddress,
			OperatorAddress:     OperatorAddress1,
			TaskId:              uint64(3),
			Phase:               2,
			TaskResponseHash:    "hash1",
			BlsSignature:        []byte("BlsSignature1"),
			TaskResponse:        []byte("TaskResponse1"),
		}
		info2 := &avstype.TaskResultInfo{
			TaskContractAddress: taskAddress,
			OperatorAddress:     OperatorAddress2,
			TaskId:              uint64(3),
			Phase:               2,
			TaskResponseHash:    "hash2",
			BlsSignature:        []byte("BlsSignature2"),
			TaskResponse:        []byte("TaskResponse2"),
		}
		info := &avstype.TaskInfo{
			TaskContractAddress:   taskAddress,
			Name:                  "test-avstask-01",
			TaskId:                uint64(3),
			Hash:                  []byte("task"),
			TaskResponsePeriod:    10,
			StartingEpoch:         5,
			TaskStatisticalPeriod: 60,
			TaskTotalPower:        sdk.Dec(sdkmath.ZeroInt()),
			SignedOperators:       []string{OperatorAddress1, OperatorAddress2},
			OperatorActivePower: &avstype.OperatorActivePowerList{
				OperatorPowerList: []*avstype.OperatorActivePowerInfo{
					{
						OperatorAddress: OperatorAddress1,
						SelfActivePower: sdk.MustNewDecFromStr("544"),
					},
					{
						OperatorAddress: OperatorAddress2,
						SelfActivePower: sdk.MustNewDecFromStr("514"),
					},
				},
			},
		}
		err := suite.App.AVSManagerKeeper.SetTaskInfo(suite.Ctx, info)
		suite.App.AVSManagerKeeper.SetTaskResultInfo(suite.Ctx, info1)
		suite.App.AVSManagerKeeper.SetTaskResultInfo(suite.Ctx, info2)
		suite.NoError(err)
	}
	testCases := []avsTestCases{
		{
			"success - existent task response",
			func() []interface{} {
				setUp()
				return []interface{}{
					common.HexToAddress(taskAddress),
					uint64(3),
				}
			},
			func(bz []byte) {
				result, err := s.precompile.Unpack(avsManagerPrecompile.MethodGetOperatorTaskResponseList, bz)
				s.Require().NoError(err)
				resInfo := result[0].([]struct {
					TaskContractAddress common.Address `json:"taskContractAddress"`
					TaskID              uint64         `json:"taskID"`
					OperatorAddress     common.Address `json:"operatorAddress"`
					TaskResponseHash    string         `json:"taskResponseHash"`
					TaskResponse        []byte         `json:"taskResponse"`
					BlsSignature        []byte         `json:"blsSignature"`
					Power               *big.Int       `json:"power"`
					Phase               uint8          `json:"phase"`
				})
				suite.Require().Equal(len(resInfo), 2)
				suite.Require().NoError(err, "failed to unpack output", err)
			},
			100000,
			false,
			"",
		},
	}
	testCases = append(testCases, baseTestCases[0])

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			contract := vm.NewContract(vm.AccountRef(suite.Address), suite.precompile, big.NewInt(0), tc.gas)

			bz, err := suite.precompile.GetOperatorTaskResponseList(suite.Ctx, contract, &method, tc.malleate())

			if tc.expErr {
				suite.Require().Error(err)
				suite.Require().Contains(err.Error(), tc.errContains)
			} else {
				suite.Require().NoError(err)
				suite.Require().NotEmpty(bz)
				tc.postCheck(bz)
			}
		})
	}
}

func (suite *AVSManagerPrecompileSuite) TestGetOperatorTaskResponse() {
	method := suite.precompile.Methods[avsManagerPrecompile.MethodGetOperatorTaskResponse]
	taskAddress := "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"
	opAccAddr := sdk.AccAddress(utiltx.GenerateAddress().Bytes())
	setUp := func() {
		info := &avstype.TaskResultInfo{
			TaskContractAddress: taskAddress,
			OperatorAddress:     opAccAddr.String(),
			TaskId:              uint64(3),
			Phase:               2,
			TaskResponseHash:    "hash",
			BlsSignature:        []byte("BlsSignature"),
			TaskResponse:        []byte("TaskResponse"),
		}

		suite.App.AVSManagerKeeper.SetTaskResultInfo(suite.Ctx, info)
	}
	testCases := []avsTestCases{
		{
			"success - existent task response",
			func() []interface{} {
				setUp()
				return []interface{}{
					common.HexToAddress(taskAddress),
					common.Address(opAccAddr.Bytes()),
					uint64(3),
				}
			},
			func(bz []byte) {
				result, err := s.precompile.Unpack(avsManagerPrecompile.MethodGetOperatorTaskResponse, bz)
				s.Require().NoError(err)
				resInfo := result[0].(struct {
					OperatorAddress     common.Address `json:"operatorAddress"`
					TaskResponseHash    string         `json:"taskResponseHash"`
					TaskResponse        []byte         `json:"taskResponse"`
					BlsSignature        []byte         `json:"blsSignature"`
					TaskContractAddress common.Address `json:"taskContractAddress"`
					TaskID              uint64         `json:"taskID"`
					Phase               uint8          `json:"phase"`
				})
				suite.Require().Equal(resInfo.TaskContractAddress, common.HexToAddress(taskAddress))
				suite.Require().NoError(err, "failed to unpack output", err)
			},
			100000,
			false,
			"",
		},
	}
	testCases = append(testCases, baseTestCases[0])

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			contract := vm.NewContract(vm.AccountRef(suite.Address), suite.precompile, big.NewInt(0), tc.gas)

			bz, err := suite.precompile.GetOperatorTaskResponse(suite.Ctx, contract, &method, tc.malleate())

			if tc.expErr {
				suite.Require().Error(err)
				suite.Require().Contains(err.Error(), tc.errContains)
			} else {
				suite.Require().NoError(err)
				suite.Require().NotEmpty(bz)
				tc.postCheck(bz)
			}
		})
	}
}
