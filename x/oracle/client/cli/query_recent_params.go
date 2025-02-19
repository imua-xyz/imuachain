//nolint:dupl
package cli

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"

	"github.com/imua-xyz/imuachain/x/oracle/types"
	"github.com/spf13/cast"
)

func CmdListRecentParams() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-recent-params",
		Short: "list all recentParams",
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

			params := &types.QueryAllRecentParamsRequest{
				Pagination: pageReq,
			}

			res, err := queryClient.RecentParamsAll(cmd.Context(), params)
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

func CmdShowRecentParams() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show-recent-params [block]",
		Short: "shows a recentParams",
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

			params := &types.QueryGetRecentParamsRequest{
				Block: argBlock,
			}

			res, err := queryClient.RecentParams(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
