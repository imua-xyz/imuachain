package keeper_test

import (
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	utiltx "github.com/imua-xyz/imuachain/testutil/tx"
	assetskeeper "github.com/imua-xyz/imuachain/x/assets/keeper"
	assetstypes "github.com/imua-xyz/imuachain/x/assets/types"
	avstypes "github.com/imua-xyz/imuachain/x/avs/types"
	delegationtypes "github.com/imua-xyz/imuachain/x/delegation/types"
	operatortypes "github.com/imua-xyz/imuachain/x/operator/types"
)

func (suite *KeeperTestSuite) TestUndelegations() {
	operatorAddress := sdk.AccAddress(utiltx.GenerateAddress().Bytes())
	operatorAddressString := operatorAddress.String()
	amountUSD := suite.App.StakingKeeper.GetMinSelfDelegation(suite.Ctx).Int64() * 5
	// register operator
	registerReq := &operatortypes.RegisterOperatorReq{
		FromAddress: operatorAddressString,
		Info: &operatortypes.OperatorInfo{
			EarningsAddr: operatorAddressString,
			ApproveAddr:  operatorAddressString,
		},
	}
	_, err := suite.OperatorMsgServer.RegisterOperator(
		sdk.WrapSDKContext(suite.Ctx), registerReq,
	)
	suite.NoError(err)
	// make deposit
	staker := utiltx.GenerateAddress()
	lzID := suite.ClientChains[0].LayerZeroChainID
	assetAddrHex := suite.Assets[0].Address
	assetAddr := common.HexToAddress(assetAddrHex)
	_, assetID := assetstypes.GetStakerIDAndAssetIDFromStr(lzID, staker.String(), assetAddrHex)
	asset, err := suite.App.AssetsKeeper.GetStakingAssetInfo(suite.Ctx, assetID)
	suite.NoError(err)
	assetDecimals := asset.AssetBasicInfo.Decimals
	amount := sdkmath.NewIntWithDecimal(
		amountUSD,
		int(assetDecimals),
	)
	depositParams := &assetskeeper.DepositWithdrawParams{
		ClientChainLzID: lzID,
		Action:          assetstypes.DepositLST,
		StakerAddress:   staker.Bytes(),
		AssetsAddress:   assetAddr.Bytes(),
		OpAmount:        amount,
	}
	_, err = suite.App.AssetsKeeper.PerformDepositOrWithdraw(suite.Ctx, depositParams)
	suite.NoError(err)
	// delegate
	delegationParams := &delegationtypes.DelegationOrUndelegationParams{
		ClientChainID:   lzID,
		AssetsAddress:   assetAddr.Bytes(),
		StakerAddress:   staker.Bytes(),
		OperatorAddress: operatorAddress,
		OpAmount:        amount,
	}
	err = suite.App.DelegationKeeper.DelegateTo(suite.Ctx, delegationParams)
	suite.NoError(err)
	// self delegate
	err = suite.App.DelegationKeeper.AssociateOperatorWithStaker(
		suite.Ctx, lzID, operatorAddress, staker.Bytes(),
	)
	suite.NoError(err)
	// opt in
	oldKey := utiltx.GenerateConsensusKey()
	chainIDWithoutRevision := avstypes.ChainIDWithoutRevision(suite.Ctx.ChainID())
	_, avsAddress := suite.App.AVSManagerKeeper.IsAVSByChainID(suite.Ctx, chainIDWithoutRevision)
	_, err = suite.OperatorMsgServer.OptIntoAVS(
		sdk.WrapSDKContext(suite.Ctx),
		&operatortypes.OptIntoAVSReq{
			FromAddress:   operatorAddressString,
			AvsAddress:    avsAddress,
			PublicKeyJSON: oldKey.ToJSON(),
		},
	)
	suite.NoError(err)
	suite.CheckLengthOfValidatorUpdates(1, []int64{amountUSD}, "opt in")
	// now undelegate 1/5
	initialUndelegationID := uint64(0)
	txHash := common.BytesToHash([]byte("txhash1")) // not validated
	undelegationParams := &delegationtypes.DelegationOrUndelegationParams{
		ClientChainID:   lzID,
		AssetsAddress:   assetAddr.Bytes(),
		StakerAddress:   staker.Bytes(),
		OperatorAddress: operatorAddress,
		OpAmount:        amount.Quo(sdkmath.NewInt(5)),
		TxHash:          txHash,
	}
	err = suite.App.DelegationKeeper.UndelegateFrom(suite.Ctx, undelegationParams)
	suite.NoError(err)
	recordKey := delegationtypes.GetUndelegationRecordKey(
		uint64(suite.Ctx.BlockHeight()), initialUndelegationID, txHash.String(), operatorAddressString,
	)
	suite.CheckLengthOfValidatorUpdates(1, []int64{amountUSD * 4 / 5}, "undelegate 1/5")
	// wait for it to be released
	epochsUntilUnbonded := suite.App.StakingKeeper.GetEpochsUntilUnbonded(suite.Ctx)
	for i := 0; i < int(epochsUntilUnbonded); i++ {
		suite.Equal(
			uint64(1), suite.App.DelegationKeeper.GetUndelegationHoldCount(suite.Ctx, recordKey),
		)
		suite.CheckLengthOfValidatorUpdates(0, nil, "moving forward one epoch")
	}
	suite.Equal(
		uint64(0), suite.App.DelegationKeeper.GetUndelegationHoldCount(suite.Ctx, recordKey),
	)

	// then opt out
	_, err = suite.OperatorMsgServer.OptOutOfAVS(
		sdk.WrapSDKContext(suite.Ctx),
		&operatortypes.OptOutOfAVSReq{
			FromAddress: operatorAddressString,
			AvsAddress:  avsAddress,
		},
	)
	suite.NoError(err)
	suite.CheckLengthOfValidatorUpdates(1, []int64{0}, "opt out")
	forwardEpochs := 2
	for i := 0; i < forwardEpochs; i++ {
		suite.CheckLengthOfValidatorUpdates(0, nil, "moving forward one epoch")
	}
	// undelegate the remainder, in an epoch different from the opt out epoch
	txHash = common.BytesToHash([]byte("txhash2")) // not validated
	undelegationParams = &delegationtypes.DelegationOrUndelegationParams{
		ClientChainID:   lzID,
		AssetsAddress:   assetAddr.Bytes(),
		StakerAddress:   staker.Bytes(),
		OperatorAddress: operatorAddress,
		OpAmount:        amount.Quo(sdkmath.NewInt(5)).Mul(sdkmath.NewInt(4)),
		TxHash:          txHash,
	}
	err = suite.App.DelegationKeeper.UndelegateFrom(suite.Ctx, undelegationParams)
	suite.NoError(err)
	recordKey = delegationtypes.GetUndelegationRecordKey(
		uint64(suite.Ctx.BlockHeight()), initialUndelegationID+1, txHash.String(), operatorAddressString,
	)
	// early release, based on the opt out epoch
	for i := 0; i < int(epochsUntilUnbonded)-forwardEpochs; i++ {
		suite.Equal(
			uint64(1), suite.App.DelegationKeeper.GetUndelegationHoldCount(suite.Ctx, recordKey),
		)
		suite.CheckLengthOfValidatorUpdates(0, nil, "moving forward one epoch")
	}
	suite.Equal(
		uint64(0), suite.App.DelegationKeeper.GetUndelegationHoldCount(suite.Ctx, recordKey),
	)
}

func (suite *KeeperTestSuite) TestUndelegationEdgeCases() {
	// register an operator and delegate to them, and then undelegate
	// no hold count should be set in that case, since they did not opt in yet
	operatorAddress := sdk.AccAddress(utiltx.GenerateAddress().Bytes())
	operatorAddressString := operatorAddress.String()
	registerReq := &operatortypes.RegisterOperatorReq{
		FromAddress: operatorAddressString,
		Info: &operatortypes.OperatorInfo{
			EarningsAddr: operatorAddressString,
			ApproveAddr:  operatorAddressString,
		},
	}
	_, err := suite.OperatorMsgServer.RegisterOperator(
		sdk.WrapSDKContext(suite.Ctx), registerReq,
	)
	suite.NoError(err)
	// make deposit
	staker := utiltx.GenerateAddress()
	lzID := suite.ClientChains[0].LayerZeroChainID
	assetAddrHex := suite.Assets[0].Address
	assetAddr := common.HexToAddress(assetAddrHex)
	_, assetID := assetstypes.GetStakerIDAndAssetIDFromStr(lzID, staker.String(), assetAddrHex)
	asset, err := suite.App.AssetsKeeper.GetStakingAssetInfo(suite.Ctx, assetID)
	suite.NoError(err)
	assetDecimals := asset.AssetBasicInfo.Decimals
	amountUSD := suite.App.StakingKeeper.GetMinSelfDelegation(suite.Ctx).Int64()
	amount := sdkmath.NewIntWithDecimal(
		amountUSD,
		int(assetDecimals),
	)
	depositParams := &assetskeeper.DepositWithdrawParams{
		ClientChainLzID: lzID,
		Action:          assetstypes.DepositLST,
		StakerAddress:   staker.Bytes(),
		AssetsAddress:   assetAddr.Bytes(),
		OpAmount:        amount.Mul(sdkmath.NewInt(5)),
	}
	_, err = suite.App.AssetsKeeper.PerformDepositOrWithdraw(suite.Ctx, depositParams)
	suite.NoError(err)
	// delegate
	delegationParams := &delegationtypes.DelegationOrUndelegationParams{
		ClientChainID:   lzID,
		AssetsAddress:   assetAddr.Bytes(),
		StakerAddress:   staker.Bytes(),
		OperatorAddress: operatorAddress,
		OpAmount:        depositParams.OpAmount,
	}
	err = suite.App.DelegationKeeper.DelegateTo(suite.Ctx, delegationParams)
	suite.NoError(err)
	// mark as self delegation
	err = suite.App.DelegationKeeper.AssociateOperatorWithStaker(
		suite.Ctx, lzID, operatorAddress, staker.Bytes(),
	)
	suite.NoError(err)
	suite.CheckLengthOfValidatorUpdates(0, []int64{}, "delegate without opt in")
	// undelegate 1/5
	txHash := common.BytesToHash([]byte("txhash1")) // not validated
	undelegationParams := &delegationtypes.DelegationOrUndelegationParams{
		ClientChainID:   lzID,
		AssetsAddress:   assetAddr.Bytes(),
		StakerAddress:   staker.Bytes(),
		OperatorAddress: operatorAddress,
		OpAmount:        amount,
		TxHash:          txHash,
	}
	initialUndelegationID := uint64(0)
	err = suite.App.DelegationKeeper.UndelegateFrom(suite.Ctx, undelegationParams)
	suite.NoError(err)
	recordKey := delegationtypes.GetUndelegationRecordKey(
		uint64(suite.Ctx.BlockHeight()), initialUndelegationID, txHash.String(), operatorAddressString,
	)
	suite.Equal(
		uint64(0), suite.App.DelegationKeeper.GetUndelegationHoldCount(suite.Ctx, recordKey),
	)
	suite.CheckLengthOfValidatorUpdates(0, []int64{}, "undelegate without opt in")
	// opt in
	oldKey := utiltx.GenerateConsensusKey()
	chainIDWithoutRevision := avstypes.ChainIDWithoutRevision(suite.Ctx.ChainID())
	_, avsAddress := suite.App.AVSManagerKeeper.IsAVSByChainID(suite.Ctx, chainIDWithoutRevision)
	_, err = suite.OperatorMsgServer.OptIntoAVS(
		sdk.WrapSDKContext(suite.Ctx),
		&operatortypes.OptIntoAVSReq{
			FromAddress:   operatorAddressString,
			AvsAddress:    avsAddress,
			PublicKeyJSON: oldKey.ToJSON(),
		},
	)
	suite.NoError(err)
	// undelegate some before the key is active
	txHash = common.BytesToHash([]byte("txhash2")) // not validated
	undelegationParams = &delegationtypes.DelegationOrUndelegationParams{
		ClientChainID:   lzID,
		AssetsAddress:   assetAddr.Bytes(),
		StakerAddress:   staker.Bytes(),
		OperatorAddress: operatorAddress,
		OpAmount:        amount,
		TxHash:          txHash,
	}
	err = suite.App.DelegationKeeper.UndelegateFrom(suite.Ctx, undelegationParams)
	suite.NoError(err)
	recordKey = delegationtypes.GetUndelegationRecordKey(
		uint64(suite.Ctx.BlockHeight()), initialUndelegationID+1, txHash.String(), operatorAddressString,
	)
	suite.Equal(
		uint64(0), suite.App.DelegationKeeper.GetUndelegationHoldCount(suite.Ctx, recordKey),
	)
	suite.CheckLengthOfValidatorUpdates(1, []int64{amountUSD * 3}, "opt in")
	// then undelegate after the key is active
	txHash = common.BytesToHash([]byte("txhash3")) // not validated
	undelegationParams = &delegationtypes.DelegationOrUndelegationParams{
		ClientChainID:   lzID,
		AssetsAddress:   assetAddr.Bytes(),
		StakerAddress:   staker.Bytes(),
		OperatorAddress: operatorAddress,
		OpAmount:        amount,
		TxHash:          txHash,
	}
	err = suite.App.DelegationKeeper.UndelegateFrom(suite.Ctx, undelegationParams)
	suite.NoError(err)
	recordKey = delegationtypes.GetUndelegationRecordKey(
		uint64(suite.Ctx.BlockHeight()), initialUndelegationID+2, txHash.String(), operatorAddressString,
	)
	suite.Equal(
		uint64(1), suite.App.DelegationKeeper.GetUndelegationHoldCount(suite.Ctx, recordKey),
	)
	suite.CheckLengthOfValidatorUpdates(1, []int64{amountUSD * 2}, "undelegate")
	// replace the key
	newKey := utiltx.GenerateConsensusKey()
	_, err = suite.OperatorMsgServer.SetConsKey(
		sdk.WrapSDKContext(suite.Ctx),
		&operatortypes.SetConsKeyReq{
			Address:       operatorAddressString,
			AvsAddress:    avsAddress,
			PublicKeyJSON: newKey.ToJSON(),
		},
	)
	suite.NoError(err)
	// undelegate now such that the key replacement edge case is triggered
	txHash = common.BytesToHash([]byte("txhash4")) // not validated
	undelegationParams = &delegationtypes.DelegationOrUndelegationParams{
		ClientChainID:   lzID,
		AssetsAddress:   assetAddr.Bytes(),
		StakerAddress:   staker.Bytes(),
		OperatorAddress: operatorAddress,
		OpAmount:        amount,
		TxHash:          txHash,
	}
	err = suite.App.DelegationKeeper.UndelegateFrom(suite.Ctx, undelegationParams)
	suite.NoError(err)
	recordKey = delegationtypes.GetUndelegationRecordKey(
		uint64(suite.Ctx.BlockHeight()), initialUndelegationID+3, txHash.String(), operatorAddressString,
	)
	suite.Equal(
		uint64(1), suite.App.DelegationKeeper.GetUndelegationHoldCount(suite.Ctx, recordKey),
	)
	suite.CheckLengthOfValidatorUpdates(2, []int64{amountUSD * 1, 0}, "replace key")
}
