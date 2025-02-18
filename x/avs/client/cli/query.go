package cli

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	"golang.org/x/xerrors"

	"github.com/ExocoreNetwork/exocore/x/avs/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
)

// GetQueryCmd returns the cli query commands for this module
func GetQueryCmd(_ string) *cobra.Command {
	// Group avs queries under a subcommand
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      fmt.Sprintf("Querying commands for the %s module", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		QueryAVSInfo(),
		QueryAVSAddrByChainID(),
		QueryTaskInfo(),
		QueryChallengeInfo(),
		QuerySubmitTaskResult(),
	)
	return cmd
}

func QueryAVSInfo() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "AVSInfo query <avsAddr>",
		Short:   "AVSInfo query",
		Long:    "AVSInfo query for current registered AVS",
		Example: "exocored query avs AVSInfo  0x598ACcB5e7F83cA6B19D70592Def9E5b25B978CA",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !common.IsHexAddress(args[0]) {
				return xerrors.Errorf("invalid avs  address,err:%s", types.ErrInvalidAddr)
			}
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			req := &types.QueryAVSInfoReq{
				AVSAddress: args[0],
			}
			res, err := queryClient.QueryAVSInfo(context.Background(), req)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// QueryAVSAddrByChainID returns a command to query AVS address by chainID
func QueryAVSAddrByChainID() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "AVSAddrByChainID <chainID>",
		Short:   "AVSAddrByChainID <chainID>",
		Long:    "AVSAddrByChainID query for AVS address by chainID",
		Example: "exocored query avs AVSAddrByChainID exocoretestnet_233-1",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			req := &types.QueryAVSAddressByChainIDReq{
				Chain: args[0],
			}
			res, err := queryClient.QueryAVSAddressByChainID(context.Background(), req)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func QueryTaskInfo() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "TaskInfo <task-address-in-hex> <task-id>",
		Short:   "Query the TaskInfo by its address and ID",
		Long:    "Query the currently registered tasks for an AVS by the task's address and ID",
		Example: "exocored query avs TaskInfo 0x96949787E6a209AFb4dE035754F79DC9982D3F2a 2",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !common.IsHexAddress(args[0]) {
				return xerrors.Errorf("invalid task  address,err:%s", types.ErrInvalidAddr)
			}
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			req := types.QueryAVSTaskInfoReq{
				TaskAddress: args[0],
				TaskId:      args[1],
			}
			res, err := queryClient.QueryAVSTaskInfo(context.Background(), &req)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func QuerySubmitTaskResult() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "SubmitTaskResult <task-address-in-hex> <task-id> <operator-addreess>",
		Short:   "Query the SubmitTaskResult by taskAddr  taskID operatorAddr",
		Long:    "Query the currently submitted Task Result",
		Example: "exocored query avs SubmitTaskResult 0x96949787E6a209AFb4dE035754F79DC9982D3F2a 2 exo1mq6pj6f5tafmgkk6lehew5radfq3w20gpegzs5",
		Args:    cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !common.IsHexAddress(args[0]) {
				return xerrors.Errorf("invalid   address,err:%s", types.ErrInvalidAddr)
			}
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			req := types.QuerySubmitTaskResultReq{
				TaskAddress:     args[0],
				TaskId:          args[1],
				OperatorAddress: args[2],
			}
			res, err := queryClient.QuerySubmitTaskResult(context.Background(), &req)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func QueryChallengeInfo() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "ChallengeInfo <task-address-in-hex> <task-id>",
		Short:   "Query the ChallengeInfo by taskAddr and taskID",
		Long:    "Query the currently Challenge Info  ",
		Example: "exocored query avs ChallengeInfo 0x96949787E6a209AFb4dE035754F79DC9982D3F2a 2",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !common.IsHexAddress(args[0]) {
				return xerrors.Errorf("invalid address,err:%s", types.ErrInvalidAddr)
			}
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			req := types.QueryChallengeInfoReq{
				TaskAddress: args[0],
				TaskId:      args[1],
			}
			res, err := queryClient.QueryChallengeInfo(context.Background(), &req)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
