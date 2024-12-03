package keeper_test

import (
	sdkmath "cosmossdk.io/math"
	assetstype "github.com/ExocoreNetwork/exocore/x/assets/types"
	"github.com/ExocoreNetwork/exocore/x/delegation/keeper"
	delegationtypes "github.com/ExocoreNetwork/exocore/x/delegation/types"
)

func (suite *DelegationTestSuite) TestTokensFromShares() {
	testCases := []struct {
		// input
		totalShare  sdkmath.LegacyDec
		totalAmount sdkmath.Int
		stakerShare sdkmath.LegacyDec
		// output
		stakerAmount sdkmath.Int
		innerError   error
	}{
		// error cases
		{
			totalShare:   sdkmath.LegacyNewDec(50),
			totalAmount:  sdkmath.NewInt(50),
			stakerShare:  sdkmath.LegacyNewDec(51),
			innerError:   delegationtypes.ErrInsufficientShares,
			stakerAmount: sdkmath.ZeroInt(),
		},
		{
			totalShare:   sdkmath.LegacyZeroDec(),
			totalAmount:  sdkmath.NewInt(50),
			stakerShare:  sdkmath.LegacyZeroDec(),
			innerError:   delegationtypes.ErrDivisorIsZero,
			stakerAmount: sdkmath.ZeroInt(),
		},

		// the share will be equal to the amount if there isn't a slash event
		{
			totalShare:   sdkmath.LegacyNewDec(50),
			totalAmount:  sdkmath.NewInt(50),
			stakerShare:  sdkmath.LegacyZeroDec(),
			innerError:   nil,
			stakerAmount: sdkmath.ZeroInt(),
		},
		{
			totalShare:   sdkmath.LegacyNewDec(50),
			totalAmount:  sdkmath.NewInt(50),
			stakerShare:  sdkmath.LegacyMustNewDecFromStr("3.4"),
			innerError:   nil,
			stakerAmount: sdkmath.NewInt(3),
		},
		{
			totalShare:   sdkmath.LegacyNewDec(50),
			totalAmount:  sdkmath.NewInt(50),
			stakerShare:  sdkmath.LegacyNewDec(50),
			innerError:   nil,
			stakerAmount: sdkmath.NewInt(50),
		},

		// the share will be greater than the amount if there is a slash event
		{
			totalShare:   sdkmath.LegacyNewDec(70),
			totalAmount:  sdkmath.NewInt(50),
			stakerShare:  sdkmath.LegacyZeroDec(),
			innerError:   nil,
			stakerAmount: sdkmath.ZeroInt(),
		},
		{
			totalShare:   sdkmath.LegacyNewDec(70),
			totalAmount:  sdkmath.NewInt(50),
			stakerShare:  sdkmath.LegacyMustNewDecFromStr("3.4"),
			innerError:   nil,
			stakerAmount: sdkmath.NewInt(2),
		},
		{
			totalShare:   sdkmath.LegacyNewDec(70),
			totalAmount:  sdkmath.NewInt(50),
			stakerShare:  sdkmath.LegacyNewDec(70),
			innerError:   nil,
			stakerAmount: sdkmath.NewInt(50),
		},

		// all exit
		{
			totalShare:   sdkmath.LegacyZeroDec(),
			stakerShare:  sdkmath.LegacyZeroDec(),
			totalAmount:  sdkmath.ZeroInt(),
			stakerAmount: sdkmath.ZeroInt(),
			innerError:   nil,
		},
	}

	for _, testCase := range testCases {
		amount, err := keeper.TokensFromShares(testCase.stakerShare, testCase.totalShare, testCase.totalAmount)
		if testCase.innerError != nil {
			suite.ErrorContains(err, testCase.innerError.Error())
		} else {
			suite.NoError(err)
			suite.Equal(testCase.stakerAmount, amount)
		}
	}
}

func (suite *DelegationTestSuite) TestSharesFromTokens() {
	testCases := []struct {
		// input
		totalShare   sdkmath.LegacyDec
		totalAmount  sdkmath.Int
		stakerAmount sdkmath.Int

		// output
		stakerShare sdkmath.LegacyDec
		innerError  error
	}{
		// error cases
		{
			totalShare:   sdkmath.LegacyNewDec(50),
			totalAmount:  sdkmath.ZeroInt(),
			stakerAmount: sdkmath.ZeroInt(),
			innerError:   delegationtypes.ErrDivisorIsZero,
			stakerShare:  sdkmath.LegacyZeroDec(),
		},

		// the share will be equal to the amount if there isn't a slash event
		{
			totalShare:   sdkmath.LegacyNewDec(50),
			totalAmount:  sdkmath.NewInt(50),
			stakerAmount: sdkmath.ZeroInt(),
			innerError:   nil,
			stakerShare:  sdkmath.LegacyZeroDec(),
		},
		{
			totalShare:   sdkmath.LegacyNewDec(50),
			totalAmount:  sdkmath.NewInt(50),
			stakerAmount: sdkmath.NewInt(51),
			innerError:   nil,
			stakerShare:  sdkmath.LegacyNewDec(51),
		},
		{
			totalShare:   sdkmath.LegacyNewDec(50),
			totalAmount:  sdkmath.NewInt(50),
			stakerAmount: sdkmath.NewInt(3),
			innerError:   nil,
			stakerShare:  sdkmath.LegacyNewDec(3),
		},
		{
			totalShare:   sdkmath.LegacyNewDec(50),
			totalAmount:  sdkmath.NewInt(50),
			stakerAmount: sdkmath.NewInt(50),
			innerError:   nil,
			stakerShare:  sdkmath.LegacyNewDec(50),
		},

		// the share will be greater than the amount if there is a slash event
		{
			totalShare:   sdkmath.LegacyNewDec(70),
			totalAmount:  sdkmath.NewInt(50),
			stakerAmount: sdkmath.ZeroInt(),
			innerError:   nil,
			stakerShare:  sdkmath.LegacyZeroDec(),
		},
		{
			totalShare:   sdkmath.LegacyNewDec(70),
			totalAmount:  sdkmath.NewInt(50),
			stakerAmount: sdkmath.NewInt(2),
			innerError:   nil,
			stakerShare:  sdkmath.LegacyMustNewDecFromStr("2.8"),
		},
		{
			totalShare:   sdkmath.LegacyNewDec(70),
			totalAmount:  sdkmath.NewInt(50),
			stakerAmount: sdkmath.NewInt(50),
			innerError:   nil,
			stakerShare:  sdkmath.LegacyNewDec(70),
		},

		// all exit
		{
			totalShare:   sdkmath.LegacyZeroDec(),
			totalAmount:  sdkmath.ZeroInt(),
			stakerAmount: sdkmath.ZeroInt(),
			innerError:   nil,
			stakerShare:  sdkmath.LegacyZeroDec(),
		},
	}

	for _, testCase := range testCases {
		share, err := keeper.SharesFromTokens(testCase.totalShare, testCase.stakerAmount, testCase.totalAmount)
		if testCase.innerError != nil {
			suite.ErrorContains(err, testCase.innerError.Error())
		} else {
			suite.NoError(err)
			suite.Equal(testCase.stakerShare, share)
		}
	}
}

func (suite *DelegationTestSuite) TestCalculateShare() {
	assetAmount := sdkmath.NewInt(10)
	// test the case that the operator doesn't exist
	_, assetID := assetstype.GetStakerIDAndAssetID(suite.clientChainLzID, nil, suite.assetAddr[:])
	share, err := suite.App.DelegationKeeper.CalculateShare(suite.Ctx, suite.opAccAddr, assetID, assetAmount)
	suite.NoError(err)
	suite.Equal(sdkmath.LegacyNewDecFromBigInt(assetAmount.BigInt()), share)

	// test the case that the asset amount of operator is zero
	err = suite.App.AssetsKeeper.UpdateOperatorAssetState(suite.Ctx, suite.opAccAddr, assetID, assetstype.DeltaOperatorSingleAsset{
		TotalAmount: sdkmath.ZeroInt(),
		TotalShare:  sdkmath.LegacyZeroDec(),
	})
	suite.NoError(err)
	share, err = suite.App.DelegationKeeper.CalculateShare(suite.Ctx, suite.opAccAddr, assetID, assetAmount)
	suite.NoError(err)
	suite.Equal(sdkmath.LegacyNewDecFromBigInt(assetAmount.BigInt()), share)

	// test normal cases
	err = suite.App.AssetsKeeper.UpdateOperatorAssetState(suite.Ctx, suite.opAccAddr, assetID, assetstype.DeltaOperatorSingleAsset{
		TotalAmount: sdkmath.NewInt(50),
		TotalShare:  sdkmath.LegacyNewDec(60),
	})
	suite.NoError(err)
	share, err = suite.App.DelegationKeeper.CalculateShare(suite.Ctx, suite.opAccAddr, assetID, assetAmount)
	suite.NoError(err)
	suite.Equal(sdkmath.LegacyNewDec(12), share)
}

func (suite *DelegationTestSuite) TestValidateUndelegationAmount() {
	suite.basicPrepare()
	suite.prepareDeposit(suite.depositAmount)
	suite.prepareDelegation(suite.delegationAmount, suite.opAccAddr)
	stakerID, assetID := assetstype.GetStakerIDAndAssetID(suite.clientChainLzID, suite.Address[:], suite.assetAddr[:])

	undelegationAmount := sdkmath.ZeroInt()
	_, err := suite.App.DelegationKeeper.ValidateUndelegationAmount(suite.Ctx, suite.opAccAddr, stakerID, assetID, undelegationAmount)
	suite.Error(err, delegationtypes.ErrAmountIsNotPositive)

	undelegationAmount = sdkmath.NewInt(10)
	share, err := suite.App.DelegationKeeper.ValidateUndelegationAmount(suite.Ctx, suite.opAccAddr, stakerID, assetID, undelegationAmount)
	suite.NoError(err)
	suite.Equal(sdkmath.LegacyNewDecFromBigInt(undelegationAmount.BigInt()), share)

	// test the undelegation amount is greater than the delegated amount
	undelegationAmount = suite.delegationAmount.Add(sdkmath.NewInt(1))
	_, err = suite.App.DelegationKeeper.ValidateUndelegationAmount(suite.Ctx, suite.opAccAddr, stakerID, assetID, undelegationAmount)
	suite.Error(err, delegationtypes.ErrInsufficientShares)
}

func (suite *DelegationTestSuite) TestCalculateSlashShare() {
	suite.basicPrepare()
	suite.prepareDeposit(suite.depositAmount)
	suite.prepareDelegation(suite.delegationAmount, suite.opAccAddr)
	stakerID, assetID := assetstype.GetStakerIDAndAssetID(suite.clientChainLzID, suite.Address[:], suite.assetAddr[:])
	slashAmount := sdkmath.ZeroInt()
	_, err := suite.App.DelegationKeeper.CalculateSlashShare(suite.Ctx, suite.opAccAddr, stakerID, assetID, slashAmount)
	suite.Error(err, delegationtypes.ErrAmountIsNotPositive)

	slashAmount = sdkmath.NewInt(10)
	slashShare, err := suite.App.DelegationKeeper.CalculateSlashShare(suite.Ctx, suite.opAccAddr, stakerID, assetID, slashAmount)
	suite.NoError(err)
	suite.Equal(sdkmath.LegacyNewDecFromBigInt(slashAmount.BigInt()), slashShare)

	// test the slashAmount is greater than the delegated amount
	slashAmount = suite.delegationAmount.Add(sdkmath.NewInt(1))
	slashShare, err = suite.App.DelegationKeeper.CalculateSlashShare(suite.Ctx, suite.opAccAddr, stakerID, assetID, slashAmount)
	suite.NoError(err)
	suite.Equal(sdkmath.LegacyNewDecFromBigInt(suite.delegationAmount.BigInt()), slashShare)
}

func (suite *DelegationTestSuite) TestRemoveShareFromOperator() {
	suite.basicPrepare()
	suite.prepareDeposit(suite.depositAmount)
	suite.prepareDelegation(suite.delegationAmount, suite.opAccAddr)
	stakerID, assetID := assetstype.GetStakerIDAndAssetID(suite.clientChainLzID, suite.Address[:], suite.assetAddr[:])
	originalInfo, err := suite.App.AssetsKeeper.GetOperatorSpecifiedAssetInfo(suite.Ctx, suite.opAccAddr, assetID)
	suite.NoError(err)

	// test removing share for slash
	removeShareForSlash := sdkmath.LegacyMustNewDecFromStr("10.1")
	amount := removeShareForSlash.TruncateInt()
	assetAmount, err := suite.App.DelegationKeeper.RemoveShareFromOperator(suite.Ctx, false, suite.opAccAddr, stakerID, assetID, removeShareForSlash)
	suite.NoError(err)
	suite.Equal(amount, assetAmount)

	info, err := suite.App.AssetsKeeper.GetOperatorSpecifiedAssetInfo(suite.Ctx, suite.opAccAddr, assetID)
	suite.NoError(err)
	expectedInfo := *originalInfo
	expectedInfo.TotalAmount = originalInfo.TotalAmount.Sub(amount)
	expectedInfo.TotalShare = originalInfo.TotalShare.Sub(removeShareForSlash)
	suite.Equal(expectedInfo, *info)

	originalInfo = info
	// test removing share for undelegation
	removeShareForUndelegation := sdkmath.LegacyMustNewDecFromStr("5.5")
	amount = removeShareForUndelegation.TruncateInt()
	assetAmount, err = suite.App.DelegationKeeper.RemoveShareFromOperator(suite.Ctx, true, suite.opAccAddr, stakerID, assetID, removeShareForUndelegation)
	suite.NoError(err)
	suite.Equal(amount, assetAmount)

	info, err = suite.App.AssetsKeeper.GetOperatorSpecifiedAssetInfo(suite.Ctx, suite.opAccAddr, assetID)
	suite.NoError(err)
	expectedInfo = *originalInfo
	expectedInfo.TotalAmount = originalInfo.TotalAmount.Sub(amount)
	expectedInfo.TotalShare = originalInfo.TotalShare.Sub(removeShareForUndelegation)
	expectedInfo.PendingUndelegationAmount = originalInfo.PendingUndelegationAmount.Add(amount)
	suite.Equal(expectedInfo, *info)
}

func (suite *DelegationTestSuite) TestRemoveShare() {
	suite.basicPrepare()
	suite.prepareDeposit(suite.depositAmount)
	suite.prepareDelegation(suite.delegationAmount, suite.opAccAddr)
	stakerID, assetID := assetstype.GetStakerIDAndAssetID(suite.clientChainLzID, suite.Address[:], suite.assetAddr[:])
	removeShare := sdkmath.LegacyNewDec(10)
	removeToken, err := suite.App.DelegationKeeper.RemoveShare(suite.Ctx, false, suite.opAccAddr, stakerID, assetID, removeShare)
	suite.NoError(err)
	suite.Equal(removeShare.TruncateInt(), removeToken)
	delegationInfo, err := suite.App.DelegationKeeper.GetSingleDelegationInfo(suite.Ctx, stakerID, assetID, suite.opAccAddr.String())
	suite.NoError(err)
	remainShare := sdkmath.LegacyNewDecFromBigInt(suite.delegationAmount.BigInt()).Sub(removeShare)
	suite.Equal(remainShare, delegationInfo.UndelegatableShare)
	stakerMap, err := suite.App.DelegationKeeper.GetStakersByOperator(suite.Ctx, suite.opAccAddr.String(), assetID)
	suite.NoError(err)
	suite.Contains(stakerMap.Stakers, stakerID)

	removeShare = remainShare
	removeToken, err = suite.App.DelegationKeeper.RemoveShare(suite.Ctx, true, suite.opAccAddr, stakerID, assetID, removeShare)
	suite.NoError(err)
	suite.Equal(removeShare.TruncateInt(), removeToken)
	delegationInfo, err = suite.App.DelegationKeeper.GetSingleDelegationInfo(suite.Ctx, stakerID, assetID, suite.opAccAddr.String())
	suite.NoError(err)
	suite.Equal(sdkmath.LegacyZeroDec(), delegationInfo.UndelegatableShare)
	suite.Equal(removeShare.TruncateInt(), delegationInfo.WaitUndelegationAmount)
	stakerAssetInfo, err := suite.App.AssetsKeeper.GetStakerSpecifiedAssetInfo(suite.Ctx, stakerID, assetID)
	suite.NoError(err)
	suite.Equal(removeShare.TruncateInt(), stakerAssetInfo.PendingUndelegationAmount)
	stakerMap, err = suite.App.DelegationKeeper.GetStakersByOperator(suite.Ctx, suite.opAccAddr.String(), assetID)
	suite.NoError(err)
	suite.NotContains(stakerMap.Stakers, stakerID)
}
