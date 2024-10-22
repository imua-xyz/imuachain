package keeper_test

import (
	"testing"

	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v7/modules/core/24-host"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	keepertest "github.com/ExocoreNetwork/exocore/testutil/keeper"
)

func TestKeeper_ChanCloseInit(t *testing.T) {
	k, ctx, mocks := keepertest.NewSubscriberKeeper(t)

	testCases := []struct {
		name      string
		portID    string
		channelID string
		setup     func()
		expError  bool
	}{
		{
			name:      "success",
			portID:    "port-1",
			channelID: "channel-1",
			setup: func() {
				capName := host.ChannelCapabilityPath("port-1", "channel-1")
				capability := &capabilitytypes.Capability{
					Index: 1,
				}
				mocks.ScopedKeeper.EXPECT().GetCapability(gomock.Any(), capName).Return(capability, true)
				mocks.ChannelKeeper.EXPECT().ChanCloseInit(gomock.Any(), "port-1", "channel-1", capability).Return(nil)
			},
			expError: false,
		},
		{
			name:      "capability not found",
			portID:    "port-2",
			channelID: "channel-2",
			setup: func() {
				capName := host.ChannelCapabilityPath("port-2", "channel-2")
				mocks.ScopedKeeper.EXPECT().GetCapability(gomock.Any(), capName).Return(nil, false)
			},
			expError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()
			err := k.ChanCloseInit(ctx, tc.portID, tc.channelID)
			if tc.expError {
				require.Error(t, err)
				require.ErrorIs(t, err, channeltypes.ErrChannelCapabilityNotFound)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
