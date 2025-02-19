package types

import (
	"encoding/binary"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

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
	TimeLayout          = "2006-01-02 15:04:05"

	DelimiterForCombinedKey = byte('/')
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

func ConsAddrStrFromCreator(creator string) (string, error) {
	accAddress, err := sdk.AccAddressFromBech32(creator)
	if err != nil {
		return "", err
	}
	return sdk.ConsAddress(accAddress).String(), nil
}
