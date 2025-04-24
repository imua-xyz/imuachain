package cli

import (
	"context"
	"fmt"
	"strconv"

	"github.com/cosmos/gogoproto/proto"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	assetstypes "github.com/imua-xyz/imuachain/x/assets/types"
	"golang.org/x/xerrors"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/imua-xyz/imuachain/x/feedistribution/types"
	"github.com/spf13/cobra"
)

// GetQueryCmd returns the cli query commands for this module
func GetQueryCmd(_ string) *cobra.Command {
	// Group fee distribution queries under a subcommand
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      fmt.Sprintf("Querying commands for the %s module", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		CmdQueryParams(),
		CmdQueryAVSRewardAsset(),
		CmdQueryRewardAssetsByAVS(),
		CmdQueryAVSRewardParam(),
		CmdQueryAVSCommunityPool(),
		CmdQueryAVSRewardDistribution(),
		CmdQueryOperatorAccumulatedCommission(),
		CmdQueryOperatorCurrentRewards(),
		CmdQueryOperatorHistoricalRewards(),
		CmdQueryOperatorOutstandingRewards(),
		CmdQueryOperatorSlashEvent(),
		CmdQueryStakerOutstandingRewards(),
		CmdQueryStakeChangeDelegations(),
		CmdQueryDelegationStartingInfo(),
		CmdQueryStakerUnclaimedRewards(),
	)
	return cmd
}

func newAVSCmd(
	use, short, long string,
	queryFunc func(types.QueryClient, context.Context, *types.AVSRequest) (proto.Message, error),
) *cobra.Command {
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Long:  long,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !common.IsHexAddress(args[0]) {
				return xerrors.Errorf("invalid avs address, err: %s", types.ErrInvalidCliCmdArg)
			}

			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)
			req := &types.AVSRequest{
				Avs: args[0], // the RPC is case-insensitive with respect to AVSAddr.
			}
			res, err := queryFunc(queryClient, context.Background(), req)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func newOperatorAVSCmd(
	use, short, long string,
	queryFunc func(types.QueryClient, context.Context, *types.OperatorAVSRequest) (proto.Message, error),
) *cobra.Command {
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Long:  long,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := sdk.AccAddressFromBech32(args[0])
			if err != nil {
				return xerrors.Errorf("invalid operator address, err: %s", err.Error())
			}
			if !common.IsHexAddress(args[1]) {
				return xerrors.Errorf("invalid avs address, err: %s", types.ErrInvalidCliCmdArg)
			}

			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)
			req := &types.OperatorAVSRequest{
				Operator: args[0],
				Avs:      args[1], // the RPC will convert to lowercase
			}
			res, err := queryFunc(queryClient, context.Background(), req)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func newOperatorAssetEpochUint64Cmd(
	use, short, long string,
	makeRequest func(string, string, string, uint64) proto.Message,
	queryFunc func(types.QueryClient, context.Context, proto.Message) (proto.Message, error),
) *cobra.Command {
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Long:  long,
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Validate operator
			if _, err := sdk.AccAddressFromBech32(args[0]); err != nil {
				return xerrors.Errorf("invalid operator address, err: %s", err.Error())
			}
			// Validate asset ID
			if _, _, err := assetstypes.ValidateID(args[1], false, false); err != nil {
				return errorsmod.Wrap(types.ErrInvalidCliCmdArg, err.Error())
			}
			// Parse final uint64 param (period or epochNumber)
			number, err := strconv.ParseUint(args[3], 10, 64)
			if err != nil {
				return errorsmod.Wrap(types.ErrInvalidCliCmdArg, err.Error())
			}

			// Setup query
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)
			req := makeRequest(args[0], args[1], args[2], number)
			res, err := queryFunc(queryClient, context.Background(), req)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func newEpochOperatorAssetCmd(
	use, short, long string,
	makeRequest func(string, string, string) proto.Message,
	queryFunc func(types.QueryClient, context.Context, proto.Message) (proto.Message, error),
) *cobra.Command {
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Long:  long,
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := sdk.AccAddressFromBech32(args[1]); err != nil {
				return xerrors.Errorf("invalid operator address, err: %s", err.Error())
			}
			if _, _, err := assetstypes.ValidateID(args[2], false, false); err != nil {
				return errorsmod.Wrap(types.ErrInvalidCliCmdArg, err.Error())
			}

			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)
			req := makeRequest(args[0], args[1], args[2])
			res, err := queryFunc(queryClient, context.Background(), req)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func CmdQueryParams() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "show module params",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.Params(cmd.Context(), &types.QueryParamsRequest{})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func CmdQueryAVSRewardAsset() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "avs-reward-asset [avsAddr] [assetID]",
		Short: "Get avs reward asset",
		Long:  "Get avs reward asset by the address and assetID",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !common.IsHexAddress(args[0]) {
				return xerrors.Errorf("invalid avs address,err:%s", types.ErrInvalidCliCmdArg)
			}
			if _, _, err := assetstypes.ValidateID(args[1], false, false); err != nil {
				return errorsmod.Wrap(types.ErrInvalidCliCmdArg, err.Error())
			}

			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)
			req := &types.QueryAVSRewardAssetRequest{
				Avs:     args[0], // the RPC will convert to lowercase
				AssetId: args[1], // the RPC will convert to lowercase
			}
			res, err := queryClient.AVSRewardAsset(context.Background(), req)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func CmdQueryRewardAssetsByAVS() *cobra.Command {
	return newAVSCmd(
		"reward-assets-by-avs [avsAddr]",
		"all reward assets for the avs",
		"Get all reward assets for the avs",
		func(q types.QueryClient, ctx context.Context, req *types.AVSRequest) (proto.Message, error) {
			return q.RewardAssetsByAVS(ctx, req)
		},
	)
}

func CmdQueryAVSRewardParam() *cobra.Command {
	return newAVSCmd(
		"avs-reward-param [avsAddr]",
		"get the reward parameter for the avs",
		"get the reward parameter for the avs",
		func(q types.QueryClient, ctx context.Context, req *types.AVSRequest) (proto.Message, error) {
			return q.AVSRewardParam(ctx, req)
		},
	)
}

func CmdQueryAVSCommunityPool() *cobra.Command {
	return newAVSCmd(
		"avs-community-pool [avsAddr]",
		"get the community fee pool for the avs",
		"get the community fee pool for the avs",
		func(q types.QueryClient, ctx context.Context, req *types.AVSRequest) (proto.Message, error) {
			return q.AVSCommunityPool(ctx, req)
		},
	)
}

func CmdQueryAVSRewardDistribution() *cobra.Command {
	return newAVSCmd(
		"avs-reward-distribution [avsAddr]",
		"get the reward distribution for the avs",
		"get the reward distribution for the avs",
		func(q types.QueryClient, ctx context.Context, req *types.AVSRequest) (proto.Message, error) {
			return q.AVSRewardDistribution(ctx, req)
		},
	)
}

func CmdQueryOperatorOutstandingRewards() *cobra.Command {
	return newOperatorAVSCmd(
		"operator-outstanding-rewards [operator] [avsAddr]",
		"get the outstanding rewards for the operator",
		"get the outstanding rewards for the operator",
		func(q types.QueryClient, ctx context.Context, req *types.OperatorAVSRequest) (proto.Message, error) {
			return q.OperatorOutstandingRewards(ctx, req)
		},
	)
}

func CmdQueryStakerOutstandingRewards() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "staker-outstanding-rewards [stakerID] [avsAddr]",
		Short: "get the outstanding rewards for the staker",
		Long:  "get the outstanding rewards for the staker",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, _, err := assetstypes.ValidateID(args[0], false, false); err != nil {
				return errorsmod.Wrap(types.ErrInvalidCliCmdArg, err.Error())
			}
			if !common.IsHexAddress(args[1]) {
				return xerrors.Errorf("invalid avs address,err:%s", types.ErrInvalidCliCmdArg)
			}

			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)
			req := &types.QueryStakerOutstandingRewardsRequest{
				StakerId: args[0], // the RPC will convert to lowercase
				Avs:      args[1], // the RPC will convert to lowercase
			}
			res, err := queryClient.StakerOutstandingRewards(context.Background(), req)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func CmdQueryStakeChangeDelegations() *cobra.Command {
	return newEpochOperatorAssetCmd(
		"stake-change-delegations [epochIdentifier] [operator] [assetID]",
		"get the delegations whose stake has changed",
		"Get the delegations whose stake has changed in a specific epoch",
		func(epochIdentifier, operator, assetID string) proto.Message {
			return &types.QueryStakeChangeDelegationsRequest{
				EpochIdentifier: epochIdentifier,
				Operator:        operator,
				AssetId:         assetID,
			}
		},
		func(client types.QueryClient, ctx context.Context, req proto.Message) (proto.Message, error) {
			return client.StakeChangeDelegations(ctx, req.(*types.QueryStakeChangeDelegationsRequest))
		},
	)
}

func CmdQueryDelegationStartingInfo() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delegation-starting-info [stakerID] [assetID] [operator] [epochIdentifier]",
		Short: "get the starting information of a delegation",
		Long:  "get the starting information of a delegation",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, _, err := assetstypes.ValidateID(args[0], false, false); err != nil {
				return errorsmod.Wrap(types.ErrInvalidCliCmdArg, err.Error())
			}
			if _, _, err := assetstypes.ValidateID(args[1], false, false); err != nil {
				return errorsmod.Wrap(types.ErrInvalidCliCmdArg, err.Error())
			}
			_, err := sdk.AccAddressFromBech32(args[2])
			if err != nil {
				return xerrors.Errorf("invalid operator address,err:%s", err.Error())
			}

			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)
			req := &types.QueryDelegationStartingInfoRequest{
				StakerId:        args[0], // the RPC will convert to lowercase
				AssetId:         args[1], // the RPC will convert to lowercase
				Operator:        args[2],
				EpochIdentifier: args[3],
			}
			res, err := queryClient.DelegationStartingInfo(context.Background(), req)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func CmdQueryOperatorHistoricalRewards() *cobra.Command {
	return newOperatorAssetEpochUint64Cmd(
		"operator-historical-rewards [operator] [assetID] [epochIdentifier] [period]",
		"get the historical rewards for an operator",
		"get the historical rewards for an operator",
		func(operator, assetID, epochID string, period uint64) proto.Message {
			return &types.QueryOperatorHistoricalRewardsRequest{
				Operator:        operator,
				AssetId:         assetID,
				EpochIdentifier: epochID,
				Period:          period,
			}
		},
		func(q types.QueryClient, ctx context.Context, req proto.Message) (proto.Message, error) {
			return q.OperatorHistoricalRewards(ctx, req.(*types.QueryOperatorHistoricalRewardsRequest))
		},
	)
}

func CmdQueryOperatorSlashEvent() *cobra.Command {
	return newOperatorAssetEpochUint64Cmd(
		"operator-slash-event [operator] [assetID] [epochIdentifier] [epochNumber]",
		"get the the operator slash event",
		"get the the operator slash event",
		func(operator, assetID, epochID string, epochNumber uint64) proto.Message {
			return &types.QueryOperatorSlashEventRequest{
				Operator:        operator,
				AssetId:         assetID,
				EpochIdentifier: epochID,
				EpochNumber:     epochNumber,
			}
		},
		func(q types.QueryClient, ctx context.Context, req proto.Message) (proto.Message, error) {
			return q.OperatorSlashEvent(ctx, req.(*types.QueryOperatorSlashEventRequest))
		},
	)
}

func CmdQueryOperatorCurrentRewards() *cobra.Command {
	return newEpochOperatorAssetCmd(
		"operator-current-rewards [epochIdentifier] [operator] [assetID]",
		"get the current rewards for an operator",
		"Get the current rewards for an operator in a specific epoch",
		func(epochIdentifier, operator, assetID string) proto.Message {
			return &types.QueryOperatorCurrentRewardsRequest{
				EpochIdentifier: epochIdentifier,
				Operator:        operator,
				AssetId:         assetID,
			}
		},
		func(client types.QueryClient, ctx context.Context, req proto.Message) (proto.Message, error) {
			return client.OperatorCurrentRewards(ctx, req.(*types.QueryOperatorCurrentRewardsRequest))
		},
	)
}

func CmdQueryOperatorAccumulatedCommission() *cobra.Command {
	return newOperatorAVSCmd(
		"operator-accumulated-commission [operator] [avsAddr]",
		"get the historical rewards for an operator",
		"get the historical rewards for an operator",
		func(q types.QueryClient, ctx context.Context, req *types.OperatorAVSRequest) (proto.Message, error) {
			return q.OperatorAccumulatedCommission(ctx, req)
		},
	)
}

func CmdQueryStakerUnclaimedRewards() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "staker-unclaimed-rewards [stakerID]",
		Short: "get the unclaimed rewards for a staker",
		Long:  "get the unclaimed rewards for a staker",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, _, err := assetstypes.ValidateID(args[0], false, false); err != nil {
				return errorsmod.Wrap(types.ErrInvalidCliCmdArg, err.Error())
			}

			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)
			req := &types.QueryStakerUnclaimedRewardsRequest{
				StakerId: args[0], // the RPC is case-insensitive with respect to AVSAddr.
			}
			res, err := queryClient.StakerUnclaimedRewards(context.Background(), req)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
