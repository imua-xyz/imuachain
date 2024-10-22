package keeper_test

import (
	"testing"

	testutilkeeper "github.com/ExocoreNetwork/exocore/testutil/keeper"
	commontypes "github.com/ExocoreNetwork/exocore/x/appchain/common/types"
	"github.com/ExocoreNetwork/exocore/x/appchain/subscriber/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	transfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestEndBlockSendRewards(t *testing.T) {
	keeper, ctx, mocks := testutilkeeper.NewSubscriberKeeper(t)

	// Set up expectations
	mocks.BankKeeper.EXPECT().GetAllBalances(gomock.Any(), gomock.Any()).Return(sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(1000)))).AnyTimes()
	mocks.AccountKeeper.EXPECT().GetModuleAccount(gomock.Any(), gomock.Any()).Return(&mockModuleAccount{sdk.AccAddress(address.Module("fee_collector", []byte{}))}).AnyTimes()
	mocks.BankKeeper.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	// Set up params
	params := commontypes.DefaultSubscriberParams()
	params.BlocksPerDistributionTransmission = 100
	keeper.SetSubscriberParams(ctx, params)

	// Test when it's not time to send rewards
	keeper.SetLastRewardTransmissionHeight(ctx, ctx.BlockHeight())
	keeper.EndBlockSendRewards(ctx)

	// Test when it's time to send rewards
	keeper.SetLastRewardTransmissionHeight(ctx, ctx.BlockHeight()-100)
	mocks.ChannelKeeper.EXPECT().GetChannel(gomock.Any(), gomock.Any(), gomock.Any()).Return(channeltypes.Channel{State: channeltypes.OPEN}, true)
	mocks.IBCTransferKeeper.EXPECT().Transfer(gomock.Any(), gomock.Any()).Return(nil, nil)
	keeper.EndBlockSendRewards(ctx)

	// Verify that LastRewardTransmissionHeight was updated
	require.Equal(t, ctx.BlockHeight(), keeper.GetLastRewardTransmissionHeight(ctx))
}

func TestSplitRewardsInternally(t *testing.T) {
	keeper, ctx, mocks := testutilkeeper.NewSubscriberKeeper(t)

	// Set up expectations
	feePoolAddr := sdk.AccAddress(address.Module("fee_collector", []byte{}))
	mocks.AccountKeeper.EXPECT().GetModuleAccount(gomock.Any(), "fee_collector").Return(sdk.AccAddress(feePoolAddr))

	initialBalance := sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(1000)))
	mocks.BankKeeper.EXPECT().GetAllBalances(gomock.Any(), feePoolAddr).Return(initialBalance)

	// Set up params
	params := commontypes.DefaultSubscriberParams()
	params.SubscriberRedistributionFraction = "0.3"
	keeper.SetSubscriberParams(ctx, params)

	// Expect two transfers
	mocks.BankKeeper.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), "fee_collector", types.SubscriberRedistributeName, gomock.Any()).Return(nil)
	mocks.BankKeeper.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), "fee_collector", types.SubscriberToSendToCoordinatorName, gomock.Any()).Return(nil)

	keeper.SplitRewardsInternally(ctx)
}

func TestShouldSendRewardsToCoordinator(t *testing.T) {
	keeper, ctx, _ := testutilkeeper.NewSubscriberKeeper(t)

	// Set up params
	params := commontypes.DefaultSubscriberParams()
	params.BlocksPerDistributionTransmission = 100
	keeper.SetSubscriberParams(ctx, params)

	// Test when it's not time to send rewards
	keeper.SetLastRewardTransmissionHeight(ctx, ctx.BlockHeight()-99)
	require.False(t, keeper.ShouldSendRewardsToCoordinator(ctx))

	// Test when it's time to send rewards
	keeper.SetLastRewardTransmissionHeight(ctx, ctx.BlockHeight()-100)
	require.True(t, keeper.ShouldSendRewardsToCoordinator(ctx))
}

func TestSendRewardsToCoordinator(t *testing.T) {
	keeper, ctx, mocks := testutilkeeper.NewSubscriberKeeper(t)

	// Set up expectations
	mocks.ChannelKeeper.EXPECT().GetChannel(gomock.Any(), transfertypes.PortID, gomock.Any()).Return(channeltypes.Channel{State: channeltypes.OPEN}, true)

	toSendAddr := sdk.AccAddress(address.Module(types.SubscriberToSendToCoordinatorName, []byte{}))
	mocks.AccountKeeper.EXPECT().GetModuleAccount(gomock.Any(), types.SubscriberToSendToCoordinatorName).Return(sdk.AccAddress(toSendAddr))

	// Set up params
	params := commontypes.DefaultSubscriberParams()
	params.RewardDenom = "stake"
	params.CoordinatorFeePoolAddrStr = "cosmos1qyqszqgpqyqszqgpqyqszqgpqyqszqgpqyqszqgpqyqszqgpqyqs"
	keeper.SetSubscriberParams(ctx, params)

	// Test when balance is zero
	mocks.BankKeeper.EXPECT().GetBalance(gomock.Any(), toSendAddr, "stake").Return(sdk.NewCoin("stake", sdk.ZeroInt()))
	err := keeper.SendRewardsToCoordinator(ctx)
	require.NoError(t, err)

	// Test when balance is non-zero
	mocks.BankKeeper.EXPECT().GetBalance(gomock.Any(), toSendAddr, "stake").Return(sdk.NewCoin("stake", sdk.NewInt(1000)))
	mocks.IBCTransferKeeper.EXPECT().Transfer(gomock.Any(), gomock.Any()).Return(nil, nil)
	err = keeper.SendRewardsToCoordinator(ctx)
	require.NoError(t, err)
}

type mockModuleAccount struct {
	sdk.AccAddress
}

func (m *mockModuleAccount) GetAddress() sdk.AccAddress {
	return m.AccAddress
}

func (m *mockModuleAccount) SetAddress(addr sdk.AccAddress) error {
	m.AccAddress = addr
	return nil
}

func (m *mockModuleAccount) GetPubKey() cryptotypes.PubKey {
	return nil
}

func (m *mockModuleAccount) SetPubKey(pubKey cryptotypes.PubKey) error {
	return nil
}

func (m *mockModuleAccount) GetAccountNumber() uint64 {
	return 0
}

func (m *mockModuleAccount) SetAccountNumber(num uint64) error {
	return nil
}

func (m *mockModuleAccount) GetSequence() uint64 {
	return 0
}

func (m *mockModuleAccount) SetSequence(seq uint64) error {
	return nil
}

func (m *mockModuleAccount) GetName() string {
	return "mock"
}

func (m *mockModuleAccount) GetPermissions() []string {
	return []string{}
}

func (m *mockModuleAccount) HasPermission(string) bool {
	return false
}

func (m *mockModuleAccount) String() string {
	return "mock"
}

func (m *mockModuleAccount) ProtoMessage() {}

func (m *mockModuleAccount) Reset() {}

var _ authtypes.ModuleAccountI = &mockModuleAccount{}
