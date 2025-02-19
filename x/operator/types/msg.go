package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	keytypes "github.com/imua-xyz/imuachain/types/keys"
)

const (
	// TypeRegisterOperatorReq is the type for the RegisterOperatorReq message.
	TypeRegisterOperatorReq = "register_operator"
	// TypeSetConsKeyReq is the type for the SetConsKeyReq message.
	TypeSetConsKeyReq = "set_cons_key"
	// TypeOptIntoAVSReq is the type for the OptIntoAVSReq message.
	TypeOptIntoAVSReq = "opt_into_avs"
	// TypeOptOutOfAVSReq is the type for the OptOutOfAVSReq message.
	TypeOptOutOfAVSReq = "opt_out_of_avs"
)

// interface guards
var (
	_ sdk.Msg = &RegisterOperatorReq{}
	_ sdk.Msg = &OptIntoAVSReq{}
	_ sdk.Msg = &OptOutOfAVSReq{}
	_ sdk.Msg = &SetConsKeyReq{}
)

// GetSigners returns the expected signers for the message.
func (m *RegisterOperatorReq) GetSigners() []sdk.AccAddress {
	addr := sdk.MustAccAddressFromBech32(m.FromAddress)
	return []sdk.AccAddress{addr}
}

// ValidateBasic does a sanity check of the provided data
func (m *RegisterOperatorReq) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.FromAddress); err != nil {
		return errorsmod.Wrap(err, "invalid from address")
	}
	return m.Info.ValidateBasic()
}

// Route returns the transaction route.
func (m *RegisterOperatorReq) Route() string {
	return RouterKey
}

// Type returns the transaction type.
func (m *RegisterOperatorReq) Type() string {
	return TypeRegisterOperatorReq
}

// GetSignBytes returns the bytes all expected signers must sign over.
func (m *RegisterOperatorReq) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(m))
}

// GetSigners returns the expected signers for the message.
func (m *SetConsKeyReq) GetSigners() []sdk.AccAddress {
	addr := sdk.MustAccAddressFromBech32(m.Address)
	return []sdk.AccAddress{addr}
}

// ValidateBasic does a sanity check of the provided data
func (m *SetConsKeyReq) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Address); err != nil {
		return errorsmod.Wrap(err, "invalid from address")
	}
	if !common.IsHexAddress(m.AvsAddress) {
		return errorsmod.Wrap(ErrParameterInvalid, "invalid AVS address")
	}
	if wrappedKey := keytypes.NewWrappedConsKeyFromJSON(m.PublicKeyJSON); wrappedKey == nil {
		return errorsmod.Wrapf(ErrParameterInvalid, "invalid public key")
	}
	return nil
}

// Route returns the transaction route.
func (m *SetConsKeyReq) Route() string {
	return RouterKey
}

// Type returns the transaction type.
func (m *SetConsKeyReq) Type() string {
	return TypeSetConsKeyReq
}

// GetSignBytes returns the bytes all expected signers must sign over.
func (m *SetConsKeyReq) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(m))
}

// GetSigners returns the expected signers for the message.
func (m *OptIntoAVSReq) GetSigners() []sdk.AccAddress {
	addr := sdk.MustAccAddressFromBech32(m.FromAddress)
	return []sdk.AccAddress{addr}
}

// ValidateBasic does a sanity check of the provided data
func (m *OptIntoAVSReq) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.FromAddress); err != nil {
		return errorsmod.Wrap(err, "invalid from address")
	}
	if !common.IsHexAddress(m.AvsAddress) {
		return errorsmod.Wrap(ErrParameterInvalid, "invalid AVS address")
	}
	// cannot check whether a public key is required or not,
	// since that is a stateful check
	if key := m.PublicKeyJSON; len(key) > 0 {
		if wrappedKey := keytypes.NewWrappedConsKeyFromJSON(key); wrappedKey == nil {
			return errorsmod.Wrapf(ErrParameterInvalid, "invalid public key")
		}
	}
	return nil
}

// Route returns the transaction route.
func (m *OptIntoAVSReq) Route() string {
	return RouterKey
}

// Type returns the transaction type.
func (m *OptIntoAVSReq) Type() string {
	return TypeOptIntoAVSReq
}

// GetSignBytes returns the bytes all expected signers must sign over.
func (m *OptIntoAVSReq) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(m))
}

// GetSigners returns the expected signers for the message.
func (m *OptOutOfAVSReq) GetSigners() []sdk.AccAddress {
	addr := sdk.MustAccAddressFromBech32(m.FromAddress)
	return []sdk.AccAddress{addr}
}

// ValidateBasic does a sanity check of the provided data
func (m *OptOutOfAVSReq) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.FromAddress); err != nil {
		return errorsmod.Wrap(err, "invalid from address")
	}
	if m.AvsAddress == "" {
		return errorsmod.Wrap(ErrParameterInvalid, "AVS address is empty")
	}
	if !common.IsHexAddress(m.AvsAddress) {
		return errorsmod.Wrap(
			ErrParameterInvalid, "AVS address is not a valid hex address or chain id",
		)
	}
	return nil
}

// Route returns the transaction route. This must be specified for successful signing.
func (m *OptOutOfAVSReq) Route() string {
	return RouterKey
}

// Type returns the transaction type.
func (m *OptOutOfAVSReq) Type() string {
	return TypeOptOutOfAVSReq
}

// GetSignBytes returns the bytes all expected signers must sign over.
func (m *OptOutOfAVSReq) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(m))
}
