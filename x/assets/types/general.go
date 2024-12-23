package types

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/ExocoreNetwork/exocore/utils"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

const (
	ExocoreChainLzID  = 0
	ExocoreAssetAddr  = "0x0000000000000000000000000000000000000000"
	ExocoreAssetID    = "0x0000000000000000000000000000000000000000_0x0"
	ExocoreAssetDenom = utils.BaseDenom

	FillCharForRestakingAssetAddr = 0xee
)

const (
	CrossChainActionLength       = 1
	CrossChainOpAmountLength     = 32
	GeneralAssetsAddrLength      = 32
	GeneralClientChainAddrLength = 32
	ClientChainLzIDIndexInTopics = 0
	ExoCoreOperatorAddrLength    = 42

	// MaxDecimal is set to prevent the overflow
	// during the calculation of share and usd value.
	MaxDecimal                  = 18
	MaxChainTokenNameLength     = 50
	MaxChainTokenMetaInfoLength = 200

	MinClientChainAddrLength = 20
)

const (
	DepositLST CrossChainOpType = iota
	WithdrawLST
	DepositNST
	WithdrawNST
	WithdrawReward
	DelegateTo
	UndelegateFrom
	Slash
)

type GeneralAssetsAddr [32]byte

type GeneralClientChainAddr [32]byte

type CrossChainOpType uint8

type WithdrawerAddress [32]byte

// DeltaStakerSingleAsset This is a struct to describe the desired change that matches with
// the StakerAssetInfo
type DeltaStakerSingleAsset StakerAssetInfo

// DeltaOperatorSingleAsset This is a struct to describe the desired change that matches
// with the OperatorAssetInfo
type DeltaOperatorSingleAsset OperatorAssetInfo

type CreateQueryContext func(height int64, prove bool) (sdk.Context, error)

// StakerBalance is a struct to describe the balance of a staker for a specific asset
// balance = withdrawable + delegated + pendingUndelegated
// pendingUndelegated is the amount of the asset that is during unbonding period and not yet withdrawable
// it would finally be withdrawable after the unbonding period, but the final amount may be less than the pendingUndelegated
// because of the penalty during the unbonding period
type StakerBalance struct {
	StakerID           string
	AssetID            string
	Balance            *big.Int
	Withdrawable       *big.Int
	Delegated          *big.Int
	PendingUndelegated *big.Int
	TotalDeposited     *big.Int
}

// GetStakerIDAndAssetID stakerID = stakerAddress+'_'+clientChainLzID,assetID =
// assetAddress+'_'+clientChainLzID
func GetStakerIDAndAssetID(
	clientChainLzID uint64,
	stakerAddress []byte,
	assetsAddress []byte,
) (stakerID string, assetID string) {
	clientChainLzIDStr := hexutil.EncodeUint64(clientChainLzID)
	if stakerAddress != nil {
		stakerID = strings.Join([]string{hexutil.Encode(stakerAddress), clientChainLzIDStr}, utils.DelimiterForID)
	}

	if assetsAddress != nil {
		assetID = strings.Join([]string{hexutil.Encode(assetsAddress), clientChainLzIDStr}, utils.DelimiterForID)
	}
	return
}

// GetStakerIDAndAssetIDFromStr stakerID = stakerAddress+'_'+clientChainLzID,assetID =
// assetAddress+'_'+clientChainLzID, NOTE: the stakerAddress and assetsAddress should be in hex format
func GetStakerIDAndAssetIDFromStr(
	clientChainLzID uint64,
	stakerAddress string,
	assetsAddress string,
) (stakerID string, assetID string) {
	// hexutil always returns lowercase values
	clientChainLzIDStr := hexutil.EncodeUint64(clientChainLzID)
	if stakerAddress != "" {
		stakerID = strings.Join(
			[]string{strings.ToLower(stakerAddress), clientChainLzIDStr},
			utils.DelimiterForID,
		)
	}

	if assetsAddress != "" {
		assetID = strings.Join(
			[]string{strings.ToLower(assetsAddress), clientChainLzIDStr},
			utils.DelimiterForID,
		)
	}
	return
}

// UpdateAssetValue It's used to update asset state,negative or positive `changeValue`
// represents a decrease or increase in the asset state
// newValue = valueToUpdate + changeVale
func UpdateAssetValue(valueToUpdate *math.Int, changeValue *math.Int) error {
	if valueToUpdate == nil || changeValue == nil {
		return errorsmod.Wrap(
			ErrInputPointerIsNil,
			fmt.Sprintf("valueToUpdate:%v,changeValue:%v", valueToUpdate, changeValue),
		)
	}

	if !changeValue.IsNil() {
		if changeValue.IsNegative() {
			if valueToUpdate.LT(changeValue.Neg()) {
				return errorsmod.Wrap(
					ErrSubAmountIsMoreThanOrigin,
					fmt.Sprintf(
						"valueToUpdate:%s,changeValue:%s",
						*valueToUpdate,
						*changeValue,
					),
				)
			}
		}
		if !changeValue.IsZero() {
			*valueToUpdate = valueToUpdate.Add(*changeValue)
		}
	}
	return nil
}

// UpdateAssetDecValue It's used to update asset state,negative or positive `changeValue`
// represents a decrease or increase in the asset state
// newValue = valueToUpdate + changeVale
func UpdateAssetDecValue(valueToUpdate *math.LegacyDec, changeValue *math.LegacyDec) error {
	if valueToUpdate == nil || changeValue == nil {
		return errorsmod.Wrap(
			ErrInputPointerIsNil,
			fmt.Sprintf("valueToUpdate:%v,changeValue:%v", valueToUpdate, changeValue),
		)
	}

	if !changeValue.IsNil() {
		if changeValue.IsNegative() {
			if valueToUpdate.LT(changeValue.Neg()) {
				return errorsmod.Wrap(
					ErrSubAmountIsMoreThanOrigin,
					fmt.Sprintf(
						"valueToUpdate:%s,changeValue:%s",
						*valueToUpdate,
						*changeValue,
					),
				)
			}
		}
		if !changeValue.IsZero() {
			*valueToUpdate = valueToUpdate.Add(*changeValue)
		}
	}
	return nil
}

// GenerateNSTAddr we use a virtual address that is padding by 0xee
// to represent the address of native restaking asset. It's okay because we can distinguish
// which client chain's native asset it is through the clientChainID in the assetID.
func GenerateNSTAddr(clientChainAddrLength uint32) []byte {
	address := make([]byte, clientChainAddrLength)
	for i := range address {
		address[i] = FillCharForRestakingAssetAddr
	}
	return address
}

func IsNST(assetID string) bool {
	assetAddr, _, err := ParseID(assetID)
	if err != nil {
		return false
	}
	addressBytes, err := hexutil.Decode(assetAddr)
	if err != nil {
		return false
	}
	isNativeRestakingAsset := true
	for i := range addressBytes {
		if addressBytes[i] != FillCharForRestakingAssetAddr {
			isNativeRestakingAsset = false
			break
		}
	}
	return isNativeRestakingAsset
}
