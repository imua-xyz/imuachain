package avs_test

import (
	"fmt"
	epochstypes "github.com/ExocoreNetwork/exocore/x/epochs/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/prysmaticlabs/prysm/v4/crypto/bls/blst"
	"math/big"
	"time"

	utiltx "github.com/ExocoreNetwork/exocore/testutil/tx"

	sdkmath "cosmossdk.io/math"

	avsManagerPrecompile "github.com/ExocoreNetwork/exocore/precompiles/avs"
	exocmn "github.com/ExocoreNetwork/exocore/precompiles/common"
	assetstype "github.com/ExocoreNetwork/exocore/x/assets/types"
	avstype "github.com/ExocoreNetwork/exocore/x/avs/types"
	operatorKeeper "github.com/ExocoreNetwork/exocore/x/operator/keeper"
	"github.com/ExocoreNetwork/exocore/x/operator/types"

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
	method := suite.precompile.Methods[avsManagerPrecompile.MethodGetOptinOperators]
	operatorAddress, avsAddr, slashContract := "exo18cggcpvwspnd5c6ny8wrqxpffj5zmhklprtnph", suite.Address, "0xDF907c29719154eb9872f021d21CAE6E5025d7aB"

	operatorOptIn := func() {
		optedInfo := &types.OptedInfo{
			SlashContract: slashContract,
			// #nosec G701
			OptedInHeight:  uint64(suite.Ctx.BlockHeight()),
			OptedOutHeight: types.DefaultOptedOutHeight,
		}
		err := suite.App.OperatorKeeper.SetOptedInfo(suite.Ctx, operatorAddress, avsAddr.String(), optedInfo)
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
			errContains: fmt.Sprintf(exocmn.ErrContractInputParaOrType, 0, "string", "0x0000000000000000000000000000000000000000"),
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
				err := suite.precompile.UnpackIntoInterface(&out, avsManagerPrecompile.MethodGetOptinOperators, bz)
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
				var out []string
				err := suite.precompile.UnpackIntoInterface(&out, avsManagerPrecompile.MethodGetOptinOperators, bz)
				suite.Require().NoError(err, "failed to unpack output", err)
				suite.Require().Equal(1, len(out))
				suite.Require().Equal(operatorAddress, out[0])
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

			bz, err := suite.precompile.GetOptedInOperatorAccAddrs(suite.Ctx, contract, &method, tc.malleate())

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
	expectedUSDvalue := sdkmath.LegacyNewDec(0)

	setUp := func() {
		suite.prepare()
		// register the new token
		usdcAddr := common.HexToAddress("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48")
		usdcClientChainAsset := assetstype.AssetInfo{
			Name:             "USD coin",
			Symbol:           "USDC",
			Address:          usdcAddr.String(),
			Decimals:         6,
			LayerZeroChainID: 101,
			MetaInfo:         "USDC",
		}
		err := suite.App.AssetsKeeper.SetStakingAssetInfo(
			suite.Ctx,
			&assetstype.StakingAssetInfo{
				AssetBasicInfo:     usdcClientChainAsset,
				StakingTotalAmount: sdkmath.NewInt(0),
			},
		)
		suite.NoError(err)
		// register the new AVS
		suite.prepareAvs([]string{"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48_0x65", "0xdac17f958d2ee523a2206206994597c13d831ec7_0x65"}, utiltx.GenerateAddress().String())
		// opt in
		err = suite.App.OperatorKeeper.OptIn(suite.Ctx, suite.operatorAddr, suite.avsAddr)
		suite.NoError(err)
		usdtPrice, err := suite.App.OperatorKeeper.OracleInterface().GetSpecifiedAssetsPrice(suite.Ctx, suite.assetID)
		suite.NoError(err)
		usdtValue := operatorKeeper.CalculateUSDValue(suite.delegationAmount, usdtPrice.Value, suite.assetDecimal, usdtPrice.Decimal)
		// deposit and delegate another asset to the operator
		suite.NoError(err)
		suite.prepareDeposit(usdcAddr, sdkmath.NewInt(1e8))
		usdcPrice, err := suite.App.OperatorKeeper.OracleInterface().GetSpecifiedAssetsPrice(suite.Ctx, suite.assetID)
		suite.NoError(err)
		delegatedAmount := sdkmath.NewIntWithDecimal(8, 7)
		suite.prepareDelegation(true, usdcAddr, delegatedAmount)

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
					common.HexToAddress(suite.avsAddr),
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
	expectedUSDvalue := sdkmath.LegacyNewDec(0)

	setUp := func() {
		suite.prepare()
		// register the new token
		usdcAddr := common.HexToAddress("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48")
		usdcClientChainAsset := assetstype.AssetInfo{
			Name:             "USD coin",
			Symbol:           "USDC",
			Address:          usdcAddr.String(),
			Decimals:         6,
			LayerZeroChainID: 101,
			MetaInfo:         "USDC",
		}
		err := suite.App.AssetsKeeper.SetStakingAssetInfo(
			suite.Ctx,
			&assetstype.StakingAssetInfo{
				AssetBasicInfo:     usdcClientChainAsset,
				StakingTotalAmount: sdkmath.NewInt(0),
			},
		)
		suite.NoError(err)
		// register the new AVS
		suite.prepareAvs([]string{"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48_0x65", "0xdac17f958d2ee523a2206206994597c13d831ec7_0x65"}, utiltx.GenerateAddress().String())
		// opt in
		err = suite.App.OperatorKeeper.OptIn(suite.Ctx, suite.operatorAddr, suite.avsAddr)
		suite.NoError(err)
		usdtPrice, err := suite.App.OperatorKeeper.OracleInterface().GetSpecifiedAssetsPrice(suite.Ctx, suite.assetID)
		suite.NoError(err)
		usdtValue := operatorKeeper.CalculateUSDValue(suite.delegationAmount, usdtPrice.Value, suite.assetDecimal, usdtPrice.Decimal)
		// deposit and delegate another asset to the operator
		suite.NoError(err)
		suite.prepareDeposit(usdcAddr, sdkmath.NewInt(1e8))
		usdcPrice, err := suite.App.OperatorKeeper.OracleInterface().GetSpecifiedAssetsPrice(suite.Ctx, suite.assetID)
		suite.NoError(err)
		delegatedAmount := sdkmath.NewIntWithDecimal(8, 7)
		suite.prepareDelegation(true, usdcAddr, delegatedAmount)

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
					common.HexToAddress(suite.avsAddr),
					suite.operatorAddr.String(),
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
	method := suite.precompile.Methods[avsManagerPrecompile.MethodGetRegisteredPubkey]
	privateKey, err := blst.RandKey()
	suite.NoError(err)
	operatorAddr := "exo13h6xg79g82e2g2vhjwg7j4r2z2hlncelwutkjr"

	publicKey := privateKey.PublicKey()
	setUp := func() {
		suite.prepareOperator()

		blsPub := &avstype.BlsPubKeyInfo{
			Operator: operatorAddr,
			PubKey:   publicKey.Marshal(),
			Name:     "",
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
					operatorAddr,
				}
			},
			func(bz []byte) {
				var out []byte
				err := suite.precompile.UnpackIntoInterface(&out, avsManagerPrecompile.MethodGetRegisteredPubkey, bz)
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

			bz, err := suite.precompile.GetRegisteredPubkey(suite.Ctx, contract, &method, tc.malleate())

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
		avsOwnerAddress := []string{"exo13h6xg79g82e2g2vhjwg7j4r2z2hlncelwutkjr", "exo13h6xg79g82e2g2vhjwg7j4r2z2hlncelwutkj1", "exo13h6xg79g82e2g2vhjwg7j4r2z2hlncelwutkj2"}
		assetID := suite.AssetIDs
		avs := &avstype.AVSInfo{
			Name:                avsName,
			AvsAddress:          avsAddress,
			SlashAddr:           utiltx.GenerateAddress().String(),
			AvsOwnerAddress:     avsOwnerAddress,
			AssetIDs:            assetID,
			AvsUnbondingPeriod:  uint64(testAVSUnbondingPeriod),
			MinSelfDelegation:   uint64(testMinSelfDelegation),
			EpochIdentifier:     epochstypes.DayEpochID,
			StartingEpoch:       uint64(testStartingEpoch),
			MinOptInOperators:   uint64(testMinOptInOperators),
			MinTotalStakeAmount: uint64(testMinTotalStakeAmount),
			AvsSlash:            sdk.MustNewDecFromStr("0.001"),
			AvsReward:           sdk.MustNewDecFromStr("0.002"),
			TaskAddr:            "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
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
	opAccAddr, _ := sdk.AccAddressFromBech32("exo13h6xg79g82e2g2vhjwg7j4r2z2hlncelwutkjr")

	testCases := []avsTestCases{
		{
			"success - existent operator",
			func() []interface{} {
				suite.prepareOperator()
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
			TaskTotalPower:        sdk.Dec(sdkmath.NewInt(0)),
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
				var out []uint64

				err := suite.precompile.UnpackIntoInterface(&out, avsManagerPrecompile.MethodGetTaskInfo, bz)
				suite.Require().NoError(err, "failed to unpack output", err)
				suite.Require().Equal([]uint64{5, 10, 60}, out)
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
