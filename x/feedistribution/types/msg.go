package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	assetstypes "github.com/imua-xyz/imuachain/x/assets/types"
)

const (
	// TypeMsgUpdateParams is the type for the MsgUpdateParams tx.
	TypeMsgUpdateParams                  = "update_params"
	TypeMsgWithdrawDogfoodCommission     = "withdraw_dogfood_commission"
	TypeMsgClaimAndWithdrawDogfoodReward = "claim_and_withdraw_dogfood_reward"
	TypeMsgUpdateStakerRewardParams      = "update_staker_reward_params"
	TypeMsgUndelegateReward              = "undelegate_reward"
)

var (
	_ sdk.Msg = &MsgUpdateParams{}
	_ sdk.Msg = &MsgWithdrawDogfoodCommission{}
	_ sdk.Msg = &MsgClaimAndWithdrawDogfoodReward{}
	_ sdk.Msg = &MsgUpdateStakerRewardParams{}
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
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(m))
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
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(m))
}

// ValidateBasic does a sanity check on the provided data.
func (m *MsgClaimAndWithdrawDogfoodReward) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.FromAddress); err != nil {
		return errorsmod.Wrap(err, "invalid operator address")
	}
	if m.Amount.IsNil() || m.Amount.IsNegative() {
		return ErrInvalidInputParameter.Wrapf("invalid amount:%v", m.Amount)
	}
	return nil
}

// Route returns the transaction route.
func (m *MsgClaimAndWithdrawDogfoodReward) Route() string {
	return RouterKey
}

// Type returns the transaction type.
func (m *MsgClaimAndWithdrawDogfoodReward) Type() string {
	return TypeMsgClaimAndWithdrawDogfoodReward
}

// GetSigners returns the expected signers for a MsgUpdateParams message.
func (m *MsgClaimAndWithdrawDogfoodReward) GetSigners() []sdk.AccAddress {
	addr := sdk.MustAccAddressFromBech32(m.FromAddress)
	return []sdk.AccAddress{addr}
}

// GetSignBytes implements the LegacyMsg interface.
func (m *MsgClaimAndWithdrawDogfoodReward) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(m))
}

// ValidateBasic does a sanity check on the provided data.
func (m *MsgUpdateStakerRewardParams) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.FromAddress); err != nil {
		return errorsmod.Wrap(err, "invalid from address")
	}
	err := m.RewardParams.Validate()
	if err != nil {
		return ErrInvalidInputParameter.Wrapf("invalid reward parameters,err:%s", err)
	}
	return nil
}

// Route returns the transaction route.
func (m *MsgUpdateStakerRewardParams) Route() string {
	return RouterKey
}

// Type returns the transaction type.
func (m *MsgUpdateStakerRewardParams) Type() string {
	return TypeMsgUpdateStakerRewardParams
}

// GetSigners returns the expected signers for a MsgUpdateParams message.
func (m *MsgUpdateStakerRewardParams) GetSigners() []sdk.AccAddress {
	addr := sdk.MustAccAddressFromBech32(m.FromAddress)
	return []sdk.AccAddress{addr}
}

// GetSignBytes implements the LegacyMsg interface.
func (m *MsgUpdateStakerRewardParams) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(m))
}

// ValidateBasic does a sanity check on the provided data.
func (m *MsgUndelegateReward) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.FromAddress); err != nil {
		return errorsmod.Wrap(err, "invalid from address")
	}
	_, _, err := assetstypes.ValidateID(m.AssetId, false, false)
	if err != nil {
		return ErrInvalidInputParameter.Wrapf("invalid assetID:%s,err:%s", m.AssetId, err)
	}
	_, err = sdk.AccAddressFromBech32(m.OperatorAddr)
	if err != nil {
		return ErrInvalidInputParameter.Wrapf("invalid operator address:%s,err:%s", m.OperatorAddr, err)
	}
	if m.Amount.IsNil() || !m.Amount.IsPositive() {
		return ErrInvalidInputParameter.Wrapf("invalid amount:%v", m.Amount)
	}
	return nil
}

// Route returns the transaction route.
func (m *MsgUndelegateReward) Route() string {
	return RouterKey
}

// Type returns the transaction type.
func (m *MsgUndelegateReward) Type() string {
	return TypeMsgUndelegateReward
}

// GetSigners returns the expected signers for a MsgUpdateParams message.
func (m *MsgUndelegateReward) GetSigners() []sdk.AccAddress {
	addr := sdk.MustAccAddressFromBech32(m.FromAddress)
	return []sdk.AccAddress{addr}
}

// GetSignBytes implements the LegacyMsg interface.
func (m *MsgUndelegateReward) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(m))
}
