package types

import (
	"fmt"
	"strings"

	"github.com/imua-xyz/imuachain/utils"

	errorsmod "cosmossdk.io/errors"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// constants
const (
	// ModuleName module name
	ModuleName = "assets"

	// StoreKey to be used when creating the KVStore
	StoreKey = ModuleName

	// RouterKey to be used for message routing
	RouterKey = ModuleName
)

// ModuleAddress is the native module address for EVM
var ModuleAddress common.Address

func init() {
	ModuleAddress = common.BytesToAddress(authtypes.NewModuleAddress(ModuleName).Bytes())
}

// prefix bytes for the reStaking assets manage store
const (
	prefixClientChainInfo = iota + 1
	prefixRestakingAssetInfo
	prefixRestakerAssetInfo
	prefixOperatorAssetInfo
	prefixOperatorOptedInMiddlewareAssetInfo

	// unused prefixes for backward compatibility
	// originally: KeyPrefixReStakerExoCoreAddr and Reverse
	_
	_

	prefixParams
)

// KVStore key prefixes
var (
	// KeyPrefixClientChainInfo key->value: clientChainID -> ClientChainInfo
	KeyPrefixClientChainInfo = []byte{prefixClientChainInfo}

	// KeyPrefixReStakingAssetInfo AssetID = AssetAddr+'_'+ clientChainID
	// KeyPrefixReStakingAssetInfo key->value: AssetID-> StakingAssetInfo
	// the `_` will only be used as a separated character for the stakerID and assetID
	KeyPrefixReStakingAssetInfo = []byte{prefixRestakingAssetInfo}

	// KeyPrefixReStakerAssetInfos restakerID = clientChainAddr+'_'+clientChainID
	// KeyPrefixReStakerAssetInfos key->value: restakerID+'/'+AssetID-> StakerAssetInfo
	// the `/` will be used as a separated character for the other joined keys.
	KeyPrefixReStakerAssetInfos = []byte{prefixRestakerAssetInfo}

	// KeyPrefixOperatorAssetInfos key->value: operatorAddr+'/'+AssetID-> OperatorAssetInfo
	// or operatorAddr->mapping(AssetID->OperatorAssetInfo) ?
	KeyPrefixOperatorAssetInfos = []byte{prefixOperatorAssetInfo}

	// KeyPrefixParams This is a key prefix for module parameter
	KeyPrefixParams = []byte{prefixParams}
	ParamsKey       = []byte("Params")
)

func GetJoinedStoreKey(keys ...string) []byte {
	return []byte(strings.Join(keys, utils.DelimiterForCombinedKey))
}

func GetJoinedStoreKeyForPrefix(keys ...string) []byte {
	ret := []byte(strings.Join(keys, utils.DelimiterForCombinedKey))
	ret = append(ret, []byte(utils.DelimiterForCombinedKey)...)
	return ret
}

func ParseJoinedKey(key []byte) (keys []string, err error) {
	stringList := strings.Split(string(key), utils.DelimiterForCombinedKey)
	return stringList, nil
}

func IsJoinedStoreKey(key string) bool {
	return strings.Contains(key, utils.DelimiterForCombinedKey)
}

func ParseJoinedStoreKey(key []byte, number int) (keys []string, err error) {
	stringList := strings.Split(string(key), utils.DelimiterForCombinedKey)
	if len(stringList) != number {
		return nil, errorsmod.Wrap(
			ErrParseJoinedKey,
			fmt.Sprintf(
				"expected length:%d,actual length:%d,the stringList is:%v",
				number,
				len(stringList),
				stringList,
			),
		)
	}
	return stringList, nil
}

// ParseID parses the key and returns the client address and the ID.
// It constraints the key to be in the format of "clientAddress_0xid"
// The 0xid must be in hex.
func ParseID(key string) (string, uint64, error) {
	keys := strings.Split(key, utils.DelimiterForID)
	if len(keys) != 2 {
		return "", 0, errorsmod.Wrap(ErrParseAssetsStateKey, fmt.Sprintf("invalid length:%s", key))
	}
	if len(keys[0]) == 0 {
		return "", 0, errorsmod.Wrap(ErrParseAssetsStateKey, fmt.Sprintf("empty key:%s", key))
	}
	var id uint64
	var err error
	if id, err = hexutil.DecodeUint64(keys[1]); err != nil {
		return "", 0, errorsmod.Wrap(ErrParseAssetsStateKey, fmt.Sprintf("not a number :%s", key))
	}
	return keys[0], id, nil
}

// ValidateID validates the key and returns the client address and the ID
// The flags used by it are (1) checkLowercase and (2) validateEth.
// If the former is true, it will check that the provided key is lowercase.
// If the latter is true, it will check that the parsed address is a valid Ethereum address.
func ValidateID(key string, checkLowercase bool, validateEth bool) (string, uint64, error) {
	if checkLowercase && key != strings.ToLower(key) {
		return "", 0, errorsmod.Wrapf(ErrParseAssetsStateKey, "ID not lowercase: %s", key)
	}
	// parse it
	var clientAddress string
	var lzID uint64
	var err error
	if clientAddress, lzID, err = ParseID(key); err != nil {
		return "", 0, errorsmod.Wrapf(
			ErrParseAssetsStateKey, "invalid key: %s", key,
		)
	}
	// check hex address
	if validateEth && !common.IsHexAddress(clientAddress) {
		return "", 0, errorsmod.Wrapf(
			ErrParseAssetsStateKey, "not hex address %s: %s",
			key, clientAddress,
		)
	}
	return clientAddress, lzID, nil
}
