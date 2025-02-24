package cli

import (
	"fmt"
	"strconv"

	epochsTypes "github.com/imua-xyz/imuachain/x/epochs/types"

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
		Use:   "update-params",
		Short: "update params-update msg of the module",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			cliCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			sender := cliCtx.GetFromAddress()
			communityInteger, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return err
			}
			if err := epochsTypes.ValidateEpochIdentifierString(args[0]); err != nil {
				return err
			}
			communityPrecise, err := strconv.ParseInt(args[2], 10, 64)
			if err != nil {
				return err
			}
			communityTax := sdk.NewDecWithPrec(communityInteger, communityPrecise)
			msg := &types.MsgUpdateParams{
				Authority: sender.String(),
				Params: types.Params{
					EpochIdentifier: args[0],
					CommunityTax:    communityTax,
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
