//nolint:dupl
package cli

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"

	"github.com/imua-xyz/imuachain/x/oracle/types"
	"github.com/spf13/cast"
)

func CmdListRecentMsg() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-recent-msg",
		Short: "list all recentMsg",
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryAllRecentMsgRequest{
				Pagination: pageReq,
			}

			res, err := queryClient.RecentMsgAll(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddPaginationFlagsToCmd(cmd, cmd.Use)
	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func CmdShowRecentMsg() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show-recent-msg [block]",
		Short: "shows a recentMsg",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			argBlock, err := cast.ToUint64E(args[0])
			if err != nil {
				return err
			}

			params := &types.QueryGetRecentMsgRequest{
				Block: argBlock,
			}

			res, err := queryClient.RecentMsg(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
