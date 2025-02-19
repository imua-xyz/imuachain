package cli

import (
	"fmt"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/imua-xyz/imuachain/x/dogfood/types"
)

// GetQueryCmd returns the cli query commands for this module
func GetQueryCmd(string) *cobra.Command {
	// Group dogfood queries under a subcommand
	cmd := &cobra.Command{
		Use: types.ModuleName,
		Short: fmt.Sprintf(
			"Querying commands for the %s module",
			types.ModuleName,
		),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(CmdQueryParams())
	cmd.AddCommand(CmdQueryOptOutsToFinish())
	cmd.AddCommand(CmdQueryOperatorOptOutFinishEpoch())
	cmd.AddCommand(CmdQueryUndelegationsToMature())
	cmd.AddCommand(CmdQueryUndelegationMaturityEpoch())
	cmd.AddCommand(CmdQueryValidator())
	cmd.AddCommand(CmdQueryValidators())

	return cmd
}

func CmdQueryParams() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "shows the parameters of the module",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			res, err := queryClient.Params(cmd.Context(), &types.QueryParamsRequest{})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func CmdQueryOptOutsToFinish() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "opt-outs-to-finish [epoch]",
		Short: "shows the operator addresses whose opt out matures at the provided epoch",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)
			epoch := args[0]
			cEpoch, err := strconv.ParseInt(epoch, 10, 64)
			if err != nil {
				return err
			}
			res, err := queryClient.OptOutsToFinish(
				cmd.Context(), &types.QueryOptOutsToFinishRequest{Epoch: cEpoch},
			)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func CmdQueryOperatorOptOutFinishEpoch() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "operator-opt-out-finish-epoch [operator]",
		Short: "shows the epoch at which an operator's opt out will be finished",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)
			operator := args[0]
			res, err := queryClient.OperatorOptOutFinishEpoch(
				cmd.Context(), &types.QueryOperatorOptOutFinishEpochRequest{
					OperatorAccAddr: operator,
				},
			)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func CmdQueryUndelegationsToMature() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "undelegations-to-mature [epoch]",
		Short: "shows the undelegations that will mature at the provided epoch",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)
			epoch := args[0]
			cEpoch, err := strconv.ParseInt(epoch, 10, 64)
			if err != nil {
				return err
			}
			res, err := queryClient.UndelegationsToMature(
				cmd.Context(), &types.QueryUndelegationsToMatureRequest{Epoch: cEpoch},
			)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func CmdQueryUndelegationMaturityEpoch() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "undelegation-maturity-epoch [recordKey]",
		Short: "shows the epoch at which an undelegation record will be mature",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)
			recordKey := args[0]
			res, err := queryClient.UndelegationMaturityEpoch(
				cmd.Context(), &types.QueryUndelegationMaturityEpochRequest{RecordKey: recordKey},
			)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func CmdQueryValidator() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validator [consensus-address]",
		Short: "shows the validator information for the provided consensus address",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)
			address := args[0]
			consAddress, err := sdk.ConsAddressFromBech32(address)
			if err != nil {
				return err
			}
			res, err := queryClient.Validator(
				cmd.Context(), &types.QueryValidatorRequest{ConsAddr: consAddress.String()},
			)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func CmdQueryValidators() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validators",
		Short: "shows all validators",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)
			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}
			res, err := queryClient.Validators(
				cmd.Context(), &types.QueryAllValidatorsRequest{Pagination: pageReq},
			)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
