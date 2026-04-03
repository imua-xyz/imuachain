package cli

import (
	"encoding/json"
	"fmt"
	"strconv"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/imua-xyz/imuachain/x/oracle/types"
	"github.com/spf13/cobra"
)

// CmdQueryOutboundMessages queries the outbound message queue for a destination chain.
// This uses direct KV store queries via ABCI since the outbound queue
// is not part of the proto-generated query service.
func CmdQueryOutboundMessages() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "outbound-messages [dst-chain-id]",
		Short: "Query pending outbound messages for a destination chain",
		Long:  "Query the outbound message queue that relayers poll to deliver cross-chain responses.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			dstChainID, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid dst-chain-id: %w", err)
			}

			afterSeq, _ := cmd.Flags().GetUint64("after-seq")
			limit, _ := cmd.Flags().GetUint64("limit")
			if limit == 0 {
				limit = 100
			}

			// Query head and tail to determine queue bounds
			head, err := queryStoreUint64(clientCtx, types.OutboundHeadKey(dstChainID))
			if err != nil {
				return err
			}
			tail, err := queryStoreUint64(clientCtx, types.OutboundTailKey(dstChainID))
			if err != nil {
				return err
			}

			if tail <= head {
				fmt.Println("[]")
				return nil
			}

			// Iterate queue items
			type outboundMsg struct {
				DstChainID uint64 `json:"dst_chain_id"`
				SeqNum     uint64 `json:"seq_num"`
				Nonce      uint64 `json:"nonce"`
				PayloadHex string `json:"payload_hex"`
				Height     int64  `json:"height"`
			}

			var msgs []outboundMsg
			for idx := head; idx < tail && uint64(len(msgs)) < limit; idx++ {
				itemKey := types.OutboundItemKey(dstChainID, idx)
				resp, err := clientCtx.QueryABCI(abci.RequestQuery{
					Path: "store/" + types.StoreKey + "/key",
					Data: itemKey,
				})
				if err != nil || len(resp.Value) == 0 {
					continue
				}
				var msg outboundMsg
				if err := json.Unmarshal(resp.Value, &msg); err != nil {
					continue
				}
				if msg.SeqNum <= afterSeq {
					continue
				}
				msgs = append(msgs, msg)
			}

			out, err := json.MarshalIndent(msgs, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(out))
			return nil
		},
	}

	cmd.Flags().Uint64("after-seq", 0, "Only return messages with seq > after-seq")
	cmd.Flags().Uint64("limit", 100, "Maximum number of messages to return")
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func queryStoreUint64(clientCtx client.Context, key []byte) (uint64, error) {
	resp, err := clientCtx.QueryABCI(abci.RequestQuery{
		Path: "store/" + types.StoreKey + "/key",
		Data: key,
	})
	if err != nil {
		return 0, err
	}
	if len(resp.Value) == 0 {
		return 0, nil
	}
	v, err := types.BytesToUint64(resp.Value)
	if err != nil {
		return 0, nil
	}
	return v, nil
}
