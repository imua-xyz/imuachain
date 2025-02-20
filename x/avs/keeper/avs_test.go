package keeper_test

import (
	"math/big"
	"strings"
	"time"

	"cosmossdk.io/math"
	"github.com/ethereum/go-ethereum/common"
	assetstypes "github.com/imua-xyz/imuachain/x/assets/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	utiltx "github.com/evmos/evmos/v16/testutil/tx"
	"github.com/imua-xyz/imuachain/x/avs/types"
	avstypes "github.com/imua-xyz/imuachain/x/avs/types"
	delegationtypes "github.com/imua-xyz/imuachain/x/delegation/types"
	epochstypes "github.com/imua-xyz/imuachain/x/epochs/types"
	operatorTypes "github.com/imua-xyz/imuachain/x/operator/types"
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
		Action:                avstypes.RegisterAction,
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
		Action:               avstypes.DeRegisterAction,
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

	avsParams.Action = avstypes.RegisterAction
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
}

func (suite *AVSTestSuite) TestUpdateAVSInfoWithOperator_Register() {
	avsAddress := suite.avsAddress
	operatorAddress := sdk.AccAddress(utiltx.GenerateAddress().Bytes())

	operatorParams := &avstypes.OperatorOptParams{
		AvsAddress:      avsAddress,
		Action:          avstypes.RegisterAction,
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
