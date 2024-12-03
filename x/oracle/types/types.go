package types

import (
	"encoding/binary"

	sdkmath "cosmossdk.io/math"
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

const (
	DefaultPriceValue   = 1
	DefaultPriceDecimal = 0

	NSTIDPrefix = "NST"
)

var DelimiterForCombinedKey = byte('/')

func Uint64Bytes(value uint64) []byte {
	valueBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(valueBytes, value)
	return valueBytes
}
