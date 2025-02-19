package keeper_test

import (
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	utiltx "github.com/imua-xyz/imuachain/testutil/tx"
	keytypes "github.com/imua-xyz/imuachain/types/keys"
	assetskeeper "github.com/imua-xyz/imuachain/x/assets/keeper"
	assetstypes "github.com/imua-xyz/imuachain/x/assets/types"
	avstypes "github.com/imua-xyz/imuachain/x/avs/types"
	delegationtypes "github.com/imua-xyz/imuachain/x/delegation/types"
	operatortypes "github.com/imua-xyz/imuachain/x/operator/types"
)

func (suite *KeeperTestSuite) TestSameEpochOperations() {
	// generate addresses and register operators
	operatorAddress := sdk.AccAddress(utiltx.GenerateAddress().Bytes())
	operatorAddressString := operatorAddress.String()
	amountUSD := suite.App.StakingKeeper.GetMinSelfDelegation(suite.Ctx).Int64()
	setUp := func() {
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
	}
	// generate keys, and get the AVS address
	oldKey := utiltx.GenerateConsensusKey()
	newKey := utiltx.GenerateConsensusKey()
	chainIDWithoutRevision := avstypes.ChainIDWithoutRevision(suite.Ctx.ChainID())
	_, avsAddress := suite.App.AVSManagerKeeper.IsAVSByChainID(suite.Ctx, chainIDWithoutRevision)

	// now define the operations
	type funcThatReturnsError func() error
	optIn := funcThatReturnsError(func() error {
		_, err := suite.OperatorMsgServer.OptIntoAVS(
			sdk.WrapSDKContext(suite.Ctx),
			&operatortypes.OptIntoAVSReq{
				FromAddress:   operatorAddressString,
				AvsAddress:    avsAddress,
				PublicKeyJSON: oldKey.ToJSON(),
			},
		)
		return err
	})
	optOut := funcThatReturnsError(func() error {
		_, err := suite.OperatorMsgServer.OptOutOfAVS(
			sdk.WrapSDKContext(suite.Ctx),
			&operatortypes.OptOutOfAVSReq{
				FromAddress: operatorAddressString,
				AvsAddress:  avsAddress,
			},
		)
		return err
	})
	setKey := funcThatReturnsError(func() error {
		_, err := suite.OperatorMsgServer.SetConsKey(
			sdk.WrapSDKContext(suite.Ctx),
			&operatortypes.SetConsKeyReq{
				Address:       operatorAddressString,
				AvsAddress:    avsAddress,
				PublicKeyJSON: newKey.ToJSON(),
			},
		)
		return err
	})
	testcases := []struct {
		name            string
		operations      []funcThatReturnsError
		errValues       []error
		expUpdatesCount int
		powers          []int64
		validatorKey    keytypes.WrappedConsKey
	}{
		{
			name: "opt in - base case",
			operations: []funcThatReturnsError{
				optIn,
			},
			errValues:       []error{nil},
			expUpdatesCount: 1,
			powers:          []int64{amountUSD},
			validatorKey:    oldKey,
		},
		{
			name: "opt out without opting in",
			operations: []funcThatReturnsError{
				optOut,
			},
			errValues: []error{operatortypes.ErrNotOptedIn},
		},
		{
			name: "set key without opting in",
			operations: []funcThatReturnsError{
				setKey,
			},
			errValues: []error{operatortypes.ErrNotOptedIn},
		},
		{
			name: "opt in then replace",
			operations: []funcThatReturnsError{
				optIn, setKey,
			},
			errValues:       []error{nil, nil},
			expUpdatesCount: 1,
			powers:          []int64{amountUSD},
			validatorKey:    newKey,
		},
		{
			name: "opt in then opt out",
			operations: []funcThatReturnsError{
				optIn, optOut,
			},
			errValues:       []error{nil, nil},
			expUpdatesCount: 0,
			powers:          []int64{},
		},
		{
			name: "opt in then replace then opt out",
			operations: []funcThatReturnsError{
				optIn, setKey, optOut,
			},
			errValues:       []error{nil, nil, nil},
			expUpdatesCount: 0,
			powers:          []int64{},
		},
	}
	for _, tc := range testcases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			setUp()
			suite.Require().Equal(
				len(tc.operations), len(tc.errValues),
				"unequal `operations` and `errValues` length",
			)
			for i := range tc.operations {
				suite.ErrorIs(tc.operations[i](), tc.errValues[i])
			}
			suite.CheckLengthOfValidatorUpdates(
				tc.expUpdatesCount, tc.powers, tc.name,
			)
			if tc.validatorKey != nil {
				suite.CheckValidatorFound(
					tc.validatorKey, true, chainIDWithoutRevision, operatorAddress,
				)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestDifferentEpochOperations() {
	// generate addresses and register operators
	operatorAddress := sdk.AccAddress(utiltx.GenerateAddress().Bytes())
	operatorAddressString := operatorAddress.String()
	amountUSD := suite.App.StakingKeeper.GetMinSelfDelegation(suite.Ctx).Int64()
	setUp := func() {
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
	}
	// generate keys, and get the AVS address
	oldKey := utiltx.GenerateConsensusKey()
	newKey := utiltx.GenerateConsensusKey()
	chainIDWithoutRevision := avstypes.ChainIDWithoutRevision(suite.Ctx.ChainID())
	_, avsAddress := suite.App.AVSManagerKeeper.IsAVSByChainID(suite.Ctx, chainIDWithoutRevision)

	// now define the operations
	type funcThatReturnsError func() error
	optIn := funcThatReturnsError(func() error {
		_, err := suite.OperatorMsgServer.OptIntoAVS(
			sdk.WrapSDKContext(suite.Ctx),
			&operatortypes.OptIntoAVSReq{
				FromAddress:   operatorAddressString,
				AvsAddress:    avsAddress,
				PublicKeyJSON: oldKey.ToJSON(),
			},
		)
		return err
	})
	optOut := funcThatReturnsError(func() error {
		_, err := suite.OperatorMsgServer.OptOutOfAVS(
			sdk.WrapSDKContext(suite.Ctx),
			&operatortypes.OptOutOfAVSReq{
				FromAddress: operatorAddressString,
				AvsAddress:  avsAddress,
			},
		)
		return err
	})
	setKey := funcThatReturnsError(func() error {
		_, err := suite.OperatorMsgServer.SetConsKey(
			sdk.WrapSDKContext(suite.Ctx),
			&operatortypes.SetConsKeyReq{
				Address:       operatorAddressString,
				AvsAddress:    avsAddress,
				PublicKeyJSON: newKey.ToJSON(),
			},
		)
		return err
	})
	testcases := []struct {
		name            string
		operations      []funcThatReturnsError
		errValues       []error
		expUpdatesCount []int
		powers          [][]int64
		validatorKeys   []keytypes.WrappedConsKey
		ultimateKey     keytypes.WrappedConsKey
		absentKeys      []keytypes.WrappedConsKey
	}{
		{
			name: "opt in - base case",
			operations: []funcThatReturnsError{
				optIn,
			},
			errValues:       []error{nil},
			expUpdatesCount: []int{1},
			powers: [][]int64{
				{amountUSD},
			},
			validatorKeys: []keytypes.WrappedConsKey{oldKey},
			ultimateKey:   oldKey,
			absentKeys:    []keytypes.WrappedConsKey{newKey},
		},
		{
			name: "opt out without opting in",
			operations: []funcThatReturnsError{
				optOut,
			},
			errValues:       []error{operatortypes.ErrNotOptedIn},
			expUpdatesCount: []int{0},
			powers: [][]int64{
				{},
			},
			validatorKeys: []keytypes.WrappedConsKey{nil},
			ultimateKey:   nil,
			absentKeys:    []keytypes.WrappedConsKey{oldKey, newKey},
		},
		{
			name: "set key without opting in",
			operations: []funcThatReturnsError{
				setKey,
			},
			errValues:       []error{operatortypes.ErrNotOptedIn},
			expUpdatesCount: []int{0},
			powers: [][]int64{
				{},
			},
			validatorKeys: []keytypes.WrappedConsKey{nil},
			ultimateKey:   nil,
			absentKeys:    []keytypes.WrappedConsKey{oldKey, newKey},
		},
		{
			name: "opt in then replace",
			operations: []funcThatReturnsError{
				optIn, setKey,
			},
			errValues:       []error{nil, nil},
			expUpdatesCount: []int{1, 2},
			powers: [][]int64{
				{amountUSD},
				{amountUSD, 0},
			},
			validatorKeys: []keytypes.WrappedConsKey{
				oldKey, newKey,
			},
			ultimateKey: newKey,
			absentKeys:  []keytypes.WrappedConsKey{oldKey},
		},
		{
			name: "opt in then opt out",
			operations: []funcThatReturnsError{
				optIn, optOut,
			},
			errValues:       []error{nil, nil},
			expUpdatesCount: []int{1, 1},
			powers: [][]int64{
				{amountUSD},
				{0},
			},
			validatorKeys: []keytypes.WrappedConsKey{oldKey, nil},
			ultimateKey:   nil,
			absentKeys:    []keytypes.WrappedConsKey{oldKey, newKey},
		},
		{
			name: "opt in then replace then opt out",
			operations: []funcThatReturnsError{
				optIn, setKey, optOut,
			},
			errValues:       []error{nil, nil, nil},
			expUpdatesCount: []int{1, 2, 1},
			powers: [][]int64{
				{amountUSD},
				{amountUSD, 0},
				{0},
			},
			validatorKeys: []keytypes.WrappedConsKey{oldKey, newKey, nil},
			ultimateKey:   nil,
			absentKeys:    []keytypes.WrappedConsKey{oldKey, newKey},
		},
		{
			name: "opt in then opt out then opt in",
			operations: []funcThatReturnsError{
				optIn, optOut, optIn,
			},
			errValues:       []error{nil, nil, operatortypes.ErrAlreadyRemovingKey},
			expUpdatesCount: []int{1, 1, 0},
			powers: [][]int64{
				{amountUSD},
				{0},
				{},
			},
			validatorKeys: []keytypes.WrappedConsKey{oldKey, nil, nil},
			ultimateKey:   nil,
			absentKeys:    []keytypes.WrappedConsKey{oldKey, newKey},
		},
	}
	for _, tc := range testcases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			setUp()
			suite.Require().Equal(
				len(tc.operations), len(tc.errValues),
				"unequal `operations` and `errValues` length",
			)
			suite.Require().Equal(
				len(tc.operations), len(tc.expUpdatesCount),
				"unequal `operations` and `expUpdatesCount` length",
			)
			suite.Require().Equal(
				len(tc.operations), len(tc.powers),
				"unequal `operations` and `powers` length",
			)
			suite.Require().Equal(
				len(tc.operations), len(tc.validatorKeys),
				"unequal `operations` and `validatorKeys` length",
			)
			for i := range tc.operations {
				expErr := tc.errValues[i]
				suite.ErrorIs(tc.operations[i](), expErr)
				if expErr == nil {
					suite.CheckLengthOfValidatorUpdates(
						tc.expUpdatesCount[i], tc.powers[i], tc.name,
					)
					if tc.validatorKeys[i] != nil {
						suite.CheckValidatorFound(
							tc.validatorKeys[i], true, chainIDWithoutRevision, operatorAddress,
						)
					}
				}
			}
			for i := 0; i < int(s.App.StakingKeeper.GetEpochsUntilUnbonded(s.Ctx)); i++ {
				suite.CommitAfter(suite.EpochDuration)
				suite.Commit()
			}
			if tc.ultimateKey != nil {
				suite.CheckValidatorFound(
					tc.ultimateKey, true, chainIDWithoutRevision, operatorAddress,
				)
			}
			for _, key := range tc.absentKeys {
				suite.CheckValidatorFound(
					key, false, chainIDWithoutRevision, operatorAddress,
				)
			}
		})
	}
}
