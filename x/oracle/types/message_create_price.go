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

func (msg *MsgCreatePrice) IsNotTwoPhases() bool {
	return msg.Phase == AggregationPhaseUnspecified
}

func (msg *MsgCreatePrice) IsPhaseOne() bool {
	return msg.Phase == AggregationPhaseOne
}

func (msg *MsgCreatePrice) IsPhaseTwo() bool {
	return msg.Phase == AggregationPhaseTwo
}

// NOTE: this should be the only way a MsgCreatePriceRawData is derived
// GetRawData returns wether this is a message with piece of rawData, and parse rawData piece if true
// NOTE: all method for MsgCreatePriceRawData is assumed that the MsgCreatePriceRawData is derived from MsgCreatePrice by 'feederManager.GetRawData' which had done the basic veirfy, so we don't do that repeatedly

func (msgP *MsgCreatePriceRawData) PieceIndex() uint32 {
	return msgP.Piece.Index
}

func (msgP *MsgCreatePriceRawData) GetPieceWithProof() *PieceWithProof {
	return msgP.Piece
}

func (msgP *MsgCreatePriceRawData) GetPieceRawData() []byte {
	return msgP.Piece.RawData
}
