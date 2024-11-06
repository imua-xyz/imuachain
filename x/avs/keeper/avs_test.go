package keeper_test

import (
	"cosmossdk.io/math"
	assetstypes "github.com/ExocoreNetwork/exocore/x/assets/types"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"strings"
	"time"

	"github.com/ExocoreNetwork/exocore/x/avs/types"
	avstypes "github.com/ExocoreNetwork/exocore/x/avs/types"
	delegationtypes "github.com/ExocoreNetwork/exocore/x/delegation/types"
	epochstypes "github.com/ExocoreNetwork/exocore/x/epochs/types"
	operatorTypes "github.com/ExocoreNetwork/exocore/x/operator/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	utiltx "github.com/evmos/evmos/v16/testutil/tx"
)

func (suite *AVSTestSuite) TestAVS() {
	avsName := "avsTest"
	avsAddress := suite.avsAddress
	avsOwnerAddress := []string{"exo13h6xg79g82e2g2vhjwg7j4r2z2hlncelwutkjr", "exo13h6xg79g82e2g2vhjwg7j4r2z2hlncelwutkj1", "exo13h6xg79g82e2g2vhjwg7j4r2z2hlncelwutkj2"}
	assetID := suite.AssetIDs
	avs := &types.AVSInfo{
		Name:                avsName,
		AvsAddress:          avsAddress.String(),
		SlashAddr:           utiltx.GenerateAddress().String(),
		AvsOwnerAddress:     avsOwnerAddress,
		AssetIDs:            assetID,
		AvsUnbondingPeriod:  2,
		MinSelfDelegation:   10,
		EpochIdentifier:     epochstypes.DayEpochID,
		StartingEpoch:       1,
		MinOptInOperators:   100,
		MinTotalStakeAmount: 1000,
		AvsSlash:            sdk.MustNewDecFromStr("0.001"),
		AvsReward:           sdk.MustNewDecFromStr("0.002"),
		TaskAddr:            "exo13h6xg79g82e2g2vhjwg7j4r2z2hlncelwutkjr",
	}

	err := suite.App.AVSManagerKeeper.SetAVSInfo(suite.Ctx, avs)
	suite.NoError(err)

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
	avsOwnerAddress := []string{"exo13h6xg79g82e2g2vhjwg7j4r2z2hlncelwutkjr", "exo13h6xg79g82e2g2vhjwg7j4r2z2hlncelwutkj1", "exo13h6xg79g82e2g2vhjwg7j4r2z2hlncelwutkj2"}
	assetID := suite.AssetIDs

	avsParams := &types.AVSRegisterOrDeregisterParams{
		AvsName:            avsName,
		AvsAddress:         common.HexToAddress(avsAddres),
		Action:             avstypes.RegisterAction,
		RewardContractAddr: common.HexToAddress(rewardAddress),
		AvsOwnerAddress:    avsOwnerAddress,
		AssetID:            assetID,
		MinSelfDelegation:  uint64(10),
		UnbondingPeriod:    uint64(2),
		SlashContractAddr:  common.HexToAddress(slashAddress),
		EpochIdentifier:    epochstypes.DayEpochID,
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
	avsOwnerAddress := []string{"exo13h6xg79g82e2g2vhjwg7j4r2z2hlncelwutkjr", "exo13h6xg79g82e2g2vhjwg7j4r2z2hlncelwutkj1", "exo13h6xg79g82e2g2vhjwg7j4r2z2hlncelwutkj2"}
	assetID := suite.AssetIDs

	avsParams := &types.AVSRegisterOrDeregisterParams{
		AvsName:           avsName,
		AvsAddress:        common.HexToAddress(avsAddress),
		Action:            avstypes.DeRegisterAction,
		AvsOwnerAddress:   avsOwnerAddress,
		AssetID:           assetID,
		MinSelfDelegation: uint64(10),
		UnbondingPeriod:   uint64(2),
		SlashContractAddr: common.HexToAddress(slashAddress),
		EpochIdentifier:   epochstypes.DayEpochID,
	}

	err := suite.App.AVSManagerKeeper.UpdateAVSInfo(suite.Ctx, avsParams)
	suite.Error(err)
	suite.Contains(err.Error(), types.ErrUnregisterNonExistent.Error())

	avsParams.Action = avstypes.RegisterAction
	err = suite.App.AVSManagerKeeper.UpdateAVSInfo(suite.Ctx, avsParams)
	suite.NoError(err)
	info, err := suite.App.AVSManagerKeeper.GetAVSInfo(suite.Ctx, avsAddress)
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
	avsParams.CallerAddress, err = sdk.AccAddressFromBech32("exo13h6xg79g82e2g2vhjwg7j4r2z2hlncelwutkjr")
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
