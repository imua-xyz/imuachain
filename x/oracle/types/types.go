package types

import (
	"bytes"
	"encoding/binary"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type PieceWithProof struct {
	Index   uint32
	RawData []byte
	//	Proof   []*HashNode
	Proof Proof
	// reference to the tx including this piece
	Tx sdk.Tx
}

func (p *PieceWithProof) ProofSize() uint32 {
	// #nosec G115
	return uint32(len(p.Proof))
}

func (p *PieceWithProof) HasIndexOnProofPath(index uint32) bool {
	for _, pn := range p.Proof {
		if index == pn.Index {
			return true
		}
	}
	return false
}

func (p *PieceWithProof) EqualsTo(p2 *PieceWithProof) bool {
	if p.Index != p2.Index {
		return false
	}
	if !bytes.Equal(p.RawData, p2.RawData) {
		return false
	}
	if len(p.Proof) != len(p2.Proof) {
		return false
	}

	// we require these to be exactly the same(same order) which is identical with anteHandler proofPath check
	for i, pn := range p.Proof {
		if pn.Index != p2.Proof[i].Index {
			return false
		}
		if !bytes.Equal(pn.Hash, p2.Proof[i].Hash) {
			return false
		}
	}
	return true
}

// MsgCreatePriceRawData defined as alias of MsgCreatePrice with rawData related method to get rid of redundant checking wehter the MsgCreatePrice is with a valid RawData message
// TODO: add filed 'parsed' into this struct to avoid redundant parse
// type MsgCreatePriceRawData MsgCreatePrice
type MsgCreatePriceRawData struct {
	*MsgCreatePrice
	Piece *PieceWithProof
}

type OracleInfo struct {
	Chain struct {
		Name string
		Desc string
	}
	Token struct {
		Name string `json:"name"`
		Desc string
		// Chain struct {
		// 	Name string `json:"name"`
		// 	Desc string `json:"desc"`
		// } `json:"chain"`
		Decimal  string `json:"decimal"`
		Contract string `json:"contract"`
		AssetID  string `json:"asset_id"`
	} `json:"token"`
	Feeder struct {
		// Start    string `json:"start"`
		// End      string `json:"end"`
		Interval string `json:"interval"`
		// RuleID   string `json:"rule_id"`
	} `json:"feeder"`
	AssetID string `json:"asset_id"`
}

type Price struct {
	Value   sdkmath.Int
	Decimal uint8
}

type AggFinalPrice struct {
	FeederID uint64
	SourceID uint64
	DetID    string
	Price    string
}

type NSTType string

const (
	NSTIDPrefix         = "nst_"
	ETHChain    NSTType = "eth"
	SOLANAChain NSTType = "solana"

	ETHMainnetChainID  = "0x7595"
	ETHLocalnetChainID = "0x65"
	ETHHoleskyChainID  = "0x9d19"
	ETHSepoliaChainID  = "0x9ce1"
	NSTETHAssetAddr    = "0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"

	DefaultPriceValue   = 1
	DefaultPriceDecimal = 0

	MaxPageLimit = 100

	SourceChainlinkName = "Chainlink"
	SourceChainlinkID   = 1

	RuleIDAll  = 1
	TimeLayout = "2006-01-02 15:04:05"

	DelimiterForCombinedKey = byte('/')

	NilDetID = ""

	DelimiterForBase64 = "|"
)

var (
	NSTChains = map[NSTType][]string{
		ETHChain: {ETHMainnetChainID, ETHLocalnetChainID, ETHHoleskyChainID, ETHSepoliaChainID},
	}
	NSTChainsInverted = map[string]NSTType{
		ETHMainnetChainID:  ETHChain,
		ETHLocalnetChainID: ETHChain,
		ETHHoleskyChainID:  ETHChain,
		ETHSepoliaChainID:  ETHChain,
	}
	NSTAssetAddr = map[NSTType]string{
		ETHChain: NSTETHAssetAddr,
	}
)

func Uint64Bytes(value uint64) []byte {
	valueBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(valueBytes, value)
	return valueBytes
}

func BytesToUint64(bz []byte) uint64 {
	return binary.BigEndian.Uint64(bz)
}

func Uint32Bytes(value uint32) []byte {
	valueBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(valueBytes, value)
	return valueBytes
}

func BytesToUint32(bz []byte) uint32 {
	return binary.BigEndian.Uint32(bz)
}

func ConsAddrStrFromCreator(creator string) (string, error) {
	accAddress, err := sdk.AccAddressFromBech32(creator)
	if err != nil {
		return "", err
	}
	return sdk.ConsAddress(accAddress).String(), nil
}
