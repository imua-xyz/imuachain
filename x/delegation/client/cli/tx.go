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
		Use:   "update-params <instant-undelegation-penalty> <max-undelegation-completions>",
		Short: "update the parameters of the delegation module",
		Long: `Update module parameters (governance authority only).
Arguments:
  - instant-undelegation-penalty: penalty (in basis points) for instant undelegation
  - max-undelegation-completions: max undelegations completed per block (0 = no limit)`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			sender := clientCtx.GetFromAddress()
			instantPenalty, err := strconv.ParseUint(args[0], 10, 32)
			if err != nil {
				return err
			}

			maxUndelegationCompletions, err := strconv.ParseUint(args[1], 10, 32)
			if err != nil {
				return err
			}

			msg := &delegationtype.MsgUpdateParams{
				Authority: sender.String(),
				Params: delegationtype.Params{
					InstantUndelegationPenalty: uint32(instantPenalty),
					MaxUndelegationCompletions: uint32(maxUndelegationCompletions),
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
