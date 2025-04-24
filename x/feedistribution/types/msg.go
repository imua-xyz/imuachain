package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	// TypeMsgUpdateParams is the type for the MsgUpdateParams tx.
	TypeMsgUpdateParams              = "update_params"
	TypeMsgWithdrawDogfoodCommission = "withdraw_dogfood_commission"
)

var (
	_ sdk.Msg = &MsgUpdateParams{}
	_ sdk.Msg = &MsgWithdrawDogfoodCommission{}
)

// ValidateBasic does a sanity check on the provided data.
func (m *MsgUpdateParams) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errorsmod.Wrap(err, "invalid authority address")
	}

	if err := m.Params.Validate(); err != nil {
		return err
	}

	return nil
}

// Route returns the transaction route.
func (m *MsgUpdateParams) Route() string {
	return RouterKey
}

// Type returns the transaction type.
func (m *MsgUpdateParams) Type() string {
	return TypeMsgUpdateParams
}

// GetSigners returns the expected signers for a MsgUpdateParams message.
func (m *MsgUpdateParams) GetSigners() []sdk.AccAddress {
	addr := sdk.MustAccAddressFromBech32(m.Authority)
	return []sdk.AccAddress{addr}
}

// GetSignBytes implements the LegacyMsg interface.
func (m *MsgUpdateParams) GetSignBytes() []byte {
	return nil
}

// ValidateBasic does a sanity check on the provided data.
func (m *MsgWithdrawDogfoodCommission) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.FromAddress); err != nil {
		return errorsmod.Wrap(err, "invalid operator address")
	}
	return nil
}

// Route returns the transaction route.
func (m *MsgWithdrawDogfoodCommission) Route() string {
	return RouterKey
}

// Type returns the transaction type.
func (m *MsgWithdrawDogfoodCommission) Type() string {
	return TypeMsgWithdrawDogfoodCommission
}

// GetSigners returns the expected signers for a MsgUpdateParams message.
func (m *MsgWithdrawDogfoodCommission) GetSigners() []sdk.AccAddress {
	addr := sdk.MustAccAddressFromBech32(m.FromAddress)
	return []sdk.AccAddress{addr}
}

// GetSignBytes implements the LegacyMsg interface.
func (m *MsgWithdrawDogfoodCommission) GetSignBytes() []byte {
	return nil
}
