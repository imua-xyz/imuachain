package keeper

import (
	context "context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	assetstypes "github.com/imua-xyz/imuachain/x/assets/types"
	"github.com/imua-xyz/imuachain/x/delegation/types"
	"github.com/minio/sha256-simd"
)

var _ types.MsgServer = &Keeper{}

// DelegateAssetToOperator todo: Delegation and Undelegation from exoCore chain directly will be implemented in future.At the moment,they are executed from client chain
func (k *Keeper) DelegateAssetToOperator(goCtx context.Context, msg *types.MsgDelegation) (*types.DelegationResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	logger := k.Logger(ctx)
	// TODO: currently we only support delegation with native token by invoking service
	if msg.AssetID != assetstypes.ExocoreAssetID {
		logger.Error("failed to delegate asset", "error", types.ErrNotSupportYet, "assetID", msg.AssetID)
		return nil, types.ErrNotSupportYet.Wrap("assets other than native token are not supported yet")
	}
	logger.Info("DelegateAssetToOperator-nativeToken", "msg", msg)

	fromAddr := sdk.MustAccAddressFromBech32(msg.BaseInfo.FromAddress)
	nonce, err := k.accountKeeper.GetSequence(ctx, fromAddr)
	if err != nil {
		logger.Error("failed to get nonce", "error", err)
		return nil, err
	}
	txBytes := ctx.TxBytes()
	txHash := sha256.Sum256(txBytes)
	combined := fmt.Sprintf("%s-%d", txHash, nonce)
	uniqueHash := sha256.Sum256([]byte(combined))

	// test for refactor
	delegationParamsList := newDelegationParams(msg.BaseInfo, assetstypes.ExocoreAssetAddr, assetstypes.ExocoreChainLzID, nonce, uniqueHash)
	for _, delegationParams := range delegationParamsList {
		if err := k.DelegateTo(ctx, delegationParams); err != nil {
			logger.Error("failed to delegate asset", "error", err, "delegationParams", delegationParams)
			return &types.DelegationResponse{}, err

		}
	}

	return &types.DelegationResponse{}, nil
}

func (k *Keeper) UndelegateAssetFromOperator(goCtx context.Context, msg *types.MsgUndelegation) (*types.UndelegationResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	logger := k.Logger(ctx)
	// TODO: currently we only support undelegation with native token by invoking service
	if msg.AssetID != assetstypes.ExocoreAssetID {
		logger.Error("failed to undelegate asset", "error", types.ErrNotSupportYet, "assetID", msg.AssetID)
		return nil, types.ErrNotSupportYet.Wrap("assets other than native token are not supported yet")
	}
	logger.Info("UndelegateAssetFromOperator", "msg", msg)

	fromAddr := sdk.MustAccAddressFromBech32(msg.BaseInfo.FromAddress)
	nonce, err := k.accountKeeper.GetSequence(ctx, fromAddr)
	if err != nil {
		logger.Error("failed to get nonce", "error", err)
		return nil, err
	}
	txBytes := ctx.TxBytes()
	txHash := sha256.Sum256(txBytes)
	combined := fmt.Sprintf("%s-%d", txHash, nonce)
	uniqueHash := sha256.Sum256([]byte(combined))

	inputParamsList := newDelegationParams(msg.BaseInfo, assetstypes.ExocoreAssetAddr, assetstypes.ExocoreChainLzID, nonce, uniqueHash)
	for _, inputParams := range inputParamsList {
		if err := k.UndelegateFrom(ctx, inputParams); err != nil {
			return nil, err
		}
	}
	return &types.UndelegationResponse{}, nil
}

func newDelegationParams(baseInfo *types.DelegationIncOrDecInfo, assetAddrStr string, clientChainLzID, txNonce uint64, txHash common.Hash) []*types.DelegationOrUndelegationParams {
	stakerAddr := sdk.MustAccAddressFromBech32(baseInfo.FromAddress).Bytes()
	res := make([]*types.DelegationOrUndelegationParams, 0, 1)
	for _, kv := range baseInfo.PerOperatorAmounts {
		operatorAddr := sdk.MustAccAddressFromBech32(kv.Key)
		inputParams := types.NewDelegationOrUndelegationParams(
			clientChainLzID,
			assetstypes.DelegateTo,
			common.HexToAddress(assetAddrStr).Bytes(),
			operatorAddr,
			stakerAddr,
			kv.Value.Amount,
			txNonce,
			txHash,
		)
		res = append(res, inputParams)
	}
	return res
}
