package keeper_test

import (
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	keepertest "github.com/ExocoreNetwork/exocore/testutil/keeper"
	"github.com/ExocoreNetwork/exocore/x/appchain/common/types"
	coordinatortypes "github.com/ExocoreNetwork/exocore/x/appchain/coordinator/types"
	avstypes "github.com/ExocoreNetwork/exocore/x/avs/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
)

func TestOnRecvSlashPacket(t *testing.T) {
	k, ctx, mocks := keepertest.NewCoordinatorKeeper(t)

	testCases := []struct {
		name          string
		setupMocks    func()
		packet        channeltypes.Packet
		data          types.SlashPacketData
		expectedError error
	}{
		{
			name:       "unknown channel",
			setupMocks: func() {},
			packet:     channeltypes.Packet{DestinationChannel: "unknown-channel"},
			data:       types.SlashPacketData{},
			expectedError: coordinatortypes.ErrUnknownSubscriberChannelID.Wrapf(
				"slash packet on unknown-channel",
			),
		},
		{
			name: "invalid packet data",
			setupMocks: func() {
				k.SetChainForChannel(ctx, "channel-0", "chain-0")
				// mocks.ChannelKeeper.EXPECT().GetChannel(gomock.Any(), gomock.Any(), gomock.Any()).Return(channeltypes.Channel{}, true)
			},
			packet: channeltypes.Packet{DestinationChannel: "channel-0"},
			data:   types.SlashPacketData{},
			expectedError: types.ErrInvalidPacketData.Wrapf(
				"invalid slash packet: empty validator address",
			),
		},
		{
			name: "successful slash packet",
			setupMocks: func() {
				k.SetChainForChannel(ctx, "channel-0", "chain-0")
				k.MapHeightToChainVscID(ctx, "chain-0", 1, 1)
				mocks.OperatorKeeper.EXPECT().GetOperatorAddressForChainIDAndConsAddr(gomock.Any(), gomock.Any(), gomock.Any()).Return(true, sdk.AccAddress([]byte("hello"))).Times(2)
				mocks.AVSKeeper.EXPECT().IsAVSByChainID(gomock.Any(), gomock.Any()).Return(true, avstypes.GenerateAVSAddr("chain-0"))
				mocks.OperatorKeeper.EXPECT().ApplySlashForHeight(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			},
			packet: channeltypes.Packet{DestinationChannel: "channel-0"},
			data: *types.NewSlashPacketData(
				abci.Validator{Address: []byte("validator"), Power: 1}, 1, stakingtypes.Infraction_INFRACTION_DOWNTIME,
			),
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.setupMocks()
			result, err := k.OnRecvSlashPacket(ctx, tc.packet, tc.data)
			if tc.expectedError != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tc.expectedError)
			} else {
				require.NoError(t, err)
				require.Equal(t, types.SlashPacketHandledResult.Bytes(), result)
			}
		})
	}
}

func TestOnRecvVscMaturedPacket(t *testing.T) {
	k, ctx, mocks := keepertest.NewCoordinatorKeeper(t)
	_ = mocks

	testCases := []struct {
		name          string
		setupMocks    func()
		packet        channeltypes.Packet
		data          types.VscMaturedPacketData
		expectedError error
	}{
		{
			name:       "unknown channel",
			setupMocks: func() {},
			packet:     channeltypes.Packet{DestinationChannel: "unknown-channel"},
			data:       types.VscMaturedPacketData{},
			expectedError: coordinatortypes.ErrUnknownSubscriberChannelID.Wrapf(
				"vsc matured packet on unknown-channel",
			),
		},
		{
			name: "successful vsc matured packet",
			setupMocks: func() {
				k.SetChainForChannel(ctx, "channel-0", "chain-0")
				// mocks.ChannelKeeper.EXPECT().GetChannel(gomock.Any(), gomock.Any(), gomock.Any()).Return(channeltypes.Channel{}, true)
			},
			packet:        channeltypes.Packet{DestinationChannel: "channel-0"},
			data:          types.VscMaturedPacketData{ValsetUpdateID: 1},
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.setupMocks()
			err := k.OnRecvVscMaturedPacket(ctx, tc.packet, tc.data)
			if tc.expectedError != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tc.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
