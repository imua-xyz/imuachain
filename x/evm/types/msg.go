package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	evmostypes "github.com/evmos/evmos/v16/x/evm/types"
)

const (
	TypeMsgCallContract = "call_contract"
)

var _ sdk.Msg = &MsgCallContract{}

// GetSigners returns the expected signers for a MsgCallContract message.
func (m *MsgCallContract) GetSigners() []sdk.AccAddress {
	addr := sdk.MustAccAddressFromBech32(m.Authority)
	return []sdk.AccAddress{addr}
}

// ValidateBasic does a sanity check of the provided data
func (m *MsgCallContract) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errorsmod.Wrap(err, "invalid authority address")
	}
	// data can be empty for calling fallback payable functions
	// to can be empty for contract creation
	// gas limit can be 0 for caller to use the block max gas limit
	// thus, no more validation can be applied.
	return nil
}

// Route returns the transaction route.
func (m *MsgCallContract) Route() string {
	return evmostypes.RouterKey
}

// Type returns the transaction type.
func (m *MsgCallContract) Type() string {
	return TypeMsgCallContract
}

// GetSignBytes returns the bytes all expected signers must sign over.
func (m *MsgCallContract) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(m))
}
