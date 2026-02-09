package cli

import (
	"fmt"
	"strconv"

	"github.com/imua-xyz/imuachain/utils"
	assetstypes "github.com/imua-xyz/imuachain/x/assets/types"

	sdkmath "cosmossdk.io/math"

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
	cmd.AddCommand(
		CmdUpdateParams(),
		CmdWithdrawDogfoodCommission(),
		CmdClaimAndWithdrawDogfoodReward(),
		CmdUpdateStakerRewardParams(),
		CmdUndelegateReward(),
	)
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

func CmdWithdrawDogfoodCommission() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "withdraw-dogfood-commission",
		Short:   "withdraw the dogfood commission for an operator",
		Example: "imua tx feedistribution withdraw-dogfood-commission",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			msg := &types.MsgWithdrawDogfoodCommission{
				FromAddress: clientCtx.GetFromAddress().String(),
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdClaimAndWithdrawDogfoodReward() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "claim-and-withdraw-dogfood-reward",
		Short:   "claim and withdraw the dogfood reward for a staker from imua chain",
		Example: "imua tx feedistribution claim-and-withdraw-dogfood-reward amount(0 to withdraw all rewards)",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			msg := &types.MsgClaimAndWithdrawDogfoodReward{
				FromAddress: clientCtx.GetFromAddress().String(),
			}
			amount, ok := sdkmath.NewIntFromString(args[0])
			if !ok || amount.IsNegative() {
				return types.ErrInvalidCliCmdArg.Wrapf("invalid input amount: %s", args[0])
			}
			msg.Amount = amount
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdUpdateStakerRewardParams() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "update-staker-reward-params",
		Short:   "set or update the reward parameters for the staker",
		Example: "imua tx feedistribution update-staker-reward-params true im18cggcpvwspnd5c6ny8wrqxpffj5zmhkl3agtrj",
		Args:    cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			redelegateReward, err := strconv.ParseBool(args[0])
			if err != nil {
				return err
			}
			msg := &types.MsgUpdateStakerRewardParams{
				FromAddress: clientCtx.GetFromAddress().String(),
				RewardParams: types.StakerRewardParams{
					RedelegateReward: redelegateReward,
				},
			}
			if redelegateReward {
				if len(args) < 2 {
					return types.ErrInvalidCliCmdArg.Wrap("missing redelegate operator address when redelegateReward=true")
				}
				_, err = sdk.AccAddressFromBech32(args[1])
				if err != nil {
					return err
				}
				msg.RewardParams.RedelegateOperatorAddr = args[1]
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdUndelegateReward() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "undelegate-reward asset-id operator amount optional(--instant-unbonding true)",
		Short:   "undelegate reward for the staker on IMUA chain",
		Example: "imua tx feedistribution undelegate-reward 0xdac17f958d2ee523a2206206994597c13d831ec7_0x65 im18cggcpvwspnd5c6ny8wrqxpffj5zmhkl3agtrj 10",
		Args:    cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			instantUnbonding, err := cmd.Flags().GetBool(utils.FlagInstantUnbonding)
			if err != nil {
				return err
			}
			assetID := args[0]
			_, _, err = assetstypes.ValidateID(assetID, false, false)
			if err != nil {
				return types.ErrInvalidCliCmdArg.Wrapf("invalid assetID:%s,err:%s", assetID, err)
			}
			_, err = sdk.AccAddressFromBech32(args[1])
			if err != nil {
				return types.ErrInvalidCliCmdArg.Wrapf("invalid operator address:%s,err:%s", args[1], err)
			}
			amount, ok := sdkmath.NewIntFromString(args[2])
			if !ok || !amount.IsPositive() {
				return types.ErrInvalidCliCmdArg.Wrapf("invalid amount:%s", args[2])
			}

			msg := &types.MsgUndelegateReward{
				FromAddress:      clientCtx.GetFromAddress().String(),
				AssetId:          assetID,
				OperatorAddr:     args[1],
				Amount:           amount,
				InstantUnbonding: instantUnbonding,
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	cmd.Flags().Bool(utils.FlagInstantUnbonding, false, "indicate whether it's an instant undelegation")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
