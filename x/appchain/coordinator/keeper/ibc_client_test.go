package keeper_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	sdk "github.com/cosmos/cosmos-sdk/types"

	testkeeper "github.com/ExocoreNetwork/exocore/testutil/keeper"
	keytypes "github.com/ExocoreNetwork/exocore/types/keys"
	commontypes "github.com/ExocoreNetwork/exocore/x/appchain/common/types"
	"github.com/ExocoreNetwork/exocore/x/appchain/coordinator/keeper"
	"github.com/ExocoreNetwork/exocore/x/appchain/coordinator/types"
	epochstypes "github.com/ExocoreNetwork/exocore/x/epochs/types"

	testutiltx "github.com/ExocoreNetwork/exocore/testutil/tx"
	"go.uber.org/mock/gomock"

	commitmenttypes "github.com/cosmos/ibc-go/v7/modules/core/23-commitment/types"
	ibctmtypes "github.com/cosmos/ibc-go/v7/modules/light-clients/07-tendermint"
)

type IBCClientTestSuite struct {
	suite.Suite

	ctx    sdk.Context
	keeper keeper.Keeper
	mocks  testkeeper.MockedKeepers
}

func (suite *IBCClientTestSuite) SetupTest() {
	suite.keeper, suite.ctx, suite.mocks = testkeeper.NewCoordinatorKeeper(suite.T())
}

func TestIBCClientTestSuite(t *testing.T) {
	suite.Run(t, new(IBCClientTestSuite))
}

func (suite *IBCClientTestSuite) TestActivateScheduledChains() {
	testCases := []struct {
		name            string
		setupMocks      func()
		epochIdentifier string
		epochNumber     int64
	}{
		{
			name: "Successful activation of one scheduled chain",
			setupMocks: func() {
				pendingChains := types.PendingSubscriberChainRequests{
					List: []types.RegisterSubscriberChainRequest{
						{
							ChainID:              "chain-1",
							FromAddress:          sdk.AccAddress(testutiltx.GenerateAddress().Bytes()).String(),
							EpochIdentifier:      "day",
							AssetIDs:             []string{"asset-1"},
							MinSelfDelegationUsd: 100,
							MaxValidators:        100,
							SubscriberParams: commontypes.SubscriberParams{
								UnbondingPeriod: time.Hour * 24 * 14,
							},
						},
					},
				}
				for _, chain := range pendingChains.List {
					suite.keeper.AppendPendingSubChain(suite.ctx, "day", 1, &chain)
				}
				suite.mocks.StakingKeeper.EXPECT().UnbondingTime(gomock.Any()).Return(time.Hour * 24 * 14)
				mockConsensusState := &ibctmtypes.ConsensusState{
					// Fill with appropriate test data
					Timestamp:          time.Now(),
					Root:               commitmenttypes.NewMerkleRoot([]byte("root")),
					NextValidatorsHash: []byte("next_validators_hash"),
				}
				suite.mocks.ClientKeeper.EXPECT().GetSelfConsensusState(gomock.Any(), gomock.Any()).Return(mockConsensusState, nil)
				suite.mocks.OperatorKeeper.EXPECT().GetActiveOperatorsForChainID(gomock.Any(), gomock.Any()).Return(
					[]sdk.AccAddress{
						sdk.AccAddress(testutiltx.GenerateAddress().Bytes()),
						sdk.AccAddress(testutiltx.GenerateAddress().Bytes())},
					[]keytypes.WrappedConsKey{
						testutiltx.GenerateConsensusKey(),
						testutiltx.GenerateConsensusKey(),
					},
				)
				suite.mocks.OperatorKeeper.EXPECT().GetVotePowerForChainID(gomock.Any(), gomock.Any(), gomock.Any()).Return([]int64{100, 200}, nil)
				suite.mocks.ClientKeeper.EXPECT().CreateClient(gomock.Any(), gomock.Any(), gomock.Any()).Return("clientID", nil)
				suite.mocks.EpochsKeeper.EXPECT().GetEpochInfo(gomock.Any(), gomock.Any()).Return(epochstypes.EpochInfo{CurrentEpoch: 1}, true)
			},
			epochIdentifier: "day",
			epochNumber:     1,
		},
		{
			name: "Activation of multiple scheduled chains",
			setupMocks: func() {
				pendingChains := types.PendingSubscriberChainRequests{
					List: []types.RegisterSubscriberChainRequest{
						{
							ChainID:              "chain-1",
							FromAddress:          sdk.AccAddress(testutiltx.GenerateAddress().Bytes()).String(),
							EpochIdentifier:      "week",
							AssetIDs:             []string{"asset-1"},
							MinSelfDelegationUsd: 100,
							MaxValidators:        100,
							SubscriberParams: commontypes.SubscriberParams{
								UnbondingPeriod: time.Hour * 24 * 14,
							},
						},
						{
							ChainID:              "chain-2",
							FromAddress:          sdk.AccAddress(testutiltx.GenerateAddress().Bytes()).String(),
							EpochIdentifier:      "week",
							AssetIDs:             []string{"asset-2"},
							MinSelfDelegationUsd: 200,
							MaxValidators:        50,
							SubscriberParams: commontypes.SubscriberParams{
								UnbondingPeriod: time.Hour * 24 * 21,
							},
						},
					},
				}
				for _, chain := range pendingChains.List {
					suite.keeper.AppendPendingSubChain(suite.ctx, "week", 1, &chain)
				}
				suite.mocks.StakingKeeper.EXPECT().UnbondingTime(gomock.Any()).Return(time.Hour * 24 * 14).Times(2)
				mockConsensusState := &ibctmtypes.ConsensusState{
					Timestamp:          time.Now(),
					Root:               commitmenttypes.NewMerkleRoot([]byte("root")),
					NextValidatorsHash: []byte("next_validators_hash"),
				}
				suite.mocks.ClientKeeper.EXPECT().GetSelfConsensusState(gomock.Any(), gomock.Any()).Return(mockConsensusState, nil).Times(2)
				suite.mocks.OperatorKeeper.EXPECT().GetActiveOperatorsForChainID(gomock.Any(), gomock.Any()).Return(
					[]sdk.AccAddress{
						sdk.AccAddress(testutiltx.GenerateAddress().Bytes()),
						sdk.AccAddress(testutiltx.GenerateAddress().Bytes())},
					[]keytypes.WrappedConsKey{
						testutiltx.GenerateConsensusKey(),
						testutiltx.GenerateConsensusKey(),
					},
				).Times(2)
				suite.mocks.OperatorKeeper.EXPECT().GetVotePowerForChainID(gomock.Any(), gomock.Any(), gomock.Any()).Return([]int64{100, 200}, nil).Times(2)
				suite.mocks.ClientKeeper.EXPECT().CreateClient(gomock.Any(), gomock.Any(), gomock.Any()).Return("clientID1", nil)
				suite.mocks.ClientKeeper.EXPECT().CreateClient(gomock.Any(), gomock.Any(), gomock.Any()).Return("clientID2", nil)
				suite.mocks.EpochsKeeper.EXPECT().GetEpochInfo(gomock.Any(), gomock.Any()).Return(epochstypes.EpochInfo{CurrentEpoch: 1}, true).Times(2)
			},
			epochIdentifier: "week",
			epochNumber:     1,
		},
		{
			name: "Activation with one chain failing",
			setupMocks: func() {
				pendingChains := types.PendingSubscriberChainRequests{
					List: []types.RegisterSubscriberChainRequest{
						{
							ChainID:              "chain-1",
							FromAddress:          sdk.AccAddress(testutiltx.GenerateAddress().Bytes()).String(),
							EpochIdentifier:      "month",
							AssetIDs:             []string{"asset-1"},
							MinSelfDelegationUsd: 100,
							MaxValidators:        100,
							SubscriberParams: commontypes.SubscriberParams{
								UnbondingPeriod: time.Hour * 24 * 14,
							},
						},
						{
							ChainID:              "chain-2",
							FromAddress:          sdk.AccAddress(testutiltx.GenerateAddress().Bytes()).String(),
							EpochIdentifier:      "month",
							AssetIDs:             []string{"asset-2"},
							MinSelfDelegationUsd: 200,
							MaxValidators:        50,
							SubscriberParams: commontypes.SubscriberParams{
								UnbondingPeriod: time.Hour * 24 * 21,
							},
						},
					},
				}
				for _, chain := range pendingChains.List {
					suite.keeper.AppendPendingSubChain(suite.ctx, "month", 1, &chain)
				}
				suite.mocks.StakingKeeper.EXPECT().UnbondingTime(gomock.Any()).Return(time.Hour * 24 * 14).Times(2)
				mockConsensusState := &ibctmtypes.ConsensusState{
					Timestamp:          time.Now(),
					Root:               commitmenttypes.NewMerkleRoot([]byte("root")),
					NextValidatorsHash: []byte("next_validators_hash"),
				}
				suite.mocks.ClientKeeper.EXPECT().GetSelfConsensusState(gomock.Any(), gomock.Any()).Return(mockConsensusState, nil).Times(2)
				suite.mocks.OperatorKeeper.EXPECT().GetActiveOperatorsForChainID(gomock.Any(), gomock.Any()).Return(
					[]sdk.AccAddress{
						sdk.AccAddress(testutiltx.GenerateAddress().Bytes()),
						sdk.AccAddress(testutiltx.GenerateAddress().Bytes())},
					[]keytypes.WrappedConsKey{
						testutiltx.GenerateConsensusKey(),
						testutiltx.GenerateConsensusKey(),
					},
				).Times(2)
				suite.mocks.OperatorKeeper.EXPECT().GetVotePowerForChainID(gomock.Any(), gomock.Any(), gomock.Any()).Return([]int64{100, 200}, nil)
				suite.mocks.OperatorKeeper.EXPECT().GetVotePowerForChainID(gomock.Any(), gomock.Any(), gomock.Any()).Return([]int64{}, fmt.Errorf("error getting vote power"))
				suite.mocks.ClientKeeper.EXPECT().CreateClient(gomock.Any(), gomock.Any(), gomock.Any()).Return("clientID1", nil)
				suite.mocks.EpochsKeeper.EXPECT().GetEpochInfo(gomock.Any(), gomock.Any()).Return(epochstypes.EpochInfo{CurrentEpoch: 1}, true)
				suite.mocks.AVSKeeper.EXPECT().IsAVSByChainID(gomock.Any(), gomock.Any()).Return(true, testutiltx.GenerateAddress().String())
				suite.mocks.AVSKeeper.EXPECT().DeleteAVSInfo(gomock.Any(), gomock.Any()).Return(nil)
			},
			epochIdentifier: "month",
			epochNumber:     1,
		},
		{
			name: "No chains to activate",
			setupMocks: func() {
				// No pending chains, so no mocks needed
			},
			epochIdentifier: "year",
			epochNumber:     1,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // Reset state for each test case
			if tc.setupMocks != nil {
				tc.setupMocks()
			}

			suite.keeper.ActivateScheduledChains(suite.ctx, tc.epochIdentifier, tc.epochNumber)

			// Verify pending chains are cleared
			pendingChains := suite.keeper.GetPendingSubChains(suite.ctx, tc.epochIdentifier, uint64(tc.epochNumber))
			suite.Empty(pendingChains.List, "Pending chains should be cleared")
		})
	}
}
