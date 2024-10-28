package cli

import (
	"fmt"
	// "strings"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	// "github.com/cosmos/cosmos-sdk/client/flags"
	// sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/ExocoreNetwork/exocore/x/oracle/types"
)

// GetQueryCmd returns the cli query commands for this module
func GetQueryCmd(_ string) *cobra.Command {
	// Group oracle queries under a subcommand
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      fmt.Sprintf("Querying commands for the %s module", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(CmdQueryParams())
	// cmd.AddCommand(CmdListPrices())
	cmd.AddCommand(CmdShowPrices())
	cmd.AddCommand(CmdShowLatestPrice())
	cmd.AddCommand(CmdShowValidatorUpdateBlock())
	cmd.AddCommand(CmdShowIndexRecentParams())
	cmd.AddCommand(CmdShowIndexRecentMsg())
	cmd.AddCommand(CmdListRecentMsg())
	cmd.AddCommand(CmdShowRecentMsg())
	cmd.AddCommand(CmdListRecentParams())
	cmd.AddCommand(CmdShowRecentParams())
	cmd.AddCommand(CmdQueryStakerInfo())
	cmd.AddCommand(CmdQueryStakerInfos())
	cmd.AddCommand(CmdQueryStakerList())
	cmd.AddCommand(CmdQueryTokenIndexes())
	// this line is used by starport scaffolding # 1

	return cmd
}
