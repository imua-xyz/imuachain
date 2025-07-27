package cli

import (
	"fmt"
	"strings"

	types "github.com/imua-xyz/imuachain/x/immint/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	FlagMintDenom             = "mint-denom"
	FlagEpochReward           = "epoch-reward"
	FlagEpochIdentifier       = "epoch-identifier"
	FlagEnableInflationParams = "enable-inflation-params"
	FlagInflationStartTime    = "inflation-start-time"
	FlagInflationRatios       = "inflation-ratios"
)

// NewTxCmd returns a root CLI command handler for deposit commands
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "immint subcommands",
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
// Since such messages are only executed if signed by the (governance) authority, this command
// is not useful for end users, unless they are the authority.
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

			msg, err := newBuildUpdateParamsMsg(clientCtx, cmd.Flags())
			if err != nil {
				return err
			}
			// this calls ValidateBasic internally so we don't need to do that.
			return tx.GenerateOrBroadcastTxWithFactory(clientCtx, txf, msg)
		},
	}

	f := cmd.Flags()
	f.String(
		FlagMintDenom, "", "The mint denomination",
	)
	f.String(
		FlagEpochReward, "", "The amount of the mint denomination to mint, per epoch (as a string)",
	)
	f.String(
		FlagEpochIdentifier, "", "The identifier of the epoch at which it should be minted",
	)
	f.Bool(
		FlagEnableInflationParams, false, "Enable the inflation parameters",
	)
	f.Int64(
		FlagInflationStartTime, 0, "The Unix timestamp (in seconds) when inflation starts",
	)
	f.String(
		FlagInflationRatios, "", "The annual inflation ratios, example: 0.1,0.2,0.3",
	)

	// transaction level flags from the SDK
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

func newBuildUpdateParamsMsg(
	clientCtx client.Context, fs *pflag.FlagSet,
) (*types.MsgUpdateParams, error) {
	sender := clientCtx.GetFromAddress()
	// #nosec G703 // this only errors if the flag isn't defined.
	mintDenom, _ := fs.GetString(FlagMintDenom)
	// #nosec G703 // this only errors if the flag isn't defined.
	epochIdentifier, _ := fs.GetString(FlagEpochIdentifier)
	// #nosec G703 // this only errors if the flag isn't defined.
	epochRewardStr, _ := fs.GetString(FlagEpochReward)
	res, ok := sdk.NewIntFromString(epochRewardStr)
	if !ok {
		// if the string is invalid, default to nil.
		// the `nil` will be overridden by the current value during
		// message execution.
		// setting 0 here would be bad, since a value of 0
		// is considered valid.
		res = sdkmath.Int{}
	}
	// #nosec G703 // this only errors if the flag isn't defined.
	enableInflationParams, _ := fs.GetBool(FlagEnableInflationParams)
	// #nosec G703 // this only errors if the flag isn't defined.
	inflationStartTime, _ := fs.GetInt64(FlagInflationStartTime)
	if inflationStartTime < 0 {
		return nil, fmt.Errorf("negative start time: %d", inflationStartTime)
	}
	// #nosec G703 // this only errors if the flag isn't defined.
	inflationRatiosStr, _ := fs.GetString(FlagInflationRatios)
	inflationRatiosStrList := strings.Split(inflationRatiosStr, ",")
	inflationRatiosDecList := make([]sdk.Dec, len(inflationRatiosStrList))
	for i, ratioStr := range inflationRatiosStrList {
		ratio, err := sdk.NewDecFromStr(ratioStr)
		if err != nil {
			return nil, fmt.Errorf("invalid inflation ratio %s at index %d in input %s", ratioStr, i, inflationRatiosStr)
		}
		inflationRatiosDecList[i] = ratio
	}

	msg := &types.MsgUpdateParams{
		Authority: sender.String(),
		Params: types.Params{
			MintDenom:       mintDenom,
			EpochReward:     res,
			EpochIdentifier: epochIdentifier,
			InflationParams: types.InflationParams{
				Enable:          enableInflationParams,
				StartTime:       inflationStartTime,
				AnnualInflation: inflationRatiosDecList,
			},
		},
	}
	return msg, nil
}
