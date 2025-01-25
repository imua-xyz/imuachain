package types

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const TypeMsgPriceFeed = "create_price"

var _ sdk.Msg = &MsgPriceFeed{}

func NewMsgPriceFeed(creator string, feederID uint64, prices []*PriceSource, basedBlock uint64, nonce int32) *MsgPriceFeed {
	return &MsgPriceFeed{
		Creator:    creator,
		FeederID:   feederID,
		Prices:     prices,
		BasedBlock: basedBlock,
		Nonce:      nonce,
	}
}

func (msg *MsgPriceFeed) Route() string {
	return RouterKey
}

func (msg *MsgPriceFeed) Type() string {
	return TypeMsgPriceFeed
}

func (msg *MsgPriceFeed) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgPriceFeed) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgPriceFeed) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid creator address (%s)", err)
	}
	return nil
}
