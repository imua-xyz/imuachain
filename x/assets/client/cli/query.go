package cli

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	errorsmod "cosmossdk.io/errors"
	"github.com/imua-xyz/imuachain/x/assets/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"
)

// GetQueryCmd returns the parent command for all incentives CLI query commands.
func GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Querying commands for the assets module",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		GetParamsCmd(),
		QueClientChainInfoByIndex(),
		QueAllClientChainInfo(),
		QueStakingAssetInfo(),
		QueAllStakingAssetsInfo(),
		QueStakerAssetInfos(),
		QueStakerSpecifiedAssetAmount(),
		QueOperatorAssetInfos(),
		QueOperatorSpecifiedAssetAmount(),
	)
	return cmd
}

// GetParamsCmd queries the module parameters
func GetParamsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Gets assets module params",
		Long:  "Gets assets module params",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			req := &types.QueryParamsRequest{}

			res, err := queryClient.Params(context.Background(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// QueClientChainInfoByIndex queries the client chain info by index
func QueClientChainInfoByIndex() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "client-chain <clientChainID>",
		Short: "Get client chain info by client chain Id",
		Long:  "Get client chain info by client chain Id",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			clientChainID, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return errorsmod.Wrap(types.ErrInvalidCliCmdArg, err.Error())
			}
			queryClient := types.NewQueryClient(clientCtx)
			req := &types.QueryClientChainInfo{
				ChainIndex: clientChainID,
			}
			res, err := queryClient.QueClientChainInfoByIndex(context.Background(), req)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// QueAllClientChainInfo queries all client chain info
func QueAllClientChainInfo() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "all-client-chains",
		Short: "Get all client chain info",
		Long:  "Get all client chain info",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			req := &types.QueryAllClientChainInfo{}
			res, err := queryClient.QueAllClientChainInfo(context.Background(), req)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// QueStakingAssetInfo queries staking asset info
func QueStakingAssetInfo() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "staking-asset <assetAddr> <clientChainID>",
		Short: "Get staking asset info",
		Long:  "Get staking asset info",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			clientChainID, err := strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				return errorsmod.Wrap(types.ErrInvalidCliCmdArg, fmt.Sprintf("error arg is:%v", args[1]))
			}

			_, assetID := types.GetStakerIDAndAssetIDFromStr(clientChainID, "", args[0])
			queryClient := types.NewQueryClient(clientCtx)
			req := &types.QueryStakingAssetInfo{
				AssetId: assetID, // already lowercase
			}
			res, err := queryClient.QueStakingAssetInfo(context.Background(), req)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// QueAllStakingAssetsInfo queries all staking asset info
func QueAllStakingAssetsInfo() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "all-staking-assets",
		Short: "Get all staking asset info",
		Long:  "Get all staking asset info",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			req := &types.QueryAllStakingAssetsInfo{}
			res, err := queryClient.QueAllStakingAssetsInfo(context.Background(), req)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// QueStakerAssetInfos queries staker asset info
func QueStakerAssetInfos() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "staker-assets <stakerID>",
		Short: "Get staker asset state",
		Long:  "Get staker asset state",
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

			queryClient := types.NewQueryClient(clientCtx)
			req := &types.QueryStakerAssetInfo{
				StakerId: strings.ToLower(stakerID),
			}
			res, err := queryClient.QueStakerAssetInfos(context.Background(), req)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// QueStakerSpecifiedAssetAmount queries staker specified asset info
func QueStakerSpecifiedAssetAmount() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "staker-asset-amount <clientChainID> <stakerAddr> <assetAddr>",
		Short: "Get staker specified asset state",
		Long:  "Get staker specified asset state",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			clientChainLzID, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return errorsmod.Wrap(types.ErrInvalidCliCmdArg, err.Error())
			}
			stakerID, assetID := types.GetStakerIDAndAssetIDFromStr(clientChainLzID, args[1], args[2])
			req := &types.QuerySpecifiedAssetAmountReq{
				StakerId: stakerID, // already lowercase
				AssetId:  assetID,  // already lowercase
			}
			res, err := queryClient.QueStakerSpecifiedAssetAmount(context.Background(), req)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// QueOperatorAssetInfos queries operator asset info
func QueOperatorAssetInfos() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "operator-assets <operatorAddr>",
		Short: "Get operator asset state",
		Long:  "Get operator asset state",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			accAddr, err := sdk.AccAddressFromBech32(args[0])
			if err != nil {
				return errorsmod.Wrap(types.ErrInvalidCliCmdArg, err.Error())
			}
			req := &types.QueryOperatorAssetInfos{
				OperatorAddr: accAddr.String(), // already lowercase
			}
			res, err := queryClient.QueOperatorAssetInfos(context.Background(), req)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// QueOperatorSpecifiedAssetAmount queries specified operator asset info
func QueOperatorSpecifiedAssetAmount() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "operator-asset-amount <operatorAddr> <clientChainID> <assetAddr>",
		Short: "Get operator specified asset state",
		Long:  "Get operator specified asset state",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			clientChainLzID, err := strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				return errorsmod.Wrap(types.ErrInvalidCliCmdArg, err.Error())
			}
			_, assetID := types.GetStakerIDAndAssetIDFromStr(clientChainLzID, "", args[2])
			accAddr, err := sdk.AccAddressFromBech32(args[0])
			if err != nil {
				return errorsmod.Wrap(types.ErrInvalidCliCmdArg, err.Error())
			}
			queryClient := types.NewQueryClient(clientCtx)
			req := &types.QueryOperatorSpecifiedAssetAmountReq{
				OperatorAddr: accAddr.String(), // already lowercase
				AssetId:      assetID,          // already lowercase
			}
			res, err := queryClient.QueOperatorSpecifiedAssetAmount(context.Background(), req)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
