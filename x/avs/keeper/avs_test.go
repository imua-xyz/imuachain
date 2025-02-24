package keeper_test

import (
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	testutiltx "github.com/imua-xyz/imuachain/testutil/tx"
	"github.com/prysmaticlabs/prysm/v4/crypto/bls/blst"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	"github.com/ethereum/go-ethereum/common"
	assetstypes "github.com/imua-xyz/imuachain/x/assets/types"

	"github.com/imua-xyz/imuachain/x/avs/types"
	delegationtypes "github.com/imua-xyz/imuachain/x/delegation/types"
	epochstypes "github.com/imua-xyz/imuachain/x/epochs/types"
	operatorTypes "github.com/imua-xyz/imuachain/x/operator/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	utiltx "github.com/evmos/evmos/v16/testutil/tx"
	avstypes "github.com/imua-xyz/imuachain/x/avs/types"
)

func (suite *AVSTestSuite) TestAVS() {
	avsName := "avsTest"
	avsAddress := suite.avsAddress
	avsOwnerAddress := []string{
		sdk.AccAddress(utiltx.GenerateAddress().Bytes()).String(),
		sdk.AccAddress(utiltx.GenerateAddress().Bytes()).String(),
		sdk.AccAddress(utiltx.GenerateAddress().Bytes()).String(),
	}
	assetIDs := suite.AssetIDs
	operatorAddress := sdk.AccAddress(utiltx.GenerateAddress().Bytes()).String()

	avs := &types.AVSInfo{
		Name:                avsName,
		AvsAddress:          avsAddress.String(),
		SlashAddress:        utiltx.GenerateAddress().String(),
		AvsOwnerAddresses:   avsOwnerAddress,
		AssetIDs:            assetIDs,
		AvsUnbondingPeriod:  2,
		MinSelfDelegation:   10,
		EpochIdentifier:     epochstypes.DayEpochID,
		StartingEpoch:       1,
		MinOptInOperators:   100,
		MinTotalStakeAmount: 1000,
		AvsSlash:            sdk.MustNewDecFromStr("0.001"),
		AvsReward:           sdk.MustNewDecFromStr("0.002"),
		TaskAddress:         sdk.AccAddress(utiltx.GenerateAddress().Bytes()).String(),
		WhitelistAddresses:  []string{operatorAddress},
	}

	err := suite.App.AVSManagerKeeper.SetAVSInfo(suite.Ctx, avs)
	suite.NoError(err)

	whitelisted, err := suite.App.AVSManagerKeeper.IsWhitelisted(suite.Ctx, avsAddress.String(), operatorAddress)
	suite.NoError(err)
	suite.Equal(whitelisted, true)
	info, err := suite.App.AVSManagerKeeper.GetAVSInfo(suite.Ctx, avsAddress.String())

	suite.NoError(err)
	suite.Equal(avsAddress.String(), info.GetInfo().AvsAddress)

	var avsList []types.AVSInfo
	suite.App.AVSManagerKeeper.IterateAVSInfo(suite.Ctx, func(_ int64, epochEndAVSInfo types.AVSInfo) (stop bool) {
		avsList = append(avsList, epochEndAVSInfo)
		return false
	})
	suite.Equal(len(avsList), 2) // + dogfood avs
	suite.CommitAfter(48*time.Hour + time.Nanosecond)
	// commit will run the EndBlockers for the current block, call app.Commit
	// and then run the BeginBlockers for the next block with the new time.
	// during the BeginBlocker, the epoch will be incremented.
	epoch, found := suite.App.EpochsKeeper.GetEpochInfo(suite.Ctx, epochstypes.DayEpochID)
	suite.Equal(found, true)
	suite.Equal(epoch.CurrentEpoch, int64(2))
	suite.CommitAfter(48*time.Hour + time.Nanosecond)
}

func (suite *AVSTestSuite) TestUpdateAVSInfo_Register() {
	avsName, avsAddres, slashAddress, rewardAddress := "avsTest", "0xDF907c29719154eb9872f021d21CAE6E5025d7aB", "0xDF907c29719154eb9872f021d21CAE6E5025d7aB", "0xDF907c29719154eb9872f021d21CAE6E5025d7aB"
	avsOwnerAddress := []string{
		sdk.AccAddress(utiltx.GenerateAddress().Bytes()).String(),
		sdk.AccAddress(utiltx.GenerateAddress().Bytes()).String(),
		sdk.AccAddress(utiltx.GenerateAddress().Bytes()).String(),
	}
	assetIDs := suite.AssetIDs

	avsParams := &types.AVSRegisterOrDeregisterParams{
		AvsName:               avsName,
		AvsAddress:            common.HexToAddress(avsAddres),
		Action:                types.RegisterAction,
		RewardContractAddress: common.HexToAddress(rewardAddress),
		AvsOwnerAddresses:     avsOwnerAddress,
		AssetIDs:              assetIDs,
		MinSelfDelegation:     uint64(10),
		UnbondingPeriod:       uint64(2),
		SlashContractAddress:  common.HexToAddress(slashAddress),
		EpochIdentifier:       epochstypes.DayEpochID,
	}

	err := suite.App.AVSManagerKeeper.UpdateAVSInfo(suite.Ctx, avsParams)
	suite.NoError(err)

	info, err := suite.App.AVSManagerKeeper.GetAVSInfo(suite.Ctx, avsAddres)

	suite.NoError(err)
	suite.Equal(strings.ToLower(avsAddres), info.GetInfo().AvsAddress)

	err = suite.App.AVSManagerKeeper.UpdateAVSInfo(suite.Ctx, avsParams)
	suite.Error(err)
	suite.Contains(err.Error(), types.ErrAlreadyRegistered.Error())
}

func (suite *AVSTestSuite) TestUpdateAVSInfo_DeRegister() {
	// Test case setup
	avsName, avsAddress, slashAddress := "avsTest", suite.avsAddress.String(), "0xDF907c29719154eb9872f021d21CAE6E5025d7aB"
	avsOwnerAddress := []string{
		sdk.AccAddress(utiltx.GenerateAddress().Bytes()).String(),
		sdk.AccAddress(utiltx.GenerateAddress().Bytes()).String(),
		sdk.AccAddress(utiltx.GenerateAddress().Bytes()).String(),
	}
	assetIDs := suite.AssetIDs

	avsParams := &types.AVSRegisterOrDeregisterParams{
		AvsName:              avsName,
		AvsAddress:           common.HexToAddress(avsAddress),
		Action:               types.DeRegisterAction,
		AvsOwnerAddresses:    avsOwnerAddress,
		AssetIDs:             assetIDs,
		MinSelfDelegation:    uint64(10),
		UnbondingPeriod:      uint64(2),
		SlashContractAddress: common.HexToAddress(slashAddress),
		EpochIdentifier:      epochstypes.DayEpochID,
	}

	err := suite.App.AVSManagerKeeper.UpdateAVSInfo(suite.Ctx, avsParams)
	suite.Error(err)
	suite.Contains(err.Error(), types.ErrUnregisterNonExistent.Error())

	avsParams.Action = types.RegisterAction
	err = suite.App.AVSManagerKeeper.UpdateAVSInfo(suite.Ctx, avsParams)
	suite.NoError(err)
	info, err := suite.App.AVSManagerKeeper.GetAVSInfo(suite.Ctx, avsAddress)
	suite.NoError(err)
	suite.Equal(strings.ToLower(avsAddress), info.GetInfo().AvsAddress)

	epoch, _ := suite.App.EpochsKeeper.GetEpochInfo(suite.Ctx, epochstypes.DayEpochID)
	// Numbered loops for epoch ends
	for epochEnd := epoch.CurrentEpoch; epochEnd <= int64(info.Info.StartingEpoch)+2; epochEnd++ {
		suite.CommitAfter(time.Hour * 24)
		epoch, found := suite.App.EpochsKeeper.GetEpochInfo(suite.Ctx, epochstypes.DayEpochID)
		suite.Equal(found, true)
		suite.Equal(epoch.CurrentEpoch, epochEnd+1)
	}

	avsParams.Action = avstypes.DeRegisterAction
	avsParams.CallerAddress, err = sdk.AccAddressFromBech32(avsOwnerAddress[0])

	suite.NoError(err)
	err = suite.App.AVSManagerKeeper.UpdateAVSInfo(suite.Ctx, avsParams)
	suite.NoError(err)
	info, err = suite.App.AVSManagerKeeper.GetAVSInfo(suite.Ctx, avsAddress)
	suite.Error(err)
	suite.Contains(err.Error(), types.ErrNoKeyInTheStore.Error())
	suite.Nil(info)
}

func (suite *AVSTestSuite) TestUpdateAVSInfoWithOperator_Register() {
	avsAddress := suite.avsAddress
	operatorAddress := sdk.AccAddress(utiltx.GenerateAddress().Bytes())

	operatorParams := &types.OperatorOptParams{
		AvsAddress:      avsAddress,
		Action:          types.RegisterAction,
		OperatorAddress: operatorAddress,
	}
	//  operator Not Exist
	err := suite.App.AVSManagerKeeper.OperatorOptAction(suite.Ctx, operatorParams)
	suite.Error(err)
	suite.Contains(err.Error(), delegationtypes.ErrOperatorNotExist.Error())

	// register operator but avs not register
	// register operator
	registerReq := &operatorTypes.RegisterOperatorReq{
		FromAddress: operatorAddress.String(),
		Info: &operatorTypes.OperatorInfo{
			EarningsAddr: operatorAddress.String(),
			ApproveAddr:  operatorAddress.String(),
		},
	}
	_, err = suite.OperatorMsgServer.RegisterOperator(sdk.WrapSDKContext(suite.Ctx), registerReq)
	suite.NoError(err)
	suite.TestAVS() // registers the AVS

	asset := suite.Assets[0]
	_, assetID := assetstypes.GetStakerIDAndAssetIDFromStr(asset.LayerZeroChainID, "", asset.Address)
	selfDelegateAmount := big.NewInt(10)
	minPrecisionSelfDelegateAmount := big.NewInt(0).Mul(selfDelegateAmount, big.NewInt(0).Exp(big.NewInt(10), big.NewInt(int64(asset.Decimals)), nil))
	err = suite.App.AssetsKeeper.UpdateOperatorAssetState(suite.Ctx, operatorAddress, assetID, assetstypes.DeltaOperatorSingleAsset{
		TotalAmount:   math.NewIntFromBigInt(minPrecisionSelfDelegateAmount),
		TotalShare:    math.LegacyNewDecFromBigInt(minPrecisionSelfDelegateAmount),
		OperatorShare: math.LegacyNewDecFromBigInt(minPrecisionSelfDelegateAmount),
	})
	suite.NoError(err)
	err = suite.App.AVSManagerKeeper.OperatorOptAction(suite.Ctx, operatorParams)
	suite.NoError(err)
}

func (suite *AVSTestSuite) TestAddressSwitch() {
	addr := common.HexToAddress("0x8dF46478a83Ab2a429979391E9546A12AfF9E33f")
	var accAddress sdk.AccAddress = addr[:]
	suite.Equal("im13h6xg79g82e2g2vhjwg7j4r2z2hlncel7zgwsx", accAddress.String())
	commonAddress := common.Address(accAddress)
	suite.Equal(common.HexToAddress("0x8dF46478a83Ab2a429979391E9546A12AfF9E33f"), commonAddress)
}

func (suite *AVSTestSuite) TestRegisterBLSPublicKey() {
	type testCase struct {
		name          string
		setupParams   func() *types.BlsParams
		errorContains string
	}

	testCases := []testCase{
		{
			name: "successful registration",
			setupParams: func() *types.BlsParams {
				privateKey, err := blst.RandKey()
				suite.NoError(err)
				operatorAddress := sdk.AccAddress(utiltx.GenerateAddress().Bytes())
				msg := fmt.Sprintf(types.BLSMessageToSign, types.ChainIDWithoutRevision(suite.Ctx.ChainID()), operatorAddress.String())
				hashedMsg := crypto.Keccak256Hash([]byte(msg))
				sig := privateKey.Sign(hashedMsg.Bytes())

				return &types.BlsParams{
					OperatorAddress:             operatorAddress,
					AvsAddress:                  testutiltx.GenerateAddress(),
					PubKey:                      privateKey.PublicKey().Marshal(),
					PubKeyRegistrationSignature: sig.Marshal(),
				}
			},
		},
		{
			name: "reuse BLS key - same operator + avs",
			setupParams: func() *types.BlsParams {
				privateKey, err := blst.RandKey()
				suite.NoError(err)
				operatorAddress := sdk.AccAddress(utiltx.GenerateAddress().Bytes())
				msg := fmt.Sprintf(types.BLSMessageToSign, types.ChainIDWithoutRevision(suite.Ctx.ChainID()), operatorAddress.String())
				hashedMsg := crypto.Keccak256Hash([]byte(msg))
				sig := privateKey.Sign(hashedMsg.Bytes())

				params := types.BlsParams{
					OperatorAddress:             operatorAddress,
					AvsAddress:                  testutiltx.GenerateAddress(),
					PubKey:                      privateKey.PublicKey().Marshal(),
					PubKeyRegistrationSignature: sig.Marshal(),
				}
				err = suite.App.AVSManagerKeeper.RegisterBLSPublicKey(suite.Ctx, &params)
				suite.NoError(err)
				return &params
			},
			errorContains: errorsmod.Wrap(
				types.ErrAlreadyExists,
				"a key has already been set for this operator and avs",
			).Error(),
		},
		{
			name: "reuse BLS key - different operator + same avs",
			setupParams: func() *types.BlsParams {
				privateKey, err := blst.RandKey()
				suite.NoError(err)
				operatorAddress := sdk.AccAddress(utiltx.GenerateAddress().Bytes())
				msg := fmt.Sprintf(types.BLSMessageToSign, types.ChainIDWithoutRevision(suite.Ctx.ChainID()), operatorAddress.String())
				hashedMsg := crypto.Keccak256Hash([]byte(msg))
				sig := privateKey.Sign(hashedMsg.Bytes())

				params := types.BlsParams{
					OperatorAddress:             operatorAddress,
					AvsAddress:                  testutiltx.GenerateAddress(),
					PubKey:                      privateKey.PublicKey().Marshal(),
					PubKeyRegistrationSignature: sig.Marshal(),
				}
				err = suite.App.AVSManagerKeeper.RegisterBLSPublicKey(suite.Ctx, &params)
				suite.NoError(err)

				anotherOperatorAddress := sdk.AccAddress(utiltx.GenerateAddress().Bytes())
				msg = fmt.Sprintf(types.BLSMessageToSign, types.ChainIDWithoutRevision(suite.Ctx.ChainID()), anotherOperatorAddress.String())
				hashedMsg = crypto.Keccak256Hash([]byte(msg))
				sig = privateKey.Sign(hashedMsg.Bytes())

				return &types.BlsParams{
					OperatorAddress:             anotherOperatorAddress,
					AvsAddress:                  params.AvsAddress,
					PubKey:                      privateKey.PublicKey().Marshal(),
					PubKeyRegistrationSignature: sig.Marshal(),
				}
			},
			errorContains: errorsmod.Wrap(
				types.ErrAlreadyExists,
				"this BLS key is already in use",
			).Error(),
		},
		{
			name: "wrong chain ID",
			setupParams: func() *types.BlsParams {
				privateKey, err := blst.RandKey()
				suite.NoError(err)
				operatorAddress := sdk.AccAddress(utiltx.GenerateAddress().Bytes())
				msg := fmt.Sprintf(types.BLSMessageToSign, "onemorechain_211", operatorAddress.String())
				hashedMsg := crypto.Keccak256Hash([]byte(msg))
				sig := privateKey.Sign(hashedMsg.Bytes())

				return &types.BlsParams{
					OperatorAddress:             operatorAddress,
					AvsAddress:                  testutiltx.GenerateAddress(),
					PubKey:                      privateKey.PublicKey().Marshal(),
					PubKeyRegistrationSignature: sig.Marshal(),
				}
			},
			errorContains: types.ErrSigNotMatchPubKey.Error(),
		},
		{
			name: "wrong operator address in signature",
			setupParams: func() *types.BlsParams {
				privateKey, err := blst.RandKey()
				suite.NoError(err)
				operatorAddress := sdk.AccAddress(utiltx.GenerateAddress().Bytes())
				anotherOperatorAddress := sdk.AccAddress(utiltx.GenerateAddress().Bytes())
				msg := fmt.Sprintf(types.BLSMessageToSign, types.ChainIDWithoutRevision(suite.Ctx.ChainID()), anotherOperatorAddress.String())
				hashedMsg := crypto.Keccak256Hash([]byte(msg))
				sig := privateKey.Sign(hashedMsg.Bytes())

				return &types.BlsParams{
					OperatorAddress:             operatorAddress,
					AvsAddress:                  testutiltx.GenerateAddress(),
					PubKey:                      privateKey.PublicKey().Marshal(),
					PubKeyRegistrationSignature: sig.Marshal(),
				}
			},
			errorContains: types.ErrSigNotMatchPubKey.Error(),
		},
		{
			name: "mismatched BLS key",
			setupParams: func() *types.BlsParams {
				privateKey, err := blst.RandKey()
				suite.NoError(err)
				operatorAddress := sdk.AccAddress(utiltx.GenerateAddress().Bytes())
				msg := fmt.Sprintf(types.BLSMessageToSign, types.ChainIDWithoutRevision(suite.Ctx.ChainID()), operatorAddress.String())
				hashedMsg := crypto.Keccak256Hash([]byte(msg))
				sig := privateKey.Sign(hashedMsg.Bytes())
				// generate a different private key
				anotherPrivateKey, err := blst.RandKey()
				suite.NoError(err)
				anotherPublicKey := anotherPrivateKey.PublicKey()
				return &types.BlsParams{
					OperatorAddress: operatorAddress,
					AvsAddress:      testutiltx.GenerateAddress(),
					// provide a different public key than the one which signed it, so that verification fails
					PubKey:                      anotherPublicKey.Marshal(),
					PubKeyRegistrationSignature: sig.Marshal(),
				}
			},
			errorContains: types.ErrSigNotMatchPubKey.Error(),
		},
		{
			name: "invalid public key format",
			setupParams: func() *types.BlsParams {
				privateKey, err := blst.RandKey()
				suite.NoError(err)
				operatorAddress := sdk.AccAddress(utiltx.GenerateAddress().Bytes())
				msg := fmt.Sprintf(types.BLSMessageToSign, types.ChainIDWithoutRevision(suite.Ctx.ChainID()), operatorAddress.String())
				hashedMsg := crypto.Keccak256Hash([]byte(msg))
				sig := privateKey.Sign(hashedMsg.Bytes())
				return &types.BlsParams{
					OperatorAddress:             operatorAddress,
					AvsAddress:                  testutiltx.GenerateAddress(),
					PubKey:                      []byte("invalid"),
					PubKeyRegistrationSignature: sig.Marshal(),
				}
			},
			errorContains: types.ErrParsePubKey.Error(),
		},
		{
			name: "invalid signature format",
			setupParams: func() *types.BlsParams {
				privateKey, err := blst.RandKey()
				suite.NoError(err)
				operatorAddress := sdk.AccAddress(utiltx.GenerateAddress().Bytes())
				return &types.BlsParams{
					OperatorAddress:             operatorAddress,
					AvsAddress:                  testutiltx.GenerateAddress(),
					PubKey:                      privateKey.PublicKey().Marshal(),
					PubKeyRegistrationSignature: []byte("invalid"),
				}
			},
			errorContains: types.ErrSigNotMatchPubKey.Error(),
		},
		{
			name: "empty operator address",
			setupParams: func() *types.BlsParams {
				operatorAddress := sdk.AccAddress{}
				privateKey, err := blst.RandKey()
				suite.NoError(err)
				msg := fmt.Sprintf(types.BLSMessageToSign, types.ChainIDWithoutRevision(suite.Ctx.ChainID()), operatorAddress.String())
				hashedMsg := crypto.Keccak256Hash([]byte(msg))
				sig := privateKey.Sign(hashedMsg.Bytes())
				return &types.BlsParams{
					OperatorAddress:             operatorAddress,
					AvsAddress:                  testutiltx.GenerateAddress(),
					PubKey:                      privateKey.PublicKey().Marshal(),
					PubKeyRegistrationSignature: sig.Marshal(),
				}
			},
			errorContains: types.ErrInvalidAddr.Error(),
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			params := tc.setupParams()
			err := suite.App.AVSManagerKeeper.RegisterBLSPublicKey(suite.Ctx, params)

			if tc.errorContains != "" {
				suite.Error(err)
				suite.Contains(err.Error(), tc.errorContains)
			} else {
				suite.NoError(err)
			}
		})
	}
}
