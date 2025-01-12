package keeper

import (
	context "context"
	"fmt"

	assetstypes "github.com/ExocoreNetwork/exocore/x/assets/types"
	"github.com/ExocoreNetwork/exocore/x/delegation/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/minio/sha256-simd"
)

var _ types.MsgServer = &Keeper{}

// DelegateAssetToOperator delegates asset to operator. Currently, it only supports native token
func (k *Keeper) DelegateAssetToOperator(
	goCtx context.Context, msg *types.MsgDelegation,
) (*types.DelegationResponse, error) {
	// TODO: currently we only support delegation with native token by invoking service
	ctx := sdk.UnwrapSDKContext(goCtx)
	logger := k.Logger(ctx)
	// no need to validate whether assetID == native token, since that is done by ValidateBasic.
	logger.Info("DelegateAssetToOperator-nativeToken", "msg", msg)

	delegationParamsList := newDelegationParams(
		msg.BaseInfo, assetstypes.ExocoreAssetAddr, assetstypes.ExocoreChainLzID, common.Hash{},
	)
	cachedCtx, writeFunc := ctx.CacheContext()
	for _, delegationParams := range delegationParamsList {
		if err := k.DelegateTo(cachedCtx, delegationParams); err != nil {
			logger.Error(
				"failed to delegate asset",
				"error", err,
				"delegationParams", delegationParams,
			)
			// fail all delegations if one fails
			return nil, err
		}
	}
	writeFunc()

	return &types.DelegationResponse{}, nil
}

// UndelegateAssetFromOperator undelegates asset from operator. Currently, it only supports
// native token.
func (k *Keeper) UndelegateAssetFromOperator(
	goCtx context.Context, msg *types.MsgUndelegation,
) (*types.UndelegationResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	logger := k.Logger(ctx)
	logger.Info("UndelegateAssetFromOperator", "msg", msg)
	// can use `Must` since pre-validated
	fromAddr := sdk.MustAccAddressFromBech32(msg.BaseInfo.FromAddress)
	// no need to check that `assetID` is native token, since that is done by ValidateBasic.
	// create nonce and unique hash
	nonce, err := k.accountKeeper.GetSequence(ctx, fromAddr)
	if err != nil {
		logger.Error("failed to get nonce", "error", err)
		return nil, err
	}
	txBytes := ctx.TxBytes()
	txHash := sha256.Sum256(txBytes)
	combined := fmt.Sprintf("%s-%d", txHash, nonce)
	uniqueHash := sha256.Sum256([]byte(combined))

	inputParamsList := newDelegationParams(
		msg.BaseInfo, assetstypes.ExocoreAssetAddr, assetstypes.ExocoreChainLzID, uniqueHash,
	)
	cachedCtx, writeFunc := ctx.CacheContext()
	for _, inputParams := range inputParamsList {
		if err := k.UndelegateFrom(cachedCtx, inputParams); err != nil {
			return nil, err
		}
	}
	writeFunc()
	return &types.UndelegationResponse{}, nil
}

// newDelegationParams creates delegation params from the given base info.
func newDelegationParams(
	baseInfo *types.DelegationIncOrDecInfo,
	assetAddrStr string, clientChainLzID uint64,
	txHash common.Hash,
) []*types.DelegationOrUndelegationParams {
	// can use `Must` since pre-validated
	stakerAddr := sdk.MustAccAddressFromBech32(baseInfo.FromAddress).Bytes()
	res := make([]*types.DelegationOrUndelegationParams, 0, 1)
	for _, kv := range baseInfo.PerOperatorAmounts {
		// can use `Must` since pre-validated
		operatorAddr := sdk.MustAccAddressFromBech32(kv.Key)
		inputParams := types.NewDelegationOrUndelegationParams(
			clientChainLzID,
			assetstypes.DelegateTo,
			common.HexToAddress(assetAddrStr).Bytes(),
			operatorAddr,
			stakerAddr,
			kv.Value.Amount,
			txHash,
		)
		res = append(res, inputParams)
	}
	return res
}
