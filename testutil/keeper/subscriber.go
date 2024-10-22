package keeper

import (
	"testing"

	tmdb "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/libs/log"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/store"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	commontypes "github.com/ExocoreNetwork/exocore/x/appchain/common/types"
	"github.com/ExocoreNetwork/exocore/x/appchain/subscriber/keeper"
	"github.com/ExocoreNetwork/exocore/x/appchain/subscriber/types"
)

// SubscriberMockedKeepers contains all the mocked keepers
type SubscriberMockedKeepers struct {
	AccountKeeper     *commontypes.MockAccountKeeper
	BankKeeper        *commontypes.MockBankKeeper
	ScopedKeeper      *commontypes.MockScopedKeeper
	PortKeeper        *commontypes.MockPortKeeper
	ClientKeeper      *commontypes.MockClientKeeper
	ConnectionKeeper  *commontypes.MockConnectionKeeper
	ChannelKeeper     *commontypes.MockChannelKeeper
	IBCCoreKeeper     *commontypes.MockIBCCoreKeeper
	IBCTransferKeeper *commontypes.MockIBCTransferKeeper
}

func NewSubscriberKeeper(t testing.TB) (keeper.Keeper, sdk.Context, SubscriberMockedKeepers) {
	storeKey := sdk.NewKVStoreKey(types.StoreKey)
	memStoreKey := storetypes.NewMemoryStoreKey(types.MemStoreKey)

	db := tmdb.NewMemDB()
	stateStore := store.NewCommitMultiStore(db)
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(memStoreKey, storetypes.StoreTypeMemory, nil)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cryptocodec.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.TestingLogger())

	// Create mock controller
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	// Create mock keepers
	mockedKeepers := SubscriberMockedKeepers{
		AccountKeeper:     commontypes.NewMockAccountKeeper(ctrl),
		BankKeeper:        commontypes.NewMockBankKeeper(ctrl),
		ScopedKeeper:      commontypes.NewMockScopedKeeper(ctrl),
		PortKeeper:        commontypes.NewMockPortKeeper(ctrl),
		ClientKeeper:      commontypes.NewMockClientKeeper(ctrl),
		ConnectionKeeper:  commontypes.NewMockConnectionKeeper(ctrl),
		ChannelKeeper:     commontypes.NewMockChannelKeeper(ctrl),
		IBCCoreKeeper:     commontypes.NewMockIBCCoreKeeper(ctrl),
		IBCTransferKeeper: commontypes.NewMockIBCTransferKeeper(ctrl),
	}

	k := keeper.NewKeeper(
		cdc,
		storeKey,
		mockedKeepers.AccountKeeper,
		mockedKeepers.BankKeeper,
		mockedKeepers.ScopedKeeper,
		mockedKeepers.PortKeeper,
		mockedKeepers.ClientKeeper,
		mockedKeepers.ConnectionKeeper,
		mockedKeepers.ChannelKeeper,
		mockedKeepers.IBCCoreKeeper,
		mockedKeepers.IBCTransferKeeper,
		"fee_collector", // feeCollectorName
	)

	// Initialize params if needed
	// k.SetParams(ctx, types.DefaultParams())

	return k, ctx, mockedKeepers
}
