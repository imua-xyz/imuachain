package types

import "strings"

const (
	// NativeTokenKeyPrefix is the prefix to retrieve all NativeToken
	NativeTokenKeyPrefix           = "NativeToken/"
	NativeTokenPriceKeyPrefix      = NativeTokenKeyPrefix + "price/value/"
	NativeTokenStakerInfoKeyPrefix = NativeTokenKeyPrefix + "stakerInfo/value/"
	NativeTokenStakerListKeyPrefix = NativeTokenKeyPrefix + "stakerList/value/"
	NativeTokenVersionKeyPrefix    = NativeTokenKeyPrefix + "version/"
)

// NativeTokenStakerKeyPrefix returns the prefix for stakerInfo key
// NativetToken/stakerInfo/value/assetID/
func NativeTokenStakerKeyPrefix(assetID string) []byte {
	if len(assetID) == 0 {
		return []byte(NativeTokenStakerInfoKeyPrefix)
	}
	assetID += "/"
	return append([]byte(NativeTokenStakerInfoKeyPrefix), []byte(assetID)...)
}

// NativeTokenStakerKey returns stakerKey
// NativeToken/stakerInfo/value/assetID/stakerAddr
func NativeTokenStakerKey(assetID, stakerAddr string) []byte {
	return append(NativeTokenStakerKeyPrefix(assetID), []byte(stakerAddr)...)
}

// NativeTokenStakerListKey returns stakerList key
// NativeToken/stakerList/value/assetID
func NativeTokenStakerListKey(assetID string) []byte {
	return append([]byte(NativeTokenStakerListKeyPrefix), []byte(assetID)...)
}

// ParseNativeTokenStakerKey retieve assetID and stakerAddr from stakerInfoKey
// assetID/stakerAddr -> {assetID, stakerAddr}
func ParseNativeTokenStakerKey(key []byte) (assetID, stakerAddr string) {
	parsed := strings.Split(string(key), "/")
	if len(parsed) != 2 {
		panic("key of stakerInfo must be construct by 2 infos: assetID/stakerAddr")
	}
	return parsed[0], parsed[1]
}

func NativeTokenVersionKey(assetID string) []byte {
	return append([]byte(NativeTokenVersionKeyPrefix), []byte(assetID)...)
}
