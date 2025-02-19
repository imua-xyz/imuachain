package keeper_test

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	utiltx "github.com/evmos/evmos/v16/testutil/tx"
	avstypes "github.com/imua-xyz/imuachain/x/avs/keeper"
	"github.com/imua-xyz/imuachain/x/avs/types"
	delegationtypes "github.com/imua-xyz/imuachain/x/delegation/types"
	epochstypes "github.com/imua-xyz/imuachain/x/epochs/types"
	operatorTypes "github.com/imua-xyz/imuachain/x/operator/types"
)

func (suite *AVSTestSuite) TestAVS() {
	avsName := "avsTest"
	avsAddress := suite.avsAddress
	avsOwnerAddress := []string{"exo13h6xg79g82e2g2vhjwg7j4r2z2hlncelwutkjr", "exo13h6xg79g82e2g2vhjwg7j4r2z2hlncelwutkj1", "exo13h6xg79g82e2g2vhjwg7j4r2z2hlncelwutkj2"}
	assetID := []string{"0xdac17f958d2ee523a2206206994597c13d831ec7_0x65"}
	avs := &types.AVSInfo{
		Name:                avsName,
		AvsAddress:          avsAddress.String(),
		SlashAddr:           utiltx.GenerateAddress().String(),
		AvsOwnerAddress:     avsOwnerAddress,
		AssetIDs:            assetID,
		AvsUnbondingPeriod:  7,
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
	avsName, avsAddres, slashAddress, rewardAddress := "avsTest", "exo18cggcpvwspnd5c6ny8wrqxpffj5zmhklprtnph", "0xDF907c29719154eb9872f021d21CAE6E5025d7aB", "0xDF907c29719154eb9872f021d21CAE6E5025d7aB"
	avsOwnerAddress := []string{"exo13h6xg79g82e2g2vhjwg7j4r2z2hlncelwutkjr", "exo13h6xg79g82e2g2vhjwg7j4r2z2hlncelwutkj1", "exo13h6xg79g82e2g2vhjwg7j4r2z2hlncelwutkj2"}
	assetID := []string{"11", "22", "33"}

	avsParams := &types.AVSRegisterOrDeregisterParams{
		AvsName:            avsName,
		AvsAddress:         avsAddres,
		Action:             avstypes.RegisterAction,
		RewardContractAddr: rewardAddress,
		AvsOwnerAddress:    avsOwnerAddress,
		AssetID:            assetID,
		MinSelfDelegation:  uint64(10),
		UnbondingPeriod:    uint64(7),
		SlashContractAddr:  slashAddress,
		EpochIdentifier:    epochstypes.DayEpochID,
	}

	err := suite.App.AVSManagerKeeper.UpdateAVSInfo(suite.Ctx, avsParams)
	suite.NoError(err)

	info, err := suite.App.AVSManagerKeeper.GetAVSInfo(suite.Ctx, avsAddres)

	suite.NoError(err)
	suite.Equal(avsAddres, info.GetInfo().AvsAddress)

	err = suite.App.AVSManagerKeeper.UpdateAVSInfo(suite.Ctx, avsParams)
	suite.Error(err)
	suite.Contains(err.Error(), types.ErrAlreadyRegistered.Error())
}

func (suite *AVSTestSuite) TestUpdateAVSInfo_DeRegister() {
	// Test case setup
	avsName, avsAddres, slashAddress := "avsTest", suite.avsAddress.String(), "exo13h6xg79g82e2g2vhjwg7j4r2z2hlncelwutash"
	avsOwnerAddress := []string{"exo13h6xg79g82e2g2vhjwg7j4r2z2hlncelwutkjr", "exo13h6xg79g82e2g2vhjwg7j4r2z2hlncelwutkj1", "exo13h6xg79g82e2g2vhjwg7j4r2z2hlncelwutkj2"}
	assetID := []string{"11", "22", "33", "44", "55"} // Multiple assets

	avsParams := &types.AVSRegisterOrDeregisterParams{
		AvsName:           avsName,
		AvsAddress:        avsAddres,
		Action:            avstypes.DeRegisterAction,
		AvsOwnerAddress:   avsOwnerAddress,
		AssetID:           assetID,
		MinSelfDelegation: uint64(10),
		UnbondingPeriod:   uint64(7),
		SlashContractAddr: slashAddress,
		EpochIdentifier:   epochstypes.DayEpochID,
	}

	err := suite.App.AVSManagerKeeper.UpdateAVSInfo(suite.Ctx, avsParams)
	suite.Error(err)
	suite.Contains(err.Error(), types.ErrUnregisterNonExistent.Error())

	avsParams.Action = avstypes.RegisterAction
	err = suite.App.AVSManagerKeeper.UpdateAVSInfo(suite.Ctx, avsParams)
	suite.NoError(err)
	info, err := suite.App.AVSManagerKeeper.GetAVSInfo(suite.Ctx, avsAddres)
	suite.Equal(avsAddres, info.GetInfo().AvsAddress)

	avsParams.Action = avstypes.DeRegisterAction
	avsParams.CallerAddress = "exo13h6xg79g82e2g2vhjwg7j4r2z2hlncelwutkjr"
	err = suite.App.AVSManagerKeeper.UpdateAVSInfo(suite.Ctx, avsParams)
	suite.NoError(err)
	info, err = suite.App.AVSManagerKeeper.GetAVSInfo(suite.Ctx, avsAddres)
	suite.Error(err)
	suite.Contains(err.Error(), types.ErrNoKeyInTheStore.Error())
}

func (suite *AVSTestSuite) TestUpdateAVSInfoWithOperator_Register() {
	avsAddress := suite.avsAddress
	operatorAddress := sdk.AccAddress(utiltx.GenerateAddress().Bytes()).String()

	opAccAddr, err := sdk.AccAddressFromBech32(operatorAddress)
	suite.NoError(err)
	operatorParams := &avstypes.OperatorOptParams{
		AvsAddress:      avsAddress.String(),
		Action:          avstypes.RegisterAction,
		OperatorAddress: operatorAddress,
	}
	//  operator Not Exist
	err = suite.App.AVSManagerKeeper.OperatorOptAction(suite.Ctx, operatorParams)
	suite.Error(err)
	suite.Contains(err.Error(), delegationtypes.ErrOperatorNotExist.Error())

	// register operator but avs not register
	// register operator
	registerReq := &operatorTypes.RegisterOperatorReq{
		FromAddress: opAccAddr.String(),
		Info: &operatorTypes.OperatorInfo{
			EarningsAddr: opAccAddr.String(),
		},
	}
	_, err = suite.OperatorMsgServer.RegisterOperator(sdk.WrapSDKContext(suite.Ctx), registerReq)
	suite.NoError(err)
	suite.TestAVS() // registers the AVS

	err = suite.App.AVSManagerKeeper.OperatorOptAction(suite.Ctx, operatorParams)
	suite.NoError(err)
}
