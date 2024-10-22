package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ExocoreNetwork/exocore/testutil/keeper"
	commontypes "github.com/ExocoreNetwork/exocore/x/appchain/common/types"
)

func TestKeeper_PendingPackets(t *testing.T) {
	k, ctx, _ := keeper.NewSubscriberKeeper(t)

	t.Run("AppendPendingPacket and GetPendingPackets", func(t *testing.T) {
		packetType := commontypes.VscMaturedPacket
		packet := commontypes.NewVscMaturedPacketData(1)
		wrappedPacket := &commontypes.SubscriberPacketData_VscMaturedPacketData{VscMaturedPacketData: packet}

		// Append a packet
		k.AppendPendingPacket(
			ctx, packetType,
			wrappedPacket,
		)

		// Get all pending packets
		packets := k.GetPendingPackets(ctx)
		require.Len(t, packets, 1)
		require.Equal(t, packetType, packets[0].Type)
		require.Equal(t, packet.ValsetUpdateID, wrappedPacket.VscMaturedPacketData.ValsetUpdateID)
	})

	t.Run("GetAllPendingPacketsWithIdx", func(t *testing.T) {
		// Append two more packets
		k.AppendPendingPacket(
			ctx, commontypes.VscMaturedPacket,
			&commontypes.SubscriberPacketData_VscMaturedPacketData{VscMaturedPacketData: commontypes.NewVscMaturedPacketData(2)},
		)
		k.AppendPendingPacket(
			ctx, commontypes.VscMaturedPacket,
			&commontypes.SubscriberPacketData_VscMaturedPacketData{VscMaturedPacketData: commontypes.NewVscMaturedPacketData(3)},
		)

		packetsWithIdx := k.GetAllPendingPacketsWithIdx(ctx)
		require.Len(t, packetsWithIdx, 3)
		require.Equal(t, uint64(0), packetsWithIdx[0].Idx)
		require.Equal(t, uint64(1), packetsWithIdx[1].Idx)
		require.Equal(t, uint64(2), packetsWithIdx[2].Idx)
	})

	t.Run("DeletePendingDataPackets", func(t *testing.T) {
		// Delete the second packet
		k.DeletePendingDataPackets(ctx, 1)

		packetsWithIdx := k.GetAllPendingPacketsWithIdx(ctx)
		require.Len(t, packetsWithIdx, 2)
		require.Equal(t, uint64(0), packetsWithIdx[0].Idx)
		require.Equal(t, uint64(2), packetsWithIdx[1].Idx)
	})

	t.Run("DeleteAllPendingDataPackets", func(t *testing.T) {
		k.DeleteAllPendingDataPackets(ctx)

		packets := k.GetPendingPackets(ctx)
		require.Empty(t, packets)
	})

	t.Run("getAndIncrementPendingPacketsIdx", func(t *testing.T) {
		k.DeleteAllPendingDataPackets(ctx)
		// This is an internal method, so we'll test it indirectly
		for i := 0; i < 5; i++ {
			k.AppendPendingPacket(
				ctx, commontypes.VscMaturedPacket,
				&commontypes.SubscriberPacketData_VscMaturedPacketData{VscMaturedPacketData: commontypes.NewVscMaturedPacketData(uint64(i))},
			)
		}

		packetsWithIdx := k.GetAllPendingPacketsWithIdx(ctx)
		require.Len(t, packetsWithIdx, 5)
		firstIdx := packetsWithIdx[0].Idx
		for i, packet := range packetsWithIdx {
			require.Equal(t, firstIdx+uint64(i), packet.Idx)
		}
	})
}
