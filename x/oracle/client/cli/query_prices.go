package cli

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"

	"github.com/imua-xyz/imuachain/x/oracle/types"
	"github.com/spf13/cast"
)

// func CmdListPrices() *cobra.Command {
//	cmd := &cobra.Command{
//		Use:   "list-prices",
//		Short: "list all prices",
//		RunE: func(cmd *cobra.Command, args []string) error {
//			clientCtx, err := client.GetClientQueryContext(cmd)
//			if err != nil {
//				return err
//			}
//
//			pageReq, err := client.ReadPageRequest(cmd.Flags())
//			if err != nil {
//				return err
//			}
//
//			queryClient := types.NewQueryClient(clientCtx)
//
//			params := &types.QueryAllPricesRequest{
//				Pagination: pageReq,
//			}
//
//			res, err := queryClient.PricesAll(cmd.Context(), params)
//			if err != nil {
//				return err
//			}
//
//			return clientCtx.PrintProto(res)
//		},
//	}
//
//	flags.AddPaginationFlagsToCmd(cmd, cmd.Use)
//	flags.AddQueryFlagsToCmd(cmd)
//
//	return cmd
//}

func CmdShowPrices() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show-prices [token-id]",
		Short: "shows price of the latest round for a specific token",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			argTokenID, err := cast.ToUint64E(args[0])
			if err != nil {
				return err
			}

			params := &types.QueryGetPricesRequest{
				TokenId: argTokenID,
			}

			res, err := queryClient.Prices(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func CmdShowLatestPrice() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show-latest-price [token-id]",
		Short: "shows the latest price of a specific token",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			argTokenID, err := cast.ToUint64E(args[0])
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			input := &types.QueryGetLatestPriceRequest{
				TokenId: argTokenID,
			}
			res, err := queryClient.LatestPrice(cmd.Context(), input)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
