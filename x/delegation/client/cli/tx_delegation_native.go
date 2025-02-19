package cli

import (
	"errors"

	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"

	"github.com/imua-xyz/imuachain/x/delegation/types"
)

func CmdDelegate() *cobra.Command {
	cmd := &cobra.Command{
		// TODO: only support native token for now
		Use:   "delegate asset-id operator amount",
		Short: "Broadcast a transaction to delegate amount of native token to the operator",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			assetID, operatorAddrStr, amount, err := parseArgs(args)
			if err != nil {
				return err
			}
			msg := types.NewMsgDelegation(assetID, clientCtx.GetFromAddress().String(), []types.KeyValue{{Key: operatorAddrStr, Value: &types.ValueField{Amount: amount}}})

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdUndelegate() *cobra.Command {
	cmd := &cobra.Command{
		// TODO: only support native token for now
		Use:   "undelegate asset-id operator amount",
		Short: "Broadcast a transaction to undelegate amount of native token from the operator",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			assetID, operatorAddrStr, amount, err := parseArgs(args)
			if err != nil {
				return err
			}

			msg := types.NewMsgUndelegation(assetID, clientCtx.GetFromAddress().String(), []types.KeyValue{{Key: operatorAddrStr, Value: &types.ValueField{Amount: amount}}})

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func parseArgs(args []string) (string, string, sdkmath.Int, error) {
	if len(args) != 3 {
		return "", "", sdkmath.ZeroInt(), errors.New("3 arguments needed")
	}

	assetID := args[0]
	operatorAddrStr := args[1]

	amount, ok := sdkmath.NewIntFromString(args[2])
	if !ok || amount.LTE(sdkmath.ZeroInt()) {
		return "", "", sdkmath.ZeroInt(), errors.New("amount invalid")
	}

	return assetID, operatorAddrStr, amount, nil
}
