package keeper_test

import (
	"time"

	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	sdkmath "cosmossdk.io/math"

	utiltx "github.com/ExocoreNetwork/exocore/testutil/tx"
	keytypes "github.com/ExocoreNetwork/exocore/types/keys"
	assetskeeper "github.com/ExocoreNetwork/exocore/x/assets/keeper"
	assetstypes "github.com/ExocoreNetwork/exocore/x/assets/types"
	avstypes "github.com/ExocoreNetwork/exocore/x/avs/types"
	delegationtypes "github.com/ExocoreNetwork/exocore/x/delegation/types"
	operatorkeeper "github.com/ExocoreNetwork/exocore/x/operator/keeper"
	operatortypes "github.com/ExocoreNetwork/exocore/x/operator/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
)

func (suite *KeeperTestSuite) TestEpochHooks() {
	suite.prepare()
	epoch, _ := suite.App.EpochsKeeper.GetEpochInfo(suite.Ctx, suite.App.StakingKeeper.GetEpochIdentifier(suite.Ctx))
	currentEpoch := epoch.CurrentEpoch
	suite.Assert().Equal(currentEpoch, epoch.CurrentEpoch)

	epsilon := time.Nanosecond // negligible amount of buffer duration
	suite.Commit()
	suite.CommitAfter(time.Hour*24 + epsilon - time.Minute)
	allValidators := suite.App.StakingKeeper.GetAllExocoreValidators(suite.Ctx) // GetAllValidators(suite.Ctx)
	for i, val := range allValidators {
		pk, err := val.ConsPubKey()
		if err != nil {
			suite.Ctx.Logger().Error("Failed to deserialize public key; skipping", "error", err, "i", i)
			continue
		}
		validatorDetail, found := suite.App.StakingKeeper.ValidatorByConsAddrForChainID(
			suite.Ctx, sdk.GetConsAddress(pk), avstypes.ChainIDWithoutRevision(suite.Ctx.ChainID()),
		)
		if !found {
			suite.Ctx.Logger().Error("Operator address not found; skipping", "consAddress", sdk.GetConsAddress(pk), "i", i)
			continue
		}
		valBz := validatorDetail.GetOperator()
		currentRewards := suite.App.DistrKeeper.GetValidatorOutstandingRewards(suite.Ctx, valBz)
		suite.Require().NotNil(currentRewards)
	}
}

func (suite *KeeperTestSuite) prepare() {
	// registration and associated checks
	operatorAddress := sdk.AccAddress(utiltx.GenerateAddress().Bytes())
	operatorAddressString := operatorAddress.String()
	registerReq := &operatortypes.RegisterOperatorReq{
		FromAddress: operatorAddressString,
		Info: &operatortypes.OperatorInfo{
			EarningsAddr: operatorAddressString,
			ApproveAddr:  operatorAddressString,
			Commission: stakingtypes.Commission{
				CommissionRates: stakingtypes.CommissionRates{
					Rate:          sdk.NewDec(1),
					MaxRate:       sdk.NewDec(1),
					MaxChangeRate: sdk.NewDec(1),
				},
			},
		},
	}
	_, err := suite.OperatorMsgServer.RegisterOperator(sdk.WrapSDKContext(suite.Ctx), registerReq)
	suite.NoError(err)
	suite.CheckLengthOfValidatorUpdates(0, nil, "register operator but don't opt in")

	// opt-in with a key
	chainIDWithoutRevision := avstypes.ChainIDWithoutRevision(suite.Ctx.ChainID())
	_, avsAddress := suite.App.AVSManagerKeeper.IsAVSByChainID(suite.Ctx, chainIDWithoutRevision)
	key := utiltx.GenerateConsensusKey()
	_, err = suite.OperatorMsgServer.OptIntoAVS(sdk.WrapSDKContext(suite.Ctx), &operatortypes.OptIntoAVSReq{
		FromAddress:   operatorAddressString,
		AvsAddress:    avsAddress,
		PublicKeyJSON: key.ToJSON(),
	})
	suite.Error(err)
	// opted in but not enough self-delegation
	suite.CheckLengthOfValidatorUpdates(0, nil, "opt in but no delegation")

	// now make a deposit slightly below the min self delegation
	staker := utiltx.GenerateAddress()
	minSelfDelegation := suite.App.StakingKeeper.GetMinSelfDelegation(suite.Ctx)
	// figure out decimals
	lzID := suite.ClientChains[0].LayerZeroChainID
	assetAddrHex := suite.Assets[0].Address
	assetAddr := common.HexToAddress(assetAddrHex)
	_, assetID := assetstypes.GetStakerIDAndAssetIDFromStr(lzID, staker.String(), assetAddrHex)
	asset, err := suite.App.AssetsKeeper.GetStakingAssetInfo(suite.Ctx, assetID)
	suite.NoError(err)
	assetDecimals := asset.AssetBasicInfo.Decimals
	amount := sdkmath.NewIntWithDecimal(
		minSelfDelegation.Int64(), int(assetDecimals),
	).Sub(sdkmath.NewInt(1))
	depositParams := &assetskeeper.DepositWithdrawParams{
		ClientChainLzID: lzID,
		Action:          assetstypes.DepositLST,
		StakerAddress:   staker.Bytes(),
		AssetsAddress:   assetAddr.Bytes(),
		OpAmount:        amount,
	}
	_, err = suite.App.AssetsKeeper.PerformDepositOrWithdraw(suite.Ctx, depositParams)
	suite.NoError(err)
	suite.CheckLengthOfValidatorUpdates(0, nil, "deposit but don't delegate")
	// then delegate it
	delegationParams := &delegationtypes.DelegationOrUndelegationParams{
		ClientChainID:   lzID,
		AssetsAddress:   assetAddr.Bytes(),
		StakerAddress:   staker.Bytes(),
		OperatorAddress: operatorAddress,
		OpAmount:        amount,
	}
	err = suite.App.DelegationKeeper.DelegateTo(suite.Ctx, delegationParams)
	suite.NoError(err)
	suite.CheckLengthOfValidatorUpdates(0, nil, "delegate but not self delegate")
	// mark it as self delegation
	err = suite.App.DelegationKeeper.AssociateOperatorWithStaker(
		suite.Ctx, lzID, operatorAddress, staker.Bytes(),
	)
	suite.NoError(err)
	suite.CheckLengthOfValidatorUpdates(0, nil, "self delegate but below min")

	// go above the minimum - first, deposit only
	additionalAmount := sdkmath.NewIntWithDecimal(2, int(assetDecimals))
	depositParams = &assetskeeper.DepositWithdrawParams{
		ClientChainLzID: lzID,
		Action:          assetstypes.DepositLST,
		StakerAddress:   staker.Bytes(),
		AssetsAddress:   assetAddr.Bytes(),
		OpAmount:        additionalAmount,
	}
	_, err = suite.App.AssetsKeeper.PerformDepositOrWithdraw(suite.Ctx, depositParams)
	suite.NoError(err)
	suite.CheckLengthOfValidatorUpdates(0, nil, "deposit above min but don't delegate")
	delegationParams = &delegationtypes.DelegationOrUndelegationParams{
		ClientChainID:   lzID,
		AssetsAddress:   assetAddr.Bytes(),
		StakerAddress:   staker.Bytes(),
		OperatorAddress: operatorAddress,
		OpAmount:        depositParams.OpAmount,
	}
	err = suite.App.DelegationKeeper.DelegateTo(suite.Ctx, delegationParams)
	suite.NoError(err)
	totalAmount := amount.Add(additionalAmount)
	totalAmountInUSD := operatorkeeper.CalculateUSDValue(
		totalAmount,
		sdkmath.NewInt(1), // asset price
		assetDecimals,
		0, // price decimals
	)
	// Opt in again once the self-delegation requirement is met.
	_, err = suite.OperatorMsgServer.OptIntoAVS(sdk.WrapSDKContext(suite.Ctx), &operatortypes.OptIntoAVSReq{
		FromAddress:   operatorAddressString,
		AvsAddress:    avsAddress,
		PublicKeyJSON: key.ToJSON(),
	})
	suite.NoError(err)
	suite.CheckLengthOfValidatorUpdates(
		1, []int64{totalAmountInUSD.TruncateInt64()}, "delegate above min",
	)
	epoch, _ := suite.App.EpochsKeeper.GetEpochInfo(suite.Ctx, suite.App.StakingKeeper.GetEpochIdentifier(suite.Ctx))
	currentEpoch := epoch.CurrentEpoch
	suite.Assert().Equal(currentEpoch, epoch.CurrentEpoch)
	suite.CheckValidatorFound(key, true, chainIDWithoutRevision, operatorAddress)
}

func (suite *KeeperTestSuite) CheckValidatorFound(
	key keytypes.WrappedConsKey, expected bool,
	chainIDWithoutRevision string,
	operatorAddress sdk.AccAddress,
) {
	validator, found := suite.App.OperatorKeeper.ValidatorByConsAddrForChainID(
		suite.Ctx, key.ToConsAddr(), chainIDWithoutRevision,
	)
	suite.Equal(expected, found, "validator found by key")
	if expected && found {
		suite.Equal(
			sdk.ValAddress(operatorAddress).String(),
			validator.OperatorAddress,
			"ValAddress derived from AccAddress",
		)
	}
}

func (suite *KeeperTestSuite) CheckLengthOfValidatorUpdates(
	expected int, powers []int64, msgAndArgs ...interface{},
) {
	suite.Require().Equal(len(powers), expected, "unequal `expected` and `powers` length")
	// we commit one block and one epoch, to check after both
	suite.Commit()
	// at one block, no change
	updates := suite.App.StakingKeeper.GetValidatorUpdates(suite.Ctx)
	suite.Assert().Equal(0, len(updates), msgAndArgs)
	// go forward 1 epoch in time
	epoch, _ := suite.App.EpochsKeeper.GetEpochInfo(suite.Ctx, suite.App.StakingKeeper.GetEpochIdentifier(suite.Ctx))
	currentEpoch := epoch.CurrentEpoch
	suite.CommitAfter(suite.EpochDuration)
	epoch, _ = suite.App.EpochsKeeper.GetEpochInfo(suite.Ctx, suite.App.StakingKeeper.GetEpochIdentifier(suite.Ctx))
	// and validate that the epoch has changed
	suite.Assert().Equal(currentEpoch+1, epoch.CurrentEpoch, msgAndArgs)
	// the epoch hook is called during BeginBlock, and it updates the
	// validator set during EndBlock.
	// ideally, the updates below should be the expected length, however,
	// Commit runs the current EndBlocker and the following BeginBlocker
	// and leaves you in the middle of the 2 blocks. hence, we need to add
	// one more block to get the correct updates.
	updates = suite.App.StakingKeeper.GetValidatorUpdates(suite.Ctx)
	suite.Assert().Equal(0, len(updates), msgAndArgs)
	// add one block
	suite.Commit()
	updates = suite.App.StakingKeeper.GetValidatorUpdates(suite.Ctx)
	suite.Assert().Equal(expected, len(updates), msgAndArgs)
	for i := 0; i < expected; i++ {
		suite.Assert().Equal(
			powers[i], updates[i].Power, msgAndArgs...,
		)
	}
}
