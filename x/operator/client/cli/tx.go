package cli

import (
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"

	errorsmod "cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingcli "github.com/cosmos/cosmos-sdk/x/staking/client/cli"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/imua-xyz/imuachain/x/operator/types"
)

const (
	FlagClientChainData           = "client-chain-data"
	FlagDisableRewardsCompounding = "disable-rewards-compounding"
)

// NewTxCmd returns a root CLI command handler for deposit commands
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Operator transaction subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		CmdRegisterOperator(),
		CmdOptIntoAVS(),
		CmdOptOutOfAVS(),
		// TODO: while the operator module is storing the consensus keys for now
		// are they really a property of the operator or of the respective AVS?
		// operator vs dogfood vs appchain coordinator
		CmdSetConsKey(),
		CmdEditOperator(),
		CmdUpdateRewardCompoundingFlag(),
		CmdUpdateCommissionRate(),
		CmdUpdateParams(),
	)
	return txCmd
}

func flagSetDescriptionCreate(fs *flag.FlagSet) {
	fs.String(stakingcli.FlagMoniker, "", "The operator's name")
	fs.String(stakingcli.FlagIdentity, "", "The optional identity signature (ex. UPort or Keybase)")
	fs.String(stakingcli.FlagWebsite, "", "The operator's (optional) website")
	fs.String(stakingcli.FlagSecurityContact, "", "The operator's (optional) security contact email")
	fs.String(stakingcli.FlagDetails, "", "The operator's (optional) details")
}

func flagSetDescriptionEdit(fs *flag.FlagSet) {
	fs.String(stakingcli.FlagEditMoniker, stakingtypes.DoNotModifyDesc, "The operator's name")
	fs.String(stakingcli.FlagIdentity, stakingtypes.DoNotModifyDesc, "The (optional) identity signature (ex. UPort or Keybase)")
	fs.String(stakingcli.FlagWebsite, stakingtypes.DoNotModifyDesc, "The operator's (optional) website")
	fs.String(stakingcli.FlagSecurityContact, stakingtypes.DoNotModifyDesc, "The operator's (optional) security contact email")
	fs.String(stakingcli.FlagDetails, stakingtypes.DoNotModifyDesc, "The operator's (optional) details")
}

// CmdRegisterOperator returns a CLI command handler for creating a RegisterOperatorReq
// transaction.
func CmdRegisterOperator() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register-operator",
		Short: "register to become an operator",
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			txf, err := tx.NewFactoryCLI(clientCtx, cmd.Flags())
			if err != nil {
				return err
			}

			msg, err := newBuildRegisterOperatorMsg(clientCtx, cmd.Flags())
			if err != nil {
				return err
			}

			// this calls ValidateBasic internally so we don't need to do that.
			return tx.GenerateOrBroadcastTxWithFactory(clientCtx, txf, msg)
		},
	}

	f := cmd.Flags()
	// clientChainLzID:ClientChainEarningsAddr
	f.StringArray(
		FlagClientChainData, []string{}, "The client chain's address to receive earnings; "+
			"can be supplied multiple times. "+
			"Format: <client-chain-id>:<client-chain-earnings-addr>",
	)
	flagSetDescriptionCreate(f)
	f.Bool(FlagDisableRewardsCompounding, false, "indicate whether to disable the compounding of unclaimed rewards")
	f.AddFlagSet(stakingcli.FlagSetCommissionCreate())

	// required flags
	// name of the operator
	_ = cmd.MarkFlagRequired(stakingcli.FlagMoniker)

	// transaction level flags from the SDK
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func newBuildRegisterOperatorMsg(
	clientCtx client.Context, fs *flag.FlagSet,
) (*types.RegisterOperatorReq, error) {
	sender := clientCtx.GetFromAddress()
	moniker, _ := fs.GetString(stakingcli.FlagMoniker)
	identity, _ := fs.GetString(stakingcli.FlagIdentity)
	website, _ := fs.GetString(stakingcli.FlagWebsite)
	security, _ := fs.GetString(stakingcli.FlagSecurityContact)
	details, _ := fs.GetString(stakingcli.FlagDetails)
	description := stakingtypes.NewDescription(
		moniker,
		identity,
		website,
		security,
		details,
	)
	msg := &types.RegisterOperatorReq{
		Info: &types.OperatorInfo{
			OperatorAddr: sender.String(),
			Description:  description,
		},
	}
	clientChainEarningAddress := &types.ClientChainEarningAddrList{}
	// #nosec G703
	ccData, _ := fs.GetStringArray(FlagClientChainData)
	clientChainEarningAddress.EarningInfoList = make(
		[]*types.ClientChainEarningAddrInfo, len(ccData),
	)
	for i, arg := range ccData {
		strList := strings.Split(arg, ":")
		if len(strList) != 2 {
			return nil, errorsmod.Wrapf(
				types.ErrCliCmdInputArg, "the error input arg is:%s", arg,
			)
		}
		// note that this is not the hex value but the decimal number.
		clientChainLzID, err := strconv.ParseUint(strList[0], 10, 64)
		if err != nil {
			return nil, errorsmod.Wrapf(
				types.ErrCliCmdInputArg, "the error input arg is:%s", arg,
			)
		}
		clientChainEarningAddress.EarningInfoList[i] = &types.ClientChainEarningAddrInfo{
			LzClientChainID: clientChainLzID, ClientChainEarningAddr: strList[1],
		}
	}
	msg.Info.ClientChainEarningsAddr = clientChainEarningAddress
	// get the initial commission parameters
	// #nosec G703
	rateStr, _ := fs.GetString(stakingcli.FlagCommissionRate)
	// #nosec G703
	maxRateStr, _ := fs.GetString(stakingcli.FlagCommissionMaxRate)
	// #nosec G703
	maxChangeRateStr, _ := fs.GetString(stakingcli.FlagCommissionMaxChangeRate)
	commission, err := buildCommission(rateStr, maxRateStr, maxChangeRateStr)
	if err != nil {
		return nil, err
	}
	msg.Info.Commission = commission

	disableRewardsCompounding, _ := fs.GetBool(FlagDisableRewardsCompounding)
	msg.Info.DisableCompoundRewards = disableRewardsCompounding
	return msg, nil
}

func buildCommission(rateStr, maxRateStr, maxChangeRateStr string) (
	commission stakingtypes.Commission, err error,
) {
	if rateStr == "" || maxRateStr == "" || maxChangeRateStr == "" {
		return commission, errorsmod.Wrap(
			types.ErrCliCmdInputArg, "must specify all validator commission parameters",
		)
	}

	rate, err := sdk.NewDecFromStr(rateStr)
	if err != nil {
		return commission, err
	}

	maxRate, err := sdk.NewDecFromStr(maxRateStr)
	if err != nil {
		return commission, err
	}

	maxChangeRate, err := sdk.NewDecFromStr(maxChangeRateStr)
	if err != nil {
		return commission, err
	}

	commission = stakingtypes.NewCommission(rate, maxRate, maxChangeRate)

	return commission, nil
}

// CmdOptIntoAVS returns a CLI command handler for creating a OptIntoAVSReq transaction.
func CmdOptIntoAVS() *cobra.Command {
	cmd := &cobra.Command{
		// square brackets are optional while angle brackets are required arguments.
		Use:     "opt-into-avs <avs-address> [public-key-in-JSON-format]",
		Short:   "opt into an AVS by specifying its address, with an optional public key",
		Example: "imua tx operator opt-into-avs 0x0000000000000000000000000000000000000000 $(imuad tendermint show-validator)",
		Args:    cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			msg := &types.OptIntoAVSReq{
				FromAddress: clientCtx.GetFromAddress().String(),
				AvsAddress:  args[0],
			}
			if len(args) == 2 {
				msg.PublicKeyJSON = args[1]
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CmdOptOutOfAVS returns a CLI command handler for creating a OptOutOfAVSReq transaction.
func CmdOptOutOfAVS() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "opt-out-of-avs <avs-address>",
		Short: "opt out of an AVS by specifying its address",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			msg := &types.OptOutOfAVSReq{
				FromAddress: clientCtx.GetFromAddress().String(),
				AvsAddress:  args[0],
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CmdSetConsKey returns a CLI command handler for creating a SetConsKeyReq transaction.
func CmdSetConsKey() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-cons-key <chain-id> <public-key-in-JSON>",
		Short: "set the consensus key for a chain",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			msg := &types.SetConsKeyReq{
				Address:       clientCtx.GetFromAddress().String(),
				AvsAddress:    args[0],
				PublicKeyJSON: args[1],
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CmdEditOperator returns a CLI command handler for creating a EditOperatorReq transaction.
func CmdEditOperator() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit-operator",
		Short: "edit the description info of an operator",
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			moniker, _ := cmd.Flags().GetString(stakingcli.FlagEditMoniker)
			identity, _ := cmd.Flags().GetString(stakingcli.FlagIdentity)
			website, _ := cmd.Flags().GetString(stakingcli.FlagWebsite)
			security, _ := cmd.Flags().GetString(stakingcli.FlagSecurityContact)
			details, _ := cmd.Flags().GetString(stakingcli.FlagDetails)
			description := stakingtypes.NewDescription(moniker, identity, website, security, details)
			msg := &types.EditOperatorReq{
				Address:     clientCtx.GetFromAddress().String(),
				Description: description,
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	f := cmd.Flags()
	flagSetDescriptionEdit(f)

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CmdUpdateRewardCompoundingFlag returns a CLI command handler for creating a UpdateRewardCompoundingFlagReq transaction.
func CmdUpdateRewardCompoundingFlag() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-reward-compounding-flag <disable-rewards-compounding>",
		Short: "update the reward compounding flag of an operator",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			disableRewardsCompounding, err := strconv.ParseBool(args[0])
			if err != nil {
				return err
			}
			msg := &types.UpdateRewardCompoundingFlagReq{
				Address:                clientCtx.GetFromAddress().String(),
				DisableCompoundRewards: disableRewardsCompounding,
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CmdUpdateCommissionRate returns a CLI command handler for creating a UpdateCommissionRateReq transaction.
func CmdUpdateCommissionRate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-commission-rate <operator-address> <commission-rate>",
		Short: "update the commission rate of an operator",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			commissionRate, err := sdk.NewDecFromStr(args[1])
			if err != nil {
				return err
			}
			// validate the commission rate
			if commissionRate.IsNegative() {
				return errorsmod.Wrap(types.ErrCliCmdInputArg, "commission rate cannot be negative")
			}
			// 0 may also be supported, if minCommissionRate is 0
			// but that is stateful and we do not have access to state here.
			msg := &types.UpdateCommissionRateReq{
				Address:        clientCtx.GetFromAddress().String(),
				CommissionRate: commissionRate,
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CmdUpdateParams returns a CLI command handler for creating a MsgUpdateParams transaction.
func CmdUpdateParams() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-params <min-commission-rate> <min-commission-update-interval>",
		Short: "update the parameters of the operator module",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			minCommissionRate, err := sdk.NewDecFromStr(args[0])
			if err != nil {
				return err
			}
			minCommissionUpdateInterval, err := time.ParseDuration(args[1])
			if err != nil {
				return err
			}
			msg := &types.MsgUpdateParams{
				Authority: clientCtx.GetFromAddress().String(),
				Params: types.Params{
					MinCommissionRate:           minCommissionRate,
					MinCommissionUpdateInterval: minCommissionUpdateInterval,
				},
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
