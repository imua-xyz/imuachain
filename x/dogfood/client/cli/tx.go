package cli

import (
	sdkmath "cosmossdk.io/math"

	types "github.com/imua-xyz/imuachain/x/dogfood/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	FlagEpochsUntilUnbonded = "epochs-until-unbonded"
	FlagEpochIdentifier     = "epoch-identifier"
	FlagMaxValidators       = "max-validators"
	FlagHistoricalEntries   = "historical-entries"
	FlagAssetIDs            = "asset-ids"
	FlagMinSelfDelegation   = "min-self-delegation"
)

// NewTxCmd returns a root CLI command handler for dogfood commands
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "dogfood subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		CmdUpdateParams(),
	)
	return txCmd
}

// CmdUpdateParams returns a CLI command handler for creating a MsgUpdateParams transaction.
func CmdUpdateParams() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-params",
		Short: "update the parameters of the module",
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			txf, err := tx.NewFactoryCLI(clientCtx, cmd.Flags())
			if err != nil {
				return err
			}

			msg := newBuildUpdateParamsMsg(clientCtx, cmd.Flags())

			// this calls ValidateBasic internally so we don't need to do that.
			return tx.GenerateOrBroadcastTxWithFactory(clientCtx, txf, msg)
		},
	}

	f := cmd.Flags()
	f.Uint32(
		FlagEpochsUntilUnbonded, 0, "The number of epochs until unbonding",
	)
	f.String(
		FlagEpochIdentifier, "", "The identifier of the epoch at which the validator set changes",
	)
	f.Uint32(
		FlagMaxValidators, 0, "The maximum number of validators",
	)
	f.Uint32(
		FlagHistoricalEntries, 0, "The number of historical entries stored for IBC",
	)
	f.StringSlice(
		FlagAssetIDs, []string{}, "The asset ids to consider for the module",
	)
	f.Uint64(
		FlagMinSelfDelegation, 0, "The minimum self delegation amount (must be supplied even if unchanged)",
	)
	// #nosec G703 // this only errors if the flag isn't defined.
	_ = cmd.MarkFlagRequired(FlagMinSelfDelegation)

	// transaction level flags from the SDK
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

func newBuildUpdateParamsMsg(
	clientCtx client.Context, fs *pflag.FlagSet,
) *types.MsgUpdateParams {
	sender := clientCtx.GetFromAddress()
	// #nosec G703 // this only errors if the flag isn't defined.
	epochs, _ := fs.GetUint32(FlagEpochsUntilUnbonded)
	// #nosec G703 // this only errors if the flag isn't defined.
	epochIdentifier, _ := fs.GetString(FlagEpochIdentifier)
	// #nosec G703 // this only errors if the flag isn't defined.
	maxVals, _ := fs.GetUint32(FlagMaxValidators)
	// #nosec G703 // this only errors if the flag isn't defined.
	historicalEntries, _ := fs.GetUint32(FlagHistoricalEntries)
	// #nosec G703 // this only errors if the flag isn't defined.
	assetIDs, _ := fs.GetStringSlice(FlagAssetIDs)
	// #nosec G703 // this only errors if the flag isn't defined.
	minSelfDelegation, _ := fs.GetUint64(FlagMinSelfDelegation)
	msg := &types.MsgUpdateParams{
		Authority: sender.String(),
		Params: types.Params{
			EpochsUntilUnbonded: epochs,
			EpochIdentifier:     epochIdentifier,
			MaxValidators:       maxVals,
			HistoricalEntries:   historicalEntries,
			AssetIDs:            assetIDs,
			MinSelfDelegation:   sdkmath.NewIntFromUint64(minSelfDelegation),
		},
	}
	return msg
}
