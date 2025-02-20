package avs_test

import (
	"math/big"
	"strconv"
	"time"

	"cosmossdk.io/math"
	assetstypes "github.com/imua-xyz/imuachain/x/assets/types"
	"github.com/imua-xyz/imuachain/x/avs/types"

	sdkmath "cosmossdk.io/math"
	operatorKeeper "github.com/imua-xyz/imuachain/x/operator/keeper"

	"github.com/cometbft/cometbft/libs/rand"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	utiltx "github.com/evmos/evmos/v16/testutil/tx"
	"github.com/evmos/evmos/v16/x/evm/statedb"
	evmtypes "github.com/evmos/evmos/v16/x/evm/types"
	"github.com/imua-xyz/imuachain/app"
	"github.com/imua-xyz/imuachain/precompiles/avs"
	epochstypes "github.com/imua-xyz/imuachain/x/epochs/types"
	operatortypes "github.com/imua-xyz/imuachain/x/operator/types"
)

func (suite *AVSManagerPrecompileSuite) TestIsTransaction() {
	testCases := []struct {
		name   string
		method string
		isTx   bool
	}{
		{
			avs.MethodRegisterAVS,
			suite.precompile.Methods[avs.MethodRegisterAVS].Name,
			true,
		},
		{
			avs.MethodDeregisterAVS,
			suite.precompile.Methods[avs.MethodDeregisterAVS].Name,
			true,
		},
		{
			avs.MethodUpdateAVS,
			suite.precompile.Methods[avs.MethodUpdateAVS].Name,
			true,
		},
		{
			avs.MethodRegisterOperatorToAVS,
			suite.precompile.Methods[avs.MethodRegisterOperatorToAVS].Name,
			true,
		},
		{
			avs.MethodDeregisterOperatorFromAVS,
			suite.precompile.Methods[avs.MethodDeregisterOperatorFromAVS].Name,
			true,
		},
		{
			avs.MethodCreateAVSTask,
			suite.precompile.Methods[avs.MethodCreateAVSTask].Name,
			true,
		},
		{
			avs.MethodRegisterBLSPublicKey,
			suite.precompile.Methods[avs.MethodRegisterBLSPublicKey].Name,
			true,
		},
		{
			avs.MethodChallenge,
			suite.precompile.Methods[avs.MethodChallenge].Name,
			true,
		},
		{
			avs.MethodOperatorSubmitTask,
			suite.precompile.Methods[avs.MethodOperatorSubmitTask].Name,
			true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.Require().Equal(suite.precompile.IsTransaction(tc.method), tc.isTx)
		})
	}
}

func (s *AVSManagerPrecompileSuite) TestRegisterAVS() {
	// Default variables used during tests.
	gas := uint64(2_000)
	senderAddress := utiltx.GenerateAddress()
	avsName, slashAddress, rewardAddress := "avsTest", "0xDF907c29719154eb9872f021d21CAE6E5025d7aB", "0xDF907c29719154eb9872f021d21CAE6E5025d7aB"
	avsOwnerAddresses := []common.Address{
		s.Address,
		utiltx.GenerateAddress(),
		utiltx.GenerateAddress(),
	}
	imWhiteListAddresses := []common.Address{
		utiltx.GenerateAddress(),
		utiltx.GenerateAddress(),
	}
	assetIDs := s.AssetIDs
	minStakeAmount, taskAddress := uint64(3), "0xDF907c29719154eb9872f021d21CAE6E5025d7aB"
	avsUnbondingPeriod, minSelfDelegation := uint64(3), uint64(3)
	epochIdentifier := epochstypes.DayEpochID
	method := s.precompile.Methods[avs.MethodRegisterAVS]
	testCases := []struct {
		name        string
		sender      common.Address
		origin      common.Address
		malleate    func() []interface{}
		ibcSetup    bool
		expError    bool
		errContains string
	}{
		{
			name:   "pass for avs-registered",
			sender: senderAddress,
			origin: senderAddress,
			malleate: func() []interface{} {
				return []interface{}{
					avs.Params{
						Sender:              senderAddress,
						AvsName:             avsName,
						MinStakeAmount:      minStakeAmount,
						TaskAddress:         common.HexToAddress(taskAddress),
						SlashAddress:        common.HexToAddress(slashAddress),
						RewardAddress:       common.HexToAddress(rewardAddress),
						AvsOwnerAddresses:   avsOwnerAddresses,
						WhitelistAddresses:  imWhiteListAddresses,
						AssetIDs:            assetIDs,
						AvsUnbondingPeriod:  avsUnbondingPeriod,
						MinSelfDelegation:   minSelfDelegation,
						EpochIdentifier:     epochIdentifier,
						MiniOptInOperators:  1,
						MinTotalStakeAmount: 1,
						AvsRewardProportion: 5,
						AvsSlashProportion:  5,
					},
				}
			},
			expError: false,
			ibcSetup: true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			contract := vm.NewContract(vm.AccountRef(tc.sender), s.precompile, big.NewInt(0), gas)
			_, err := s.precompile.RegisterAVS(
				s.Ctx,
				tc.origin,
				contract,
				s.StateDB,
				&method,
				tc.malleate(),
			)
			if tc.expError {
				s.Require().ErrorContains(err, tc.errContains)
			} else {
				s.Require().NoError(err)
			}
		})
	}
}

func (suite *AVSManagerPrecompileSuite) TestDeregisterAVS() {
	avsName := "avsTest"
	commonMalleate := func() (common.Address, []byte) {
		// prepare the call input for avs test
		input, err := suite.precompile.Pack(
			avs.MethodDeregisterAVS,
			suite.Address,
			avsName,
		)
		suite.Require().NoError(err, "failed to pack input")
		return suite.Address, input
	}
	successRet, err := suite.precompile.Methods[avs.MethodDeregisterAVS].Outputs.Pack(true)
	suite.Require().NoError(err)
	setUp := func() {
		slashAddress, rewardAddress := "0xDF907c29719154eb9872f021d21CAE6E5025d7aB", "0xDF907c29719154eb9872f021d21CAE6E5025d7aB"
		avsOwnerAddresses := []string{
			sdk.AccAddress(suite.Address.Bytes()).String(),
			sdk.AccAddress(utiltx.GenerateAddress().Bytes()).String(),
			sdk.AccAddress(utiltx.GenerateAddress().Bytes()).String(),
		}
		assetIDs := suite.AssetIDs
		minStakeAmount, taskAddress := uint64(3), "0xDF907c29719154eb9872f021d21CAE6E5025d7aB"
		avsUnbondingPeriod, minSelfDelegation := uint64(3), uint64(3)
		epochIdentifier := epochstypes.DayEpochID
		params := []uint64{2, 3, 4, 4}
		avs := &types.AVSInfo{
			Name:                avsName,
			AvsAddress:          suite.Address.String(),
			SlashAddress:        slashAddress,
			RewardAddress:       rewardAddress,
			AvsOwnerAddresses:   avsOwnerAddresses,
			AssetIDs:            assetIDs,
			AvsUnbondingPeriod:  avsUnbondingPeriod,
			MinSelfDelegation:   minSelfDelegation,
			EpochIdentifier:     epochIdentifier,
			StartingEpoch:       1,
			TaskAddress:         taskAddress,
			MinStakeAmount:      minStakeAmount,
			MinOptInOperators:   params[0],
			MinTotalStakeAmount: params[1],
			AvsReward:           sdk.MustNewDecFromStr(strconv.Itoa(int(params[1]))),
			AvsSlash:            sdk.MustNewDecFromStr(strconv.Itoa(int(params[2]))),
		}

		err := suite.App.AVSManagerKeeper.SetAVSInfo(suite.Ctx, avs)
		suite.NoError(err)
		for i := 0; i < int(avsUnbondingPeriod)+2; i++ {
			suite.CommitAfter(time.Hour*24 + time.Nanosecond)
		}
	}
	testcases := []struct {
		name        string
		malleate    func() (common.Address, []byte)
		readOnly    bool
		expPass     bool
		errContains string
		returnBytes []byte
	}{
		{
			name: "pass for avs-deregister",
			malleate: func() (common.Address, []byte) {
				setUp()
				return commonMalleate()
			},
			readOnly:    false,
			expPass:     true,
			returnBytes: successRet,
		},
	}

	for _, tc := range testcases {
		tc := tc
		suite.Run(tc.name, func() {
			baseFee := suite.App.FeeMarketKeeper.GetBaseFee(suite.Ctx)

			// malleate testcase
			caller, input := tc.malleate()

			contract := vm.NewPrecompile(vm.AccountRef(caller), suite.precompile, big.NewInt(0), uint64(1e6))
			contract.Input = input

			contractAddress := contract.Address()
			// Build and sign Ethereum transaction
			txArgs := evmtypes.EvmTxArgs{
				ChainID:   suite.App.EvmKeeper.ChainID(),
				Nonce:     0,
				To:        &contractAddress,
				Amount:    nil,
				GasLimit:  100000,
				GasPrice:  app.MainnetMinGasPrices.BigInt(),
				GasFeeCap: baseFee,
				GasTipCap: big.NewInt(1),
				Accesses:  &ethtypes.AccessList{},
			}
			msgEthereumTx := evmtypes.NewTx(&txArgs)

			msgEthereumTx.From = suite.Address.String()
			err := msgEthereumTx.Sign(suite.EthSigner, suite.Signer)
			suite.Require().NoError(err, "failed to sign Ethereum message")

			// Instantiate config
			proposerAddress := suite.Ctx.BlockHeader().ProposerAddress
			cfg, err := suite.App.EvmKeeper.EVMConfig(suite.Ctx, proposerAddress, suite.App.EvmKeeper.ChainID())
			suite.Require().NoError(err, "failed to instantiate EVM config")

			msg, err := msgEthereumTx.AsMessage(suite.EthSigner, baseFee)
			suite.Require().NoError(err, "failed to instantiate Ethereum message")

			// Instantiate EVM
			evm := suite.App.EvmKeeper.NewEVM(
				suite.Ctx, msg, cfg, nil, suite.StateDB,
			)

			params := suite.App.EvmKeeper.GetParams(suite.Ctx)
			activePrecompiles := params.GetActivePrecompilesAddrs()
			precompileMap := suite.App.EvmKeeper.Precompiles(activePrecompiles...)
			err = vm.ValidatePrecompiles(precompileMap, activePrecompiles)
			suite.Require().NoError(err, "invalid precompiles", activePrecompiles)
			evm.WithPrecompiles(precompileMap, activePrecompiles)

			// Run precompiled contract
			bz, err := suite.precompile.Run(evm, contract, tc.readOnly)

			// Check results
			if tc.expPass {
				suite.Require().NoError(err, "expected no error when running the precompile")
				suite.Require().Equal(tc.returnBytes, bz, "the return doesn't match the expected result")
			} else {
				suite.Require().Error(err, "expected error to be returned when running the precompile")
				suite.Require().Nil(bz, "expected returned bytes to be nil")
				suite.Require().ErrorContains(err, tc.errContains)
			}
		})
	}
}

func (suite *AVSManagerPrecompileSuite) TestUpdateAVS() {
	gas := uint64(2_000)
	senderAddress := utiltx.GenerateAddress()
	avsName, slashAddress, rewardAddress := "avsTest-update", "0xDF907c29719154eb9872f021d21CAE6E5025d7aB", "0xDF907c29719154eb9872f021d21CAE6E5025d7aB"
	avsOwnerAddresses := []common.Address{
		s.Address,
		utiltx.GenerateAddress(),
		utiltx.GenerateAddress(),
	}
	imWhiteListAddresses := []common.Address{
		utiltx.GenerateAddress(),
		utiltx.GenerateAddress(),
	}
	assetIDs := s.AssetIDs
	minStakeAmount, taskAddr := uint64(3), "0xDF907c29719154eb9872f021d21CAE6E5025d7aB"
	avsUnbondingPeriod, minSelfDelegation := uint64(3), uint64(3)
	epochIdentifier := epochstypes.DayEpochID
	method := s.precompile.Methods[avs.MethodUpdateAVS]
	testCases := []struct {
		name        string
		sender      common.Address
		origin      common.Address
		malleate    func() []interface{}
		ibcSetup    bool
		expError    bool
		errContains string
	}{
		{
			name:   "pass for avs-update",
			sender: senderAddress,
			origin: senderAddress,
			malleate: func() []interface{} {
				return []interface{}{
					avs.Params{
						Sender:              senderAddress,
						AvsName:             avsName,
						MinStakeAmount:      minStakeAmount,
						TaskAddress:         common.HexToAddress(taskAddr),
						SlashAddress:        common.HexToAddress(slashAddress),
						RewardAddress:       common.HexToAddress(rewardAddress),
						AvsOwnerAddresses:   avsOwnerAddresses,
						WhitelistAddresses:  imWhiteListAddresses,
						AssetIDs:            assetIDs,
						AvsUnbondingPeriod:  avsUnbondingPeriod,
						MinSelfDelegation:   minSelfDelegation,
						EpochIdentifier:     epochIdentifier,
						MiniOptInOperators:  1,
						MinTotalStakeAmount: 1,
						AvsRewardProportion: 5,
						AvsSlashProportion:  5,
					},
				}
			},
			expError: false,
			ibcSetup: true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			contract := vm.NewContract(vm.AccountRef(tc.sender), s.precompile, big.NewInt(0), gas)
			_, err := s.precompile.RegisterAVS(
				s.Ctx,
				tc.origin,
				contract,
				s.StateDB,
				&method,
				tc.malleate(),
			)
			if tc.expError {
				s.Require().ErrorContains(err, tc.errContains)
			} else {
				s.Require().NoError(err)
			}
		})
	}
}

func (suite *AVSManagerPrecompileSuite) TestRegisterOperatorToAVS() {
	// from := s.Address
	operatorAddress := sdk.AccAddress(suite.Address.Bytes())
	assetIDs := suite.AssetIDs
	minStakeAmount, taskAddress := uint64(3), "0x3e108c058e8066DA635321Dc3018294cA82ddEdf"
	avsUnbondingPeriod, minSelfDelegation := uint64(3), uint64(3)
	epochIdentifier := epochstypes.DayEpochID
	params := []uint64{2, 3, 4, 4}

	registerOperator := func() {
		registerReq := &operatortypes.RegisterOperatorReq{
			FromAddress: operatorAddress.String(),
			Info: &operatortypes.OperatorInfo{
				EarningsAddr: operatorAddress.String(),
				ApproveAddr:  operatorAddress.String(),
			},
		}
		_, err := suite.OperatorMsgServer.RegisterOperator(sdk.WrapSDKContext(suite.Ctx), registerReq)
		suite.NoError(err)
		asset := suite.Assets[0]
		_, assetID := assetstypes.GetStakerIDAndAssetIDFromStr(asset.LayerZeroChainID, "", asset.Address)
		selfDelegateAmount := big.NewInt(10)
		minPrecisionSelfDelegateAmount := big.NewInt(0).Mul(selfDelegateAmount, big.NewInt(0).Exp(big.NewInt(10), big.NewInt(int64(asset.Decimals)), nil))
		err = suite.App.AssetsKeeper.UpdateOperatorAssetState(suite.Ctx, operatorAddress, assetID, assetstypes.DeltaOperatorSingleAsset{
			TotalAmount:   math.NewIntFromBigInt(minPrecisionSelfDelegateAmount),
			TotalShare:    math.LegacyNewDecFromBigInt(minPrecisionSelfDelegateAmount),
			OperatorShare: math.LegacyNewDecFromBigInt(minPrecisionSelfDelegateAmount),
		})
	}
	avsName, slashAddress, rewardAddress := "avsTest", "0xDF907c29719154eb9872f021d21CAE6E5025d7aB", "0xDF907c29719154eb9872f021d21CAE6E5025d7aB"
	avsOwnerAddresses := []string{
		sdk.AccAddress(suite.Address.Bytes()).String(),
		sdk.AccAddress(utiltx.GenerateAddress().Bytes()).String(),
		sdk.AccAddress(utiltx.GenerateAddress().Bytes()).String(),
	}

	setUp := func() {
		avs := &types.AVSInfo{
			Name:                avsName,
			AvsAddress:          suite.Address.String(),
			SlashAddress:        slashAddress,
			RewardAddress:       rewardAddress,
			AvsOwnerAddresses:   avsOwnerAddresses,
			AssetIDs:            assetIDs,
			AvsUnbondingPeriod:  avsUnbondingPeriod,
			MinSelfDelegation:   minSelfDelegation,
			EpochIdentifier:     epochIdentifier,
			StartingEpoch:       1,
			TaskAddress:         taskAddress,
			MinStakeAmount:      minStakeAmount,
			MinOptInOperators:   params[0],
			MinTotalStakeAmount: params[1],
			AvsReward:           sdk.MustNewDecFromStr(strconv.Itoa(int(params[1]))),
			AvsSlash:            sdk.MustNewDecFromStr(strconv.Itoa(int(params[2]))),
		}

		err := suite.App.AVSManagerKeeper.SetAVSInfo(suite.Ctx, avs)
		suite.NoError(err)
	}
	commonMalleate := func() (common.Address, []byte) {
		input, err := suite.precompile.Pack(
			avs.MethodRegisterOperatorToAVS,
			suite.Address,
		)
		suite.Require().NoError(err, "failed to pack input")
		return suite.Address, input
	}
	successRet, err := suite.precompile.Methods[avs.MethodRegisterAVS].Outputs.Pack(true)
	suite.Require().NoError(err)

	testcases := []struct {
		name        string
		malleate    func() (common.Address, []byte)
		readOnly    bool
		expPass     bool
		errContains string
		returnBytes []byte
	}{
		{
			name: "pass for operator opt-in avs",
			malleate: func() (common.Address, []byte) {
				registerOperator()
				setUp()
				avsAddress, intput := commonMalleate()
				asset := suite.Assets[0]
				_, defaultAssetID := assetstypes.GetStakerIDAndAssetIDFromStr(asset.LayerZeroChainID, "", asset.Address)
				err = suite.App.AVSManagerKeeper.UpdateAVSInfo(suite.Ctx, &types.AVSRegisterOrDeregisterParams{
					Action:     types.UpdateAction,
					AvsAddress: avsAddress,
					AssetIDs:   []string{defaultAssetID},
				})
				suite.NoError(err)
				return avsAddress, intput
			},
			readOnly:    false,
			expPass:     true,
			returnBytes: successRet,
		},
	}

	for _, tc := range testcases {
		tc := tc
		suite.Run(tc.name, func() {
			baseFee := suite.App.FeeMarketKeeper.GetBaseFee(suite.Ctx)

			// malleate testcase
			caller, input := tc.malleate()
			contract := vm.NewPrecompile(vm.AccountRef(caller), suite.precompile, big.NewInt(0), uint64(1e6))
			contract.Input = input
			contract.CallerAddress = caller

			contractAddress := contract.Address()
			// Build and sign Ethereum transaction
			txArgs := evmtypes.EvmTxArgs{
				ChainID:   suite.App.EvmKeeper.ChainID(),
				Nonce:     0,
				To:        &contractAddress,
				Amount:    nil,
				GasLimit:  100000,
				GasPrice:  app.MainnetMinGasPrices.BigInt(),
				GasFeeCap: baseFee,
				GasTipCap: big.NewInt(1),
				Accesses:  &ethtypes.AccessList{},
			}
			msgEthereumTx := evmtypes.NewTx(&txArgs)

			msgEthereumTx.From = suite.Address.String()
			err := msgEthereumTx.Sign(suite.EthSigner, suite.Signer)
			suite.Require().NoError(err, "failed to sign Ethereum message")

			// Instantiate config
			proposerAddress := suite.Ctx.BlockHeader().ProposerAddress
			cfg, err := suite.App.EvmKeeper.EVMConfig(suite.Ctx, proposerAddress, suite.App.EvmKeeper.ChainID())
			suite.Require().NoError(err, "failed to instantiate EVM config")

			msg, err := msgEthereumTx.AsMessage(suite.EthSigner, baseFee)
			suite.Require().NoError(err, "failed to instantiate Ethereum message")

			// Instantiate EVM
			evm := suite.App.EvmKeeper.NewEVM(
				suite.Ctx, msg, cfg, nil, suite.StateDB,
			)

			params := suite.App.EvmKeeper.GetParams(suite.Ctx)
			activePrecompiles := params.GetActivePrecompilesAddrs()
			precompileMap := suite.App.EvmKeeper.Precompiles(activePrecompiles...)
			err = vm.ValidatePrecompiles(precompileMap, activePrecompiles)
			suite.Require().NoError(err, "invalid precompiles", activePrecompiles)
			evm.WithPrecompiles(precompileMap, activePrecompiles)

			// Run precompiled contract
			bz, err := suite.precompile.Run(evm, contract, tc.readOnly)

			// Check results
			if tc.expPass {
				suite.Require().NoError(err, "expected no error when running the precompile")
				suite.Require().Equal(tc.returnBytes, bz, "the return doesn't match the expected result")
			} else {
				suite.Require().Error(err, "expected error to be returned when running the precompile")
				suite.Require().Nil(bz, "expected returned bytes to be nil")
				suite.Require().ErrorContains(err, tc.errContains)
			}
		})
	}
}

func (suite *AVSManagerPrecompileSuite) TestDeregisterOperatorFromAVS() {
	// from := s.Address
	operatorAddress := sdk.AccAddress(suite.Address.Bytes())
	assetIDs := suite.AssetIDs
	minStakeAmount, taskAddress := uint64(3), "0x3e108c058e8066DA635321Dc3018294cA82ddEdf"
	avsUnbondingPeriod, minSelfDelegation := uint64(3), uint64(3)
	epochIdentifier := epochstypes.DayEpochID
	params := []uint64{2, 3, 4, 4}

	registerOperator := func() {
		registerReq := &operatortypes.RegisterOperatorReq{
			FromAddress: operatorAddress.String(),
			Info: &operatortypes.OperatorInfo{
				EarningsAddr: operatorAddress.String(),
				ApproveAddr:  operatorAddress.String(),
			},
		}
		_, err := suite.OperatorMsgServer.RegisterOperator(sdk.WrapSDKContext(suite.Ctx), registerReq)
		suite.NoError(err)
		asset := suite.Assets[0]
		_, assetID := assetstypes.GetStakerIDAndAssetIDFromStr(asset.LayerZeroChainID, "", asset.Address)
		selfDelegateAmount := big.NewInt(10)
		minPrecisionSelfDelegateAmount := big.NewInt(0).Mul(selfDelegateAmount, big.NewInt(0).Exp(big.NewInt(10), big.NewInt(int64(asset.Decimals)), nil))
		err = suite.App.AssetsKeeper.UpdateOperatorAssetState(suite.Ctx, operatorAddress, assetID, assetstypes.DeltaOperatorSingleAsset{
			TotalAmount:   math.NewIntFromBigInt(minPrecisionSelfDelegateAmount),
			TotalShare:    math.LegacyNewDecFromBigInt(minPrecisionSelfDelegateAmount),
			OperatorShare: math.LegacyNewDecFromBigInt(minPrecisionSelfDelegateAmount),
		})
	}
	avsName, slashAddress, rewardAddress := "avsTest", "0xDF907c29719154eb9872f021d21CAE6E5025d7aB", "0xDF907c29719154eb9872f021d21CAE6E5025d7aB"
	avsOwnerAddresses := []string{
		sdk.AccAddress(suite.Address.Bytes()).String(),
		sdk.AccAddress(utiltx.GenerateAddress().Bytes()).String(),
		sdk.AccAddress(utiltx.GenerateAddress().Bytes()).String(),
	}

	setUp := func() {
		avsInfo := &types.AVSInfo{
			Name:                avsName,
			AvsAddress:          suite.Address.String(),
			SlashAddress:        slashAddress,
			RewardAddress:       rewardAddress,
			AvsOwnerAddresses:   avsOwnerAddresses,
			AssetIDs:            assetIDs,
			AvsUnbondingPeriod:  avsUnbondingPeriod,
			MinSelfDelegation:   minSelfDelegation,
			EpochIdentifier:     epochIdentifier,
			StartingEpoch:       1,
			TaskAddress:         taskAddress,
			MinStakeAmount:      minStakeAmount,
			MinOptInOperators:   params[0],
			MinTotalStakeAmount: params[1],
			AvsReward:           sdk.MustNewDecFromStr(strconv.Itoa(int(params[1]))),
			AvsSlash:            sdk.MustNewDecFromStr(strconv.Itoa(int(params[2]))),
		}

		err := suite.App.AVSManagerKeeper.SetAVSInfo(suite.Ctx, avsInfo)
		suite.NoError(err)
	}
	optIn := func() {
		operatorParams := &types.OperatorOptParams{}
		operatorParams.OperatorAddress = operatorAddress
		operatorParams.AvsAddress = suite.Address
		operatorParams.Action = types.RegisterAction

		err := suite.App.AVSManagerKeeper.OperatorOptAction(suite.Ctx, operatorParams)
		suite.NoError(err)
	}
	commonMalleate := func() (common.Address, []byte) {
		input, err := suite.precompile.Pack(
			avs.MethodDeregisterOperatorFromAVS,
			suite.Address,
		)
		suite.Require().NoError(err, "failed to pack input")
		return suite.Address, input
	}
	successRet, err := suite.precompile.Methods[avs.MethodDeregisterOperatorFromAVS].Outputs.Pack(true)
	suite.Require().NoError(err)

	testcases := []struct {
		name        string
		malleate    func() (common.Address, []byte)
		readOnly    bool
		expPass     bool
		errContains string
		returnBytes []byte
	}{
		{
			name: "pass for operator opt-out avs",
			malleate: func() (common.Address, []byte) {
				registerOperator()
				setUp()
				optIn()
				return commonMalleate()
			},
			readOnly:    false,
			expPass:     true,
			returnBytes: successRet,
		},
	}

	for _, tc := range testcases {
		tc := tc
		suite.Run(tc.name, func() {
			baseFee := suite.App.FeeMarketKeeper.GetBaseFee(suite.Ctx)

			// malleate testcase
			caller, input := tc.malleate()
			contract := vm.NewPrecompile(vm.AccountRef(caller), suite.precompile, big.NewInt(0), uint64(1e6))
			contract.Input = input
			contract.CallerAddress = caller

			contractAddress := contract.Address()
			// Build and sign Ethereum transaction
			txArgs := evmtypes.EvmTxArgs{
				ChainID:   suite.App.EvmKeeper.ChainID(),
				Nonce:     0,
				To:        &contractAddress,
				Amount:    nil,
				GasLimit:  100000,
				GasPrice:  app.MainnetMinGasPrices.BigInt(),
				GasFeeCap: baseFee,
				GasTipCap: big.NewInt(1),
				Accesses:  &ethtypes.AccessList{},
			}
			msgEthereumTx := evmtypes.NewTx(&txArgs)

			msgEthereumTx.From = suite.Address.String()
			err := msgEthereumTx.Sign(suite.EthSigner, suite.Signer)
			suite.Require().NoError(err, "failed to sign Ethereum message")

			// Instantiate config
			proposerAddress := suite.Ctx.BlockHeader().ProposerAddress
			cfg, err := suite.App.EvmKeeper.EVMConfig(suite.Ctx, proposerAddress, suite.App.EvmKeeper.ChainID())
			suite.Require().NoError(err, "failed to instantiate EVM config")

			msg, err := msgEthereumTx.AsMessage(suite.EthSigner, baseFee)
			suite.Require().NoError(err, "failed to instantiate Ethereum message")

			// Instantiate EVM
			evm := suite.App.EvmKeeper.NewEVM(
				suite.Ctx, msg, cfg, nil, suite.StateDB,
			)

			params := suite.App.EvmKeeper.GetParams(suite.Ctx)
			activePrecompiles := params.GetActivePrecompilesAddrs()
			precompileMap := suite.App.EvmKeeper.Precompiles(activePrecompiles...)
			err = vm.ValidatePrecompiles(precompileMap, activePrecompiles)
			suite.Require().NoError(err, "invalid precompiles", activePrecompiles)
			evm.WithPrecompiles(precompileMap, activePrecompiles)

			// Run precompiled contract
			bz, err := suite.precompile.Run(evm, contract, tc.readOnly)

			// Check results
			if tc.expPass {
				suite.Require().NoError(err, "expected no error when running the precompile")
				suite.Require().Equal(tc.returnBytes, bz, "the return doesn't match the expected result")
			} else {
				suite.Require().Error(err, "expected error to be returned when running the precompile")
				suite.Require().Nil(bz, "expected returned bytes to be nil")
				suite.Require().ErrorContains(err, tc.errContains)
			}
		})
	}
}

// TestRun tests the precompiles Run method reg avstask.
func (suite *AVSManagerPrecompileSuite) TestRunRegTaskInfo() {
	taskAddress := utiltx.GenerateAddress()
	setUp := func() {
		suite.prepare()
		// register the new token
		usdcAddress := common.HexToAddress("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48")
		usdcClientChainAsset := assetstypes.AssetInfo{
			Name:             "USD coin",
			Symbol:           "USDC",
			Address:          usdcAddress.String(),
			Decimals:         6,
			LayerZeroChainID: 101,
			MetaInfo:         "USDC",
		}
		err := suite.App.AssetsKeeper.SetStakingAssetInfo(
			suite.Ctx,
			&assetstypes.StakingAssetInfo{
				AssetBasicInfo:     usdcClientChainAsset,
				StakingTotalAmount: sdkmath.ZeroInt(),
			},
		)
		suite.NoError(err)
		// register the new AVS
		suite.prepareAvs([]string{"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48_0x65", "0xdac17f958d2ee523a2206206994597c13d831ec7_0x65"}, taskAddress.String())
		// opt in
		err = suite.App.OperatorKeeper.OptIn(suite.Ctx, suite.operatorAddress, suite.avsAddress)
		suite.NoError(err)
		usdtPrice, err := suite.App.OperatorKeeper.OracleInterface().GetSpecifiedAssetsPrice(suite.Ctx, suite.assetID)
		suite.NoError(err)
		operatorKeeper.CalculateUSDValue(suite.delegationAmount, usdtPrice.Value, suite.assetDecimal, usdtPrice.Decimal)
		// deposit and delegate another asset to the operator
		suite.NoError(err)
		suite.prepareDeposit(usdcAddress, sdkmath.NewInt(1e8))
		usdcPrice, err := suite.App.OperatorKeeper.OracleInterface().GetSpecifiedAssetsPrice(suite.Ctx, suite.assetID)
		suite.NoError(err)
		delegatedAmount := sdkmath.NewIntWithDecimal(8, 7)
		suite.prepareDelegation(true, usdcAddress, delegatedAmount)

		// updating the new voting power
		operatorKeeper.CalculateUSDValue(suite.delegationAmount, usdcPrice.Value, suite.assetDecimal, usdcPrice.Decimal)
		suite.CommitAfter(time.Hour*3 + time.Nanosecond)
	}
	commonMalleate := func() (common.Address, []byte) {
		input, err := suite.precompile.Pack(
			avs.MethodCreateAVSTask,
			suite.Address,
			"test-avstask",
			rand.Bytes(3),
			uint64(3),
			uint64(3),
			uint8(3),
			uint64(3),
		)
		suite.Require().NoError(err, "failed to pack input")
		return suite.Address, input
	}
	successRet, err := suite.precompile.Methods[avs.MethodCreateAVSTask].Outputs.Pack(uint64(1))
	suite.Require().NoError(err)
	testcases := []struct {
		name        string
		malleate    func() (common.Address, []byte)
		readOnly    bool
		expPass     bool
		errContains string
		returnBytes []byte
	}{
		{
			name: "pass - avstask via pre-compiles",
			malleate: func() (common.Address, []byte) {
				suite.Require().NoError(err)
				setUp()
				return commonMalleate()
			},
			returnBytes: successRet,
			readOnly:    false,
			expPass:     true,
		},
	}
	for _, tc := range testcases {
		tc := tc
		suite.Run(tc.name, func() {
			baseFee := suite.App.FeeMarketKeeper.GetBaseFee(suite.Ctx)

			// malleate testcase
			caller, input := tc.malleate()

			contract := vm.NewPrecompile(vm.AccountRef(caller), suite.precompile, big.NewInt(0), uint64(1e6))
			contract.Input = input
			contract.CallerAddress = taskAddress

			contractAddress := contract.Address()
			// Build and sign Ethereum transaction
			txArgs := evmtypes.EvmTxArgs{
				ChainID:   suite.App.EvmKeeper.ChainID(),
				Nonce:     0,
				To:        &contractAddress,
				Amount:    nil,
				GasLimit:  100000,
				GasPrice:  app.MainnetMinGasPrices.BigInt(),
				GasFeeCap: baseFee,
				GasTipCap: big.NewInt(1),
				Accesses:  &ethtypes.AccessList{},
			}
			msgEthereumTx := evmtypes.NewTx(&txArgs)

			msgEthereumTx.From = suite.Address.String()
			err := msgEthereumTx.Sign(suite.EthSigner, suite.Signer)
			suite.Require().NoError(err, "failed to sign Ethereum message")

			// Instantiate config
			proposerAddress := suite.Ctx.BlockHeader().ProposerAddress
			cfg, err := suite.App.EvmKeeper.EVMConfig(suite.Ctx, proposerAddress, suite.App.EvmKeeper.ChainID())
			suite.Require().NoError(err, "failed to instantiate EVM config")

			msg, err := msgEthereumTx.AsMessage(suite.EthSigner, baseFee)
			suite.Require().NoError(err, "failed to instantiate Ethereum message")

			// Create StateDB
			suite.StateDB = statedb.New(suite.Ctx, suite.App.EvmKeeper, statedb.NewEmptyTxConfig(common.BytesToHash(suite.Ctx.HeaderHash().Bytes())))
			// Instantiate EVM
			evm := suite.App.EvmKeeper.NewEVM(
				suite.Ctx, msg, cfg, nil, suite.StateDB,
			)
			params := suite.App.EvmKeeper.GetParams(suite.Ctx)
			activePrecompiles := params.GetActivePrecompilesAddrs()
			precompileMap := suite.App.EvmKeeper.Precompiles(activePrecompiles...)
			err = vm.ValidatePrecompiles(precompileMap, activePrecompiles)
			suite.Require().NoError(err, "invalid precompiles", activePrecompiles)
			evm.WithPrecompiles(precompileMap, activePrecompiles)

			// Run precompiled contract
			bz, err := suite.precompile.Run(evm, contract, tc.readOnly)

			// Check results
			if tc.expPass {
				suite.Require().NoError(err, "expected no error when running the precompile")
				suite.Require().Equal(tc.returnBytes, bz, "the return doesn't match the expected result")
			} else {
				suite.Require().Error(err, "expected error to be returned when running the precompile")
				suite.Require().Nil(bz, "expected returned bytes to be nil")
				suite.Require().ErrorContains(err, tc.errContains)
			}
		})
	}
}
