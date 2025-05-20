package cli

import (
	"strconv"

	delegationtype "github.com/imua-xyz/imuachain/x/delegation/types"
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
)

// NewTxCmd returns a root CLI command handler for deposit commands
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        delegationtype.ModuleName,
		Short:                      "delegation subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		// add tx commands
		CmdDelegate(),
		CmdUndelegate(),
		CmdUpdateParams(),
	)
	return txCmd
}

// CmdUpdateParams returns a CLI command handler for creating a MsgUpdateParams transaction.
// Since such messages are only executed if signed by the (governance) authority, this command
// is not useful for end users, unless they are the authority.
func CmdUpdateParams() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-instant-undelegation-params",
		Short: "update the instant undelegation penalty parameters of the module",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			sender := clientCtx.GetFromAddress()
			instantPenalty, err := strconv.ParseInt(args[0], 10, 32)
			if err != nil {
				return err
			}

			msg := &delegationtype.MsgUpdateParams{
				Authority: sender.String(),
				Params: delegationtype.Params{
					InstantUndelegationPenalty: uint32(instantPenalty),
				},
			}

			// this calls ValidateBasic internally so we don't need to do that.
			txf, err := tx.NewFactoryCLI(clientCtx, cmd.Flags())
			if err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxWithFactory(clientCtx, txf, msg)
		},
	}

	// transaction level flags from the SDK
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
