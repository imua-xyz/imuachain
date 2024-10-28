package cli

import (
	"context"
	"strings"

	"github.com/ExocoreNetwork/exocore/x/avs/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"golang.org/x/xerrors"

	operatortypes "github.com/ExocoreNetwork/exocore/x/operator/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
)

type GenericQueryParams struct {
	queryClient operatortypes.QueryClient
	clientCtx   client.Context
}

func ValidOperatorAVSAddr(cmd *cobra.Command, originalOperatorAddr, originalAVSAddr string) (*operatortypes.OperatorAVSAddress, *GenericQueryParams, error) {
	_, err := sdk.AccAddressFromBech32(originalOperatorAddr)
	if err != nil {
		return nil, nil, xerrors.Errorf("invalid operator address,err:%s", err.Error())
	}
	if !common.IsHexAddress(originalAVSAddr) {
		return nil, nil, xerrors.Errorf("invalid avs address,err:%s", types.ErrInvalidAddr)
	}
	clientCtx, err := client.GetClientQueryContext(cmd)
	if err != nil {
		return nil, nil, err
	}
	queryClient := operatortypes.NewQueryClient(clientCtx)
	return &operatortypes.OperatorAVSAddress{
		OperatorAddr: originalOperatorAddr,
		AvsAddress:   strings.ToLower(originalAVSAddr),
	}, &GenericQueryParams{queryClient: queryClient, clientCtx: clientCtx}, nil
}

// GetQueryCmd returns the parent command for all incentives CLI query commands.
func GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        operatortypes.ModuleName,
		Short:                      "Querying commands for the operator module",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		GetOperatorInfo(),
		GetAllOperators(),
		GetOperatorConsKey(),
		GetOperatorConsAddress(),
		GetAllOperatorKeys(),
		GetAllOperatorConsAddrs(),
		QueryOperatorUSDValue(),
		QueryAVSUSDValue(),
		QueryOperatorSlashInfo(),
		QueryAllOperatorsWithOptInAVS(),
		QueryAllAVSsByOperator(),
		GetOptInfo(),
	)
	return cmd
}

// GetOperatorInfo queries operator info
func GetOperatorInfo() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-operator-info <operatorAddr>",
		Short: "Get operator info",
		Long:  "Get operator info",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := sdk.AccAddressFromBech32(args[0])
			if err != nil {
				return xerrors.Errorf("invalid operator address,err:%s", err.Error())
			}
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := operatortypes.NewQueryClient(clientCtx)
			req := &operatortypes.GetOperatorInfoReq{
				OperatorAddr: args[0],
			}
			res, err := queryClient.QueryOperatorInfo(context.Background(), req)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// GetAllOperators queries all operators
func GetAllOperators() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-all-operators",
		Short: "Get all operators",
		Long:  "Get all operator account addresses",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			queryClient := operatortypes.NewQueryClient(clientCtx)
			req := &operatortypes.QueryAllOperatorsRequest{
				Pagination: pageReq,
			}
			res, err := queryClient.QueryAllOperators(context.Background(), req)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// GetOperatorConsKey queries operator consensus key for the provided chain ID
func GetOperatorConsKey() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-operator-cons-key <operator_address> <chain_id>",
		Short: "Get operator consensus key",
		Long:  "Get operator consensus key for the provided chain ID",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := sdk.AccAddressFromBech32(args[0])
			if err != nil {
				return xerrors.Errorf("invalid operator address,err:%s", err.Error())
			}
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := operatortypes.NewQueryClient(clientCtx)
			req := &operatortypes.QueryOperatorConsKeyRequest{
				OperatorAccAddr: args[0],
				Chain:           args[1],
			}
			res, err := queryClient.QueryOperatorConsKeyForChainID(
				context.Background(), req,
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

// GetAllOperatorKeys queries all operators for the provided chain ID and their
// consensus keys
func GetAllOperatorKeys() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-all-operators-by-chain-id <chain_id>",
		Short: "Get all operators for the provided chain ID",
		Long:  "Get all operators for the provided chain ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			queryClient := operatortypes.NewQueryClient(clientCtx)
			req := &operatortypes.QueryAllOperatorConsKeysByChainIDRequest{
				Chain:      args[0],
				Pagination: pageReq,
			}
			res, err := queryClient.QueryAllOperatorConsKeysByChainID(
				context.Background(), req,
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

// GetOperatorConsAddress queries operator consensus address for the provided chain ID
func GetOperatorConsAddress() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-operator-cons-address <operator_address> <chain_id>",
		Short: "Get operator consensus address",
		Long:  "Get operator consensus address for the provided chain ID",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			_, err = sdk.AccAddressFromBech32(args[0])
			if err != nil {
				return xerrors.Errorf("invalid operator address,err:%s", err.Error())
			}

			queryClient := operatortypes.NewQueryClient(clientCtx)
			req := &operatortypes.QueryOperatorConsAddressRequest{
				OperatorAccAddr: args[0],
				Chain:           args[1],
			}
			res, err := queryClient.QueryOperatorConsAddressForChainID(
				context.Background(), req,
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

// GetAllOperatorConsAddrs queries all operators for the provided chain ID and their
// consensus addresses
func GetAllOperatorConsAddrs() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-all-operator-cons-addrs <chain_id>",
		Short: "Get all operators for the provided chain ID",
		Long:  "Get all operators for the provided chain ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			queryClient := operatortypes.NewQueryClient(clientCtx)
			req := &operatortypes.QueryAllOperatorConsAddrsByChainIDRequest{
				Chain:      args[0],
				Pagination: pageReq,
			}
			res, err := queryClient.QueryAllOperatorConsAddrsByChainID(
				context.Background(), req,
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

// QueryOperatorUSDValue queries the opted-in USD value for the operator
func QueryOperatorUSDValue() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "QueryOperatorUSDValue <operatorAddr> <avsAddr>",
		Short:   "Get the opted-in USD value",
		Long:    "Get the opted-in USD value for the operator",
		Example: "exocored query operator QueryOperatorUSDValue exo1c5x7mxphvgavjhu0au9jjqnfqcyspevtyy27mz 0xaa089ba103f765fcea44808bd3d4073523254c57",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			validOperatorAVSAddr, genericQueryParams, err := ValidOperatorAVSAddr(cmd, args[0], args[1])
			if err != nil {
				return err
			}
			res, err := genericQueryParams.queryClient.QueryOperatorUSDValue(context.Background(), &operatortypes.QueryOperatorUSDValueRequest{
				OperatorAVSAddress: validOperatorAVSAddr,
			})
			if err != nil {
				return err
			}
			return genericQueryParams.clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// QueryAVSUSDValue queries the USD value for the avs
func QueryAVSUSDValue() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "QueryAVSUSDValue <avsAddr>",
		Short: "Get the USD value for the avs",
		Long:  "Get the USD value for the avs",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !common.IsHexAddress(args[0]) {
				return xerrors.Errorf("invalid avs address,err:%s", types.ErrInvalidAddr)
			}
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := operatortypes.NewQueryClient(clientCtx)
			req := &operatortypes.QueryAVSUSDValueRequest{
				AVSAddress: strings.ToLower(args[0]),
			}
			res, err := queryClient.QueryAVSUSDValue(context.Background(), req)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// QueryOperatorSlashInfo queries the slash information for the specified operator and AVS
func QueryOperatorSlashInfo() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "QueryOperatorSlashInfo <operatorAddr> <avsAddr>",
		Short:   "Get the the slash information for the operator",
		Long:    "Get the the slash information for the operator",
		Example: "exocored query operator QueryOperatorSlashInfo exo1c5x7mxphvgavjhu0au9jjqnfqcyspevtyy27mz 0xaa089ba103f765fcea44808bd3d4073523254c57",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			validOperatorAVSAddr, genericQueryParams, err := ValidOperatorAVSAddr(cmd, args[0], args[1])
			if err != nil {
				return err
			}
			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}
			req := &operatortypes.QueryOperatorSlashInfoRequest{
				OperatorAVSAddress: validOperatorAVSAddr,
				Pagination:         pageReq,
			}
			res, err := genericQueryParams.queryClient.QueryOperatorSlashInfo(context.Background(), req)
			if err != nil {
				return err
			}
			return genericQueryParams.clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// QueryAllOperatorsWithOptInAVS queries all operators
func QueryAllOperatorsWithOptInAVS() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "get-operator-list <avsAddr>",
		Short:   "Get list of operators by AVS",
		Long:    "Get the list of operators who have opted in to the specified AVS",
		Example: "exocored query operator get-operator-list 0x598ACcB5e7F83cA6B19D70592Def9E5b25B978CA",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !common.IsHexAddress(args[0]) {
				return xerrors.Errorf("invalid  address,err:%s", types.ErrInvalidAddr)
			}
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := operatortypes.NewQueryClient(clientCtx)
			req := operatortypes.QueryAllOperatorsByOptInAVSRequest{
				Avs: strings.ToLower(args[0]),
			}
			res, err := queryClient.QueryAllOperatorsWithOptInAVS(context.Background(), &req)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// QueryAllAVSsByOperator queries all avs
func QueryAllAVSsByOperator() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "get-avs-list <operatorAddr>",
		Short:   "Get list of AVSs by operator",
		Long:    "Get a list of AVSs to which an operator has opted in",
		Example: "exocored query operator get-avs-list exo1mq6pj6f5tafmgkk6lehew5radfq3w20gpegzs5",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			addr, err := sdk.AccAddressFromBech32(args[0])
			if err != nil {
				return xerrors.Errorf("invalid  address,err:%s", types.ErrInvalidAddr)
			}
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := operatortypes.NewQueryClient(clientCtx)
			req := operatortypes.QueryAllAVSsByOperatorRequest{
				Operator: addr.String(),
			}
			res, err := queryClient.QueryAllAVSsByOperator(context.Background(), &req)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// GetOptInfo queries opt info
func GetOptInfo() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "GetOptInfo <operatorAddr> <avsAddr>",
		Short:   "Get opt info",
		Long:    "Get opt info of specified operator and AVS",
		Example: "exocored query operator GetOptInfo exo1c5x7mxphvgavjhu0au9jjqnfqcyspevtyy27mz 0xaa089ba103f765fcea44808bd3d4073523254c57",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			validOperatorAVSAddr, genericQueryParams, err := ValidOperatorAVSAddr(cmd, args[0], args[1])
			if err != nil {
				return err
			}
			res, err := genericQueryParams.queryClient.QueryOptInfo(context.Background(), &operatortypes.QueryOptInfoRequest{
				OperatorAVSAddress: validOperatorAVSAddr,
			})
			if err != nil {
				return err
			}
			return genericQueryParams.clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
