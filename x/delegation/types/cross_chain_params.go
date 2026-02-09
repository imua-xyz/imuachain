package types

import (
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	assetstype "github.com/imua-xyz/imuachain/x/assets/types"
)

type DelegationOrUndelegationParams struct {
	ClientChainID   uint64
	AssetsAddress   []byte
	OperatorAddress sdk.AccAddress
	StakerAddress   []byte
	OpAmount        sdkmath.Int
	TxHash          common.Hash
	// todo: The operator approved signature might be needed here in future

	// indicator for instant unbonding, default is false.
	InstantUnbonding bool
	// indicates whether it's a delegation/undelegation regarding reward.
	RewardAsset           bool
	RewardAssetID         string
	RewardStakerID        string
	ReduceDelegationShare func(ctx sdk.Context, stakerID, assetID string,
		operatorAccAddr sdk.AccAddress, instantSlashRatio sdk.Dec, amount sdkmath.Int, preOperatorAssetState assetstype.OperatorAssetInfo) ([]UndelegationAmountPerAVS, sdkmath.Int, error)
	RewardUndelegations []UndelegationAmountPerAVS
}

func NewDelegationOrUndelegationParams(
	clientChainID uint64,
	assetsAddress []byte,
	operatorAddress sdk.AccAddress,
	stakerAddress []byte,
	opAmount sdkmath.Int,
	txHash common.Hash,
	instantUnbonding bool,
) *DelegationOrUndelegationParams {
	return &DelegationOrUndelegationParams{
		ClientChainID:    clientChainID,
		AssetsAddress:    assetsAddress,
		OperatorAddress:  operatorAddress,
		StakerAddress:    stakerAddress,
		OpAmount:         opAmount,
		TxHash:           txHash,
		InstantUnbonding: instantUnbonding,
	}
}
