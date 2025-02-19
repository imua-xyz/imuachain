package cli

import (
	delegationtype "github.com/imua-xyz/imuachain/x/delegation/types"
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
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
	)
	return txCmd
}
