package cli

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	"golang.org/x/xerrors"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/imua-xyz/imuachain/x/avs/types"
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
		QueryAVSAddressByChainID(),
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
		Example: fmt.Sprintf("%s query avs AVSInfo 0x598ACcB5e7F83cA6B19D70592Def9E5b25B978CA", version.AppName),
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			queryClient, clientCtx, err := commonQuerySetup(cmd, args[0])
			if err != nil {
				return err
			}
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

// QueryAVSAddressByChainID returns a command to query AVS address by chainID
func QueryAVSAddressByChainID() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "AVSAddrByChainID <chainID>",
		Short:   "AVSAddrByChainID <chainID>",
		Long:    "AVSAddrByChainID query for AVS address by chainID",
		Example: fmt.Sprintf("%s query avs AVSAddrByChainID imuachaintestnet_233-1", version.AppName),
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// pass the default address so it doesn't matter
			queryClient, clientCtx, err := commonQuerySetup(cmd, common.Address{}.Hex())
			if err != nil {
				return err
			}

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

// commonQuerySetup handles the common setup logic for query commands
func commonQuerySetup(cmd *cobra.Command, taskAddress string) (types.QueryClient, client.Context, error) {
	if !common.IsHexAddress(taskAddress) {
		return nil, client.Context{}, xerrors.Errorf("invalid address,err:%s", types.ErrInvalidAddr)
	}

	clientCtx, err := client.GetClientQueryContext(cmd)
	if err != nil {
		return nil, client.Context{}, err
	}

	queryClient := types.NewQueryClient(clientCtx)
	return queryClient, clientCtx, nil
}

func QueryTaskInfo() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "TaskInfo <task-address-in-hex> <task-id>",
		Short:   "Query the TaskInfo by its address and ID",
		Long:    "Query the currently registered tasks for an AVS by the task's address and ID",
		Example: fmt.Sprintf("%s query avs TaskInfo 0x96949787E6a209AFb4dE035754F79DC9982D3F2a 2", version.AppName),
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			queryClient, clientCtx, err := commonQuerySetup(cmd, args[0])
			if err != nil {
				return err
			}

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
		Example: fmt.Sprintf("%s query avs SubmitTaskResult 0x96949787E6a209AFb4dE035754F79DC9982D3F2a 2 im18cggcpvwspnd5c6ny8wrqxpffj5zmhkl3agtrj", version.AppName),
		Args:    cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			queryClient, clientCtx, err := commonQuerySetup(cmd, args[0])
			if err != nil {
				return err
			}
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
		Long:    "Query the currently Challenge Info",
		Example: fmt.Sprintf("%s query avs ChallengeInfo 0x96949787E6a209AFb4dE035754F79DC9982D3F2a 2", version.AppName),
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			queryClient, clientCtx, err := commonQuerySetup(cmd, args[0])
			if err != nil {
				return err
			}

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
