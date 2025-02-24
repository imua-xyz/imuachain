package cli

import (
	"context"
	"strconv"
	"strings"

	epochstypes "github.com/imua-xyz/imuachain/x/epochs/types"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/imua-xyz/imuachain/x/assets/types"
	delegationtype "github.com/imua-xyz/imuachain/x/delegation/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
)

// GetQueryCmd returns the parent command for all incentives CLI query commands.
func GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        delegationtype.ModuleName,
		Short:                      "Querying commands for the delegation module",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		QuerySingleDelegationInfo(),
		QueryDelegationInfo(),
		QueryUndelegations(),
		QueryUndelegationsByEpochInfo(),
		QueryUndelegationHoldCount(),
		QueryAssociatedOperatorByStaker(),
		QueryAssociatedStakersByOperator(),
		QueryDelegatedStakersByOperator(),
	)
	return cmd
}

// QuerySingleDelegationInfo queries the single delegation info
func QuerySingleDelegationInfo() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delegation <clientChainID> <stakerAddr> <assetAddr> <operatorAddr>",
		Short: "Get single delegation info",
		Long:  "Get single delegation info",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := delegationtype.NewQueryClient(clientCtx)
			clientChainLzID, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return errorsmod.Wrap(types.ErrInvalidCliCmdArg, err.Error())
			}
			stakerID, assetID := types.GetStakerIDAndAssetIDFromStr(clientChainLzID, args[1], args[2])
			accAddr, err := sdk.AccAddressFromBech32(args[3])
			if err != nil {
				return errorsmod.Wrap(types.ErrInvalidCliCmdArg, err.Error())
			}
			req := &delegationtype.SingleDelegationInfoReq{
				StakerId:     stakerID,         // already lowercase
				AssetId:      assetID,          // already lowercase
				OperatorAddr: accAddr.String(), // already lowercase
			}
			res, err := queryClient.QuerySingleDelegationInfo(context.Background(), req)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// QueryDelegationInfo queries delegation info
func QueryDelegationInfo() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delegations <stakerID> <assetID>",
		Short: "Get delegation info",
		Long:  "Get delegation info",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			stakerID := strings.ToLower(args[0])
			if _, _, err := types.ValidateID(stakerID, false, false); err != nil {
				return errorsmod.Wrap(types.ErrInvalidCliCmdArg, err.Error())
			}
			assetID := strings.ToLower(args[1])
			if _, _, err := types.ValidateID(assetID, false, false); err != nil {
				return errorsmod.Wrap(types.ErrInvalidCliCmdArg, err.Error())
			}
			queryClient := delegationtype.NewQueryClient(clientCtx)
			req := &delegationtype.DelegationInfoReq{
				StakerId: strings.ToLower(stakerID),
				AssetId:  strings.ToLower(assetID),
			}
			res, err := queryClient.QueryDelegationInfo(context.Background(), req)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// QueryUndelegations queries all undelegations for staker and asset
func QueryUndelegations() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "undelegations <stakerID> <assetID>",
		Short: "Get undelegations",
		Long:  "Get undelegations",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := delegationtype.NewQueryClient(clientCtx)
			_, _, err = types.ValidateID(args[0], false, false)
			if err != nil {
				return err
			}
			_, _, err = types.ValidateID(args[1], false, false)
			if err != nil {
				return err
			}
			req := &delegationtype.UndelegationsReq{
				StakerId: strings.ToLower(args[0]),
				AssetId:  strings.ToLower(args[1]),
			}
			res, err := queryClient.QueryUndelegations(context.Background(), req)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// QueryUndelegationsByEpochInfo queries all undelegations waiting to be completed by epoch info.
func QueryUndelegationsByEpochInfo() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "undelegations-by-epoch-info <epoch_identifier> <epoch_number>",
		Short: "Get undelegations waiting to be completed",
		Long:  "Get undelegations waiting to be completed",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			epochNumber, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return err
			}
			err = epochstypes.ValidateEpochIdentifierString(args[0])
			if err != nil {
				return err
			}
			queryClient := delegationtype.NewQueryClient(clientCtx)
			req := &delegationtype.UndelegationsByEpochInfoReq{
				EpochIdentifier: args[0],
				EpochNumber:     epochNumber,
			}
			res, err := queryClient.QueryUndelegationsByEpochInfo(context.Background(), req)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// QueryUndelegationHoldCount queries undelegation hold count for a record key.
func QueryUndelegationHoldCount() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "undelegation-hold-count <stakerID> <assetID> <undelegationID>",
		Short: "Get undelegation hold count",
		Long:  "Get undelegation hold count",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			_, _, err = types.ValidateID(args[0], false, false)
			if err != nil {
				return err
			}
			_, _, err = types.ValidateID(args[1], false, false)
			if err != nil {
				return err
			}
			undelegationID, err := strconv.ParseUint(args[2], 10, 64)
			if err != nil {
				return err
			}
			queryClient := delegationtype.NewQueryClient(clientCtx)
			req := &delegationtype.UndelegationHoldCountReq{
				StakerId:       strings.ToLower(args[0]),
				AssetId:        strings.ToLower(args[1]),
				UndelegationId: undelegationID,
			}
			res, err := queryClient.QueryUndelegationHoldCount(context.Background(), req)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// QueryAssociatedOperatorByStaker queries the operator owner of the specified staker
func QueryAssociatedOperatorByStaker() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "associated-operator-by-staker <stakerID>",
		Short: "Get the associated operator for the specified staker",
		Long:  "Get the associated operator for the specified staker",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			stakerID := args[0]
			if _, _, err := types.ValidateID(stakerID, false, false); err != nil {
				return errorsmod.Wrap(types.ErrInvalidCliCmdArg, err.Error())
			}
			queryClient := delegationtype.NewQueryClient(clientCtx)
			req := &delegationtype.QueryAssociatedOperatorByStakerReq{
				StakerId: strings.ToLower(stakerID),
			}
			res, err := queryClient.QueryAssociatedOperatorByStaker(context.Background(), req)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func QueryAssociatedStakersByOperator() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "associated-stakers-by-operator <operatorAddr>",
		Short: "Get the associated stakers for the specified operator",
		Long:  "Get the associated stakers for the specified operator",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			accAddr, err := sdk.AccAddressFromBech32(args[0])
			if err != nil {
				return errorsmod.Wrap(types.ErrInvalidCliCmdArg, err.Error())
			}
			queryClient := delegationtype.NewQueryClient(clientCtx)
			req := &delegationtype.QueryAssociatedStakersByOperatorReq{
				Operator: accAddr.String(),
			}
			res, err := queryClient.QueryAssociatedStakersByOperator(context.Background(), req)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func QueryDelegatedStakersByOperator() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delegated-stakers-by-operator <operatorAddr> <assetID>",
		Short: "Get the delegated stakers for the specified operator and asset",
		Long:  "Get the delegated stakers for the specified operator and asset",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			accAddr, err := sdk.AccAddressFromBech32(args[0])
			if err != nil {
				return errorsmod.Wrap(types.ErrInvalidCliCmdArg, err.Error())
			}
			assetID := strings.ToLower(args[1])
			if _, _, err := types.ValidateID(assetID, false, false); err != nil {
				return errorsmod.Wrap(types.ErrInvalidCliCmdArg, err.Error())
			}
			queryClient := delegationtype.NewQueryClient(clientCtx)
			req := &delegationtype.QueryDelegatedStakersByOperatorReq{
				Operator: accAddr.String(),
				AssetId:  assetID,
			}
			res, err := queryClient.QueryDelegatedStakersByOperator(context.Background(), req)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
