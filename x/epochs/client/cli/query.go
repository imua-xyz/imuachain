package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/version"

	"github.com/imua-xyz/imuachain/x/epochs/types"
)

// GetQueryCmd returns the cli query commands for this module
func GetQueryCmd(string) *cobra.Command {
	cmd := &cobra.Command{
		Use: types.ModuleName,
		Short: fmt.Sprintf(
			"Querying commands for the %s module",
			types.ModuleName,
		),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(GetCmdEpochsInfos())
	cmd.AddCommand(GetCmdCurrentEpoch())
	cmd.AddCommand(GetCmdEpochInfo())

	return cmd
}

// GetCmdEpochsInfos returns a command for querying all the currently available epoch infos.
func GetCmdEpochsInfos() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "epoch-infos",
		Short: "Query all epochInfos",
		Example: strings.TrimSpace(
			fmt.Sprintf(`$ %s query %s epoch-infos`,
				version.AppName, types.ModuleName,
			),
		),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			req := &types.QueryEpochsInfoRequest{
				Pagination: pageReq,
			}

			res, err := queryClient.EpochInfos(cmd.Context(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

// GetCmdCurrentEpoch provides current epoch number, given the identifier.
func GetCmdCurrentEpoch() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "current-epoch",
		Short: "Query current epoch number for the given identifier",
		Example: strings.TrimSpace(
			fmt.Sprintf(`$ %s query %s current-epoch week`,
				version.AppName, types.ModuleName,
			),
		),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			res, err := queryClient.CurrentEpoch(cmd.Context(), &types.QueryCurrentEpochRequest{
				Identifier: args[0],
			})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

// GetCmdEpochInfo provides epoch info for the given identifier.
func GetCmdEpochInfo() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "epoch-info <identifier>",
		Short: "Query epoch info for the given identifier",
		Example: strings.TrimSpace(
			fmt.Sprintf(`$ %s query %s epoch-info week`,
				version.AppName, types.ModuleName,
			),
		),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			res, err := queryClient.EpochInfo(cmd.Context(), &types.QueryEpochInfoRequest{
				Identifier: args[0],
			})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
