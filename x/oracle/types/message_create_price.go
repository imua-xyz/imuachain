package types

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const TypeMsgCreatePrice = "create_price"

var _ sdk.Msg = &MsgCreatePrice{}

func NewMsgCreatePrice(creator string, feederID uint64, prices []*PriceSource, basedBlock uint64, nonce int32) *MsgCreatePrice {
	return &MsgCreatePrice{
		Creator:    creator,
		FeederID:   feederID,
		Prices:     prices,
		BasedBlock: basedBlock,
		Nonce:      nonce,
	}
}

func (msg *MsgCreatePrice) Route() string {
	return RouterKey
}

func (msg *MsgCreatePrice) Type() string {
	return TypeMsgCreatePrice
}

func (msg *MsgCreatePrice) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgCreatePrice) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgCreatePrice) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid creator address (%s)", err)
	}
	return nil
}
