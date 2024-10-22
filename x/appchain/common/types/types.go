package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
)

const maxLogSize = 1024

// NewResultAcknowledgementWithLog creates a result acknowledgement with a log message.
func NewResultAcknowledgementWithLog(ctx sdk.Context, res []byte) channeltypes.Acknowledgement {
	if len(res) > maxLogSize {
		ctx.Logger().Info(
			"IBC ResultAcknowledgement constructed",
			"res_size", len(res),
			"res_preview", string(res[:maxLogSize]),
		)
	} else {
		ctx.Logger().Info(
			"IBC ResultAcknowledgement constructed",
			"res_size", len(res),
			"res", string(res),
		)
	}
	return channeltypes.NewResultAcknowledgement(res)
}

// NewErrorAcknowledgementWithLog creates an error acknowledgement with a log message.
func NewErrorAcknowledgementWithLog(ctx sdk.Context, err error) channeltypes.Acknowledgement {
	ctx.Logger().Error("IBC ErrorAcknowledgement constructed", "error", err)
	return channeltypes.NewErrorAcknowledgement(err)
}
