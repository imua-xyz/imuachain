package cli

import (
	"errors"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/imua-xyz/imuachain/x/oracle/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdCreatePrice() *cobra.Command {
	cmd := &cobra.Command{
		// TODO: support v1 single sourceID for temporary
		Use:   "create-price feederid basedblock nonce sourceid decimal price timestamp detid optinoal(price timestamp detid) optional(desc)",
		Short: "Broadcast message create-price",
		Args:  cobra.MinimumNArgs(8),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			feederID, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil || feederID < 1 {
				return errors.New("feederID invalid")
			}
			basedBlock, err := strconv.ParseUint(args[1], 10, 64)
			if err != nil || basedBlock < 1 {
				return errors.New("basedBlock invalid")
			}
			nonce, err := strconv.ParseInt(args[2], 10, 32)
			if err != nil || nonce < 1 || nonce > 3 {
				return errors.New("nonce invalid")
			}
			sourceID, err := strconv.ParseUint(args[3], 10, 64)
			if err != nil || sourceID < 1 {
				return errors.New("sourceID invalid")
			}
			decimal, err := strconv.ParseInt(args[4], 10, 32)
			if err != nil || decimal < 0 {
				return errors.New("decimal invalid")
			}
			prices := []*types.PriceSource{
				{
					SourceID: sourceID,
					Prices:   make([]*types.PriceTimeDetID, 0, 1),
					Desc:     "",
				},
			}
			argLength := len(args) - 5
			i := 5
			for argLength > 2 {
				price := args[i]
				timestamp := args[i+1]
				detID := args[i+2]
				argLength -= 3
				i += 3
				prices[0].Prices = append(prices[0].Prices, &types.PriceTimeDetID{
					Price: price,
					// #nosec G115
					Decimal:   int32(decimal),
					Timestamp: timestamp,
					DetID:     detID,
				})
			}
			if argLength == 1 {
				prices[0].Desc = args[i+1]
			}

			msg := types.NewMsgCreatePrice(
				clientCtx.GetFromAddress().String(),
				feederID,
				prices,
				basedBlock,
				// #nosec G115
				int32(nonce),
			)
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
