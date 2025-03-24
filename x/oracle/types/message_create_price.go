package types

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

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

func NewMsgCreatePrice2Phase(creator string, feederID uint64, prices []*PriceSource, basedBlock uint64, nonce int32) *MsgCreatePrice {
	return &MsgCreatePrice{
		Creator:    creator,
		FeederID:   feederID,
		Prices:     prices,
		BasedBlock: basedBlock,
		Nonce:      nonce,
		Phase:      AggregationPhaseOne,
	}
}

func NewMsgCreatePrice2Phase2(creator string, feederID uint64, prices []*PriceSource, basedBlock uint64, nonce int32) *MsgCreatePrice {
	return &MsgCreatePrice{
		Creator:    creator,
		FeederID:   feederID,
		Prices:     prices,
		BasedBlock: basedBlock,
		Nonce:      nonce,
		Phase:      AggregationPhaseTwo,
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
	if len(msg.Prices) == 0 {
		return errors.New("prices should not be empty, at least one source")
	}
	if msg.IsSinglePhase() {
		for _, p := range msg.Prices {
			if len(p.Prices) == 0 {
				return errors.New("prices should not be empty, at least one price")
			}
			deterministic := len(p.Prices[0].DetID) > 0
			seenDetID := make(map[string]struct{})
			for _, pp := range p.Prices {
				if deterministic {
					if len(pp.DetID) == 0 {
						return errors.New("detID should be all set or all empty for the same source")
					}
					if _, ok := seenDetID[pp.DetID]; ok {
						return fmt.Errorf("duplicate detID: %s", pp.DetID)
					}
					seenDetID[pp.DetID] = struct{}{}
				} else if len(pp.Timestamp) == 0 {
					return errors.New("either detID or timestamp should be assigned with valid value")
				}
				if len(pp.Price) == 0 {
					return errors.New("price data should not be empty")
				}
			}
		}
	} else {
		if len(msg.Prices) != 1 {
			return errors.New("2-phases aggregation should have exactly one source")
		}
		if msg.IsPhaseOne() {
			if len(msg.Prices[0].Prices) != 1 {
				return errors.New("message for 1st-phase aggregation should have exactly one price")
			}
			if len(msg.Prices[0].Prices[0].DetID) == 0 {
				return errors.New("detID should not be empty")
			}
			p := msg.Prices[0].Prices[0]
			// rootHash.leafCount, at least 32+1 = 33 bytes
			if len(p.Price) < 33 {
				return fmt.Errorf("price data is too short: %d, expect at least: %d", len(p.Price), 33)
			}
			leafCountStr := p.Price[32:]
			leafCount, err := strconv.ParseUint(leafCountStr, 10, 32)
			if err != nil || leafCount < 1 || leafCount > MaxPieceCount {
				return fmt.Errorf("invalid leafCount: %s, maximum: %d", leafCountStr, MaxPieceCount)
			}
		} else if msg.IsPhaseTwo() {
			l := len(msg.Prices[0].Prices)
			if l < 1 || l > 2 {
				return fmt.Errorf("message for 2nd-phase aggregation should have 1 or 2 prices, got: %d", l)
			}
			piece := msg.Prices[0].Prices[0]
			lPiece := len(piece.Price)
			if lPiece == 0 || lPiece > MaxPieceSize {
				return fmt.Errorf("invalid piece size: %d, maximum: %d", lPiece, MaxPieceSize)
			}
			pieceIndex, err := strconv.ParseUint(piece.DetID, 16, 32)
			if err != nil || pieceIndex >= MaxPieceCount {
				return fmt.Errorf("invalid pieceIndex: %s, maximum: %d", piece.DetID, MaxPieceCount)
			}

			if l == 2 {
				proofHashes := msg.Prices[0].Prices[1].Price
				proofIndexes := msg.Prices[0].Prices[1].DetID
				hashList := strings.Split(proofHashes, DelimiterForBase64)
				indexList := strings.Split(proofIndexes, DelimiterForBase64)
				if len(hashList) != len(indexList) || len(hashList) == 0 {
					return fmt.Errorf("invalid proofHashes: %s, proofIndexes: %s", proofHashes, proofIndexes)
				}
				for _, idx := range indexList {
					_, err := strconv.ParseUint(idx, 16, 32)
					if err != nil {
						return fmt.Errorf("invalid proofIndex: %s", idx)
					}
				}
				for _, hash := range hashList {
					decodedHash, err := base64.StdEncoding.DecodeString(hash)
					if err != nil || len(decodedHash) != 32 {
						return fmt.Errorf("invalid proofHash: hash_base64:%s, error:%s", hash, err)
					}
				}
			}
		}
	}
	return nil
}

func (msg *MsgCreatePrice) IsSinglePhase() bool {
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
