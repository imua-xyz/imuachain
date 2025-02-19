package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	// TypeMsgUpdateParams is the type for the MsgUpdateParams tx.
	TypeMsgUpdateParams = "update_params"
)

// interface guards
var (
	_ sdk.Msg = &MsgUpdateParams{}
)

// GetSigners returns the expected signers for the message.
func (m *MsgUpdateParams) GetSigners() []sdk.AccAddress {
	addr := sdk.MustAccAddressFromBech32(m.Authority)
	return []sdk.AccAddress{addr}
}

// ValidateBasic does a sanity check of the provided data
func (m *MsgUpdateParams) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errorsmod.Wrap(err, "invalid from address")
	}
	return m.Params.Validate()
}

// Route returns the transaction route.
func (m *MsgUpdateParams) Route() string {
	return RouterKey
}

// Type returns the transaction type.
func (m *MsgUpdateParams) Type() string {
	return TypeMsgUpdateParams
}

// GetSignBytes returns the bytes all expected signers must sign over.
func (m *MsgUpdateParams) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(m))
}
