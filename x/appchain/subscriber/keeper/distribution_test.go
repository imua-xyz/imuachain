package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ExocoreNetwork/exocore/testutil/keeper"
	transfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
)

func TestKeeper_ChannelOpenInit(t *testing.T) {
	k, ctx, mocks := keeper.NewSubscriberKeeper(t)

	msg := &channeltypes.MsgChannelOpenInit{
		PortId: "test-port",
		Channel: channeltypes.Channel{
			State:    channeltypes.INIT,
			Ordering: channeltypes.UNORDERED,
			Counterparty: channeltypes.Counterparty{
				PortId:    "counterparty-port",
				ChannelId: "",
			},
			ConnectionHops: []string{"connection-0"},
			Version:        "1",
		},
	}

	expectedResponse := &channeltypes.MsgChannelOpenInitResponse{}

	mocks.IBCCoreKeeper.EXPECT().
		ChannelOpenInit(gomock.Any(), msg).
		Return(expectedResponse, nil)

	response, err := k.ChannelOpenInit(ctx, msg)

	require.NoError(t, err)
	assert.Equal(t, expectedResponse, response)
}

func TestKeeper_TransferChannelExists(t *testing.T) {
	k, ctx, mocks := keeper.NewSubscriberKeeper(t)

	testCases := []struct {
		name      string
		channelID string
		exists    bool
	}{
		{
			name:      "Existing channel",
			channelID: "channel-0",
			exists:    true,
		},
		{
			name:      "Non-existing channel",
			channelID: "channel-1",
			exists:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mocks.ChannelKeeper.EXPECT().
				GetChannel(ctx, transfertypes.PortID, tc.channelID).
				Return(channeltypes.Channel{}, tc.exists)

			exists := k.TransferChannelExists(ctx, tc.channelID)
			assert.Equal(t, tc.exists, exists)
		})
	}
}

func TestKeeper_GetConnectionHops(t *testing.T) {
	k, ctx, mocks := keeper.NewSubscriberKeeper(t)

	testCases := []struct {
		name          string
		srcPort       string
		srcChan       string
		expectedHops  []string
		expectedError bool
	}{
		{
			name:          "Existing channel",
			srcPort:       "test-port",
			srcChan:       "channel-0",
			expectedHops:  []string{"connection-0"},
			expectedError: false,
		},
		{
			name:          "Non-existing channel",
			srcPort:       "test-port",
			srcChan:       "channel-1",
			expectedHops:  []string{},
			expectedError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.expectedError {
				mocks.ChannelKeeper.EXPECT().
					GetChannel(ctx, tc.srcPort, tc.srcChan).
					Return(channeltypes.Channel{}, false)
			} else {
				mocks.ChannelKeeper.EXPECT().
					GetChannel(ctx, tc.srcPort, tc.srcChan).
					Return(channeltypes.Channel{ConnectionHops: tc.expectedHops}, true)
			}

			hops, err := k.GetConnectionHops(ctx, tc.srcPort, tc.srcChan)

			if tc.expectedError {
				require.Error(t, err)
				assert.Empty(t, hops)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectedHops, hops)
			}
		})
	}
}
