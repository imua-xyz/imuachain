package types

import (
	"testing"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/imua-xyz/imuachain/testutil/sample"

	// sdkerrors "cosmossdk.io/errors""
	"github.com/stretchr/testify/require"
)

func TestMsgPriceFeed_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgPriceFeed
		err  error
	}{
		{
			name: "invalid address",
			msg: MsgPriceFeed{
				Creator: "invalid_address",
			},
			err: sdkerrors.ErrInvalidAddress,
		}, {
			name: "valid address",
			msg: MsgPriceFeed{
				Creator: sample.AccAddress(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
		})
	}
}
