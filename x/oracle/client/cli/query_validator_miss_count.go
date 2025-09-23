package cli

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/imua-xyz/imuachain/x/oracle/types"
	"github.com/spf13/cobra"
)

func CmdQueryValidatorMissCount() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validator-miss-count [validator-address]",
		Short: "Query the miss count of a validator",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			if _, err := sdk.ConsAddressFromBech32(args[0]); err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.ValidatorMissCount(cmd.Context(), &types.QueryValidatorMissCountRequest{
				Validator: args[0],
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
