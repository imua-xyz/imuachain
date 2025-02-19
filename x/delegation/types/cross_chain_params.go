package types

import (
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	assetstype "github.com/imua-xyz/imuachain/x/assets/types"
)

type DelegationOrUndelegationParams struct {
	ClientChainID   uint64
	Action          assetstype.CrossChainOpType
	AssetsAddress   []byte
	OperatorAddress sdk.AccAddress
	StakerAddress   []byte
	OpAmount        sdkmath.Int
	TxHash          common.Hash
	// todo: The operator approved signature might be needed here in future
}

func NewDelegationOrUndelegationParams(
	clientChainID uint64,
	action assetstype.CrossChainOpType,
	assetsAddress []byte,
	operatorAddress sdk.AccAddress,
	stakerAddress []byte,
	opAmount sdkmath.Int,
	txHash common.Hash,
) *DelegationOrUndelegationParams {
	return &DelegationOrUndelegationParams{
		ClientChainID:   clientChainID,
		Action:          action,
		AssetsAddress:   assetsAddress,
		OperatorAddress: operatorAddress,
		StakerAddress:   stakerAddress,
		OpAmount:        opAmount,
		TxHash:          txHash,
	}
}
