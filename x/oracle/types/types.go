package types

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type PieceWithProof struct {
	Index   uint32
	RawData []byte
	//	Proof   []*HashNode
	Proof     Proof
	BaseBlock uint64
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
	NSTTokenPrefix         = "nst"
	NSTIDPrefix            = "nst_"
	XChainIDPrefix         = "xchain_"
	ETHChain       NSTType = "eth"
	SOLChain       NSTType = "sol"

	ETHMainnetChainID  = "0x7595"
	ETHLocalnetChainID = "0x65"
	ETHHoleskyChainID  = "0x9d19"
	ETHSepoliaChainID  = "0x9ce1"
	SOLLocalChainID    = "0x123"
	NSTETHAssetAddr    = "0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
	NSTSOLAssetAddr    = "0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"

	DefaultPriceValue   = 0
	DefaultPriceDecimal = 0

	MaxPageLimit = 100

	SourceChainlinkName = "Chainlink"
	SourceChainlinkID   = 1

	RuleIDAll  = 1
	TimeLayout = "2006-01-02 15:04:05"

	DelimiterForCombinedKey = byte('/')

	NilDetID = ""

	DelimiterForBase64 = "|"

	// 8192 pieces with 48,000 bytes piece size can support for 48000*8192 = 393,216,000 bytes = 393.216 MB rawData
	// we limit the pieceCount to 8192 results to max deepth of merkle tree to be 13 excluding the root, which
	// means the max proofPath length is 13 with size of 13*(32+4)= 468 bytes which is acceptable compared to the default pieceSize 48,000 bytes
	// however, 8192 pieces means we need the corresponding tokenfeeder to have a interval of more than 8192 which equals to about 7 hours
	// the bigger the rawData, the longer the interval would be
	MaxPieceCount = 8192
	MaxPieceSize  = 48000
	// as for default limitation, one nst is able to take up to 2.4% capablity of one block, 10 NST would at most take up to 24% of one block
	// 24% means: we got 200000*20*10=40,000,000 validators have their balance changed information at the same block
	MaxNSTCount = 10
	// we limit the max stakers per NST to 200,000 to avoid the rawData of balance change being too large
	// 200,000 stakers with 20 validators per staker could be 4,000,000 validators in total, which is large enough for NST scenario
	// 200,000 stakers roughly would use at most 200,000 * (4+8) = 2,400,000 bytes = 2.4 MB rawData
	// with default 48,000 bytes piece size, it would be 50 pieces results to a tokeFeeder with about 50 intervals which equals to 2.5 minutes frequency of updating
	MaxStakersPerNST = 200_000
	// we limit the max validators per staker to 20 to avoid the validatorList being too large for a staker
	// and the nst balance could be quite big for 20 validators, if a user wants to staker more, they should create another staker
	MaxValidatorsPerStaker = 20
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
		SOLLocalChainID:    SOLChain,
	}
	NSTAssetAddr = map[NSTType]string{
		ETHChain: NSTETHAssetAddr,
		SOLChain: NSTSOLAssetAddr,
	}

	DelimiterForCombinedKeyBytes = []byte{DelimiterForCombinedKey}
)

func Uint64Bytes(value uint64) []byte {
	valueBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(valueBytes, value)
	return valueBytes
}

func BytesToUint64(bz []byte) (uint64, error) {
	if len(bz) < 8 {
		return 0, fmt.Errorf("invalid length of bytes when convert to uint64: %d", len(bz))
	}
	return binary.BigEndian.Uint64(bz), nil
}

func Uint32Bytes(value uint32) []byte {
	valueBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(valueBytes, value)
	return valueBytes
}

func BytesToUint32(bz []byte) (uint32, error) {
	if len(bz) < 4 {
		return 0, fmt.Errorf("invalid length of bytes when convert to uint32: %d", len(bz))
	}
	return binary.BigEndian.Uint32(bz), nil
}

func ConsAddrStrFromCreator(creator string) (string, error) {
	accAddress, err := sdk.AccAddressFromBech32(creator)
	if err != nil {
		return "", err
	}
	return sdk.ConsAddress(accAddress).String(), nil
}

func GetClientChainIDFromNSTAssetID(assetID string) (uint64, bool) {
	if ccIDStr, ok := strings.CutPrefix(strings.ToLower(assetID), NSTIDPrefix); ok {
		ccID, err := hexutil.DecodeUint64(ccIDStr)
		if err == nil {
			return ccID, true
		}
	}
	return 0, false
}

func NSTAssetIDFromClientChainID(chainID uint64) string {
	chainIDStr := hexutil.EncodeUint64(chainID)
	if NSTChain, ok := NSTChainsInverted[chainIDStr]; ok {
		return fmt.Sprintf("%s_%s", NSTAssetAddr[NSTChain], chainIDStr)
	}
	return ""
}

func AppendMultiple(slice []byte, elems ...[]byte) []byte {
	for _, elem := range elems {
		slice = append(slice, elem...)
	}
	return slice
}
