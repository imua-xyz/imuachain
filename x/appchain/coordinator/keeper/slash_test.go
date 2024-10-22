package keeper_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	testkeeper "github.com/ExocoreNetwork/exocore/testutil/keeper"
	commontypes "github.com/ExocoreNetwork/exocore/x/appchain/common/types"
	avstypes "github.com/ExocoreNetwork/exocore/x/avs/types"
	abci "github.com/cometbft/cometbft/abci/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func TestKeeper_ValidateSlashPacket(t *testing.T) {
	k, ctx, mocks := testkeeper.NewCoordinatorKeeper(t)

	chainID := "test-chain"
	valsetUpdateID := uint64(1)
	validatorAddress := []byte("validator-address")

	// Set up mock expectations
	mocks.AVSKeeper.EXPECT().IsAVSByChainID(gomock.Any(), chainID).Return(
		true, avstypes.GenerateAVSAddr(chainID),
	).AnyTimes()
	mocks.OperatorKeeper.EXPECT().GetOperatorAddressForChainIDAndConsAddr(
		gomock.Any(), chainID, sdk.ConsAddress(validatorAddress),
	).Return(true, sdk.AccAddress{}).AnyTimes()

	t.Run("valid packet", func(t *testing.T) {
		k.MapHeightToChainVscID(ctx, chainID, valsetUpdateID, 100)

		data := commontypes.NewSlashPacketData(
			abci.Validator{
				Address: validatorAddress,
			},
			valsetUpdateID,
			stakingtypes.Infraction_INFRACTION_DOWNTIME,
		)

		err := k.ValidateSlashPacket(ctx, chainID, *data)
		require.NoError(t, err)
	})

	t.Run("invalid valset update ID", func(t *testing.T) {
		data := commontypes.NewSlashPacketData(
			abci.Validator{
				Address: validatorAddress,
			},
			999,
			stakingtypes.Infraction_INFRACTION_DOWNTIME,
		)

		err := k.ValidateSlashPacket(ctx, chainID, *data)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid chainID")
	})

}

func TestKeeper_HandleSlashPacket(t *testing.T) {
	k, ctx, mocks := testkeeper.NewCoordinatorKeeper(t)

	chainID := "test-chain"
	valsetUpdateID := uint64(1)
	validatorAddress := []byte("validator-address")
	operatorAddress := sdk.AccAddress("operator-address")
	avsAddress := avstypes.GenerateAVSAddr(chainID)

	// Set up mock expectations
	mocks.AVSKeeper.EXPECT().IsAVSByChainID(gomock.Any(), chainID).Return(true, avsAddress).AnyTimes()
	mocks.OperatorKeeper.EXPECT().GetOperatorAddressForChainIDAndConsAddr(
		gomock.Any(), chainID, sdk.ConsAddress(validatorAddress),
	).Return(true, operatorAddress).AnyTimes()
	mocks.OperatorKeeper.EXPECT().ApplySlashForHeight(
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
	).Return(nil).AnyTimes()

	t.Run("handle slash packet", func(t *testing.T) {
		k.MapHeightToChainVscID(ctx, chainID, valsetUpdateID, 100)
		k.SetSubSlashFractionDowntime(ctx, chainID, "0.1")
		k.SetSubDowntimeJailDuration(ctx, chainID, time.Hour)

		data := commontypes.NewSlashPacketData(
			abci.Validator{
				Address: validatorAddress,
			},
			valsetUpdateID,
			stakingtypes.Infraction_INFRACTION_DOWNTIME,
		)

		k.HandleSlashPacket(ctx, chainID, *data)

		// Check if slash ack was appended
		slashAcks := k.GetSlashAcks(ctx, chainID)
		require.Len(t, slashAcks.List, 1)
		require.Equal(t, sdk.ConsAddress(validatorAddress).Bytes(), slashAcks.List[0])
	})
}

func TestKeeper_SlashAcks(t *testing.T) {
	k, ctx, _ := testkeeper.NewCoordinatorKeeper(t)

	chainID := "test-chain"
	consAddress1 := sdk.ConsAddress("validator1")
	consAddress2 := sdk.ConsAddress("validator2")

	t.Run("append and get slash acks", func(t *testing.T) {
		k.AppendSlashAck(ctx, chainID, consAddress1)
		k.AppendSlashAck(ctx, chainID, consAddress2)

		slashAcks := k.GetSlashAcks(ctx, chainID)
		require.Len(t, slashAcks.List, 2)
		require.Equal(t, consAddress1.Bytes(), slashAcks.List[0])
		require.Equal(t, consAddress2.Bytes(), slashAcks.List[1])
	})

	t.Run("consume slash acks", func(t *testing.T) {
		consumedAcks := k.ConsumeSlashAcks(ctx, chainID)
		require.Len(t, consumedAcks, 2)
		require.Equal(t, consAddress1.Bytes(), consumedAcks[0])
		require.Equal(t, consAddress2.Bytes(), consumedAcks[1])

		// Check that acks were cleared
		remainingAcks := k.GetSlashAcks(ctx, chainID)
		require.Empty(t, remainingAcks.List)
	})
}

func TestKeeper_SubSlashFractionAndJailDuration(t *testing.T) {
	k, ctx, _ := testkeeper.NewCoordinatorKeeper(t)

	chainID := "test-chain"

	t.Run("set and get sub slash fraction downtime", func(t *testing.T) {
		k.SetSubSlashFractionDowntime(ctx, chainID, "0.1")
		fraction := k.GetSubSlashFractionDowntime(ctx, chainID)
		require.Equal(t, "0.1", fraction)
	})

	t.Run("set and get sub slash fraction double sign", func(t *testing.T) {
		k.SetSubSlashFractionDoubleSign(ctx, chainID, "0.5")
		fraction := k.GetSubSlashFractionDoubleSign(ctx, chainID)
		require.Equal(t, "0.5", fraction)
	})

	t.Run("set and get sub downtime jail duration", func(t *testing.T) {
		duration := 24 * time.Hour
		k.SetSubDowntimeJailDuration(ctx, chainID, duration)
		retrievedDuration := k.GetSubDowntimeJailDuration(ctx, chainID)
		require.Equal(t, duration, retrievedDuration)
	})
}
