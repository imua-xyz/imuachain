package keeper_test

import (
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

func (suite *KeeperTestSuite) TestBasicOperations() {
	// registration and associated checks
	operatorAddress := sdk.AccAddress(utiltx.GenerateAddress().Bytes())
	operatorAddressString := operatorAddress.String()
	registerReq := &operatortypes.RegisterOperatorReq{
		FromAddress: operatorAddressString,
		Info: &operatortypes.OperatorInfo{
			EarningsAddr: operatorAddressString,
		},
	}
	_, err := suite.OperatorMsgServer.RegisterOperator(sdk.WrapSDKContext(suite.Ctx), registerReq)
	suite.NoError(err)
	suite.CheckLengthOfValidatorUpdates(0, nil, "register operator but don't opt in")

	// opt-in with a key - it will fail because there is not enough self delegation to opt-in
	chainIDWithoutRevision := avstypes.ChainIDWithoutRevision(suite.Ctx.ChainID())
	found, avsAddress := suite.App.AVSManagerKeeper.IsAVSByChainID(suite.Ctx, chainIDWithoutRevision)
	suite.True(found, "AVS not found")
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
	err = suite.App.AssetsKeeper.PerformDepositOrWithdraw(suite.Ctx, depositParams)
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
	err = suite.App.AssetsKeeper.PerformDepositOrWithdraw(suite.Ctx, depositParams)
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
	// opt in successfully after the self delegation meet the requirement
	_, err = suite.OperatorMsgServer.OptIntoAVS(sdk.WrapSDKContext(suite.Ctx), &operatortypes.OptIntoAVSReq{
		FromAddress:   operatorAddressString,
		AvsAddress:    avsAddress,
		PublicKeyJSON: key.ToJSON(),
	})
	suite.NoError(err)
	suite.CheckLengthOfValidatorUpdates(
		1, []int64{totalAmountInUSD.TruncateInt64()}, "delegate above min",
	)

	// now, deposit and delegate using another address that is not associated with the operator
	amount = sdkmath.NewIntWithDecimal(1, int(assetDecimals))
	staker = utiltx.GenerateAddress()
	depositParams = &assetskeeper.DepositWithdrawParams{
		ClientChainLzID: lzID,
		Action:          assetstypes.DepositLST,
		StakerAddress:   staker.Bytes(),
		AssetsAddress:   assetAddr.Bytes(),
		OpAmount:        amount,
	}
	err = suite.App.AssetsKeeper.PerformDepositOrWithdraw(suite.Ctx, depositParams)
	suite.NoError(err)
	suite.CheckLengthOfValidatorUpdates(0, nil, "deposit (non-self) but don't delegate")
	delegationParams = &delegationtypes.DelegationOrUndelegationParams{
		ClientChainID:   lzID,
		AssetsAddress:   assetAddr.Bytes(),
		StakerAddress:   staker.Bytes(),
		OperatorAddress: operatorAddress,
		OpAmount:        amount,
	}
	err = suite.App.DelegationKeeper.DelegateTo(suite.Ctx, delegationParams)
	suite.NoError(err)
	totalAmount = totalAmount.Add(amount)
	totalAmountInUSD = operatorkeeper.CalculateUSDValue(
		totalAmount,
		sdkmath.NewInt(1), // asset price
		assetDecimals,
		0, // price decimals
	)
	suite.CheckLengthOfValidatorUpdates(
		1, []int64{totalAmountInUSD.TruncateInt64()}, "delegate (non-self)",
	)

	// then, check if 2 delegations within an epoch to the same operator work fine
	amount = sdkmath.NewIntWithDecimal(5000, int(assetDecimals))
	for i := 0; i < 2; i++ {
		staker := utiltx.GenerateAddress()
		depositParams = &assetskeeper.DepositWithdrawParams{
			ClientChainLzID: lzID,
			Action:          assetstypes.DepositLST,
			StakerAddress:   staker.Bytes(),
			AssetsAddress:   assetAddr.Bytes(),
			OpAmount:        amount,
		}
		err = suite.App.AssetsKeeper.PerformDepositOrWithdraw(suite.Ctx, depositParams)
		suite.NoError(err)
		delegationParams = &delegationtypes.DelegationOrUndelegationParams{
			ClientChainID:   lzID,
			AssetsAddress:   assetAddr.Bytes(),
			StakerAddress:   staker.Bytes(),
			OperatorAddress: operatorAddress,
			OpAmount:        amount,
		}
		err = suite.App.DelegationKeeper.DelegateTo(suite.Ctx, delegationParams)
		suite.NoError(err)
		totalAmount = totalAmount.Add(amount)
	}
	totalAmountInUSD = operatorkeeper.CalculateUSDValue(
		totalAmount,
		sdkmath.NewInt(1), // asset price
		assetDecimals,
		0, // price decimals
	)
	suite.CheckLengthOfValidatorUpdates(
		1, []int64{totalAmountInUSD.TruncateInt64()}, "delegate 2 stakers to the same operator",
	)

	// next, do 2 delegations to 2 different operators
	additionalPower := int64(5000)
	amount = sdkmath.NewIntWithDecimal(additionalPower, int(assetDecimals))
	for i := 0; i < 2; i++ {
		staker := utiltx.GenerateAddress()
		depositParams = &assetskeeper.DepositWithdrawParams{
			ClientChainLzID: lzID,
			Action:          assetstypes.DepositLST,
			StakerAddress:   staker.Bytes(),
			AssetsAddress:   assetAddr.Bytes(),
			OpAmount:        amount,
		}
		err = suite.App.AssetsKeeper.PerformDepositOrWithdraw(suite.Ctx, depositParams)
		suite.NoError(err)
		delegationParams = &delegationtypes.DelegationOrUndelegationParams{
			ClientChainID:   lzID,
			AssetsAddress:   assetAddr.Bytes(),
			StakerAddress:   staker.Bytes(),
			OperatorAddress: suite.Operators[i],
			OpAmount:        amount,
		}
		err = suite.App.DelegationKeeper.DelegateTo(suite.Ctx, delegationParams)
		suite.NoError(err)
	}
	suite.CheckLengthOfValidatorUpdates(
		2, []int64{
			suite.Powers[0] + additionalPower,
			suite.Powers[1] + additionalPower,
		}, "delegate 2 stakers to 2 operators",
	)

	// we will now test the key replacement case
	newKey := utiltx.GenerateConsensusKey()
	suite.CheckValidatorFound(key, true, chainIDWithoutRevision, operatorAddress)
	suite.CheckValidatorFound(newKey, false, chainIDWithoutRevision, operatorAddress)
	_, err = suite.OperatorMsgServer.SetConsKey(
		sdk.WrapSDKContext(suite.Ctx),
		&operatortypes.SetConsKeyReq{
			Address:       operatorAddressString,
			PublicKeyJSON: newKey.ToJSON(),
			AvsAddress:    avsAddress,
		},
	)
	suite.NoError(err)
	epochsUntilUnbonded := suite.App.StakingKeeper.GetEpochsUntilUnbonded(suite.Ctx)
	currentEpochInfo, _ := suite.App.EpochsKeeper.GetEpochInfo(
		suite.Ctx,
		suite.App.StakingKeeper.GetEpochIdentifier(suite.Ctx),
	)
	unbondingEpoch := suite.App.StakingKeeper.GetUnbondingCompletionEpoch(suite.Ctx)
	suite.Equal(
		currentEpochInfo.CurrentEpoch+int64(epochsUntilUnbonded),
		unbondingEpoch,
		"unbonding epoch",
	)
	// before committing, verify that the validator can still be found from\
	// its old consensus address
	suite.CheckValidatorFound(key, true, chainIDWithoutRevision, operatorAddress)
	suite.CheckValidatorFound(newKey, true, chainIDWithoutRevision, operatorAddress)
	// after committing but before the epoch ends, check that the validator
	// can be fetched by the old consensus address
	suite.Commit()
	suite.CheckValidatorFound(key, true, chainIDWithoutRevision, operatorAddress)
	suite.CheckValidatorFound(newKey, true, chainIDWithoutRevision, operatorAddress)
	// it will have two updates - one for the old key to 0 and one for the new key
	suite.CheckLengthOfValidatorUpdates(2, []int64{totalAmountInUSD.TruncateInt64(), 0}, "key replacement")
	// check prune data
	oldConsAddr := key.ToConsAddr()
	toPrune := suite.App.StakingKeeper.GetConsensusAddrsToPrune(suite.Ctx, unbondingEpoch)
	suite.Equal(1, len(toPrune), "consensus addresses count to prune")
	suite.Equal(oldConsAddr.Bytes(), toPrune[0], "consensus address value to prune")

	// now we go forward some epochs to check if it is pruned
	// we have already gone ahead one epoch in the CheckLengthOfValidatorUpdates call
	// say currentEpochInfo.CurrentEpoch is 1 and epochsUntilUnbonded is 7
	// the new key goes into effect at the end of epoch 1 (beginning of epoch 2)
	// it should be cleared at epoch the end of epoch 8 (beginning of epoch 9)
	// when starting this loop, we will start at epoch 2 and each CommitAfter
	// will add one more epoch.
	for i := 1; i <= int(epochsUntilUnbonded); i++ {
		// after commitment, the validator should be found both by oldKey and newKey
		// until pruning of the old key is done at the request of this module.
		suite.CheckValidatorFound(key, true, chainIDWithoutRevision, operatorAddress)
		suite.CheckValidatorFound(newKey, true, chainIDWithoutRevision, operatorAddress)
		// end the epoch   2, 3, 4, 5, 6, 7, 8
		// start the epoch 3, 4, 5, 6, 7, 8, 9
		suite.CommitAfter(suite.EpochDuration)
		// add the next block; its EndBlocker is where the magic happens
		suite.Commit()
	}
	suite.CheckValidatorFound(key, false, chainIDWithoutRevision, operatorAddress)
	suite.CheckValidatorFound(newKey, true, chainIDWithoutRevision, operatorAddress)
	toPrune = suite.App.StakingKeeper.GetConsensusAddrsToPrune(suite.Ctx, unbondingEpoch)
	suite.Equal(0, len(toPrune), "consensus addresses count to prune")

	// now test opt out
	_, err = suite.OperatorMsgServer.OptOutOfAVS(
		sdk.WrapSDKContext(suite.Ctx),
		&operatortypes.OptOutOfAVSReq{
			FromAddress: operatorAddressString,
			AvsAddress:  avsAddress,
		},
	)
	suite.NoError(err)
	suite.CheckValidatorFound(newKey, true, chainIDWithoutRevision, operatorAddress)
	suite.CheckLengthOfValidatorUpdates(1, []int64{0}, "opt out")
	// after the opt out unbonding is complete, we can't fetch the validator object
	for i := 1; i <= int(epochsUntilUnbonded); i++ {
		suite.CheckValidatorFound(newKey, true, chainIDWithoutRevision, operatorAddress)
		suite.CommitAfter(suite.EpochDuration)
		suite.Commit()
	}
	suite.CheckValidatorFound(newKey, false, chainIDWithoutRevision, operatorAddress)
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
