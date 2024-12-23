package types

import (
	"strings"

	assetstypes "github.com/ExocoreNetwork/exocore/x/assets/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// constants
const (
	// ModuleName module name
	ModuleName = "delegation"

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

const (
	prefixRestakerDelegationInfo = iota + 1
	prefixStakersByOperator
	prefixUndelegationInfo

	prefixStakerUndelegationInfo

	prefixPendingUndelegations

	// used to store the undelegation hold count
	prefixUndelegationOnHold

	prefixAssociatedOperatorByStaker
)

var (
	// KeyPrefixRestakerDelegationInfo restakerID = clientChainAddr+'_'+ExoCoreChainIndex
	// KeyPrefixRestakerDelegationInfo
	// key-value:
	// restakerID +'/'+assetID+'/'+operatorAddr -> DelegationAmounts
	KeyPrefixRestakerDelegationInfo = []byte{prefixRestakerDelegationInfo}

	// KeyPrefixStakersByOperator key->value: operatorAddr+'/'+assetID -> stakerList
	KeyPrefixStakersByOperator = []byte{prefixStakersByOperator}

	// KeyPrefixUndelegationInfo singleRecordKey = operatorAddr+'/'+BlockHeight+'/'+LzNonce+'/'+txHash
	// singleRecordKey -> UndelegationRecord
	KeyPrefixUndelegationInfo = []byte{prefixUndelegationInfo}
	// KeyPrefixStakerUndelegationInfo restakerID+'/'+assetID+'/'+LzNonce -> singleRecordKey
	KeyPrefixStakerUndelegationInfo = []byte{prefixStakerUndelegationInfo}
	// KeyPrefixPendingUndelegations completeHeight +'/'+LzNonce -> singleRecordKey
	KeyPrefixPendingUndelegations = []byte{prefixPendingUndelegations}

	// KeyPrefixAssociatedOperatorByStaker stakerID -> operator address
	KeyPrefixAssociatedOperatorByStaker = []byte{prefixAssociatedOperatorByStaker}
)

func IteratorPrefixForStakerAsset(stakerID, assetID string) []byte {
	tmp := []byte(strings.Join([]string{stakerID, assetID}, "/"))
	tmp = append(tmp, '/')
	return tmp
}

func ParseStakerAssetIDAndOperator(key []byte) (keys *SingleDelegationInfoReq, err error) {
	stringList, err := assetstypes.ParseJoinedStoreKey(key, 3)
	if err != nil {
		return nil, err
	}
	return &SingleDelegationInfoReq{StakerId: stringList[0], AssetId: stringList[1], OperatorAddr: stringList[2]}, nil
}

// GetUndelegationRecordKey returns the key for the undelegation record. The caller must ensure that the parameters
// are valid; this function performs no validation whatsoever.
func GetUndelegationRecordKey(blockHeight, lzNonce uint64, txHash string, operatorAddr string) []byte {
	return []byte(strings.Join([]string{operatorAddr, hexutil.EncodeUint64(blockHeight), hexutil.EncodeUint64(lzNonce), txHash}, "/"))
}

type UndelegationKeyFields struct {
	BlockHeight  uint64
	LzNonce      uint64
	TxHash       string
	OperatorAddr string
}

func ParseUndelegationRecordKey(key []byte) (field *UndelegationKeyFields, err error) {
	stringList, err := assetstypes.ParseJoinedStoreKey(key, 4)
	if err != nil {
		return nil, err
	}
	operatorAccAddr, err := sdk.AccAddressFromBech32(stringList[0])
	if err != nil {
		return nil, err
	}
	height, err := hexutil.DecodeUint64(stringList[1])
	if err != nil {
		return nil, err
	}
	lzNonce, err := hexutil.DecodeUint64(stringList[2])
	if err != nil {
		return nil, err
	}
	hash := stringList[3]
	// when a key is originally made, it is created with hash.Hex(), which
	// we reverse by using common.HexToHash. to that end, this validation
	// is accurate.
	if len(common.HexToHash(hash)) != common.HashLength {
		return nil, ErrInvalidHash
	}
	return &UndelegationKeyFields{
		OperatorAddr: operatorAccAddr.String(),
		BlockHeight:  height,
		LzNonce:      lzNonce,
		TxHash:       hash,
	}, nil
}

func GetStakerUndelegationRecordKey(stakerID, assetID string, lzNonce uint64) []byte {
	return []byte(strings.Join([]string{stakerID, assetID, hexutil.EncodeUint64(lzNonce)}, "/"))
}

func GetPendingUndelegationRecordKey(height, lzNonce uint64) []byte {
	return []byte(strings.Join([]string{hexutil.EncodeUint64(height), hexutil.EncodeUint64(lzNonce)}, "/"))
}

// GetUndelegationOnHoldKey returns the key for the undelegation hold count
func GetUndelegationOnHoldKey(recordKey []byte) []byte {
	return append([]byte{prefixUndelegationOnHold}, recordKey...)
}
