package types

import (
	errorsmod "cosmossdk.io/errors"
	assetstype "github.com/ExocoreNetwork/exocore/x/assets/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	_ sdk.Msg = &MsgDelegation{}
	_ sdk.Msg = &MsgUndelegation{}
)

// GetSigners returns the expected signers for a MsgUpdateParams message.
func (m *MsgDelegation) GetSigners() []sdk.AccAddress {
	addr := sdk.MustAccAddressFromBech32(m.BaseInfo.FromAddress)
	return []sdk.AccAddress{addr}
}

// ValidateBasic does a sanity check of the provided data
func (m *MsgDelegation) ValidateBasic() error {
	return validateDelegationInfo(m.AssetId, m.BaseInfo)
}

// new message to delegate asset to operator
func NewMsgDelegation(
	assetID string, fromAddress string, amountPerOperator []KeyValue,
) *MsgDelegation {
	baseInfo := &DelegationIncOrDecInfo{
		FromAddress:        fromAddress,
		PerOperatorAmounts: make([]KeyValue, 0, 1),
	}
	for _, kv := range amountPerOperator {
		baseInfo.PerOperatorAmounts = append(
			baseInfo.PerOperatorAmounts,
			KeyValue{Key: kv.Key, Value: kv.Value},
		)
	}
	return &MsgDelegation{
		AssetId:  assetID,
		BaseInfo: baseInfo,
	}
}

// GetSignBytes implements the LegacyMsg interface.
func (m *MsgDelegation) GetSignBytes() []byte {
	return nil
}

// GetSigners returns the expected signers for a MsgUpdateParams message.
func (m *MsgUndelegation) GetSigners() []sdk.AccAddress {
	addr := sdk.MustAccAddressFromBech32(m.BaseInfo.FromAddress)
	return []sdk.AccAddress{addr}
}

// ValidateBasic does a sanity check of the provided data
func (m *MsgUndelegation) ValidateBasic() error {
	return validateDelegationInfo(m.AssetId, m.BaseInfo)
}

// GetSignBytes implements the LegacyMsg interface.
func (m *MsgUndelegation) GetSignBytes() []byte {
	return nil
}

// new message to delegate asset to operator
func NewMsgUndelegation(assetID, fromAddress string, amountPerOperator []KeyValue) *MsgUndelegation {
	baseInfo := &DelegationIncOrDecInfo{
		FromAddress:        fromAddress,
		PerOperatorAmounts: make([]KeyValue, 0, 1),
	}
	for _, kv := range amountPerOperator {
		baseInfo.PerOperatorAmounts = append(
			baseInfo.PerOperatorAmounts,
			KeyValue{Key: kv.Key, Value: kv.Value},
		)
	}
	return &MsgUndelegation{
		AssetId:  assetID,
		BaseInfo: baseInfo,
	}
}

// validateDelegationInfo validates the delegation or undelegation info.
// (1) the operator amounts are positive, and the operator addresses are valid.
// (2) the assetID is native only, since only native token is supported for this mechanism.
// (3) the from address is valid.
// TODO: delegation and undelegation have the same params, try to use one single message with
// different flag to indicate action:delegation/undelegation
func validateDelegationInfo(assetID string, baseInfo *DelegationIncOrDecInfo) error {
	for _, kv := range baseInfo.PerOperatorAmounts {
		if _, err := sdk.AccAddressFromBech32(kv.Key); err != nil {
			return errorsmod.Wrap(err, "invalid operator address delegateTO")
		}
		if !kv.Value.Amount.IsPositive() {
			return ErrAmountIsNotPositive.Wrapf(
				"amount should be positive, got %s", kv.Value.Amount.String(),
			)
		}
	}
	if assetID != assetstype.ExocoreAssetID {
		return ErrInvalidAssetID.Wrapf(
			"only nativeToken is support, expected:%s,got:%s", assetstype.ExocoreAssetID, assetID,
		)
	}
	if _, err := sdk.AccAddressFromBech32(baseInfo.FromAddress); err != nil {
		return errorsmod.Wrap(err, "invalid from address")
	}
	return nil
}
