package types

import (
	"fmt"
	"strings"

	"github.com/imua-xyz/imuachain/utils"
	"golang.org/x/xerrors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/ethereum/go-ethereum/common"
	assetstypes "github.com/imua-xyz/imuachain/x/assets/types"
)

// constants
const (
	// ModuleName module name
	ModuleName = "delegation"

	// StoreKey to be used when creating the KVStore
	StoreKey = ModuleName

	// RouterKey to be used for message routing
	RouterKey = ModuleName

	// AccAddressLength is used to parse the key, because the length isn't padded in the key
	// This might be removed if the address length is padded in the key
	AccAddressLength = 20

	// ByteLengthForUint64 the type of chainID length is uint64, uint64 has 8 bytes.
	ByteLengthForUint64 = 8
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

	prefixForLastUndelegationID
)

var (
	// KeyPrefixRestakerDelegationInfo restakerID = clientChainAddr+'_'+ImuachainIndex
	// KeyPrefixRestakerDelegationInfo
	// key-value:
	// restakerID +'/'+assetID+'/'+operatorAddr -> DelegationAmounts
	KeyPrefixRestakerDelegationInfo = []byte{prefixRestakerDelegationInfo}

	// KeyPrefixStakersByOperator key->value: operatorAddr+'/'+assetID -> stakerList
	KeyPrefixStakersByOperator = []byte{prefixStakersByOperator}

	// KeyPrefixLastUndelegationID key-value:
	// prefixForLastUndelegationID -> uint64
	// We use an incrementing number to identify undelegations because we support different
	// assets across multiple client chains and the Imuachain chain.
	KeyPrefixLastUndelegationID = []byte{prefixForLastUndelegationID}

	// KeyPrefixUndelegationInfo singleRecordKey = operatorAddr+BlockHeight+UndelegationID+txHash
	// it can be constructed by GetUndelegationRecordKey
	// singleRecordKey -> UndelegationRecord
	KeyPrefixUndelegationInfo = []byte{prefixUndelegationInfo}
	// KeyPrefixStakerUndelegationInfo restakerID+'/'+assetID+'/'+UndelegationID -> singleRecordKey
	KeyPrefixStakerUndelegationInfo = []byte{prefixStakerUndelegationInfo}
	// KeyPrefixPendingUndelegations
	// key=epochIdentifierLength+completedEpochIdentifier+completedEpochNumber+UndelegationID
	// it can be constructed by GetPendingUndelegationRecordKey
	// key -> singleRecordKey
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
func GetUndelegationRecordKey(blockHeight, undelegationID uint64, txHash string, operatorAddr string) []byte {
	// we can use `Must` here because we stored this record ourselves.
	operatorAccAddress := sdk.MustAccAddressFromBech32(operatorAddr)
	return utils.AppendMany(
		// operator address,20bytes
		operatorAccAddress,
		// Append the height,8bytes
		sdk.Uint64ToBigEndian(blockHeight),
		// Append the undelegationID,8bytes
		sdk.Uint64ToBigEndian(undelegationID),
		// Append txHash,32bytes
		common.HexToHash(txHash).Bytes(),
	)
}

// GetKey returns the key for the undelegation record
func (r *UndelegationRecord) GetKey() []byte {
	return GetUndelegationRecordKey(r.BlockNumber, r.UndelegationId, r.TxHash, r.OperatorAddr)
}

type UndelegationKeyFields struct {
	BlockHeight    uint64
	UndelegationID uint64
	TxHash         string
	OperatorAddr   string
}

func ParseUndelegationRecordKey(key []byte) (field *UndelegationKeyFields, err error) {
	expectLength := AccAddressLength + 2*ByteLengthForUint64 + common.HashLength
	if len(key) != expectLength {
		return nil, xerrors.Errorf(
			"invalid undelegation record key, expectedLength:%d,actualLength:%d",
			expectLength, len(key))
	}
	// operator accAddress: 20bytes
	startIndex := 0
	operatorAccAddr := sdk.AccAddress(key[startIndex : startIndex+AccAddressLength])
	// the height type is uint64: 8bytes
	startIndex += AccAddressLength
	height := sdk.BigEndianToUint64(key[startIndex : startIndex+ByteLengthForUint64])
	// the undelegationID type is uint64: 8bytes
	startIndex += ByteLengthForUint64
	undelegationID := sdk.BigEndianToUint64(key[startIndex : startIndex+ByteLengthForUint64])
	// txHash: 32bytes
	startIndex += ByteLengthForUint64
	txHash := common.BytesToHash(key[startIndex : startIndex+common.HashLength])
	return &UndelegationKeyFields{
		OperatorAddr:   operatorAccAddr.String(),
		BlockHeight:    height,
		UndelegationID: undelegationID,
		TxHash:         txHash.String(),
	}, nil
}

func GetStakerUndelegationRecordKey(stakerID, assetID string, undelegationID uint64) []byte {
	idStr := fmt.Sprintf("%021d", undelegationID)
	return []byte(strings.Join([]string{stakerID, assetID, idStr}, "/"))
}

type PendingUndelegationKeyFields struct {
	EpochIdentifier string
	EpochNumber     uint64
	UndelegationID  uint64
}

func GetPendingUndelegationRecordKey(epochIdentifier string, epochNumber int64, undelegationID uint64) []byte {
	return utils.AppendMany(
		// length of identifier,8bytes
		sdk.Uint64ToBigEndian(uint64(len(epochIdentifier))),
		// epoch identifier, length = len(epochIdentifier)
		[]byte(epochIdentifier),
		// Append the epoch number,8bytes
		sdk.Uint64ToBigEndian(uint64(epochNumber)),
		// Append the undelegationID,8bytes
		sdk.Uint64ToBigEndian(undelegationID),
	)
}

func ParsePendingUndelegationKey(key []byte) (field *PendingUndelegationKeyFields, err error) {
	if len(key) <= 3*ByteLengthForUint64 {
		return nil, xerrors.New("ParsePendingUndelegationKey,key length is too short to contain epoch info and nonce")
	}
	identifierLen := sdk.BigEndianToUint64(key[0:ByteLengthForUint64])
	if uint64(len(key)) != uint64(3*ByteLengthForUint64)+identifierLen {
		return nil, xerrors.Errorf("ParsePendingUndelegationKey,key length is invalid,expect:%d,actual:%d", uint64(3*ByteLengthForUint64)+identifierLen, len(key))
	}
	epochIdentifier := string(key[ByteLengthForUint64 : ByteLengthForUint64+identifierLen])
	epochNumber := sdk.BigEndianToUint64(key[ByteLengthForUint64+identifierLen : ByteLengthForUint64*2+identifierLen])
	undelegationID := sdk.BigEndianToUint64(key[ByteLengthForUint64*2+identifierLen:])
	return &PendingUndelegationKeyFields{
		EpochIdentifier: epochIdentifier,
		EpochNumber:     epochNumber,
		UndelegationID:  undelegationID,
	}, nil
}

// GetUndelegationOnHoldKey returns the key for the undelegation hold count
func GetUndelegationOnHoldKey(recordKey []byte) []byte {
	return append([]byte{prefixUndelegationOnHold}, recordKey...)
}
