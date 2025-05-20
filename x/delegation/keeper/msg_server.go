package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	assetstypes "github.com/imua-xyz/imuachain/x/assets/types"
	"github.com/imua-xyz/imuachain/x/delegation/types"
	"github.com/minio/sha256-simd"
)

var _ types.MsgServer = &msgServer{}

type msgServer struct {
	Keeper
}

func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

// DelegateAssetToOperator delegates asset to operator. Currently, it only supports native token
func (k *Keeper) DelegateAssetToOperator(
	goCtx context.Context, msg *types.MsgDelegation,
) (*types.DelegationResponse, error) {
	// TODO: currently we only support delegation with native token by invoking service
	ctx := sdk.UnwrapSDKContext(goCtx)
	logger := k.Logger(ctx)
	// no need to validate whether assetID == native token, since that is done by ValidateBasic.
	logger.Info("DelegateAssetToOperator-nativeToken", "msg", msg)

	delegationParamsList := newDelegationOrUndelegationParams(
		msg.BaseInfo, assetstypes.ImuachainAssetAddr, assetstypes.ImuachainLzID, common.Hash{}, false,
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

	instantUnbonding := msg.InstantUnbonding
	inputParamsList := newDelegationOrUndelegationParams(
		msg.BaseInfo, assetstypes.ImuachainAssetAddr, assetstypes.ImuachainLzID, uniqueHash, instantUnbonding,
	)
	cachedCtx, writeFunc := ctx.CacheContext()
	for _, inputParams := range inputParamsList {
		// Get instant unbonding flag from the message
		if instantUnbonding {
			if err := k.InstantUndelegateFrom(cachedCtx, inputParams); err != nil {
				return nil, err
			}
		} else {
			if err := k.UndelegateFrom(cachedCtx, inputParams); err != nil {
				return nil, err
			}
		}
	}
	writeFunc()
	return &types.UndelegationResponse{}, nil
}

// newDelegationOrUndelegationParams creates delegation params from the given base info.
func newDelegationOrUndelegationParams(
	baseInfo *types.DelegationIncOrDecInfo,
	assetAddrStr string, clientChainLzID uint64,
	txHash common.Hash, instantUnbonding bool,
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
			instantUnbonding,
		)
		res = append(res, inputParams)
	}
	return res
}
