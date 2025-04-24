package cli

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/imua-xyz/imuachain/x/feedistribution/types"
	"github.com/spf13/cobra"
)

// GetTxCmd returns the transaction commands for this module
func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      fmt.Sprintf("%s transactions subcommands", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	cmd.AddCommand(CmdUpdateParams())
	return cmd
}

// CmdUpdateParams is to update Params for distribution module
func CmdUpdateParams() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "update-params [community-tax]",
		Short:   "update params of the distribution module",
		Example: "imua tx feedistribution update-params 0.1",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cliCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			sender := cliCtx.GetFromAddress()
			communityTax, err := sdk.NewDecFromStr(args[0])
			if err != nil {
				return fmt.Errorf("invalid community tax:%s,err:%s", args[0], err)
			}
			msg := &types.MsgUpdateParams{
				Authority: sender.String(),
				Params: types.Params{
					CommunityTax: communityTax,
				},
			}
			// this calls ValidateBasic internally so we don't need to do that.
			return tx.GenerateOrBroadcastTxCLI(cliCtx, cmd.Flags(), msg)
		},
	}

	// transaction level flags from the SDK
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
