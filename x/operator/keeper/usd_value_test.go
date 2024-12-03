package keeper_test

import (
	"time"

	"github.com/ExocoreNetwork/exocore/x/epochs/types"

	sdkmath "cosmossdk.io/math"
	assetstype "github.com/ExocoreNetwork/exocore/x/assets/types"
	avstypes "github.com/ExocoreNetwork/exocore/x/avs/types"
	operatorKeeper "github.com/ExocoreNetwork/exocore/x/operator/keeper"
	"github.com/ethereum/go-ethereum/common"
)

const (
	MaxDecForTotalSupply = 38
)

var (
	MaxAssetTotalSupply  = sdkmath.NewIntWithDecimal(1, MaxDecForTotalSupply)
	defaultClientChainID = uint64(101)
	assetDecimal         = 6
	usdcAddr             = common.HexToAddress("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48")
	usdcAssetID          = "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48_0x65"
	usdtAddr             = common.HexToAddress("0xdac17f958d2ee523a2206206994597c13d831ec7")
	usdtAssetID          = "0xdac17f958d2ee523a2206206994597c13d831ec7_0x65"
)

func (suite *OperatorTestSuite) TestCalculateUSDValue() {
	suite.prepare()
	price, err := suite.App.OperatorKeeper.OracleInterface().GetSpecifiedAssetsPrice(suite.Ctx, suite.assetID)
	suite.NoError(err)
	usdValue := operatorKeeper.CalculateUSDValue(suite.delegationAmount, price.Value, suite.assetDecimal, price.Decimal)
	expectedValue := sdkmath.LegacyNewDecFromBigInt(suite.delegationAmount.BigInt()).QuoInt(sdkmath.NewIntWithDecimal(1, int(suite.assetDecimal)))
	suite.Equal(expectedValue, usdValue)
	suite.Equal(int64(0), usdValue.TruncateInt64())
	float64Value, err := usdValue.Float64()
	suite.NoError(err)
	suite.Equal(5e-05, float64Value)
}

func (suite *OperatorTestSuite) TestCalculatedUSDValueOverflow() {
	price := MaxAssetTotalSupply
	priceDecimal := uint8(assetstype.MaxDecimal)
	amount := MaxAssetTotalSupply
	assetDecimal := uint32(assetstype.MaxDecimal)
	usdValue := operatorKeeper.CalculateUSDValue(amount, price, assetDecimal, priceDecimal)
	expectedValue := sdkmath.LegacyNewDecFromBigInt(sdkmath.NewIntWithDecimal(1, 2*MaxDecForTotalSupply-2*assetstype.MaxDecimal).BigInt())
	suite.Equal(expectedValue, usdValue)

	priceDecimal = uint8(0)
	assetDecimal = uint32(0)
	usdValue = operatorKeeper.CalculateUSDValue(amount, price, assetDecimal, priceDecimal)
	expectedValue = sdkmath.LegacyNewDecFromBigInt(sdkmath.NewIntWithDecimal(1, 2*MaxDecForTotalSupply).BigInt())
	suite.Equal(expectedValue, usdValue)

	price = sdkmath.NewInt(1)
	priceDecimal = uint8(assetstype.MaxDecimal)
	amount = sdkmath.NewInt(1)
	assetDecimal = uint32(assetstype.MaxDecimal)
	usdValue = operatorKeeper.CalculateUSDValue(amount, price, assetDecimal, priceDecimal)
	expectedValue = sdkmath.LegacyZeroDec()
	suite.Equal(expectedValue.String(), usdValue.String())

	price = sdkmath.NewInt(1)
	priceDecimal = uint8(0)
	amount = sdkmath.NewInt(1)
	assetDecimal = uint32(assetstype.MaxDecimal)
	usdValue = operatorKeeper.CalculateUSDValue(amount, price, assetDecimal, priceDecimal)
	expectedValue = sdkmath.LegacyNewDecFromBigIntWithPrec(amount.BigInt(), sdkmath.LegacyPrecision)
	suite.Equal(expectedValue, usdValue)
	float64Value, err := usdValue.Float64()
	suite.NoError(err)
	suite.Equal(1e-18, float64Value)
}

func (suite *OperatorTestSuite) TestAVSUSDValue() {
	suite.prepare()
	// register the new token
	usdcClientChainAsset := assetstype.AssetInfo{
		Name:             "USD coin",
		Symbol:           "USDC",
		Address:          usdcAddr.String(),
		Decimals:         uint32(assetDecimal),
		LayerZeroChainID: defaultClientChainID,
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
	suite.prepareAvs([]string{usdcAssetID, usdtAssetID}, types.HourEpochID)
	// opt in
	err = suite.App.OperatorKeeper.OptIn(suite.Ctx, suite.operatorAddr, suite.avsAddr)
	suite.NoError(err)
	usdtPrice, err := suite.App.OperatorKeeper.OracleInterface().GetSpecifiedAssetsPrice(suite.Ctx, suite.assetID)
	suite.NoError(err)
	usdtValue := operatorKeeper.CalculateUSDValue(suite.delegationAmount, usdtPrice.Value, suite.assetDecimal, usdtPrice.Decimal)
	// deposit and delegate another asset to the operator
	suite.NoError(err)
	suite.prepareDeposit(suite.Address, usdcAddr, sdkmath.NewInt(1e8))
	usdcPrice, err := suite.App.OperatorKeeper.OracleInterface().GetSpecifiedAssetsPrice(suite.Ctx, suite.assetID)
	suite.NoError(err)
	delegatedAmount := sdkmath.NewIntWithDecimal(8, 7)
	suite.prepareDelegation(true, suite.Address, usdcAddr, suite.operatorAddr, delegatedAmount)

	// updating the new voting power
	usdcValue := operatorKeeper.CalculateUSDValue(suite.delegationAmount, usdcPrice.Value, suite.assetDecimal, usdcPrice.Decimal)
	expectedUSDvalue := usdcValue.Add(usdtValue)
	suite.CommitAfter(time.Hour*1 + time.Nanosecond)
	suite.CommitAfter(time.Hour*1 + time.Nanosecond)
	suite.CommitAfter(time.Hour*1 + time.Nanosecond)
	avsUSDValue, err := suite.App.OperatorKeeper.GetAVSUSDValue(suite.Ctx, suite.avsAddr)
	suite.NoError(err)
	suite.Equal(expectedUSDvalue, avsUSDValue)
	optedUSDValues, err := suite.App.OperatorKeeper.GetOperatorOptedUSDValue(suite.Ctx, suite.avsAddr, suite.operatorAddr.String())
	suite.NoError(err)
	suite.Equal(expectedUSDvalue, optedUSDValues.TotalUSDValue)
}

func (suite *OperatorTestSuite) TestVotingPowerForDogFood() {
	initialPowers := suite.Powers
	addPower := 1
	addUSDValue := sdkmath.LegacyNewDec(1)

	chainIDWithoutRevision := avstypes.ChainIDWithoutRevision(suite.Ctx.ChainID())
	avsAddress := avstypes.GenerateAVSAddr(avstypes.ChainIDWithoutRevision(suite.Ctx.ChainID()))
	// CommitAfter causes the epoch hook to be triggered, and results in writing
	// of the AVS usd value to the store.
	suite.CommitAfter(time.Hour*24 + time.Nanosecond)
	initialAVSUSDValue, err := suite.App.OperatorKeeper.GetAVSUSDValue(suite.Ctx, avsAddress)
	suite.NoError(err)
	operators, _ := suite.App.OperatorKeeper.GetActiveOperatorsForChainID(suite.Ctx, chainIDWithoutRevision)
	suite.Require().GreaterOrEqual(len(operators), 1)
	powers, err := suite.App.OperatorKeeper.GetVotePowerForChainID(
		suite.Ctx, operators, chainIDWithoutRevision,
	)
	suite.NoError(err)
	allAssets, err := suite.App.AssetsKeeper.GetAllStakingAssetsInfo(suite.Ctx)
	suite.NoError(err)
	suite.Equal(1, len(allAssets))
	var asset assetstype.AssetInfo
	for _, value := range allAssets {
		asset = value.AssetBasicInfo
	}

	assetAddr := common.HexToAddress(asset.Address)
	depositAmount := sdkmath.NewIntWithDecimal(2, int(asset.Decimals))
	delegationAmount := sdkmath.NewIntWithDecimal(int64(addPower), int(asset.Decimals))
	suite.prepareDeposit(suite.Address, assetAddr, depositAmount)
	// the order here is unknown, so we need to check which operator has the highest power
	if powers[0] > powers[1] {
		suite.operatorAddr = operators[0]
	} else {
		suite.operatorAddr = operators[1]
	}
	suite.prepareDelegation(true, suite.Address, assetAddr, suite.operatorAddr, delegationAmount)
	optedUSDValues, err := suite.App.OperatorKeeper.GetOperatorOptedUSDValue(suite.Ctx, avsAddress, suite.operatorAddr.String())
	suite.NoError(err)
	initialOperatorUSDValue := optedUSDValues.TotalUSDValue

	suite.CommitAfter(time.Hour*24 + time.Nanosecond)
	avsUSDValue, err := suite.App.OperatorKeeper.GetAVSUSDValue(suite.Ctx, avsAddress)
	suite.NoError(err)
	suite.Equal(initialAVSUSDValue.Add(addUSDValue), avsUSDValue)
	optedUSDValues, err = suite.App.OperatorKeeper.GetOperatorOptedUSDValue(suite.Ctx, avsAddress, suite.operatorAddr.String())
	suite.NoError(err)
	suite.Equal(initialOperatorUSDValue.Add(addUSDValue), optedUSDValues.TotalUSDValue)

	found, consensusKey, err := suite.App.OperatorKeeper.GetOperatorConsKeyForChainID(suite.Ctx, suite.operatorAddr, chainIDWithoutRevision)
	suite.NoError(err)
	suite.True(found)

	suite.App.StakingKeeper.MarkUpdateValidatorSetFlag(suite.Ctx)
	validatorUpdates := suite.App.StakingKeeper.EndBlock(suite.Ctx)
	suite.Equal(1, len(validatorUpdates))
	for i, update := range validatorUpdates {
		suite.Equal(*consensusKey.ToTmProtoKey(), update.PubKey)
		// since initialPowers is sorted by power, we picked the operator with the highest power
		suite.Equal(initialPowers[i]+int64(addPower), update.Power)
	}
}
