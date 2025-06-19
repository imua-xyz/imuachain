package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	_ sdk.Msg = &RegisterAVSReq{}
	_ sdk.Msg = &DeRegisterAVSReq{}
)

const (
	// TypeSubmitTaskResultReq is the type for the RegisterOperatorReq message.
	TypeSubmitTaskResultReq = "register_operator"
)

// GetSigners returns the expected signers for the message.
func (m *SubmitTaskResultReq) GetSigners() []sdk.AccAddress {
	addr := sdk.MustAccAddressFromBech32(m.FromAddress)
	return []sdk.AccAddress{addr}
}

// ValidateBasic does a sanity check of the provided data
func (m *SubmitTaskResultReq) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.FromAddress); err != nil {
		return errorsmod.Wrap(err, "invalid from address")
	}
	return nil
}

// Route returns the transaction route.
func (m *SubmitTaskResultReq) Route() string {
	return RouterKey
}

// Type returns the transaction type.
func (m *SubmitTaskResultReq) Type() string {
	return TypeSubmitTaskResultReq
}

// GetSignBytes returns the bytes all expected signers must sign over.
func (m *SubmitTaskResultReq) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(m))
}

// GetSigners returns the expected signers for a MsgUpdateParams message.
func (m *RegisterAVSReq) GetSigners() []sdk.AccAddress {
	addr := sdk.MustAccAddressFromBech32(m.FromAddress)
	return []sdk.AccAddress{addr}
}

// ValidateBasic does a sanity check of the provided data
func (m *RegisterAVSReq) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.FromAddress); err != nil {
		return errorsmod.Wrap(err, "invalid from address")
	}
	return nil
}

// GetSignBytes implements the LegacyMsg interface.
func (m *RegisterAVSReq) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(m))
}

// GetSigners returns the expected signers for a MsgUpdateParams message.
func (m *DeRegisterAVSReq) GetSigners() []sdk.AccAddress {
	addr := sdk.MustAccAddressFromBech32(m.FromAddress)
	return []sdk.AccAddress{addr}
}

// ValidateBasic does a sanity check of the provided data
func (m *DeRegisterAVSReq) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.FromAddress); err != nil {
		return errorsmod.Wrap(err, "invalid from address")
	}
	return nil
}

// GetSignBytes implements the LegacyMsg interface.
func (m *DeRegisterAVSReq) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(m))
}

// GetSigners returns the expected signers for a MsgUpdateParams message.
func (m *RegisterAVSTaskReq) GetSigners() []sdk.AccAddress {
	addr := sdk.MustAccAddressFromBech32(m.FromAddress)
	return []sdk.AccAddress{addr}
}

// ValidateBasic does a sanity check of the provided data
func (m *RegisterAVSTaskReq) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.FromAddress); err != nil {
		return errorsmod.Wrap(err, "invalid from address")
	}
	return nil
}

// GetSignBytes implements the LegacyMsg interface.
func (m *RegisterAVSTaskReq) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(m))
}
